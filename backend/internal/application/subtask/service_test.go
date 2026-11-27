package subtask_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/subtask"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

type fakeProjects struct {
	projects map[uuid.UUID]domain.Project
	members  map[uuid.UUID]bool
}

func (f *fakeProjects) Get(_ context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok || (actor.Role != domain.RoleCEO && !f.members[actor.UserID]) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return p, nil
}

type fakeStore struct {
	tasks     map[uuid.UUID]domain.Task
	columns   map[uuid.UUID]domain.StatusGroup // cột của dự án duy nhất trong test
	todo      uuid.UUID
	projLocks int
	locks     int
	moves     []subtask.MoveRecord
}

func (f *fakeStore) RecordMove(_ context.Context, r subtask.MoveRecord) error {
	f.moves = append(f.moves, r)
	return nil
}

func (f *fakeStore) InTx(_ context.Context, fn func(subtask.Store) error) error {
	snap := map[uuid.UUID]domain.Task{}
	for k, v := range f.tasks {
		snap[k] = v
	}
	if err := fn(f); err != nil {
		f.tasks = snap
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

func (f *fakeStore) List(_ context.Context, parentID uuid.UUID) ([]domain.Task, error) {
	var out []domain.Task
	for _, t := range f.tasks {
		if t.ParentID != nil && *t.ParentID == parentID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeStore) LockProject(context.Context, uuid.UUID) (domain.LockedProject, error) {
	f.projLocks++
	return domain.LockedProject{}, nil
}

func (f *fakeStore) Lock(_ context.Context, id uuid.UUID) error {
	if _, ok := f.tasks[id]; !ok {
		return domain.ErrTaskNotFound
	}
	if f.projLocks == 0 {
		return errors.New("phải khóa dự án trước")
	}
	f.locks++
	return nil
}

func (f *fakeStore) ColumnGroup(_ context.Context, _, cid uuid.UUID) (domain.StatusGroup, error) {
	g, ok := f.columns[cid]
	if !ok {
		return "", domain.ErrColumnNotFound
	}
	return g, nil
}

func (f *fakeStore) FirstTodoColumn(context.Context, uuid.UUID) (uuid.UUID, error) {
	return f.todo, nil
}

func (f *fakeStore) Create(_ context.Context, parent domain.Task, cid, by uuid.UUID, title, desc string) (uuid.UUID, error) {
	pid := parent.ID
	t := domain.Task{
		ID: uuid.New(), ProjectID: parent.ProjectID, ParentID: &pid, ColumnID: cid,
		StatusGroup: f.columns[cid], Title: title, Description: desc,
		AssigneeID: parent.AssigneeID, CreatedBy: by,
	}
	f.tasks[t.ID] = t
	return t.ID, nil
}

func (f *fakeStore) Move(_ context.Context, id, cid uuid.UUID) error {
	t, ok := f.tasks[id]
	if !ok {
		return domain.ErrTaskNotFound
	}
	t.ColumnID, t.StatusGroup = cid, f.columns[cid]
	f.tasks[id] = t
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.tasks[id]; !ok {
		return domain.ErrTaskNotFound
	}
	delete(f.tasks, id)
	return nil
}

type fixture struct {
	svc                        *subtask.Service
	st                         *fakeStore
	pm, member, other, outside authz.Principal
	ceo                        authz.Principal
	parent                     domain.Task
	todo, prog, done           uuid.UUID
}

func setup(t *testing.T) fixture {
	t.Helper()
	pid := uuid.New()
	f := fixture{
		pm:      authz.Principal{UserID: uuid.New(), Role: domain.RoleManager},
		member:  authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		other:   authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		outside: authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		ceo:     authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO},
		todo:    uuid.New(), prog: uuid.New(), done: uuid.New(),
	}
	mid := f.member.UserID
	f.parent = domain.Task{ID: uuid.New(), ProjectID: pid, AssigneeID: &mid, Title: "Cha"}
	f.st = &fakeStore{
		tasks: map[uuid.UUID]domain.Task{f.parent.ID: f.parent},
		columns: map[uuid.UUID]domain.StatusGroup{
			f.todo: domain.GroupTodo, f.prog: domain.GroupInProgress, f.done: domain.GroupDone,
		},
		todo: f.todo,
	}
	projects := &fakeProjects{
		projects: map[uuid.UUID]domain.Project{pid: {ID: pid, ManagerID: f.pm.UserID}},
		members:  map[uuid.UUID]bool{f.pm.UserID: true, f.member.UserID: true, f.other.UserID: true},
	}
	f.svc = subtask.NewService(f.st, f.st, projects)
	return f
}

func TestCreate_CF07(t *testing.T) {
	ctx := context.Background()
	t.Run("Member là người nhận của cha tự tạo con: cột todo đầu tiên, kế thừa người nhận", func(t *testing.T) {
		f := setup(t)
		got, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "  Viết API  ", Description: "x"})
		if err != nil {
			t.Fatal(err)
		}
		if got.Title != "Viết API" || got.ColumnID != f.todo || got.ParentID == nil || *got.ParentID != f.parent.ID {
			t.Fatalf("con sai: %+v", got)
		}
		if got.AssigneeID == nil || *got.AssigneeID != f.member.UserID || got.StoryPoint != nil {
			t.Fatalf("con phải kế thừa người nhận, không Story Point: %+v", got)
		}
		if f.st.projLocks != 1 || f.st.locks != 1 {
			t.Fatalf("phải khóa dự án rồi khóa cha: %d/%d", f.st.projLocks, f.st.locks)
		}
	})
	t.Run("Manager phụ trách tạo được", func(t *testing.T) {
		f := setup(t)
		if _, err := f.svc.Create(ctx, f.pm, f.parent.ID, subtask.Input{Title: "A"}); err != nil {
			t.Fatal(err)
		}
	})
	tests := []struct {
		name  string
		actor func(fixture) authz.Principal
		in    subtask.Input
		want  error
	}{
		{"Member khác không tạo con trên việc của người khác", func(f fixture) authz.Principal { return f.other }, subtask.Input{Title: "A"}, domain.ErrSubtaskNotAllowed},
		{"ngoài dự án → 404 chống dò ID", func(f fixture) authz.Principal { return f.outside }, subtask.Input{Title: "A"}, domain.ErrTaskNotFound},
		{"CEO chỉ xem", func(f fixture) authz.Principal { return f.ceo }, subtask.Input{Title: "A"}, domain.ErrForbidden},
		{"tiêu đề rỗng", func(f fixture) authz.Principal { return f.member }, subtask.Input{Title: "  "}, domain.ErrInvalidTaskTitle},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := setup(t)
			_, err := f.svc.Create(ctx, tc.actor(f), f.parent.ID, tc.in)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if len(f.st.tasks) != 1 {
				t.Fatal("không được tạo con khi lỗi")
			}
		})
	}
	t.Run("con không làm cha được (1 cấp)", func(t *testing.T) {
		f := setup(t)
		c, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Create(ctx, f.member, c.ID, subtask.Input{Title: "B"}); !errors.Is(err, domain.ErrSubtaskParentNotRoot) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestList_CF07(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	if _, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"}); err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.List(ctx, f.ceo, f.parent.ID)
	if err != nil || len(got) != 1 {
		t.Fatalf("CEO xem được con: %v %v", got, err)
	}
	if _, err := f.svc.List(ctx, f.outside, f.parent.ID); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("ngoài dự án phải 404, got %v", err)
	}
	empty := setup(t)
	got, err = empty.svc.List(ctx, empty.member, empty.parent.ID)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("không có con → slice rỗng khác nil: %v %v", got, err)
	}
}

func TestMove_CF07(t *testing.T) {
	ctx := context.Background()
	newChild := func(t *testing.T, f fixture) domain.Task {
		t.Helper()
		c, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	t.Run("Member tự chuyển con sang Done — không cần luật chuyển cột, không qua Manager", func(t *testing.T) {
		f := setup(t)
		c := newChild(t, f)
		got, err := f.svc.Move(ctx, f.member, c.ID, f.done)
		if err != nil || got.ColumnID != f.done {
			t.Fatalf("got %+v, %v", got, err)
		}
	})
	t.Run("cột không thuộc dự án", func(t *testing.T) {
		f := setup(t)
		c := newChild(t, f)
		if _, err := f.svc.Move(ctx, f.member, c.ID, uuid.New()); !errors.Is(err, domain.ErrColumnNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("Member khác không chuyển con của người khác", func(t *testing.T) {
		f := setup(t)
		c := newChild(t, f)
		if _, err := f.svc.Move(ctx, f.other, c.ID, f.prog); !errors.Is(err, domain.ErrSubtaskNotAllowed) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("không dùng API con để chuyển việc cha", func(t *testing.T) {
		f := setup(t)
		if _, err := f.svc.Move(ctx, f.member, f.parent.ID, f.done); !errors.Is(err, domain.ErrTaskNotFound) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("ngoài dự án → 404", func(t *testing.T) {
		f := setup(t)
		c := newChild(t, f)
		if _, err := f.svc.Move(ctx, f.outside, c.ID, f.prog); !errors.Is(err, domain.ErrTaskNotFound) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestDelete_CF07(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	c, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Delete(ctx, f.other, c.ID); !errors.Is(err, domain.ErrSubtaskNotAllowed) {
		t.Fatalf("Member khác không xóa được, got %v", err)
	}
	if err := f.svc.Delete(ctx, f.member, f.parent.ID); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("không xóa việc cha qua API con, got %v", err)
	}
	if err := f.svc.Delete(ctx, f.member, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.st.tasks[c.ID]; ok {
		t.Fatal("con phải bị xóa")
	}
}

// Quyền trên con đi theo người nhận của VIỆC CHA (người được giao phần việc), không theo bản sao trên con.
func TestSubtaskPermissionFollowsParent_CF07(t *testing.T) {
	ctx := context.Background()
	reassign := func(f fixture, to *uuid.UUID) {
		p := f.st.tasks[f.parent.ID]
		p.AssigneeID = to
		f.st.tasks[f.parent.ID] = p
	}
	t.Run("cha được giao lại: người nhận mới chuyển/xóa được, người cũ bị chặn", func(t *testing.T) {
		f := setup(t)
		c, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"})
		if err != nil {
			t.Fatal(err)
		}
		oid := f.other.UserID
		reassign(f, &oid)
		if _, err := f.svc.Move(ctx, f.member, c.ID, f.prog); !errors.Is(err, domain.ErrSubtaskNotAllowed) {
			t.Fatalf("người nhận cũ: %v", err)
		}
		if err := f.svc.Delete(ctx, f.member, c.ID); !errors.Is(err, domain.ErrSubtaskNotAllowed) {
			t.Fatalf("người nhận cũ xóa: %v", err)
		}
		if _, err := f.svc.Move(ctx, f.other, c.ID, f.prog); err != nil {
			t.Fatalf("người nhận mới: %v", err)
		}
		if err := f.svc.Delete(ctx, f.other, c.ID); err != nil {
			t.Fatalf("người nhận mới xóa: %v", err)
		}
	})
	t.Run("cha chưa có người nhận: chỉ Manager phụ trách", func(t *testing.T) {
		f := setup(t)
		c, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"})
		if err != nil {
			t.Fatal(err)
		}
		reassign(f, nil)
		if _, err := f.svc.Move(ctx, f.member, c.ID, f.prog); !errors.Is(err, domain.ErrSubtaskNotAllowed) {
			t.Fatalf("member: %v", err)
		}
		if _, err := f.svc.Move(ctx, f.pm, c.ID, f.prog); err != nil {
			t.Fatalf("pm: %v", err)
		}
		if err := f.svc.Delete(ctx, f.pm, c.ID); err != nil {
			t.Fatalf("pm xóa: %v", err)
		}
	})
	t.Run("CEO chỉ xem: Move/Delete bị cấm", func(t *testing.T) {
		f := setup(t)
		c, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.svc.Move(ctx, f.ceo, c.ID, f.prog); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("CEO move: %v", err)
		}
		if err := f.svc.Delete(ctx, f.ceo, c.ID); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("CEO delete: %v", err)
		}
		if _, ok := f.st.tasks[c.ID]; !ok {
			t.Fatal("con không được bị xóa")
		}
	})
	t.Run("Manager ngoài dự án → 404", func(t *testing.T) {
		f := setup(t)
		c, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"})
		if err != nil {
			t.Fatal(err)
		}
		mgr := authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
		if _, err := f.svc.Move(ctx, mgr, c.ID, f.prog); !errors.Is(err, domain.ErrTaskNotFound) {
			t.Fatalf("move: %v", err)
		}
		if err := f.svc.Delete(ctx, mgr, c.ID); !errors.Is(err, domain.ErrTaskNotFound) {
			t.Fatalf("delete: %v", err)
		}
	})
}

func TestMoveHistory_CF07(t *testing.T) {
	ctx := context.Background()
	f := setup(t)
	c, err := f.svc.Create(ctx, f.member, f.parent.ID, subtask.Input{Title: "A"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Move(ctx, f.member, c.ID, f.prog); err != nil {
		t.Fatal(err)
	}
	if len(f.st.moves) != 1 {
		t.Fatalf("phải ghi 1 dòng lịch sử, có %d", len(f.st.moves))
	}
	m := f.st.moves[0]
	if m.TaskID != c.ID || m.ProjectID != c.ProjectID || m.FromColumnID != f.todo || m.ToColumnID != f.prog || m.MovedBy != f.member.UserID {
		t.Fatalf("lịch sử sai: %+v", m)
	}
	// Chuyển vào chính cột hiện tại: không đổi gì, không ghi lịch sử.
	got, err := f.svc.Move(ctx, f.member, c.ID, f.prog)
	if err != nil || got.ColumnID != f.prog {
		t.Fatalf("cùng cột: %+v %v", got, err)
	}
	if len(f.st.moves) != 1 {
		t.Fatalf("cùng cột không ghi lịch sử, có %d", len(f.st.moves))
	}
}
