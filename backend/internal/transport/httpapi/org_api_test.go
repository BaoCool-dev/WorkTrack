package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func (f *fakeAccounts) GetUser(_ context.Context, _ authz.Principal, id uuid.UUID) (domain.User, error) {
	f.calls++
	return domain.User{ID: id, Email: "a@worktrack.local", FullName: "Nguyễn Văn An", Role: domain.RoleMember, Status: domain.UserStatusActive}, nil
}

func (f *fakeAccounts) UpdateUser(_ context.Context, _ authz.Principal, id uuid.UUID, in account.UpdateUserInput) (domain.User, error) {
	f.calls++
	if f.updateErr != nil {
		return domain.User{}, f.updateErr
	}
	return domain.User{ID: id, FullName: in.FullName, Role: in.Role, DepartmentID: in.DepartmentID, DepartmentName: "Phòng Kỹ thuật"}, nil
}

func (f *fakeAccounts) SetUserSkills(_ context.Context, _ authz.Principal, _ uuid.UUID, in []domain.UserSkillInput) ([]domain.UserSkill, error) {
	f.calls++
	f.gotSkills = in
	if f.skillsErr != nil {
		return nil, f.skillsErr
	}
	out := make([]domain.UserSkill, 0, len(in))
	for _, s := range in {
		out = append(out, domain.UserSkill{SkillID: s.SkillID, Name: "Go", Level: domain.SkillLevel(s.Level)})
	}
	return out, nil
}

// fakeOrg trả dữ liệu cố định; err (nếu có) trả cho mọi thao tác ghi.
type fakeOrg struct {
	err   error
	calls int
}

func (f *fakeOrg) ListDepartments(context.Context, authz.Principal) ([]domain.Department, error) {
	f.calls++
	root := uuid.New()
	return []domain.Department{{ID: root, Name: "Kỹ thuật", MemberCount: 3}, {ID: uuid.New(), Name: "Backend", ParentID: &root}}, nil
}

func (f *fakeOrg) CreateDepartment(_ context.Context, _ authz.Principal, name string, parent *uuid.UUID) (domain.Department, error) {
	f.calls++
	return domain.Department{ID: uuid.New(), Name: name, ParentID: parent}, f.err
}

func (f *fakeOrg) UpdateDepartment(_ context.Context, _ authz.Principal, id uuid.UUID, name string, parent *uuid.UUID) (domain.Department, error) {
	f.calls++
	return domain.Department{ID: id, Name: name, ParentID: parent}, f.err
}

func (f *fakeOrg) DeleteDepartment(context.Context, authz.Principal, uuid.UUID) error {
	f.calls++
	return f.err
}

func (f *fakeOrg) ListJobTitles(context.Context, authz.Principal) ([]domain.JobTitle, error) {
	f.calls++
	return []domain.JobTitle{{ID: uuid.New(), Name: "Lập trình viên"}}, nil
}

func (f *fakeOrg) CreateJobTitle(_ context.Context, _ authz.Principal, name string) (domain.JobTitle, error) {
	f.calls++
	return domain.JobTitle{ID: uuid.New(), Name: name}, f.err
}

func (f *fakeOrg) UpdateJobTitle(_ context.Context, _ authz.Principal, id uuid.UUID, name string) (domain.JobTitle, error) {
	f.calls++
	return domain.JobTitle{ID: id, Name: name}, f.err
}

func (f *fakeOrg) DeleteJobTitle(context.Context, authz.Principal, uuid.UUID) error {
	f.calls++
	return f.err
}

func (f *fakeOrg) ListSkills(context.Context, authz.Principal) ([]domain.Skill, error) {
	f.calls++
	return []domain.Skill{{ID: uuid.New(), Name: "Go", MemberCount: 2}}, nil
}

func (f *fakeOrg) CreateSkill(_ context.Context, _ authz.Principal, name string) (domain.Skill, error) {
	f.calls++
	return domain.Skill{ID: uuid.New(), Name: name}, f.err
}

func (f *fakeOrg) UpdateSkill(_ context.Context, _ authz.Principal, id uuid.UUID, name string) (domain.Skill, error) {
	f.calls++
	return domain.Skill{ID: id, Name: name}, f.err
}

func (f *fakeOrg) DeleteSkill(context.Context, authz.Principal, uuid.UUID) error {
	f.calls++
	return f.err
}

func newOrgTestAPI(t *testing.T, fo *fakeOrg) testAPI {
	t.Helper()
	return newTestAPI(t, func(d *Deps) { d.Org = fo })
}

// Ma trận vai trò CF-02: chỉ Admin và CEO quản lý cơ cấu tổ chức và hồ sơ nhân sự.
func TestOrgAPI_RoleMatrix_CF02(t *testing.T) {
	id := uuid.NewString()
	endpoints := []struct {
		method, path, body string
		ok                 int
	}{
		{http.MethodGet, "/api/v1/departments", "", 200},
		{http.MethodPost, "/api/v1/departments", `{"name":"Kỹ thuật"}`, 201},
		{http.MethodPut, "/api/v1/departments/" + id, `{"name":"Kỹ thuật","parentId":null}`, 200},
		{http.MethodDelete, "/api/v1/departments/" + id, "", 204},
		{http.MethodGet, "/api/v1/job-titles", "", 200},
		{http.MethodPost, "/api/v1/job-titles", `{"name":"Tester"}`, 201},
		{http.MethodPut, "/api/v1/job-titles/" + id, `{"name":"Tester"}`, 200},
		{http.MethodDelete, "/api/v1/job-titles/" + id, "", 204},
		{http.MethodGet, "/api/v1/skills", "", 200},
		{http.MethodPost, "/api/v1/skills", `{"name":"React"}`, 201},
		{http.MethodPut, "/api/v1/skills/" + id, `{"name":"React"}`, 200},
		{http.MethodDelete, "/api/v1/skills/" + id, "", 204},
		{http.MethodGet, "/api/v1/users/" + id, "", 200},
		{http.MethodPut, "/api/v1/users/" + id, `{"fullName":"Nguyễn Văn An","role":"member"}`, 200},
		{http.MethodPut, "/api/v1/users/" + id + "/skills", `{"skills":[]}`, 200},
	}
	roles := []struct {
		token   string
		allowed bool
	}{
		{"valid-admin", true}, {"valid-ceo", true}, {"valid-manager", false}, {"valid-member", false},
	}
	for _, ep := range endpoints {
		for _, r := range roles {
			t.Run(ep.method+" "+ep.path+" "+r.token, func(t *testing.T) {
				fo := &fakeOrg{}
				a := newOrgTestAPI(t, fo)
				rec := a.do(t, ep.method, ep.path, ep.body, withToken(r.token))
				if r.allowed && rec.Code != ep.ok {
					t.Fatalf("status = %d, muốn %d; body %s", rec.Code, ep.ok, rec.Body.String())
				}
				if !r.allowed {
					if rec.Code != http.StatusForbidden {
						t.Fatalf("status = %d, muốn 403", rec.Code)
					}
					if fo.calls+a.acc.calls != 0 {
						t.Fatal("service không được gọi khi bị chặn quyền")
					}
				}
			})
		}
		t.Run(ep.method+" "+ep.path+" không token", func(t *testing.T) {
			rec := newOrgTestAPI(t, &fakeOrg{}).do(t, ep.method, ep.path, ep.body)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, muốn 401", rec.Code)
			}
		})
	}
}

func TestOrgAPI_DomainErrorMapping_CF02(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		err    error
		method string
		path   string
		body   string
		status int
		code   string
	}{
		{domain.ErrDepartmentCycle, http.MethodPut, "/api/v1/departments/" + id, `{"name":"A","parentId":"` + id + `"}`, 422, "DEPARTMENT_CYCLE"},
		{domain.ErrDepartmentNameTaken, http.MethodPost, "/api/v1/departments", `{"name":"A"}`, 409, "DEPARTMENT_NAME_TAKEN"},
		{domain.ErrDepartmentHasChildren, http.MethodDelete, "/api/v1/departments/" + id, "", 409, "DEPARTMENT_HAS_CHILDREN"},
		{domain.ErrDepartmentInUse, http.MethodDelete, "/api/v1/departments/" + id, "", 409, "DEPARTMENT_IN_USE"},
		{domain.ErrDepartmentNotFound, http.MethodDelete, "/api/v1/departments/" + id, "", 404, "DEPARTMENT_NOT_FOUND"},
		{domain.ErrInvalidOrgName, http.MethodPost, "/api/v1/skills", `{"name":"   "}`, 422, "ORG_NAME_INVALID"},
		{domain.ErrSkillNameTaken, http.MethodPost, "/api/v1/skills", `{"name":"Go"}`, 409, "SKILL_NAME_TAKEN"},
		{domain.ErrSkillInUse, http.MethodDelete, "/api/v1/skills/" + id, "", 409, "SKILL_IN_USE"},
		{domain.ErrJobTitleInUse, http.MethodDelete, "/api/v1/job-titles/" + id, "", 409, "JOB_TITLE_IN_USE"},
		{domain.ErrJobTitleNameTaken, http.MethodPut, "/api/v1/job-titles/" + id, `{"name":"Tester"}`, 409, "JOB_TITLE_NAME_TAKEN"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			rec := newOrgTestAPI(t, &fakeOrg{err: tt.err}).do(t, tt.method, tt.path, tt.body, withToken("valid-admin"))
			if rec.Code != tt.status || errCode(t, rec) != tt.code {
				t.Fatalf("got %d %s, muốn %d %s", rec.Code, rec.Body.String(), tt.status, tt.code)
			}
		})
	}
}

func TestOrgAPI_BadInput_CF02(t *testing.T) {
	tests := []struct {
		name, method, path, body string
	}{
		{"id sai định dạng", http.MethodDelete, "/api/v1/skills/abc", ""},
		{"thiếu tên", http.MethodPost, "/api/v1/departments", `{}`},
		{"parentId sai", http.MethodPost, "/api/v1/departments", `{"name":"A","parentId":"x"}`},
		{"trường lạ", http.MethodPost, "/api/v1/job-titles", `{"name":"A","extra":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fo := &fakeOrg{}
			rec := newOrgTestAPI(t, fo).do(t, tt.method, tt.path, tt.body, withToken("valid-admin"))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, muốn 400; body %s", rec.Code, rec.Body.String())
			}
			if fo.calls != 0 {
				t.Fatal("service không được gọi khi input sai")
			}
		})
	}
}

func TestOrgAPI_ListDepartmentsShape_CF02(t *testing.T) {
	rec := newOrgTestAPI(t, &fakeOrg{}).do(t, http.MethodGet, "/api/v1/departments", "", withToken("valid-ceo"))
	var body struct {
		Items []departmentResponse `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 2 || body.Items[0].ParentID != nil || body.Items[1].ParentID == nil || body.Items[0].MemberCount != 3 {
		t.Fatalf("body = %+v", body)
	}
}

func TestUserAPI_UpdateRejectsEmailChange_CF02(t *testing.T) {
	a := newOrgTestAPI(t, &fakeOrg{})
	rec := a.do(t, http.MethodPut, "/api/v1/users/"+uuid.NewString(),
		`{"fullName":"A","role":"member","email":"moi@x.vn"}`, withToken("valid-admin"))
	if rec.Code != http.StatusBadRequest || a.acc.calls != 0 {
		t.Fatalf("status = %d, calls = %d", rec.Code, a.acc.calls)
	}
}

func TestUserAPI_UpdateErrors_CF02(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrCannotChangeOwnRole, 422, "USER_CANNOT_CHANGE_OWN_ROLE"},
		{domain.ErrDepartmentNotFound, 404, "DEPARTMENT_NOT_FOUND"},
		{domain.ErrJobTitleNotFound, 404, "JOB_TITLE_NOT_FOUND"},
		{domain.ErrForbidden, 403, "FORBIDDEN"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			a := newTestAPI(t, func(d *Deps) { d.Accounts = &fakeAccounts{updateErr: tt.err} })
			rec := a.do(t, http.MethodPut, "/api/v1/users/"+uuid.NewString(), `{"fullName":"A","role":"ceo"}`, withToken("valid-admin"))
			if rec.Code != tt.status || errCode(t, rec) != tt.code {
				t.Fatalf("got %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestUserAPI_SetSkills_CF02(t *testing.T) {
	sk := uuid.NewString()
	t.Run("hợp lệ", func(t *testing.T) {
		a := newOrgTestAPI(t, &fakeOrg{})
		rec := a.do(t, http.MethodPut, "/api/v1/users/"+uuid.NewString()+"/skills",
			`{"skills":[{"skillId":"`+sk+`","level":"senior"}]}`, withToken("valid-admin"))
		if rec.Code != 200 || len(a.acc.gotSkills) != 1 || a.acc.gotSkills[0].Level != "senior" {
			t.Fatalf("got %d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("cấp độ sai", func(t *testing.T) {
		a := newTestAPI(t, func(d *Deps) { d.Accounts = &fakeAccounts{skillsErr: domain.ErrInvalidSkillLevel} })
		rec := a.do(t, http.MethodPut, "/api/v1/users/"+uuid.NewString()+"/skills",
			`{"skills":[{"skillId":"`+sk+`","level":"expert"}]}`, withToken("valid-admin"))
		if rec.Code != 422 || errCode(t, rec) != "SKILL_LEVEL_INVALID" {
			t.Fatalf("got %d %s", rec.Code, rec.Body.String())
		}
	})
	t.Run("trùng kỹ năng", func(t *testing.T) {
		a := newTestAPI(t, func(d *Deps) { d.Accounts = &fakeAccounts{skillsErr: domain.ErrDuplicateSkill} })
		rec := a.do(t, http.MethodPut, "/api/v1/users/"+uuid.NewString()+"/skills",
			`{"skills":[{"skillId":"`+sk+`","level":"beginner"},{"skillId":"`+sk+`","level":"senior"}]}`, withToken("valid-admin"))
		if rec.Code != 422 || errCode(t, rec) != "SKILL_DUPLICATE" {
			t.Fatalf("got %d %s", rec.Code, rec.Body.String())
		}
	})
}

func TestUserAPI_ListFilters_CF02(t *testing.T) {
	dep := uuid.New()
	a := newOrgTestAPI(t, &fakeOrg{})
	rec := a.do(t, http.MethodGet, "/api/v1/users?search=an&role=manager&status=active&departmentId="+dep.String(), "", withToken("valid-admin"))
	if rec.Code != 200 {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	f := a.acc.gotFilter
	if f.Search != "an" || f.Role == nil || *f.Role != domain.RoleManager || f.Status == nil || f.DepartmentID == nil || *f.DepartmentID != dep {
		t.Fatalf("filter = %+v", f)
	}
	for _, q := range []string{"role=boss", "status=x", "departmentId=x", "jobTitleId=1"} {
		rec := a.do(t, http.MethodGet, "/api/v1/users?"+q, "", withToken("valid-admin"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d", q, rec.Code)
		}
	}
}
