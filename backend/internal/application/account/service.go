package account

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/securetoken"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// FullNameMaxRunes giới hạn độ dài họ tên.
const FullNameMaxRunes = 100

// Service cài đặt các use case quản lý tài khoản.
type Service struct {
	users   UserStore
	mailer  Mailer
	baseURL string
	now     func() time.Time
	log     *slog.Logger
}

// NewService tạo Service. baseURL là địa chỉ frontend (vd. http://localhost:5173).
func NewService(users UserStore, mailer Mailer, baseURL string, now func() time.Time, log *slog.Logger) *Service {
	return &Service{users: users, mailer: mailer, baseURL: strings.TrimRight(baseURL, "/"), now: now, log: log}
}

// CreateUserInput là dữ liệu Admin nhập.
type CreateUserInput struct {
	Email    string
	FullName string
	Role     domain.Role
	// DepartmentID, JobTitleID tùy chọn (CF-02).
	DepartmentID *uuid.UUID
	JobTitleID   *uuid.UUID
}

// SearchMaxRunes giới hạn độ dài từ khóa tìm kiếm.
const SearchMaxRunes = 100

// CreateUserResult cho biết tài khoản đã tạo và email kích hoạt có gửi được không.
type CreateUserResult struct {
	User      domain.User
	EmailSent bool
}

// CreateUser tạo tài khoản pending và gửi email kích hoạt (B7).
// Lỗi gửi email không làm hỏng việc tạo; Admin có thể gửi lại.
func (s *Service) CreateUser(ctx context.Context, actor authz.Principal, in CreateUserInput) (CreateUserResult, error) {
	if err := authz.Require(actor, authz.ManageUsers); err != nil {
		return CreateUserResult{}, err
	}
	role, err := domain.ParseRole(string(in.Role))
	if err != nil {
		return CreateUserResult{}, err
	}
	if role == domain.RoleCEO {
		if err := authz.Require(actor, authz.AssignCEORole); err != nil {
			return CreateUserResult{}, err
		}
	}
	name, err := normalizeFullName(in.FullName)
	if err != nil {
		return CreateUserResult{}, err
	}
	email := domain.NormalizeEmail(in.Email)

	raw, hash, err := securetoken.New()
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("generate activation token: %w", err)
	}
	now := s.now()
	exp := now.Add(domain.ActivationTokenTTL)
	id, err := s.users.Create(ctx, NewUser{
		Email: email, FullName: name, Role: role,
		ActivationTokenHash: hash, ActivationExpiresAt: exp, CreatedBy: actor.UserID,
		DepartmentID: in.DepartmentID, JobTitleID: in.JobTitleID,
	})
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("create user: %w", err)
	}
	u := domain.User{
		ID: id, Email: email, FullName: name, Role: role,
		Status: domain.UserStatusPending, ActivationExpiresAt: &exp, CreatedAt: now,
		DepartmentID: in.DepartmentID, JobTitleID: in.JobTitleID, Skills: []domain.UserSkill{},
	}
	sent := true
	if err := s.send(ctx, u, raw, exp); err != nil {
		s.log.WarnContext(ctx, "send activation email failed", "user_id", id, "err", err)
		sent = false
	}
	return CreateUserResult{User: u, EmailSent: sent}, nil
}

// ListUsers trả danh sách tài khoản có lọc và phân trang; kỹ năng của cả trang
// được đọc bằng một truy vấn (không N+1).
func (s *Service) ListUsers(ctx context.Context, actor authz.Principal, f UserFilter, p domain.PageRequest) (domain.Page[domain.User], error) {
	if err := authz.Require(actor, authz.ManageUsers); err != nil {
		return domain.Page[domain.User]{}, err
	}
	f.Search = strings.TrimSpace(f.Search)
	if utf8.RuneCountInString(f.Search) > SearchMaxRunes {
		return domain.Page[domain.User]{}, domain.ErrInvalidSearch
	}
	items, total, err := s.users.List(ctx, f, p)
	if err != nil {
		return domain.Page[domain.User]{}, fmt.Errorf("list users: %w", err)
	}
	if items == nil {
		items = []domain.User{}
	}
	if err := s.attachSkills(ctx, items); err != nil {
		return domain.Page[domain.User]{}, err
	}
	return domain.Page[domain.User]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: total}, nil
}

// GetUser trả hồ sơ một tài khoản kèm phòng ban, chức danh và kỹ năng.
func (s *Service) GetUser(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.User, error) {
	if err := authz.Require(actor, authz.ManageUsers); err != nil {
		return domain.User{}, err
	}
	u, err := s.users.GetProfile(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	one := []domain.User{u}
	if err := s.attachSkills(ctx, one); err != nil {
		return domain.User{}, err
	}
	return one[0], nil
}

// UpdateUserInput là phần hồ sơ Admin được sửa. Email không đổi được.
type UpdateUserInput struct {
	FullName     string
	Role         domain.Role
	DepartmentID *uuid.UUID
	JobTitleID   *uuid.UUID
}

// UpdateUser sửa họ tên, vai trò, phòng ban, chức danh.
// Chỉ CEO được đụng tới tài khoản CEO hoặc gán vai trò CEO; không ai tự đổi vai trò của mình.
func (s *Service) UpdateUser(ctx context.Context, actor authz.Principal, id uuid.UUID, in UpdateUserInput) (domain.User, error) {
	if err := authz.Require(actor, authz.ManageUsers); err != nil {
		return domain.User{}, err
	}
	role, err := domain.ParseRole(string(in.Role))
	if err != nil {
		return domain.User{}, err
	}
	name, err := normalizeFullName(in.FullName)
	if err != nil {
		return domain.User{}, err
	}
	cur, err := s.targetUser(ctx, actor, id)
	if err != nil {
		return domain.User{}, err
	}
	if role == domain.RoleCEO {
		if err := authz.Require(actor, authz.AssignCEORole); err != nil {
			return domain.User{}, err
		}
	}
	if id == actor.UserID && role != cur.Role {
		return domain.User{}, domain.ErrCannotChangeOwnRole
	}
	if err := s.users.UpdateProfile(ctx, ProfileUpdate{
		ID: id, FullName: name, Role: role, DepartmentID: in.DepartmentID, JobTitleID: in.JobTitleID,
	}); err != nil {
		return domain.User{}, fmt.Errorf("update user: %w", err)
	}
	return s.GetUser(ctx, actor, id)
}

// SetUserSkills thay toàn bộ hồ sơ kỹ năng của một người (Beginner/Intermediate/Senior).
func (s *Service) SetUserSkills(ctx context.Context, actor authz.Principal, id uuid.UUID, in []domain.UserSkillInput) ([]domain.UserSkill, error) {
	if err := authz.Require(actor, authz.ManageUsers); err != nil {
		return nil, err
	}
	skills, err := domain.ValidateUserSkills(in)
	if err != nil {
		return nil, err
	}
	if _, err := s.targetUser(ctx, actor, id); err != nil {
		return nil, err
	}
	if err := s.users.ReplaceSkills(ctx, id, skills); err != nil {
		return nil, fmt.Errorf("replace skills: %w", err)
	}
	u, err := s.GetUser(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	return u.Skills, nil
}

// targetUser đọc tài khoản sắp bị sửa; tài khoản CEO chỉ CEO được sửa.
func (s *Service) targetUser(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return domain.User{}, fmt.Errorf("get user: %w", err)
	}
	if u.Role == domain.RoleCEO {
		if err := authz.Require(actor, authz.AssignCEORole); err != nil {
			return domain.User{}, err
		}
	}
	return u, nil
}

func (s *Service) attachSkills(ctx context.Context, users []domain.User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(users))
	for i, u := range users {
		ids[i] = u.ID
	}
	byUser, err := s.users.SkillsOf(ctx, ids)
	if err != nil {
		return fmt.Errorf("load skills: %w", err)
	}
	for i := range users {
		if sk := byUser[users[i].ID]; sk != nil {
			users[i].Skills = sk
		} else {
			users[i].Skills = []domain.UserSkill{}
		}
	}
	return nil
}

func normalizeFullName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" || utf8.RuneCountInString(name) > FullNameMaxRunes {
		return "", domain.ErrInvalidFullName
	}
	return name, nil
}

// ResendActivation cấp token mới (token cũ mất hiệu lực) và gửi lại email.
func (s *Service) ResendActivation(ctx context.Context, actor authz.Principal, id uuid.UUID) error {
	if err := authz.Require(actor, authz.ManageUsers); err != nil {
		return err
	}
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}
	if u.Role == domain.RoleCEO {
		if err := authz.Require(actor, authz.AssignCEORole); err != nil {
			return err
		}
	}
	if u.Status != domain.UserStatusPending {
		return domain.ErrUserAlreadyActive
	}
	raw, hash, err := securetoken.New()
	if err != nil {
		return fmt.Errorf("generate activation token: %w", err)
	}
	exp := s.now().Add(domain.ActivationTokenTTL)
	if err := s.users.SetActivationToken(ctx, id, hash, exp); err != nil {
		return fmt.Errorf("set activation token: %w", err)
	}
	if err := s.send(ctx, u, raw, exp); err != nil {
		s.log.WarnContext(ctx, "resend activation email failed", "user_id", id, "err", err)
		return domain.ErrEmailDeliveryFailed
	}
	return nil
}

func (s *Service) send(ctx context.Context, u domain.User, raw string, exp time.Time) error {
	return s.mailer.SendActivation(ctx, ActivationEmail{
		To: u.Email, FullName: u.FullName, ExpiresAt: exp,
		Link: s.baseURL + "/activate?token=" + url.QueryEscape(raw),
	})
}
