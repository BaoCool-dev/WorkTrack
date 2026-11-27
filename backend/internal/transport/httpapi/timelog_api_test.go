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
		{domain.ErrInvalidLogMinutes, http.StatusBadRequest, "TIME_LOG_MINUTES_INVALID"},
		{domain.ErrInvalidLogNote, http.StatusBadRequest, "TIME_LOG_NOTE_INVALID"},
		{domain.ErrInvalidLogDate, http.StatusBadRequest, "TIME_LOG_DATE_INVALID"},
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
