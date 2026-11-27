//go:build integration

package postgres

// Test sinh từ thiết kế test case (docs/testing/test-cases.md): ràng buộc CHECK ở CSDL là
// lớp bảo vệ cuối cùng cho R04 (Story Point chỉ ở việc cha) và giới hạn WIP 1..20 (CF-05),
// kể cả khi có ai đó ghi thẳng vào CSDL bỏ qua tầng application.

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
)

func wantCheckViolation(t *testing.T, err error, constraint string) {
	t.Helper()
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != constraint {
		t.Fatalf("mong đợi vi phạm CHECK %s, nhận %v", constraint, err)
	}
}

func TestTC_Integration_DBConstraints_CF06_CF05(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	users := NewUserRepo(pool)
	mgrID := newIntegrationUser(t, users, "tc-mgr@worktrack.local", domain.RoleManager)
	mgr := authz.Principal{UserID: mgrID, Role: domain.RoleManager}

	projects := project.NewService(NewProjectRepo(pool), NewProjectTx(pool))
	settings := setting.NewService(NewSettingRepo(pool))
	tasks := task.NewService(NewTaskRepo(pool), NewTaskTx(pool), projects, settings, realtime.NewHub())

	p, err := projects.Create(ctx, mgr, project.Input{Name: "Dự án kiểm thử ràng buộc"})
	if err != nil {
		t.Fatal(err)
	}
	sp := 5
	parent, err := tasks.Create(ctx, mgr, p.ID, task.Input{Title: "Việc cha", StoryPoint: &sp})
	if err != nil {
		t.Fatal(err)
	}

	insertChild := func(storyPoint any) error {
		_, err := pool.Exec(ctx, `
			INSERT INTO task (project_id, column_id, parent_task_id, title, story_point, position, created_by)
			SELECT project_id, column_id, id, 'Việc con', $2, 0, created_by FROM task WHERE id = $1`,
			parent.ID, storyPoint)
		return err
	}

	t.Run("TC_TASK_038_viec_con_co_story_point_bi_chan", func(t *testing.T) {
		wantCheckViolation(t, insertChild(3), "ck_task_story_point_parent_only")
	})
	t.Run("TC_TASK_039_viec_con_khong_story_point_hop_le", func(t *testing.T) {
		if err := insertChild(nil); err != nil {
			t.Fatalf("việc con không Story Point phải hợp lệ: %v", err)
		}
	})
	t.Run("TC_TASK_040_tieu_de_trang_bi_chan", func(t *testing.T) {
		_, err := pool.Exec(ctx, `UPDATE task SET title = '   ' WHERE id = $1`, parent.ID)
		wantCheckViolation(t, err, "ck_task_title_not_blank")
	})

	setWIP := func(n int) error {
		_, err := pool.Exec(ctx, `UPDATE project SET wip_limit = $2 WHERE id = $1`, p.ID, n)
		return err
	}
	t.Run("TC_KANBAN_045_wip_0_bi_chan_o_csdl", func(t *testing.T) {
		wantCheckViolation(t, setWIP(0), "ck_project_wip_limit")
	})
	t.Run("TC_KANBAN_046_wip_21_bi_chan_o_csdl", func(t *testing.T) {
		wantCheckViolation(t, setWIP(21), "ck_project_wip_limit")
	})
	t.Run("TC_KANBAN_047_wip_1_va_20_hop_le", func(t *testing.T) {
		if err := setWIP(1); err != nil {
			t.Fatal(err)
		}
		if err := setWIP(20); err != nil {
			t.Fatal(err)
		}
	})
}
