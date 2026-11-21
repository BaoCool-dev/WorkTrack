// Package task: công việc cha (CF-06) — Manager tạo, sửa, giao việc; Member tự nhận việc ở nhóm todo;
// danh sách theo dự án, "Công việc của tôi" và số WIP. Chuyển cột / kéo thả thuộc CF-04 (feature/kanban).
package task

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// SearchMaxRunes giới hạn độ dài ô tìm kiếm công việc.
const SearchMaxRunes = 100

// Filter lọc công việc cha của một dự án (đã chuẩn hóa). Search là chuỗi thô, repo tự thoát ký tự ILIKE.
type Filter struct {
	Search      string
	ColumnID    *uuid.UUID
	StatusGroup *domain.StatusGroup
	AssigneeID  *uuid.UUID
	Unassigned  bool
	LabelID     *uuid.UUID
}

// MyFilter lọc trang "Công việc của tôi".
type MyFilter struct {
	Search      string
	StatusGroup *domain.StatusGroup
}

// Fields là dữ liệu công việc đã chuẩn hóa để lưu (PUT thay toàn bộ).
type Fields struct {
	Title            string
	Description      string
	StoryPoint       *int
	EstimatedMinutes *int
	AssigneeID       *uuid.UUID
	DueDate          *time.Time
	LabelIDs         []uuid.UUID
}

// Assignable là vai trò + trạng thái của một thành viên dự án.
type Assignable struct {
	Role   domain.Role
	Status domain.UserStatus
}

// Store là port lưu công việc. Danh sách và chi tiết trả kèm nhãn.
type Store interface {
	List(ctx context.Context, projectID uuid.UUID, f Filter, p domain.PageRequest) ([]domain.Task, int, error)
	ListMine(ctx context.Context, userID uuid.UUID, f MyFilter, p domain.PageRequest) ([]domain.Task, int, error)
	// Get trả domain.ErrTaskNotFound.
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	// Lock khóa dòng công việc tới hết transaction; trả ErrTaskNotFound.
	Lock(ctx context.Context, id uuid.UUID) error
	// LockProject khóa dòng dự án; luôn gọi TRƯỚC Lock (cùng thứ tự với kéo thả Kanban) để không deadlock.
	// Trả người phụ trách và giới hạn WIP đọc trên dòng đã khóa.
	LockProject(ctx context.Context, projectID uuid.UUID) (domain.LockedProject, error)
	// FirstTodoColumn trả domain.ErrBoardNeedsTodoColumn khi dự án không có cột nhóm todo.
	FirstTodoColumn(ctx context.Context, projectID uuid.UUID) (uuid.UUID, error)
	// Create trả domain.ErrLabelNotFound khi có nhãn không tồn tại.
	Create(ctx context.Context, projectID, columnID, createdBy uuid.UUID, f Fields) (uuid.UUID, error)
	// Update thay thông tin + nhãn; trả ErrTaskNotFound hoặc ErrLabelNotFound.
	Update(ctx context.Context, id uuid.UUID, f Fields) error
	// SetAssignee trả ErrTaskNotFound.
	SetAssignee(ctx context.Context, id uuid.UUID, userID *uuid.UUID) error
	// GetAssignable trả ok=false khi userID không là thành viên dự án.
	GetAssignable(ctx context.Context, projectID, userID uuid.UUID) (a Assignable, ok bool, err error)
	CountWIP(ctx context.Context, userID uuid.UUID) (int, error)
	// CountProjectWIP đếm công việc cha nhóm in_progress của userID trong một dự án (CF-05).
	CountProjectWIP(ctx context.Context, projectID, userID uuid.UUID) (int, error)
}

// Transactor chạy fn trong một transaction; fn lỗi → rollback toàn bộ.
type Transactor interface {
	InTx(ctx context.Context, fn func(Store) error) error
}

// Projects là phần của project.Service mà task cần: kiểm quyền xem dự án (ErrProjectNotFound nếu không xem được).
type Projects interface {
	Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error)
}

// Settings cung cấp hệ số quy đổi Story Point → giờ (B1).
type Settings interface {
	HoursPerStoryPoint(ctx context.Context) (int, error)
}

// Publisher báo thay đổi lên bảng Kanban realtime (CF-04). Gọi sau commit; lỗi gửi không làm hỏng thao tác.
type Publisher interface {
	Publish(ctx context.Context, e domain.BoardEvent)
}
