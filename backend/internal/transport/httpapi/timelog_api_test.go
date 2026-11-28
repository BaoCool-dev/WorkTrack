package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/timelog"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// fakeTimeLogs giả timelog.Service để test tầng HTTP: giải mã, ánh xạ lỗi, định dạng JSON (CF-08).
type fakeTimeLogs struct {
	err     error
	called  bool
	gotID   uuid.UUID
	gotIn   timelog.Input
	gotPage domain.PageRequest
}

func (f *fakeTimeLogs) sample(taskID uuid.UUID, in timelog.Input) domain.TimeLog {
	parent := uuid.New()
	return domain.TimeLog{
		ID: uuid.New(), TaskID: taskID, TaskTitle: "Viết API", ParentID: &parent, ProjectID: uuid.New(),
		UserID: uuid.New(), UserName: "An", Minutes: in.Minutes, Note: in.Note,
		WorkDate: time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC), CreatedAt: testNow,
	}
}

func (f *fakeTimeLogs) List(_ context.Context, _ authz.Principal, taskID uuid.UUID, p domain.PageRequest) (domain.Page[domain.TimeLog], error) {
	f.called, f.gotID, f.gotPage = true, taskID, p
	l := f.sample(taskID, timelog.Input{Minutes: 90, Note: "Làm API"})
	return domain.Page[domain.TimeLog]{Items: []domain.TimeLog{l}, Page: p.Page, PageSize: p.PageSize, Total: 1}, f.err
}

func (f *fakeTimeLogs) Create(_ context.Context, _ authz.Principal, taskID uuid.UUID, in timelog.Input) (domain.TimeLog, error) {
	f.called, f.gotID, f.gotIn = true, taskID, in
	return f.sample(taskID, in), f.err
}

func (f *fakeTimeLogs) Delete(_ context.Context, _ authz.Principal, id uuid.UUID) error {
	f.called, f.gotID = true, id
	return f.err
}

func newTimeLogAPI(t *testing.T, f *fakeTimeLogs) testAPI {
	t.Helper()
	return newTestAPI(t, func(d *Deps) { d.TimeLogs = f })
}

func TestTimeLogAPI_CreateListDelete_CF08(t *testing.T) {
	f := &fakeTimeLogs{}
	a := newTimeLogAPI(t, f)
	tid := uuid.New()

	rec := a.do(t, http.MethodPost, "/api/v1/tasks/"+tid.String()+"/time-logs",
		`{"minutes":90,"note":"Viết handler","workDate":"2026-11-26"}`, withToken("valid-member"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", rec.Code, rec.Body)
	}
	if f.gotID != tid || f.gotIn.Minutes != 90 || f.gotIn.Note != "Viết handler" ||
		f.gotIn.WorkDate == nil || f.gotIn.WorkDate.Format(time.DateOnly) != "2026-11-26" {
		t.Errorf("create input: id=%v in=%+v", f.gotID, f.gotIn)
	}
	var created timeLogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Minutes != 90 || created.WorkDate != "2026-11-27" || created.User.Name != "An" || created.ParentID == nil {
		t.Errorf("create body: %+v", created)
	}

	// Không gửi workDate → service tự lấy hôm nay (giờ VN).
	rec = a.do(t, http.MethodPost, "/api/v1/tasks/"+tid.String()+"/time-logs", `{"minutes":30,"note":"x"}`, withToken("valid-member"))
	if rec.Code != http.StatusCreated || f.gotIn.WorkDate != nil {
		t.Errorf("không workDate: %d, %v", rec.Code, f.gotIn.WorkDate)
	}

	rec = a.do(t, http.MethodGet, "/api/v1/tasks/"+tid.String()+"/time-logs?page=2&pageSize=5", "", withToken("valid-ceo"))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d", rec.Code)
	}
	var page pageResponse[timeLogResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Minutes != 90 || f.gotPage.Page != 2 || f.gotPage.PageSize != 5 {
		t.Errorf("list: %+v page=%+v", page, f.gotPage)
	}

	lid := uuid.New()
	rec = a.do(t, http.MethodDelete, "/api/v1/time-logs/"+lid.String(), "", withToken("valid-member"))
	if rec.Code != http.StatusNoContent || f.gotID != lid {
		t.Errorf("delete: %d id=%v", rec.Code, f.gotID)
	}
}

func TestTimeLogAPI_ErrorMapping_CF08(t *testing.T) {
	tests := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrInvalidLogMinutes, http.StatusUnprocessableEntity, "TIME_LOG_MINUTES_INVALID"},
		{domain.ErrInvalidLogNote, http.StatusUnprocessableEntity, "TIME_LOG_NOTE_INVALID"},
		{domain.ErrInvalidLogDate, http.StatusUnprocessableEntity, "TIME_LOG_DATE_INVALID"},
		{domain.ErrTimeLogClosed, http.StatusConflict, "TIME_LOG_CLOSED"},
		{domain.ErrTimeLogNotAllowed, http.StatusForbidden, "TIME_LOG_NOT_ALLOWED"},
		{domain.ErrTaskNotFound, http.StatusNotFound, "TASK_NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			rec := newTimeLogAPI(t, &fakeTimeLogs{err: tt.err}).do(t, http.MethodPost,
				"/api/v1/tasks/"+uuid.NewString()+"/time-logs", `{"minutes":30,"note":"x"}`, withToken("valid-member"))
			if rec.Code != tt.status || errCode(t, rec) != tt.code {
				t.Errorf("got %d %s; want %d %s", rec.Code, errCode(t, rec), tt.status, tt.code)
			}
		})
	}
	rec := newTimeLogAPI(t, &fakeTimeLogs{err: domain.ErrTimeLogNotFound}).do(t, http.MethodDelete,
		"/api/v1/time-logs/"+uuid.NewString(), "", withToken("valid-member"))
	if rec.Code != http.StatusNotFound || errCode(t, rec) != "TIME_LOG_NOT_FOUND" {
		t.Errorf("delete not found: %d %s", rec.Code, errCode(t, rec))
	}
}

func TestTimeLogAPI_RequireLogin_CF08(t *testing.T) {
	for _, req := range [][3]string{
		{http.MethodGet, "/api/v1/tasks/" + uuid.NewString() + "/time-logs", ""},
		{http.MethodPost, "/api/v1/tasks/" + uuid.NewString() + "/time-logs", `{"minutes":30,"note":"x"}`},
		{http.MethodDelete, "/api/v1/time-logs/" + uuid.NewString(), ""},
	} {
		f := &fakeTimeLogs{}
		rec := newTimeLogAPI(t, f).do(t, req[0], req[1], req[2])
		if rec.Code != http.StatusUnauthorized || f.called {
			t.Errorf("%s %s: %d called=%v", req[0], req[1], rec.Code, f.called)
		}
	}
}

func TestTimeLogAPI_BadInput_CF08(t *testing.T) {
	tid := uuid.NewString()
	cases := []struct{ name, method, path, body, code string }{
		{"task id", http.MethodPost, "/api/v1/tasks/x/time-logs", `{"minutes":30,"note":"x"}`, "TASK_ID_INVALID"},
		{"log id", http.MethodDelete, "/api/v1/time-logs/x", "", "TIME_LOG_ID_INVALID"},
		{"json", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":`, "INVALID_JSON"},
		{"trường lạ", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":30,"note":"x","userId":"a"}`, "INVALID_JSON"},
		{"phút dạng chữ", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":"30","note":"x"}`, "INVALID_JSON"},
		{"ngày sai", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":30,"note":"x","workDate":"27/11/2026"}`, "DATE_INVALID"},
		{"page", http.MethodGet, "/api/v1/tasks/" + tid + "/time-logs?page=abc", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fakeTimeLogs{}
			rec := newTimeLogAPI(t, f).do(t, c.method, c.path, c.body, withToken("valid-member"))
			if rec.Code != http.StatusBadRequest || f.called {
				t.Errorf("got %d called=%v body=%s", rec.Code, f.called, rec.Body)
			}
			if c.code != "" && errCode(t, rec) != c.code {
				t.Errorf("code = %s; want %s", errCode(t, rec), c.code)
			}
		})
	}
}

// QA bổ sung — 401 khi token sai/không đúng dạng Bearer (không chỉ thiếu token) [CF-08].
func TestTimeLogAPI_InvalidToken_Returns401_CF08(t *testing.T) {
	for _, tok := range []string{"bad-token", "valid-boss"} {
		f := &fakeTimeLogs{}
		rec := newTimeLogAPI(t, f).do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/time-logs",
			`{"minutes":30,"note":"x"}`, withToken(tok))
		if rec.Code != http.StatusUnauthorized || f.called {
			t.Errorf("token %q: %d called=%v", tok, rec.Code, f.called)
		}
	}
}

// QA bổ sung — ánh xạ lỗi theo từng endpoint: vai trò chỉ xem (CEO/Admin) → 403 FORBIDDEN,
// IDOR (công việc/log của dự án khác) → 404, xóa log người khác → 403, xóa khi đã khóa → 409 [CF-08].
func TestTimeLogAPI_ErrorMappingPerEndpoint_CF08(t *testing.T) {
	tid, lid := uuid.NewString(), uuid.NewString()
	cases := []struct {
		name, method, path, body, tok string
		err                           error
		status                        int
		code                          string
	}{
		{"CEO ghi giờ", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":30,"note":"x"}`, "valid-ceo", domain.ErrForbidden, 403, "FORBIDDEN"},
		{"Admin ghi giờ (không xem được dự án)", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":30,"note":"x"}`, "valid-admin", domain.ErrTaskNotFound, 404, "TASK_NOT_FOUND"},
		{"Member khác ghi giờ", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":30,"note":"x"}`, "valid-member", domain.ErrTimeLogNotAllowed, 403, "TIME_LOG_NOT_ALLOWED"},
		{"IDOR xem lịch sử dự án khác", http.MethodGet, "/api/v1/tasks/" + tid + "/time-logs", "", "valid-member", domain.ErrTaskNotFound, 404, "TASK_NOT_FOUND"},
		{"IDOR xóa log dự án khác", http.MethodDelete, "/api/v1/time-logs/" + lid, "", "valid-member", domain.ErrTimeLogNotFound, 404, "TIME_LOG_NOT_FOUND"},
		{"xóa log của người khác", http.MethodDelete, "/api/v1/time-logs/" + lid, "", "valid-manager", domain.ErrTimeLogNotAllowed, 403, "TIME_LOG_NOT_ALLOWED"},
		{"xóa khi việc Done quá 24 giờ", http.MethodDelete, "/api/v1/time-logs/" + lid, "", "valid-member", domain.ErrTimeLogClosed, 409, "TIME_LOG_CLOSED"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := newTimeLogAPI(t, &fakeTimeLogs{err: c.err}).do(t, c.method, c.path, c.body, withToken(c.tok))
			if rec.Code != c.status || errCode(t, rec) != c.code {
				t.Errorf("got %d %s; want %d %s", rec.Code, errCode(t, rec), c.status, c.code)
			}
		})
	}
}

// QA bổ sung — đầu vào sai ở tầng HTTP: UUID sai ở GET, phút số thực, ngày không tồn tại → 400, không gọi service [CF-08].
func TestTimeLogAPI_BadInput_More_CF08(t *testing.T) {
	tid := uuid.NewString()
	cases := []struct{ name, method, path, body, code string }{
		{"task id sai ở GET", http.MethodGet, "/api/v1/tasks/123/time-logs", "", "TASK_ID_INVALID"},
		{"UUID thiếu ký tự", http.MethodPost, "/api/v1/tasks/" + tid[:35] + "/time-logs", `{"minutes":30,"note":"x"}`, "TASK_ID_INVALID"},
		{"log id SQL injection", http.MethodDelete, "/api/v1/time-logs/1%27%20OR%20%271%27=%271", "", "TIME_LOG_ID_INVALID"},
		{"phút số thực 1.5", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":1.5,"note":"x"}`, "INVALID_JSON"},
		{"phút null", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":null,"note":"x"}`, ""},
		{"ngày 30/02 không tồn tại", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":30,"note":"x","workDate":"2026-02-30"}`, "DATE_INVALID"},
		{"ngày kèm giờ", http.MethodPost, "/api/v1/tasks/" + tid + "/time-logs", `{"minutes":30,"note":"x","workDate":"2026-11-27T10:00:00Z"}`, "DATE_INVALID"},
		{"pageSize không phải số", http.MethodGet, "/api/v1/tasks/" + tid + "/time-logs?pageSize=x", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fakeTimeLogs{}
			rec := newTimeLogAPI(t, f).do(t, c.method, c.path, c.body, withToken("valid-member"))
			if c.name == "phút null" {
				// null giải mã thành 0 → để service trả TIME_LOG_MINUTES_INVALID; ở đây chỉ kiểm handler chuyển 0 đi.
				if !f.called || f.gotIn.Minutes != 0 {
					t.Errorf("phút null: called=%v in=%+v", f.called, f.gotIn)
				}
				return
			}
			if rec.Code != http.StatusBadRequest || f.called {
				t.Errorf("got %d called=%v body=%s", rec.Code, f.called, rec.Body)
			}
			if c.code != "" && errCode(t, rec) != c.code {
				t.Errorf("code = %s; want %s", errCode(t, rec), c.code)
			}
		})
	}
}

// QA bổ sung — phút 0/âm/1441 không bị handler chặn sớm: chuyển nguyên cho service (domain là nơi kiểm),
// và workDate rỗng "" coi như không gửi (hôm nay) [CF-08].
func TestTimeLogAPI_PassesRawMinutesAndEmptyDate_CF08(t *testing.T) {
	for _, m := range []int{0, -30, 1441} {
		f := &fakeTimeLogs{err: domain.ErrInvalidLogMinutes}
		body, _ := json.Marshal(map[string]any{"minutes": m, "note": "x"})
		rec := newTimeLogAPI(t, f).do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/time-logs", string(body), withToken("valid-member"))
		if !f.called || f.gotIn.Minutes != m || rec.Code != http.StatusBadRequest || errCode(t, rec) != "TIME_LOG_MINUTES_INVALID" {
			t.Errorf("minutes=%d: called=%v in=%+v status=%d", m, f.called, f.gotIn, rec.Code)
		}
	}
	f := &fakeTimeLogs{}
	rec := newTimeLogAPI(t, f).do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/time-logs",
		`{"minutes":30,"note":"x","workDate":""}`, withToken("valid-member"))
	if rec.Code != http.StatusCreated || f.gotIn.WorkDate != nil {
		t.Errorf("workDate rỗng: %d %v", rec.Code, f.gotIn.WorkDate)
	}
}

// QA bổ sung — mô tả tiếng Việt + emoji + HTML đi khứ hồi nguyên văn qua JSON [CF-08].
func TestTimeLogAPI_NoteRoundTrip_CF08(t *testing.T) {
	note := `Sửa lỗi 🐛 <script>alert("x")</script>`
	f := &fakeTimeLogs{}
	body, _ := json.Marshal(map[string]any{"minutes": 30, "note": note})
	rec := newTimeLogAPI(t, f).do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/time-logs", string(body), withToken("valid-member"))
	if rec.Code != http.StatusCreated || f.gotIn.Note != note {
		t.Fatalf("status=%d note vào service=%q", rec.Code, f.gotIn.Note)
	}
	var got timeLogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Note != note {
		t.Errorf("note trả về = %q", got.Note)
	}
}
