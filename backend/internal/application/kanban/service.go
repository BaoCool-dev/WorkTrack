package kanban

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// MoveInput là yêu cầu kéo thả một thẻ: cột đích, vị trí trong cột (0 = đầu) và version client đang thấy.
type MoveInput struct {
	ColumnID uuid.UUID
	Position int
	Version  int
}

// Service xử lý kéo thả trên bảng Kanban (CF-04) và kiểm giới hạn WIP (CF-05).
type Service struct {
	store    Store
	tx       Transactor
	projects Projects
	pub      Publisher
}

// NewService tạo Service. projects dùng để kiểm quyền xem dự án (một nơi duy nhất ở project.Service).
func NewService(store Store, tx Transactor, projects Projects, pub Publisher) *Service {
	return &Service{store: store, tx: tx, projects: projects, pub: pub}
}

// Board trả các công việc cha trên bảng của dự án (công việc con không lên bảng — CLAUDE.md §5.4).
func (s *Service) Board(ctx context.Context, actor authz.Principal, projectID uuid.UUID) ([]domain.Task, error) {
	if _, err := s.projects.Get(ctx, actor, projectID); err != nil {
		return nil, err
	}
	tasks, err := s.store.ListBoard(ctx, projectID, BoardMaxTasks)
	if err != nil {
		return nil, fmt.Errorf("list board: %w", err)
	}
	return tasks, nil
}

// Move chuyển thẻ sang cột khác hoặc đổi thứ tự trong cột.
//
// Vì sao làm trong một transaction có khóa:
//   - Khóa dòng project trước rồi mới khóa dòng task (cùng thứ tự với Claim/Update) để hai người
//     cùng kéo việc vào in_progress không cùng thấy "còn 1 chỗ WIP" và không deadlock.
//   - So version trên bản đã khóa: nếu người khác vừa sửa thẻ thì trả xung đột thay vì ghi đè.
func (s *Service) Move(ctx context.Context, actor authz.Principal, id uuid.UUID, in MoveInput) (domain.Task, error) {
	task, project, err := s.view(ctx, actor, id)
	if err != nil {
		return domain.Task{}, err
	}
	// Kiểm sớm để người không được kéo nhận lỗi quyền, không phải lỗi version hay cột.
	if _, err := domain.BoardRole(actor.UserID, actor.Role, project.ManagerID, task.AssigneeID); err != nil {
		return domain.Task{}, err
	}

	var from uuid.UUID
	err = s.tx.InTx(ctx, func(st Store) error {
		var err error
		from, err = s.moveLocked(ctx, st, actor, project, id, in)
		return err
	})
	if err != nil {
		return domain.Task{}, err
	}

	moved, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("reload task: %w", err)
	}
	// Phát sau commit: client khác chỉ nhận thay đổi đã chắc chắn lưu.
	s.pub.Publish(ctx, domain.BoardEvent{
		Type: domain.EventTaskMoved, ProjectID: project.ID, TaskID: id, ActorID: actor.UserID,
		FromColumnID: from, ToColumnID: moved.ColumnID, Position: moved.Position, Version: moved.Version,
	})
	return moved, nil
}

// view nạp công việc cha và dự án nếu actor được xem. Không lộ sự tồn tại của việc ở dự án khác.
func (s *Service) view(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Task, domain.Project, error) {
	task, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Task{}, domain.Project{}, err
	}
	if task.ParentID != nil {
		return domain.Task{}, domain.Project{}, domain.ErrTaskNotFound // con không nằm trên bảng
	}
	project, err := s.projects.Get(ctx, actor, task.ProjectID)
	if errors.Is(err, domain.ErrProjectNotFound) {
		return domain.Task{}, domain.Project{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, domain.Project{}, err
	}
	return task, project, nil
}

// moveLocked chạy trong transaction; trả cột cũ để phát sự kiện.
func (s *Service) moveLocked(ctx context.Context, st Store, actor authz.Principal, p domain.Project,
	id uuid.UUID, in MoveInput) (uuid.UUID, error) {
	task, err := lockTask(ctx, st, p.ID, id)
	if err != nil {
		return uuid.Nil, err
	}
	if task.Version != in.Version {
		return uuid.Nil, domain.ErrTaskVersionConflict
	}
	if err := s.checkRules(ctx, st, actor, p, task, in.ColumnID); err != nil {
		return uuid.Nil, err
	}
	if err := st.MoveColumn(ctx, id, in.ColumnID, task.Version); err != nil {
		return uuid.Nil, err
	}
	if err := reorder(ctx, st, p.ID, in.ColumnID, id, in.Position); err != nil {
		return uuid.Nil, err
	}
	if task.ColumnID != in.ColumnID {
		err = st.InsertMoveHistory(ctx, MoveRecord{
			TaskID: id, ProjectID: p.ID, FromColumnID: task.ColumnID, ToColumnID: in.ColumnID, MovedBy: actor.UserID,
		})
		if err != nil {
			return uuid.Nil, fmt.Errorf("insert move history: %w", err)
		}
	}
	return task.ColumnID, nil
}

// lockTask khóa project rồi task (thứ tự cố định) và đọc lại bản mới nhất sau khi khóa.
func lockTask(ctx context.Context, st Store, projectID, id uuid.UUID) (domain.Task, error) {
	if err := st.LockProject(ctx, projectID); err != nil {
		return domain.Task{}, err
	}
	if err := st.Lock(ctx, id); err != nil {
		return domain.Task{}, err
	}
	return st.Get(ctx, id)
}

// checkRules: quyền kéo → luật chuyển cột → WIP, trên dữ liệu đã khóa.
func (s *Service) checkRules(ctx context.Context, st Store, actor authz.Principal, p domain.Project,
	task domain.Task, to uuid.UUID) error {
	// Tính lại vai trò: người nhận có thể vừa đổi giữa lúc kiểm sớm và lúc khóa.
	role, err := domain.BoardRole(actor.UserID, actor.Role, p.ManagerID, task.AssigneeID)
	if err != nil {
		return err
	}
	toGroup, err := st.ColumnGroup(ctx, p.ID, to) // cột phải thuộc đúng dự án này
	if err != nil {
		return err
	}
	same := task.ColumnID == to
	allowed := same
	if !same {
		if allowed, err = st.TransitionAllowed(ctx, p.ID, task.ColumnID, to, role); err != nil {
			return fmt.Errorf("transition allowed: %w", err)
		}
	}
	err = domain.CheckMove(domain.Move{Role: role, From: task.StatusGroup, To: toGroup, SameColumn: same, Allowed: allowed})
	if err != nil {
		return err
	}
	return checkWIP(ctx, st, p, task, toGroup)
}

// checkWIP: WIP là giới hạn của NGƯỜI NHẬN việc trong dự án (chỉ đếm việc cha — CLAUDE.md §5.6),
// nên áp dụng cả khi Manager kéo hộ. Việc chưa giao ai thì không thuộc WIP của ai → bỏ qua.
func checkWIP(ctx context.Context, st Store, p domain.Project, task domain.Task, to domain.StatusGroup) error {
	if task.AssigneeID == nil || !domain.EntersWIP(task.StatusGroup, to) {
		return nil
	}
	n, err := st.CountProjectWIP(ctx, p.ID, *task.AssigneeID)
	if err != nil {
		return fmt.Errorf("count wip: %w", err)
	}
	return domain.CheckWIP(n, p.WIPLimit)
}

// reorder đặt thẻ vào vị trí mới trong cột đích rồi đánh lại số thứ tự 0..n-1 cho cả cột.
// Ghi lại cả cột thay vì dùng số thực/khoảng trống: cột nhỏ (≤ vài chục thẻ), code đơn giản, dễ đọc.
func reorder(ctx context.Context, st Store, projectID, columnID, id uuid.UUID, pos int) error {
	ids, err := st.ColumnTaskIDs(ctx, projectID, columnID)
	if err != nil {
		return fmt.Errorf("column task ids: %w", err)
	}
	if err := st.SetPositions(ctx, domain.InsertAt(ids, id, pos)); err != nil {
		return fmt.Errorf("set positions: %w", err)
	}
	return nil
}
