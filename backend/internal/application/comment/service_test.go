package comment_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/comment"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

type fakeProjects struct {
	pid     uuid.UUID
	members map[uuid.UUID]bool
}

func (f *fakeProjects) Get(_ context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error) {
	if id != f.pid || (actor.Role != domain.RoleCEO && !f.members[actor.UserID]) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return domain.Project{ID: id}, nil
}

type fakeStore struct {
	task      domain.Task
	parents   map[uuid.UUID]domain.Task
	comments  []domain.Comment
	notes     []domain.NewNotification
	failNotes bool
}

func (f *fakeStore) InTx(_ context.Context, fn func(comment.Store) error) error {
	c, n := len(f.comments), len(f.notes)
	if err := fn(f); err != nil {
		f.comments, f.notes = f.comments[:c], f.notes[:n]
		return err
	}
	return nil
}

func (f *fakeStore) GetTask(_ context.Context, id uuid.UUID) (domain.Task, error) {
	if p, ok := f.parents[id]; ok {
		return p, nil
	}
	if id != f.task.ID {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return f.task, nil
}

func (f *fakeStore) List(_ context.Context, _ uuid.UUID, _ domain.PageRequest) ([]domain.Comment, int, error) {
	return f.comments, len(f.comments), nil
}

func (f *fakeStore) Create(_ context.Context, tid, aid uuid.UUID, body string) (domain.Comment, error) {
	c := domain.Comment{ID: uuid.New(), TaskID: tid, AuthorID: aid, AuthorName: "A", Body: body}
	f.comments = append(f.comments, c)
	return c, nil
}

func (f *fakeStore) AddNotification(_ context.Context, n domain.NewNotification) error {
	if f.failNotes {
		return errors.New("db down")
	}
	f.notes = append(f.notes, n)
	return nil
}

type fixture struct {
	svc                           *comment.Service
	st                            *fakeStore
	pm, assignee, other, outsider authz.Principal
	ceo                           authz.Principal
}

func setup() fixture {
	f := fixture{
		pm:       authz.Principal{UserID: uuid.New(), Role: domain.RoleManager},
		assignee: authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		other:    authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		outsider: authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		ceo:      authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO},
	}
	pid, aid := uuid.New(), f.assignee.UserID
	f.st = &fakeStore{task: domain.Task{ID: uuid.New(), ProjectID: pid, AssigneeID: &aid, CreatedBy: f.pm.UserID}}
	projects := &fakeProjects{pid: pid, members: map[uuid.UUID]bool{
		f.pm.UserID: true, f.assignee.UserID: true, f.other.UserID: true,
	}}
	f.svc = comment.NewService(f.st, f.st, projects)
	return f
}

func TestCreateComment_CF07(t *testing.T) {
	ctx := context.Background()
	t.Run("Member khác trong dự án bình luận → báo người nhận và người tạo", func(t *testing.T) {
		f := setup()
		c, err := f.svc.Create(ctx, f.other, f.st.task.ID, "  Cần xem lại\nAPI  ")
		if err != nil {
			t.Fatal(err)
		}
		if c.Body != "Cần xem lại\nAPI" {
			t.Fatalf("body %q", c.Body)
		}
		if len(f.st.notes) != 1 {
			t.Fatalf("phải ghi 1 thông báo, got %d", len(f.st.notes))
		}
		n := f.st.notes[0]
		if n.Type != domain.NotificationTaskCommented || n.Message != "Cần xem lại API" || len(n.UserIDs) != 2 ||
			n.UserIDs[0] != f.assignee.UserID || n.UserIDs[1] != f.pm.UserID {
			t.Fatalf("thông báo sai: %+v", n)
		}
	})
	t.Run("người nhận tự bình luận → chỉ báo người tạo", func(t *testing.T) {
		f := setup()
		if _, err := f.svc.Create(ctx, f.assignee, f.st.task.ID, "ok"); err != nil {
			t.Fatal(err)
		}
		if ids := f.st.notes[0].UserIDs; len(ids) != 1 || ids[0] != f.pm.UserID {
			t.Fatalf("got %v", ids)
		}
	})
	tests := []struct {
		name  string
		actor func(fixture) authz.Principal
		body  string
		want  error
	}{
		{"ngoài dự án → 404", func(f fixture) authz.Principal { return f.outsider }, "x", domain.ErrTaskNotFound},
		{"CEO chỉ đọc", func(f fixture) authz.Principal { return f.ceo }, "x", domain.ErrForbidden},
		{"rỗng", func(f fixture) authz.Principal { return f.other }, "  \n ", domain.ErrInvalidComment},
		{"quá 2000 ký tự", func(f fixture) authz.Principal { return f.other }, strings.Repeat("a", 2001), domain.ErrInvalidComment},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := setup()
			if _, err := f.svc.Create(ctx, tc.actor(f), f.st.task.ID, tc.body); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if len(f.st.comments) != 0 {
				t.Fatal("không được lưu khi lỗi")
			}
		})
	}
	t.Run("ghi thông báo lỗi → hủy cả bình luận", func(t *testing.T) {
		f := setup()
		f.st.failNotes = true
		if _, err := f.svc.Create(ctx, f.other, f.st.task.ID, "x"); err == nil {
			t.Fatal("phải lỗi")
		}
		if len(f.st.comments) != 0 {
			t.Fatal("bình luận phải bị rollback")
		}
	})
}

func TestListComments_CF07(t *testing.T) {
	ctx := context.Background()
	f := setup()
	got, err := f.svc.List(ctx, f.ceo, f.st.task.ID, domain.NewPageRequest(1, 20))
	if err != nil || got.Items == nil || got.Total != 0 {
		t.Fatalf("CEO đọc được, rỗng khác nil: %+v %v", got, err)
	}
	if _, err := f.svc.List(ctx, f.outsider, f.st.task.ID, domain.NewPageRequest(1, 20)); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("got %v", err)
	}
}

// Bình luận ở việc con báo người nhận HIỆN TẠI của cha (quyền việc con đi theo cha, mục C),
// không báo người nhận cũ đã chép vào con lúc tạo.
func TestCommentOnSubtaskNotifiesParentAssignee_CF07(t *testing.T) {
	f := setup()
	oldAssignee := f.st.task.AssigneeID
	newAssignee := f.other.UserID
	parent := domain.Task{ID: uuid.New(), ProjectID: f.st.task.ProjectID, AssigneeID: &newAssignee, CreatedBy: f.pm.UserID}
	f.st.parents = map[uuid.UUID]domain.Task{parent.ID: parent}
	f.st.task.ParentID = &parent.ID

	if _, err := f.svc.Create(context.Background(), f.pm, f.st.task.ID, "Cập nhật"); err != nil {
		t.Fatal(err)
	}
	got := f.st.notes[0].UserIDs
	if len(got) != 1 || got[0] != newAssignee {
		t.Fatalf("người nhận thông báo = %v; muốn chỉ %v (không gồm người cũ %v)", got, newAssignee, *oldAssignee)
	}
}
