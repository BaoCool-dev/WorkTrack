package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// TaskService là các use case công việc cha CF-06 (cài đặt: task.Service). Quyền kiểm trong service.
// Chuyển cột/đổi thứ tự thuộc KanbanService (CF-04).
type TaskService interface {
	List(ctx context.Context, actor authz.Principal, projectID uuid.UUID, in task.ListInput, p domain.PageRequest) (domain.Page[domain.Task], error)
	ListMine(ctx context.Context, actor authz.Principal, in task.MyListInput, p domain.PageRequest) (domain.Page[domain.Task], error)
	MyWIP(ctx context.Context, actor authz.Principal) (int, error)
	Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Task, error)
	Create(ctx context.Context, actor authz.Principal, projectID uuid.UUID, in task.Input) (domain.Task, error)
	Update(ctx context.Context, actor authz.Principal, id uuid.UUID, in task.Input) (domain.Task, error)
	Claim(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Task, error)
}

// LabelService là danh mục nhãn (cài đặt: label.Service).
type LabelService interface {
	List(ctx context.Context, actor authz.Principal) ([]domain.Label, error)
	Create(ctx context.Context, actor authz.Principal, name string) (domain.Label, error)
	Update(ctx context.Context, actor authz.Principal, id uuid.UUID, name string) (domain.Label, error)
	Delete(ctx context.Context, actor authz.Principal, id uuid.UUID) error
}

// SettingService là tham số hệ thống B1 (cài đặt: setting.Service).
type SettingService interface {
	Get(ctx context.Context, actor authz.Principal) (setting.Settings, error)
	Update(ctx context.Context, actor authz.Principal, hours int) (setting.Settings, error)
}

type taskHandler struct {
	svc      TaskService
	labels   LabelService
	settings SettingService
	validate *validator.Validate
}

var (
	errTaskIDInvalid  = errBadRequest("TASK_ID_INVALID", "Mã công việc không hợp lệ")
	errLabelIDInvalid = errBadRequest("LABEL_ID_INVALID", "Mã nhãn không hợp lệ")
)

// Độ dài thật do domain kiểm (422); validator chỉ chặn dữ liệu quá lớn.
type taskRequest struct {
	Title       string      `json:"title" validate:"max=2000"`
	Description string      `json:"description" validate:"max=20000"`
	StoryPoint  *int        `json:"storyPoint"`
	AssigneeID  *uuid.UUID  `json:"assigneeId"`
	DueDate     *string     `json:"dueDate"`
	LabelIDs    []uuid.UUID `json:"labelIds" validate:"max=100"`
}

type labelRequest struct {
	Name string `json:"name" validate:"max=1000"`
}

type settingsRequest struct {
	StoryPointHours int `json:"storyPointHours"`
}

type taskColumnResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	StatusGroup domain.StatusGroup `json:"statusGroup"`
}

type taskResponse struct {
	ID               string             `json:"id"`
	ProjectID        string             `json:"projectId"`
	ProjectName      string             `json:"projectName"`
	ParentID         *string            `json:"parentId"` // CF-07: khác null → đây là việc con
	Title            string             `json:"title"`
	Description      string             `json:"description"`
	StoryPoint       *int               `json:"storyPoint"`
	EstimatedMinutes *int               `json:"estimatedMinutes"`
	Column           taskColumnResponse `json:"column"`
	Assignee         *refResponse       `json:"assignee"`
	DueDate          *string            `json:"dueDate"`
	Labels           []refResponse      `json:"labels"`
	CreatedBy        refResponse        `json:"createdBy"`
	Position         int                `json:"position"`
	Version          int                `json:"version"` // client gửi lại khi kéo thả để chống ghi đè
	// CF-08: giờ đã log trên chính công việc và giờ thực tế (cha + con, R: báo cáo cộng giờ con lên cha), đơn vị phút.
	LoggedMinutes    int       `json:"loggedMinutes"`
	ActualMinutes    int       `json:"actualMinutes"`
	SubtaskCount     int       `json:"subtaskCount"`
	SubtaskDoneCount int       `json:"subtaskDoneCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type labelResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	TaskCount int    `json:"taskCount"`
}

type settingsResponse struct {
	StoryPointHours int       `json:"storyPointHours"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type wipResponse struct {
	Count int `json:"count"`
}

func toTaskResponse(t domain.Task) taskResponse {
	labels := make([]refResponse, 0, len(t.Labels))
	for _, l := range t.Labels {
		labels = append(labels, refResponse{ID: l.ID.String(), Name: l.Name})
	}
	return taskResponse{
		ID: t.ID.String(), ProjectID: t.ProjectID.String(), ProjectName: t.ProjectName, ParentID: uuidString(t.ParentID),
		Title: t.Title, Description: t.Description, StoryPoint: t.StoryPoint, EstimatedMinutes: t.EstimatedMinutes,
		Column:   taskColumnResponse{ID: t.ColumnID.String(), Name: t.ColumnName, StatusGroup: t.StatusGroup},
		Assignee: toRef(t.AssigneeID, t.AssigneeName), DueDate: formatDate(t.DueDate), Labels: labels,
		CreatedBy: refResponse{ID: t.CreatedBy.String(), Name: t.CreatedByName},
		Position:  t.Position, Version: t.Version,
		LoggedMinutes: t.LoggedMinutes, ActualMinutes: t.ActualMinutes,
		SubtaskCount: t.SubtaskCount, SubtaskDoneCount: t.SubtaskDoneCount,
		CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt,
	}
}

func writeTaskPage(w http.ResponseWriter, r *http.Request, res domain.Page[domain.Task], err error) {
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]taskResponse, 0, len(res.Items))
	for _, t := range res.Items {
		items = append(items, toTaskResponse(t))
	}
	writeJSON(w, http.StatusOK, pageResponse[taskResponse]{Items: items, Page: res.Page, PageSize: res.PageSize, Total: res.Total})
}

func writeTask(w http.ResponseWriter, r *http.Request, status int, t domain.Task, err error) {
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, status, toTaskResponse(t))
}

func pageFrom(r *http.Request) (domain.PageRequest, error) {
	page, err := queryInt(r, "page", 1)
	if err != nil {
		return domain.PageRequest{}, err
	}
	size, err := queryInt(r, "pageSize", domain.DefaultPageSize)
	if err != nil {
		return domain.PageRequest{}, err
	}
	return domain.NewPageRequest(page, size), nil
}

func (h *taskHandler) decodeTask(w http.ResponseWriter, r *http.Request) (task.Input, error) {
	var req taskRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		return task.Input{}, err
	}
	due, err := parseDate(req.DueDate)
	if err != nil {
		return task.Input{}, err
	}
	return task.Input{
		Title: req.Title, Description: req.Description, StoryPoint: req.StoryPoint,
		AssigneeID: req.AssigneeID, DueDate: due, LabelIDs: req.LabelIDs,
	}, nil
}

func (h *taskHandler) listProjectTasks(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	pid, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	p, err := pageFrom(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	in := task.ListInput{Search: r.URL.Query().Get("search"), StatusGroup: r.URL.Query().Get("statusGroup"),
		Unassigned: r.URL.Query().Get("unassigned") == "true"}
	for key, dst := range map[string]**uuid.UUID{"columnId": &in.ColumnID, "assigneeId": &in.AssigneeID, "labelId": &in.LabelID} {
		if *dst, err = queryUUID(r, key); err != nil {
			writeError(w, r, err)
			return
		}
	}
	res, err := h.svc.List(r.Context(), actor, pid, in, p)
	writeTaskPage(w, r, res, err)
}

func (h *taskHandler) createTask(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	pid, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	in, err := h.decodeTask(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	t, err := h.svc.Create(r.Context(), actor, pid, in)
	writeTask(w, r, http.StatusCreated, t, err)
}

func (h *taskHandler) getTask(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	t, err := h.svc.Get(r.Context(), actor, id)
	writeTask(w, r, http.StatusOK, t, err)
}

func (h *taskHandler) updateTask(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	in, err := h.decodeTask(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	t, err := h.svc.Update(r.Context(), actor, id, in)
	writeTask(w, r, http.StatusOK, t, err)
}

func (h *taskHandler) claimTask(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "taskID", errTaskIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	t, err := h.svc.Claim(r.Context(), actor, id)
	writeTask(w, r, http.StatusOK, t, err)
}

func (h *taskHandler) listMyTasks(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	p, err := pageFrom(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	q := r.URL.Query()
	res, err := h.svc.ListMine(r.Context(), actor, task.MyListInput{Search: q.Get("search"), StatusGroup: q.Get("statusGroup")}, p)
	writeTaskPage(w, r, res, err)
}

func (h *taskHandler) myWIP(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	n, err := h.svc.MyWIP(r.Context(), actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, wipResponse{Count: n})
}

func toLabelResponse(l domain.Label) labelResponse {
	return labelResponse{ID: l.ID.String(), Name: l.Name, TaskCount: l.TaskCount}
}

func (h *taskHandler) listLabels(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	ls, err := h.labels.List(r.Context(), actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]labelResponse, 0, len(ls))
	for _, l := range ls {
		items = append(items, toLabelResponse(l))
	}
	writeJSON(w, http.StatusOK, itemsResponse[labelResponse]{Items: items})
}

func (h *taskHandler) createLabel(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	var req labelRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	l, err := h.labels.Create(r.Context(), actor, req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toLabelResponse(l))
}

func (h *taskHandler) updateLabel(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "labelID", errLabelIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req labelRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	l, err := h.labels.Update(r.Context(), actor, id, req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toLabelResponse(l))
}

func (h *taskHandler) deleteLabel(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "labelID", errLabelIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.labels.Delete(r.Context(), actor, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeSettings(w http.ResponseWriter, r *http.Request, s setting.Settings, err error) {
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, settingsResponse{StoryPointHours: s.HoursPerStoryPoint, UpdatedAt: s.UpdatedAt})
}

func (h *taskHandler) getSettings(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	s, err := h.settings.Get(r.Context(), actor)
	writeSettings(w, r, s, err)
}

func (h *taskHandler) updateSettings(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	var req settingsRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	s, err := h.settings.Update(r.Context(), actor, req.StoryPointHours)
	writeSettings(w, r, s, err)
}
