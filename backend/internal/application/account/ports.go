// Package account: Admin/CEO quản lý tài khoản người dùng (CF-01).
package account

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// NewUser là dữ liệu để tạo một tài khoản chờ kích hoạt.
type NewUser struct {
	Email               string
	FullName            string
	Role                domain.Role
	ActivationTokenHash []byte
	ActivationExpiresAt time.Time
	CreatedBy           uuid.UUID
	DepartmentID        *uuid.UUID
	JobTitleID          *uuid.UUID
}

// UserFilter lọc danh sách tài khoản (CF-02). Trường nil/rỗng = không lọc.
type UserFilter struct {
	// Search khớp một phần họ tên hoặc email, không phân biệt hoa thường.
	Search string
	Role   *domain.Role
	Status *domain.UserStatus
	// DepartmentID gồm cả phòng ban con.
	DepartmentID *uuid.UUID
	JobTitleID   *uuid.UUID
}

// ProfileUpdate là dữ liệu hồ sơ được phép sửa (email chỉ đọc).
type ProfileUpdate struct {
	ID           uuid.UUID
	FullName     string
	Role         domain.Role
	DepartmentID *uuid.UUID
	JobTitleID   *uuid.UUID
}

// UserStore là port lưu tài khoản.
type UserStore interface {
	// Create trả domain.ErrEmailTaken khi email đã tồn tại (không phân biệt hoa thường),
	// domain.ErrDepartmentNotFound / domain.ErrJobTitleNotFound khi tham chiếu không tồn tại.
	Create(ctx context.Context, u NewUser) (uuid.UUID, error)
	List(ctx context.Context, f UserFilter, p domain.PageRequest) ([]domain.User, int, error)
	// GetProfile như GetByID nhưng kèm tên phòng ban, chức danh (không kèm kỹ năng).
	GetProfile(ctx context.Context, id uuid.UUID) (domain.User, error)
	// UpdateProfile trả domain.ErrUserNotFound, ErrDepartmentNotFound, ErrJobTitleNotFound.
	UpdateProfile(ctx context.Context, u ProfileUpdate) error
	// ReplaceSkills thay toàn bộ hồ sơ kỹ năng; trả domain.ErrSkillNotFound khi kỹ năng không tồn tại.
	ReplaceSkills(ctx context.Context, userID uuid.UUID, skills []domain.UserSkill) error
	// SkillsOf đọc kỹ năng của nhiều người trong một truy vấn.
	SkillsOf(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]domain.UserSkill, error)
	// GetByID trả domain.ErrUserNotFound khi không có.
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	// SetActivationToken trả domain.ErrUserAlreadyActive khi tài khoản không còn ở trạng thái pending.
	SetActivationToken(ctx context.Context, id uuid.UUID, hash []byte, expiresAt time.Time) error
}

// ActivationEmail là nội dung email kích hoạt.
type ActivationEmail struct {
	To        string
	FullName  string
	Link      string
	ExpiresAt time.Time
}

// Mailer là port gửi email.
type Mailer interface {
	SendActivation(ctx context.Context, m ActivationEmail) error
}
