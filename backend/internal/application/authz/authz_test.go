package authz_test

import (
	"errors"
	"testing"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestCan_Matrix_CF01_B2(t *testing.T) {
	cases := []struct {
		perm  authz.Permission
		allow []domain.Role
	}{
		{authz.ManageUsers, []domain.Role{domain.RoleAdmin, domain.RoleCEO}},
		{authz.ManageOrg, []domain.Role{domain.RoleAdmin, domain.RoleCEO}},
		{authz.CreateProject, []domain.Role{domain.RoleManager}},
		{authz.ViewAllProjects, []domain.Role{domain.RoleCEO}},
		{authz.ViewCompanyDashboard, []domain.Role{domain.RoleCEO}},
		{authz.ViewTeamDashboard, []domain.Role{domain.RoleManager}},
	}
	for _, tc := range cases {
		for _, role := range domain.Roles {
			want := false
			for _, a := range tc.allow {
				if a == role {
					want = true
				}
			}
			if got := authz.Can(role, tc.perm); got != want {
				t.Errorf("Can(%s, %s) = %v, want %v", role, tc.perm, got, want)
			}
		}
	}
}

func TestCan_CEOInheritsAdmin_CF01_B2(t *testing.T) {
	for _, p := range authz.PermissionsOf(domain.RoleAdmin) {
		if !authz.Can(domain.RoleCEO, p) {
			t.Errorf("CEO thiếu quyền Admin %s", p)
		}
	}
}

func TestCan_AdminCannotCreateProject_CF01_R(t *testing.T) {
	if authz.Can(domain.RoleAdmin, authz.CreateProject) {
		t.Fatal("Admin không được tạo dự án (CLAUDE.md §5.3)")
	}
}

func TestCan_UnknownRoleOrPermission_Denied(t *testing.T) {
	if authz.Can(domain.Role("hacker"), authz.ManageUsers) {
		t.Fatal("vai trò lạ phải bị từ chối")
	}
	if authz.Can(domain.RoleCEO, authz.Permission("nope")) {
		t.Fatal("quyền lạ phải bị từ chối")
	}
}

func TestRequire_ReturnsForbidden(t *testing.T) {
	err := authz.Require(authz.Principal{Role: domain.RoleMember}, authz.ManageUsers)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("got %v, want ErrForbidden", err)
	}
	if err := authz.Require(authz.Principal{Role: domain.RoleAdmin}, authz.ManageUsers); err != nil {
		t.Fatalf("admin: %v", err)
	}
}

func TestPermissionsOf_SortedAndCopy(t *testing.T) {
	ps := authz.PermissionsOf(domain.RoleCEO)
	for i := 1; i < len(ps); i++ {
		if ps[i-1] >= ps[i] {
			t.Fatalf("chưa sắp xếp: %v", ps)
		}
	}
	if len(authz.PermissionsOf(domain.Role("x"))) != 0 {
		t.Fatal("vai trò lạ phải rỗng")
	}
	if ps := authz.PermissionsOf(domain.RoleMember); ps == nil {
		t.Fatal("trả slice rỗng, không nil (JSON [] thay vì null)")
	}
}
