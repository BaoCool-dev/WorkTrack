package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// ProjectService là các use case dự án CF-03 (cài đặt: project.Service).
// Quyền (vai trò + Manager phụ trách) được kiểm trong service — router chỉ yêu cầu đăng nhập.
type ProjectService interface {
	Create(ctx context.Context, actor authz.Principal, in project.Input) (domain.Project, error)
	List(ctx context.Context, actor authz.Principal, in project.ListInput, p domain.PageRequest) (domain.Page[domain.Project], error)
	Get(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error)
	Update(ctx context.Context, actor authz.Principal, id uuid.UUID, in project.Input) (domain.Project, error)
	ListMembers(ctx context.Context, actor authz.Principal, id uuid.UUID) ([]domain.ProjectMember, error)
	ListCandidates(ctx context.Context, actor authz.Principal, id uuid.UUID, search string) ([]domain.MemberCandidate, error)
	AddMember(ctx context.Context, actor authz.Principal, id, userID uuid.UUID) error
	RemoveMember(ctx context.Context, actor authz.Principal, id, userID uuid.UUID) error
	GetBoard(ctx context.Context, actor authz.Principal, id uuid.UUID) (project.Board, error)
	AddColumn(ctx context.Context, actor authz.Principal, id uuid.UUID, name, group string) (project.Board, error)
	UpdateColumn(ctx context.Context, actor authz.Principal, id, columnID uuid.UUID, name, group string) (project.Board, error)
	DeleteColumn(ctx context.Context, actor authz.Principal, id, columnID uuid.UUID) (project.Board, error)
	ReorderColumns(ctx context.Context, actor authz.Principal, id uuid.UUID, ids []uuid.UUID) (project.Board, error)
	ReplaceTransitions(ctx context.Context, actor authz.Principal, id uuid.UUID, ts []domain.Transition) (project.Board, error)
}

type projectHandler struct {
	svc      ProjectService
	validate *validator.Validate
}

// dateLayout: ngày bắt đầu/kết thúc dự án trao đổi dạng "YYYY-MM-DD" (không giờ, không múi giờ).
const dateLayout = time.DateOnly

var (
	errProjectIDInvalid = errBadRequest("PROJECT_ID_INVALID", "Mã dự án không hợp lệ")
	errColumnIDInvalid  = errBadRequest("COLUMN_ID_INVALID", "Mã cột không hợp lệ")
	errDateInvalid      = errBadRequest("DATE_INVALID", "Ngày phải có dạng YYYY-MM-DD")
)

// Độ dài tối đa thật sự được domain kiểm (thông điệp rõ ràng, 422); validator chỉ chặn dữ liệu quá lớn.
type projectRequest struct {
	Name        string  `json:"name" validate:"max=1000"`
	Description string  `json:"description" validate:"max=10000"`
	Status      string  `json:"status" validate:"max=20"`
	StartDate   *string `json:"startDate"`
	EndDate     *string `json:"endDate"`
}

type addMemberRequest struct {
	UserID uuid.UUID `json:"userId" validate:"required"`
}

type columnRequest struct {
	Name        string `json:"name" validate:"max=500"`
	StatusGroup string `json:"statusGroup" validate:"max=20"`
}

type columnOrderRequest struct {
	ColumnIDs []uuid.UUID `json:"columnIds" validate:"required,max=50"`
}

type transitionDTO struct {
	FromColumnID uuid.UUID             `json:"fromColumnId" validate:"required"`
	ToColumnID   uuid.UUID             `json:"toColumnId" validate:"required"`
	Role         domain.TransitionRole `json:"role" validate:"required"`
}

// 8 cột × 7 cột đích × 2 vai trò = 112 luật là tối đa có nghĩa.
type transitionsRequest struct {
	Transitions []transitionDTO `json:"transitions" validate:"max=112,dive"`
}

type projectResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	Status      domain.ProjectStatus `json:"status"`
	StartDate   *string              `json:"startDate"`
	EndDate     *string              `json:"endDate"`
	Manager     refResponse          `json:"manager"`
	MemberCount int                  `json:"memberCount"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

type projectMemberResponse struct {
	UserID   string            `json:"userId"`
	FullName string            `json:"fullName"`
	Email    string            `json:"email"`
	Role     domain.Role       `json:"role"`
	Status   domain.UserStatus `json:"status"`
	AddedAt  *time.Time        `json:"addedAt,omitempty"`
}

type columnResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Position    int                `json:"position"`
	StatusGroup domain.StatusGroup `json:"statusGroup"`
	TaskCount   int                `json:"taskCount"`
}

type boardResponse struct {
	Columns     []columnResponse `json:"columns"`
	Transitions []transitionDTO  `json:"transitions"`
}

type itemsResponse[T any] struct {
	Items []T `json:"items"`
}

func formatDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(dateLayout)
	return &s
}

func parseDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil, errDateInvalid
	}
	return &t, nil
}

func toProjectResponse(p domain.Project) projectResponse {
	return projectResponse{
		ID: p.ID.String(), Name: p.Name, Description: p.Description, Status: p.Status,
		StartDate: formatDate(p.StartDate), EndDate: formatDate(p.EndDate),
		Manager:     refResponse{ID: p.ManagerID.String(), Name: p.ManagerName},
		MemberCount: p.MemberCount, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toBoardResponse(b project.Board) boardResponse {
	cols := make([]columnResponse, 0, len(b.Columns))
	for _, c := range b.Columns {
		cols = append(cols, columnResponse{
			ID: c.ID.String(), Name: c.Name, Position: c.Position, StatusGroup: c.StatusGroup, TaskCount: c.TaskCount,
		})
	}
	ts := make([]transitionDTO, 0, len(b.Transitions))
	for _, t := range b.Transitions {
		ts = append(ts, transitionDTO{FromColumnID: t.FromColumnID, ToColumnID: t.ToColumnID, Role: t.Role})
	}
	return boardResponse{Columns: cols, Transitions: ts}
}

func pathUUID(r *http.Request, key string, bad *httpError) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		return uuid.Nil, bad
	}
	return id, nil
}

func (h *projectHandler) decodeProject(w http.ResponseWriter, r *http.Request) (project.Input, error) {
	var req projectRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		return project.Input{}, err
	}
	start, err := parseDate(req.StartDate)
	if err != nil {
		return project.Input{}, err
	}
	end, err := parseDate(req.EndDate)
	if err != nil {
		return project.Input{}, err
	}
	return project.Input{Name: req.Name, Description: req.Description, Status: req.Status, StartDate: start, EndDate: end}, nil
}

func (h *projectHandler) list(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	page, err := queryInt(r, "page", 1)
	if err != nil {
		writeError(w, r, err)
		return
	}
	size, err := queryInt(r, "pageSize", domain.DefaultPageSize)
	if err != nil {
		writeError(w, r, err)
		return
	}
	q := r.URL.Query()
	res, err := h.svc.List(r.Context(), actor, project.ListInput{Search: q.Get("search"), Status: q.Get("status")},
		domain.NewPageRequest(page, size))
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]projectResponse, 0, len(res.Items))
	for _, p := range res.Items {
		items = append(items, toProjectResponse(p))
	}
	writeJSON(w, http.StatusOK, pageResponse[projectResponse]{
		Items: items, Page: res.Page, PageSize: res.PageSize, Total: res.Total,
	})
}

func (h *projectHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	in, err := h.decodeProject(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	p, err := h.svc.Create(r.Context(), actor, in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectResponse(p))
}

func (h *projectHandler) get(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	p, err := h.svc.Get(r.Context(), actor, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(p))
}

func (h *projectHandler) update(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	in, err := h.decodeProject(w, r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	p, err := h.svc.Update(r.Context(), actor, id, in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(p))
}

func (h *projectHandler) listMembers(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	ms, err := h.svc.ListMembers(r.Context(), actor, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]projectMemberResponse, 0, len(ms))
	for _, m := range ms {
		added := m.AddedAt
		items = append(items, projectMemberResponse{
			UserID: m.UserID.String(), FullName: m.FullName, Email: m.Email, Role: m.Role, Status: m.Status, AddedAt: &added,
		})
	}
	writeJSON(w, http.StatusOK, itemsResponse[projectMemberResponse]{Items: items})
}

func (h *projectHandler) listCandidates(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	cs, err := h.svc.ListCandidates(r.Context(), actor, id, r.URL.Query().Get("search"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]projectMemberResponse, 0, len(cs))
	for _, c := range cs {
		items = append(items, projectMemberResponse{
			UserID: c.UserID.String(), FullName: c.FullName, Email: c.Email, Role: c.Role, Status: c.Status,
		})
	}
	writeJSON(w, http.StatusOK, itemsResponse[projectMemberResponse]{Items: items})
}

func (h *projectHandler) addMember(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req addMemberRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.svc.AddMember(r.Context(), actor, id, req.UserID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *projectHandler) removeMember(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	userID, err := pathUUID(r, "userID", errUserIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := h.svc.RemoveMember(r.Context(), actor, id, userID); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeBoard trả bảng sau thao tác (hoặc lỗi).
func writeBoard(w http.ResponseWriter, r *http.Request, status int, b project.Board, err error) {
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, status, toBoardResponse(b))
}

func (h *projectHandler) getBoard(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	b, err := h.svc.GetBoard(r.Context(), actor, id)
	writeBoard(w, r, http.StatusOK, b, err)
}

func (h *projectHandler) addColumn(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req columnRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	b, err := h.svc.AddColumn(r.Context(), actor, id, req.Name, req.StatusGroup)
	writeBoard(w, r, http.StatusCreated, b, err)
}

func (h *projectHandler) updateColumn(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	colID, err := pathUUID(r, "columnID", errColumnIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req columnRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	b, err := h.svc.UpdateColumn(r.Context(), actor, id, colID, req.Name, req.StatusGroup)
	writeBoard(w, r, http.StatusOK, b, err)
}

func (h *projectHandler) deleteColumn(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	colID, err := pathUUID(r, "columnID", errColumnIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	b, err := h.svc.DeleteColumn(r.Context(), actor, id, colID)
	writeBoard(w, r, http.StatusOK, b, err)
}

func (h *projectHandler) reorderColumns(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req columnOrderRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	b, err := h.svc.ReorderColumns(r.Context(), actor, id, req.ColumnIDs)
	writeBoard(w, r, http.StatusOK, b, err)
}

func (h *projectHandler) replaceTransitions(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req transitionsRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	ts := make([]domain.Transition, 0, len(req.Transitions))
	for _, t := range req.Transitions {
		ts = append(ts, domain.Transition{FromColumnID: t.FromColumnID, ToColumnID: t.ToColumnID, Role: t.Role})
	}
	b, err := h.svc.ReplaceTransitions(r.Context(), actor, id, ts)
	writeBoard(w, r, http.StatusOK, b, err)
}
