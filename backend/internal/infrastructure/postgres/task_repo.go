package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/label"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// TaskRepo cài đặt task.Store (CF-06).
type TaskRepo struct{ q sqlcgen.Querier }

var _ task.Store = (*TaskRepo)(nil)

// NewTaskRepo tạo repo trên pool hoặc trên một transaction.
func NewTaskRepo(db sqlcgen.DBTX) *TaskRepo { return &TaskRepo{q: sqlcgen.New(db)} }

// TaskTx cài đặt task.Transactor.
type TaskTx struct{ db txBeginner }

var _ task.Transactor = (*TaskTx)(nil)

// NewTaskTx tạo Transactor.
func NewTaskTx(db txBeginner) *TaskTx { return &TaskTx{db: db} }

// InTx chạy fn trong một transaction; commit khi fn trả nil.
func (t *TaskTx) InTx(ctx context.Context, fn func(task.Store) error) error {
	if err := pgx.BeginFunc(ctx, t.db, func(tx pgx.Tx) error {
		return fn(NewTaskRepo(tx))
	}); err != nil {
		return fmt.Errorf("task tx: %w", err)
	}
	return nil
}

func taskErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrTaskNotFound
	}
	return fmt.Errorf("postgres.TaskRepo: %w", err)
}

func int2(v *int) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{}
	}
	return pgtype.Int2{Int16: int16(*v), Valid: true} //nolint:gosec // Story Point đã kiểm 1..21
}

func int4(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*v), Valid: true} //nolint:gosec // phút ước tính ≤ 21×40×60
}

func int2Ptr(v pgtype.Int2) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int16)
	return &n
}

func int4Ptr(v pgtype.Int4) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int32)
	return &n
}

func optGroup(g *domain.StatusGroup) pgtype.Text {
	if g == nil {
		return pgtype.Text{}
	}
	return text(string(*g))
}

func search(q string) pgtype.Text {
	if q == "" {
		return pgtype.Text{}
	}
	return text(likeContains(q))
}

func toTask(v sqlcgen.TaskView) domain.Task {
	return domain.Task{
		ID: v.ID, ProjectID: v.ProjectID, ProjectName: v.ProjectName, ParentID: uuidPtr(v.ParentTaskID),
		ColumnID: v.ColumnID, ColumnName: v.ColumnName, StatusGroup: domain.StatusGroup(v.StatusGroup),
		Title: v.Title, Description: v.Description,
		StoryPoint: int2Ptr(v.StoryPoint), EstimatedMinutes: int4Ptr(v.EstimatedMinutes),
		AssigneeID: uuidPtr(v.AssigneeID), AssigneeName: v.AssigneeName, DueDate: datePtr(v.DueDate),
		Labels:    []domain.TaskLabel{},
		CreatedBy: v.CreatedBy, CreatedByName: v.CreatedByName,
		CreatedAt: v.CreatedAt.Time, UpdatedAt: v.UpdatedAt.Time,
	}
}

// withLabels gắn nhãn cho danh sách công việc bằng một truy vấn.
func (r *TaskRepo) withLabels(ctx context.Context, rows []sqlcgen.TaskView) ([]domain.Task, error) {
	out := make([]domain.Task, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	idx := make(map[uuid.UUID]int, len(rows))
	for i, v := range rows {
		out = append(out, toTask(v))
		ids = append(ids, v.ID)
		idx[v.ID] = i
	}
	ls, err := r.q.ListTaskLabels(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("postgres.TaskRepo labels: %w", err)
	}
	for _, l := range ls {
		i := idx[l.TaskID]
		out[i].Labels = append(out[i].Labels, domain.TaskLabel{ID: l.ID, Name: l.Name})
	}
	return out, nil
}

func limits(p domain.PageRequest) (int32, int32) {
	return int32(p.Offset()), int32(p.PageSize) //nolint:gosec // PageRequest giới hạn trong int32
}

// List trả công việc cha của dự án theo bộ lọc.
func (r *TaskRepo) List(ctx context.Context, projectID uuid.UUID, f task.Filter, p domain.PageRequest) ([]domain.Task, int, error) {
	c := sqlcgen.CountProjectTasksParams{
		ProjectID: projectID, Search: search(f.Search), ColumnID: nullUUID(f.ColumnID),
		StatusGroup: optGroup(f.StatusGroup), AssigneeID: nullUUID(f.AssigneeID),
		Unassigned: f.Unassigned, LabelID: nullUUID(f.LabelID),
	}
	total, err := r.q.CountProjectTasks(ctx, c)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.TaskRepo count: %w", err)
	}
	off, lim := limits(p)
	rows, err := r.q.ListProjectTasks(ctx, sqlcgen.ListProjectTasksParams{
		ProjectID: c.ProjectID, Search: c.Search, ColumnID: c.ColumnID, StatusGroup: c.StatusGroup,
		AssigneeID: c.AssigneeID, Unassigned: c.Unassigned, LabelID: c.LabelID,
		PageOffset: off, PageLimit: lim,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.TaskRepo list: %w", err)
	}
	out, err := r.withLabels(ctx, rows)
	return out, int(total), err
}

// ListMine trả công việc cha giao cho userID.
func (r *TaskRepo) ListMine(ctx context.Context, userID uuid.UUID, f task.MyFilter, p domain.PageRequest) ([]domain.Task, int, error) {
	total, err := r.q.CountMyTasks(ctx, sqlcgen.CountMyTasksParams{
		UserID: userID, StatusGroup: optGroup(f.StatusGroup), Search: search(f.Search),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.TaskRepo count mine: %w", err)
	}
	off, lim := limits(p)
	rows, err := r.q.ListMyTasks(ctx, sqlcgen.ListMyTasksParams{
		UserID: userID, StatusGroup: optGroup(f.StatusGroup), Search: search(f.Search),
		PageOffset: off, PageLimit: lim,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.TaskRepo list mine: %w", err)
	}
	out, err := r.withLabels(ctx, rows)
	return out, int(total), err
}

// Get trả một công việc kèm nhãn.
func (r *TaskRepo) Get(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	v, err := r.q.GetTask(ctx, id)
	if err != nil {
		return domain.Task{}, taskErr(err)
	}
	out, err := r.withLabels(ctx, []sqlcgen.TaskView{v})
	if err != nil {
		return domain.Task{}, err
	}
	return out[0], nil
}

// Lock khóa dòng công việc tới hết transaction.
func (r *TaskRepo) Lock(ctx context.Context, id uuid.UUID) error {
	if _, err := r.q.LockTask(ctx, id); err != nil {
		return taskErr(err)
	}
	return nil
}

// FirstTodoColumn trả cột todo đầu tiên của dự án.
func (r *TaskRepo) FirstTodoColumn(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error) {
	id, err := r.q.FirstTodoColumn(ctx, projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, domain.ErrBoardNeedsTodoColumn
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres.TaskRepo first todo column: %w", err)
	}
	return id, nil
}

func (r *TaskRepo) setLabels(ctx context.Context, id uuid.UUID, labels []uuid.UUID) error {
	if err := r.q.DeleteTaskLabels(ctx, id); err != nil {
		return fmt.Errorf("postgres.TaskRepo clear labels: %w", err)
	}
	if len(labels) == 0 {
		return nil
	}
	err := r.q.AddTaskLabels(ctx, sqlcgen.AddTaskLabelsParams{TaskID: id, LabelIds: labels})
	if fkViolation(err) == "fk_task_label_label" {
		return domain.ErrLabelNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres.TaskRepo add labels: %w", err)
	}
	return nil
}

// Create thêm công việc cha (chạy trong transaction để nhãn lỗi thì hủy cả việc).
func (r *TaskRepo) Create(ctx context.Context, projectID, columnID, createdBy uuid.UUID, f task.Fields) (uuid.UUID, error) {
	id, err := r.q.CreateTask(ctx, sqlcgen.CreateTaskParams{
		ProjectID: projectID, ColumnID: columnID, Title: f.Title, Description: f.Description,
		StoryPoint: int2(f.StoryPoint), EstimatedMinutes: int4(f.EstimatedMinutes),
		AssigneeID: nullUUID(f.AssigneeID), DueDate: date(f.DueDate), CreatedBy: createdBy,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres.TaskRepo create: %w", err)
	}
	if err := r.setLabels(ctx, id, f.LabelIDs); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// Update thay toàn bộ thông tin và nhãn của công việc.
func (r *TaskRepo) Update(ctx context.Context, id uuid.UUID, f task.Fields) error {
	n, err := r.q.UpdateTask(ctx, sqlcgen.UpdateTaskParams{
		ID: id, Title: f.Title, Description: f.Description,
		StoryPoint: int2(f.StoryPoint), EstimatedMinutes: int4(f.EstimatedMinutes),
		AssigneeID: nullUUID(f.AssigneeID), DueDate: date(f.DueDate),
	})
	if err != nil {
		return fmt.Errorf("postgres.TaskRepo update: %w", err)
	}
	if err := affected(n, domain.ErrTaskNotFound); err != nil {
		return err
	}
	return r.setLabels(ctx, id, f.LabelIDs)
}

// SetAssignee đặt/bỏ người nhận việc.
func (r *TaskRepo) SetAssignee(ctx context.Context, id uuid.UUID, assignee *uuid.UUID) error {
	n, err := r.q.SetTaskAssignee(ctx, sqlcgen.SetTaskAssigneeParams{ID: id, AssigneeID: nullUUID(assignee)})
	if err != nil {
		return fmt.Errorf("postgres.TaskRepo set assignee: %w", err)
	}
	return affected(n, domain.ErrTaskNotFound)
}

// GetAssignable trả vai trò + trạng thái của thành viên dự án; ok = false nếu không phải thành viên.
func (r *TaskRepo) GetAssignable(ctx context.Context, projectID, userID uuid.UUID) (task.Assignable, bool, error) {
	x, err := r.q.GetAssignableMember(ctx, sqlcgen.GetAssignableMemberParams{ProjectID: projectID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return task.Assignable{}, false, nil
	}
	if err != nil {
		return task.Assignable{}, false, fmt.Errorf("postgres.TaskRepo assignable: %w", err)
	}
	return task.Assignable{Role: domain.Role(x.RoleCode), Status: domain.UserStatus(x.Status)}, true, nil
}

// CountWIP đếm công việc cha nhóm in_progress của userID.
func (r *TaskRepo) CountWIP(ctx context.Context, userID uuid.UUID) (int, error) {
	n, err := r.q.CountWIP(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("postgres.TaskRepo wip: %w", err)
	}
	return int(n), nil
}

// LabelRepo cài đặt label.Store.
type LabelRepo struct{ q sqlcgen.Querier }

var _ label.Store = (*LabelRepo)(nil)

// NewLabelRepo tạo LabelRepo.
func NewLabelRepo(db sqlcgen.DBTX) *LabelRepo { return &LabelRepo{q: sqlcgen.New(db)} }

// List trả danh mục nhãn kèm số công việc.
func (r *LabelRepo) List(ctx context.Context) ([]domain.Label, error) {
	rows, err := r.q.ListLabels(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres.LabelRepo list: %w", err)
	}
	out := make([]domain.Label, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.Label{ID: x.ID, Name: x.Name, TaskCount: int(x.TaskCount)})
	}
	return out, nil
}

// Create thêm nhãn.
func (r *LabelRepo) Create(ctx context.Context, name string) (uuid.UUID, error) {
	id, err := r.q.CreateLabel(ctx, name)
	if isUniqueViolation(err) {
		return uuid.Nil, domain.ErrLabelNameTaken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres.LabelRepo create: %w", err)
	}
	return id, nil
}

// Update đổi tên nhãn.
func (r *LabelRepo) Update(ctx context.Context, id uuid.UUID, name string) error {
	n, err := r.q.UpdateLabel(ctx, sqlcgen.UpdateLabelParams{ID: id, Name: name})
	if isUniqueViolation(err) {
		return domain.ErrLabelNameTaken
	}
	if err != nil {
		return fmt.Errorf("postgres.LabelRepo update: %w", err)
	}
	return affected(n, domain.ErrLabelNotFound)
}

// Delete xóa nhãn; còn gắn vào công việc → ErrLabelInUse.
func (r *LabelRepo) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteLabel(ctx, id)
	if fkViolation(err) == "fk_task_label_label" {
		return domain.ErrLabelInUse
	}
	if err != nil {
		return fmt.Errorf("postgres.LabelRepo delete: %w", err)
	}
	return affected(n, domain.ErrLabelNotFound)
}

// SettingRepo cài đặt setting.Store.
type SettingRepo struct{ q sqlcgen.Querier }

var _ setting.Store = (*SettingRepo)(nil)

// NewSettingRepo tạo SettingRepo.
func NewSettingRepo(db sqlcgen.DBTX) *SettingRepo { return &SettingRepo{q: sqlcgen.New(db)} }

// GetInt đọc tham số số nguyên.
func (r *SettingRepo) GetInt(ctx context.Context, key string) (int, time.Time, error) {
	x, err := r.q.GetIntSetting(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, time.Time{}, fmt.Errorf("postgres.SettingRepo: thiếu tham số %q: %w", key, err)
	}
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("postgres.SettingRepo get: %w", err)
	}
	return int(x.IntValue), x.UpdatedAt.Time, nil
}

// SetInt ghi tham số số nguyên; vi phạm CHECK → ErrInvalidSetting.
func (r *SettingRepo) SetInt(ctx context.Context, key string, v int, by uuid.UUID) error {
	n, err := r.q.SetIntSetting(ctx, sqlcgen.SetIntSettingParams{
		Key: key, IntValue: int32(v), UpdatedBy: uuid.NullUUID{UUID: by, Valid: true}, //nolint:gosec // đã kiểm 1..40
	})
	if err != nil {
		return fmt.Errorf("postgres.SettingRepo set: %w", err)
	}
	return affected(n, domain.ErrInvalidSetting)
}
