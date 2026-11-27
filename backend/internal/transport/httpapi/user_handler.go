package httpapi

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// AccountService là các use case quản lý tài khoản (cài đặt: account.Service).
type AccountService interface {
	CreateUser(ctx context.Context, actor authz.Principal, in account.CreateUserInput) (account.CreateUserResult, error)
	ListUsers(ctx context.Context, actor authz.Principal, f account.UserFilter, p domain.PageRequest) (domain.Page[domain.User], error)
	GetUser(ctx context.Context, actor authz.Principal, id uuid.UUID) (domain.User, error)
	UpdateUser(ctx context.Context, actor authz.Principal, id uuid.UUID, in account.UpdateUserInput) (domain.User, error)
	SetUserSkills(ctx context.Context, actor authz.Principal, id uuid.UUID, in []domain.UserSkillInput) ([]domain.UserSkill, error)
	ResendActivation(ctx context.Context, actor authz.Principal, id uuid.UUID) error
}

type userHandler struct {
	svc      AccountService
	validate *validator.Validate
}

type createUserRequest struct {
	Email        string     `json:"email" validate:"required,email,max=254"`
	FullName     string     `json:"fullName" validate:"required,max=100"`
	Role         string     `json:"role" validate:"required,oneof=admin manager member ceo"`
	DepartmentID *uuid.UUID `json:"departmentId"`
	JobTitleID   *uuid.UUID `json:"jobTitleId"`
}

// updateUserRequest: email chỉ đọc nên không có trong request (DisallowUnknownFields → 400).
type updateUserRequest struct {
	FullName     string     `json:"fullName" validate:"required,max=100"`
	Role         string     `json:"role" validate:"required,oneof=admin manager member ceo"`
	DepartmentID *uuid.UUID `json:"departmentId"`
	JobTitleID   *uuid.UUID `json:"jobTitleId"`
}

type userSkillRequest struct {
	SkillID uuid.UUID `json:"skillId" validate:"required"`
	Level   string    `json:"level" validate:"required"`
}

type setSkillsRequest struct {
	Skills []userSkillRequest `json:"skills" validate:"max=50,dive"`
}

type createUserResponse struct {
	User      userResponse `json:"user"`
	EmailSent bool         `json:"emailSent"`
}

type skillsResponse struct {
	Skills []userSkillResponse `json:"skills"`
}

var errUserIDInvalid = errBadRequest("USER_ID_INVALID", "Mã tài khoản không hợp lệ")

func (h *userHandler) list(w http.ResponseWriter, r *http.Request) {
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
	f, err := parseUserFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.svc.ListUsers(r.Context(), actor, f, domain.NewPageRequest(page, size))
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]userResponse, 0, len(res.Items))
	for _, u := range res.Items {
		items = append(items, toUserResponse(u))
	}
	writeJSON(w, http.StatusOK, pageResponse[userResponse]{
		Items: items, Page: res.Page, PageSize: res.PageSize, Total: res.Total,
	})
}

// parseUserFilter đọc ?search=&role=&status=&departmentId=&jobTitleId=. Giá trị sai → 400.
func parseUserFilter(r *http.Request) (account.UserFilter, error) {
	q := r.URL.Query()
	f := account.UserFilter{Search: q.Get("search")}
	if s := q.Get("role"); s != "" {
		role, err := domain.ParseRole(s)
		if err != nil {
			return f, errValidation
		}
		f.Role = &role
	}
	if s := q.Get("status"); s != "" {
		st, err := domain.ParseUserStatus(s)
		if err != nil {
			return f, errValidation
		}
		f.Status = &st
	}
	var err error
	if f.DepartmentID, err = queryUUID(r, "departmentId"); err != nil {
		return f, err
	}
	if f.JobTitleID, err = queryUUID(r, "jobTitleId"); err != nil {
		return f, err
	}
	return f, nil
}

func (h *userHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	var req createUserRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.svc.CreateUser(r.Context(), actor, account.CreateUserInput{
		Email: req.Email, FullName: req.FullName, Role: domain.Role(req.Role),
		DepartmentID: req.DepartmentID, JobTitleID: req.JobTitleID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, createUserResponse{User: toUserResponse(res.User), EmailSent: res.EmailSent})
}

func (h *userHandler) get(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, r, errUserIDInvalid)
		return
	}
	u, err := h.svc.GetUser(r.Context(), actor, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *userHandler) update(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, r, errUserIDInvalid)
		return
	}
	var req updateUserRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	u, err := h.svc.UpdateUser(r.Context(), actor, id, account.UpdateUserInput{
		FullName: req.FullName, Role: domain.Role(req.Role),
		DepartmentID: req.DepartmentID, JobTitleID: req.JobTitleID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *userHandler) setSkills(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, r, errUserIDInvalid)
		return
	}
	var req setSkillsRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	in := make([]domain.UserSkillInput, 0, len(req.Skills))
	for _, s := range req.Skills {
		in = append(in, domain.UserSkillInput{SkillID: s.SkillID, Level: s.Level})
	}
	skills, err := h.svc.SetUserSkills(r.Context(), actor, id, in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, skillsResponse{Skills: toUserSkillResponses(skills)})
}

func (h *userHandler) resendActivation(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, r, errUserIDInvalid)
		return
	}
	if err := h.svc.ResendActivation(r.Context(), actor, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func queryInt(r *http.Request, key string, fallback int) (int, error) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, errValidation
	}
	return n, nil
}

// queryUUID trả nil khi tham số vắng; sai định dạng → 400.
func queryUUID(r *http.Request, key string) (*uuid.UUID, error) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil, errValidation
	}
	return &id, nil
}
