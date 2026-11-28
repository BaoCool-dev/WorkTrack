package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// ReviewService là use case nghiệm thu công việc cha (CF-09; cài đặt: kanban.ReviewService).
// Quyền Manager phụ trách, cột "Chờ nghiệm thu", luật chuyển cột và version đều kiểm trong service.
type ReviewService interface {
	Approve(ctx context.Context, actor authz.Principal, id uuid.UUID, in kanban.ReviewInput) (domain.Task, error)
	Reject(ctx context.Context, actor authz.Principal, id uuid.UUID, in kanban.ReviewInput) (domain.Task, error)
	Reviews(ctx context.Context, actor authz.Principal, id uuid.UUID) ([]domain.TaskReview, error)
}

type reviewHandler struct {
	svc      ReviewService
	validate *validator.Validate
}

// reviewRequest: version thẻ client đang thấy (lệch → 409). reason chỉ dùng khi Từ chối;
// độ dài 1–500 sau khi cắt khoảng trắng do domain kiểm (422 REVIEW_REASON_INVALID).
type reviewRequest struct {
	Version *int   `json:"version" validate:"required,min=1"`
	Reason  string `json:"reason" validate:"max=2000"`
}

type reviewResponse struct {
	ID           string    `json:"id"`
	ReviewerID   string    `json:"reviewerId"`
	ReviewerName string    `json:"reviewerName"`
	Decision     string    `json:"decision"`
	Reason       *string   `json:"reason"`
	CreatedAt    time.Time `json:"createdAt"`
}

func toReviewResponse(rv domain.TaskReview) reviewResponse {
	return reviewResponse{
		ID: rv.ID.String(), ReviewerID: rv.ReviewerID.String(), ReviewerName: rv.ReviewerName,
		Decision: string(rv.Decision), Reason: rv.Reason, CreatedAt: rv.CreatedAt,
	}
}

type reviewAction func(ctx context.Context, actor authz.Principal, id uuid.UUID, in kanban.ReviewInput) (domain.Task, error)

func (h *reviewHandler) approve(w http.ResponseWriter, r *http.Request) {
	h.decide(w, r, h.svc.Approve)
}
func (h *reviewHandler) reject(w http.ResponseWriter, r *http.Request) { h.decide(w, r, h.svc.Reject) }

// decide đọc chung id + body cho Duyệt và Từ chối; trả công việc sau khi chuyển cột.
func (h *reviewHandler) decide(w http.ResponseWriter, r *http.Request, act reviewAction) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req reviewRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	t, err := act(r.Context(), actor, id, kanban.ReviewInput{Version: *req.Version, Reason: req.Reason})
	writeTask(w, r, http.StatusOK, t, err)
}

// list trả lịch sử nghiệm thu của công việc, mới nhất trước (ai xem được công việc thì xem được).
func (h *reviewHandler) list(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	rs, err := h.svc.Reviews(r.Context(), actor, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]reviewResponse, 0, len(rs))
	for _, rv := range rs {
		items = append(items, toReviewResponse(rv))
	}
	writeJSON(w, http.StatusOK, itemsResponse[reviewResponse]{Items: items})
}
