package org

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Service cài đặt các use case cơ cấu tổ chức. Mọi thao tác (kể cả đọc) cần quyền ManageOrg.
type Service struct{ store Store }

// NewService tạo Service.
func NewService(store Store) *Service { return &Service{store: store} }

// ListDepartments trả toàn bộ cây phòng ban (danh sách phẳng, client tự dựng cây theo ParentID).
func (s *Service) ListDepartments(ctx context.Context, actor authz.Principal) ([]domain.Department, error) {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return nil, err
	}
	ds, err := s.store.ListDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	if ds == nil {
		ds = []domain.Department{}
	}
	return ds, nil
}

// CreateDepartment tạo phòng ban; parentID nil = phòng ban gốc.
func (s *Service) CreateDepartment(ctx context.Context, actor authz.Principal, name string, parentID *uuid.UUID) (domain.Department, error) {
	return s.saveDepartment(ctx, actor, nil, name, parentID)
}

// UpdateDepartment đổi tên và/hoặc chuyển phòng ban sang cha khác; chặn tạo vòng.
func (s *Service) UpdateDepartment(ctx context.Context, actor authz.Principal, id uuid.UUID, name string, parentID *uuid.UUID) (domain.Department, error) {
	return s.saveDepartment(ctx, actor, &id, name, parentID)
}

// saveDepartment: id nil = tạo mới. Kiểm tra cha dựa trên ảnh chụp cây hiện tại
// (hai thao tác chuyển song song có thể lọt — ghi nhận ở decisions.md mục C).
func (s *Service) saveDepartment(ctx context.Context, actor authz.Principal, id *uuid.UUID, rawName string, parentID *uuid.UUID) (domain.Department, error) {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return domain.Department{}, err
	}
	name, err := domain.NormalizeOrgName(rawName)
	if err != nil {
		return domain.Department{}, err
	}
	ds, err := s.store.ListDepartments(ctx)
	if err != nil {
		return domain.Department{}, fmt.Errorf("list departments: %w", err)
	}
	tree := make(map[uuid.UUID]*uuid.UUID, len(ds))
	for _, d := range ds {
		tree[d.ID] = d.ParentID
	}
	self := uuid.Nil
	if id != nil {
		if _, ok := tree[*id]; !ok {
			return domain.Department{}, domain.ErrDepartmentNotFound
		}
		self = *id
	}
	if err := domain.ValidateDepartmentParent(self, parentID, tree); err != nil {
		return domain.Department{}, err
	}
	if id == nil {
		newID, err := s.store.CreateDepartment(ctx, name, parentID)
		if err != nil {
			return domain.Department{}, fmt.Errorf("create department: %w", err)
		}
		return domain.Department{ID: newID, Name: name, ParentID: parentID}, nil
	}
	if err := s.store.UpdateDepartment(ctx, *id, name, parentID); err != nil {
		return domain.Department{}, fmt.Errorf("update department: %w", err)
	}
	out := domain.Department{ID: *id, Name: name, ParentID: parentID}
	for _, d := range ds {
		if d.ID == *id {
			out.MemberCount = d.MemberCount
		}
	}
	return out, nil
}

// DeleteDepartment xóa phòng ban không còn phòng ban con và không còn nhân sự.
func (s *Service) DeleteDepartment(ctx context.Context, actor authz.Principal, id uuid.UUID) error {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return err
	}
	if err := s.store.DeleteDepartment(ctx, id); err != nil {
		return fmt.Errorf("delete department: %w", err)
	}
	return nil
}

// ListJobTitles trả toàn bộ chức danh.
func (s *Service) ListJobTitles(ctx context.Context, actor authz.Principal) ([]domain.JobTitle, error) {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return nil, err
	}
	xs, err := s.store.ListJobTitles(ctx)
	if err != nil {
		return nil, fmt.Errorf("list job titles: %w", err)
	}
	if xs == nil {
		xs = []domain.JobTitle{}
	}
	return xs, nil
}

// CreateJobTitle tạo chức danh.
func (s *Service) CreateJobTitle(ctx context.Context, actor authz.Principal, rawName string) (domain.JobTitle, error) {
	name, err := s.prepare(actor, rawName)
	if err != nil {
		return domain.JobTitle{}, err
	}
	id, err := s.store.CreateJobTitle(ctx, name)
	if err != nil {
		return domain.JobTitle{}, fmt.Errorf("create job title: %w", err)
	}
	return domain.JobTitle{ID: id, Name: name}, nil
}

// UpdateJobTitle đổi tên chức danh.
func (s *Service) UpdateJobTitle(ctx context.Context, actor authz.Principal, id uuid.UUID, rawName string) (domain.JobTitle, error) {
	name, err := s.prepare(actor, rawName)
	if err != nil {
		return domain.JobTitle{}, err
	}
	if err := s.store.UpdateJobTitle(ctx, id, name); err != nil {
		return domain.JobTitle{}, fmt.Errorf("update job title: %w", err)
	}
	return domain.JobTitle{ID: id, Name: name}, nil
}

// DeleteJobTitle xóa chức danh chưa gán cho ai.
func (s *Service) DeleteJobTitle(ctx context.Context, actor authz.Principal, id uuid.UUID) error {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return err
	}
	if err := s.store.DeleteJobTitle(ctx, id); err != nil {
		return fmt.Errorf("delete job title: %w", err)
	}
	return nil
}

// ListSkills trả toàn bộ danh mục kỹ năng.
func (s *Service) ListSkills(ctx context.Context, actor authz.Principal) ([]domain.Skill, error) {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return nil, err
	}
	xs, err := s.store.ListSkills(ctx)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	if xs == nil {
		xs = []domain.Skill{}
	}
	return xs, nil
}

// CreateSkill thêm kỹ năng vào danh mục.
func (s *Service) CreateSkill(ctx context.Context, actor authz.Principal, rawName string) (domain.Skill, error) {
	name, err := s.prepare(actor, rawName)
	if err != nil {
		return domain.Skill{}, err
	}
	id, err := s.store.CreateSkill(ctx, name)
	if err != nil {
		return domain.Skill{}, fmt.Errorf("create skill: %w", err)
	}
	return domain.Skill{ID: id, Name: name}, nil
}

// UpdateSkill đổi tên kỹ năng.
func (s *Service) UpdateSkill(ctx context.Context, actor authz.Principal, id uuid.UUID, rawName string) (domain.Skill, error) {
	name, err := s.prepare(actor, rawName)
	if err != nil {
		return domain.Skill{}, err
	}
	if err := s.store.UpdateSkill(ctx, id, name); err != nil {
		return domain.Skill{}, fmt.Errorf("update skill: %w", err)
	}
	return domain.Skill{ID: id, Name: name}, nil
}

// DeleteSkill xóa kỹ năng không còn trong hồ sơ nào.
func (s *Service) DeleteSkill(ctx context.Context, actor authz.Principal, id uuid.UUID) error {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return err
	}
	if err := s.store.DeleteSkill(ctx, id); err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}

func (s *Service) prepare(actor authz.Principal, rawName string) (string, error) {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return "", err
	}
	return domain.NormalizeOrgName(rawName)
}
