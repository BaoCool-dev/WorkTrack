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
}

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
	name := strings.TrimSpace(in.FullName)
	if name == "" || utf8.RuneCountInString(name) > FullNameMaxRunes {
		return CreateUserResult{}, domain.ErrInvalidFullName
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
	})
	if err != nil {
		return CreateUserResult{}, fmt.Errorf("create user: %w", err)
	}
	u := domain.User{
		ID: id, Email: email, FullName: name, Role: role,
		Status: domain.UserStatusPending, ActivationExpiresAt: &exp, CreatedAt: now,
	}
	sent := true
	if err := s.send(ctx, u, raw, exp); err != nil {
		s.log.WarnContext(ctx, "send activation email failed", "user_id", id, "err", err)
		sent = false
	}
	return CreateUserResult{User: u, EmailSent: sent}, nil
}

// ListUsers trả danh sách tài khoản có phân trang.
func (s *Service) ListUsers(ctx context.Context, actor authz.Principal, p domain.PageRequest) (domain.Page[domain.User], error) {
	if err := authz.Require(actor, authz.ManageUsers); err != nil {
		return domain.Page[domain.User]{}, err
	}
	items, total, err := s.users.List(ctx, p)
	if err != nil {
		return domain.Page[domain.User]{}, fmt.Errorf("list users: %w", err)
	}
	if items == nil {
		items = []domain.User{}
	}
	return domain.Page[domain.User]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: total}, nil
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
