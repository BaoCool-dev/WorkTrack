package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

type fakeKanban struct {
	err      error
	calls    int
	gotID    uuid.UUID
	gotInput kanban.MoveInput
}

func (f *fakeKanban) Board(_ context.Context, _ authz.Principal, pid uuid.UUID) ([]domain.Task, error) {
	f.calls++
	f.gotID = pid
	return []domain.Task{sampleTask(uuid.New())}, f.err
}

func (f *fakeKanban) Move(_ context.Context, _ authz.Principal, id uuid.UUID, in kanban.MoveInput) (domain.Task, error) {
	f.calls++
	f.gotID, f.gotInput = id, in
	t := sampleTask(id)
	t.ColumnID, t.Position, t.Version = in.ColumnID, in.Position, in.Version+1
	return t, f.err
}

// fakeHub ghi lại ai vào phòng nào; test đẩy sự kiện qua kênh ch.
type fakeHub struct {
	joined chan uuid.UUID
	ch     chan []byte
}

func newFakeHub() *fakeHub {
	return &fakeHub{joined: make(chan uuid.UUID, 1), ch: make(chan []byte, 1)}
}

func (h *fakeHub) Join(projectID, _ uuid.UUID) (<-chan []byte, func()) {
	h.joined <- projectID
	return h.ch, func() {}
}

func newKanbanAPI(t *testing.T, k *fakeKanban, p *fakeProjects) testAPI {
	t.Helper()
	return newTestAPI(t, func(d *Deps) { d.Kanban, d.Projects = k, p })
}

func TestKanbanAPI_Move_PassesInputAndReturnsTask_CF04(t *testing.T) {
	k := &fakeKanban{}
	a := newKanbanAPI(t, k, &fakeProjects{})
	tid, col := uuid.New(), uuid.New()

	rec := a.do(t, http.MethodPost, "/api/v1/tasks/"+tid.String()+"/move",
		`{"columnId":"`+col.String()+`","position":0,"version":3}`, withToken("valid-member"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body)
	}
	// position 0 phải được truyền (con trỏ *int để phân biệt "0" với "thiếu").
	if k.gotID != tid || k.gotInput != (kanban.MoveInput{ColumnID: col, Position: 0, Version: 3}) {
		t.Errorf("input = %+v", k.gotInput)
	}
	var body taskResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Version != 4 || body.Column.ID != col.String() {
		t.Errorf("body = %+v", body)
	}
}

func TestKanbanAPI_Move_BadRequests_CF04(t *testing.T) {
	col := uuid.NewString()
	for name, body := range map[string]string{
		"thiếu cột":      `{"position":0,"version":1}`,
		"thiếu position": `{"columnId":"` + col + `","version":1}`,
		"thiếu version":  `{"columnId":"` + col + `","position":0}`,
		"position âm":    `{"columnId":"` + col + `","position":-1,"version":1}`,
		"version 0":      `{"columnId":"` + col + `","position":0,"version":0}`,
	} {
		t.Run(name, func(t *testing.T) {
			k := &fakeKanban{}
			a := newKanbanAPI(t, k, &fakeProjects{})
			rec := a.do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/move", body, withToken("valid-member"))
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d; body=%s", rec.Code, rec.Body)
			}
			if k.calls != 0 {
				t.Error("service bị gọi với dữ liệu sai")
			}
		})
	}
}

func TestKanbanAPI_Move_ErrorMapping_CF04_CF05(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrTaskVersionConflict, http.StatusConflict, "TASK_VERSION_CONFLICT"},
		{domain.ErrWIPLimitReached, http.StatusConflict, "TASK_WIP_LIMIT"},
		{domain.ErrTransitionNotAllowed, http.StatusUnprocessableEntity, "TRANSITION_NOT_ALLOWED"},
		{domain.ErrMemberTransitionToDone, http.StatusUnprocessableEntity, "TRANSITION_MEMBER_TO_DONE"},
		{domain.ErrTaskNotAssignee, http.StatusForbidden, "TASK_NOT_ASSIGNEE"},
		{domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			a := newKanbanAPI(t, &fakeKanban{err: c.err}, &fakeProjects{})
			rec := a.do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/move",
				`{"columnId":"`+uuid.NewString()+`","position":1,"version":1}`, withToken("valid-member"))
			if rec.Code != c.status || errCode(t, rec) != c.code {
				t.Errorf("got %d %s; want %d %s", rec.Code, errCode(t, rec), c.status, c.code)
			}
		})
	}
}

func TestKanbanAPI_BoardTasks_CF04(t *testing.T) {
	k := &fakeKanban{}
	a := newKanbanAPI(t, k, &fakeProjects{})
	pid := uuid.New()
	rec := a.do(t, http.MethodGet, "/api/v1/projects/"+pid.String()+"/board/tasks", "", withToken("valid-member"))
	if rec.Code != http.StatusOK || k.gotID != pid {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"items":[`) || !strings.Contains(rec.Body.String(), `"version"`) {
		t.Errorf("body = %s", rec.Body)
	}
}

func TestKanbanAPI_SetWIPLimit_CF05(t *testing.T) {
	p := &fakeProjects{}
	a := newKanbanAPI(t, &fakeKanban{}, p)
	rec := a.do(t, http.MethodPut, "/api/v1/projects/"+uuid.NewString()+"/wip-limit", `{"wipLimit":7}`, withToken("valid-manager"))
	if rec.Code != http.StatusOK || p.gotWIP != 7 || !strings.Contains(rec.Body.String(), `"wipLimit":7`) {
		t.Fatalf("status = %d wip=%d body=%s", rec.Code, p.gotWIP, rec.Body)
	}

	p.err = domain.ErrInvalidWIPLimit
	rec = a.do(t, http.MethodPut, "/api/v1/projects/"+uuid.NewString()+"/wip-limit", `{"wipLimit":99}`, withToken("valid-manager"))
	if rec.Code != http.StatusUnprocessableEntity || errCode(t, rec) != "WIP_LIMIT_INVALID" {
		t.Errorf("status = %d; body=%s", rec.Code, rec.Body)
	}
}

func TestKanbanAPI_RequiresLogin_CF04(t *testing.T) {
	k := &fakeKanban{}
	a := newKanbanAPI(t, k, &fakeProjects{})
	rec := a.do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/move", `{}`)
	if rec.Code != http.StatusUnauthorized || k.calls != 0 {
		t.Errorf("status = %d calls=%d", rec.Code, k.calls)
	}
}

// ---- WebSocket ----

func newWSServer(t *testing.T, p *fakeProjects, h *fakeHub) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewRouter(Deps{
		DB: fakePinger{}, Auth: &fakeAuth{}, Accounts: &fakeAccounts{}, Org: &fakeOrg{}, Projects: p,
		Kanban: &fakeKanban{}, Hub: h, Tokens: fakeVerifier{}, Now: func() time.Time { return testNow },
	}))
	t.Cleanup(srv.Close)
	return srv
}

func wsURL(srv *httptest.Server, pid uuid.UUID, token string) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + "/api/v1/projects/" + pid.String() + "/ws?access_token=" + token
}

func TestWS_RejectsBeforeUpgrade_CF04(t *testing.T) {
	cases := []struct {
		name   string
		token  string
		projEr error
		status int
	}{
		{"không token", "", nil, http.StatusUnauthorized},
		{"token sai", "rác", nil, http.StatusUnauthorized},
		{"không thuộc dự án", "valid-member", domain.ErrProjectNotFound, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newFakeHub()
			srv := newWSServer(t, &fakeProjects{err: c.projEr}, h)
			conn, resp, err := websocket.Dial(t.Context(), wsURL(srv, uuid.New(), c.token), nil)
			if resp != nil && resp.Body != nil {
				defer func() { _ = resp.Body.Close() }()
			}
			if err == nil {
				_ = conn.CloseNow()
				t.Fatal("kết nối phải bị từ chối")
			}
			if resp == nil || resp.StatusCode != c.status {
				t.Fatalf("resp = %v, want %d", resp, c.status)
			}
			if len(h.joined) != 0 {
				t.Error("không được vào phòng khi bị từ chối")
			}
		})
	}
}

func TestWS_MemberReceivesProjectEvents_CF04(t *testing.T) {
	h := newFakeHub()
	srv := newWSServer(t, &fakeProjects{}, h)
	pid := uuid.New()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, wsURL(srv, pid, "valid-member"), nil)
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()

	if got := <-h.joined; got != pid {
		t.Fatalf("vào phòng %s, want %s", got, pid)
	}
	h.ch <- []byte(`{"type":"task.moved"}`)
	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != `{"type":"task.moved"}` {
		t.Errorf("msg = %s", msg)
	}
	_ = conn.Close(websocket.StatusNormalClosure, "")
}
