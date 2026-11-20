package org_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/org"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

var errBoom = errors.New("boom")

// fakeStore giữ dữ liệu trong bộ nhớ; ràng buộc trùng tên mô phỏng chỉ mục unique của CSDL.
type fakeStore struct {
	depts  map[uuid.UUID]domain.Department
	titles map[uuid.UUID]domain.JobTitle
	skills map[uuid.UUID]domain.Skill
	err    error
	calls  int
}

func newStore() *fakeStore {
	return &fakeStore{
		depts:  map[uuid.UUID]domain.Department{},
		titles: map[uuid.UUID]domain.JobTitle{},
		skills: map[uuid.UUID]domain.Skill{},
	}
}

func sameParent(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func (f *fakeStore) ListDepartments(context.Context) ([]domain.Department, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	out := []domain.Department{}
	for _, d := range f.depts {
		out = append(out, d)
	}
	return out, nil
}

func (f *fakeStore) CreateDepartment(_ context.Context, name string, parent *uuid.UUID) (uuid.UUID, error) {
	f.calls++
	for _, d := range f.depts {
		if strings.EqualFold(d.Name, name) && sameParent(d.ParentID, parent) {
			return uuid.Nil, domain.ErrDepartmentNameTaken
		}
	}
	id := uuid.New()
	f.depts[id] = domain.Department{ID: id, Name: name, ParentID: parent}
	return id, nil
}

func (f *fakeStore) UpdateDepartment(_ context.Context, id uuid.UUID, name string, parent *uuid.UUID) error {
	f.calls++
	d, ok := f.depts[id]
	if !ok {
		return domain.ErrDepartmentNotFound
	}
	d.Name, d.ParentID = name, parent
	f.depts[id] = d
	return nil
}

func (f *fakeStore) DeleteDepartment(_ context.Context, id uuid.UUID) error {
	f.calls++
	if _, ok := f.depts[id]; !ok {
		return domain.ErrDepartmentNotFound
	}
	delete(f.depts, id)
	return nil
}

func (f *fakeStore) ListJobTitles(context.Context) ([]domain.JobTitle, error) {
	f.calls++
	return nil, f.err
}

func (f *fakeStore) CreateJobTitle(_ context.Context, name string) (uuid.UUID, error) {
	f.calls++
	for _, x := range f.titles {
		if strings.EqualFold(x.Name, name) {
			return uuid.Nil, domain.ErrJobTitleNameTaken
		}
	}
	id := uuid.New()
	f.titles[id] = domain.JobTitle{ID: id, Name: name}
	return id, nil
}

func (f *fakeStore) UpdateJobTitle(_ context.Context, id uuid.UUID, name string) error {
	f.calls++
	if _, ok := f.titles[id]; !ok {
		return domain.ErrJobTitleNotFound
	}
	f.titles[id] = domain.JobTitle{ID: id, Name: name}
	return nil
}

func (f *fakeStore) DeleteJobTitle(_ context.Context, id uuid.UUID) error {
	f.calls++
	if _, ok := f.titles[id]; !ok {
		return domain.ErrJobTitleNotFound
	}
	delete(f.titles, id)
	return nil
}

func (f *fakeStore) ListSkills(context.Context) ([]domain.Skill, error) {
	f.calls++
	return nil, f.err
}

func (f *fakeStore) CreateSkill(_ context.Context, name string) (uuid.UUID, error) {
	f.calls++
	id := uuid.New()
	f.skills[id] = domain.Skill{ID: id, Name: name}
	return id, nil
}

func (f *fakeStore) UpdateSkill(_ context.Context, id uuid.UUID, name string) error {
	f.calls++
	if _, ok := f.skills[id]; !ok {
		return domain.ErrSkillNotFound
	}
	f.skills[id] = domain.Skill{ID: id, Name: name}
	return nil
}

func (f *fakeStore) DeleteSkill(_ context.Context, id uuid.UUID) error {
	f.calls++
	if _, ok := f.skills[id]; !ok {
		return domain.ErrSkillNotFound
	}
	delete(f.skills, id)
	return nil
}

var (
	admin   = authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin}
	ceo     = authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO}
	manager = authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	member  = authz.Principal{UserID: uuid.New(), Role: domain.RoleMember}
)

func TestService_OnlyAdminAndCEOManageOrg_CF02(t *testing.T) {
	ctx := context.Background()
	id := uuid.New()
	calls := map[string]func(*org.Service, authz.Principal) error{
		"ListDepartments": func(s *org.Service, p authz.Principal) error { _, err := s.ListDepartments(ctx, p); return err },
		"CreateDepartment": func(s *org.Service, p authz.Principal) error {
			_, err := s.CreateDepartment(ctx, p, "Phòng A", nil)
			return err
		},
		"UpdateDepartment": func(s *org.Service, p authz.Principal) error {
			_, err := s.UpdateDepartment(ctx, p, id, "Phòng A", nil)
			return err
		},
		"DeleteDepartment": func(s *org.Service, p authz.Principal) error { return s.DeleteDepartment(ctx, p, id) },
		"ListJobTitles":    func(s *org.Service, p authz.Principal) error { _, err := s.ListJobTitles(ctx, p); return err },
		"CreateJobTitle": func(s *org.Service, p authz.Principal) error {
			_, err := s.CreateJobTitle(ctx, p, "Tester")
			return err
		},
		"UpdateJobTitle": func(s *org.Service, p authz.Principal) error {
			_, err := s.UpdateJobTitle(ctx, p, id, "Tester")
			return err
		},
		"DeleteJobTitle": func(s *org.Service, p authz.Principal) error { return s.DeleteJobTitle(ctx, p, id) },
		"ListSkills":     func(s *org.Service, p authz.Principal) error { _, err := s.ListSkills(ctx, p); return err },
		"CreateSkill":    func(s *org.Service, p authz.Principal) error { _, err := s.CreateSkill(ctx, p, "Go"); return err },
		"UpdateSkill":    func(s *org.Service, p authz.Principal) error { _, err := s.UpdateSkill(ctx, p, id, "Go"); return err },
		"DeleteSkill":    func(s *org.Service, p authz.Principal) error { return s.DeleteSkill(ctx, p, id) },
	}
	for name, call := range calls {
		for _, p := range []authz.Principal{manager, member, {}} {
			t.Run(name+"/"+string(p.Role), func(t *testing.T) {
				st := newStore()
				if err := call(org.NewService(st), p); !errors.Is(err, domain.ErrForbidden) {
					t.Fatalf("err = %v, want ErrForbidden", err)
				}
				if st.calls != 0 {
					t.Fatalf("store được gọi %d lần dù không có quyền", st.calls)
				}
			})
		}
		for _, p := range []authz.Principal{admin, ceo} {
			t.Run(name+"/"+string(p.Role), func(t *testing.T) {
				if err := call(org.NewService(newStore()), p); errors.Is(err, domain.ErrForbidden) {
					t.Fatalf("%s bị từ chối", p.Role)
				}
			})
		}
	}
}

func TestCreateDepartment_NormalizesName_CF02(t *testing.T) {
	st := newStore()
	d, err := org.NewService(st).CreateDepartment(context.Background(), admin, "  Phòng   Kỹ thuật ", nil)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "Phòng Kỹ thuật" || d.ParentID != nil || d.ID == uuid.Nil {
		t.Fatalf("department = %+v", d)
	}
	if _, err := org.NewService(st).CreateDepartment(context.Background(), admin, "   ", nil); !errors.Is(err, domain.ErrInvalidOrgName) {
		t.Fatalf("tên rỗng: err = %v", err)
	}
}

func TestCreateDepartment_UnknownParent_CF02(t *testing.T) {
	st := newStore()
	ghost := uuid.New()
	_, err := org.NewService(st).CreateDepartment(context.Background(), admin, "Nhóm Web", &ghost)
	if !errors.Is(err, domain.ErrDepartmentNotFound) {
		t.Fatalf("err = %v, want ErrDepartmentNotFound", err)
	}
}

func TestCreateDepartment_DuplicateUnderSameParent_CF02(t *testing.T) {
	st := newStore()
	s := org.NewService(st)
	ctx := context.Background()
	root, _ := s.CreateDepartment(ctx, admin, "Kỹ thuật", nil)
	if _, err := s.CreateDepartment(ctx, admin, "Nhóm Web", &root.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateDepartment(ctx, admin, "nhóm web", &root.ID); !errors.Is(err, domain.ErrDepartmentNameTaken) {
		t.Fatalf("err = %v, want ErrDepartmentNameTaken", err)
	}
	if _, err := s.CreateDepartment(ctx, admin, "Nhóm Web", nil); err != nil {
		t.Fatalf("cùng tên khác cha phải được phép: %v", err)
	}
}

func TestUpdateDepartment_RejectsCycle_CF02(t *testing.T) {
	st := newStore()
	s := org.NewService(st)
	ctx := context.Background()
	root, _ := s.CreateDepartment(ctx, admin, "Kỹ thuật", nil)
	child, _ := s.CreateDepartment(ctx, admin, "Nhóm Web", &root.ID)

	if _, err := s.UpdateDepartment(ctx, admin, root.ID, "Kỹ thuật", &child.ID); !errors.Is(err, domain.ErrDepartmentCycle) {
		t.Fatalf("đặt cha là con: err = %v, want ErrDepartmentCycle", err)
	}
	if _, err := s.UpdateDepartment(ctx, admin, root.ID, "Kỹ thuật", &root.ID); !errors.Is(err, domain.ErrDepartmentCycle) {
		t.Fatalf("đặt cha là chính nó: err = %v", err)
	}
	if got := st.depts[root.ID].ParentID; got != nil {
		t.Fatalf("store bị ghi dù vòng: parent = %v", got)
	}
	d, err := s.UpdateDepartment(ctx, admin, child.ID, "Nhóm Frontend", nil)
	if err != nil || d.Name != "Nhóm Frontend" || d.ParentID != nil {
		t.Fatalf("chuyển con lên gốc: %+v, %v", d, err)
	}
}

func TestUpdateDepartment_NotFound_CF02(t *testing.T) {
	_, err := org.NewService(newStore()).UpdateDepartment(context.Background(), admin, uuid.New(), "X", nil)
	if !errors.Is(err, domain.ErrDepartmentNotFound) {
		t.Fatalf("err = %v, want ErrDepartmentNotFound", err)
	}
}

func TestJobTitleAndSkill_ValidateName_CF02(t *testing.T) {
	ctx := context.Background()
	s := org.NewService(newStore())
	long := strings.Repeat("a", domain.OrgNameMaxRunes+1)
	if _, err := s.CreateJobTitle(ctx, admin, long); !errors.Is(err, domain.ErrInvalidOrgName) {
		t.Fatalf("CreateJobTitle quá dài: %v", err)
	}
	if _, err := s.UpdateSkill(ctx, admin, uuid.New(), ""); !errors.Is(err, domain.ErrInvalidOrgName) {
		t.Fatalf("UpdateSkill rỗng: %v", err)
	}
	jt, err := s.CreateJobTitle(ctx, admin, " Lập trình viên ")
	if err != nil || jt.Name != "Lập trình viên" {
		t.Fatalf("CreateJobTitle = %+v, %v", jt, err)
	}
	if _, err := s.CreateJobTitle(ctx, admin, "lập trình viên"); !errors.Is(err, domain.ErrJobTitleNameTaken) {
		t.Fatalf("trùng tên: %v", err)
	}
	sk, err := s.CreateSkill(ctx, ceo, "Go")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := s.UpdateSkill(ctx, admin, sk.ID, "Golang"); err != nil || got.Name != "Golang" {
		t.Fatalf("UpdateSkill = %+v, %v", got, err)
	}
	if err := s.DeleteSkill(ctx, admin, sk.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSkill(ctx, admin, sk.ID); !errors.Is(err, domain.ErrSkillNotFound) {
		t.Fatalf("xóa lần 2: %v", err)
	}
}

func TestList_EmptyIsNotNil_CF02(t *testing.T) {
	ctx := context.Background()
	s := org.NewService(newStore())
	jts, err := s.ListJobTitles(ctx, admin)
	if err != nil || jts == nil {
		t.Fatalf("ListJobTitles = %v, %v", jts, err)
	}
	sks, err := s.ListSkills(ctx, admin)
	if err != nil || sks == nil {
		t.Fatalf("ListSkills = %v, %v", sks, err)
	}
}

func TestList_WrapsStoreError_CF02(t *testing.T) {
	st := newStore()
	st.err = errBoom
	if _, err := org.NewService(st).ListDepartments(context.Background(), admin); !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want wrap errBoom", err)
	}
}
