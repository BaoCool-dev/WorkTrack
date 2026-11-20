package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// ProjectRepo cài đặt project.Store (CF-03).
type ProjectRepo struct{ q sqlcgen.Querier }

var _ project.Store = (*ProjectRepo)(nil)

// NewProjectRepo tạo repo trên pool hoặc trên một transaction.
func NewProjectRepo(db sqlcgen.DBTX) *ProjectRepo { return &ProjectRepo{q: sqlcgen.New(db)} }

// txBeginner là *pgxpool.Pool (hoặc pgx.Tx cho savepoint trong test).
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// ProjectTx cài đặt project.Transactor: mở transaction, tạo ProjectRepo trên tx, commit khi fn trả nil.
type ProjectTx struct{ db txBeginner }

var _ project.Transactor = (*ProjectTx)(nil)

// NewProjectTx tạo Transactor.
func NewProjectTx(db txBeginner) *ProjectTx { return &ProjectTx{db: db} }

// InTx chạy fn trong một transaction. Ràng buộc DEFERRABLE (vị trí cột) được kiểm khi commit.
func (t *ProjectTx) InTx(ctx context.Context, fn func(project.Store) error) error {
	// Bọc bằng %w: lỗi nghiệp vụ domain vẫn so sánh được bằng errors.Is; lỗi commit có ngữ cảnh.
	if err := pgx.BeginFunc(ctx, t.db, func(tx pgx.Tx) error {
		return fn(NewProjectRepo(tx))
	}); err != nil {
		return fmt.Errorf("project tx: %w", err)
	}
	return nil
}

func date(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func datePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	t := d.Time
	return &t
}

func projectErr(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.ErrProjectNotFound
	case isUniqueViolation(err):
		return domain.ErrProjectNameTaken
	}
	return fmt.Errorf("postgres.ProjectRepo: %w", err)
}

func columnErr(err error) error {
	if isUniqueViolation(err) {
		return domain.ErrColumnNameTaken
	}
	return fmt.Errorf("postgres.ProjectRepo: %w", err)
}

// List trả một trang dự án và tổng số.
func (r *ProjectRepo) List(ctx context.Context, f project.Filter, p domain.PageRequest) ([]domain.Project, int, error) {
	search := pgtype.Text{}
	if f.Search != "" {
		search = text(likeContains(f.Search))
	}
	status := pgtype.Text{}
	if f.Status != nil {
		status = text(string(*f.Status))
	}
	vis := nullUUID(f.VisibleTo)
	total, err := r.q.CountProjects(ctx, sqlcgen.CountProjectsParams{VisibleTo: vis, Search: search, Status: status})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.ProjectRepo count: %w", err)
	}
	rows, err := r.q.ListProjects(ctx, sqlcgen.ListProjectsParams{
		VisibleTo: vis, Search: search, Status: status,
		PageOffset: int32(p.Offset()), PageLimit: int32(p.PageSize), //nolint:gosec // PageRequest giới hạn trong int32
	})
	if err != nil {
		return nil, 0, fmt.Errorf("postgres.ProjectRepo list: %w", err)
	}
	out := make([]domain.Project, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.Project{
			ID: x.ID, Name: x.Name, Description: x.Description, Status: domain.ProjectStatus(x.Status),
			StartDate: datePtr(x.StartDate), EndDate: datePtr(x.EndDate),
			ManagerID: x.ManagerID, ManagerName: x.ManagerName, MemberCount: int(x.MemberCount),
			CreatedAt: x.CreatedAt.Time, UpdatedAt: x.UpdatedAt.Time,
		})
	}
	return out, int(total), nil
}

// Get trả một dự án.
func (r *ProjectRepo) Get(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	x, err := r.q.GetProject(ctx, id)
	if err != nil {
		return domain.Project{}, projectErr(err)
	}
	return domain.Project{
		ID: x.ID, Name: x.Name, Description: x.Description, Status: domain.ProjectStatus(x.Status),
		StartDate: datePtr(x.StartDate), EndDate: datePtr(x.EndDate),
		ManagerID: x.ManagerID, ManagerName: x.ManagerName, MemberCount: int(x.MemberCount),
		CreatedAt: x.CreatedAt.Time, UpdatedAt: x.UpdatedAt.Time,
	}, nil
}

// Lock khóa dòng dự án tới hết transaction.
func (r *ProjectRepo) Lock(ctx context.Context, id uuid.UUID) error {
	if _, err := r.q.LockProject(ctx, id); err != nil {
		return projectErr(err)
	}
	return nil
}

// Create thêm dự án.
func (r *ProjectRepo) Create(ctx context.Context, in project.NewProject) (uuid.UUID, error) {
	id, err := r.q.CreateProject(ctx, sqlcgen.CreateProjectParams{
		Name: in.Name, Description: in.Description, Status: string(in.Status),
		StartDate: date(in.StartDate), EndDate: date(in.EndDate),
		ManagerID: in.ManagerID, CreatedBy: in.CreatedBy,
	})
	if err != nil {
		return uuid.Nil, projectErr(err)
	}
	return id, nil
}

// Update sửa thông tin dự án.
func (r *ProjectRepo) Update(ctx context.Context, id uuid.UUID, f project.Fields) error {
	n, err := r.q.UpdateProject(ctx, sqlcgen.UpdateProjectParams{
		Name: f.Name, Description: f.Description, Status: string(f.Status),
		StartDate: date(f.StartDate), EndDate: date(f.EndDate), ID: id,
	})
	if err != nil {
		return projectErr(err)
	}
	return affected(n, domain.ErrProjectNotFound)
}

// IsMember cho biết userID có trong dự án không.
func (r *ProjectRepo) IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	ok, err := r.q.IsProjectMember(ctx, sqlcgen.IsProjectMemberParams{ProjectID: projectID, UserID: userID})
	if err != nil {
		return false, fmt.Errorf("postgres.ProjectRepo is member: %w", err)
	}
	return ok, nil
}

// ListMembers trả thành viên dự án.
func (r *ProjectRepo) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error) {
	rows, err := r.q.ListProjectMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("postgres.ProjectRepo list members: %w", err)
	}
	out := make([]domain.ProjectMember, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.ProjectMember{
			UserID: x.ID, FullName: x.FullName, Email: x.Email,
			Role: domain.Role(x.RoleCode), Status: domain.UserStatus(x.Status), AddedAt: x.AddedAt.Time,
		})
	}
	return out, nil
}

// AddMember thêm thành viên (đã có thì bỏ qua).
func (r *ProjectRepo) AddMember(ctx context.Context, projectID, userID uuid.UUID) error {
	if _, err := r.q.AddProjectMember(ctx, sqlcgen.AddProjectMemberParams{ProjectID: projectID, UserID: userID}); err != nil {
		if fkViolation(err) == "fk_project_member_user" {
			return domain.ErrInvalidProjectMember
		}
		if fkViolation(err) == "fk_project_member_project" {
			return domain.ErrProjectNotFound
		}
		return fmt.Errorf("postgres.ProjectRepo add member: %w", err)
	}
	return nil
}

// RemoveMember bỏ thành viên.
func (r *ProjectRepo) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	n, err := r.q.RemoveProjectMember(ctx, sqlcgen.RemoveProjectMemberParams{ProjectID: projectID, UserID: userID})
	if err != nil {
		return fmt.Errorf("postgres.ProjectRepo remove member: %w", err)
	}
	return affected(n, domain.ErrProjectMemberNotFound)
}

// ListCandidates trả người có thể thêm vào dự án.
func (r *ProjectRepo) ListCandidates(ctx context.Context, projectID uuid.UUID, search string) ([]domain.MemberCandidate, error) {
	s := pgtype.Text{}
	if search != "" {
		s = text(likeContains(search))
	}
	rows, err := r.q.ListMemberCandidates(ctx, sqlcgen.ListMemberCandidatesParams{ProjectID: projectID, Search: s})
	if err != nil {
		return nil, fmt.Errorf("postgres.ProjectRepo list candidates: %w", err)
	}
	out := make([]domain.MemberCandidate, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.MemberCandidate{
			UserID: x.ID, FullName: x.FullName, Email: x.Email,
			Role: domain.Role(x.RoleCode), Status: domain.UserStatus(x.Status),
		})
	}
	return out, nil
}

// GetCandidate trả vai trò và trạng thái của một tài khoản.
func (r *ProjectRepo) GetCandidate(ctx context.Context, userID uuid.UUID) (domain.MemberCandidate, error) {
	x, err := r.q.GetMemberCandidate(ctx, userID)
	if err != nil {
		return domain.MemberCandidate{}, notFound(err)
	}
	return domain.MemberCandidate{UserID: x.ID, Role: domain.Role(x.RoleCode), Status: domain.UserStatus(x.Status)}, nil
}

// ListColumns trả cột theo thứ tự vị trí.
func (r *ProjectRepo) ListColumns(ctx context.Context, projectID uuid.UUID) ([]domain.BoardColumn, error) {
	rows, err := r.q.ListBoardColumns(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("postgres.ProjectRepo list columns: %w", err)
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

// position chuyển vị trí sang int16 (domain giới hạn tối đa 8 cột).
func position(p int) int16 { return int16(p) } //nolint:gosec // 0 ≤ p < MaxBoardColumns

// CreateColumn thêm cột.
func (r *ProjectRepo) CreateColumn(ctx context.Context, projectID uuid.UUID, c domain.ColumnSpec, pos int) (uuid.UUID, error) {
	id, err := r.q.CreateBoardColumn(ctx, sqlcgen.CreateBoardColumnParams{
		ProjectID: projectID, Name: c.Name, Position: position(pos), StatusGroup: string(c.StatusGroup),
	})
	if err != nil {
		return uuid.Nil, columnErr(err)
	}
	return id, nil
}

// UpdateColumn đổi tên / nhóm của cột.
func (r *ProjectRepo) UpdateColumn(ctx context.Context, projectID, id uuid.UUID, c domain.ColumnSpec) error {
	n, err := r.q.UpdateBoardColumn(ctx, sqlcgen.UpdateBoardColumnParams{
		Name: c.Name, StatusGroup: string(c.StatusGroup), ID: id, ProjectID: projectID,
	})
	if err != nil {
		return columnErr(err)
	}
	return affected(n, domain.ErrColumnNotFound)
}

// SetColumnPosition đặt vị trí cột.
func (r *ProjectRepo) SetColumnPosition(ctx context.Context, projectID, id uuid.UUID, pos int) error {
	n, err := r.q.SetBoardColumnPosition(ctx, sqlcgen.SetBoardColumnPositionParams{
		Position: position(pos), ID: id, ProjectID: projectID,
	})
	if err != nil {
		return fmt.Errorf("postgres.ProjectRepo set position: %w", err)
	}
	return affected(n, domain.ErrColumnNotFound)
}

// DeleteColumn xóa cột (luật chuyển liên quan bị xóa theo ON DELETE CASCADE).
func (r *ProjectRepo) DeleteColumn(ctx context.Context, projectID, id uuid.UUID) error {
	n, err := r.q.DeleteBoardColumn(ctx, sqlcgen.DeleteBoardColumnParams{ID: id, ProjectID: projectID})
	if fkViolation(err) == "fk_task_column" { // việc được thêm vào cột giữa lúc kiểm và lúc xóa
		return domain.ErrColumnHasTasks
	}
	if err != nil {
		return fmt.Errorf("postgres.ProjectRepo delete column: %w", err)
	}
	return affected(n, domain.ErrColumnNotFound)
}

// ListTransitions trả luật chuyển của dự án.
func (r *ProjectRepo) ListTransitions(ctx context.Context, projectID uuid.UUID) ([]domain.Transition, error) {
	rows, err := r.q.ListWorkflowTransitions(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("postgres.ProjectRepo list transitions: %w", err)
	}
	out := make([]domain.Transition, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.Transition{FromColumnID: x.FromColumnID, ToColumnID: x.ToColumnID, Role: domain.TransitionRole(x.Role)})
	}
	return out, nil
}

// ReplaceTransitions xóa rồi ghi lại toàn bộ luật (gọi trong transaction).
func (r *ProjectRepo) ReplaceTransitions(ctx context.Context, projectID uuid.UUID, ts []domain.Transition) error {
	if err := r.q.DeleteWorkflowTransitions(ctx, projectID); err != nil {
		return fmt.Errorf("postgres.ProjectRepo delete transitions: %w", err)
	}
	for _, t := range ts {
		if err := r.q.CreateWorkflowTransition(ctx, sqlcgen.CreateWorkflowTransitionParams{
			ProjectID: projectID, FromColumnID: t.FromColumnID, ToColumnID: t.ToColumnID, Role: string(t.Role),
		}); err != nil {
			if fkViolation(err) != "" {
				return domain.ErrColumnNotFound
			}
			return fmt.Errorf("postgres.ProjectRepo create transition: %w", err)
		}
	}
	return nil
}
