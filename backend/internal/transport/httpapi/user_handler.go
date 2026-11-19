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
	ListUsers(ctx context.Context, actor authz.Principal, p domain.PageRequest) (domain.Page[domain.User], error)
	ResendActivation(ctx context.Context, actor authz.Principal, id uuid.UUID) error
}

type userHandler struct {
	svc      AccountService
	validate *validator.Validate
}

type createUserRequest struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	FullName string `json:"fullName" validate:"required,max=100"`
	Role     string `json:"role" validate:"required,oneof=admin manager member ceo"`
}

type createUserResponse struct {
	User      userResponse `json:"user"`
	EmailSent bool         `json:"emailSent"`
}

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
	res, err := h.svc.ListUsers(r.Context(), actor, domain.NewPageRequest(page, size))
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

func (h *userHandler) create(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	var req createUserRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.svc.CreateUser(r.Context(), actor, account.CreateUserInput{
		Email: req.Email, FullName: req.FullName, Role: domain.Role(req.Role),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, createUserResponse{User: toUserResponse(res.User), EmailSent: res.EmailSent})
}

func (h *userHandler) resendActivation(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		writeError(w, r, errBadRequest("USER_ID_INVALID", "Mã tài khoản không hợp lệ"))
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
