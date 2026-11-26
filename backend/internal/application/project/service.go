package project

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

// Input là thông tin dự án do client gửi (chưa chuẩn hóa).
type Input struct {
	Name        string
	Description string
	// Status rỗng khi tạo = "new".
	Status    string
	StartDate *time.Time
	EndDate   *time.Time
}

// ListInput là tham số lọc danh sách dự án do client gửi.
type ListInput struct {
	Search string
	Status string
}

// Service là các use case CF-03.
type Service struct {
	store Store
	tx    Transactor
}

// NewService tạo Service.
func NewService(store Store, tx Transactor) *Service {
	return &Service{store: store, tx: tx}
}

func normalize(in Input, defaultStatus domain.ProjectStatus) (Fields, error) {
	name, err := domain.NormalizeProjectName(in.Name)
	if err != nil {
		return Fields{}, err
	}
	desc, err := domain.NormalizeProjectDescription(in.Description)
	if err != nil {
		return Fields{}, err
	}
	status := defaultStatus
	if strings.TrimSpace(in.Status) != "" {
		if status, err = domain.ParseProjectStatus(strings.TrimSpace(in.Status)); err != nil {
			return Fields{}, err
		}
	}
	if err := domain.ValidateProjectDates(in.StartDate, in.EndDate); err != nil {
		return Fields{}, err
	}
	return Fields{Name: name, Description: desc, Status: status, StartDate: in.StartDate, EndDate: in.EndDate}, nil
}

// Create: chỉ Manager tạo dự án trống; người tạo là Manager phụ trách và là thành viên đầu tiên.
// Dự án mới có 5 cột mặc định và luật chuyển trạng thái mặc định.
func (s *Service) Create(ctx context.Context, actor authz.Principal, in Input) (domain.Project, error) {
	if err := authz.Require(actor, authz.CreateProject); err != nil {
		return domain.Project{}, err
	}
	f, err := normalize(in, domain.ProjectNew)
	if err != nil {
		return domain.Project{}, err
	}
	var id uuid.UUID
	err = s.tx.InTx(ctx, func(st Store) error {
		var err error
		if id, err = st.Create(ctx, NewProject{Fields: f, ManagerID: actor.UserID, CreatedBy: actor.UserID}); err != nil {
			return fmt.Errorf("create project: %w", err)
		}
		if err := st.AddMember(ctx, id, actor.UserID); err != nil {
			return fmt.Errorf("add manager as member: %w", err)
		}
		for i, c := range domain.DefaultColumns() {
			if _, err := st.CreateColumn(ctx, id, c, i); err != nil {
				return fmt.Errorf("create default column: %w", err)
			}
		}
		cols, err := st.ListColumns(ctx, id)
		if err != nil {
			return fmt.Errorf("list columns: %w", err)
		}
		if err := st.ReplaceTransitions(ctx, id, domain.DefaultTransitions(cols)); err != nil {
			return fmt.Errorf("default transitions: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Project{}, err
	}
	return s.get(ctx, id)
}

// List: CEO xem tất cả; Manager/Member chỉ dự án mình tham gia; Admin không xem dự án.
func (s *Service) List(ctx context.Context, actor authz.Principal, in ListInput, p domain.PageRequest) (domain.Page[domain.Project], error) {
	var f Filter
	switch {
	case authz.Can(actor.Role, authz.ViewAllProjects):
	case authz.Can(actor.Role, authz.ViewOwnProjects):
		uid := actor.UserID
		f.VisibleTo = &uid
	default:
		return domain.Page[domain.Project]{}, domain.ErrForbidden
	}
	f.Search = strings.TrimSpace(in.Search)
	if utf8.RuneCountInString(f.Search) > SearchMaxRunes {
		return domain.Page[domain.Project]{}, domain.ErrInvalidSearch
	}
	if st := strings.TrimSpace(in.Status); st != "" {
		ps, err := domain.ParseProjectStatus(st)
		if err != nil {
			return domain.Page[domain.Project]{}, err
		}
		f.Status = &ps
	}
	items, total, err := s.store.List(ctx, f, p)
	if err != nil {
		return domain.Page[domain.Project]{}, fmt.Errorf("list projects: %w", err)
	}
	if items == nil {
		items = []domain.Project{}
	}
	return domain.Page[domain.Project]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: total}, nil
}

func (s *Service) get(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	p, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Project{}, fmt.Errorf("get project: %w", err)
	}
	return p, nil
}

// canView: CEO xem mọi dự án; Manager/Member chỉ dự án mình là thành viên.
// Không được xem → ErrProjectNotFound (không lộ sự tồn tại của dự án).
func (s *Service) canView(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error) {
	p, err := s.get(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	if authz.Can(actor.Role, authz.ViewAllProjects) {
		return p, nil
	}
	if !authz.Can(actor.Role, authz.ViewOwnProjects) {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	if p.ManagerID == actor.UserID {
		return p, nil
	}
	ok, err := s.store.IsMember(ctx, id, actor.UserID)
	if err != nil {
		return domain.Project{}, fmt.Errorf("check member: %w", err)
	}
	if !ok {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return p, nil
}

// manage kiểm tra quyền xem rồi quyền quản lý (Manager phụ trách) — người xem được nhưng không quản lý → 403.
func (s *Service) manage(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error) {
	p, err := s.canView(ctx, actor, id)
	if err != nil {
		return domain.Project{}, err
	}
	if err := authz.RequireProjectManager(actor, p.ManagerID); err != nil {
		return domain.Project{}, err
	}
	return p, nil
}

// lockedManage chạy fn trong transaction sau khi khóa dòng dự án — tránh hai thao tác cấu hình cột song song.
func (s *Service) lockedManage(ctx context.Context, actor authz.Principal, id uuid.UUID, fn func(Store) error) error {
	if _, err := s.manage(ctx, actor, id); err != nil {
		return err
	}
	return s.tx.InTx(ctx, func(st Store) error {
		if err := st.Lock(ctx, id); err != nil {
			return fmt.Errorf("lock project: %w", err)
		}
		return fn(st)
	})
}

// SetWIPLimit: Manager phụ trách đặt giới hạn WIP mỗi người của dự án (CF-05, 1–20).
// Giảm giới hạn xuống dưới số việc đang làm không chuyển việc nào ra — chỉ chặn kéo/nhận thêm.
func (s *Service) SetWIPLimit(ctx context.Context, actor authz.Principal, id uuid.UUID, limit int) (domain.Project, error) {
	if _, err := s.manage(ctx, actor, id); err != nil {
		return domain.Project{}, err
	}
	if err := domain.ValidateWIPLimit(limit); err != nil {
		return domain.Project{}, err
	}
	if err := s.store.UpdateWIPLimit(ctx, id, limit); err != nil {
		return domain.Project{}, fmt.Errorf("update wip limit: %w", err)
	}
	return s.get(ctx, id)
}

// Get trả chi tiết dự án.
func (s *Service) Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error) {
	return s.canView(ctx, actor, id)
}

// Update sửa thông tin và trạng thái dự án (Manager phụ trách). Không đổi Manager phụ trách.
func (s *Service) Update(ctx context.Context, actor authz.Principal, id uuid.UUID, in Input) (domain.Project, error) {
	cur, err := s.manage(ctx, actor, id)
	if err != nil {
		return domain.Project{}, err
	}
	f, err := normalize(in, cur.Status)
	if err != nil {
		return domain.Project{}, err
	}
	if err := s.store.Update(ctx, id, f); err != nil {
		return domain.Project{}, fmt.Errorf("update project: %w", err)
	}
	return s.get(ctx, id)
}

// ListMembers: ai xem được dự án thì xem được thành viên.
func (s *Service) ListMembers(ctx context.Context, actor authz.Principal, id uuid.UUID) ([]domain.ProjectMember, error) {
	if _, err := s.canView(ctx, actor, id); err != nil {
		return nil, err
	}
	ms, err := s.store.ListMembers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	if ms == nil {
		ms = []domain.ProjectMember{}
	}
	return ms, nil
}

// ListCandidates: Manager phụ trách tìm người để thêm vào dự án.
func (s *Service) ListCandidates(ctx context.Context, actor authz.Principal, id uuid.UUID, search string) ([]domain.MemberCandidate, error) {
	if _, err := s.manage(ctx, actor, id); err != nil {
		return nil, err
	}
	search = strings.TrimSpace(search)
	if utf8.RuneCountInString(search) > SearchMaxRunes {
		return nil, domain.ErrInvalidSearch
	}
	cs, err := s.store.ListCandidates(ctx, id, search)
	if err != nil {
		return nil, fmt.Errorf("list candidates: %w", err)
	}
	if cs == nil {
		cs = []domain.MemberCandidate{}
	}
	return cs, nil
}

// AddMember: chỉ thêm tài khoản Manager/Member chưa bị khóa. Đã là thành viên → không lỗi.
func (s *Service) AddMember(ctx context.Context, actor authz.Principal, id, userID uuid.UUID) error {
	if _, err := s.manage(ctx, actor, id); err != nil {
		return err
	}
	c, err := s.store.GetCandidate(ctx, userID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.ErrInvalidProjectMember
	}
	if err != nil {
		return fmt.Errorf("get candidate: %w", err)
	}
	if (c.Role != domain.RoleManager && c.Role != domain.RoleMember) || c.Status == domain.UserStatusDisabled {
		return domain.ErrInvalidProjectMember
	}
	if err := s.store.AddMember(ctx, id, userID); err != nil {
		return fmt.Errorf("add member: %w", err)
	}
	return nil
}

// RemoveMember: không bỏ được Manager phụ trách.
func (s *Service) RemoveMember(ctx context.Context, actor authz.Principal, id, userID uuid.UUID) error {
	p, err := s.manage(ctx, actor, id)
	if err != nil {
		return err
	}
	if userID == p.ManagerID {
		return domain.ErrCannotRemoveProjectManager
	}
	if err := s.store.RemoveMember(ctx, id, userID); err != nil {
		return fmt.Errorf("remove member: %w", err)
	}
	return nil
}

// Board là cấu hình Kanban của dự án: cột theo thứ tự + luật chuyển.
type Board struct {
	Columns     []domain.BoardColumn
	Transitions []domain.Transition
}

// GetBoard: ai xem được dự án thì xem được cấu hình cột và luật chuyển.
func (s *Service) GetBoard(ctx context.Context, actor authz.Principal, id uuid.UUID) (Board, error) {
	if _, err := s.canView(ctx, actor, id); err != nil {
		return Board{}, err
	}
	return s.board(ctx, s.store, id)
}

func (s *Service) board(ctx context.Context, st Store, id uuid.UUID) (Board, error) {
	cols, err := st.ListColumns(ctx, id)
	if err != nil {
		return Board{}, fmt.Errorf("list columns: %w", err)
	}
	ts, err := st.ListTransitions(ctx, id)
	if err != nil {
		return Board{}, fmt.Errorf("list transitions: %w", err)
	}
	if cols == nil {
		cols = []domain.BoardColumn{}
	}
	if ts == nil {
		ts = []domain.Transition{}
	}
	return Board{Columns: cols, Transitions: ts}, nil
}

func normalizeSpec(name, group string) (domain.ColumnSpec, error) {
	n, err := domain.NormalizeColumnName(name)
	if err != nil {
		return domain.ColumnSpec{}, err
	}
	g, err := domain.ParseStatusGroup(strings.TrimSpace(group))
	if err != nil {
		return domain.ColumnSpec{}, err
	}
	return domain.ColumnSpec{Name: n, StatusGroup: g}, nil
}

// AddColumn thêm cột vào cuối bảng; cột mới nhận luật chuyển mặc định với các cột hiện có.
func (s *Service) AddColumn(ctx context.Context, actor authz.Principal, id uuid.UUID, name, group string) (Board, error) {
	spec, err := normalizeSpec(name, group)
	if err != nil {
		return Board{}, err
	}
	var out Board
	err = s.lockedManage(ctx, actor, id, func(st Store) error {
		cols, err := st.ListColumns(ctx, id)
		if err != nil {
			return fmt.Errorf("list columns: %w", err)
		}
		if err := domain.ValidateColumnSpecs(append(domain.Specs(cols), spec)); err != nil {
			return err
		}
		newID, err := st.CreateColumn(ctx, id, spec, len(cols))
		if err != nil {
			return fmt.Errorf("create column: %w", err)
		}
		all, err := st.ListColumns(ctx, id)
		if err != nil {
			return fmt.Errorf("list columns: %w", err)
		}
		ts, err := st.ListTransitions(ctx, id)
		if err != nil {
			return fmt.Errorf("list transitions: %w", err)
		}
		ts = append(ts, domain.TransitionsForColumn(all, newID)...)
		if err := st.ReplaceTransitions(ctx, id, ts); err != nil {
			return fmt.Errorf("replace transitions: %w", err)
		}
		out, err = s.board(ctx, st, id)
		return err
	})
	return out, err
}

// UpdateColumn đổi tên / nhóm trạng thái; luật không còn hợp lệ (Member vào nhóm done) bị bỏ.
func (s *Service) UpdateColumn(ctx context.Context, actor authz.Principal, id, columnID uuid.UUID, name, group string) (Board, error) {
	spec, err := normalizeSpec(name, group)
	if err != nil {
		return Board{}, err
	}
	var out Board
	err = s.lockedManage(ctx, actor, id, func(st Store) error {
		cols, err := st.ListColumns(ctx, id)
		if err != nil {
			return fmt.Errorf("list columns: %w", err)
		}
		found := false
		for i := range cols {
			if cols[i].ID == columnID {
				cols[i].Name, cols[i].StatusGroup = spec.Name, spec.StatusGroup
				found = true
			}
		}
		if !found {
			return domain.ErrColumnNotFound
		}
		if err := domain.ValidateColumnSpecs(domain.Specs(cols)); err != nil {
			return err
		}
		if err := st.UpdateColumn(ctx, id, columnID, spec); err != nil {
			return fmt.Errorf("update column: %w", err)
		}
		ts, err := st.ListTransitions(ctx, id)
		if err != nil {
			return fmt.Errorf("list transitions: %w", err)
		}
		if err := st.ReplaceTransitions(ctx, id, domain.PruneTransitions(cols, ts)); err != nil {
			return fmt.Errorf("replace transitions: %w", err)
		}
		out, err = s.board(ctx, st, id)
		return err
	})
	return out, err
}

// DeleteColumn xóa cột (còn ≥2 cột, còn cột done, cột không còn việc); đánh lại vị trí 0..n-1.
func (s *Service) DeleteColumn(ctx context.Context, actor authz.Principal, id, columnID uuid.UUID) (Board, error) {
	var out Board
	err := s.lockedManage(ctx, actor, id, func(st Store) error {
		cols, err := st.ListColumns(ctx, id)
		if err != nil {
			return fmt.Errorf("list columns: %w", err)
		}
		if err := domain.CheckColumnDeletable(cols, columnID); err != nil {
			return err
		}
		if err := st.DeleteColumn(ctx, id, columnID); err != nil {
			return fmt.Errorf("delete column: %w", err)
		}
		pos := 0
		for _, c := range cols {
			if c.ID == columnID {
				continue
			}
			if c.Position != pos {
				if err := st.SetColumnPosition(ctx, id, c.ID, pos); err != nil {
					return fmt.Errorf("renumber column: %w", err)
				}
			}
			pos++
		}
		out, err = s.board(ctx, st, id)
		return err
	})
	return out, err
}

// ReorderColumns sắp lại thứ tự cột; ids phải liệt kê đủ mỗi cột một lần.
func (s *Service) ReorderColumns(ctx context.Context, actor authz.Principal, id uuid.UUID, ids []uuid.UUID) (Board, error) {
	var out Board
	err := s.lockedManage(ctx, actor, id, func(st Store) error {
		cols, err := st.ListColumns(ctx, id)
		if err != nil {
			return fmt.Errorf("list columns: %w", err)
		}
		if err := domain.ValidateColumnOrder(cols, ids); err != nil {
			return err
		}
		for i, cid := range ids {
			if err := st.SetColumnPosition(ctx, id, cid, i); err != nil {
				return fmt.Errorf("set column position: %w", err)
			}
		}
		out, err = s.board(ctx, st, id)
		return err
	})
	return out, err
}

// ReplaceTransitions thay toàn bộ luật chuyển trạng thái của dự án (chỉ lưu cấu hình; áp dụng ở CF-04).
func (s *Service) ReplaceTransitions(ctx context.Context, actor authz.Principal, id uuid.UUID, ts []domain.Transition) (Board, error) {
	var out Board
	err := s.lockedManage(ctx, actor, id, func(st Store) error {
		cols, err := st.ListColumns(ctx, id)
		if err != nil {
			return fmt.Errorf("list columns: %w", err)
		}
		if err := domain.ValidateTransitions(cols, ts); err != nil {
			return err
		}
		if err := st.ReplaceTransitions(ctx, id, ts); err != nil {
			return fmt.Errorf("replace transitions: %w", err)
		}
		out, err = s.board(ctx, st, id)
		return err
	})
	return out, err
}
