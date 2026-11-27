package httpapi

import (
	"context"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// KanbanService là use case kéo thả trên bảng (CF-04, CF-05; cài đặt: kanban.Service).
// Quyền (Manager phụ trách / người nhận việc), luật chuyển cột, WIP và version đều kiểm trong service.
type KanbanService interface {
	Board(ctx context.Context, actor authz.Principal, projectID uuid.UUID) ([]domain.Task, error)
	Move(ctx context.Context, actor authz.Principal, id uuid.UUID, in kanban.MoveInput) (domain.Task, error)
}

type kanbanHandler struct {
	svc      KanbanService
	projects ProjectService
	validate *validator.Validate
}

// moveRequest: thả thẻ vào cột columnId ở vị trí position (0 = đầu cột).
// version là version thẻ client đang thấy — lệch thì server trả 409 thay vì ghi đè.
type moveRequest struct {
	ColumnID uuid.UUID `json:"columnId" validate:"required"`
	Position *int      `json:"position" validate:"required,min=0,max=10000"`
	Version  *int      `json:"version" validate:"required,min=1"`
}

type wipLimitRequest struct {
	WIPLimit int `json:"wipLimit"` // khoảng 1–20 do domain kiểm (422 WIP_LIMIT_INVALID)
}

// boardTasks trả mọi công việc cha của bảng (không phân trang: bảng cần đủ thẻ để kéo thả).
func (h *kanbanHandler) boardTasks(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	pid, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	tasks, err := h.svc.Board(r.Context(), actor, pid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		items = append(items, toTaskResponse(t))
	}
	writeJSON(w, http.StatusOK, itemsResponse[taskResponse]{Items: items})
}

// move chuyển cột hoặc đổi thứ tự một thẻ.
func (h *kanbanHandler) move(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req moveRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	t, err := h.svc.Move(r.Context(), actor, id, kanban.MoveInput{
		ColumnID: req.ColumnID, Position: *req.Position, Version: *req.Version,
	})
	writeTask(w, r, http.StatusOK, t, err)
}

// setWIPLimit: Manager phụ trách đổi giới hạn WIP của dự án (CF-05).
func (h *kanbanHandler) setWIPLimit(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	pid, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req wipLimitRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	p, err := h.projects.SetWIPLimit(r.Context(), actor, pid, req.WIPLimit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(p))
}
