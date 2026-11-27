package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// UserStore là phần kho tài khoản mà luồng xác thực cần.
type UserStore interface {
	// GetByEmail trả domain.ErrUserNotFound khi không có.
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	// GetByID trả domain.ErrUserNotFound khi không có.
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	// GetByActivationTokenHash trả domain.ErrUserNotFound khi không có.
	GetByActivationTokenHash(ctx context.Context, tokenHash []byte) (domain.User, error)
	// Activate đặt mật khẩu và chuyển sang active nếu token còn khớp và còn hạn tại now;
	// ngược lại domain.ErrActivationTokenInvalid.
	Activate(ctx context.Context, id uuid.UUID, tokenHash []byte, passwordHash string, now time.Time) error
	// RegisterLoginFailure tăng bộ đếm sai mật khẩu nguyên tử; lần thứ maxFailures (≥ 1) khóa tới
	// now+lockFor và đặt lại bộ đếm. Trả locked_until sau cập nhật (nil nếu chưa từng khóa).
	// Đang khóa tại now → domain.ErrAccountLocked, không cộng dồn; user không còn → domain.ErrUserNotFound.
	RegisterLoginFailure(ctx context.Context, id uuid.UUID, now time.Time, maxFailures int, lockFor time.Duration) (*time.Time, error)
	// RecordLoginSuccess đặt lại bộ đếm; trả domain.ErrAccountLocked nếu tài khoản đang bị khóa tại at
	// (một request song song vừa khóa).
	RecordLoginSuccess(ctx context.Context, id uuid.UUID, at time.Time) error
}

// RefreshTokenStore lưu refresh token (chỉ hash).
type RefreshTokenStore interface {
	Insert(ctx context.Context, t domain.RefreshToken) error
	// GetByHash trả domain.ErrRefreshTokenNotFound khi không có.
	GetByHash(ctx context.Context, tokenHash []byte) (domain.RefreshToken, error)
	// RevokeIfActive thu hồi nguyên tử; false nghĩa là token đã bị thu hồi trước đó.
	RevokeIfActive(ctx context.Context, id uuid.UUID, at time.Time) (bool, error)
	RevokeFamily(ctx context.Context, familyID uuid.UUID, at time.Time) error
}

// PasswordHasher băm/so khớp mật khẩu (bcrypt ở hạ tầng).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

// AccessTokenIssuer phát access token (JWT ở hạ tầng).
type AccessTokenIssuer interface {
	Issue(p authz.Principal, now time.Time) (token string, expiresAt time.Time, err error)
}
