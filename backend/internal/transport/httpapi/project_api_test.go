package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// fakeProjects ghi lại đầu vào và trả err (nếu có) cho mọi thao tác.
type fakeProjects struct {
	err         error
	calls       int
	gotActor    authz.Principal
	gotInput    project.Input
	gotList     project.ListInput
	gotPage     domain.PageRequest
	gotIDs      []uuid.UUID
	gotTs       []domain.Transition
	gotName     string
	gotGroup    string
	gotUserID   uuid.UUID
	gotSearch   string
	gotColumnID uuid.UUID
	gotWIP      int
}

var (
	testProjectStart = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	testBoardCols    = []domain.BoardColumn{
		{ID: uuid.New(), Name: "Cần làm", Position: 0, StatusGroup: domain.GroupTodo},
		{ID: uuid.New(), Name: "Hoàn thành", Position: 1, StatusGroup: domain.GroupDone, TaskCount: 2},
	}
)

func sampleProject(id uuid.UUID, in project.Input) domain.Project {
	name := in.Name
	if name == "" {
		name = "Cổng thông tin"
	}
	return domain.Project{
		ID: id, Name: name, Description: in.Description, Status: domain.ProjectNew,
		StartDate: &testProjectStart, ManagerID: uuid.New(), ManagerName: "Trần Quản Lý", MemberCount: 1,
		CreatedAt: testNow, UpdatedAt: testNow,
	}
}

func sampleBoard() project.Board {
	return project.Board{
		Columns: testBoardCols,
		Transitions: []domain.Transition{
			{FromColumnID: testBoardCols[0].ID, ToColumnID: testBoardCols[1].ID, Role: domain.TransitionManager},
		},
	}
}

func (f *fakeProjects) hit(actor authz.Principal) { f.calls++; f.gotActor = actor }

func (f *fakeProjects) Create(_ context.Context, a authz.Principal, in project.Input) (domain.Project, error) {
	f.hit(a)
	f.gotInput = in
	return sampleProject(uuid.New(), in), f.err
}

func (f *fakeProjects) List(_ context.Context, a authz.Principal, in project.ListInput, p domain.PageRequest) (domain.Page[domain.Project], error) {
	f.hit(a)
	f.gotList, f.gotPage = in, p
	return domain.Page[domain.Project]{Items: []domain.Project{sampleProject(uuid.New(), project.Input{})}, Page: p.Page, PageSize: p.PageSize, Total: 1}, f.err
}

func (f *fakeProjects) Get(_ context.Context, a authz.Principal, id uuid.UUID) (domain.Project, error) {
	f.hit(a)
	return sampleProject(id, project.Input{}), f.err
}

func (f *fakeProjects) Update(_ context.Context, a authz.Principal, id uuid.UUID, in project.Input) (domain.Project, error) {
	f.hit(a)
	f.gotInput = in
	return sampleProject(id, in), f.err
}

func (f *fakeProjects) SetWIPLimit(_ context.Context, a authz.Principal, id uuid.UUID, limit int) (domain.Project, error) {
	f.hit(a)
	f.gotWIP = limit
	p := sampleProject(id, project.Input{})
	p.WIPLimit = limit
	return p, f.err
}

func (f *fakeProjects) ListMembers(_ context.Context, a authz.Principal, _ uuid.UUID) ([]domain.ProjectMember, error) {
	f.hit(a)
	return []domain.ProjectMember{{UserID: uuid.New(), FullName: "Lê Thành Viên", Email: "tv@worktrack.local",
		Role: domain.RoleMember, Status: domain.UserStatusActive, AddedAt: testNow}}, f.err
}

func (f *fakeProjects) ListCandidates(_ context.Context, a authz.Principal, _ uuid.UUID, search string) ([]domain.MemberCandidate, error) {
	f.hit(a)
	f.gotSearch = search
	return []domain.MemberCandidate{{UserID: uuid.New(), FullName: "Phạm Ứng Viên", Role: domain.RoleMember, Status: domain.UserStatusActive}}, f.err
}

func (f *fakeProjects) AddMember(_ context.Context, a authz.Principal, _, userID uuid.UUID) error {
	f.hit(a)
	f.gotUserID = userID
	return f.err
}

func (f *fakeProjects) RemoveMember(_ context.Context, a authz.Principal, _, userID uuid.UUID) error {
	f.hit(a)
	f.gotUserID = userID
	return f.err
}

func (f *fakeProjects) GetBoard(_ context.Context, a authz.Principal, _ uuid.UUID) (project.Board, error) {
	f.hit(a)
	return sampleBoard(), f.err
}

func (f *fakeProjects) AddColumn(_ context.Context, a authz.Principal, _ uuid.UUID, name, group string) (project.Board, error) {
	f.hit(a)
	f.gotName, f.gotGroup = name, group
	return sampleBoard(), f.err
}

func (f *fakeProjects) UpdateColumn(_ context.Context, a authz.Principal, _, columnID uuid.UUID, name, group string) (project.Board, error) {
	f.hit(a)
	f.gotColumnID, f.gotName, f.gotGroup = columnID, name, group
	return sampleBoard(), f.err
}

func (f *fakeProjects) DeleteColumn(_ context.Context, a authz.Principal, _, columnID uuid.UUID) (project.Board, error) {
	f.hit(a)
	f.gotColumnID = columnID
	return sampleBoard(), f.err
}

func (f *fakeProjects) ReorderColumns(_ context.Context, a authz.Principal, _ uuid.UUID, ids []uuid.UUID) (project.Board, error) {
	f.hit(a)
	f.gotIDs = ids
	return sampleBoard(), f.err
}

func (f *fakeProjects) ReplaceTransitions(_ context.Context, a authz.Principal, _ uuid.UUID, ts []domain.Transition) (project.Board, error) {
	f.hit(a)
	f.gotTs = ts
	return sampleBoard(), f.err
}

func newProjectAPI(t *testing.T) (testAPI, *fakeProjects) {
	t.Helper()
	fp := &fakeProjects{}
	return newTestAPI(t, func(d *Deps) { d.Projects = fp }), fp
}

const mgrTok = "valid-manager"

func TestProjectAPI_HappyPaths_CF03(t *testing.T) {
	pid, uid, cid := uuid.New(), uuid.New(), uuid.New()
	base := "/api/v1/projects/" + pid.String()
	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"list", http.MethodGet, "/api/v1/projects?page=2&pageSize=10&search=c%E1%BB%95ng&status=new", "", http.StatusOK},
		{"create", http.MethodPost, "/api/v1/projects", `{"name":"Cổng thông tin","startDate":"2026-12-01","endDate":"2027-01-31"}`, http.StatusCreated},
		{"get", http.MethodGet, base, "", http.StatusOK},
		{"update", http.MethodPut, base, `{"name":"Mới","status":"in_progress","startDate":null,"endDate":""}`, http.StatusOK},
		{"members", http.MethodGet, base + "/members", "", http.StatusOK},
		{"candidates", http.MethodGet, base + "/member-candidates?search=ph", "", http.StatusOK},
		{"add member", http.MethodPost, base + "/members", fmt.Sprintf(`{"userId":%q}`, uid), http.StatusNoContent},
		{"remove member", http.MethodDelete, base + "/members/" + uid.String(), "", http.StatusNoContent},
		{"board", http.MethodGet, base + "/board", "", http.StatusOK},
		{"add column", http.MethodPost, base + "/columns", `{"name":"Kiểm thử","statusGroup":"in_progress"}`, http.StatusCreated},
		{"update column", http.MethodPut, base + "/columns/" + cid.String(), `{"name":"QA","statusGroup":"in_progress"}`, http.StatusOK},
		{"delete column", http.MethodDelete, base + "/columns/" + cid.String(), "", http.StatusOK},
		{"reorder", http.MethodPut, base + "/column-order", fmt.Sprintf(`{"columnIds":[%q]}`, cid), http.StatusOK},
		{"transitions", http.MethodPut, base + "/transitions",
			fmt.Sprintf(`{"transitions":[{"fromColumnId":%q,"toColumnId":%q,"role":"member"}]}`, cid, uid), http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, fp := newProjectAPI(t)
			rec := a.do(t, tc.method, tc.path, tc.body, withToken(mgrTok))
			if rec.Code != tc.want {
				t.Fatalf("status = %d, muốn %d: %s", rec.Code, tc.want, rec.Body.String())
			}
			if fp.calls != 1 || fp.gotActor.Role != domain.RoleManager {
				t.Fatalf("service gọi %d lần, actor %+v", fp.calls, fp.gotActor)
			}
		})
	}
}

func TestProjectAPI_PassesInputToService(t *testing.T) {
	pid, uid, cid := uuid.New(), uuid.New(), uuid.New()
	base := "/api/v1/projects/" + pid.String()

	a, fp := newProjectAPI(t)
	a.do(t, http.MethodGet, "/api/v1/projects?page=2&pageSize=10&search=abc&status=paused", "", withToken("valid-ceo"))
	if fp.gotList.Search != "abc" || fp.gotList.Status != "paused" || fp.gotPage.Page != 2 || fp.gotPage.PageSize != 10 {
		t.Fatalf("list input: %+v %+v", fp.gotList, fp.gotPage)
	}

	a, fp = newProjectAPI(t)
	a.do(t, http.MethodPost, "/api/v1/projects", `{"name":"X","description":"mô tả","status":"new","startDate":"2026-12-01"}`, withToken(mgrTok))
	if fp.gotInput.Name != "X" || fp.gotInput.Description != "mô tả" || fp.gotInput.Status != "new" ||
		fp.gotInput.StartDate == nil || !fp.gotInput.StartDate.Equal(testProjectStart) || fp.gotInput.EndDate != nil {
		t.Fatalf("create input: %+v", fp.gotInput)
	}

	a, fp = newProjectAPI(t)
	a.do(t, http.MethodPut, base+"/columns/"+cid.String(), `{"name":"QA","statusGroup":"in_progress"}`, withToken(mgrTok))
	if fp.gotColumnID != cid || fp.gotName != "QA" || fp.gotGroup != "in_progress" {
		t.Fatalf("update column: %v %q %q", fp.gotColumnID, fp.gotName, fp.gotGroup)
	}

	a, fp = newProjectAPI(t)
	a.do(t, http.MethodPut, base+"/column-order", fmt.Sprintf(`{"columnIds":[%q,%q]}`, cid, uid), withToken(mgrTok))
	if len(fp.gotIDs) != 2 || fp.gotIDs[0] != cid || fp.gotIDs[1] != uid {
		t.Fatalf("reorder ids: %v", fp.gotIDs)
	}

	a, fp = newProjectAPI(t)
	a.do(t, http.MethodPut, base+"/transitions",
		fmt.Sprintf(`{"transitions":[{"fromColumnId":%q,"toColumnId":%q,"role":"member"}]}`, cid, uid), withToken(mgrTok))
	if len(fp.gotTs) != 1 || fp.gotTs[0] != (domain.Transition{FromColumnID: cid, ToColumnID: uid, Role: domain.TransitionMember}) {
		t.Fatalf("transitions: %+v", fp.gotTs)
	}

	a, fp = newProjectAPI(t)
	a.do(t, http.MethodDelete, base+"/members/"+uid.String(), "", withToken(mgrTok))
	if fp.gotUserID != uid {
		t.Fatalf("remove member user = %v", fp.gotUserID)
	}

	a, fp = newProjectAPI(t)
	a.do(t, http.MethodGet, base+"/member-candidates?search=ph", "", withToken(mgrTok))
	if fp.gotSearch != "ph" {
		t.Fatalf("candidate search = %q", fp.gotSearch)
	}
}

func TestProjectAPI_ResponseShape(t *testing.T) {
	a, _ := newProjectAPI(t)
	pid := uuid.New()
	rec := a.do(t, http.MethodGet, "/api/v1/projects/"+pid.String(), "", withToken(mgrTok))
	var p map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p["id"] != pid.String() || p["startDate"] != "2026-12-01" || p["endDate"] != nil || p["status"] != "new" {
		t.Fatalf("project json: %v", p)
	}
	if m, _ := p["manager"].(map[string]any); m["name"] != "Trần Quản Lý" {
		t.Fatalf("manager json: %v", p["manager"])
	}

	rec = a.do(t, http.MethodGet, "/api/v1/projects/"+pid.String()+"/board", "", withToken(mgrTok))
	var b boardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatal(err)
	}
	if len(b.Columns) != 2 || b.Columns[1].TaskCount != 2 || b.Columns[1].StatusGroup != domain.GroupDone || len(b.Transitions) != 1 {
		t.Fatalf("board json: %s", rec.Body.String())
	}

	rec = a.do(t, http.MethodGet, "/api/v1/projects/"+pid.String()+"/members", "", withToken(mgrTok))
	var ms itemsResponse[projectMemberResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &ms); err != nil {
		t.Fatal(err)
	}
	if len(ms.Items) != 1 || ms.Items[0].AddedAt == nil || ms.Items[0].Email != "tv@worktrack.local" {
		t.Fatalf("members json: %s", rec.Body.String())
	}
}

func TestProjectAPI_RequiresAuth(t *testing.T) {
	a, fp := newProjectAPI(t)
	for _, path := range []string{"/api/v1/projects", "/api/v1/projects/" + uuid.NewString() + "/board"} {
		rec := a.do(t, http.MethodGet, path, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s không token: %d", path, rec.Code)
		}
	}
	if fp.calls != 0 {
		t.Fatal("service không được gọi khi chưa đăng nhập")
	}
}

func TestProjectAPI_BadRequests(t *testing.T) {
	pid := uuid.NewString()
	base := "/api/v1/projects/" + pid
	cases := []struct {
		name, method, path, body, code string
	}{
		{"bad project id", http.MethodGet, "/api/v1/projects/abc", "", "PROJECT_ID_INVALID"},
		{"bad project id board", http.MethodGet, "/api/v1/projects/abc/board", "", "PROJECT_ID_INVALID"},
		{"bad column id", http.MethodDelete, base + "/columns/abc", "", "COLUMN_ID_INVALID"},
		{"bad user id", http.MethodDelete, base + "/members/abc", "", "USER_ID_INVALID"},
		{"bad date", http.MethodPost, "/api/v1/projects", `{"name":"X","startDate":"01/12/2026"}`, "DATE_INVALID"},
		{"bad end date", http.MethodPut, base, `{"name":"X","endDate":"2026-13-01"}`, "DATE_INVALID"},
		{"bad json", http.MethodPost, "/api/v1/projects", `{"name":`, ""},
		{"unknown field", http.MethodPost, "/api/v1/projects", `{"name":"X","managerId":"x"}`, ""},
		{"missing user", http.MethodPost, base + "/members", `{}`, ""},
		{"missing column ids", http.MethodPut, base + "/column-order", `{}`, ""},
		{"transition missing role", http.MethodPut, base + "/transitions",
			fmt.Sprintf(`{"transitions":[{"fromColumnId":%q,"toColumnId":%q}]}`, uuid.New(), uuid.New()), ""},
		{"bad page", http.MethodGet, "/api/v1/projects?page=x", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, fp := newProjectAPI(t)
			rec := a.do(t, tc.method, tc.path, tc.body, withToken(mgrTok))
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
			}
			if tc.code != "" {
				if got := errCode(t, rec); got != tc.code {
					t.Fatalf("code = %s, muốn %s", got, tc.code)
				}
			}
			if fp.calls != 0 {
				t.Fatal("service không được gọi khi request sai")
			}
		})
	}
}

func TestProjectAPI_ErrorMapping(t *testing.T) {
	pid := uuid.NewString()
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{domain.ErrProjectNotFound, http.StatusNotFound, "PROJECT_NOT_FOUND"},
		{domain.ErrProjectMemberNotFound, http.StatusNotFound, "PROJECT_MEMBER_NOT_FOUND"},
		{domain.ErrColumnNotFound, http.StatusNotFound, "COLUMN_NOT_FOUND"},
		{domain.ErrProjectNameTaken, http.StatusConflict, "PROJECT_NAME_TAKEN"},
		{domain.ErrColumnNameTaken, http.StatusConflict, "COLUMN_NAME_TAKEN"},
		{domain.ErrColumnHasTasks, http.StatusConflict, "COLUMN_HAS_TASKS"},
		{domain.ErrInvalidProjectName, http.StatusUnprocessableEntity, "PROJECT_NAME_INVALID"},
		{domain.ErrInvalidProjectDescription, http.StatusUnprocessableEntity, "PROJECT_DESCRIPTION_INVALID"},
		{domain.ErrInvalidProjectDates, http.StatusUnprocessableEntity, "PROJECT_DATES_INVALID"},
		{domain.ErrInvalidProjectStatus, http.StatusUnprocessableEntity, "PROJECT_STATUS_INVALID"},
		{domain.ErrInvalidProjectManager, http.StatusUnprocessableEntity, "PROJECT_MANAGER_INVALID"},
		{domain.ErrInvalidProjectMember, http.StatusUnprocessableEntity, "PROJECT_MEMBER_INVALID"},
		{domain.ErrCannotRemoveProjectManager, http.StatusUnprocessableEntity, "PROJECT_CANNOT_REMOVE_MANAGER"},
		{domain.ErrInvalidColumnName, http.StatusUnprocessableEntity, "COLUMN_NAME_INVALID"},
		{domain.ErrColumnCount, http.StatusUnprocessableEntity, "COLUMN_COUNT_INVALID"},
		{domain.ErrBoardNeedsDoneColumn, http.StatusUnprocessableEntity, "BOARD_NEEDS_DONE_COLUMN"},
		{domain.ErrInvalidColumnOrder, http.StatusUnprocessableEntity, "COLUMN_ORDER_INVALID"},
		{domain.ErrInvalidTransition, http.StatusUnprocessableEntity, "TRANSITION_INVALID"},
		{domain.ErrInvalidTransitionRole, http.StatusUnprocessableEntity, "TRANSITION_ROLE_INVALID"},
		{domain.ErrMemberTransitionToDone, http.StatusUnprocessableEntity, "TRANSITION_MEMBER_TO_DONE"},
		{domain.ErrDuplicateTransition, http.StatusUnprocessableEntity, "TRANSITION_DUPLICATE"},
		{fmt.Errorf("bọc: %w", domain.ErrColumnHasTasks), http.StatusConflict, "COLUMN_HAS_TASKS"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			a, fp := newProjectAPI(t)
			fp.err = tc.err
			rec := a.do(t, http.MethodDelete, "/api/v1/projects/"+pid+"/columns/"+uuid.NewString(), "", withToken(mgrTok))
			if rec.Code != tc.status || errCode(t, rec) != tc.code {
				t.Fatalf("status %d code %s, muốn %d %s", rec.Code, errCode(t, rec), tc.status, tc.code)
			}
		})
	}
}
