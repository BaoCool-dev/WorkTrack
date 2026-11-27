package task

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Input là dữ liệu Manager nhập khi tạo/sửa công việc cha. Sửa là thay toàn bộ (PUT).
type Input struct {
	Title       string
	Description string
	StoryPoint  *int
	AssigneeID  *uuid.UUID
	DueDate     *time.Time
	LabelIDs    []uuid.UUID
}

// ListInput là bộ lọc danh sách công việc của dự án. Unassigned = true thì bỏ qua AssigneeID.
type ListInput struct {
	Search      string
	StatusGroup string
	ColumnID    *uuid.UUID
	AssigneeID  *uuid.UUID
	Unassigned  bool
	LabelID     *uuid.UUID
}

// MyListInput là bộ lọc trang "Công việc của tôi".
type MyListInput struct {
	Search      string
	StatusGroup string
}

// Service xử lý use case công việc cha (CF-06).
type Service struct {
	store    Store
	tx       Transactor
	projects Projects
	settings Settings
	pub      Publisher
}

// NewService tạo Service.
func NewService(store Store, tx Transactor, projects Projects, settings Settings, pub Publisher) *Service {
	return &Service{store: store, tx: tx, projects: projects, settings: settings, pub: pub}
}

// publish phát sự kiện bảng Kanban sau khi transaction đã commit.
func (s *Service) publish(ctx context.Context, typ string, actor authz.Principal, t domain.Task) {
	s.pub.Publish(ctx, domain.BoardEvent{
		Type: typ, ProjectID: t.ProjectID, TaskID: t.ID, ActorID: actor.UserID,
		ToColumnID: t.ColumnID, Position: t.Position, Version: t.Version,
	})
}

// lockForWrite khóa dự án rồi công việc (thứ tự cố định — xem kanban.Service.Move) và đọc bản đã khóa.
// Trả kèm giới hạn WIP đọc trên dòng dự án đã khóa (Manager vừa đổi giới hạn thì dùng số mới).
func lockForWrite(ctx context.Context, st Store, projectID, id uuid.UUID) (domain.Task, domain.LockedProject, error) {
	lp, err := st.LockProject(ctx, projectID)
	if err != nil {
		return domain.Task{}, domain.LockedProject{}, fmt.Errorf("lock project: %w", err)
	}
	if err := st.Lock(ctx, id); err != nil {
		return domain.Task{}, domain.LockedProject{}, fmt.Errorf("lock task: %w", err)
	}
	t, err := st.Get(ctx, id)
	if err != nil {
		return domain.Task{}, domain.LockedProject{}, fmt.Errorf("get task: %w", err)
	}
	return t, lp, nil
}

// checkWIP chặn khi userID đã có đủ số việc đang làm trong dự án (CF-05). Gọi sau khi đã khóa dự án.
func checkWIP(ctx context.Context, st Store, projectID uuid.UUID, lp domain.LockedProject, userID uuid.UUID) error {
	n, err := st.CountProjectWIP(ctx, projectID, userID)
	if err != nil {
		return fmt.Errorf("count project wip: %w", err)
	}
	return domain.CheckWIP(n, lp.WIPLimit)
}

// view tải công việc rồi kiểm quyền xem dự án chứa nó. Không xem được → ErrTaskNotFound (không lộ sự tồn tại).
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

// Get trả chi tiết công việc cho người xem được dự án.
func (s *Service) Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Task, error) {
	t, _, err := s.view(ctx, actor, id)
	return t, err
}

func normalizeSearch(raw string) (string, error) {
	q := strings.TrimSpace(raw)
	if utf8.RuneCountInString(q) > SearchMaxRunes {
		return "", domain.ErrInvalidSearch
	}
	return q, nil
}

func parseGroup(raw string) (*domain.StatusGroup, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	g, err := domain.ParseStatusGroup(raw)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// List trả công việc cha của dự án (người xem được dự án), có lọc và phân trang (B9).
func (s *Service) List(ctx context.Context, actor authz.Principal, projectID uuid.UUID, in ListInput, p domain.PageRequest) (domain.Page[domain.Task], error) {
	if _, err := s.projects.Get(ctx, actor, projectID); err != nil {
		return domain.Page[domain.Task]{}, err
	}
	q, err := normalizeSearch(in.Search)
	if err != nil {
		return domain.Page[domain.Task]{}, err
	}
	g, err := parseGroup(in.StatusGroup)
	if err != nil {
		return domain.Page[domain.Task]{}, err
	}
	f := Filter{Search: q, StatusGroup: g, ColumnID: in.ColumnID, Unassigned: in.Unassigned, LabelID: in.LabelID}
	if !in.Unassigned {
		f.AssigneeID = in.AssigneeID
	}
	items, total, err := s.store.List(ctx, projectID, f, p)
	if err != nil {
		return domain.Page[domain.Task]{}, fmt.Errorf("list tasks: %w", err)
	}
	return page(items, total, p), nil
}

func page(items []domain.Task, total int, p domain.PageRequest) domain.Page[domain.Task] {
	if items == nil {
		items = []domain.Task{}
	}
	return domain.Page[domain.Task]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: total}
}

// ListMine trả công việc cha giao cho chính người gọi (Manager/Member).
func (s *Service) ListMine(ctx context.Context, actor authz.Principal, in MyListInput, p domain.PageRequest) (domain.Page[domain.Task], error) {
	if err := authz.Require(actor, authz.ViewOwnProjects); err != nil {
		return domain.Page[domain.Task]{}, err
	}
	q, err := normalizeSearch(in.Search)
	if err != nil {
		return domain.Page[domain.Task]{}, err
	}
	g, err := parseGroup(in.StatusGroup)
	if err != nil {
		return domain.Page[domain.Task]{}, err
	}
	items, total, err := s.store.ListMine(ctx, actor.UserID, MyFilter{Search: q, StatusGroup: g}, p)
	if err != nil {
		return domain.Page[domain.Task]{}, fmt.Errorf("list my tasks: %w", err)
	}
	return page(items, total, p), nil
}

// MyWIP trả số công việc cha nhóm in_progress của người gọi trên mọi dự án (§5.6). Chỉ hiển thị;
// giới hạn WIP theo dự án (CF-05) chặn ở Claim/Update và kanban.Service.Move.
func (s *Service) MyWIP(ctx context.Context, actor authz.Principal) (int, error) {
	if err := authz.Require(actor, authz.ViewOwnProjects); err != nil {
		return 0, err
	}
	n, err := s.store.CountWIP(ctx, actor.UserID)
	if err != nil {
		return 0, fmt.Errorf("count wip: %w", err)
	}
	return n, nil
}

// fields chuẩn hóa Input (không I/O). Giờ ước tính điền sau theo B1.
func fields(in Input) (Fields, error) {
	title, err := domain.NormalizeTaskTitle(in.Title)
	if err != nil {
		return Fields{}, err
	}
	desc, err := domain.NormalizeTaskDescription(in.Description)
	if err != nil {
		return Fields{}, err
	}
	if in.StoryPoint != nil {
		if err := domain.ValidateStoryPoint(*in.StoryPoint); err != nil {
			return Fields{}, err
		}
	}
	labels, err := domain.NormalizeLabelIDs(in.LabelIDs)
	if err != nil {
		return Fields{}, err
	}
	return Fields{Title: title, Description: desc, StoryPoint: in.StoryPoint,
		AssigneeID: in.AssigneeID, DueDate: in.DueDate, LabelIDs: labels}, nil
}

// checkAssignee: người nhận phải là thành viên dự án, vai trò Manager/Member, chưa khóa.
func checkAssignee(ctx context.Context, st Store, projectID uuid.UUID, assignee *uuid.UUID) error {
	if assignee == nil {
		return nil
	}
	a, ok, err := st.GetAssignable(ctx, projectID, *assignee)
	if err != nil {
		return fmt.Errorf("get assignee: %w", err)
	}
	if !ok || !domain.CanBeAssigned(a.Role, a.Status) {
		return domain.ErrInvalidAssignee
	}
	return nil
}

func (s *Service) estimate(ctx context.Context, sp int) (*int, error) {
	h, err := s.settings.HoursPerStoryPoint(ctx)
	if err != nil {
		return nil, fmt.Errorf("story point hours: %w", err)
	}
	m := domain.EstimateMinutes(sp, h)
	return &m, nil
}

// Create: Manager phụ trách tạo công việc cha ở cột todo đầu tiên của dự án.
func (s *Service) Create(ctx context.Context, actor authz.Principal, projectID uuid.UUID, in Input) (domain.Task, error) {
	p, err := s.projects.Get(ctx, actor, projectID)
	if err != nil {
		return domain.Task{}, err
	}
	if err := authz.RequireProjectManager(actor, p.ManagerID); err != nil {
		return domain.Task{}, err
	}
	f, err := fields(in)
	if err != nil {
		return domain.Task{}, err
	}
	if f.StoryPoint != nil {
		if f.EstimatedMinutes, err = s.estimate(ctx, *f.StoryPoint); err != nil {
			return domain.Task{}, err
		}
	}
	var id uuid.UUID
	err = s.tx.InTx(ctx, func(st Store) error {
		if err := checkAssignee(ctx, st, projectID, f.AssigneeID); err != nil {
			return err
		}
		col, err := st.FirstTodoColumn(ctx, projectID)
		if err != nil {
			return fmt.Errorf("first todo column: %w", err)
		}
		if id, err = st.Create(ctx, projectID, col, actor.UserID, f); err != nil {
			return fmt.Errorf("create task: %w", err)
		}
		return notifyAssigned(ctx, st, actor, projectID, id, nil, f.AssigneeID)
	})
	if err != nil {
		return domain.Task{}, err
	}
	return s.reloadAndPublish(ctx, domain.EventTaskCreated, actor, id)
}

func (s *Service) reloadAndPublish(ctx context.Context, typ string, actor authz.Principal, id uuid.UUID) (domain.Task, error) {
	t, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("reload task: %w", err)
	}
	s.publish(ctx, typ, actor, t)
	return t, nil
}

// Update: Manager phụ trách sửa toàn bộ thông tin công việc cha, gồm người nhận và nhãn. Không đổi cột.
// Giờ ước tính chỉ tính lại khi Story Point đổi (đổi hệ số B1 không sửa việc cũ).
func (s *Service) Update(ctx context.Context, actor authz.Principal, id uuid.UUID, in Input) (domain.Task, error) {
	cur, p, err := s.view(ctx, actor, id)
	if err != nil {
		return domain.Task{}, err
	}
	if cur.ParentID != nil {
		return domain.Task{}, domain.ErrTaskNotFound // công việc con thuộc CF-07
	}
	if err := authz.RequireProjectManager(actor, p.ManagerID); err != nil {
		return domain.Task{}, err
	}
	f, err := fields(in)
	if err != nil {
		return domain.Task{}, err
	}
	err = s.tx.InTx(ctx, func(st Store) error {
		locked, lp, err := lockForWrite(ctx, st, p.ID, id)
		if err != nil {
			return err
		}
		switch {
		case f.StoryPoint == nil:
		case locked.StoryPoint != nil && *locked.StoryPoint == *f.StoryPoint && locked.EstimatedMinutes != nil:
			f.EstimatedMinutes = locked.EstimatedMinutes
		default:
			if f.EstimatedMinutes, err = s.estimate(ctx, *f.StoryPoint); err != nil {
				return err
			}
		}
		if err := checkAssignee(ctx, st, locked.ProjectID, f.AssigneeID); err != nil {
			return err
		}
		// Giao lại một việc ĐANG LÀM cho người khác = người đó thêm 1 việc in_progress → kiểm WIP của họ.
		if locked.StatusGroup == domain.GroupInProgress && f.AssigneeID != nil && !sameUser(locked.AssigneeID, f.AssigneeID) {
			if err := checkWIP(ctx, st, p.ID, lp, *f.AssigneeID); err != nil {
				return err
			}
		}
		if err := st.Update(ctx, id, f); err != nil {
			return fmt.Errorf("update task: %w", err)
		}
		return notifyAssigned(ctx, st, actor, locked.ProjectID, id, locked.AssigneeID, f.AssigneeID)
	})
	if err != nil {
		return domain.Task{}, err
	}
	return s.reloadAndPublish(ctx, domain.EventTaskUpdated, actor, id)
}

// notifyAssigned báo "được giao việc" cho người nhận mới (CF-07). Ghi cùng transaction với thay đổi
// để không có thông báo cho một lần giao bị rollback. Không có người nhận mới → không ghi gì.
func notifyAssigned(ctx context.Context, st Store, actor authz.Principal, projectID, taskID uuid.UUID, old, cur *uuid.UUID) error {
	to := domain.AssignmentRecipient(actor.UserID, old, cur)
	if to == nil {
		return nil
	}
	n := domain.NewNotification{UserIDs: []uuid.UUID{*to}, Type: domain.NotificationTaskAssigned,
		TaskID: taskID, ProjectID: projectID, ActorID: actor.UserID}
	if err := st.AddNotification(ctx, n); err != nil {
		return fmt.Errorf("notify assigned: %w", err)
	}
	return nil
}

func sameUser(a, b *uuid.UUID) bool {
	return a != nil && b != nil && *a == *b
}

// Claim: Member tự nhận công việc cha chưa có người nhận, đang ở nhóm todo. Không chuyển cột.
func (s *Service) Claim(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Task, error) {
	cur, p, err := s.view(ctx, actor, id)
	if err != nil {
		return domain.Task{}, err
	}
	if cur.ParentID != nil {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err := authz.Require(actor, authz.ClaimTask); err != nil {
		return domain.Task{}, err
	}
	err = s.tx.InTx(ctx, func(st Store) error {
		t, lp, err := lockForWrite(ctx, st, p.ID, id)
		if err != nil {
			return err
		}
		if t.AssigneeID != nil {
			return domain.ErrTaskAlreadyAssigned
		}
		if t.StatusGroup != domain.GroupTodo {
			return domain.ErrTaskNotClaimable
		}
		a, ok, err := st.GetAssignable(ctx, t.ProjectID, actor.UserID)
		if err != nil {
			return fmt.Errorf("get assignable: %w", err)
		}
		if !ok || !domain.CanBeAssigned(a.Role, a.Status) {
			return domain.ErrForbidden
		}
		// Đã đủ WIP thì không nhận thêm việc: làm xong việc đang dở trước (CF-05, quyết định mục C).
		if err := checkWIP(ctx, st, p.ID, lp, actor.UserID); err != nil {
			return err
		}
		uid := actor.UserID
		if err := st.SetAssignee(ctx, id, &uid); err != nil {
			return fmt.Errorf("set assignee: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Task{}, err
	}
	return s.reloadAndPublish(ctx, domain.EventTaskUpdated, actor, id)
}
