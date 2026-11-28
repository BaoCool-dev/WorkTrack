//go:build integration

package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/subtask"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
)

// QA bổ sung CF-09 trên PostgreSQL thật: bảng quyết định vai trò, việc con chưa xong, version, CHECK.

func newReviewSvc(e *kanbanEnv) *kanban.ReviewService {
	return kanban.NewReviewService(NewReviewRepo(e.db), NewReviewTx(e.db), e.projects, realtime.NewHub())
}

// toReview đưa việc cha mới vào cột "Chờ nghiệm thu" bằng thao tác của chính Member được giao.
func toReview(t *testing.T, e *kanbanEnv, title string) domain.Task {
	t.Helper()
	tk := e.newTask(t, title)
	if _, err := e.move(t, e.mem, tk.ID, "In Progress", 0); err != nil {
		t.Fatalf("member → In Progress: %v", err)
	}
	cur, err := e.move(t, e.mem, tk.ID, "Chờ nghiệm thu", 0)
	if err != nil {
		t.Fatalf("member → Chờ nghiệm thu: %v", err)
	}
	return cur
}

func countReviews(t *testing.T, e *kanbanEnv, taskID any) int {
	t.Helper()
	var n int
	if err := e.db.QueryRow(t.Context(), `SELECT count(*) FROM task_review WHERE task_id = $1`, taskID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Member (người nhận), Manager khác là thành viên, CEO → 403; Admin → 404. Không dòng task_review nào được ghi.
func TestReviewRepo_RoleDecisionTable_DB_CF09(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	users := NewUserRepo(e.db)
	otherMgr := authz.Principal{UserID: newIntegrationUser(t, users, "mgr2@worktrack.local", domain.RoleManager), Role: domain.RoleManager}
	if err := e.projects.AddMember(ctx, e.mgr, e.p.ID, otherMgr.UserID); err != nil {
		t.Fatal(err)
	}
	ceo := authz.Principal{UserID: newIntegrationUser(t, users, "ceo@worktrack.local", domain.RoleCEO), Role: domain.RoleCEO}
	admin := authz.Principal{UserID: newIntegrationUser(t, users, "admin@worktrack.local", domain.RoleAdmin), Role: domain.RoleAdmin}
	svc := newReviewSvc(e)
	tk := toReview(t, e, "Phân quyền nghiệm thu")

	for _, c := range []struct {
		name  string
		actor authz.Principal
		want  error
	}{
		{"Member được giao", e.mem, domain.ErrForbidden},
		{"Manager khác (thành viên)", otherMgr, domain.ErrForbidden},
		{"CEO", ceo, domain.ErrForbidden},
		{"Admin", admin, domain.ErrTaskNotFound},
	} {
		t.Run(c.name, func(t *testing.T) {
			if _, err := svc.Approve(ctx, c.actor, tk.ID, kanban.ReviewInput{Version: tk.Version}); !errors.Is(err, c.want) {
				t.Errorf("Approve = %v, want %v", err, c.want)
			}
			if _, err := svc.Reject(ctx, c.actor, tk.ID, kanban.ReviewInput{Version: tk.Version, Reason: "x"}); !errors.Is(err, c.want) {
				t.Errorf("Reject = %v, want %v", err, c.want)
			}
		})
	}
	// CEO xem được lịch sử; Admin không.
	if _, err := svc.Reviews(ctx, ceo, tk.ID); err != nil {
		t.Errorf("CEO xem lịch sử: %v", err)
	}
	if _, err := svc.Reviews(ctx, admin, tk.ID); !errors.Is(err, domain.ErrTaskNotFound) {
		t.Errorf("Admin xem lịch sử = %v, want ErrTaskNotFound", err)
	}
	if n := countReviews(t, e, tk.ID); n != 0 {
		t.Fatalf("task_review = %d, want 0", n)
	}
	cur, err := e.repo.Get(ctx, tk.ID)
	if err != nil || cur.ColumnID != e.cols["Chờ nghiệm thu"] || cur.Version != tk.Version {
		t.Fatalf("việc bị đổi: %+v %v", cur, err)
	}
}

// Còn việc con chưa xong: Member vẫn đưa cha vào Chờ nghiệm thu, Manager vẫn Duyệt; việc con giữ nguyên cột.
func TestReviewRepo_PendingSubtasksDoNotBlock_DB_CF09(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	subs := subtask.NewService(NewSubtaskRepo(e.db), NewSubtaskTx(e.db), e.projects)
	parent := e.newTask(t, "Cha có con dở")
	child, err := subs.Create(ctx, e.mem, parent.ID, subtask.Input{Title: "Con chưa xong"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.move(t, e.mem, parent.ID, "In Progress", 0); err != nil {
		t.Fatal(err)
	}
	cur, err := e.move(t, e.mem, parent.ID, "Chờ nghiệm thu", 0)
	if err != nil {
		t.Fatalf("còn con chưa xong vẫn phải cho vào Chờ nghiệm thu: %v", err)
	}
	if cur.SubtaskCount != 1 || cur.SubtaskDoneCount != 0 {
		t.Fatalf("đếm con = %d/%d, want 0/1", cur.SubtaskDoneCount, cur.SubtaskCount)
	}
	done, err := newReviewSvc(e).Approve(ctx, e.mgr, parent.ID, kanban.ReviewInput{Version: cur.Version})
	if err != nil || done.ColumnID != e.cols["Done"] {
		t.Fatalf("Duyệt khi còn con dở = %+v %v", done, err)
	}
	c, err := e.repo.Get(ctx, child.ID)
	if err != nil || c.ColumnID != child.ColumnID {
		t.Fatalf("việc con bị đổi cột: %+v %v", c, err)
	}
}

// Version cũ (người khác vừa sửa) → 409, không ghi task_review, không thông báo.
func TestReviewRepo_StaleVersion_DB_CF09(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	svc := newReviewSvc(e)
	tk := toReview(t, e, "Xung đột phiên bản")
	if _, err := svc.Reject(ctx, e.mgr, tk.ID, kanban.ReviewInput{Version: tk.Version - 1, Reason: "lý do"}); !errors.Is(err, domain.ErrTaskVersionConflict) {
		t.Fatalf("Reject version cũ = %v, want ErrTaskVersionConflict", err)
	}
	var notes int
	if err := e.db.QueryRow(ctx, `SELECT count(*) FROM notification WHERE task_id = $1 AND type = $2`,
		tk.ID, string(domain.NotificationReviewRejected)).Scan(&notes); err != nil || notes != 0 {
		t.Fatalf("thông báo = %d %v, want 0", notes, err)
	}
	if n := countReviews(t, e, tk.ID); n != 0 {
		t.Fatalf("task_review = %d, want 0", n)
	}
}

// Lưới cuối ở CSDL: ck_task_review_reason và ck_task_review_decision (BVA 0/1/500/501, EP quyết định).
func TestReviewRepo_CheckConstraints_DB_CF09(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	tk := e.newTask(t, "Ràng buộc")
	from, to := e.cols["Chờ nghiệm thu"], e.cols["Done"]
	insert := func(decision string, reason *string) error {
		_, err := e.db.Exec(ctx, `INSERT INTO task_review (task_id, project_id, reviewer_id, decision, reason, from_column_id, to_column_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`, tk.ID, e.p.ID, e.mgr.UserID, decision, reason, from, to)
		return err
	}
	str := func(s string) *string { return &s }
	for _, c := range []struct {
		name       string
		decision   string
		reason     *string
		constraint string // rỗng = phải ghi được
	}{
		{"rejected 1 ký tự", "rejected", str("x"), ""},
		{"rejected 500 ký tự có dấu", "rejected", str(strings.Repeat("ạ", 500)), ""},
		{"approved không lý do", "approved", nil, ""},
		{"rejected rỗng", "rejected", str(""), "ck_task_review_reason"},
		{"rejected NULL", "rejected", nil, "ck_task_review_reason"},
		{"rejected 501 ký tự", "rejected", str(strings.Repeat("a", 501)), "ck_task_review_reason"},
		{"approved kèm lý do", "approved", str("x"), "ck_task_review_reason"},
		{"quyết định lạ", "pending", nil, "ck_task_review_decision"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := insert(c.decision, c.reason)
			if c.constraint == "" {
				if err != nil {
					t.Fatalf("muốn ghi được, got %v", err)
				}
				return
			}
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != c.constraint {
				t.Fatalf("muốn 23514 %s, got %v", c.constraint, err)
			}
		})
	}
}
