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
}

// UserStore là port lưu tài khoản.
type UserStore interface {
	// Create trả domain.ErrEmailTaken khi email đã tồn tại (không phân biệt hoa thường).
	Create(ctx context.Context, u NewUser) (uuid.UUID, error)
	List(ctx context.Context, p domain.PageRequest) ([]domain.User, int, error)
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
