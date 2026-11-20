//go:build integration

package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/label"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Chạy Service thật trên Postgres: cột todo đầu tiên, nhãn, quy đổi B1, nhận việc, lọc, IDOR, WIP.
func TestTaskRepo_ServiceFlow_CF06(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	users := NewUserRepo(pool)
	mgrID := newIntegrationUser(t, users, "mgr@worktrack.local", domain.RoleManager)
	memID := newIntegrationUser(t, users, "mem@worktrack.local", domain.RoleMember)
	outID := newIntegrationUser(t, users, "out@worktrack.local", domain.RoleMember)
	adminID := newIntegrationUser(t, users, "admin@worktrack.local", domain.RoleAdmin)

	projects := project.NewService(NewProjectRepo(pool), NewProjectTx(pool))
	settings := setting.NewService(NewSettingRepo(pool))
	tasks := task.NewService(NewTaskRepo(pool), NewTaskTx(pool), projects, settings)
	labels := label.NewService(NewLabelRepo(pool))
	mgr := authz.Principal{UserID: mgrID, Role: domain.RoleManager}
	mem := authz.Principal{UserID: memID, Role: domain.RoleMember}
	out := authz.Principal{UserID: outID, Role: domain.RoleMember}
	admin := authz.Principal{UserID: adminID, Role: domain.RoleAdmin}

	// Tham số B1 mặc định 4 giờ/SP, Admin sửa được.
	s, err := settings.Get(ctx, mem)
	if err != nil || s.HoursPerStoryPoint != 4 {
		t.Fatalf("settings mặc định: %+v %v", s, err)
	}
	if _, err := settings.Update(ctx, admin, 6); err != nil {
		t.Fatal(err)
	}

	lb, err := labels.Create(ctx, admin, "Backend")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := labels.Create(ctx, admin, "backend"); !errors.Is(err, domain.ErrLabelNameTaken) {
		t.Fatalf("trùng nhãn: %v", err)
	}

	p, err := projects.Create(ctx, mgr, project.Input{Name: "Cổng thông tin"})
	if err != nil {
		t.Fatal(err)
	}
	if err := projects.AddMember(ctx, mgr, p.ID, memID); err != nil {
		t.Fatal(err)
	}

	sp := 3
	due := time.Date(2026, 12, 15, 0, 0, 0, 0, time.UTC)
	tk, err := tasks.Create(ctx, mgr, p.ID, task.Input{Title: "Thiết kế CSDL", StoryPoint: &sp, DueDate: &due, LabelIDs: []uuid.UUID{lb.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if tk.StatusGroup != domain.GroupTodo || tk.ColumnName == "" || tk.EstimatedMinutes == nil || *tk.EstimatedMinutes != 3*6*60 ||
		len(tk.Labels) != 1 || tk.DueDate == nil || !tk.DueDate.Equal(due) || tk.CreatedByName == "" {
		t.Fatalf("việc vừa tạo sai: %+v", tk)
	}

	// Đổi hệ số không sửa giờ đã chụp của việc cũ.
	if _, err := settings.Update(ctx, admin, 8); err != nil {
		t.Fatal(err)
	}
	got, err := tasks.Get(ctx, mem, tk.ID)
	if err != nil || *got.EstimatedMinutes != 3*6*60 {
		t.Fatalf("giờ ước tính bị đổi: %+v %v", got, err)
	}

	// Gán nhãn không tồn tại → LABEL_NOT_FOUND; người ngoài dự án → ASSIGNEE_INVALID.
	if _, err := tasks.Create(ctx, mgr, p.ID, task.Input{Title: "x", LabelIDs: []uuid.UUID{uuid.New()}}); !errors.Is(err, domain.ErrLabelNotFound) {
		t.Fatalf("nhãn lạ: %v", err)
	}
	if _, err := tasks.Create(ctx, mgr, p.ID, task.Input{Title: "x", AssigneeID: &outID}); !errors.Is(err, domain.ErrInvalidAssignee) {
		t.Fatalf("người ngoài: %v", err)
	}

	// IDOR: Member ngoài dự án không xem được.
	if _, err := tasks.Get(ctx, out, tk.ID); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("IDOR: %v", err)
	}

	// Member nhận việc; lần hai → đã có người nhận.
	if c, err := tasks.Claim(ctx, mem, tk.ID); err != nil || c.AssigneeID == nil || *c.AssigneeID != memID || c.AssigneeName == "" {
		t.Fatalf("nhận việc: %+v %v", c, err)
	}
	if _, err := tasks.Claim(ctx, mem, tk.ID); !errors.Is(err, domain.ErrTaskAlreadyAssigned) {
		t.Fatalf("nhận lại: %v", err)
	}

	// Việc thứ hai chưa giao để lọc.
	if _, err := tasks.Create(ctx, mgr, p.ID, task.Input{Title: "Viết API"}); err != nil {
		t.Fatal(err)
	}
	page, err := tasks.List(ctx, mem, p.ID, task.ListInput{Unassigned: true}, domain.NewPageRequest(1, 20))
	if err != nil || page.Total != 1 || page.Items[0].Title != "Viết API" {
		t.Fatalf("lọc chưa giao: %+v %v", page, err)
	}
	page, err = tasks.List(ctx, mem, p.ID, task.ListInput{LabelID: &lb.ID, Search: "csdl"}, domain.NewPageRequest(1, 20))
	if err != nil || page.Total != 1 || page.Items[0].ID != tk.ID {
		t.Fatalf("lọc nhãn + tìm: %+v %v", page, err)
	}
	page, err = tasks.List(ctx, mem, p.ID, task.ListInput{}, domain.NewPageRequest(2, 1))
	if err != nil || page.Total != 2 || len(page.Items) != 1 {
		t.Fatalf("phân trang: %+v %v", page, err)
	}

	mine, err := tasks.ListMine(ctx, mem, task.MyListInput{}, domain.NewPageRequest(1, 20))
	if err != nil || mine.Total != 1 || mine.Items[0].ProjectName != "Cổng thông tin" {
		t.Fatalf("việc của tôi: %+v %v", mine, err)
	}
	// Nhận việc không đổi cột (CF-06 chưa có chuyển cột) → việc còn ở nhóm todo, WIP = 0.
	if n, err := tasks.MyWIP(ctx, mem); err != nil || n != 0 {
		t.Fatalf("WIP khi còn ở todo: %d %v", n, err)
	}
	// Đặt việc vào cột in_progress bằng SQL (giả lập Kanban sau này) → WIP = 1.
	if _, err := pool.Exec(ctx, `UPDATE task SET column_id = (
		SELECT id FROM board_column WHERE project_id = $1 AND status_group = 'in_progress' ORDER BY position LIMIT 1
	) WHERE id = $2`, p.ID, tk.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := tasks.MyWIP(ctx, mem); err != nil || n != 1 {
		t.Fatalf("WIP khi ở in_progress: %d %v", n, err)
	}
	// Việc con ở cột in_progress không tính vào WIP (CLAUDE.md §5.6).
	if _, err := pool.Exec(ctx, `INSERT INTO task (project_id, column_id, parent_task_id, title, assignee_id, created_by)
		SELECT project_id, column_id, id, 'Việc con', $2, $3 FROM task WHERE id = $1`, tk.ID, memID, memID); err != nil {
		t.Fatal(err)
	}
	if n, err := tasks.MyWIP(ctx, mem); err != nil || n != 1 {
		t.Fatalf("WIP đếm cả việc con: %d %v", n, err)
	}
	// Việc con có Story Point → CHECK ck_task_story_point_parent_only chặn (CLAUDE.md §5.4).
	if _, err := pool.Exec(ctx, `INSERT INTO task (project_id, column_id, parent_task_id, title, story_point, created_by)
		SELECT project_id, column_id, id, 'Con có SP', 3, $2 FROM task WHERE id = $1`, tk.ID, memID); err == nil ||
		!strings.Contains(err.Error(), "ck_task_story_point_parent_only") {
		t.Fatalf("con có SP phải bị CHECK chặn: %v", err)
	}

	// Sửa: bỏ nhãn, đổi SP → chụp lại theo hệ số hiện tại (8).
	sp5 := 5
	up, err := tasks.Update(ctx, mgr, tk.ID, task.Input{Title: "Thiết kế CSDL v2", StoryPoint: &sp5, AssigneeID: &memID})
	if err != nil || len(up.Labels) != 0 || *up.EstimatedMinutes != 5*8*60 || up.Title != "Thiết kế CSDL v2" {
		t.Fatalf("sửa: %+v %v", up, err)
	}

	// Nhãn đang dùng không xóa được; sau khi gỡ thì xóa được.
	tk3, err := tasks.Create(ctx, mgr, p.ID, task.Input{Title: "Có nhãn", LabelIDs: []uuid.UUID{lb.ID}})
	if err != nil {
		t.Fatal(err)
	}
	if err := labels.Delete(ctx, admin, lb.ID); !errors.Is(err, domain.ErrLabelInUse) {
		t.Fatalf("xóa nhãn đang dùng: %v", err)
	}
	ls, err := labels.List(ctx, mem)
	if err != nil || len(ls) != 1 || ls[0].TaskCount != 1 {
		t.Fatalf("danh mục nhãn: %+v %v", ls, err)
	}
	if _, err := tasks.Update(ctx, mgr, tk3.ID, task.Input{Title: "Có nhãn"}); err != nil {
		t.Fatal(err)
	}
	if err := labels.Delete(ctx, admin, lb.ID); err != nil {
		t.Fatalf("xóa nhãn: %v", err)
	}

	// Cột còn việc không xóa được.
	b, err := projects.GetBoard(ctx, mgr, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := projects.DeleteColumn(ctx, mgr, p.ID, tk.ColumnID); !errors.Is(err, domain.ErrColumnHasTasks) {
		t.Fatalf("xóa cột còn việc: %v (cột %d)", err, len(b.Columns))
	}
}
