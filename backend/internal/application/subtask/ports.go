// Package subtask: công việc con 1 cấp (CF-07). Member là người nhận của việc cha tự tạo việc con,
// tự chuyển cột việc con của mình — KHÔNG đi qua luật workflow_transition và không tính WIP (CLAUDE.md §5.4, §5.6).
// Việc con không có Story Point (CHECK ở CSDL) và không nghiệm thu riêng.
package subtask

import (
	"context"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Store là port lưu công việc con.
type Store interface {
	// Get trả domain.ErrTaskNotFound (dùng cho cả cha và con).
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	// List trả các con của một việc cha, theo thứ tự tạo.
	List(ctx context.Context, parentID uuid.UUID) ([]domain.Task, error)
	// LockProject khóa dòng dự án — luôn gọi TRƯỚC Lock (cùng thứ tự với Kanban) để không deadlock.
	LockProject(ctx context.Context, projectID uuid.UUID) (domain.LockedProject, error)
	// Lock khóa dòng công việc; trả ErrTaskNotFound.
	Lock(ctx context.Context, id uuid.UUID) error
	// ColumnGroup trả nhóm của cột; cột không thuộc dự án → ErrColumnNotFound.
	ColumnGroup(ctx context.Context, projectID, columnID uuid.UUID) (domain.StatusGroup, error)
	// FirstTodoColumn trả ErrBoardNeedsTodoColumn khi dự án không có cột nhóm todo.
	FirstTodoColumn(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error)
	// Create tạo con của parent (cùng dự án, kế thừa người nhận của cha); trả ErrSubtaskParentNotRoot.
	Create(ctx context.Context, parent domain.Task, columnID, createdBy uuid.UUID, title, description string) (uuid.UUID, error)
	// Move đổi cột của con; trả ErrTaskNotFound.
	Move(ctx context.Context, id, columnID uuid.UUID) error
	// Delete xóa con; trả ErrTaskNotFound.
	Delete(ctx context.Context, id uuid.UUID) error
	// RecordMove ghi một dòng task_move_history (cùng giao dịch với Move).
	RecordMove(ctx context.Context, r MoveRecord) error
}

// Transactor chạy fn trong một transaction; fn lỗi → rollback toàn bộ.
type Transactor interface {
	InTx(ctx context.Context, fn func(Store) error) error
}

// Projects kiểm quyền xem dự án (ErrProjectNotFound nếu không xem được).
type Projects interface {
	Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error)
}

// MoveRecord là một dòng task_move_history khi con đổi cột — để báo cáo luồng việc đọc được cả việc con.
type MoveRecord struct{ TaskID, ProjectID, FromColumnID, ToColumnID, MovedBy uuid.UUID }
