package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// CF-09 — repo cho nghiệm thu. Nhúng TaskRepo để dùng lại toàn bộ kanban.Store
// (khóa, luật chuyển cột, WIP, lịch sử di chuyển) — không viết lại luật chuyển cột.

// ReviewRepo cài đặt kanban.ReviewStore.
type ReviewRepo struct{ *TaskRepo }

var _ kanban.ReviewStore = ReviewRepo{}

// NewReviewRepo tạo repo trên pool hoặc trên một transaction.
func NewReviewRepo(db sqlcgen.DBTX) ReviewRepo { return ReviewRepo{NewTaskRepo(db)} }

// ReviewTx cài đặt kanban.ReviewTransactor.
type ReviewTx struct{ db txBeginner }

var _ kanban.ReviewTransactor = (*ReviewTx)(nil)

// NewReviewTx tạo Transactor.
func NewReviewTx(db txBeginner) *ReviewTx { return &ReviewTx{db: db} }

// InTx chạy fn trong một transaction; commit khi fn trả nil.
func (t *ReviewTx) InTx(ctx context.Context, fn func(kanban.ReviewStore) error) error {
	if err := pgx.BeginFunc(ctx, t.db, func(tx pgx.Tx) error {
		return fn(NewReviewRepo(tx))
	}); err != nil {
		return fmt.Errorf("review tx: %w", err)
	}
	return nil
}

// ListColumns trả cột của bảng theo vị trí — dùng để xác định cột "Chờ nghiệm thu".
func (r ReviewRepo) ListColumns(ctx context.Context, projectID uuid.UUID) ([]domain.BoardColumn, error) {
	rows, err := r.q.ListBoardColumns(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("postgres.ReviewRepo list columns: %w", err)
	}
	out := make([]domain.BoardColumn, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.BoardColumn{
			ID: x.ID, ProjectID: x.ProjectID, Name: x.Name, Position: int(x.Position),
			StatusGroup: domain.StatusGroup(x.StatusGroup), TaskCount: int(x.TaskCount),
		})
	}
	return out, nil
}

// InsertReview ghi một lần Duyệt/Từ chối; lý do NULL khi Duyệt.
func (r ReviewRepo) InsertReview(ctx context.Context, rec kanban.ReviewRecord) error {
	reason := pgtype.Text{}
	if rec.Reason != nil {
		reason = text(*rec.Reason)
	}
	err := r.q.InsertTaskReview(ctx, sqlcgen.InsertTaskReviewParams{
		TaskID: rec.TaskID, ProjectID: rec.ProjectID, ReviewerID: rec.ReviewerID,
		Decision: string(rec.Decision), Reason: reason,
		FromColumnID: rec.FromColumnID, ToColumnID: rec.ToColumnID,
	})
	if err != nil {
		return fmt.Errorf("postgres.ReviewRepo insert: %w", err)
	}
	return nil
}

// ListReviews trả lịch sử nghiệm thu, mới nhất trước.
func (r ReviewRepo) ListReviews(ctx context.Context, taskID uuid.UUID) ([]domain.TaskReview, error) {
	rows, err := r.q.ListTaskReviews(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("postgres.ReviewRepo list: %w", err)
	}
	out := make([]domain.TaskReview, 0, len(rows))
	for _, x := range rows {
		var reason *string
		if x.Reason.Valid {
			s := x.Reason.String
			reason = &s
		}
		out = append(out, domain.TaskReview{
			ID: x.ID, TaskID: x.TaskID, ReviewerID: x.ReviewerID, ReviewerName: x.ReviewerName,
			Decision: domain.ReviewDecision(x.Decision), Reason: reason, CreatedAt: x.CreatedAt.Time,
		})
	}
	return out, nil
}
