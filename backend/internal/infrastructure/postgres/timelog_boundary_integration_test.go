//go:build integration

package postgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/timelog"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
)

// QA bổ sung CF-08 trên Postgres thật: phân quyền theo vai trò, giá trị biên lưu nguyên văn,
// biên 24 giờ sau Done (phía trong), phân trang lịch sử.

type tlFixture struct {
	pool     *pgxpool.Pool
	logs     *timelog.Service
	mgr, mem authz.Principal
	projID   uuid.UUID
	parentID uuid.UUID
}

func newTLFixture(t *testing.T) tlFixture {
	t.Helper()
	pool := newTestPool(t)
	ctx := t.Context()
	users := NewUserRepo(pool)
	mgrID := newIntegrationUser(t, users, "mgr@worktrack.local", domain.RoleManager)
	memID := newIntegrationUser(t, users, "mem@worktrack.local", domain.RoleMember)
	hub := realtime.NewHub()
	projects := project.NewService(NewProjectRepo(pool), NewProjectTx(pool))
	tasks := task.NewService(NewTaskRepo(pool), NewTaskTx(pool), projects, setting.NewService(NewSettingRepo(pool)), hub)
	mgr := authz.Principal{UserID: mgrID, Role: domain.RoleManager}
	p, err := projects.Create(ctx, mgr, project.Input{Name: "Dự án QA CF08"})
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
	return tlFixture{
		pool:     pool,
		logs:     timelog.NewService(NewTimeLogRepo(pool), NewTimeLogTx(pool), projects, hub, time.Now),
		mgr:      mgr,
		mem:      authz.Principal{UserID: memID, Role: domain.RoleMember},
		projID:   p.ID,
		parentID: parent.ID,
	}
}

// TC_LOG_101..106 — bảng quyết định vai trò × quan hệ với việc, trên CSDL thật [CF-08].
func TestTimeLog_PermissionMatrix_DB_CF08(t *testing.T) {
	f := newTLFixture(t)
	ctx := t.Context()
	users := NewUserRepo(f.pool)
	projects := project.NewService(NewProjectRepo(f.pool), NewProjectTx(f.pool))

	ceo := authz.Principal{UserID: newIntegrationUser(t, users, "ceo@worktrack.local", domain.RoleCEO), Role: domain.RoleCEO}
	admin := authz.Principal{UserID: newIntegrationUser(t, users, "admin@worktrack.local", domain.RoleAdmin), Role: domain.RoleAdmin}
	otherMemID := newIntegrationUser(t, users, "mem2@worktrack.local", domain.RoleMember)
	if err := projects.AddMember(ctx, f.mgr, f.projID, otherMemID); err != nil {
		t.Fatal(err)
	}
	otherMem := authz.Principal{UserID: otherMemID, Role: domain.RoleMember}
	outsider := authz.Principal{UserID: newIntegrationUser(t, users, "out@worktrack.local", domain.RoleMember), Role: domain.RoleMember}
	otherMgr := authz.Principal{UserID: newIntegrationUser(t, users, "mgr2@worktrack.local", domain.RoleManager), Role: domain.RoleManager}

	in := timelog.Input{Minutes: 15, Note: "Thử quyền"}
	cases := []struct {
		name    string
		actor   authz.Principal
		wantErr error
	}{
		{"TC_LOG_101 CEO chỉ xem, không log", ceo, domain.ErrForbidden},
		{"TC_LOG_102 Admin không xem dự án → 404", admin, domain.ErrTaskNotFound},
		{"TC_LOG_103 Member cùng dự án nhưng không phải người nhận", otherMem, domain.ErrTimeLogNotAllowed},
		{"TC_LOG_104 Member ngoài dự án → 404", outsider, domain.ErrTaskNotFound},
		{"TC_LOG_105 Manager dự án khác → 404", otherMgr, domain.ErrTaskNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := f.logs.Create(ctx, c.actor, f.parentID, in); !errors.Is(err, c.wantErr) {
				t.Fatalf("got %v, want %v", err, c.wantErr)
			}
		})
	}
	var n int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM time_log`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("không được có log nào sau các lần bị từ chối: n=%d err=%v", n, err)
	}

	// TC_LOG_106 — CEO xem lịch sử được (chỉ xem), người ngoài thì 404.
	if _, err := f.logs.Create(ctx, f.mem, f.parentID, in); err != nil {
		t.Fatal(err)
	}
	if page, err := f.logs.List(ctx, ceo, f.parentID, domain.PageRequest{Page: 1, PageSize: 20}); err != nil || page.Total != 1 {
		t.Fatalf("CEO xem lịch sử: %+v %v", page, err)
	}
	if _, err := f.logs.List(ctx, outsider, f.parentID, domain.PageRequest{Page: 1, PageSize: 20}); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Fatalf("người ngoài xem lịch sử: %v", err)
	}
	// Manager phụ trách cũng được log (không chỉ người nhận).
	if _, err := f.logs.Create(ctx, f.mgr, f.parentID, in); err != nil {
		t.Fatalf("Manager phụ trách log: %v", err)
	}
}

// TC_LOG_107..109 — biên trên được lưu nguyên văn: 1440 phút, 500 emoji (code point), HTML không bị biến đổi [CF-08].
func TestTimeLog_BoundaryValuesStored_DB_CF08(t *testing.T) {
	f := newTLFixture(t)
	ctx := t.Context()
	repo := NewTimeLogRepo(f.pool)
	cases := []struct {
		name    string
		minutes int
		note    string
	}{
		{"TC_LOG_107 1440 phút", 1440, "Cả ngày"},
		{"TC_LOG_108 500 emoji", 1, strings.Repeat("😀", 500)},
		{"TC_LOG_109 HTML + tiếng Việt", 60, `<script>alert("xss")</script> Sửa lỗi đăng nhập`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, err := f.logs.Create(ctx, f.mem, f.parentID, timelog.Input{Minutes: c.minutes, Note: c.note})
			if err != nil {
				t.Fatal(err)
			}
			got, err := repo.GetLog(ctx, l.ID)
			if err != nil || got.Minutes != c.minutes || got.Note != c.note {
				t.Fatalf("lưu sai: %+v %v", got, err)
			}
		})
	}
	// CHECK ở CSDL: 1441 phút và mô tả rỗng bị chặn kể cả khi ghi thẳng.
	_, err := f.pool.Exec(ctx, `UPDATE time_log SET minutes = 1441`)
	wantCheckViolation(t, err, "ck_time_log_minutes")
}

// TC_LOG_110 — Done cách đây 23 giờ (phía trong biên 24 giờ) vẫn log được [CF-08].
func TestTimeLog_DoneWithin24h_DB_CF08(t *testing.T) {
	f := newTLFixture(t)
	ctx := t.Context()
	_, err := f.pool.Exec(ctx, `
		WITH d AS (SELECT id FROM board_column WHERE project_id = $1 AND status_group = 'done' ORDER BY position LIMIT 1)
		UPDATE task SET column_id = (SELECT id FROM d) WHERE id = $2`, f.projID, f.parentID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(ctx, `
		INSERT INTO task_move_history (task_id, project_id, from_column_id, to_column_id, moved_by, moved_at)
		SELECT $1, $2, gen_random_uuid(), t.column_id, $3, now() - interval '23 hours' FROM task t WHERE t.id = $1`,
		f.parentID, f.projID, f.mgr.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.logs.Create(ctx, f.mem, f.parentID, timelog.Input{Minutes: 10, Note: "Kịp giờ"}); err != nil {
		t.Fatalf("Done 23h vẫn phải log được: %v", err)
	}
}

// TC_LOG_111 — lịch sử phân trang, mới nhất trước, Total đúng [CF-08].
func TestTimeLog_ListPagination_DB_CF08(t *testing.T) {
	f := newTLFixture(t)
	ctx := t.Context()
	var ids []uuid.UUID
	for i, note := range []string{"Một", "Hai", "Ba"} {
		l, err := f.logs.Create(ctx, f.mem, f.parentID, timelog.Input{Minutes: 10 + i, Note: note})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, l.ID)
	}
	p1, err := f.logs.List(ctx, f.mem, f.parentID, domain.PageRequest{Page: 1, PageSize: 2})
	if err != nil || p1.Total != 3 || len(p1.Items) != 2 || p1.Items[0].ID != ids[2] || p1.Items[1].ID != ids[1] {
		t.Fatalf("trang 1: %+v %v", p1, err)
	}
	p2, err := f.logs.List(ctx, f.mem, f.parentID, domain.PageRequest{Page: 2, PageSize: 2})
	if err != nil || p2.Total != 3 || len(p2.Items) != 1 || p2.Items[0].ID != ids[0] {
		t.Fatalf("trang 2: %+v %v", p2, err)
	}
	p3, err := f.logs.List(ctx, f.mem, f.parentID, domain.PageRequest{Page: 3, PageSize: 2})
	if err != nil || p3.Total != 3 || len(p3.Items) != 0 {
		t.Fatalf("trang 3 (ngoài phạm vi): %+v %v", p3, err)
	}
}
