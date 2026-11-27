package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/comment"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/notification"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/subtask"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// CF-07 — repo cho công việc con, bình luận, thông báo.

// insertNotifications ghi cùng một thông báo cho nhiều người; danh sách rỗng thì bỏ qua.
func insertNotifications(ctx context.Context, q sqlcgen.Querier, n domain.NewNotification) error {
	if len(n.UserIDs) == 0 {
		return nil
	}
	actor := n.ActorID
	err := q.InsertNotifications(ctx, sqlcgen.InsertNotificationsParams{
		UserIds: n.UserIDs, Type: string(n.Type), TaskID: n.TaskID, ProjectID: n.ProjectID,
		ActorID: nullUUID(&actor), Message: n.Message,
	})
	if err != nil {
		return fmt.Errorf("postgres insert notifications: %w", err)
	}
	return nil
}

// AddNotification ghi thông báo "được giao việc" trong cùng transaction với thay đổi công việc.
func (r *TaskRepo) AddNotification(ctx context.Context, n domain.NewNotification) error {
	return insertNotifications(ctx, r.q, n)
}

// SubtaskRepo cài đặt subtask.Store. Nhúng TaskRepo để dùng lại Get/Lock/LockProject/ColumnGroup/FirstTodoColumn.
type SubtaskRepo struct{ *TaskRepo }

var _ subtask.Store = SubtaskRepo{}

// NewSubtaskRepo tạo repo trên pool hoặc trên một transaction.
func NewSubtaskRepo(db sqlcgen.DBTX) SubtaskRepo { return SubtaskRepo{NewTaskRepo(db)} }

// SubtaskTx cài đặt subtask.Transactor.
type SubtaskTx struct{ db txBeginner }

var _ subtask.Transactor = (*SubtaskTx)(nil)

// NewSubtaskTx tạo Transactor.
func NewSubtaskTx(db txBeginner) *SubtaskTx { return &SubtaskTx{db: db} }

// InTx chạy fn trong một transaction; commit khi fn trả nil.
func (t *SubtaskTx) InTx(ctx context.Context, fn func(subtask.Store) error) error {
	if err := pgx.BeginFunc(ctx, t.db, func(tx pgx.Tx) error {
		return fn(NewSubtaskRepo(tx))
	}); err != nil {
		return fmt.Errorf("subtask tx: %w", err)
	}
	return nil
}

// List trả các con của một việc cha.
func (r SubtaskRepo) List(ctx context.Context, parentID uuid.UUID) ([]domain.Task, error) {
	rows, err := r.q.ListSubtasks(ctx, uuid.NullUUID{UUID: parentID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("postgres.SubtaskRepo list: %w", err)
	}
	return r.withLabels(ctx, rows)
}

// Create thêm việc con cùng dự án với cha, kế thừa người nhận của cha.
func (r SubtaskRepo) Create(ctx context.Context, parent domain.Task, columnID, createdBy uuid.UUID, title, description string) (uuid.UUID, error) {
	pid := parent.ID
	id, err := r.q.CreateSubtask(ctx, sqlcgen.CreateSubtaskParams{
		ProjectID: parent.ProjectID, ColumnID: columnID, ParentTaskID: nullUUID(&pid),
		Title: title, Description: description, AssigneeID: nullUUID(parent.AssigneeID), CreatedBy: createdBy,
	})
	if err == nil {
		return id, nil
	}
	if checkViolation(err) == "ck_task_one_level" {
		return uuid.Nil, domain.ErrSubtaskParentNotRoot
	}
	switch fkViolation(err) {
	case "fk_task_column":
		return uuid.Nil, domain.ErrColumnNotFound
	case "fk_task_parent":
		return uuid.Nil, domain.ErrTaskNotFound
	}
	return uuid.Nil, fmt.Errorf("postgres.SubtaskRepo create: %w", err)
}

// Move đổi cột của việc con (truy vấn chỉ khớp dòng có parent_task_id).
func (r SubtaskRepo) Move(ctx context.Context, id, columnID uuid.UUID) error {
	n, err := r.q.MoveSubtask(ctx, sqlcgen.MoveSubtaskParams{ID: id, ColumnID: columnID})
	if fkViolation(err) == "fk_task_column" {
		return domain.ErrColumnNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres.SubtaskRepo move: %w", err)
	}
	return affected(n, domain.ErrTaskNotFound)
}

// Delete xóa việc con.
func (r SubtaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteSubtask(ctx, id)
	if err != nil {
		return fmt.Errorf("postgres.SubtaskRepo delete: %w", err)
	}
	return affected(n, domain.ErrTaskNotFound)
}

// CommentRepo cài đặt comment.Store.
type CommentRepo struct{ q sqlcgen.Querier }

var _ comment.Store = (*CommentRepo)(nil)

// NewCommentRepo tạo repo trên pool hoặc trên một transaction.
func NewCommentRepo(db sqlcgen.DBTX) *CommentRepo { return &CommentRepo{q: sqlcgen.New(db)} }

// CommentTx cài đặt comment.Transactor.
type CommentTx struct{ db txBeginner }

var _ comment.Transactor = (*CommentTx)(nil)

// NewCommentTx tạo Transactor.
func NewCommentTx(db txBeginner) *CommentTx { return &CommentTx{db: db} }

// InTx chạy fn trong một transaction; commit khi fn trả nil.
func (t *CommentTx) InTx(ctx context.Context, fn func(comment.Store) error) error {
	if err := pgx.BeginFunc(ctx, t.db, func(tx pgx.Tx) error {
		return fn(NewCommentRepo(tx))
	}); err != nil {
		return fmt.Errorf("comment tx: %w", err)
	}
	return nil
}

// GetTask trả công việc (không cần nhãn).
func (r *CommentRepo) GetTask(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	v, err := r.q.GetTask(ctx, id)
	if err != nil {
		return domain.Task{}, taskErr(err)
	}
	return toTask(v), nil
}

// List trả bình luận cũ trước, mới sau.
func (r *CommentRepo) List(ctx context.Context, taskID uuid.UUID, p domain.PageRequest) ([]domain.Comment, int, error) {
	total, err := r.q.CountTaskComments(ctx, taskID)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.CommentRepo count: %w", err)
	}
	off, lim := limits(p)
	rows, err := r.q.ListTaskComments(ctx, sqlcgen.ListTaskCommentsParams{TaskID: taskID, PageOffset: off, PageLimit: lim})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.CommentRepo list: %w", err)
	}
	out := make([]domain.Comment, 0, len(rows))
	for _, c := range rows {
		out = append(out, domain.Comment{
			ID: c.ID, TaskID: c.TaskID, AuthorID: c.AuthorID, AuthorName: c.AuthorName,
			Body: c.Body, CreatedAt: c.CreatedAt.Time,
		})
	}
	return out, int(total), nil
}

// Create lưu bình luận rồi đọc lại kèm tên tác giả.
func (r *CommentRepo) Create(ctx context.Context, taskID, authorID uuid.UUID, body string) (domain.Comment, error) {
	row, err := r.q.CreateTaskComment(ctx, sqlcgen.CreateTaskCommentParams{TaskID: taskID, AuthorID: authorID, Body: body})
	if fkViolation(err) == "fk_task_comment_task" {
		return domain.Comment{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Comment{}, fmt.Errorf("postgres.CommentRepo create: %w", err)
	}
	c, err := r.q.GetTaskComment(ctx, row.ID)
	if err != nil {
		return domain.Comment{}, fmt.Errorf("postgres.CommentRepo get: %w", err)
	}
	return domain.Comment{
		ID: c.ID, TaskID: c.TaskID, AuthorID: c.AuthorID, AuthorName: c.AuthorName,
		Body: c.Body, CreatedAt: c.CreatedAt.Time,
	}, nil
}

// AddNotification ghi thông báo bình luận trong cùng transaction.
func (r *CommentRepo) AddNotification(ctx context.Context, n domain.NewNotification) error {
	return insertNotifications(ctx, r.q, n)
}

// NotificationRepo cài đặt notification.Store.
type NotificationRepo struct{ q sqlcgen.Querier }

var _ notification.Store = (*NotificationRepo)(nil)

// NewNotificationRepo tạo NotificationRepo.
func NewNotificationRepo(db sqlcgen.DBTX) *NotificationRepo {
	return &NotificationRepo{q: sqlcgen.New(db)}
}

// List trả thông báo của userID, mới nhất trước.
func (r *NotificationRepo) List(ctx context.Context, userID uuid.UUID, unreadOnly bool, p domain.PageRequest) ([]domain.Notification, int, error) {
	total, err := r.q.CountNotifications(ctx, sqlcgen.CountNotificationsParams{UserID: userID, UnreadOnly: unreadOnly})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.NotificationRepo count: %w", err)
	}
	off, lim := limits(p)
	rows, err := r.q.ListNotifications(ctx, sqlcgen.ListNotificationsParams{
		UserID: userID, UnreadOnly: unreadOnly, PageOffset: off, PageLimit: lim,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.NotificationRepo list: %w", err)
	}
	out := make([]domain.Notification, 0, len(rows))
	for _, n := range rows {
		out = append(out, domain.Notification{
			ID: n.ID, Type: domain.NotificationType(n.Type), TaskID: n.TaskID, TaskTitle: n.TaskTitle,
			ProjectID: n.ProjectID, ProjectName: n.ProjectName, ActorID: uuidPtr(n.ActorID),
			ActorName: n.ActorName, Message: n.Message, ReadAt: timePtr(n.ReadAt), CreatedAt: n.CreatedAt.Time,
		})
	}
	return out, int(total), nil
}

// CountUnread đếm thông báo chưa đọc.
func (r *NotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := r.q.CountUnreadNotifications(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("postgres.NotificationRepo unread: %w", err)
	}
	return int(n), nil
}

// MarkRead đánh dấu một thông báo của userID là đã đọc (đã đọc rồi vẫn tính là có).
func (r *NotificationRepo) MarkRead(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	n, err := r.q.MarkNotificationRead(ctx, sqlcgen.MarkNotificationReadParams{ID: id, UserID: userID})
	if err != nil {
		return false, fmt.Errorf("postgres.NotificationRepo mark read: %w", err)
	}
	return n > 0, nil
}

// MarkAllRead đánh dấu mọi thông báo chưa đọc của userID.
func (r *NotificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := r.q.MarkAllNotificationsRead(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("postgres.NotificationRepo mark all: %w", err)
	}
	return int(n), nil
}
