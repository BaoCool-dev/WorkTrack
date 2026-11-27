package notification

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Service xử lý use case chuông thông báo. Mọi vai trò đăng nhập đều có chuông của riêng mình.
type Service struct{ store Store }

// NewService tạo Service.
func NewService(store Store) *Service { return &Service{store: store} }

// List trả thông báo mới nhất trước.
func (s *Service) List(ctx context.Context, actor authz.Principal, unreadOnly bool, p domain.PageRequest) (domain.Page[domain.Notification], error) {
	items, total, err := s.store.List(ctx, actor.UserID, unreadOnly, p)
	if err != nil {
		return domain.Page[domain.Notification]{}, fmt.Errorf("list notifications: %w", err)
	}
	if items == nil {
		items = []domain.Notification{}
	}
	return domain.Page[domain.Notification]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: total}, nil
}

// UnreadCount trả số thông báo chưa đọc (số trên chuông; FE hỏi định kỳ).
func (s *Service) UnreadCount(ctx context.Context, actor authz.Principal) (int, error) {
	n, err := s.store.CountUnread(ctx, actor.UserID)
	if err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}
	return n, nil
}

// MarkRead đánh dấu một thông báo đã đọc. Thông báo của người khác trả 404 như không tồn tại.
// Đánh dấu lại thông báo đã đọc vẫn thành công (idempotent).
func (s *Service) MarkRead(ctx context.Context, actor authz.Principal, id uuid.UUID) error {
	ok, err := s.store.MarkRead(ctx, id, actor.UserID)
	if err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	if !ok {
		return domain.ErrNotificationNotFound
	}
	return nil
}

// MarkAllRead đánh dấu mọi thông báo chưa đọc; trả số dòng đã đổi.
func (s *Service) MarkAllRead(ctx context.Context, actor authz.Principal) (int, error) {
	n, err := s.store.MarkAllRead(ctx, actor.UserID)
	if err != nil {
		return 0, fmt.Errorf("mark all read: %w", err)
	}
	return n, nil
}
