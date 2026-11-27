package httpapi

import (
	"time"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// userResponse là hồ sơ tài khoản trả về client. Không bao giờ chứa hash mật khẩu/token.
type userResponse struct {
	ID          string             `json:"id"`
	Email       string             `json:"email"`
	FullName    string             `json:"fullName"`
	Role        domain.Role        `json:"role"`
	Status      domain.UserStatus  `json:"status"`
	LastLoginAt *time.Time         `json:"lastLoginAt"`
	CreatedAt   time.Time          `json:"createdAt"`
	Permissions []authz.Permission `json:"permissions"`
}

func toUserResponse(u domain.User) userResponse {
	return userResponse{
		ID:          u.ID.String(),
		Email:       u.Email,
		FullName:    u.FullName,
		Role:        u.Role,
		Status:      u.Status,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		Permissions: authz.PermissionsOf(u.Role),
	}
}

// pageResponse theo B9: {items, page, pageSize, total}.
type pageResponse[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}
