// Package kanban: bảng Kanban của dự án — tải bảng, kéo thả công việc cha giữa các cột và đổi thứ tự
// trong cột (CF-04), chặn vượt giới hạn WIP khi kéo vào nhóm in_progress (CF-05).
// Công việc con kéo thả trong chi tiết công việc cha thuộc CF-07.
package kanban

import (
	"context"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// BoardMaxTasks là giới hạn an toàn số công việc cha tải lên một bảng.
const BoardMaxTasks = 500

// MoveRecord là một dòng lịch sử chuyển cột.
type MoveRecord struct {
	TaskID, ProjectID, FromColumnID, ToColumnID, MovedBy uuid.UUID
}

// Store là port lưu trữ bảng Kanban.
type Store interface {
	// Get trả domain.ErrTaskNotFound.
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	// ListBoard trả công việc cha của dự án theo thứ tự cột rồi vị trí trong cột (kèm nhãn).
	ListBoard(ctx context.Context, projectID uuid.UUID, limit int) ([]domain.Task, error)
	// LockProject khóa dòng dự án tới hết transaction: mọi lần kéo trong dự án xếp hàng,
	// nên phép đếm WIP và phép đánh lại vị trí không bị hai request song song làm sai.
	// Trả người phụ trách và giới hạn WIP đọc trên dòng đã khóa; dự án không còn → ErrProjectNotFound.
	LockProject(ctx context.Context, projectID uuid.UUID) (domain.LockedProject, error)
	// Lock khóa dòng công việc; trả ErrTaskNotFound.
	Lock(ctx context.Context, id uuid.UUID) error
	// ColumnGroup trả nhóm trạng thái của cột; cột không thuộc dự án → ErrColumnNotFound.
	ColumnGroup(ctx context.Context, projectID, columnID uuid.UUID) (domain.StatusGroup, error)
	// TransitionAllowed cho biết bảng workflow_transition có luật from → to cho role không.
	TransitionAllowed(ctx context.Context, projectID, from, to uuid.UUID, role domain.TransitionRole) (bool, error)
	// CountProjectWIP đếm công việc cha nhóm in_progress của userID trong dự án.
	CountProjectWIP(ctx context.Context, projectID, userID uuid.UUID) (int, error)
	// MoveColumn đặt cột mới và tăng version khi version khớp; lệch → ErrTaskVersionConflict.
	MoveColumn(ctx context.Context, id, columnID uuid.UUID, version int) error
	// ColumnTaskIDs trả id công việc cha trong cột theo thứ tự hiện tại.
	ColumnTaskIDs(ctx context.Context, projectID, columnID uuid.UUID) ([]uuid.UUID, error)
	// SetPositions đặt position = chỉ số trong ids (0, 1, 2…).
	SetPositions(ctx context.Context, ids []uuid.UUID) error
	InsertMoveHistory(ctx context.Context, r MoveRecord) error
}

// Transactor chạy fn trong một transaction; fn lỗi → rollback toàn bộ.
type Transactor interface {
	InTx(ctx context.Context, fn func(Store) error) error
}

// Projects là phần của project.Service mà kanban cần: quyền xem dự án (ErrProjectNotFound nếu không xem được).
type Projects interface {
	Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error)
}

// Publisher phát sự kiện tới các client đang mở bảng. Gọi SAU commit; không trả lỗi
// vì realtime chỉ là phần thêm — mất một sự kiện thì client vẫn đúng khi tải lại.
type Publisher interface {
	Publish(ctx context.Context, e domain.BoardEvent)
}
