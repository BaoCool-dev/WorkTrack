// Package notification: chuông thông báo của chính người dùng (CF-07).
// Mọi thao tác lọc theo actor.UserID — không ai đọc/đánh dấu thông báo của người khác.
package notification

import (
	"context"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Store là port đọc/ghi thông báo.
type Store interface {
	List(ctx context.Context, userID uuid.UUID, unreadOnly bool, p domain.PageRequest) ([]domain.Notification, int, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	// MarkRead trả false nếu thông báo không tồn tại hoặc không thuộc userID.
	MarkRead(ctx context.Context, id, userID uuid.UUID) (bool, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error)
}
