package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/timelog"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// CF-08: Log time trên công việc cha và con. Tầng này chỉ giải mã/định dạng —
// quyền (Manager dự án hoặc người nhận việc gốc), phút > 0, luật Done quá 24 giờ kiểm trong service/domain.

// TimeLogService là port tầng HTTP cần cho Log time.
type TimeLogService interface {
	List(ctx context.Context, actor authz.Principal, taskID uuid.UUID, p domain.PageRequest) (domain.Page[domain.TimeLog], error)
	Create(ctx context.Context, actor authz.Principal, taskID uuid.UUID, in timelog.Input) (domain.TimeLog, error)
	Delete(ctx context.Context, actor authz.Principal, logID uuid.UUID) error
}

var errTimeLogIDInvalid = errBadRequest("TIME_LOG_ID_INVALID", "Mã log giờ không hợp lệ")

type timeLogHandler struct {
	svc      TimeLogService
	validate *validator.Validate
}

// timeLogRequest không có userId: luôn ghi cho chính người gọi (Member không log hộ người khác).
// Phút và ghi chú kiểm ở domain để thông báo tiếng Việt thống nhất.
type timeLogRequest struct {
	Minutes  int     `json:"minutes"`
	Note     string  `json:"note"`
	WorkDate *string `json:"workDate"` // YYYY-MM-DD theo giờ Việt Nam; bỏ trống → hôm nay
}

type timeLogResponse struct {
	ID        string      `json:"id"`
	TaskID    string      `json:"taskId"`
	TaskTitle string      `json:"taskTitle"`
	ParentID  *string     `json:"parentId"`
	User      refResponse `json:"user"`
	Minutes   int         `json:"minutes"`
	Note      string      `json:"note"`
	WorkDate  string      `json:"workDate"`
	CreatedAt time.Time   `json:"createdAt"`
}

func toTimeLogResponse(l domain.TimeLog) timeLogResponse {
	var parent *string
	if l.ParentID != nil {
		s := l.ParentID.String()
		parent = &s
	}
	return timeLogResponse{
		ID: l.ID.String(), TaskID: l.TaskID.String(), TaskTitle: l.TaskTitle, ParentID: parent,
		User:    refResponse{ID: l.UserID.String(), Name: l.UserName},
		Minutes: l.Minutes, Note: l.Note, WorkDate: l.WorkDate.Format(dateLayout), CreatedAt: l.CreatedAt,
	}
}

// list: lịch sử log của công việc; với việc cha gồm cả log của các việc con.
func (h *timeLogHandler) list(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	taskID, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	p, err := pageFrom(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.svc.List(r.Context(), actor, taskID, p)
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]timeLogResponse, 0, len(res.Items))
	for _, l := range res.Items {
		out = append(out, toTimeLogResponse(l))
	}
	writeJSON(w, http.StatusOK, pageResponse[timeLogResponse]{Items: out, Page: res.Page, PageSize: res.PageSize, Total: res.Total})
}

func (h *timeLogHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	taskID, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req timeLogRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	day, err := parseDate(req.WorkDate)
	if err != nil {
		writeError(w, r, err)
		return
	}
	l, err := h.svc.Create(r.Context(), actor, taskID, timelog.Input{Minutes: req.Minutes, Note: req.Note, WorkDate: day})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTimeLogResponse(l))
}

func (h *timeLogHandler) delete(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "logID", errTimeLogIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.svc.Delete(r.Context(), actor, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
