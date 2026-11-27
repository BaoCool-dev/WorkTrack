package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/timelog"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// CF-08 — repo cho Log time.

// TimeLogRepo cài đặt timelog.Store. Nhúng TaskRepo để dùng lại Get/Lock/LockProject.
type TimeLogRepo struct{ *TaskRepo }

var _ timelog.Store = TimeLogRepo{}

// NewTimeLogRepo tạo repo trên pool hoặc trên một transaction.
func NewTimeLogRepo(db sqlcgen.DBTX) TimeLogRepo { return TimeLogRepo{NewTaskRepo(db)} }

// TimeLogTx cài đặt timelog.Transactor.
type TimeLogTx struct{ db txBeginner }

var _ timelog.Transactor = (*TimeLogTx)(nil)

// NewTimeLogTx tạo Transactor.
func NewTimeLogTx(db txBeginner) *TimeLogTx { return &TimeLogTx{db: db} }

// InTx chạy fn trong một transaction; commit khi fn trả nil.
func (t *TimeLogTx) InTx(ctx context.Context, fn func(timelog.Store) error) error {
	if err := pgx.BeginFunc(ctx, t.db, func(tx pgx.Tx) error {
		return fn(NewTimeLogRepo(tx))
	}); err != nil {
		return fmt.Errorf("timelog tx: %w", err)
	}
	return nil
}

// LastDoneAt trả lần gần nhất công việc vào cột nhóm done; nil khi chưa có lịch sử.
func (r TimeLogRepo) LastDoneAt(ctx context.Context, taskID uuid.UUID) (*time.Time, error) {
	v, err := r.q.LastDoneAt(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("postgres.TimeLogRepo last done: %w", err)
	}
	return timePtr(v), nil
}

// Create lưu một dòng log. CHECK ở CSDL là lưới an toàn sau validate ở domain.
func (r TimeLogRepo) Create(ctx context.Context, l timelog.NewLog) (uuid.UUID, error) {
	id, err := r.q.CreateTimeLog(ctx, sqlcgen.CreateTimeLogParams{
		TaskID: l.TaskID, ProjectID: l.ProjectID, UserID: l.UserID,
		Minutes: int32(l.Minutes), //nolint:gosec // domain giới hạn 1..1440
		Note:    l.Note, WorkDate: pgtype.Date{Time: l.WorkDate, Valid: true},
	})
	switch checkViolation(err) {
	case "ck_time_log_minutes":
		return uuid.Nil, domain.ErrInvalidLogMinutes
	case "ck_time_log_note":
		return uuid.Nil, domain.ErrInvalidLogNote
	}
	if fkViolation(err) == "fk_time_log_task" {
		return uuid.Nil, domain.ErrTaskNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres.TimeLogRepo create: %w", err)
	}
	return id, nil
}

// GetLog trả một dòng log kèm tên công việc và người ghi.
func (r TimeLogRepo) GetLog(ctx context.Context, id uuid.UUID) (domain.TimeLog, error) {
	v, err := r.q.GetTimeLog(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.TimeLog{}, domain.ErrTimeLogNotFound
	}
	if err != nil {
		return domain.TimeLog{}, fmt.Errorf("postgres.TimeLogRepo get: %w", err)
	}
	return toTimeLog(sqlcgen.ListTaskTimeLogsRow(v)), nil
}

// ListLogs trả lịch sử log (gồm log của việc con nếu taskID là việc cha), mới nhất trước.
func (r TimeLogRepo) ListLogs(ctx context.Context, taskID uuid.UUID, p domain.PageRequest) ([]domain.TimeLog, int, error) {
	total, err := r.q.CountTaskTimeLogs(ctx, taskID)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.TimeLogRepo count: %w", err)
	}
	off, lim := limits(p)
	rows, err := r.q.ListTaskTimeLogs(ctx, sqlcgen.ListTaskTimeLogsParams{TaskID: taskID, SkipRows: off, MaxRows: lim})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.TimeLogRepo list: %w", err)
	}
	out := make([]domain.TimeLog, 0, len(rows))
	for _, v := range rows {
		out = append(out, toTimeLog(v))
	}
	return out, int(total), nil
}

// DeleteLog xóa một dòng log.
func (r TimeLogRepo) DeleteLog(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteTimeLog(ctx, id)
	if err != nil {
		return fmt.Errorf("postgres.TimeLogRepo delete: %w", err)
	}
	return affected(n, domain.ErrTimeLogNotFound)
}

func toTimeLog(v sqlcgen.ListTaskTimeLogsRow) domain.TimeLog {
	return domain.TimeLog{
		ID: v.ID, TaskID: v.TaskID, TaskTitle: v.TaskTitle, ParentID: uuidPtr(v.ParentTaskID),
		ProjectID: v.ProjectID, UserID: v.UserID, UserName: v.UserName, Minutes: int(v.Minutes),
		Note: v.Note, WorkDate: v.WorkDate.Time, CreatedAt: v.CreatedAt.Time,
	}
}
