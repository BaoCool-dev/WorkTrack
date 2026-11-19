package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshTokenTTL — B4: refresh 7 ngày, xoay vòng mỗi lần làm mới.
const (
	RefreshTokenTTL = 7 * 24 * time.Hour
	AccessTokenTTL  = 15 * time.Minute
)

// RefreshToken là một phiên làm mới. Mọi token sinh ra từ một lần đăng nhập chung FamilyID;
// dùng lại token đã thu hồi → thu hồi cả họ (phát hiện token bị đánh cắp).
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FamilyID  uuid.UUID
	TokenHash []byte
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// IsExpired: hết hạn khi now >= ExpiresAt.
func (t RefreshToken) IsExpired(now time.Time) bool { return !now.Before(t.ExpiresAt) }

// IsRevoked cho biết token đã bị thu hồi (đã dùng hoặc đã đăng xuất).
func (t RefreshToken) IsRevoked() bool { return t.RevokedAt != nil }
