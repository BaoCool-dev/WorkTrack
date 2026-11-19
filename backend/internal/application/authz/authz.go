// Package authz là nơi DUY NHẤT quyết định "vai trò nào được làm gì" (CLAUDE.md §5.1).
// Handler và use case chỉ gọi Can/Require, không tự so sánh vai trò.
package authz

import (
	"slices"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Principal là người dùng đã xác thực của một request.
type Principal struct {
	UserID uuid.UUID
	Role   domain.Role
}

// Permission là quyền theo vai trò (không phụ thuộc dự án).
type Permission string

const (
	ManageUsers          Permission = "users.manage"
	ManageOrg            Permission = "org.manage"
	CreateProject        Permission = "projects.create"
	ViewAllProjects      Permission = "projects.view_all"
	ViewCompanyDashboard Permission = "dashboard.company.view"
	ViewTeamDashboard    Permission = "dashboard.team.view"
)

var adminPerms = []Permission{ManageUsers, ManageOrg}

// B2: CEO có toàn bộ quyền Admin + xem mọi dự án + Dashboard toàn công ty (chỉ xem).
// Admin không tạo dự án; chỉ Manager tạo dự án (CLAUDE.md §5.3).
var matrix = map[domain.Role][]Permission{
	domain.RoleAdmin:   adminPerms,
	domain.RoleCEO:     append(slices.Clone(adminPerms), ViewAllProjects, ViewCompanyDashboard),
	domain.RoleManager: {CreateProject, ViewTeamDashboard},
	domain.RoleMember:  {},
}

// Can trả true nếu vai trò có quyền p. Vai trò/quyền lạ luôn bị từ chối.
func Can(role domain.Role, p Permission) bool {
	return slices.Contains(matrix[role], p)
}

// Require trả domain.ErrForbidden khi principal không có quyền p.
func Require(pr Principal, p Permission) error {
	if !Can(pr.Role, p) {
		return domain.ErrForbidden
	}
	return nil
}

// PermissionsOf trả danh sách quyền đã sắp xếp (bản sao, không nil) — dùng cho /auth/me.
func PermissionsOf(role domain.Role) []Permission {
	ps := slices.Clone(matrix[role])
	if ps == nil {
		ps = []Permission{}
	}
	slices.Sort(ps)
	return ps
}
