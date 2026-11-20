package httpapi

import (
	"time"

	"github.com/google/uuid"

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
	// CF-02: cơ cấu tổ chức (null = chưa gán).
	Department *refResponse        `json:"department"`
	JobTitle   *refResponse        `json:"jobTitle"`
	Skills     []userSkillResponse `json:"skills"`
}

// refResponse là tham chiếu gọn tới một mục danh mục (phòng ban, chức danh).
type refResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type userSkillResponse struct {
	SkillID string            `json:"skillId"`
	Name    string            `json:"name"`
	Level   domain.SkillLevel `json:"level"`
}

func toRef(id *uuid.UUID, name string) *refResponse {
	if id == nil {
		return nil
	}
	return &refResponse{ID: id.String(), Name: name}
}

func toUserSkillResponses(xs []domain.UserSkill) []userSkillResponse {
	out := make([]userSkillResponse, 0, len(xs))
	for _, x := range xs {
		out = append(out, userSkillResponse{SkillID: x.SkillID.String(), Name: x.Name, Level: x.Level})
	}
	return out
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
		Department:  toRef(u.DepartmentID, u.DepartmentName),
		JobTitle:    toRef(u.JobTitleID, u.JobTitleName),
		Skills:      toUserSkillResponses(u.Skills),
	}
}

// pageResponse theo B9: {items, page, pageSize, total}.
type pageResponse[T any] struct {
	Items    []T `json:"items"`
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}
