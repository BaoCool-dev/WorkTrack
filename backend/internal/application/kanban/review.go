package kanban

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// CF-09 — nghiệm thu công việc cha: Manager phụ trách Duyệt (→ nhóm done) hoặc Từ chối có lý do
// (→ cột "đang làm" đầu tiên, tính vào tỷ lệ làm lại). Member đưa việc vào cột "Chờ nghiệm thu"
// bằng thao tác kéo thả bình thường (Service.Move).

// ReviewRecord là một dòng task_review.
type ReviewRecord struct {
	TaskID, ProjectID, ReviewerID uuid.UUID
	Decision                      domain.ReviewDecision
	Reason                        *string
	FromColumnID, ToColumnID      uuid.UUID
}

// ReviewStore mở rộng Store của bảng Kanban với những gì nghiệm thu cần.
type ReviewStore interface {
	Store
	// ListColumns trả các cột của bảng theo vị trí.
	ListColumns(ctx context.Context, projectID uuid.UUID) ([]domain.BoardColumn, error)
	InsertReview(ctx context.Context, r ReviewRecord) error
	// ListReviews trả lịch sử nghiệm thu, mới nhất trước.
	ListReviews(ctx context.Context, taskID uuid.UUID) ([]domain.TaskReview, error)
	AddNotification(ctx context.Context, n domain.NewNotification) error
}

// ReviewTransactor chạy fn trong một transaction; fn lỗi → rollback toàn bộ.
type ReviewTransactor interface {
	InTx(ctx context.Context, fn func(ReviewStore) error) error
}

// ReviewInput: version client đang thấy; Reason chỉ dùng khi Từ chối.
type ReviewInput struct {
	Version int
	Reason  string
}

// ReviewService xử lý Duyệt / Từ chối.
//
// Vì sao dùng lại Service.moveLocked thay vì tự UPDATE cột: nghiệm thu thực chất là một lần chuyển cột
// do Manager làm, nên phải qua đúng các luật của feature/kanban (workflow_transition, nhóm trạng thái,
// WIP, version, lịch sử chuyển cột). Viết lại sẽ có hai bộ luật dễ lệch nhau.
type ReviewService struct {
	board *Service
	store ReviewStore
	tx    ReviewTransactor
}

// NewReviewService tạo ReviewService.
func NewReviewService(store ReviewStore, tx ReviewTransactor, projects Projects, pub Publisher) *ReviewService {
	return &ReviewService{board: &Service{store: store, projects: projects, pub: pub}, store: store, tx: tx}
}

// Approve: Duyệt → cột done đầu tiên.
func (s *ReviewService) Approve(ctx context.Context, actor authz.Principal, id uuid.UUID, in ReviewInput) (domain.Task, error) {
	return s.decide(ctx, actor, id, in.Version, domain.ReviewApproved, nil)
}

// Reject: Từ chối, bắt buộc lý do 1–500 ký tự → về cột in_progress đầu tiên, báo người nhận việc.
func (s *ReviewService) Reject(ctx context.Context, actor authz.Principal, id uuid.UUID, in ReviewInput) (domain.Task, error) {
	return s.decide(ctx, actor, id, in.Version, domain.ReviewRejected, &in.Reason)
}

// Reviews trả lịch sử nghiệm thu cho người xem được công việc.
func (s *ReviewService) Reviews(ctx context.Context, actor authz.Principal, id uuid.UUID) ([]domain.TaskReview, error) {
	if _, _, err := s.board.view(ctx, actor, id); err != nil {
		return nil, err
	}
	rs, err := s.store.ListReviews(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	return rs, nil
}

func (s *ReviewService) decide(ctx context.Context, actor authz.Principal, id uuid.UUID, version int,
	d domain.ReviewDecision, rawReason *string) (domain.Task, error) {
	task, project, err := s.board.view(ctx, actor, id)
	if err != nil {
		return domain.Task{}, err
	}
	// Kiểm sớm: người không có quyền nhận 403, không phải lỗi cột hay version.
	if err := domain.CanReview(actor.UserID, actor.Role, project.ManagerID); err != nil {
		return domain.Task{}, err
	}
	// Kiểm lý do sau quyền: Member gửi lý do rỗng phải nhận 403, không phải 422.
	var reason *string
	if rawReason != nil {
		r, err := domain.NormalizeReviewReason(*rawReason)
		if err != nil {
			return domain.Task{}, err
		}
		reason = &r
	}

	var from, to uuid.UUID
	err = s.tx.InTx(ctx, func(st ReviewStore) error {
		var err error
		from, to, err = s.decideLocked(ctx, st, actor, project, task, version, d, reason)
		return err
	})
	if err != nil {
		return domain.Task{}, err
	}

	moved, err := s.store.Get(ctx, id)
	if err != nil {
		return domain.Task{}, fmt.Errorf("reload task: %w", err)
	}
	s.board.pub.Publish(ctx, domain.BoardEvent{
		Type: domain.EventTaskMoved, ProjectID: project.ID, TaskID: id, ActorID: actor.UserID,
		FromColumnID: from, ToColumnID: to, Position: moved.Position, Version: moved.Version,
	})
	return moved, nil
}

// decideLocked chạy trong transaction: kiểm lại trên dữ liệu đã khóa, chuyển cột, ghi nghiệm thu, báo.
func (s *ReviewService) decideLocked(ctx context.Context, st ReviewStore, actor authz.Principal, p domain.Project,
	task domain.Task, version int, d domain.ReviewDecision, reason *string) (uuid.UUID, uuid.UUID, error) {
	locked, lp, err := lockTask(ctx, st, p.ID, task.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	// Manager phụ trách có thể vừa đổi giữa lúc kiểm sớm và lúc khóa.
	if err := domain.CanReview(actor.UserID, actor.Role, lp.ManagerID); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	cols, err := st.ListColumns(ctx, p.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("list columns: %w", err)
	}
	rc, err := domain.ReviewTargets(cols)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if locked.Version != version {
		return uuid.Nil, uuid.Nil, domain.ErrTaskVersionConflict
	}
	if locked.ColumnID != rc.Review {
		return uuid.Nil, uuid.Nil, domain.ErrTaskNotInReview
	}
	to := rc.Approve
	if d == domain.ReviewRejected {
		to = rc.Reject
	}
	// Position 0: thẻ hiện ở đầu cột đích để người nhận thấy ngay.
	from, err := s.board.moveLocked(ctx, st, actor, p, task.ID, MoveInput{ColumnID: to, Position: 0, Version: version})
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	err = st.InsertReview(ctx, ReviewRecord{
		TaskID: task.ID, ProjectID: p.ID, ReviewerID: actor.UserID, Decision: d, Reason: reason,
		FromColumnID: from, ToColumnID: to,
	})
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("insert review: %w", err)
	}
	if d == domain.ReviewRejected {
		if err := notifyRejected(ctx, st, actor, p.ID, locked, *reason); err != nil {
			return uuid.Nil, uuid.Nil, err
		}
	}
	return from, to, nil
}

// notifyRejected báo người nhận việc và người tạo (trừ chính Manager) — họ cần biết phải làm lại gì.
func notifyRejected(ctx context.Context, st ReviewStore, actor authz.Principal, projectID uuid.UUID,
	t domain.Task, reason string) error {
	ids := domain.NotificationRecipients(actor.UserID, t.AssigneeID, &t.CreatedBy)
	if len(ids) == 0 {
		return nil
	}
	err := st.AddNotification(ctx, domain.NewNotification{
		UserIDs: ids, Type: domain.NotificationReviewRejected, TaskID: t.ID, ProjectID: projectID,
		ActorID: actor.UserID, Message: domain.NotificationExcerpt(reason),
	})
	if err != nil {
		return fmt.Errorf("add notification: %w", err)
	}
	return nil
}
