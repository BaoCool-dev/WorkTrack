package subtask

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Input là dữ liệu tạo công việc con. Không có Story Point, người nhận lấy theo việc cha.
type Input struct {
	Title       string
	Description string
}

// Service xử lý use case công việc con (CF-07).
type Service struct {
	store    Store
	tx       Transactor
	projects Projects
}

// NewService tạo Service.
func NewService(store Store, tx Transactor, projects Projects) *Service {
	return &Service{store: store, tx: tx, projects: projects}
}

// view đọc công việc và kiểm người gọi xem được dự án. Không xem được → ErrTaskNotFound (404, chống dò ID).
func (s *Service) view(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Task, domain.Project, error) {
	t, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Task{}, domain.Project{}, fmt.Errorf("get task: %w", err)
	}
	p, err := s.projects.Get(ctx, actor, t.ProjectID)
	if errors.Is(err, domain.ErrProjectNotFound) {
		return domain.Task{}, domain.Project{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, domain.Project{}, err
	}
	return t, p, nil
}

// viewChild như view nhưng id phải là công việc con: API con không được dùng để sửa việc cha.
func (s *Service) viewChild(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Task, domain.Project, error) {
	t, p, err := s.view(ctx, actor, id)
	if err != nil {
		return domain.Task{}, domain.Project{}, err
	}
	if t.ParentID == nil {
		return domain.Task{}, domain.Project{}, domain.ErrTaskNotFound
	}
	return t, p, nil
}

// List trả các con của một việc cha cho ai xem được dự án (kể cả CEO).
func (s *Service) List(ctx context.Context, actor authz.Principal, parentID uuid.UUID) ([]domain.Task, error) {
	if _, _, err := s.view(ctx, actor, parentID); err != nil {
		return nil, err
	}
	items, err := s.store.List(ctx, parentID)
	if err != nil {
		return nil, fmt.Errorf("list subtasks: %w", err)
	}
	if items == nil {
		items = []domain.Task{}
	}
	return items, nil
}

// Create: người nhận của việc cha (hoặc Manager phụ trách) chia việc cha thành việc con.
// Con vào cột todo đầu tiên, kế thừa người nhận của cha.
func (s *Service) Create(ctx context.Context, actor authz.Principal, parentID uuid.UUID, in Input) (domain.Task, error) {
	parent, p, err := s.view(ctx, actor, parentID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := domain.CheckSubtaskParent(parent); err != nil {
		return domain.Task{}, err
	}
	if err := domain.CheckSubtaskActor(actor.UserID, actor.Role, p.ManagerID, parent.AssigneeID); err != nil {
		return domain.Task{}, err
	}
	title, err := domain.NormalizeTaskTitle(in.Title)
	if err != nil {
		return domain.Task{}, err
	}
	desc, err := domain.NormalizeTaskDescription(in.Description)
	if err != nil {
		return domain.Task{}, err
	}
	var id uuid.UUID
	err = s.tx.InTx(ctx, func(st Store) error {
		// Khóa cha để đọc người nhận mới nhất: Manager vừa giao lại việc thì con theo người nhận mới.
		locked, err := lockTask(ctx, st, parent.ProjectID, parentID)
		if err != nil {
			return err
		}
		if err := domain.CheckSubtaskActor(actor.UserID, actor.Role, p.ManagerID, locked.AssigneeID); err != nil {
			return err
		}
		col, err := st.FirstTodoColumn(ctx, locked.ProjectID)
		if err != nil {
			return fmt.Errorf("first todo column: %w", err)
		}
		id, err = st.Create(ctx, locked, col, actor.UserID, title, desc)
		if err != nil {
			return fmt.Errorf("create subtask: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Task{}, err
	}
	return s.reload(ctx, id)
}

// Move chuyển con sang cột bất kỳ của dự án — KHÔNG qua luật workflow_transition (R: con tự do).
// Quyền đi theo người nhận của VIỆC CHA (người được giao phần việc) hoặc Manager của dự án;
// đọc cha trong giao dịch đã khóa dự án nên không lệch khi cha vừa được giao lại.
// Chuyển vào chính cột hiện tại là no-op, không ghi lịch sử.
func (s *Service) Move(ctx context.Context, actor authz.Principal, id, columnID uuid.UUID) (domain.Task, error) {
	cur, p, err := s.viewChild(ctx, actor, id)
	if err != nil {
		return domain.Task{}, err
	}
	err = s.tx.InTx(ctx, func(st Store) error {
		locked, err := lockChildAndCheck(ctx, st, actor, p.ManagerID, cur.ProjectID, id)
		if err != nil {
			return err
		}
		if _, err := st.ColumnGroup(ctx, locked.ProjectID, columnID); err != nil {
			return fmt.Errorf("column group: %w", err)
		}
		if locked.ColumnID == columnID {
			return nil
		}
		if err := st.Move(ctx, id, columnID); err != nil {
			return fmt.Errorf("move subtask: %w", err)
		}
		rec := MoveRecord{TaskID: id, ProjectID: locked.ProjectID, FromColumnID: locked.ColumnID, ToColumnID: columnID, MovedBy: actor.UserID}
		if err := st.RecordMove(ctx, rec); err != nil {
			return fmt.Errorf("record move: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Task{}, err
	}
	return s.reload(ctx, id)
}

// Delete: cùng quyền với Move.
func (s *Service) Delete(ctx context.Context, actor authz.Principal, id uuid.UUID) error {
	cur, p, err := s.viewChild(ctx, actor, id)
	if err != nil {
		return err
	}
	return s.tx.InTx(ctx, func(st Store) error {
		if _, err := lockChildAndCheck(ctx, st, actor, p.ManagerID, cur.ProjectID, id); err != nil {
			return err
		}
		if err := st.Delete(ctx, id); err != nil {
			return fmt.Errorf("delete subtask: %w", err)
		}
		return nil
	})
}

// lockChildAndCheck khóa dự án → con, đọc việc cha rồi kiểm tra quyền theo người nhận của cha.
func lockChildAndCheck(ctx context.Context, st Store, actor authz.Principal, managerID, projectID, id uuid.UUID) (domain.Task, error) {
	locked, err := lockTask(ctx, st, projectID, id)
	if err != nil {
		return domain.Task{}, err
	}
	if locked.ParentID == nil {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	parent, err := st.Get(ctx, *locked.ParentID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get parent: %w", err)
	}
	if err := domain.CheckSubtaskActor(actor.UserID, actor.Role, managerID, parent.AssigneeID); err != nil {
		return domain.Task{}, err
	}
	return locked, nil
}

// lockTask khóa dự án rồi công việc (thứ tự cố định như kanban.Service.Move) và đọc bản đã khóa.
func lockTask(ctx context.Context, st Store, projectID, id uuid.UUID) (domain.Task, error) {
	if _, err := st.LockProject(ctx, projectID); err != nil {
		return domain.Task{}, fmt.Errorf("lock project: %w", err)
	}
	if err := st.Lock(ctx, id); err != nil {
		return domain.Task{}, fmt.Errorf("lock task: %w", err)
	}
	t, err := st.Get(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

func (s *Service) reload(ctx context.Context, id uuid.UUID) (domain.Task, error) {
	t, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("reload subtask: %w", err)
	}
	return t, nil
}
