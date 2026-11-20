package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/org"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
)

// OrgRepo cài đặt org.Store (phòng ban, chức danh, danh mục kỹ năng — CF-02).
type OrgRepo struct{ q sqlcgen.Querier }

var _ org.Store = (*OrgRepo)(nil)

// NewOrgRepo tạo OrgRepo từ một DBTX (pool hoặc transaction).
func NewOrgRepo(db sqlcgen.DBTX) *OrgRepo { return &OrgRepo{q: sqlcgen.New(db)} }

func (r *OrgRepo) ListDepartments(ctx context.Context) ([]domain.Department, error) {
	rows, err := r.q.ListDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres.OrgRepo.ListDepartments: %w", err)
	}
	out := make([]domain.Department, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.Department{ID: x.ID, Name: x.Name, ParentID: uuidPtr(x.ParentID), MemberCount: int(x.MemberCount)})
	}
	return out, nil
}

func (r *OrgRepo) CreateDepartment(ctx context.Context, name string, parentID *uuid.UUID) (uuid.UUID, error) {
	id, err := r.q.CreateDepartment(ctx, sqlcgen.CreateDepartmentParams{Name: name, ParentID: nullUUID(parentID)})
	if err := departmentWriteError(err); err != nil {
		return uuid.Nil, fmt.Errorf("postgres.OrgRepo.CreateDepartment: %w", err)
	}
	return id, nil
}

func (r *OrgRepo) UpdateDepartment(ctx context.Context, id uuid.UUID, name string, parentID *uuid.UUID) error {
	n, err := r.q.UpdateDepartment(ctx, sqlcgen.UpdateDepartmentParams{Name: name, ParentID: nullUUID(parentID), ID: id})
	if err := departmentWriteError(err); err != nil {
		return fmt.Errorf("postgres.OrgRepo.UpdateDepartment: %w", err)
	}
	return affected(n, domain.ErrDepartmentNotFound)
}

func departmentWriteError(err error) error {
	switch {
	case err == nil:
		return nil
	case isUniqueViolation(err):
		return domain.ErrDepartmentNameTaken
	case fkViolation(err) == "fk_department_parent":
		return domain.ErrDepartmentNotFound
	}
	return err
}

func (r *OrgRepo) DeleteDepartment(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteDepartment(ctx, id)
	switch fkViolation(err) {
	case "fk_department_parent":
		return domain.ErrDepartmentHasChildren
	case "fk_app_user_department":
		return domain.ErrDepartmentInUse
	}
	if err != nil {
		return fmt.Errorf("postgres.OrgRepo.DeleteDepartment: %w", err)
	}
	return affected(n, domain.ErrDepartmentNotFound)
}

func (r *OrgRepo) ListJobTitles(ctx context.Context) ([]domain.JobTitle, error) {
	rows, err := r.q.ListJobTitles(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres.OrgRepo.ListJobTitles: %w", err)
	}
	out := make([]domain.JobTitle, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.JobTitle{ID: x.ID, Name: x.Name, MemberCount: int(x.MemberCount)})
	}
	return out, nil
}

func (r *OrgRepo) CreateJobTitle(ctx context.Context, name string) (uuid.UUID, error) {
	id, err := r.q.CreateJobTitle(ctx, name)
	if isUniqueViolation(err) {
		return uuid.Nil, domain.ErrJobTitleNameTaken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres.OrgRepo.CreateJobTitle: %w", err)
	}
	return id, nil
}

func (r *OrgRepo) UpdateJobTitle(ctx context.Context, id uuid.UUID, name string) error {
	n, err := r.q.UpdateJobTitle(ctx, sqlcgen.UpdateJobTitleParams{Name: name, ID: id})
	if isUniqueViolation(err) {
		return domain.ErrJobTitleNameTaken
	}
	if err != nil {
		return fmt.Errorf("postgres.OrgRepo.UpdateJobTitle: %w", err)
	}
	return affected(n, domain.ErrJobTitleNotFound)
}

func (r *OrgRepo) DeleteJobTitle(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteJobTitle(ctx, id)
	if fkViolation(err) == "fk_app_user_job_title" {
		return domain.ErrJobTitleInUse
	}
	if err != nil {
		return fmt.Errorf("postgres.OrgRepo.DeleteJobTitle: %w", err)
	}
	return affected(n, domain.ErrJobTitleNotFound)
}

func (r *OrgRepo) ListSkills(ctx context.Context) ([]domain.Skill, error) {
	rows, err := r.q.ListSkills(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres.OrgRepo.ListSkills: %w", err)
	}
	out := make([]domain.Skill, 0, len(rows))
	for _, x := range rows {
		out = append(out, domain.Skill{ID: x.ID, Name: x.Name, MemberCount: int(x.MemberCount)})
	}
	return out, nil
}

func (r *OrgRepo) CreateSkill(ctx context.Context, name string) (uuid.UUID, error) {
	id, err := r.q.CreateSkill(ctx, name)
	if isUniqueViolation(err) {
		return uuid.Nil, domain.ErrSkillNameTaken
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("postgres.OrgRepo.CreateSkill: %w", err)
	}
	return id, nil
}

func (r *OrgRepo) UpdateSkill(ctx context.Context, id uuid.UUID, name string) error {
	n, err := r.q.UpdateSkill(ctx, sqlcgen.UpdateSkillParams{Name: name, ID: id})
	if isUniqueViolation(err) {
		return domain.ErrSkillNameTaken
	}
	if err != nil {
		return fmt.Errorf("postgres.OrgRepo.UpdateSkill: %w", err)
	}
	return affected(n, domain.ErrSkillNotFound)
}

func (r *OrgRepo) DeleteSkill(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteSkill(ctx, id)
	if fkViolation(err) == "fk_user_skill_skill" {
		return domain.ErrSkillInUse
	}
	if err != nil {
		return fmt.Errorf("postgres.OrgRepo.DeleteSkill: %w", err)
	}
	return affected(n, domain.ErrSkillNotFound)
}

func affected(n int64, notFoundErr error) error {
	if n == 0 {
		return notFoundErr
	}
	return nil
}
