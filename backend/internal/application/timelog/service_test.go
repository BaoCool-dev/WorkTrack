package timelog_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/timelog"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

var now = time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC) // 10:00 giờ Việt Nam

type fakeProjects struct {
	pid     uuid.UUID
	manager uuid.UUID
	members map[uuid.UUID]bool
}

func (f *fakeProjects) Get(_ context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error) {
	if id != f.pid || (actor.Role != domain.RoleCEO && !f.members[actor.UserID]) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return domain.Project{ID: id, ManagerID: f.manager}, nil
}

type fakeStore struct {
	tasks   map[uuid.UUID]domain.Task
	doneAt  map[uuid.UUID]time.Time
	logs    []domain.TimeLog
	locks   []string
	failNew bool
}

func (f *fakeStore) InTx(_ context.Context, fn func(timelog.Store) error) error {
	saved := append([]domain.TimeLog(nil), f.logs...)
	if err := fn(f); err != nil {
		f.logs = saved
		return err
	}
	return nil
}

func (f *fakeStore) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return t, nil
}

func (f *fakeStore) LockProject(_ context.Context, _ uuid.UUID) (domain.LockedProject, error) {
	f.locks = append(f.locks, "project")
	return domain.LockedProject{}, nil
}

func (f *fakeStore) Lock(_ context.Context, id uuid.UUID) error {
	f.locks = append(f.locks, "task")
	if _, ok := f.tasks[id]; !ok {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (f *fakeStore) LastDoneAt(_ context.Context, id uuid.UUID) (*time.Time, error) {
	if t, ok := f.doneAt[id]; ok {
		return &t, nil
	}
	return nil, nil
}

func (f *fakeStore) Create(_ context.Context, l timelog.NewLog) (uuid.UUID, error) {
	if f.failNew {
		return uuid.Nil, errors.New("db down")
	}
	t := f.tasks[l.TaskID]
	f.logs = append(f.logs, domain.TimeLog{
		ID: uuid.New(), TaskID: l.TaskID, ParentID: t.ParentID, ProjectID: l.ProjectID,
		UserID: l.UserID, Minutes: l.Minutes, Note: l.Note, WorkDate: l.WorkDate, CreatedAt: now,
	})
	return f.logs[len(f.logs)-1].ID, nil
}

func (f *fakeStore) GetLog(_ context.Context, id uuid.UUID) (domain.TimeLog, error) {
	for _, l := range f.logs {
		if l.ID == id {
			return l, nil
		}
	}
	return domain.TimeLog{}, domain.ErrTimeLogNotFound
}

func (f *fakeStore) ListLogs(_ context.Context, id uuid.UUID, _ domain.PageRequest) ([]domain.TimeLog, int, error) {
	var out []domain.TimeLog
	for _, l := range f.logs {
		if l.TaskID == id || (l.ParentID != nil && *l.ParentID == id) {
			out = append(out, l)
		}
	}
	return out, len(out), nil
}

func (f *fakeStore) DeleteLog(_ context.Context, id uuid.UUID) error {
	for i, l := range f.logs {
		if l.ID == id {
			f.logs = append(f.logs[:i], f.logs[i+1:]...)
			return nil
		}
	}
	return domain.ErrTimeLogNotFound
}

type fakePub struct{ events []domain.BoardEvent }

func (p *fakePub) Publish(_ context.Context, e domain.BoardEvent) { p.events = append(p.events, e) }

type fixture struct {
	svc                             *timelog.Service
	st                              *fakeStore
	pub                             *fakePub
	pm, assignee, other, outsider   authz.Principal
	ceo                             authz.Principal
	parentID, childID, otherChildID uuid.UUID
}

func setup() fixture {
	pid := uuid.New()
	f := fixture{
		pm:       authz.Principal{UserID: uuid.New(), Role: domain.RoleManager},
		assignee: authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		other:    authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		outsider: authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		ceo:      authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO},
		parentID: uuid.New(), childID: uuid.New(), otherChildID: uuid.New(),
	}
	created := now.Add(-72 * time.Hour)
	a := f.assignee.UserID
	parent := domain.Task{ID: f.parentID, ProjectID: pid, StatusGroup: domain.GroupInProgress, AssigneeID: &a, CreatedAt: created, Version: 3}
	// Con có bản sao người nhận khác (cha vừa được giao lại): quyền phải theo người nhận của cha.
	o := f.other.UserID
	child := domain.Task{ID: f.childID, ProjectID: pid, ParentID: &f.parentID, StatusGroup: domain.GroupTodo, AssigneeID: &o, CreatedAt: created}
	f.st = &fakeStore{
		tasks:  map[uuid.UUID]domain.Task{f.parentID: parent, f.childID: child},
		doneAt: map[uuid.UUID]time.Time{},
	}
	f.pub = &fakePub{}
	projects := &fakeProjects{pid: pid, manager: f.pm.UserID, members: map[uuid.UUID]bool{
		f.pm.UserID: true, f.assignee.UserID: true, f.other.UserID: true,
	}}
	f.svc = timelog.NewService(f.st, f.st, projects, f.pub, func() time.Time { return now })
	return f
}

func (f fixture) setGroup(id uuid.UUID, g domain.StatusGroup, doneAt time.Time) {
	t := f.st.tasks[id]
	t.StatusGroup = g
	f.st.tasks[id] = t
	f.st.doneAt[id] = doneAt
}

func TestCreatePermissions_CF08(t *testing.T) {
	cases := []struct {
		name string
		who  func(fixture) authz.Principal
		task func(fixture) uuid.UUID
		want error
	}{
		{"người nhận ghi việc cha", func(f fixture) authz.Principal { return f.assignee }, func(f fixture) uuid.UUID { return f.parentID }, nil},
		{"người nhận cha ghi việc con", func(f fixture) authz.Principal { return f.assignee }, func(f fixture) uuid.UUID { return f.childID }, nil},
		{"Manager phụ trách", func(f fixture) authz.Principal { return f.pm }, func(f fixture) uuid.UUID { return f.parentID }, nil},
		{"thành viên khác (kể cả bản sao người nhận trên con)", func(f fixture) authz.Principal { return f.other }, func(f fixture) uuid.UUID { return f.childID }, domain.ErrTimeLogNotAllowed},
		{"CEO chỉ xem", func(f fixture) authz.Principal { return f.ceo }, func(f fixture) uuid.UUID { return f.parentID }, domain.ErrForbidden},
		{"ngoài dự án → 404", func(f fixture) authz.Principal { return f.outsider }, func(f fixture) uuid.UUID { return f.parentID }, domain.ErrTaskNotFound},
		{"không tồn tại", func(f fixture) authz.Principal { return f.assignee }, func(fixture) uuid.UUID { return uuid.New() }, domain.ErrTaskNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := setup()
			l, err := f.svc.Create(context.Background(), c.who(f), c.task(f), timelog.Input{Minutes: 90, Note: "  Viết API  "})
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
			if c.want != nil {
				if len(f.st.logs) != 0 || len(f.pub.events) != 0 {
					t.Fatalf("không được ghi/phát sự kiện khi lỗi")
				}
				return
			}
			if l.Minutes != 90 || l.Note != "Viết API" || l.UserID != c.who(f).UserID {
				t.Fatalf("log = %+v", l)
			}
			if want := time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC); !l.WorkDate.Equal(want) {
				t.Fatalf("workDate = %v, want hôm nay giờ VN", l.WorkDate)
			}
		})
	}
}

func TestCreatePublishesRootTask_CF08(t *testing.T) {
	f := setup()
	if _, err := f.svc.Create(context.Background(), f.assignee, f.childID, timelog.Input{Minutes: 30, Note: "x"}); err != nil {
		t.Fatal(err)
	}
	if len(f.pub.events) != 1 {
		t.Fatalf("events = %d", len(f.pub.events))
	}
	e := f.pub.events[0]
	if e.Type != domain.EventTaskUpdated || e.TaskID != f.parentID || e.ActorID != f.assignee.UserID || e.Version != 3 {
		t.Fatalf("event = %+v, muốn task.updated cho việc CHA", e)
	}
	if len(f.st.locks) < 2 || f.st.locks[0] != "project" {
		t.Fatalf("phải khóa dự án trước công việc: %v", f.st.locks)
	}
}

func TestCreateValidation_CF08(t *testing.T) {
	future := now.Add(48 * time.Hour)
	beforeCreate := now.Add(-10 * 24 * time.Hour)
	cases := []struct {
		name string
		in   timelog.Input
		want error
	}{
		{"0 phút", timelog.Input{Minutes: 0, Note: "x"}, domain.ErrInvalidLogMinutes},
		{"âm", timelog.Input{Minutes: -5, Note: "x"}, domain.ErrInvalidLogMinutes},
		{"quá 1440", timelog.Input{Minutes: 1441, Note: "x"}, domain.ErrInvalidLogMinutes},
		{"mô tả trống", timelog.Input{Minutes: 10, Note: "   "}, domain.ErrInvalidLogNote},
		{"ngày tương lai", timelog.Input{Minutes: 10, Note: "x", WorkDate: &future}, domain.ErrInvalidLogDate},
		{"trước ngày tạo việc", timelog.Input{Minutes: 10, Note: "x", WorkDate: &beforeCreate}, domain.ErrInvalidLogDate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := setup()
			if _, err := f.svc.Create(context.Background(), f.assignee, f.parentID, c.in); !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestCreateDoneWindow_CF08(t *testing.T) {
	cases := []struct {
		name       string
		task       string // "parent" | "child"
		closeWho   string // việc nào đã Done
		doneBefore time.Duration
		want       error
	}{
		{"cha Done 23 giờ → được", "parent", "parent", 23 * time.Hour, nil},
		{"cha Done đúng 24 giờ → được", "parent", "parent", 24 * time.Hour, nil},
		{"cha Done 25 giờ → chặn", "parent", "parent", 25 * time.Hour, domain.ErrTimeLogClosed},
		{"con Done 25 giờ → chặn", "child", "child", 25 * time.Hour, domain.ErrTimeLogClosed},
		{"log vào con khi cha Done 25 giờ → chặn", "child", "parent", 25 * time.Hour, domain.ErrTimeLogClosed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := setup()
			ids := map[string]uuid.UUID{"parent": f.parentID, "child": f.childID}
			f.setGroup(ids[c.closeWho], domain.GroupDone, now.Add(-c.doneBefore))
			_, err := f.svc.Create(context.Background(), f.assignee, ids[c.task], timelog.Input{Minutes: 10, Note: "x"})
			if !errors.Is(err, c.want) {
				t.Fatalf("err = %v, want %v", err, c.want)
			}
		})
	}
}

func TestCreateDoneWithoutHistoryUsesUpdatedAt_CF08(t *testing.T) {
	f := setup()
	p := f.st.tasks[f.parentID]
	p.StatusGroup = domain.GroupDone
	p.UpdatedAt = now.Add(-30 * time.Hour)
	f.st.tasks[f.parentID] = p
	if _, err := f.svc.Create(context.Background(), f.assignee, f.parentID, timelog.Input{Minutes: 10, Note: "x"}); !errors.Is(err, domain.ErrTimeLogClosed) {
		t.Fatalf("err = %v, want ErrTimeLogClosed", err)
	}
}

func TestCreateRollbackOnStoreError_CF08(t *testing.T) {
	f := setup()
	f.st.failNew = true
	if _, err := f.svc.Create(context.Background(), f.assignee, f.parentID, timelog.Input{Minutes: 10, Note: "x"}); err == nil {
		t.Fatal("muốn lỗi")
	}
	if len(f.pub.events) != 0 {
		t.Fatal("không phát sự kiện khi giao dịch lỗi")
	}
}

func TestListIncludesChildren_CF08(t *testing.T) {
	f := setup()
	ctx := context.Background()
	for _, id := range []uuid.UUID{f.parentID, f.childID} {
		if _, err := f.svc.Create(ctx, f.assignee, id, timelog.Input{Minutes: 15, Note: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := f.svc.List(ctx, f.ceo, f.parentID, domain.PageRequest{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("page = %+v", page)
	}
	if _, err := f.svc.List(ctx, f.outsider, f.parentID, domain.PageRequest{Page: 1, PageSize: 20}); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("outsider err = %v", err)
	}
	empty, err := f.svc.List(ctx, f.other, f.childID, domain.PageRequest{Page: 1, PageSize: 20})
	if err != nil || empty.Items == nil {
		t.Fatalf("items phải là slice rỗng không nil: %v %v", empty.Items, err)
	}
}

func TestDelete_CF08(t *testing.T) {
	ctx := context.Background()
	t.Run("tác giả xóa được, phát sự kiện cho cha", func(t *testing.T) {
		f := setup()
		l, _ := f.svc.Create(ctx, f.assignee, f.childID, timelog.Input{Minutes: 10, Note: "x"})
		f.pub.events = nil
		if err := f.svc.Delete(ctx, f.assignee, l.ID); err != nil {
			t.Fatal(err)
		}
		if len(f.st.logs) != 0 || len(f.pub.events) != 1 || f.pub.events[0].TaskID != f.parentID {
			t.Fatalf("logs=%d events=%+v", len(f.st.logs), f.pub.events)
		}
	})
	t.Run("người khác (kể cả Manager) không xóa", func(t *testing.T) {
		f := setup()
		l, _ := f.svc.Create(ctx, f.assignee, f.parentID, timelog.Input{Minutes: 10, Note: "x"})
		if err := f.svc.Delete(ctx, f.pm, l.ID); !errors.Is(err, domain.ErrTimeLogNotAllowed) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("ngoài dự án → 404", func(t *testing.T) {
		f := setup()
		l, _ := f.svc.Create(ctx, f.assignee, f.parentID, timelog.Input{Minutes: 10, Note: "x"})
		if err := f.svc.Delete(ctx, f.outsider, l.ID); !errors.Is(err, domain.ErrTimeLogNotFound) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("đã Done quá 24 giờ → khóa", func(t *testing.T) {
		f := setup()
		l, _ := f.svc.Create(ctx, f.assignee, f.parentID, timelog.Input{Minutes: 10, Note: "x"})
		f.setGroup(f.parentID, domain.GroupDone, now.Add(-25*time.Hour))
		if err := f.svc.Delete(ctx, f.assignee, l.ID); !errors.Is(err, domain.ErrTimeLogClosed) {
			t.Fatalf("err = %v", err)
		}
		if len(f.st.logs) != 1 {
			t.Fatal("không được xóa")
		}
	})
	t.Run("không tồn tại", func(t *testing.T) {
		f := setup()
		if err := f.svc.Delete(ctx, f.assignee, uuid.New()); !errors.Is(err, domain.ErrTimeLogNotFound) {
			t.Fatalf("err = %v", err)
		}
	})
}
