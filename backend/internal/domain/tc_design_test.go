package domain_test

// Test sinh từ thiết kế test case (docs/testing/test-cases.md).
// Mỗi subtest mang đúng mã TC_<MODULE>_<NNN> để đối chiếu hai chiều với bảng test case.
// Kỹ thuật: EP + BVA (min-1, min, max, max+1), bảng quyết định, chuyển trạng thái.

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// errCase là một dòng bảng: mã TC, đầu vào đã chuẩn bị sẵn trong fn, lỗi mong đợi (nil = hợp lệ).
type errCase struct {
	tc   string
	fn   func() error
	want error
}

func runErrCases(t *testing.T, cases []errCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			got := c.fn()
			if c.want == nil && got != nil {
				t.Fatalf("%s: mong đợi hợp lệ, nhận lỗi %v", c.tc, got)
			}
			if c.want != nil && !errors.Is(got, c.want) {
				t.Fatalf("%s: mong đợi %v, nhận %v", c.tc, c.want, got)
			}
		})
	}
}

func discard[T any](_ T, err error) error { return err }

// ---------- CF-01: mật khẩu (BVA theo rune và byte), trạng thái tài khoản, khóa đăng nhập ----------

func TestTC_AUTH_PasswordBVA_CF01(t *testing.T) {
	pw := func(n int) string { return strings.Repeat("a", n-1) + "1" } // n ký tự, có chữ + số
	runErrCases(t, []errCase{
		{"TC_AUTH_010_7_ky_tu", func() error { return domain.ValidatePassword(pw(7)) }, domain.ErrPasswordTooWeak},
		{"TC_AUTH_011_8_ky_tu", func() error { return domain.ValidatePassword(pw(8)) }, nil},
		{"TC_AUTH_012_72_byte", func() error { return domain.ValidatePassword(pw(72)) }, nil},
		{"TC_AUTH_013_73_byte", func() error { return domain.ValidatePassword(pw(73)) }, domain.ErrPasswordTooWeak},
		{"TC_AUTH_014_chi_chu", func() error { return domain.ValidatePassword("abcdefgh") }, domain.ErrPasswordTooWeak},
		{"TC_AUTH_015_chi_so", func() error { return domain.ValidatePassword("12345678") }, domain.ErrPasswordTooWeak},
		// 8 rune nhưng hơn 8 byte: biên dưới tính theo ký tự, không theo byte.
		{"TC_AUTH_016_tieng_viet_8_rune", func() error { return domain.ValidatePassword("Mậtkhẩu1") }, nil},
		// 25 ký tự "ệ" (3 byte) = 75 byte > 72: biên trên tính theo byte vì bcrypt cắt phần dư.
		{"TC_AUTH_017_73_byte_da_byte", func() error { return domain.ValidatePassword(strings.Repeat("ệ", 24) + "1a") }, domain.ErrPasswordTooWeak},
	})
}

func TestTC_AUTH_AccountState_CF01(t *testing.T) {
	// Bảng quyết định: trạng thái tài khoản × (mật khẩu đã đúng) → cho đăng nhập?
	runErrCases(t, []errCase{
		{"TC_AUTH_018_active", func() error { return domain.User{Status: domain.UserStatusActive}.CheckCanLogin() }, nil},
		{"TC_AUTH_019_pending", func() error { return domain.User{Status: domain.UserStatusPending}.CheckCanLogin() }, domain.ErrAccountNotActive},
		{"TC_AUTH_020_disabled", func() error { return domain.User{Status: domain.UserStatusDisabled}.CheckCanLogin() }, domain.ErrAccountDisabled},
		{"TC_AUTH_021_status_la", func() error { return discard(domain.ParseUserStatus("locked")) }, domain.ErrInvalidUserStatus},
	})
}

func TestTC_AUTH_LockWindow_CF01(t *testing.T) {
	// Chuyển trạng thái "Đang khóa" → "Mở khóa" đúng tại mốc locked_until (BVA trên thời gian).
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	until := now.Add(domain.LoginLockDuration)
	u := domain.User{LockedUntil: &until}
	cases := []struct {
		tc   string
		at   time.Time
		user domain.User
		want bool
	}{
		{"TC_AUTH_022_truoc_moc_1s", until.Add(-time.Second), u, true},
		{"TC_AUTH_023_dung_moc", until, u, false},
		{"TC_AUTH_024_sau_moc", until.Add(time.Second), u, false},
		{"TC_AUTH_025_chua_tung_khoa", now, domain.User{}, false},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			if got := c.user.IsLocked(c.at); got != c.want {
				t.Fatalf("IsLocked = %v, mong đợi %v", got, c.want)
			}
		})
	}
	t.Run("TC_AUTH_026_hang_so_chinh_sach", func(t *testing.T) {
		if domain.MaxFailedLogins != 5 || domain.LoginLockDuration != 15*time.Minute || domain.ActivationTokenTTL != 72*time.Hour {
			t.Fatal("chính sách B7/B8 bị đổi: 5 lần sai, khóa 15 phút, link kích hoạt 72 giờ")
		}
	})
}

func TestTC_AUTH_RoleAndEmail_CF01(t *testing.T) {
	for _, r := range []string{"admin", "manager", "member", "ceo"} {
		t.Run("TC_AUTH_027_vai_tro_hop_le_"+r, func(t *testing.T) {
			if _, err := domain.ParseRole(r); err != nil {
				t.Fatalf("vai trò %q phải hợp lệ: %v", r, err)
			}
		})
	}
	runErrCases(t, []errCase{
		{"TC_AUTH_028_vai_tro_viet_hoa", func() error { return discard(domain.ParseRole("Admin")) }, domain.ErrInvalidRole},
		{"TC_AUTH_029_vai_tro_rong", func() error { return discard(domain.ParseRole("")) }, domain.ErrInvalidRole},
		{"TC_AUTH_030_vai_tro_la", func() error { return discard(domain.ParseRole("superadmin")) }, domain.ErrInvalidRole},
	})
	t.Run("TC_AUTH_031_chuan_hoa_email", func(t *testing.T) {
		if got := domain.NormalizeEmail("  An.Nguyen@WorkTrack.LOCAL "); got != "an.nguyen@worktrack.local" {
			t.Fatalf("NormalizeEmail = %q", got)
		}
	})
}

// ---------- CF-02: tên danh mục, kỹ năng, cây phòng ban, phân trang ----------

func TestTC_ORG_NameBVA_CF02(t *testing.T) {
	runErrCases(t, []errCase{
		{"TC_ORG_010_rong", func() error { return discard(domain.NormalizeOrgName("")) }, domain.ErrInvalidOrgName},
		{"TC_ORG_011_chi_khoang_trang", func() error { return discard(domain.NormalizeOrgName("   ")) }, domain.ErrInvalidOrgName},
		{"TC_ORG_012_1_ky_tu", func() error { return discard(domain.NormalizeOrgName("A")) }, nil},
		{"TC_ORG_013_100_ky_tu", func() error { return discard(domain.NormalizeOrgName(strings.Repeat("Đ", 100))) }, nil},
		{"TC_ORG_014_101_ky_tu", func() error { return discard(domain.NormalizeOrgName(strings.Repeat("Đ", 101))) }, domain.ErrInvalidOrgName},
	})
	t.Run("TC_ORG_015_gop_khoang_trang", func(t *testing.T) {
		if got, _ := domain.NormalizeOrgName("  Phòng   Kỹ thuật "); got != "Phòng Kỹ thuật" {
			t.Fatalf("NormalizeOrgName = %q", got)
		}
	})
}

func TestTC_ORG_SkillsBVA_CF02(t *testing.T) {
	skills := func(n int, level string) []domain.UserSkillInput {
		out := make([]domain.UserSkillInput, n)
		for i := range out {
			out[i] = domain.UserSkillInput{SkillID: uuid.New(), Level: level}
		}
		return out
	}
	same := uuid.New()
	runErrCases(t, []errCase{
		{"TC_ORG_016_0_ky_nang", func() error { return discard(domain.ValidateUserSkills(nil)) }, nil},
		{"TC_ORG_017_50_ky_nang", func() error { return discard(domain.ValidateUserSkills(skills(50, "senior"))) }, nil},
		{"TC_ORG_018_51_ky_nang", func() error { return discard(domain.ValidateUserSkills(skills(51, "senior"))) }, domain.ErrTooManySkills},
		{"TC_ORG_019_muc_viet_hoa", func() error { return discard(domain.ValidateUserSkills(skills(1, "Senior"))) }, domain.ErrInvalidSkillLevel},
		{"TC_ORG_020_muc_rong", func() error { return discard(domain.ValidateUserSkills(skills(1, ""))) }, domain.ErrInvalidSkillLevel},
		{"TC_ORG_021_trung_ky_nang", func() error {
			return discard(domain.ValidateUserSkills([]domain.UserSkillInput{{SkillID: same, Level: "beginner"}, {SkillID: same, Level: "senior"}}))
		}, domain.ErrDuplicateSkill},
	})
}

func TestTC_ORG_DepartmentTree_CF02(t *testing.T) {
	// Cây: A ← B ← C (C là cháu của A).
	a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	tree := map[uuid.UUID]*uuid.UUID{a: nil, b: &a, c: &b, d: nil}
	missing := uuid.New()
	runErrCases(t, []errCase{
		{"TC_ORG_022_khong_co_cha", func() error { return domain.ValidateDepartmentParent(d, nil, tree) }, nil},
		{"TC_ORG_023_cha_hop_le", func() error { return domain.ValidateDepartmentParent(d, &c, tree) }, nil},
		{"TC_ORG_024_cha_khong_ton_tai", func() error { return domain.ValidateDepartmentParent(d, &missing, tree) }, domain.ErrDepartmentNotFound},
		{"TC_ORG_025_cha_la_chinh_no", func() error { return domain.ValidateDepartmentParent(a, &a, tree) }, domain.ErrDepartmentCycle},
		{"TC_ORG_026_cha_la_chau", func() error { return domain.ValidateDepartmentParent(a, &c, tree) }, domain.ErrDepartmentCycle},
	})
}

func TestTC_ORG_PageBVA_CF02(t *testing.T) {
	cases := []struct {
		tc               string
		page, size       int
		wantPage, wantSz int
	}{
		{"TC_ORG_027_size_0", 1, 0, 1, domain.DefaultPageSize},
		{"TC_ORG_028_size_1", 1, 1, 1, 1},
		{"TC_ORG_029_size_100", 1, 100, 1, 100},
		{"TC_ORG_030_size_101", 1, 101, 1, domain.MaxPageSize},
		{"TC_ORG_031_page_0", 0, 20, 1, 20},
		{"TC_ORG_032_page_vuot_tran", domain.MaxPage + 1, 20, domain.MaxPage, 20},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			p := domain.NewPageRequest(c.page, c.size)
			if p.Page != c.wantPage || p.PageSize != c.wantSz {
				t.Fatalf("NewPageRequest(%d,%d) = %+v", c.page, c.size, p)
			}
		})
	}
}

// ---------- CF-03: dự án, cột Kanban ----------

func TestTC_PROJ_FieldsBVA_CF03(t *testing.T) {
	d1 := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	d0 := d1.AddDate(0, 0, -1)
	runErrCases(t, []errCase{
		{"TC_PROJ_010_ten_rong", func() error { return discard(domain.NormalizeProjectName("  ")) }, domain.ErrInvalidProjectName},
		{"TC_PROJ_011_ten_1", func() error { return discard(domain.NormalizeProjectName("A")) }, nil},
		{"TC_PROJ_012_ten_150", func() error { return discard(domain.NormalizeProjectName(strings.Repeat("ư", 150))) }, nil},
		{"TC_PROJ_013_ten_151", func() error { return discard(domain.NormalizeProjectName(strings.Repeat("ư", 151))) }, domain.ErrInvalidProjectName},
		{"TC_PROJ_014_mo_ta_2000", func() error { return discard(domain.NormalizeProjectDescription(strings.Repeat("x", 2000))) }, nil},
		{"TC_PROJ_015_mo_ta_2001", func() error { return discard(domain.NormalizeProjectDescription(strings.Repeat("x", 2001))) }, domain.ErrInvalidProjectDescription},
		{"TC_PROJ_016_ket_thuc_truoc_bat_dau", func() error { return domain.ValidateProjectDates(&d1, &d0) }, domain.ErrInvalidProjectDates},
		{"TC_PROJ_017_ket_thuc_bang_bat_dau", func() error { return domain.ValidateProjectDates(&d1, &d1) }, nil},
		{"TC_PROJ_018_trang_thai_la", func() error { return discard(domain.ParseProjectStatus("archived")) }, domain.ErrInvalidProjectStatus},
	})
}

func TestTC_PROJ_ColumnsBVA_CF03(t *testing.T) {
	cols := func(n int) []domain.ColumnSpec {
		out := make([]domain.ColumnSpec, n)
		for i := range out {
			out[i] = domain.ColumnSpec{Name: "Cột " + strings.Repeat("I", i+1), StatusGroup: domain.GroupTodo}
		}
		if n > 0 {
			out[n-1].StatusGroup = domain.GroupDone
		}
		return out
	}
	runErrCases(t, []errCase{
		{"TC_PROJ_019_ten_cot_50", func() error { return discard(domain.NormalizeColumnName(strings.Repeat("c", 50))) }, nil},
		{"TC_PROJ_020_ten_cot_51", func() error { return discard(domain.NormalizeColumnName(strings.Repeat("c", 51))) }, domain.ErrInvalidColumnName},
		{"TC_PROJ_021_1_cot", func() error { return domain.ValidateColumnSpecs(cols(1)) }, domain.ErrColumnCount},
		{"TC_PROJ_022_2_cot", func() error { return domain.ValidateColumnSpecs(cols(2)) }, nil},
		{"TC_PROJ_023_8_cot", func() error { return domain.ValidateColumnSpecs(cols(8)) }, nil},
		{"TC_PROJ_024_9_cot", func() error { return domain.ValidateColumnSpecs(cols(9)) }, domain.ErrColumnCount},
		{"TC_PROJ_025_thieu_cot_done", func() error {
			return domain.ValidateColumnSpecs([]domain.ColumnSpec{{Name: "A", StatusGroup: domain.GroupTodo}, {Name: "B", StatusGroup: domain.GroupInProgress}})
		}, domain.ErrBoardNeedsDoneColumn},
		{"TC_PROJ_026_trung_ten_khac_hoa_thuong", func() error {
			return domain.ValidateColumnSpecs([]domain.ColumnSpec{{Name: "Done", StatusGroup: domain.GroupTodo}, {Name: "done", StatusGroup: domain.GroupDone}})
		}, domain.ErrColumnNameTaken},
		{"TC_PROJ_027_nhom_la", func() error { return discard(domain.ParseStatusGroup("review")) }, domain.ErrInvalidStatusGroup},
	})
}

func TestTC_PROJ_DeleteColumn_CF03(t *testing.T) {
	todo := domain.BoardColumn{ID: uuid.New(), StatusGroup: domain.GroupTodo}
	busy := domain.BoardColumn{ID: uuid.New(), StatusGroup: domain.GroupInProgress, TaskCount: 1}
	done := domain.BoardColumn{ID: uuid.New(), StatusGroup: domain.GroupDone}
	runErrCases(t, []errCase{
		{"TC_PROJ_028_con_dung_2_cot", func() error { return domain.CheckColumnDeletable([]domain.BoardColumn{todo, done}, todo.ID) }, domain.ErrColumnCount},
		{"TC_PROJ_029_cot_done_duy_nhat", func() error { return domain.CheckColumnDeletable([]domain.BoardColumn{todo, busy, done}, done.ID) }, domain.ErrBoardNeedsDoneColumn},
		{"TC_PROJ_030_cot_con_viec", func() error { return domain.CheckColumnDeletable([]domain.BoardColumn{todo, busy, done}, busy.ID) }, domain.ErrColumnHasTasks},
		{"TC_PROJ_031_xoa_hop_le", func() error { return domain.CheckColumnDeletable([]domain.BoardColumn{todo, busy, done}, todo.ID) }, nil},
		{"TC_PROJ_032_cot_khong_ton_tai", func() error { return domain.CheckColumnDeletable([]domain.BoardColumn{todo, busy, done}, uuid.New()) }, domain.ErrColumnNotFound},
	})
}

// ---------- CF-04/05: vai trò trên bảng, luật chuyển cột, WIP ----------

func TestTC_KANBAN_WIPLimitBVA_CF05(t *testing.T) {
	runErrCases(t, []errCase{
		{"TC_KANBAN_010_wip_0", func() error { return domain.ValidateWIPLimit(0) }, domain.ErrInvalidWIPLimit},
		{"TC_KANBAN_011_wip_1", func() error { return domain.ValidateWIPLimit(1) }, nil},
		{"TC_KANBAN_012_wip_20", func() error { return domain.ValidateWIPLimit(20) }, nil},
		{"TC_KANBAN_013_wip_21", func() error { return domain.ValidateWIPLimit(21) }, domain.ErrInvalidWIPLimit},
		// CheckWIP: đang có current việc cha in_progress, giới hạn 5.
		{"TC_KANBAN_014_dang_4_gioi_han_5", func() error { return domain.CheckWIP(4, 5) }, nil},
		{"TC_KANBAN_015_dang_5_gioi_han_5", func() error { return domain.CheckWIP(5, 5) }, domain.ErrWIPLimitReached},
		{"TC_KANBAN_016_dang_6_gioi_han_5", func() error { return domain.CheckWIP(6, 5) }, domain.ErrWIPLimitReached},
	})
}

func TestTC_KANBAN_EntersWIP_CF05(t *testing.T) {
	g := struct{ todo, ip, done domain.StatusGroup }{domain.GroupTodo, domain.GroupInProgress, domain.GroupDone}
	cases := []struct {
		tc       string
		from, to domain.StatusGroup
		want     bool
	}{
		{"TC_KANBAN_017_todo_vao_in_progress", g.todo, g.ip, true},
		{"TC_KANBAN_018_in_progress_sang_cho_nghiem_thu", g.ip, g.ip, false},
		{"TC_KANBAN_019_in_progress_sang_done", g.ip, g.done, false},
		{"TC_KANBAN_020_done_mo_lai_vao_in_progress", g.done, g.ip, true},
		{"TC_KANBAN_021_todo_sang_todo", g.todo, g.todo, false},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			if got := domain.EntersWIP(c.from, c.to); got != c.want {
				t.Fatalf("EntersWIP(%s,%s) = %v", c.from, c.to, got)
			}
		})
	}
}

func TestTC_KANBAN_BoardRoleDecision_CF04(t *testing.T) {
	mgr, other, mem := uuid.New(), uuid.New(), uuid.New()
	type out struct {
		role domain.TransitionRole
		err  error
	}
	cases := []struct {
		tc       string
		actor    uuid.UUID
		role     domain.Role
		assignee *uuid.UUID
		want     out
	}{
		{"TC_KANBAN_022_manager_phu_trach_the_cua_nguoi_khac", mgr, domain.RoleManager, &mem, out{domain.TransitionManager, nil}},
		{"TC_KANBAN_023_manager_phu_trach_the_chua_giao", mgr, domain.RoleManager, nil, out{domain.TransitionManager, nil}},
		{"TC_KANBAN_024_manager_khac_duoc_giao", other, domain.RoleManager, &other, out{domain.TransitionMember, nil}},
		{"TC_KANBAN_025_manager_khac_khong_duoc_giao", other, domain.RoleManager, &mem, out{"", domain.ErrTaskNotAssignee}},
		{"TC_KANBAN_026_member_duoc_giao", mem, domain.RoleMember, &mem, out{domain.TransitionMember, nil}},
		{"TC_KANBAN_027_member_the_nguoi_khac", mem, domain.RoleMember, &other, out{"", domain.ErrTaskNotAssignee}},
		{"TC_KANBAN_028_member_the_chua_giao", mem, domain.RoleMember, nil, out{"", domain.ErrTaskNotAssignee}},
		{"TC_KANBAN_029_ceo", uuid.New(), domain.RoleCEO, nil, out{"", domain.ErrForbidden}},
		{"TC_KANBAN_030_admin", uuid.New(), domain.RoleAdmin, nil, out{"", domain.ErrForbidden}},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			got, err := domain.BoardRole(c.actor, c.role, mgr, c.assignee)
			if c.want.err != nil {
				if !errors.Is(err, c.want.err) {
					t.Fatalf("lỗi = %v, mong đợi %v", err, c.want.err)
				}
				return
			}
			if err != nil || got != c.want.role {
				t.Fatalf("BoardRole = %q, %v; mong đợi %q", got, err, c.want.role)
			}
		})
	}
}

func TestTC_KANBAN_MoveDecision_CF04(t *testing.T) {
	// Bảng quyết định: vai trò trên bảng × nhóm nguồn × nhóm đích × cùng cột × có luật.
	const M, B = domain.TransitionManager, domain.TransitionMember
	todo, ip, done := domain.GroupTodo, domain.GroupInProgress, domain.GroupDone
	cases := []struct {
		tc   string
		m    domain.Move
		want error
	}{
		{"TC_KANBAN_031_cung_cot_member_trong_done", domain.Move{Role: B, From: done, To: done, SameColumn: true}, nil},
		{"TC_KANBAN_032_member_todo_in_progress_co_luat", domain.Move{Role: B, From: todo, To: ip, Allowed: true}, nil},
		{"TC_KANBAN_033_member_todo_in_progress_khong_luat", domain.Move{Role: B, From: todo, To: ip}, domain.ErrTransitionNotAllowed},
		{"TC_KANBAN_034_member_in_progress_done_co_luat", domain.Move{Role: B, From: ip, To: done, Allowed: true}, domain.ErrMemberTransitionToDone},
		{"TC_KANBAN_035_member_done_todo", domain.Move{Role: B, From: done, To: todo, Allowed: true}, domain.ErrMemberTransitionToDone},
		{"TC_KANBAN_036_member_done_sang_done_khac", domain.Move{Role: B, From: done, To: done, Allowed: true}, domain.ErrMemberTransitionToDone},
		{"TC_KANBAN_037_manager_in_progress_done_co_luat", domain.Move{Role: M, From: ip, To: done, Allowed: true}, nil},
		{"TC_KANBAN_038_manager_done_todo_co_luat", domain.Move{Role: M, From: done, To: todo, Allowed: true}, nil},
		{"TC_KANBAN_039_manager_khong_luat", domain.Move{Role: M, From: todo, To: done}, domain.ErrTransitionNotAllowed},
		{"TC_KANBAN_040_member_in_progress_todo_co_luat", domain.Move{Role: B, From: ip, To: todo, Allowed: true}, nil},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			got := domain.CheckMove(c.m)
			if (c.want == nil && got != nil) || (c.want != nil && !errors.Is(got, c.want)) {
				t.Fatalf("CheckMove(%+v) = %v, mong đợi %v", c.m, got, c.want)
			}
		})
	}
}

func TestTC_KANBAN_InsertAtBVA_CF04(t *testing.T) {
	a, b, x := uuid.New(), uuid.New(), uuid.New()
	cases := []struct {
		tc   string
		pos  int
		want []uuid.UUID
	}{
		{"TC_KANBAN_041_vi_tri_am", -1, []uuid.UUID{x, a, b}},
		{"TC_KANBAN_042_vi_tri_0", 0, []uuid.UUID{x, a, b}},
		{"TC_KANBAN_043_vi_tri_cuoi", 2, []uuid.UUID{a, b, x}},
		{"TC_KANBAN_044_vi_tri_vuot_cuoi", 10000, []uuid.UUID{a, b, x}},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			got := domain.InsertAt([]uuid.UUID{a, b}, x, c.pos)
			if len(got) != len(c.want) {
				t.Fatalf("InsertAt = %v", got)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("InsertAt pos=%d = %v, mong đợi %v", c.pos, got, c.want)
				}
			}
		})
	}
}

// ---------- CF-06: công việc ----------

func TestTC_TASK_FieldsBVA_CF06(t *testing.T) {
	runErrCases(t, []errCase{
		{"TC_TASK_010_tieu_de_rong", func() error { return discard(domain.NormalizeTaskTitle(" \t ")) }, domain.ErrInvalidTaskTitle},
		{"TC_TASK_011_tieu_de_1", func() error { return discard(domain.NormalizeTaskTitle("A")) }, nil},
		{"TC_TASK_012_tieu_de_200", func() error { return discard(domain.NormalizeTaskTitle(strings.Repeat("ộ", 200))) }, nil},
		{"TC_TASK_013_tieu_de_201", func() error { return discard(domain.NormalizeTaskTitle(strings.Repeat("ộ", 201))) }, domain.ErrInvalidTaskTitle},
		{"TC_TASK_014_mo_ta_5000", func() error { return discard(domain.NormalizeTaskDescription(strings.Repeat("m", 5000))) }, nil},
		{"TC_TASK_015_mo_ta_5001", func() error { return discard(domain.NormalizeTaskDescription(strings.Repeat("m", 5001))) }, domain.ErrInvalidTaskDescription},
	})
	t.Run("TC_TASK_016_tieu_de_script_giu_nguyen_van_ban", func(t *testing.T) {
		// Đoán lỗi XSS: domain lưu nguyên văn (không thực thi); việc thoát ký tự thuộc tầng hiển thị React.
		in := "<script>alert(1)</script>"
		if got, err := domain.NormalizeTaskTitle(in); err != nil || got != in {
			t.Fatalf("NormalizeTaskTitle = %q, %v", got, err)
		}
	})
}

func TestTC_TASK_StoryPointBVA_CF06(t *testing.T) {
	runErrCases(t, []errCase{
		{"TC_TASK_017_sp_0", func() error { return domain.ValidateStoryPoint(0) }, domain.ErrInvalidStoryPoint},
		{"TC_TASK_018_sp_1", func() error { return domain.ValidateStoryPoint(1) }, nil},
		{"TC_TASK_019_sp_4_ngoai_thang", func() error { return domain.ValidateStoryPoint(4) }, domain.ErrInvalidStoryPoint},
		{"TC_TASK_020_sp_21", func() error { return domain.ValidateStoryPoint(21) }, nil},
		{"TC_TASK_021_sp_22", func() error { return domain.ValidateStoryPoint(22) }, domain.ErrInvalidStoryPoint},
		{"TC_TASK_022_sp_am", func() error { return domain.ValidateStoryPoint(-1) }, domain.ErrInvalidStoryPoint},
		{"TC_TASK_023_he_so_0", func() error { return domain.ValidateStoryPointHours(0) }, domain.ErrInvalidSetting},
		{"TC_TASK_024_he_so_1", func() error { return domain.ValidateStoryPointHours(1) }, nil},
		{"TC_TASK_025_he_so_40", func() error { return domain.ValidateStoryPointHours(40) }, nil},
		{"TC_TASK_026_he_so_41", func() error { return domain.ValidateStoryPointHours(41) }, domain.ErrInvalidSetting},
	})
	t.Run("TC_TASK_027_quy_doi_gio", func(t *testing.T) {
		if got := domain.EstimateMinutes(3, 4); got != 720 {
			t.Fatalf("EstimateMinutes(3,4) = %d, mong đợi 720 (3 SP × 4 giờ × 60)", got)
		}
	})
}

func TestTC_TASK_LabelsBVA_CF06(t *testing.T) {
	ids := func(n int) []uuid.UUID {
		out := make([]uuid.UUID, n)
		for i := range out {
			out[i] = uuid.New()
		}
		return out
	}
	dup := ids(10)
	dup = append(dup, dup[0]) // 11 phần tử nhưng chỉ 10 nhãn khác nhau
	runErrCases(t, []errCase{
		{"TC_TASK_028_0_nhan", func() error { return discard(domain.NormalizeLabelIDs(nil)) }, nil},
		{"TC_TASK_029_10_nhan", func() error { return discard(domain.NormalizeLabelIDs(ids(10))) }, nil},
		{"TC_TASK_030_11_nhan", func() error { return discard(domain.NormalizeLabelIDs(ids(11))) }, domain.ErrTooManyLabels},
		{"TC_TASK_031_11_phan_tu_trung_con_10", func() error { return discard(domain.NormalizeLabelIDs(dup)) }, nil},
		{"TC_TASK_032_nhan_nil_uuid", func() error { return discard(domain.NormalizeLabelIDs([]uuid.UUID{uuid.Nil})) }, domain.ErrLabelNotFound},
	})
}

func TestTC_TASK_AssigneeDecision_CF06(t *testing.T) {
	cases := []struct {
		tc     string
		role   domain.Role
		status domain.UserStatus
		want   bool
	}{
		{"TC_TASK_033_manager_active", domain.RoleManager, domain.UserStatusActive, true},
		{"TC_TASK_034_member_active", domain.RoleMember, domain.UserStatusActive, true},
		{"TC_TASK_035_member_disabled", domain.RoleMember, domain.UserStatusDisabled, false},
		{"TC_TASK_036_admin_active", domain.RoleAdmin, domain.UserStatusActive, false},
		{"TC_TASK_037_ceo_active", domain.RoleCEO, domain.UserStatusActive, false},
	}
	for _, c := range cases {
		t.Run(c.tc, func(t *testing.T) {
			if got := domain.CanBeAssigned(c.role, c.status); got != c.want {
				t.Fatalf("CanBeAssigned(%s,%s) = %v", c.role, c.status, got)
			}
		})
	}
}
