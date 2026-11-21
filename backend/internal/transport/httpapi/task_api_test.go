package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

type fakeTasks struct {
	err       error
	calls     int
	gotActor  authz.Principal
	gotID     uuid.UUID
	gotInput  task.Input
	gotList   task.ListInput
	gotMine   task.MyListInput
	gotPage   domain.PageRequest
	gotLabel  string
	gotHours  int
	labelErr  error
	settingEr error
}

func sampleTask(id uuid.UUID) domain.Task {
	sp, est := 3, 720
	a := uuid.New()
	return domain.Task{
		ID: id, ProjectID: uuid.New(), ProjectName: "Cổng thông tin", ColumnID: uuid.New(), ColumnName: "Cần làm",
		StatusGroup: domain.GroupTodo, Title: "Thiết kế CSDL", StoryPoint: &sp, EstimatedMinutes: &est,
		AssigneeID: &a, AssigneeName: "Nguyễn Văn A", DueDate: &testProjectStart,
		Labels:    []domain.TaskLabel{{ID: uuid.New(), Name: "backend"}},
		CreatedBy: uuid.New(), CreatedByName: "Trần Quản Lý", CreatedAt: testNow, UpdatedAt: testNow,
	}
}

func (f *fakeTasks) hit(a authz.Principal) { f.calls++; f.gotActor = a }

func (f *fakeTasks) page(p domain.PageRequest) domain.Page[domain.Task] {
	return domain.Page[domain.Task]{Items: []domain.Task{sampleTask(uuid.New())}, Page: p.Page, PageSize: p.PageSize, Total: 1}
}

func (f *fakeTasks) List(_ context.Context, a authz.Principal, pid uuid.UUID, in task.ListInput, p domain.PageRequest) (domain.Page[domain.Task], error) {
	f.hit(a)
	f.gotID, f.gotList, f.gotPage = pid, in, p
	return f.page(p), f.err
}

func (f *fakeTasks) ListMine(_ context.Context, a authz.Principal, in task.MyListInput, p domain.PageRequest) (domain.Page[domain.Task], error) {
	f.hit(a)
	f.gotMine, f.gotPage = in, p
	return f.page(p), f.err
}

func (f *fakeTasks) MyWIP(_ context.Context, a authz.Principal) (int, error) {
	f.hit(a)
	return 2, f.err
}

func (f *fakeTasks) Get(_ context.Context, a authz.Principal, id uuid.UUID) (domain.Task, error) {
	f.hit(a)
	f.gotID = id
	return sampleTask(id), f.err
}

func (f *fakeTasks) Create(_ context.Context, a authz.Principal, pid uuid.UUID, in task.Input) (domain.Task, error) {
	f.hit(a)
	f.gotID, f.gotInput = pid, in
	return sampleTask(uuid.New()), f.err
}

func (f *fakeTasks) Update(_ context.Context, a authz.Principal, id uuid.UUID, in task.Input) (domain.Task, error) {
	f.hit(a)
	f.gotID, f.gotInput = id, in
	return sampleTask(id), f.err
}

func (f *fakeTasks) Claim(_ context.Context, a authz.Principal, id uuid.UUID) (domain.Task, error) {
	f.hit(a)
	f.gotID = id
	return sampleTask(id), f.err
}

type fakeLabels struct{ f *fakeTasks }

func (l fakeLabels) List(_ context.Context, a authz.Principal) ([]domain.Label, error) {
	l.f.hit(a)
	return []domain.Label{{ID: uuid.New(), Name: "backend", TaskCount: 1}}, l.f.labelErr
}

func (l fakeLabels) Create(_ context.Context, a authz.Principal, name string) (domain.Label, error) {
	l.f.hit(a)
	l.f.gotLabel = name
	return domain.Label{ID: uuid.New(), Name: name}, l.f.labelErr
}

func (l fakeLabels) Update(_ context.Context, a authz.Principal, id uuid.UUID, name string) (domain.Label, error) {
	l.f.hit(a)
	l.f.gotID, l.f.gotLabel = id, name
	return domain.Label{ID: id, Name: name}, l.f.labelErr
}

func (l fakeLabels) Delete(_ context.Context, a authz.Principal, id uuid.UUID) error {
	l.f.hit(a)
	l.f.gotID = id
	return l.f.labelErr
}

type fakeSettings struct{ f *fakeTasks }

func (s fakeSettings) Get(_ context.Context, a authz.Principal) (setting.Settings, error) {
	s.f.hit(a)
	return setting.Settings{HoursPerStoryPoint: 4, UpdatedAt: testNow}, s.f.settingEr
}

func (s fakeSettings) Update(_ context.Context, a authz.Principal, h int) (setting.Settings, error) {
	s.f.hit(a)
	s.f.gotHours = h
	return setting.Settings{HoursPerStoryPoint: h, UpdatedAt: testNow}, s.f.settingEr
}

func newTaskAPI(t *testing.T, f *fakeTasks) testAPI {
	t.Helper()
	return newTestAPI(t, func(d *Deps) {
		d.Tasks, d.Labels, d.Settings = f, fakeLabels{f}, fakeSettings{f}
	})
}

func TestTaskAPI_HappyPaths_CF06(t *testing.T) {
	pid, tid, lid := uuid.New(), uuid.New(), uuid.New()
	cases := []struct {
		method, path, body string
		status             int
	}{
		{http.MethodGet, "/api/v1/projects/" + pid.String() + "/tasks", "", http.StatusOK},
		{http.MethodPost, "/api/v1/projects/" + pid.String() + "/tasks", `{"title":"Việc mới"}`, http.StatusCreated},
		{http.MethodGet, "/api/v1/tasks/" + tid.String(), "", http.StatusOK},
		{http.MethodPut, "/api/v1/tasks/" + tid.String(), `{"title":"Sửa"}`, http.StatusOK},
		{http.MethodPost, "/api/v1/tasks/" + tid.String() + "/claim", "", http.StatusOK},
		{http.MethodGet, "/api/v1/me/tasks", "", http.StatusOK},
		{http.MethodGet, "/api/v1/me/wip", "", http.StatusOK},
		{http.MethodGet, "/api/v1/labels", "", http.StatusOK},
		{http.MethodPost, "/api/v1/labels", `{"name":"frontend"}`, http.StatusCreated},
		{http.MethodPut, "/api/v1/labels/" + lid.String(), `{"name":"ui"}`, http.StatusOK},
		{http.MethodDelete, "/api/v1/labels/" + lid.String(), "", http.StatusNoContent},
		{http.MethodGet, "/api/v1/settings", "", http.StatusOK},
		{http.MethodPut, "/api/v1/settings", `{"storyPointHours":6}`, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			f := &fakeTasks{}
			rec := newTaskAPI(t, f).do(t, tc.method, tc.path, tc.body, withToken("valid-manager"))
			if rec.Code != tc.status || f.calls != 1 {
				t.Fatalf("status %d calls %d body %s", rec.Code, f.calls, rec.Body.String())
			}
			if f.gotActor.Role != domain.RoleManager {
				t.Fatalf("actor %+v", f.gotActor)
			}
		})
	}
}

func TestTaskAPI_PassesInputToService(t *testing.T) {
	pid, cid, aid, lid := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	t.Run("lọc và phân trang danh sách", func(t *testing.T) {
		f := &fakeTasks{}
		path := "/api/v1/projects/" + pid.String() + "/tasks?search=csdl&statusGroup=todo&columnId=" + cid.String() +
			"&assigneeId=" + aid.String() + "&labelId=" + lid.String() + "&unassigned=true&page=2&pageSize=10"
		rec := newTaskAPI(t, f).do(t, http.MethodGet, path, "", withToken("valid-member"))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		g := f.gotList
		if f.gotID != pid || g.Search != "csdl" || g.StatusGroup != "todo" || !g.Unassigned ||
			g.ColumnID == nil || *g.ColumnID != cid || g.AssigneeID == nil || *g.AssigneeID != aid ||
			g.LabelID == nil || *g.LabelID != lid || f.gotPage.Page != 2 || f.gotPage.PageSize != 10 {
			t.Fatalf("list input %+v page %+v", g, f.gotPage)
		}
	})

	t.Run("tạo việc đủ trường", func(t *testing.T) {
		f := &fakeTasks{}
		body := `{"title":"Thiết kế","description":"mô tả","storyPoint":5,"assigneeId":"` + aid.String() +
			`","dueDate":"2026-12-15","labelIds":["` + lid.String() + `"]}`
		rec := newTaskAPI(t, f).do(t, http.MethodPost, "/api/v1/projects/"+pid.String()+"/tasks", body, withToken("valid-manager"))
		if rec.Code != http.StatusCreated {
			t.Fatalf("status %d %s", rec.Code, rec.Body.String())
		}
		in := f.gotInput
		if in.Title != "Thiết kế" || in.Description != "mô tả" || in.StoryPoint == nil || *in.StoryPoint != 5 ||
			in.AssigneeID == nil || *in.AssigneeID != aid || in.DueDate == nil || in.DueDate.Format("2006-01-02") != "2026-12-15" ||
			len(in.LabelIDs) != 1 || in.LabelIDs[0] != lid {
			t.Fatalf("input %+v", in)
		}
	})

	t.Run("việc của tôi", func(t *testing.T) {
		f := &fakeTasks{}
		newTaskAPI(t, f).do(t, http.MethodGet, "/api/v1/me/tasks?search=a&statusGroup=in_progress", "", withToken("valid-member"))
		if f.gotMine.Search != "a" || f.gotMine.StatusGroup != "in_progress" || f.gotPage.Page != 1 || f.gotPage.PageSize != domain.DefaultPageSize {
			t.Fatalf("mine %+v page %+v", f.gotMine, f.gotPage)
		}
	})

	t.Run("nhãn và cấu hình", func(t *testing.T) {
		f := &fakeTasks{}
		a := newTaskAPI(t, f)
		a.do(t, http.MethodPost, "/api/v1/labels", `{"name":"  api "}`, withToken("valid-admin"))
		if f.gotLabel != "  api " {
			t.Fatalf("label %q", f.gotLabel)
		}
		a.do(t, http.MethodPut, "/api/v1/settings", `{"storyPointHours":8}`, withToken("valid-admin"))
		if f.gotHours != 8 {
			t.Fatalf("hours %d", f.gotHours)
		}
	})
}

func TestTaskAPI_ResponseShape(t *testing.T) {
	f := &fakeTasks{}
	a := newTaskAPI(t, f)
	tid := uuid.New()
	rec := a.do(t, http.MethodGet, "/api/v1/tasks/"+tid.String(), "", withToken("valid-member"))
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"id", "projectId", "projectName", "title", "description", "storyPoint", "estimatedMinutes",
		"column", "assignee", "dueDate", "labels", "createdBy", "createdAt", "updatedAt"} {
		if _, ok := m[k]; !ok {
			t.Errorf("thiếu trường %s", k)
		}
	}
	if m["id"] != tid.String() || m["dueDate"] != "2026-12-01" || m["estimatedMinutes"] != float64(720) {
		t.Fatalf("body %v", m)
	}
	col, _ := m["column"].(map[string]any)
	if col["statusGroup"] != "todo" || col["name"] != "Cần làm" {
		t.Fatalf("column %v", col)
	}

	rec = a.do(t, http.MethodGet, "/api/v1/me/wip", "", withToken("valid-member"))
	if rec.Body.String() != "{\"count\":2}\n" && rec.Body.String() != `{"count":2}` {
		t.Fatalf("wip %s", rec.Body.String())
	}
	rec = a.do(t, http.MethodGet, "/api/v1/settings", "", withToken("valid-member"))
	var s settingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil || s.StoryPointHours != 4 {
		t.Fatalf("settings %s", rec.Body.String())
	}
	rec = a.do(t, http.MethodGet, "/api/v1/labels", "", withToken("valid-member"))
	var ls itemsResponse[labelResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &ls); err != nil || len(ls.Items) != 1 || ls.Items[0].TaskCount != 1 {
		t.Fatalf("labels %s", rec.Body.String())
	}
}

func TestTaskAPI_RequiresAuth(t *testing.T) {
	f := &fakeTasks{}
	a := newTaskAPI(t, f)
	for _, p := range []string{"/api/v1/projects/" + uuid.NewString() + "/tasks", "/api/v1/tasks/" + uuid.NewString(),
		"/api/v1/me/tasks", "/api/v1/me/wip", "/api/v1/labels", "/api/v1/settings"} {
		if rec := a.do(t, http.MethodGet, p, ""); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s: %d", p, rec.Code)
		}
	}
	if f.calls != 0 {
		t.Fatal("service bị gọi khi chưa đăng nhập")
	}
}

func TestTaskAPI_BadRequests(t *testing.T) {
	pid := uuid.NewString()
	cases := []struct {
		name, method, path, body, code string
	}{
		{"mã dự án sai", http.MethodGet, "/api/v1/projects/abc/tasks", "", "PROJECT_ID_INVALID"},
		{"mã việc sai", http.MethodGet, "/api/v1/tasks/abc", "", "TASK_ID_INVALID"},
		{"mã nhãn sai", http.MethodDelete, "/api/v1/labels/abc", "", "LABEL_ID_INVALID"},
		{"page không phải số", http.MethodGet, "/api/v1/projects/" + pid + "/tasks?page=x", "", "VALIDATION_FAILED"},
		{"columnId sai", http.MethodGet, "/api/v1/projects/" + pid + "/tasks?columnId=x", "", "VALIDATION_FAILED"},
		{"hạn chót sai", http.MethodPost, "/api/v1/projects/" + pid + "/tasks", `{"title":"a","dueDate":"15/12/2026"}`, "DATE_INVALID"},
		{"trường lạ", http.MethodPost, "/api/v1/projects/" + pid + "/tasks", `{"title":"a","columnId":"x"}`, ""},
		{"assignee sai", http.MethodPost, "/api/v1/projects/" + pid + "/tasks", `{"title":"a","assigneeId":"x"}`, ""},
		{"hours không phải số", http.MethodPut, "/api/v1/settings", `{"storyPointHours":"4"}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeTasks{}
			rec := newTaskAPI(t, f).do(t, tc.method, tc.path, tc.body, withToken("valid-manager"))
			if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status %d %s", rec.Code, rec.Body.String())
			}
			if tc.code != "" {
				if got := errCode(t, rec); got != tc.code {
					t.Fatalf("code %s, muốn %s", got, tc.code)
				}
			}
			if f.calls != 0 {
				t.Fatal("service không được gọi khi request sai")
			}
		})
	}
}

func TestTaskAPI_ErrorMapping_CF06(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrTaskNotFound, http.StatusNotFound, "TASK_NOT_FOUND"},
		{domain.ErrInvalidTaskTitle, http.StatusUnprocessableEntity, "TASK_TITLE_INVALID"},
		{domain.ErrInvalidTaskDescription, http.StatusUnprocessableEntity, "TASK_DESCRIPTION_INVALID"},
		{domain.ErrInvalidStoryPoint, http.StatusUnprocessableEntity, "TASK_STORY_POINT_INVALID"},
		{domain.ErrTooManyLabels, http.StatusUnprocessableEntity, "TASK_TOO_MANY_LABELS"},
		{domain.ErrLabelNotFound, http.StatusNotFound, "LABEL_NOT_FOUND"},
		{domain.ErrInvalidAssignee, http.StatusUnprocessableEntity, "TASK_ASSIGNEE_INVALID"},
		{domain.ErrTaskAlreadyAssigned, http.StatusConflict, "TASK_ALREADY_ASSIGNED"},
		{domain.ErrTaskNotClaimable, http.StatusConflict, "TASK_NOT_CLAIMABLE"},
		{domain.ErrBoardNeedsTodoColumn, http.StatusConflict, "BOARD_NEEDS_TODO_COLUMN"},
		{domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			f := &fakeTasks{err: tc.err}
			rec := newTaskAPI(t, f).do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/claim", "", withToken("valid-member"))
			if rec.Code != tc.status || errCode(t, rec) != tc.code {
				t.Fatalf("status %d code %s", rec.Code, errCode(t, rec))
			}
		})
	}
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrLabelNameTaken, http.StatusConflict, "LABEL_NAME_TAKEN"},
		{domain.ErrLabelInUse, http.StatusConflict, "LABEL_IN_USE"},
	} {
		f := &fakeTasks{labelErr: tc.err}
		rec := newTaskAPI(t, f).do(t, http.MethodDelete, "/api/v1/labels/"+uuid.NewString(), "", withToken("valid-admin"))
		if rec.Code != tc.status || errCode(t, rec) != tc.code {
			t.Errorf("%s: status %d code %s", tc.code, rec.Code, errCode(t, rec))
		}
	}
	f := &fakeTasks{settingEr: domain.ErrInvalidSetting}
	rec := newTaskAPI(t, f).do(t, http.MethodPut, "/api/v1/settings", `{"storyPointHours":99}`, withToken("valid-admin"))
	if rec.Code != http.StatusUnprocessableEntity || errCode(t, rec) != "SETTING_INVALID" {
		t.Fatalf("setting: %d %s", rec.Code, errCode(t, rec))
	}
}

// Quyền do service quyết định; handler phải trả 403 FORBIDDEN cho mọi thao tác ghi bị từ chối.
func TestTaskAPI_ForbiddenWrites_CF06(t *testing.T) {
	id := uuid.NewString()
	cases := []struct {
		name, method, path, body, token string
	}{
		{"Member sửa công việc", http.MethodPut, "/api/v1/tasks/" + id, `{"title":"x","labelIds":[]}`, "valid-member"},
		{"CEO sửa công việc", http.MethodPut, "/api/v1/tasks/" + id, `{"title":"x","labelIds":[]}`, "valid-ceo"},
		{"CEO nhận việc", http.MethodPost, "/api/v1/tasks/" + id + "/claim", "", "valid-ceo"},
		{"Manager nhận việc", http.MethodPost, "/api/v1/tasks/" + id + "/claim", "", "valid-manager"},
		{"Member tạo công việc", http.MethodPost, "/api/v1/projects/" + id + "/tasks", `{"title":"x","labelIds":[]}`, "valid-member"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeTasks{err: domain.ErrForbidden}
			rec := newTaskAPI(t, f).do(t, tc.method, tc.path, tc.body, withToken(tc.token))
			if rec.Code != http.StatusForbidden || errCode(t, rec) != "FORBIDDEN" {
				t.Fatalf("status %d code %s", rec.Code, errCode(t, rec))
			}
			if f.calls != 1 {
				t.Fatalf("service phải được gọi đúng 1 lần, gọi %d", f.calls)
			}
		})
	}
}
