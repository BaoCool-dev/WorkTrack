// Package setting: tham số cấu hình hệ thống Admin/CEO sửa được. Hiện có hệ số quy đổi Story Point → giờ (B1).
package setting

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Store là port đọc/ghi tham số số nguyên.
type Store interface {
	GetInt(ctx context.Context, key string) (int, time.Time, error)
	SetInt(ctx context.Context, key string, v int, by uuid.UUID) error
}

// Settings là bộ tham số trả cho client.
type Settings struct {
	HoursPerStoryPoint int
	UpdatedAt          time.Time
}

// Service xử lý use case cấu hình.
type Service struct{ store Store }

// NewService tạo Service.
func NewService(store Store) *Service { return &Service{store: store} }

// HoursPerStoryPoint trả hệ số B1 hiện hành (dùng nội bộ khi tính giờ ước tính).
func (s *Service) HoursPerStoryPoint(ctx context.Context) (int, error) {
	v, _, err := s.store.GetInt(ctx, domain.SettingStoryPointHours)
	if err != nil {
		return 0, fmt.Errorf("get story point hours: %w", err)
	}
	return v, nil
}

// Get trả cấu hình cho mọi vai trò hợp lệ (form công việc hiển thị giờ quy đổi).
func (s *Service) Get(ctx context.Context, actor authz.Principal) (Settings, error) {
	if !slices.Contains(domain.Roles, actor.Role) {
		return Settings{}, domain.ErrForbidden
	}
	v, at, err := s.store.GetInt(ctx, domain.SettingStoryPointHours)
	if err != nil {
		return Settings{}, fmt.Errorf("get story point hours: %w", err)
	}
	return Settings{HoursPerStoryPoint: v, UpdatedAt: at}, nil
}

// Update đổi hệ số B1. Công việc đã có giờ ước tính giữ nguyên giá trị cũ.
func (s *Service) Update(ctx context.Context, actor authz.Principal, hours int) (Settings, error) {
	if err := authz.Require(actor, authz.ManageSettings); err != nil {
		return Settings{}, err
	}
	if err := domain.ValidateStoryPointHours(hours); err != nil {
		return Settings{}, err
	}
	if err := s.store.SetInt(ctx, domain.SettingStoryPointHours, hours, actor.UserID); err != nil {
		return Settings{}, fmt.Errorf("set story point hours: %w", err)
	}
	return s.Get(ctx, actor)
}
