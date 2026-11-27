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
	ManageUsers     Permission = "users.manage"
	ManageOrg       Permission = "org.manage"
	CreateProject   Permission = "projects.create"
	ViewAllProjects Permission = "projects.view_all"
	// ViewOwnProjects: xem các dự án mình là thành viên (Manager, Member). Admin không có quyền này.
	ViewOwnProjects      Permission = "projects.view_own"
	ViewCompanyDashboard Permission = "dashboard.company.view"
	ViewTeamDashboard    Permission = "dashboard.team.view"
	// AssignCEORole: chỉ CEO được tạo tài khoản vai trò CEO (Admin không tự nâng quyền lên CEO).
	AssignCEORole Permission = "users.assign_ceo"
	// ManageSettings: sửa tham số hệ thống, vd. hệ số quy đổi Story Point → giờ (B1). CF-06.
	ManageSettings Permission = "settings.manage"
	// ClaimTask: tự nhận công việc chưa có người nhận ở nhóm todo (CF-06). Chỉ Member;
	// Manager giao việc (kể cả cho chính mình) qua chức năng sửa công việc.
	ClaimTask Permission = "tasks.claim"
)

var adminPerms = []Permission{ManageUsers, ManageOrg, ManageSettings}

// B2: CEO có toàn bộ quyền Admin + xem mọi dự án + Dashboard toàn công ty (chỉ xem).
// Admin không tạo dự án; chỉ Manager tạo dự án (CLAUDE.md §5.3).
var matrix = map[domain.Role][]Permission{
	domain.RoleAdmin:   adminPerms,
	domain.RoleCEO:     append(slices.Clone(adminPerms), AssignCEORole, ViewAllProjects, ViewCompanyDashboard),
	domain.RoleManager: {CreateProject, ViewOwnProjects, ViewTeamDashboard},
	domain.RoleMember:  {ViewOwnProjects, ClaimTask},
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

// RequireProjectManager: chỉ Manager phụ trách dự án (managerID) được sửa thông tin, thành viên
// và cấu hình Kanban của dự án đó (CF-03). Manager khác, CEO, Admin, Member → ErrForbidden.
func RequireProjectManager(pr Principal, managerID uuid.UUID) error {
	if !Can(pr.Role, CreateProject) || pr.UserID != managerID {
		return domain.ErrForbidden
	}
	return nil
}
