package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// TaskRepo cũng cài đặt kanban.Store (CF-04, CF-05): bảng Kanban đọc/ghi cùng bảng task.
var _ kanban.Store = (*TaskRepo)(nil)

// KanbanTx cài đặt kanban.Transactor. Tách khỏi TaskTx vì chữ ký InTx khác kiểu Store.
type KanbanTx struct{ db txBeginner }

var _ kanban.Transactor = (*KanbanTx)(nil)

// NewKanbanTx tạo Transactor cho use case Kanban.
func NewKanbanTx(db txBeginner) *KanbanTx { return &KanbanTx{db: db} }

// InTx chạy fn trong một transaction; commit khi fn trả nil.
func (t *KanbanTx) InTx(ctx context.Context, fn func(kanban.Store) error) error {
	if err := pgx.BeginFunc(ctx, t.db, func(tx pgx.Tx) error {
		return fn(NewTaskRepo(tx))
	}); err != nil {
		return fmt.Errorf("kanban tx: %w", err)
	}
	return nil
}

// ListBoard trả các công việc cha của dự án kèm nhãn, theo cột rồi vị trí.
func (r *TaskRepo) ListBoard(ctx context.Context, projectID uuid.UUID, limit int) ([]domain.Task, error) {
	rows, err := r.q.ListBoardTasks(ctx, sqlcgen.ListBoardTasksParams{
		ProjectID: projectID, MaxRows: int32(limit), //nolint:gosec // limit = kanban.BoardMaxTasks
	})
	if err != nil {
		return nil, fmt.Errorf("postgres.TaskRepo list board: %w", err)
	}
	return r.withLabels(ctx, rows)
}

// LockProject khóa dòng dự án — luôn khóa TRƯỚC dòng task để không deadlock.
func (r *TaskRepo) LockProject(ctx context.Context, projectID uuid.UUID) error {
	if _, err := r.q.LockProject(ctx, projectID); err != nil {
		return projectErr(err)
	}
	return nil
}

// ColumnGroup trả nhóm trạng thái của cột; cột không thuộc dự án → ErrColumnNotFound.
func (r *TaskRepo) ColumnGroup(ctx context.Context, projectID, columnID uuid.UUID) (domain.StatusGroup, error) {
	g, err := r.q.GetColumnGroup(ctx, sqlcgen.GetColumnGroupParams{ProjectID: projectID, ID: columnID})
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrColumnNotFound
	}
	if err != nil {
		return "", fmt.Errorf("postgres.TaskRepo column group: %w", err)
	}
	return domain.StatusGroup(g), nil
}

// TransitionAllowed: có dòng (from, to, role) trong workflow_transition của dự án không.
func (r *TaskRepo) TransitionAllowed(ctx context.Context, projectID, from, to uuid.UUID, role domain.TransitionRole) (bool, error) {
	ok, err := r.q.TransitionAllowed(ctx, sqlcgen.TransitionAllowedParams{
		ProjectID: projectID, FromColumnID: from, ToColumnID: to, Role: string(role),
	})
	if err != nil {
		return false, fmt.Errorf("postgres.TaskRepo transition: %w", err)
	}
	return ok, nil
}

// CountProjectWIP đếm công việc cha nhóm in_progress của userID trong một dự án.
func (r *TaskRepo) CountProjectWIP(ctx context.Context, projectID, userID uuid.UUID) (int, error) {
	n, err := r.q.CountProjectWIP(ctx, sqlcgen.CountProjectWIPParams{ProjectID: projectID, AssigneeID: userID})
	if err != nil {
		return 0, fmt.Errorf("postgres.TaskRepo project wip: %w", err)
	}
	return int(n), nil
}

// MoveColumn đổi cột và tăng version; 0 dòng = version đã đổi → xung đột (khóa dòng đã chắc task tồn tại).
func (r *TaskRepo) MoveColumn(ctx context.Context, id, columnID uuid.UUID, version int) error {
	n, err := r.q.MoveTaskColumn(ctx, sqlcgen.MoveTaskColumnParams{
		ColumnID: columnID, ID: id, Version: int32(version), //nolint:gosec // version là số đếm nhỏ
	})
	if err != nil {
		return fmt.Errorf("postgres.TaskRepo move: %w", err)
	}
	return affected(n, domain.ErrTaskVersionConflict)
}

// ColumnTaskIDs trả thứ tự hiện tại của công việc cha trong cột.
func (r *TaskRepo) ColumnTaskIDs(ctx context.Context, projectID, columnID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := r.q.ListColumnTaskIDs(ctx, sqlcgen.ListColumnTaskIDsParams{ProjectID: projectID, ColumnID: columnID})
	if err != nil {
		return nil, fmt.Errorf("postgres.TaskRepo column ids: %w", err)
	}
	return ids, nil
}

// SetPositions ghi vị trí 0..n-1 theo thứ tự ids.
func (r *TaskRepo) SetPositions(ctx context.Context, ids []uuid.UUID) error {
	if err := r.q.SetTaskPositions(ctx, ids); err != nil {
		return fmt.Errorf("postgres.TaskRepo positions: %w", err)
	}
	return nil
}

// InsertMoveHistory ghi một dòng lịch sử chuyển cột.
func (r *TaskRepo) InsertMoveHistory(ctx context.Context, m kanban.MoveRecord) error {
	err := r.q.InsertTaskMoveHistory(ctx, sqlcgen.InsertTaskMoveHistoryParams{
		TaskID: m.TaskID, ProjectID: m.ProjectID, FromColumnID: m.FromColumnID, ToColumnID: m.ToColumnID, MovedBy: m.MovedBy,
	})
	if err != nil {
		return fmt.Errorf("postgres.TaskRepo move history: %w", err)
	}
	return nil
}
