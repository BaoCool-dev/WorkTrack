// Package org: Admin/CEO quản lý cơ cấu tổ chức — phòng ban, chức danh, danh mục kỹ năng (CF-02).
package org

import (
	"context"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Store là port lưu cơ cấu tổ chức. Tên đã được chuẩn hóa trước khi gọi.
type Store interface {
	// ListDepartments trả toàn bộ cây (không phân trang — danh mục nhỏ).
	ListDepartments(ctx context.Context) ([]domain.Department, error)
	// CreateDepartment trả ErrDepartmentNameTaken (trùng tên cùng cha) hoặc ErrDepartmentNotFound (cha không còn).
	CreateDepartment(ctx context.Context, name string, parentID *uuid.UUID) (uuid.UUID, error)
	// UpdateDepartment trả ErrDepartmentNotFound hoặc ErrDepartmentNameTaken.
	UpdateDepartment(ctx context.Context, id uuid.UUID, name string, parentID *uuid.UUID) error
	// DeleteDepartment trả ErrDepartmentNotFound, ErrDepartmentHasChildren hoặc ErrDepartmentInUse.
	DeleteDepartment(ctx context.Context, id uuid.UUID) error

	ListJobTitles(ctx context.Context) ([]domain.JobTitle, error)
	// CreateJobTitle/UpdateJobTitle trả ErrJobTitleNameTaken; Update/Delete trả ErrJobTitleNotFound.
	CreateJobTitle(ctx context.Context, name string) (uuid.UUID, error)
	UpdateJobTitle(ctx context.Context, id uuid.UUID, name string) error
	// DeleteJobTitle trả ErrJobTitleInUse khi còn nhân sự mang chức danh.
	DeleteJobTitle(ctx context.Context, id uuid.UUID) error

	ListSkills(ctx context.Context) ([]domain.Skill, error)
	// CreateSkill/UpdateSkill trả ErrSkillNameTaken; Update/Delete trả ErrSkillNotFound.
	CreateSkill(ctx context.Context, name string) (uuid.UUID, error)
	UpdateSkill(ctx context.Context, id uuid.UUID, name string) error
	// DeleteSkill trả ErrSkillInUse khi kỹ năng còn trong hồ sơ nhân sự.
	DeleteSkill(ctx context.Context, id uuid.UUID) error
}
