package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/subtask"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// CF-07: việc con, bình luận, thông báo. Tầng này chỉ giải mã/định dạng —
// quyền (thành viên dự án, người nhận việc cha, chủ thông báo) kiểm trong service.

// SubtaskService là port tầng HTTP cần cho việc con.
type SubtaskService interface {
	List(ctx context.Context, actor authz.Principal, parentID uuid.UUID) ([]domain.Task, error)
	Create(ctx context.Context, actor authz.Principal, parentID uuid.UUID, in subtask.Input) (domain.Task, error)
	Move(ctx context.Context, actor authz.Principal, id, columnID uuid.UUID) (domain.Task, error)
	Delete(ctx context.Context, actor authz.Principal, id uuid.UUID) error
}

// CommentService là port tầng HTTP cần cho bình luận.
type CommentService interface {
	List(ctx context.Context, actor authz.Principal, taskID uuid.UUID, p domain.PageRequest) (domain.Page[domain.Comment], error)
	Create(ctx context.Context, actor authz.Principal, taskID uuid.UUID, body string) (domain.Comment, error)
}

// NotificationService là port tầng HTTP cần cho chuông thông báo.
type NotificationService interface {
	List(ctx context.Context, actor authz.Principal, unreadOnly bool, p domain.PageRequest) (domain.Page[domain.Notification], error)
	UnreadCount(ctx context.Context, actor authz.Principal) (int, error)
	MarkRead(ctx context.Context, actor authz.Principal, id uuid.UUID) error
	MarkAllRead(ctx context.Context, actor authz.Principal) (int, error)
}

var (
	errSubtaskIDInvalid      = errBadRequest("SUBTASK_ID_INVALID", "Mã công việc con không hợp lệ")
	errNotificationIDInvalid = errBadRequest("NOTIFICATION_ID_INVALID", "Mã thông báo không hợp lệ")
	errUnreadInvalid         = errBadRequest("QUERY_INVALID", "Tham số unread phải là true hoặc false")
)

type collabHandler struct {
	subtasks      SubtaskService
	comments      CommentService
	notifications NotificationService
	validate      *validator.Validate
}

// subtaskRequest cố ý KHÔNG có storyPoint: R04 — Story Point chỉ ở việc cha.
// decodeAndValidate từ chối trường lạ nên client gửi storyPoint sẽ nhận 400.
type subtaskRequest struct {
	Title       string `json:"title" validate:"required,max=200"`
	Description string `json:"description" validate:"max=5000"`
}

type subtaskMoveRequest struct {
	ColumnID uuid.UUID `json:"columnId" validate:"required"`
}

type commentRequest struct {
	Body string `json:"body"` // độ dài/khoảng trắng kiểm ở domain để có thông báo tiếng Việt thống nhất
}

type commentResponse struct {
	ID        string      `json:"id"`
	TaskID    string      `json:"taskId"`
	Author    refResponse `json:"author"`
	Body      string      `json:"body"`
	CreatedAt time.Time   `json:"createdAt"`
}

type notificationResponse struct {
	ID        string       `json:"id"`
	Type      string       `json:"type"`
	Task      refResponse  `json:"task"`
	Project   refResponse  `json:"project"`
	Actor     *refResponse `json:"actor"`
	Message   string       `json:"message"`
	Read      bool         `json:"read"`
	CreatedAt time.Time    `json:"createdAt"`
}

type countBody struct {
	Count int `json:"count"`
}

func toCommentResponse(c domain.Comment) commentResponse {
	return commentResponse{
		ID: c.ID.String(), TaskID: c.TaskID.String(),
		Author: refResponse{ID: c.AuthorID.String(), Name: c.AuthorName},
		Body:   c.Body, CreatedAt: c.CreatedAt,
	}
}

func toNotificationResponse(n domain.Notification) notificationResponse {
	return notificationResponse{
		ID: n.ID.String(), Type: string(n.Type),
		Task:    refResponse{ID: n.TaskID.String(), Name: n.TaskTitle},
		Project: refResponse{ID: n.ProjectID.String(), Name: n.ProjectName},
		Actor:   toRef(n.ActorID, n.ActorName),
		Message: n.Message, Read: n.ReadAt != nil, CreatedAt: n.CreatedAt,
	}
}

// --- Việc con ---

func (h *collabHandler) listSubtasks(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	parentID, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items, err := h.subtasks.List(r.Context(), actor, parentID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	out := make([]taskResponse, 0, len(items))
	for _, t := range items {
		out = append(out, toTaskResponse(t))
	}
	writeJSON(w, http.StatusOK, itemsResponse[taskResponse]{Items: out})
}

func (h *collabHandler) createSubtask(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	parentID, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req subtaskRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	t, err := h.subtasks.Create(r.Context(), actor, parentID, subtask.Input{Title: req.Title, Description: req.Description})
	writeTask(w, r, http.StatusCreated, t, err)
}

// moveSubtask: việc con KHÔNG đi qua luật workflow_transition (R05) — service chỉ
// kiểm cột thuộc bảng của dự án và người gọi là người nhận việc cha hoặc Manager.
func (h *collabHandler) moveSubtask(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "subtaskID", errSubtaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req subtaskMoveRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	t, err := h.subtasks.Move(r.Context(), actor, id, req.ColumnID)
	writeTask(w, r, http.StatusOK, t, err)
}

func (h *collabHandler) deleteSubtask(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "subtaskID", errSubtaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.subtasks.Delete(r.Context(), actor, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Bình luận ---

func (h *collabHandler) listComments(w http.ResponseWriter, r *http.Request) {
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
	res, err := h.comments.List(r.Context(), actor, taskID, p)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]commentResponse, 0, len(res.Items))
	for _, c := range res.Items {
		items = append(items, toCommentResponse(c))
	}
	writeJSON(w, http.StatusOK, pageResponse[commentResponse]{Items: items, Page: res.Page, PageSize: res.PageSize, Total: res.Total})
}

func (h *collabHandler) createComment(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	taskID, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req commentRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := h.comments.Create(r.Context(), actor, taskID, req.Body)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCommentResponse(c))
}

// --- Thông báo (luôn của chính người gọi; không có tham số userId) ---

func (h *collabHandler) listNotifications(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	p, err := pageFrom(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	unread := false
	switch r.URL.Query().Get("unread") {
	case "", "false":
	case "true":
		unread = true
	default:
		writeError(w, r, errUnreadInvalid)
		return
	}
	res, err := h.notifications.List(r.Context(), actor, unread, p)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]notificationResponse, 0, len(res.Items))
	for _, n := range res.Items {
		items = append(items, toNotificationResponse(n))
	}
	writeJSON(w, http.StatusOK, pageResponse[notificationResponse]{Items: items, Page: res.Page, PageSize: res.PageSize, Total: res.Total})
}

func (h *collabHandler) unreadCount(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	n, err := h.notifications.UnreadCount(r.Context(), actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, countBody{Count: n})
}

func (h *collabHandler) markRead(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "notificationID", errNotificationIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.notifications.MarkRead(r.Context(), actor, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *collabHandler) markAllRead(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	n, err := h.notifications.MarkAllRead(r.Context(), actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, countBody{Count: n})
}

func uuidString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}
