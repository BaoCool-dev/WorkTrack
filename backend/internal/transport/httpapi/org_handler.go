package httpapi

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// OrgService là các use case cơ cấu tổ chức (cài đặt: org.Service) — CF-02.
type OrgService interface {
	ListDepartments(ctx context.Context, actor authz.Principal) ([]domain.Department, error)
	CreateDepartment(ctx context.Context, actor authz.Principal, name string, parentID *uuid.UUID) (domain.Department, error)
	UpdateDepartment(ctx context.Context, actor authz.Principal, id uuid.UUID, name string, parentID *uuid.UUID) (domain.Department, error)
	DeleteDepartment(ctx context.Context, actor authz.Principal, id uuid.UUID) error

	ListJobTitles(ctx context.Context, actor authz.Principal) ([]domain.JobTitle, error)
	CreateJobTitle(ctx context.Context, actor authz.Principal, name string) (domain.JobTitle, error)
	UpdateJobTitle(ctx context.Context, actor authz.Principal, id uuid.UUID, name string) (domain.JobTitle, error)
	DeleteJobTitle(ctx context.Context, actor authz.Principal, id uuid.UUID) error

	ListSkills(ctx context.Context, actor authz.Principal) ([]domain.Skill, error)
	CreateSkill(ctx context.Context, actor authz.Principal, name string) (domain.Skill, error)
	UpdateSkill(ctx context.Context, actor authz.Principal, id uuid.UUID, name string) (domain.Skill, error)
	DeleteSkill(ctx context.Context, actor authz.Principal, id uuid.UUID) error
}

type orgHandler struct {
	svc      OrgService
	validate *validator.Validate
}

type departmentRequest struct {
	Name     string     `json:"name" validate:"required,max=200"`
	ParentID *uuid.UUID `json:"parentId"`
}

type nameRequest struct {
	Name string `json:"name" validate:"required,max=200"`
}

type departmentResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parentId"`
	MemberCount int     `json:"memberCount"`
}

// catalogItemResponse dùng chung cho chức danh và kỹ năng.
type catalogItemResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MemberCount int    `json:"memberCount"`
}

type listResponse[T any] struct {
	Items []T `json:"items"`
}

var errOrgIDInvalid = errBadRequest("ORG_ID_INVALID", "Mã không hợp lệ")

func toDepartmentResponse(d domain.Department) departmentResponse {
	var parent *string
	if d.ParentID != nil {
		s := d.ParentID.String()
		parent = &s
	}
	return departmentResponse{ID: d.ID.String(), Name: d.Name, ParentID: parent, MemberCount: d.MemberCount}
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, errOrgIDInvalid
	}
	return id, nil
}

func (h *orgHandler) listDepartments(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	ds, err := h.svc.ListDepartments(r.Context(), actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]departmentResponse, 0, len(ds))
	for _, d := range ds {
		items = append(items, toDepartmentResponse(d))
	}
	writeJSON(w, http.StatusOK, listResponse[departmentResponse]{Items: items})
}

func (h *orgHandler) createDepartment(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	var req departmentRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	d, err := h.svc.CreateDepartment(r.Context(), actor, req.Name, req.ParentID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toDepartmentResponse(d))
}

func (h *orgHandler) updateDepartment(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	var req departmentRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	d, err := h.svc.UpdateDepartment(r.Context(), actor, id, req.Name, req.ParentID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toDepartmentResponse(d))
}

func (h *orgHandler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	h.del(w, r, h.svc.DeleteDepartment)
}

func (h *orgHandler) listJobTitles(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	xs, err := h.svc.ListJobTitles(r.Context(), actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]catalogItemResponse, 0, len(xs))
	for _, x := range xs {
		items = append(items, catalogItemResponse{ID: x.ID.String(), Name: x.Name, MemberCount: x.MemberCount})
	}
	writeJSON(w, http.StatusOK, listResponse[catalogItemResponse]{Items: items})
}

func (h *orgHandler) createJobTitle(w http.ResponseWriter, r *http.Request) {
	h.saveNamed(w, r, false, func(ctx context.Context, a authz.Principal, id uuid.UUID, name string) (catalogItemResponse, error) {
		x, err := h.svc.CreateJobTitle(ctx, a, name)
		return catalogItemResponse{ID: x.ID.String(), Name: x.Name, MemberCount: x.MemberCount}, err
	})
}

func (h *orgHandler) updateJobTitle(w http.ResponseWriter, r *http.Request) {
	h.saveNamed(w, r, true, func(ctx context.Context, a authz.Principal, id uuid.UUID, name string) (catalogItemResponse, error) {
		x, err := h.svc.UpdateJobTitle(ctx, a, id, name)
		return catalogItemResponse{ID: x.ID.String(), Name: x.Name, MemberCount: x.MemberCount}, err
	})
}

func (h *orgHandler) deleteJobTitle(w http.ResponseWriter, r *http.Request) {
	h.del(w, r, h.svc.DeleteJobTitle)
}

func (h *orgHandler) listSkills(w http.ResponseWriter, r *http.Request) {
	actor, _ := principalFrom(r.Context())
	xs, err := h.svc.ListSkills(r.Context(), actor)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]catalogItemResponse, 0, len(xs))
	for _, x := range xs {
		items = append(items, catalogItemResponse{ID: x.ID.String(), Name: x.Name, MemberCount: x.MemberCount})
	}
	writeJSON(w, http.StatusOK, listResponse[catalogItemResponse]{Items: items})
}

func (h *orgHandler) createSkill(w http.ResponseWriter, r *http.Request) {
	h.saveNamed(w, r, false, func(ctx context.Context, a authz.Principal, id uuid.UUID, name string) (catalogItemResponse, error) {
		x, err := h.svc.CreateSkill(ctx, a, name)
		return catalogItemResponse{ID: x.ID.String(), Name: x.Name, MemberCount: x.MemberCount}, err
	})
}

func (h *orgHandler) updateSkill(w http.ResponseWriter, r *http.Request) {
	h.saveNamed(w, r, true, func(ctx context.Context, a authz.Principal, id uuid.UUID, name string) (catalogItemResponse, error) {
		x, err := h.svc.UpdateSkill(ctx, a, id, name)
		return catalogItemResponse{ID: x.ID.String(), Name: x.Name, MemberCount: x.MemberCount}, err
	})
}

func (h *orgHandler) deleteSkill(w http.ResponseWriter, r *http.Request) {
	h.del(w, r, h.svc.DeleteSkill)
}

// saveNamed xử lý chung tạo (201) / sửa (200) một mục danh mục chỉ có tên.
func (h *orgHandler) saveNamed(w http.ResponseWriter, r *http.Request, withID bool,
	save func(context.Context, authz.Principal, uuid.UUID, string) (catalogItemResponse, error),
) {
	actor, _ := principalFrom(r.Context())
	id := uuid.Nil
	if withID {
		var err error
		if id, err = pathID(r); err != nil {
			writeError(w, r, err)
			return
		}
	}
	var req nameRequest
	if err := decodeAndValidate(w, r, h.validate, &req); err != nil {
		writeError(w, r, err)
		return
	}
	out, err := save(r.Context(), actor, id, req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	status := http.StatusOK
	if !withID {
		status = http.StatusCreated
	}
	writeJSON(w, status, out)
}

func (h *orgHandler) del(w http.ResponseWriter, r *http.Request, fn func(context.Context, authz.Principal, uuid.UUID) error) {
	actor, _ := principalFrom(r.Context())
	id, err := pathID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := fn(r.Context(), actor, id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
