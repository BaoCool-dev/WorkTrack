package timelog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Service xử lý use case Log time (CF-08).
type Service struct {
	store    Store
	tx       Transactor
	projects Projects
	pub      Publisher
	now      func() time.Time
}

// NewService tạo Service. now cho phép test cố định đồng hồ (luật 24 giờ, ngày mặc định).
func NewService(store Store, tx Transactor, projects Projects, pub Publisher, now func() time.Time) *Service {
	return &Service{store: store, tx: tx, projects: projects, pub: pub, now: now}
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

// List trả lịch sử log của công việc cho ai xem được dự án (kể cả CEO).
// Với việc cha, danh sách gồm cả log của các con — khớp với tổng "giờ thực tế" trên thẻ.
func (s *Service) List(ctx context.Context, actor authz.Principal, taskID uuid.UUID, p domain.PageRequest) (domain.Page[domain.TimeLog], error) {
	if _, _, err := s.view(ctx, actor, taskID); err != nil {
		return domain.Page[domain.TimeLog]{}, err
	}
	items, total, err := s.store.ListLogs(ctx, taskID, p)
	if err != nil {
		return domain.Page[domain.TimeLog]{}, fmt.Errorf("list time logs: %w", err)
	}
	if items == nil {
		items = []domain.TimeLog{}
	}
	return domain.Page[domain.TimeLog]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: total}, nil
}

// Create ghi giờ lên công việc (cha hoặc con).
// Kiểm quyền hai lần: trước giao dịch để trả lỗi sớm, và trong giao dịch đã khóa
// để không lệch khi Manager vừa giao lại việc hoặc vừa kéo việc sang Done.
func (s *Service) Create(ctx context.Context, actor authz.Principal, taskID uuid.UUID, in Input) (domain.TimeLog, error) {
	t, p, err := s.view(ctx, actor, taskID)
	if err != nil {
		return domain.TimeLog{}, err
	}
	if _, err := s.checkActor(ctx, s.store, actor, p.ManagerID, t); err != nil {
		return domain.TimeLog{}, err
	}
	note, err := domain.NormalizeTimeLog(in.Minutes, in.Note)
	if err != nil {
		return domain.TimeLog{}, err
	}
	now := s.now()
	workDate, err := domain.ResolveWorkDate(in.WorkDate, t.CreatedAt, now)
	if err != nil {
		return domain.TimeLog{}, err
	}
	var id, rootID uuid.UUID
	err = s.tx.InTx(ctx, func(st Store) error {
		locked, err := lockTask(ctx, st, t.ProjectID, taskID)
		if err != nil {
			return err
		}
		root, err := s.checkActor(ctx, st, actor, p.ManagerID, locked)
		if err != nil {
			return err
		}
		if err := checkWindows(ctx, st, now, locked, root); err != nil {
			return err
		}
		rootID = root.ID
		id, err = st.Create(ctx, NewLog{
			TaskID: taskID, ProjectID: locked.ProjectID, UserID: actor.UserID,
			Minutes: in.Minutes, Note: note, WorkDate: workDate,
		})
		if err != nil {
			return fmt.Errorf("create time log: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.TimeLog{}, err
	}
	s.publishRoot(ctx, actor, rootID)
	l, err := s.store.GetLog(ctx, id)
	if err != nil {
		return domain.TimeLog{}, fmt.Errorf("reload time log: %w", err)
	}
	return l, nil
}

// Delete: chỉ tác giả xóa dòng log của mình, cùng luật 24 giờ như khi ghi.
func (s *Service) Delete(ctx context.Context, actor authz.Principal, logID uuid.UUID) error {
	l, err := s.store.GetLog(ctx, logID)
	if err != nil {
		return fmt.Errorf("get time log: %w", err)
	}
	if _, _, err := s.view(ctx, actor, l.TaskID); err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			return domain.ErrTimeLogNotFound
		}
		return err
	}
	if err := domain.CheckTimeLogDelete(actor.UserID, l); err != nil {
		return err
	}
	var rootID uuid.UUID
	err = s.tx.InTx(ctx, func(st Store) error {
		locked, err := lockTask(ctx, st, l.ProjectID, l.TaskID)
		if err != nil {
			return err
		}
		root, err := rootOf(ctx, st, locked)
		if err != nil {
			return err
		}
		if err := checkWindows(ctx, st, s.now(), locked, root); err != nil {
			return err
		}
		rootID = root.ID
		if err := st.DeleteLog(ctx, logID); err != nil {
			return fmt.Errorf("delete time log: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.publishRoot(ctx, actor, rootID)
	return nil
}

// checkActor kiểm quyền ghi giờ theo người nhận của việc GỐC (việc con → việc cha, như CF-07).
// Trả việc gốc để dùng cho luật 24 giờ và sự kiện realtime.
func (s *Service) checkActor(ctx context.Context, st Store, actor authz.Principal, managerID uuid.UUID, t domain.Task) (domain.Task, error) {
	root, err := rootOf(ctx, st, t)
	if err != nil {
		return domain.Task{}, err
	}
	if err := domain.CheckTimeLogActor(actor.UserID, actor.Role, managerID, root.AssigneeID); err != nil {
		return domain.Task{}, err
	}
	return root, nil
}

// rootOf trả việc cha nếu t là việc con, ngược lại trả chính t.
func rootOf(ctx context.Context, st Store, t domain.Task) (domain.Task, error) {
	if t.ParentID == nil {
		return t, nil
	}
	parent, err := st.Get(ctx, *t.ParentID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get parent: %w", err)
	}
	return parent, nil
}

// checkWindows áp luật "không log vào việc đã Done quá 24 giờ" cho công việc và việc cha của nó:
// cha đã nghiệm thu xong thì giờ của con (cộng lên cha trong báo cáo) cũng phải khóa.
func checkWindows(ctx context.Context, st Store, now time.Time, tasks ...domain.Task) error {
	for _, t := range tasks {
		if t.StatusGroup != domain.GroupDone {
			continue
		}
		doneAt, err := st.LastDoneAt(ctx, t.ID)
		if err != nil {
			return fmt.Errorf("last done at: %w", err)
		}
		// Không có lịch sử chuyển cột (dữ liệu cũ/seed) → dùng lần cập nhật cuối làm mốc.
		at := t.UpdatedAt
		if doneAt != nil {
			at = *doneAt
		}
		if err := domain.CheckLogWindow(t.StatusGroup, at, now); err != nil {
			return err
		}
	}
	return nil
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

// publishRoot báo thẻ việc gốc đổi giờ thực tế để mọi người đang mở bảng làm mới thanh giờ.
// Đọc lại sau commit để gửi version mới nhất; lỗi đọc chỉ bỏ qua sự kiện (dữ liệu đã lưu xong).
func (s *Service) publishRoot(ctx context.Context, actor authz.Principal, rootID uuid.UUID) {
	t, err := s.store.Get(ctx, rootID)
	if err != nil {
		return
	}
	s.pub.Publish(ctx, domain.BoardEvent{
		Type: domain.EventTaskUpdated, ProjectID: t.ProjectID, TaskID: t.ID, ActorID: actor.UserID,
		ToColumnID: t.ColumnID, Position: t.Position, Version: t.Version,
	})
}
