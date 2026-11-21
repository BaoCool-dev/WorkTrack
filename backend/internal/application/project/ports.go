// Package project: Manager tạo và cấu hình dự án — thông tin, trạng thái, thành viên,
// cột Kanban và luật chuyển trạng thái (CF-03). Việc ÁP DỤNG luật khi kéo thẻ thuộc CF-04.
package project

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// SearchMaxRunes giới hạn độ dài ô tìm kiếm dự án / ứng viên thành viên.
const SearchMaxRunes = 100

// Filter lọc danh sách dự án. VisibleTo nil = xem tất cả (CEO); có giá trị = chỉ dự án người đó tham gia.
type Filter struct {
	VisibleTo *uuid.UUID
	Search    string
	Status    *domain.ProjectStatus
}

// Fields là phần thông tin dự án Manager nhập (đã chuẩn hóa trước khi lưu).
type Fields struct {
	Name        string
	Description string
	Status      domain.ProjectStatus
	StartDate   *time.Time
	EndDate     *time.Time
}

// NewProject là dữ liệu tạo dự án.
type NewProject struct {
	Fields
	ManagerID uuid.UUID
	CreatedBy uuid.UUID
}

// Store là port lưu dự án. Các thao tác ghi nhiều bước được gọi bên trong Transactor.InTx.
type Store interface {
	List(ctx context.Context, f Filter, p domain.PageRequest) ([]domain.Project, int, error)
	// Get trả domain.ErrProjectNotFound.
	Get(ctx context.Context, id uuid.UUID) (domain.Project, error)
	// Lock khóa dòng dự án tới hết transaction (FOR UPDATE); trả ErrProjectNotFound.
	Lock(ctx context.Context, id uuid.UUID) error
	// Create trả domain.ErrProjectNameTaken.
	Create(ctx context.Context, in NewProject) (uuid.UUID, error)
	// Update trả ErrProjectNotFound hoặc ErrProjectNameTaken.
	Update(ctx context.Context, id uuid.UUID, f Fields) error
	// UpdateWIPLimit trả ErrProjectNotFound (CF-05).
	UpdateWIPLimit(ctx context.Context, id uuid.UUID, limit int) error

	IsMember(ctx context.Context, projectID, userID uuid.UUID) (bool, error)
	ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.ProjectMember, error)
	// AddMember idempotent (đã là thành viên thì bỏ qua).
	AddMember(ctx context.Context, projectID, userID uuid.UUID) error
	// RemoveMember trả domain.ErrProjectMemberNotFound.
	RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error
	// ListCandidates: Manager/Member chưa khóa, chưa là thành viên, tối đa 20 người.
	ListCandidates(ctx context.Context, projectID uuid.UUID, search string) ([]domain.MemberCandidate, error)
	// GetCandidate trả domain.ErrUserNotFound.
	GetCandidate(ctx context.Context, userID uuid.UUID) (domain.MemberCandidate, error)

	// ListColumns trả cột theo thứ tự vị trí.
	ListColumns(ctx context.Context, projectID uuid.UUID) ([]domain.BoardColumn, error)
	// CreateColumn trả domain.ErrColumnNameTaken.
	CreateColumn(ctx context.Context, projectID uuid.UUID, c domain.ColumnSpec, position int) (uuid.UUID, error)
	// UpdateColumn trả ErrColumnNotFound hoặc ErrColumnNameTaken.
	UpdateColumn(ctx context.Context, projectID, id uuid.UUID, c domain.ColumnSpec) error
	// SetColumnPosition trả ErrColumnNotFound. Ràng buộc vị trí kiểm ở cuối transaction.
	SetColumnPosition(ctx context.Context, projectID, id uuid.UUID, position int) error
	// DeleteColumn trả ErrColumnNotFound.
	DeleteColumn(ctx context.Context, projectID, id uuid.UUID) error

	ListTransitions(ctx context.Context, projectID uuid.UUID) ([]domain.Transition, error)
	// ReplaceTransitions thay toàn bộ luật của dự án (chỉ gọi trong InTx).
	ReplaceTransitions(ctx context.Context, projectID uuid.UUID, ts []domain.Transition) error
}

// Transactor chạy fn trong một transaction; fn lỗi → rollback toàn bộ.
type Transactor interface {
	InTx(ctx context.Context, fn func(Store) error) error
}
