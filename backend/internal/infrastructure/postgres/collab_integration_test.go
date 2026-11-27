//go:build integration

package postgres

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/comment"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/notification"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/subtask"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
)

// CF-07 trên Postgres thật: việc con 1 cấp, bình luận, thông báo (giao việc, bình luận), đọc/chưa đọc.
func TestCollab_ServiceFlow_CF07(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	users := NewUserRepo(pool)
	mgrID := newIntegrationUser(t, users, "mgr@worktrack.local", domain.RoleManager)
	memID := newIntegrationUser(t, users, "mem@worktrack.local", domain.RoleMember)
	outID := newIntegrationUser(t, users, "out@worktrack.local", domain.RoleMember)

	projects := project.NewService(NewProjectRepo(pool), NewProjectTx(pool))
	settings := setting.NewService(NewSettingRepo(pool))
	tasks := task.NewService(NewTaskRepo(pool), NewTaskTx(pool), projects, settings, realtime.NewHub())
	subtasks := subtask.NewService(NewSubtaskRepo(pool), NewSubtaskTx(pool), projects)
	comments := comment.NewService(NewCommentRepo(pool), NewCommentTx(pool), projects)
	notifs := notification.NewService(NewNotificationRepo(pool))
	mgr := authz.Principal{UserID: mgrID, Role: domain.RoleManager}
	mem := authz.Principal{UserID: memID, Role: domain.RoleMember}
	out := authz.Principal{UserID: outID, Role: domain.RoleMember}

	p, err := projects.Create(ctx, mgr, project.Input{Name: "Dự án CF07"})
	if err != nil {
		t.Fatal(err)
	}
	if err := projects.AddMember(ctx, mgr, p.ID, memID); err != nil {
		t.Fatal(err)
	}
	other, err := projects.Create(ctx, mgr, project.Input{Name: "Dự án khác"})
	if err != nil {
		t.Fatal(err)
	}

	// Giao việc khi tạo → Member có 1 thông báo task_assigned.
	parent, err := tasks.Create(ctx, mgr, p.ID, task.Input{Title: "Việc cha", AssigneeID: &memID})
	if err != nil {
		t.Fatal(err)
	}
	if n, err := notifs.UnreadCount(ctx, mem); err != nil || n != 1 {
		t.Fatalf("thông báo giao việc: %d %v", n, err)
	}

	// Member là người nhận tạo con → kế thừa người nhận, ở nhóm todo, không Story Point.
	child, err := subtasks.Create(ctx, mem, parent.ID, subtask.Input{Title: "Việc con"})
	if err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID || child.AssigneeID == nil || *child.AssigneeID != memID ||
		child.StatusGroup != domain.GroupTodo || child.StoryPoint != nil {
		t.Fatalf("việc con sai: %+v", child)
	}
	// Con chỉ 1 cấp.
	if _, err := subtasks.Create(ctx, mem, child.ID, subtask.Input{Title: "Cháu"}); !errors.Is(err, domain.ErrSubtaskParentNotRoot) {
		t.Fatalf("cháu: %v", err)
	}
	// Người ngoài dự án → 404.
	if _, err := subtasks.Create(ctx, out, parent.ID, subtask.Input{Title: "x"}); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("người ngoài: %v", err)
	}
	list, err := subtasks.List(ctx, mem, parent.ID)
	if err != nil || len(list) != 1 || list[0].ID != child.ID {
		t.Fatalf("danh sách con: %+v %v", list, err)
	}

	// Chuyển cột tự do (không qua workflow_transition); cột dự án khác → không tìm thấy.
	board, err := projects.GetBoard(ctx, mem, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	var inProgress uuid.UUID
	for _, c := range board.Columns {
		if c.StatusGroup == domain.GroupInProgress {
			inProgress = c.ID
			break
		}
	}
	moved, err := subtasks.Move(ctx, mem, child.ID, inProgress)
	if err != nil || moved.StatusGroup != domain.GroupInProgress {
		t.Fatalf("chuyển con: %+v %v", moved, err)
	}
	otherBoard, err := projects.GetBoard(ctx, mgr, other.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := subtasks.Move(ctx, mem, child.ID, otherBoard.Columns[0].ID); !errors.Is(err, domain.ErrColumnNotFound) {
		t.Fatalf("cột dự án khác: %v", err)
	}
	// API con không dùng cho việc cha.
	if _, err := subtasks.Move(ctx, mgr, parent.ID, inProgress); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("move việc cha qua API con: %v", err)
	}

	// Bình luận của Manager → báo cho Member (người nhận); người tạo là chính Manager nên không tự báo.
	c, err := comments.Create(ctx, mgr, parent.ID, "  Nhớ cập nhật tiến độ  ")
	if err != nil || c.Body != "Nhớ cập nhật tiến độ" || c.AuthorName == "" {
		t.Fatalf("bình luận: %+v %v", c, err)
	}
	if n, _ := notifs.UnreadCount(ctx, mgr); n != 0 {
		t.Fatalf("tác giả không nhận thông báo của chính mình: %d", n)
	}
	// Bình luận của Member → báo cho Manager (người tạo).
	if _, err := comments.Create(ctx, mem, parent.ID, "Đã rõ"); err != nil {
		t.Fatal(err)
	}
	if n, _ := notifs.UnreadCount(ctx, mgr); n != 1 {
		t.Fatalf("người tạo nhận thông báo bình luận: %d", n)
	}
	if _, err := comments.Create(ctx, mem, parent.ID, "   "); !errors.Is(err, domain.ErrInvalidComment) {
		t.Fatalf("bình luận rỗng: %v", err)
	}
	if _, err := comments.List(ctx, out, parent.ID, domain.NewPageRequest(1, 20)); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("IDOR bình luận: %v", err)
	}
	cp, err := comments.List(ctx, mem, parent.ID, domain.NewPageRequest(1, 20))
	if err != nil || cp.Total != 2 || cp.Items[0].Body != "Nhớ cập nhật tiến độ" {
		t.Fatalf("danh sách bình luận: %+v %v", cp, err)
	}

	// Chuông của Member: giao việc + bình luận, mới nhất trước.
	np, err := notifs.List(ctx, mem, true, domain.NewPageRequest(1, 20))
	if err != nil || np.Total != 2 || np.Items[0].Type != domain.NotificationTaskCommented ||
		np.Items[0].TaskTitle != "Việc cha" || np.Items[0].ProjectName != "Dự án CF07" || np.Items[0].ActorName == "" {
		t.Fatalf("chuông member: %+v %v", np, err)
	}
	// Người khác không đánh dấu đọc được thông báo của Member (IDOR).
	if err := notifs.MarkRead(ctx, mgr, np.Items[0].ID); !errors.Is(err, domain.ErrNotificationNotFound) {
		t.Fatalf("IDOR đánh dấu đọc: %v", err)
	}
	if err := notifs.MarkRead(ctx, mem, np.Items[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := notifs.MarkRead(ctx, mem, np.Items[0].ID); err != nil {
		t.Fatalf("đánh dấu đọc lần hai phải idempotent: %v", err)
	}
	if n, _ := notifs.UnreadCount(ctx, mem); n != 1 {
		t.Fatalf("sau đánh dấu 1: %d", n)
	}
	if n, err := notifs.MarkAllRead(ctx, mem); err != nil || n != 1 {
		t.Fatalf("đọc tất cả: %d %v", n, err)
	}
	all, err := notifs.List(ctx, mem, false, domain.NewPageRequest(1, 20))
	if err != nil || all.Total != 2 || all.Items[0].ReadAt == nil {
		t.Fatalf("danh sách đầy đủ: %+v %v", all, err)
	}

	// Xóa con; xóa lần hai → không tìm thấy.
	if err := subtasks.Delete(ctx, mem, child.ID); err != nil {
		t.Fatal(err)
	}
	if err := subtasks.Delete(ctx, mem, child.ID); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("xóa lại: %v", err)
	}
}
