// Package timelog: ghi giờ thực tế trên từng công việc (CF-08 — Task-based Timesheet).
// Người nhận việc (việc con: người nhận của việc cha) hoặc Manager phụ trách ghi giờ;
// ai xem được dự án thì xem được lịch sử. Không ghi/xóa giờ khi việc đã Done quá 24 giờ (CLAUDE.md §5.7).
package timelog

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Store là port lưu Log time.
type Store interface {
	// Get trả công việc (cha hoặc con); domain.ErrTaskNotFound.
	Get(ctx context.Context, id uuid.UUID) (domain.Task, error)
	// LockProject khóa dòng dự án — luôn gọi TRƯỚC Lock (cùng thứ tự với Kanban) để không deadlock.
	LockProject(ctx context.Context, projectID uuid.UUID) (domain.LockedProject, error)
	// Lock khóa dòng công việc; trả ErrTaskNotFound.
	Lock(ctx context.Context, id uuid.UUID) error
	// LastDoneAt trả lần gần nhất công việc được chuyển vào cột nhóm done; nil nếu chưa có lịch sử.
	LastDoneAt(ctx context.Context, taskID uuid.UUID) (*time.Time, error)
	// Create lưu một dòng log, trả id.
	Create(ctx context.Context, l NewLog) (uuid.UUID, error)
	// GetLog trả domain.ErrTimeLogNotFound.
	GetLog(ctx context.Context, id uuid.UUID) (domain.TimeLog, error)
	// ListLogs: log của công việc, với việc cha gồm cả log của các con; mới nhất trước.
	ListLogs(ctx context.Context, taskID uuid.UUID, p domain.PageRequest) ([]domain.TimeLog, int, error)
	// DeleteLog trả domain.ErrTimeLogNotFound.
	DeleteLog(ctx context.Context, id uuid.UUID) error
}

// Transactor chạy fn trong một transaction; fn lỗi → rollback toàn bộ.
type Transactor interface {
	InTx(ctx context.Context, fn func(Store) error) error
}

// Projects kiểm quyền xem dự án (ErrProjectNotFound nếu không xem được).
type Projects interface {
	Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error)
}

// Publisher phát sự kiện bảng Kanban (thẻ việc cha cập nhật thanh giờ ở mọi người đang mở dự án).
type Publisher interface {
	Publish(ctx context.Context, e domain.BoardEvent)
}

// NewLog là dữ liệu một dòng log đã kiểm tra xong.
type NewLog struct {
	TaskID, ProjectID, UserID uuid.UUID
	Minutes                   int
	Note                      string
	WorkDate                  time.Time
}

// Input là dữ liệu người dùng gửi lên. WorkDate nil → hôm nay theo giờ Việt Nam.
type Input struct {
	Minutes  int
	Note     string
	WorkDate *time.Time
}
