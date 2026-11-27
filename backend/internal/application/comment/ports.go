// Package comment: bình luận gắn công việc (CF-07). Ai xem được dự án thì đọc được;
// chỉ Manager/Member viết. Bình luận mới báo cho người nhận và người tạo công việc.
package comment

import (
	"context"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Store là port lưu bình luận.
type Store interface {
	// GetTask trả domain.ErrTaskNotFound.
	GetTask(ctx context.Context, id uuid.UUID) (domain.Task, error)
	List(ctx context.Context, taskID uuid.UUID, p domain.PageRequest) ([]domain.Comment, int, error)
	// Create lưu bình luận và trả bản đầy đủ (kèm tên tác giả); ErrTaskNotFound nếu việc vừa bị xóa.
	Create(ctx context.Context, taskID, authorID uuid.UUID, body string) (domain.Comment, error)
	// AddNotification ghi thông báo cho n.UserIDs; danh sách rỗng thì không làm gì.
	AddNotification(ctx context.Context, n domain.NewNotification) error
}

// Transactor chạy fn trong một transaction: bình luận và thông báo cùng thành công hoặc cùng hủy.
type Transactor interface {
	InTx(ctx context.Context, fn func(Store) error) error
}

// Projects kiểm quyền xem dự án (ErrProjectNotFound nếu không xem được).
type Projects interface {
	Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error)
}
