//go:build integration

package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestOrgRepo_DepartmentTree_CF02(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	repo := NewOrgRepo(pool)

	root, err := repo.CreateDepartment(ctx, "Khối Kỹ thuật", nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := repo.CreateDepartment(ctx, "Nhóm Backend", &root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateDepartment(ctx, "nhóm backend", &root); !errors.Is(err, domain.ErrDepartmentNameTaken) {
		t.Fatalf("trùng tên cùng cha: %v", err)
	}
	if _, err := repo.CreateDepartment(ctx, "Nhóm Backend", nil); err != nil {
		t.Fatalf("cùng tên khác cha phải được phép: %v", err)
	}
	if _, err := repo.CreateDepartment(ctx, "Khối kỹ thuật", nil); !errors.Is(err, domain.ErrDepartmentNameTaken) {
		t.Fatalf("trùng tên ở gốc: %v", err)
	}
	missing := uuid.New()
	if _, err := repo.CreateDepartment(ctx, "X", &missing); !errors.Is(err, domain.ErrDepartmentNotFound) {
		t.Fatalf("cha không tồn tại: %v", err)
	}
	if err := repo.UpdateDepartment(ctx, missing, "X", nil); !errors.Is(err, domain.ErrDepartmentNotFound) {
		t.Fatalf("sửa phòng ban không có: %v", err)
	}
	if err := repo.DeleteDepartment(ctx, root); !errors.Is(err, domain.ErrDepartmentHasChildren) {
		t.Fatalf("xóa khi còn con: %v", err)
	}

	users := NewUserRepo(pool)
	uid, err := users.Create(ctx, account.NewUser{
		Email: "an@worktrack.local", FullName: "Nguyễn Văn An", Role: domain.RoleMember,
		DepartmentID: &child, ActivationTokenHash: []byte("h"), ActivationExpiresAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteDepartment(ctx, child); !errors.Is(err, domain.ErrDepartmentInUse) {
		t.Fatalf("xóa khi còn nhân sự: %v", err)
	}

	ds, err := repo.ListDepartments(ctx)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[uuid.UUID]int{}
	for _, d := range ds {
		counts[d.ID] = d.MemberCount
	}
	if len(ds) != 3 || counts[child] != 1 || counts[root] != 0 {
		t.Fatalf("list = %+v", ds)
	}

	// Lọc theo phòng ban gốc phải gồm cả nhân sự ở phòng ban con.
	got, total, err := users.List(ctx, account.UserFilter{DepartmentID: &root}, domain.NewPageRequest(1, 20))
	if err != nil || total != 1 || len(got) != 1 || got[0].ID != uid || got[0].DepartmentName != "Nhóm Backend" {
		t.Fatalf("lọc cây: %+v %d %v", got, total, err)
	}
}

func TestOrgRepo_CatalogsAndSkills_CF02(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	repo := NewOrgRepo(pool)
	users := NewUserRepo(pool)

	jt, err := repo.CreateJobTitle(ctx, "Lập trình viên")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateJobTitle(ctx, "LẬP TRÌNH VIÊN"); !errors.Is(err, domain.ErrJobTitleNameTaken) {
		t.Fatalf("trùng chức danh: %v", err)
	}
	goID, _ := repo.CreateSkill(ctx, "Go")
	reactID, _ := repo.CreateSkill(ctx, "React")
	if _, err := repo.CreateSkill(ctx, "go"); !errors.Is(err, domain.ErrSkillNameTaken) {
		t.Fatalf("trùng kỹ năng: %v", err)
	}
	if err := repo.UpdateSkill(ctx, reactID, "Go"); !errors.Is(err, domain.ErrSkillNameTaken) {
		t.Fatalf("đổi tên trùng: %v", err)
	}
	if err := repo.UpdateSkill(ctx, uuid.New(), "Rust"); !errors.Is(err, domain.ErrSkillNotFound) {
		t.Fatalf("sửa kỹ năng không có: %v", err)
	}

	uid, err := users.Create(ctx, account.NewUser{
		Email: "binh@worktrack.local", FullName: "Trần Thị Bình", Role: domain.RoleMember,
		JobTitleID: &jt, ActivationTokenHash: []byte("h"), ActivationExpiresAt: time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := users.ReplaceSkills(ctx, uid, []domain.UserSkill{
		{SkillID: goID, Level: domain.SkillSenior}, {SkillID: reactID, Level: domain.SkillBeginner},
	}); err != nil {
		t.Fatal(err)
	}
	if err := users.ReplaceSkills(ctx, uid, []domain.UserSkill{{SkillID: goID, Level: domain.SkillIntermediate}}); err != nil {
		t.Fatal(err)
	}
	m, err := users.SkillsOf(ctx, []uuid.UUID{uid})
	if err != nil || len(m[uid]) != 1 || m[uid][0].Level != domain.SkillIntermediate || m[uid][0].Name != "Go" {
		t.Fatalf("skills = %+v %v", m, err)
	}
	if err := users.ReplaceSkills(ctx, uid, []domain.UserSkill{{SkillID: uuid.New(), Level: domain.SkillBeginner}}); !errors.Is(err, domain.ErrSkillNotFound) {
		t.Fatalf("kỹ năng không tồn tại: %v", err)
	}
	if err := repo.DeleteSkill(ctx, goID); !errors.Is(err, domain.ErrSkillInUse) {
		t.Fatalf("xóa kỹ năng đang dùng: %v", err)
	}
	if err := repo.DeleteSkill(ctx, reactID); err != nil {
		t.Fatalf("xóa kỹ năng không dùng: %v", err)
	}
	if err := repo.DeleteJobTitle(ctx, jt); !errors.Is(err, domain.ErrJobTitleInUse) {
		t.Fatalf("xóa chức danh đang dùng: %v", err)
	}

	// Sửa hồ sơ: email giữ nguyên; tham chiếu sai bị từ chối.
	bad := uuid.New()
	if err := users.UpdateProfile(ctx, account.ProfileUpdate{ID: uid, FullName: "B", Role: domain.RoleMember, DepartmentID: &bad}); !errors.Is(err, domain.ErrDepartmentNotFound) {
		t.Fatalf("phòng ban sai: %v", err)
	}
	if err := users.UpdateProfile(ctx, account.ProfileUpdate{ID: uid, FullName: "Trần Bình", Role: domain.RoleManager}); err != nil {
		t.Fatal(err)
	}
	u, err := users.GetProfile(ctx, uid)
	if err != nil || u.FullName != "Trần Bình" || u.Role != domain.RoleManager || u.Email != "binh@worktrack.local" || u.JobTitleID != nil {
		t.Fatalf("profile = %+v %v", u, err)
	}

	// Tìm kiếm không phân biệt hoa thường, ký tự % được escape.
	got, total, err := users.List(ctx, account.UserFilter{Search: "BÌNH"}, domain.NewPageRequest(1, 20))
	if err != nil || total != 1 || len(got) != 1 {
		t.Fatalf("search: %d %v", total, err)
	}
	if _, total, _ := users.List(ctx, account.UserFilter{Search: "%"}, domain.NewPageRequest(1, 20)); total != 0 {
		t.Fatalf("%% phải được escape, total = %d", total)
	}
}
