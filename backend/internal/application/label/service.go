// Package label: danh mục nhãn công việc dùng chung toàn công ty (CF-06). Admin/CEO quản lý; mọi người dùng đọc được.
package label

import (
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Store là port lưu nhãn. Tên đã chuẩn hóa trước khi gọi.
type Store interface {
	// List trả toàn bộ nhãn kèm số công việc đang gắn, sắp theo tên.
	List(ctx context.Context) ([]domain.Label, error)
	// Create trả ErrLabelNameTaken khi trùng tên (không phân biệt hoa thường).
	Create(ctx context.Context, name string) (uuid.UUID, error)
	// Update trả ErrLabelNotFound hoặc ErrLabelNameTaken.
	Update(ctx context.Context, id uuid.UUID, name string) error
	// Delete trả ErrLabelNotFound hoặc ErrLabelInUse khi nhãn còn gắn vào công việc.
	Delete(ctx context.Context, id uuid.UUID) error
}

// Service xử lý use case danh mục nhãn.
type Service struct{ store Store }

// NewService tạo Service.
func NewService(store Store) *Service { return &Service{store: store} }

// List trả danh mục nhãn cho mọi vai trò hợp lệ (cần để gắn/lọc công việc).
func (s *Service) List(ctx context.Context, actor authz.Principal) ([]domain.Label, error) {
	if !slices.Contains(domain.Roles, actor.Role) {
		return nil, domain.ErrForbidden
	}
	xs, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list labels: %w", err)
	}
	if xs == nil {
		xs = []domain.Label{}
	}
	return xs, nil
}

func prepare(actor authz.Principal, raw string) (string, error) {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return "", err
	}
	return domain.NormalizeOrgName(raw)
}

// Create thêm nhãn.
func (s *Service) Create(ctx context.Context, actor authz.Principal, raw string) (domain.Label, error) {
	name, err := prepare(actor, raw)
	if err != nil {
		return domain.Label{}, err
	}
	id, err := s.store.Create(ctx, name)
	if err != nil {
		return domain.Label{}, fmt.Errorf("create label: %w", err)
	}
	return domain.Label{ID: id, Name: name}, nil
}

// Update đổi tên nhãn.
func (s *Service) Update(ctx context.Context, actor authz.Principal, id uuid.UUID, raw string) (domain.Label, error) {
	name, err := prepare(actor, raw)
	if err != nil {
		return domain.Label{}, err
	}
	if err := s.store.Update(ctx, id, name); err != nil {
		return domain.Label{}, fmt.Errorf("update label: %w", err)
	}
	return domain.Label{ID: id, Name: name}, nil
}

// Delete xóa nhãn không còn gắn vào công việc nào.
func (s *Service) Delete(ctx context.Context, actor authz.Principal, id uuid.UUID) error {
	if err := authz.Require(actor, authz.ManageOrg); err != nil {
		return err
	}
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete label: %w", err)
	}
	return nil
}
