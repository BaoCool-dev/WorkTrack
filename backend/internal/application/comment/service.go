package comment

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Service xử lý use case bình luận (CF-07).
type Service struct {
	store    Store
	tx       Transactor
	projects Projects
}

// NewService tạo Service.
func NewService(store Store, tx Transactor, projects Projects) *Service {
	return &Service{store: store, tx: tx, projects: projects}
}

// view: không xem được dự án → ErrTaskNotFound (404, chống dò ID).
func (s *Service) view(ctx context.Context, actor authz.Principal, taskID uuid.UUID) (domain.Task, error) {
	t, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task: %w", err)
	}
	if _, err := s.projects.Get(ctx, actor, t.ProjectID); err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		return domain.Task{}, err
	}
	return t, nil
}

// List trả bình luận của công việc, cũ trước mới sau.
func (s *Service) List(ctx context.Context, actor authz.Principal, taskID uuid.UUID, p domain.PageRequest) (domain.Page[domain.Comment], error) {
	if _, err := s.view(ctx, actor, taskID); err != nil {
		return domain.Page[domain.Comment]{}, err
	}
	items, total, err := s.store.List(ctx, taskID, p)
	if err != nil {
		return domain.Page[domain.Comment]{}, fmt.Errorf("list comments: %w", err)
	}
	if items == nil {
		items = []domain.Comment{}
	}
	return domain.Page[domain.Comment]{Items: items, Page: p.Page, PageSize: p.PageSize, Total: total}, nil
}

// Create viết bình luận và báo cho người nhận + người tạo công việc (trừ chính người viết).
// Hai thao tác chung một transaction để không có thông báo trỏ tới bình luận không tồn tại.
func (s *Service) Create(ctx context.Context, actor authz.Principal, taskID uuid.UUID, body string) (domain.Comment, error) {
	t, err := s.view(ctx, actor, taskID)
	if err != nil {
		return domain.Comment{}, err
	}
	if err := domain.CanComment(actor.Role); err != nil {
		return domain.Comment{}, err
	}
	b, err := domain.NormalizeComment(body)
	if err != nil {
		return domain.Comment{}, err
	}
	var out domain.Comment
	err = s.tx.InTx(ctx, func(st Store) error {
		c, err := st.Create(ctx, t.ID, actor.UserID, b)
		if err != nil {
			return fmt.Errorf("create comment: %w", err)
		}
		out = c
		createdBy := t.CreatedBy
		assignee, err := s.currentAssignee(ctx, st, t)
		if err != nil {
			return err
		}
		n := domain.NewNotification{
			UserIDs: domain.NotificationRecipients(actor.UserID, assignee, &createdBy),
			Type:    domain.NotificationTaskCommented, TaskID: t.ID, ProjectID: t.ProjectID,
			ActorID: actor.UserID, Message: domain.NotificationExcerpt(b),
		}
		if err := st.AddNotification(ctx, n); err != nil {
			return fmt.Errorf("notify comment: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return out, nil
}

// currentAssignee trả người đang chịu trách nhiệm công việc. Với việc con, quyền đi theo
// người nhận của cha (decisions.md mục C) — người nhận chép vào con lúc tạo có thể đã cũ
// nếu cha được giao lại, nên báo cho người nhận hiện tại của cha.
func (s *Service) currentAssignee(ctx context.Context, st Store, t domain.Task) (*uuid.UUID, error) {
	if t.ParentID == nil {
		return t.AssigneeID, nil
	}
	parent, err := st.GetTask(ctx, *t.ParentID)
	if err != nil {
		return nil, fmt.Errorf("get parent task: %w", err)
	}
	return parent.AssigneeID, nil
}
