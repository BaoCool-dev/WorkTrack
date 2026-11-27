//go:build integration

package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/subtask"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/timelog"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
)

// CF-08 trên Postgres thật: ghi giờ cha + con, giờ thực tế cộng dồn ở task_view, lịch sử, luật 24 giờ sau Done.
func TestTimeLog_ServiceFlow_CF08(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	users := NewUserRepo(pool)
	mgrID := newIntegrationUser(t, users, "mgr@worktrack.local", domain.RoleManager)
	memID := newIntegrationUser(t, users, "mem@worktrack.local", domain.RoleMember)

	hub := realtime.NewHub()
	projects := project.NewService(NewProjectRepo(pool), NewProjectTx(pool))
	tasks := task.NewService(NewTaskRepo(pool), NewTaskTx(pool), projects, setting.NewService(NewSettingRepo(pool)), hub)
	subtasks := subtask.NewService(NewSubtaskRepo(pool), NewSubtaskTx(pool), projects)
	logs := timelog.NewService(NewTimeLogRepo(pool), NewTimeLogTx(pool), projects, hub, time.Now)
	mgr := authz.Principal{UserID: mgrID, Role: domain.RoleManager}
	mem := authz.Principal{UserID: memID, Role: domain.RoleMember}

	p, err := projects.Create(ctx, mgr, project.Input{Name: "Dự án CF08"})
	if err != nil {
		t.Fatal(err)
	}
	if err := projects.AddMember(ctx, mgr, p.ID, memID); err != nil {
		t.Fatal(err)
	}
	parent, err := tasks.Create(ctx, mgr, p.ID, task.Input{Title: "Việc cha", AssigneeID: &memID})
	if err != nil {
		t.Fatal(err)
	}
	child, err := subtasks.Create(ctx, mem, parent.ID, subtask.Input{Title: "Việc con"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := logs.Create(ctx, mem, parent.ID, timelog.Input{Minutes: 90, Note: "Phân tích"}); err != nil {
		t.Fatal(err)
	}
	l, err := logs.Create(ctx, mem, child.ID, timelog.Input{Minutes: 30, Note: "Code"})
	if err != nil {
		t.Fatal(err)
	}
	if l.ParentID == nil || *l.ParentID != parent.ID || l.UserName == "" || l.TaskTitle != "Việc con" {
		t.Fatalf("log việc con sai: %+v", l)
	}
	// Giờ con cộng lên cha (R: Actual = giờ cha + giờ con).
	got, err := NewTaskRepo(pool).Get(ctx, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LoggedMinutes != 90 || got.ActualMinutes != 120 || got.SubtaskCount != 1 || got.SubtaskDoneCount != 0 {
		t.Fatalf("giờ thực tế cha: %+v", got)
	}
	// Lịch sử của cha gồm cả log của con.
	page, err := logs.List(ctx, mgr, parent.ID, domain.PageRequest{Page: 1, PageSize: 20})
	if err != nil || page.Total != 2 || len(page.Items) != 2 {
		t.Fatalf("lịch sử cha: %+v %v", page, err)
	}
	// CHECK ở CSDL là lưới cuối: ghi thẳng phút âm bị chặn.
	_, err = pool.Exec(ctx, `UPDATE time_log SET minutes = -5 WHERE id = $1`, l.ID)
	wantCheckViolation(t, err, "ck_time_log_minutes")

	// Xóa log của chính mình.
	if err := logs.Delete(ctx, mem, l.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := NewTimeLogRepo(pool).GetLog(ctx, l.ID); !errors.Is(err, domain.ErrTimeLogNotFound) {
		t.Fatalf("log đã xóa: %v", err)
	}

	// Cha vào Done cách đây 25 giờ → không log được vào cha lẫn con.
	_, err = pool.Exec(ctx, `
		WITH d AS (SELECT id FROM board_column WHERE project_id = $1 AND status_group = 'done' ORDER BY position LIMIT 1)
		UPDATE task SET column_id = (SELECT id FROM d) WHERE id = $2`, p.ID, parent.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO task_move_history (task_id, project_id, from_column_id, to_column_id, moved_by, moved_at)
		SELECT $1, $2, gen_random_uuid(), t.column_id, $3, now() - interval '25 hours' FROM task t WHERE t.id = $1`,
		parent.ID, p.ID, mgrID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := logs.Create(ctx, mem, parent.ID, timelog.Input{Minutes: 10, Note: "Muộn"}); !errors.Is(err, domain.ErrTimeLogClosed) {
		t.Fatalf("log cha đã Done > 24h: %v", err)
	}
	if _, err := logs.Create(ctx, mem, child.ID, timelog.Input{Minutes: 10, Note: "Muộn"}); !errors.Is(err, domain.ErrTimeLogClosed) {
		t.Fatalf("log con khi cha Done > 24h: %v", err)
	}
}
