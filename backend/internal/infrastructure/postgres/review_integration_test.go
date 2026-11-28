//go:build integration

package postgres

import (
	"errors"
	"testing"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
)

// Đưa việc vào "Chờ nghiệm thu" rồi Từ chối, Duyệt — kiểm bảng task_review, task_move_history và thông báo thật.
func TestReviewRepo_RejectThenApprove_CF09(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	svc := kanban.NewReviewService(NewReviewRepo(e.db), NewReviewTx(e.db), e.projects, realtime.NewHub())
	tk := e.newTask(t, "Nghiệm thu")
	if _, err := e.move(t, e.mgr, tk.ID, "In Progress", 0); err != nil {
		t.Fatal(err)
	}
	cur, err := e.move(t, e.mgr, tk.ID, "Chờ nghiệm thu", 0)
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Reject(ctx, e.mgr, tk.ID, kanban.ReviewInput{Version: cur.Version, Reason: "  Thiếu test  "})
	if err != nil {
		t.Fatal(err)
	}
	if got.ColumnID != e.cols["In Progress"] {
		t.Fatalf("Từ chối → cột %v, want In Progress", got.ColumnID)
	}
	var notes int
	if err := e.db.QueryRow(ctx, `SELECT count(*) FROM notification WHERE task_id = $1 AND type = $2 AND user_id = $3`,
		tk.ID, string(domain.NotificationReviewRejected), e.mem.UserID).Scan(&notes); err != nil || notes != 1 {
		t.Fatalf("thông báo từ chối = %d %v, want 1", notes, err)
	}

	// Không ở cột Chờ nghiệm thu → không được Duyệt.
	if _, err := svc.Approve(ctx, e.mgr, tk.ID, kanban.ReviewInput{Version: got.Version}); !errors.Is(err, domain.ErrTaskNotInReview) {
		t.Fatalf("Approve ngoài cột review = %v, want ErrTaskNotInReview", err)
	}

	cur, err = e.move(t, e.mgr, tk.ID, "Chờ nghiệm thu", 0)
	if err != nil {
		t.Fatal(err)
	}
	done, err := svc.Approve(ctx, e.mgr, tk.ID, kanban.ReviewInput{Version: cur.Version})
	if err != nil {
		t.Fatal(err)
	}
	if done.ColumnID != e.cols["Done"] {
		t.Fatalf("Duyệt → cột %v, want Done", done.ColumnID)
	}

	hist, err := svc.Reviews(ctx, e.mem, tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 2 || hist[0].Decision != domain.ReviewApproved || hist[0].Reason != nil ||
		hist[1].Decision != domain.ReviewRejected || hist[1].Reason == nil || *hist[1].Reason != "Thiếu test" {
		t.Fatalf("lịch sử = %+v", hist)
	}
	if hist[0].ReviewerName == "" {
		t.Fatal("thiếu tên người nghiệm thu")
	}
	var moves int
	if err := e.db.QueryRow(ctx, `SELECT count(*) FROM task_move_history WHERE task_id = $1`, tk.ID).Scan(&moves); err != nil || moves != 5 {
		t.Fatalf("task_move_history = %d %v, want 5 (2 đi tới + Từ chối + quay lại + Duyệt)", moves, err)
	}

	// Cột "Chờ nghiệm thu" đã trống nhưng còn lịch sử nghiệm thu: FK RESTRICT chặn xóa để giữ số liệu tỷ lệ làm lại.
	err = NewProjectRepo(e.db).DeleteColumn(ctx, e.p.ID, e.cols["Chờ nghiệm thu"])
	if !errors.Is(err, domain.ErrColumnHasReviews) {
		t.Fatalf("xóa cột có lịch sử nghiệm thu = %v, want ErrColumnHasReviews", err)
	}
}
