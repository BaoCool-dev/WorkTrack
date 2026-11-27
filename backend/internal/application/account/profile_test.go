package account_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func seedUser(st *fakeStore, role domain.Role) uuid.UUID {
	id := uuid.New()
	st.users[id] = domain.User{
		ID: id, Email: "an.nguyen@worktrack.local", FullName: "Nguyễn Văn An",
		Role: role, Status: domain.UserStatusActive,
	}
	return id
}

func TestCreateUser_WithDepartmentAndJobTitle_CF02(t *testing.T) {
	svc, st, _ := newSvc()
	dep, jt := uuid.New(), uuid.New()
	in := validInput(domain.RoleMember)
	in.DepartmentID, in.JobTitleID = &dep, &jt
	res, err := svc.CreateUser(context.Background(), actor(domain.RoleAdmin), in)
	if err != nil {
		t.Fatal(err)
	}
	if *st.created[0].DepartmentID != dep || *st.created[0].JobTitleID != jt {
		t.Fatalf("NewUser = %+v", st.created[0])
	}
	if *res.User.DepartmentID != dep || *res.User.JobTitleID != jt {
		t.Fatalf("User = %+v", res.User)
	}
}

func TestListUsers_PassesFilterAndLoadsSkillsOnce_CF02(t *testing.T) {
	svc, st, _ := newSvc()
	a, b := seedUser(st, domain.RoleMember), seedUser(st, domain.RoleManager)
	st.listOut = []domain.User{st.users[a], st.users[b]}
	st.skills[a] = []domain.UserSkill{{SkillID: uuid.New(), Name: "Go", Level: domain.SkillSenior}}
	role := domain.RoleMember
	f := account.UserFilter{Search: "an", Role: &role}
	got, err := svc.ListUsers(context.Background(), actor(domain.RoleAdmin), f, domain.NewPageRequest(1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if st.filter.Search != "an" || *st.filter.Role != domain.RoleMember {
		t.Fatalf("filter = %+v", st.filter)
	}
	if st.skillCalls != 1 {
		t.Fatalf("đọc kỹ năng %d lần, muốn 1 (không N+1)", st.skillCalls)
	}
	if len(got.Items[0].Skills) != 1 || got.Items[1].Skills == nil || len(got.Items[1].Skills) != 0 {
		t.Fatalf("skills = %+v / %+v", got.Items[0].Skills, got.Items[1].Skills)
	}
}

func TestListUsers_TrimsSearch_CF02(t *testing.T) {
	svc, st, _ := newSvc()
	_, err := svc.ListUsers(context.Background(), actor(domain.RoleAdmin), account.UserFilter{Search: "  chi  "}, domain.NewPageRequest(1, 20))
	if err != nil || st.filter.Search != "chi" {
		t.Fatalf("search = %q, err %v", st.filter.Search, err)
	}
	if _, err := svc.ListUsers(context.Background(), actor(domain.RoleAdmin), account.UserFilter{Search: strings.Repeat("a", 101)}, domain.NewPageRequest(1, 20)); !errors.Is(err, domain.ErrInvalidSearch) {
		t.Fatalf("search quá dài: %v", err)
	}
}

func TestGetUser_IncludesSkills_CF02(t *testing.T) {
	svc, st, _ := newSvc()
	id := seedUser(st, domain.RoleMember)
	st.skills[id] = []domain.UserSkill{{SkillID: uuid.New(), Name: "React", Level: domain.SkillBeginner}}
	u, err := svc.GetUser(context.Background(), actor(domain.RoleAdmin), id)
	if err != nil || len(u.Skills) != 1 {
		t.Fatalf("u = %+v, err %v", u, err)
	}
	if _, err := svc.GetUser(context.Background(), actor(domain.RoleMember), id); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("member: %v", err)
	}
	if _, err := svc.GetUser(context.Background(), actor(domain.RoleAdmin), uuid.New()); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("not found: %v", err)
	}
}

func TestUpdateUser_UpdatesProfile_CF02(t *testing.T) {
	svc, st, _ := newSvc()
	id := seedUser(st, domain.RoleMember)
	dep := uuid.New()
	u, err := svc.UpdateUser(context.Background(), actor(domain.RoleAdmin), id, account.UpdateUserInput{
		FullName: "  Nguyễn Văn Bình ", Role: domain.RoleManager, DepartmentID: &dep,
	})
	if err != nil {
		t.Fatal(err)
	}
	if u.FullName != "Nguyễn Văn Bình" || u.Role != domain.RoleManager || *u.DepartmentID != dep || u.JobTitleID != nil {
		t.Fatalf("u = %+v", u)
	}
	if u.Email != "an.nguyen@worktrack.local" {
		t.Fatal("email không đổi khi sửa (chỉ đọc)")
	}
}

func TestUpdateUser_Rules_CF02(t *testing.T) {
	ctx := context.Background()
	in := func(r domain.Role) account.UpdateUserInput {
		return account.UpdateUserInput{FullName: "Trần Thị Dung", Role: r}
	}
	t.Run("manager bị từ chối", func(t *testing.T) {
		svc, st, _ := newSvc()
		id := seedUser(st, domain.RoleMember)
		if _, err := svc.UpdateUser(ctx, actor(domain.RoleManager), id, in(domain.RoleMember)); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("admin không nâng ai lên CEO", func(t *testing.T) {
		svc, st, _ := newSvc()
		id := seedUser(st, domain.RoleMember)
		if _, err := svc.UpdateUser(ctx, actor(domain.RoleAdmin), id, in(domain.RoleCEO)); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("err = %v", err)
		}
		if len(st.updates) != 0 {
			t.Fatal("không được ghi")
		}
	})
	t.Run("admin không sửa tài khoản CEO", func(t *testing.T) {
		svc, st, _ := newSvc()
		id := seedUser(st, domain.RoleCEO)
		if _, err := svc.UpdateUser(ctx, actor(domain.RoleAdmin), id, in(domain.RoleCEO)); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("CEO được đổi vai trò CEO", func(t *testing.T) {
		svc, st, _ := newSvc()
		id := seedUser(st, domain.RoleMember)
		if _, err := svc.UpdateUser(ctx, actor(domain.RoleCEO), id, in(domain.RoleCEO)); err != nil {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("không tự đổi vai trò của mình", func(t *testing.T) {
		svc, st, _ := newSvc()
		id := seedUser(st, domain.RoleAdmin)
		me := actor(domain.RoleAdmin)
		me.UserID = id
		if _, err := svc.UpdateUser(ctx, me, id, in(domain.RoleMember)); !errors.Is(err, domain.ErrCannotChangeOwnRole) {
			t.Fatalf("err = %v", err)
		}
		if _, err := svc.UpdateUser(ctx, me, id, in(domain.RoleAdmin)); err != nil {
			t.Fatalf("tự sửa tên (giữ vai trò) phải được: %v", err)
		}
	})
	t.Run("validate", func(t *testing.T) {
		svc, st, _ := newSvc()
		id := seedUser(st, domain.RoleMember)
		if _, err := svc.UpdateUser(ctx, actor(domain.RoleAdmin), id, in("boss")); !errors.Is(err, domain.ErrInvalidRole) {
			t.Fatalf("role: %v", err)
		}
		bad := in(domain.RoleMember)
		bad.FullName = " "
		if _, err := svc.UpdateUser(ctx, actor(domain.RoleAdmin), id, bad); !errors.Is(err, domain.ErrInvalidFullName) {
			t.Fatalf("name: %v", err)
		}
		if _, err := svc.UpdateUser(ctx, actor(domain.RoleAdmin), uuid.New(), in(domain.RoleMember)); !errors.Is(err, domain.ErrUserNotFound) {
			t.Fatalf("not found: %v", err)
		}
	})
}

func TestSetUserSkills_CF02(t *testing.T) {
	ctx := context.Background()
	svc, st, _ := newSvc()
	id := seedUser(st, domain.RoleMember)
	goID := uuid.New()
	got, err := svc.SetUserSkills(ctx, actor(domain.RoleAdmin), id, []domain.UserSkillInput{{SkillID: goID, Level: "senior"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(st.skills[id]) != 1 || st.skills[id][0].Level != domain.SkillSenior || len(got) != 1 {
		t.Fatalf("stored = %+v, got = %+v", st.skills[id], got)
	}
	if _, err := svc.SetUserSkills(ctx, actor(domain.RoleAdmin), id, []domain.UserSkillInput{{SkillID: goID, Level: "expert"}}); !errors.Is(err, domain.ErrInvalidSkillLevel) {
		t.Fatalf("level: %v", err)
	}
	if _, err := svc.SetUserSkills(ctx, actor(domain.RoleMember), id, nil); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("member: %v", err)
	}
	if _, err := svc.SetUserSkills(ctx, actor(domain.RoleAdmin), uuid.New(), nil); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("not found: %v", err)
	}
	got, err = svc.SetUserSkills(ctx, actor(domain.RoleAdmin), id, nil)
	if err != nil || got == nil || len(st.skills[id]) != 0 {
		t.Fatalf("xóa hết kỹ năng: got=%v err=%v", got, err)
	}
	ceoID := seedUser(st, domain.RoleCEO)
	if _, err := svc.SetUserSkills(ctx, actor(domain.RoleAdmin), ceoID, nil); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("admin sửa hồ sơ CEO: %v", err)
	}
}
