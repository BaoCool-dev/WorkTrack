package authz_test

// Test sinh từ thiết kế test case (docs/testing/test-cases.md) — bảng quyết định
// "vai trò × quan hệ với dự án" cho quyền quản lý dự án (R03: chỉ Manager phụ trách được sửa).

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestTC_PROJ_RequireProjectManagerDecision_CF03(t *testing.T) {
	owner := uuid.New()
	cases := []struct {
		tc    string
		pr    authz.Principal
		allow bool
	}{
		{"TC_PROJ_033_manager_phu_trach", authz.Principal{UserID: owner, Role: domain.RoleManager}, true},
		{"TC_PROJ_034_manager_khac", authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}, false},
		{"TC_PROJ_035_member_trung_id", authz.Principal{UserID: owner, Role: domain.RoleMember}, false},
		{"TC_PROJ_036_admin", authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin}, false},
		{"TC_PROJ_037_ceo_chi_xem", authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO}, false},
		{"TC_PROJ_038_vai_tro_la", authz.Principal{UserID: owner, Role: domain.Role("root")}, false},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			err := authz.RequireProjectManager(c.pr, owner)
			if c.allow && err != nil {
				t.Fatalf("mong đợi cho phép, nhận %v", err)
			}
			if !c.allow && !errors.Is(err, domain.ErrForbidden) {
				t.Fatalf("mong đợi ErrForbidden, nhận %v", err)
			}
		})
	}
}

func TestTC_AUTH_RoleWritePermissions_CF01(t *testing.T) {
	// Một ca cho mỗi vai trò: số quyền ghi đúng ma trận B2; CEO không có quyền tạo dự án (R: chỉ Manager).
	cases := []struct {
		tc     string
		role   domain.Role
		create bool
		claim  bool
		users  bool
	}{
		{"TC_AUTH_032_admin", domain.RoleAdmin, false, false, true},
		{"TC_AUTH_033_manager", domain.RoleManager, true, false, false},
		{"TC_AUTH_034_member", domain.RoleMember, false, true, false},
		{"TC_AUTH_035_ceo", domain.RoleCEO, false, false, true},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			if authz.Can(c.role, authz.CreateProject) != c.create ||
				authz.Can(c.role, authz.ClaimTask) != c.claim ||
				authz.Can(c.role, authz.ManageUsers) != c.users {
				t.Fatalf("ma trận quyền của %s sai: %v", c.role, authz.PermissionsOf(c.role))
			}
		})
	}
}
