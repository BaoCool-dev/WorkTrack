package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// fakeReviews ghi lại hành động và input để test kiểm handler truyền đúng.
type fakeReviews struct {
	err      error
	calls    int
	action   string
	gotID    uuid.UUID
	gotInput kanban.ReviewInput
}

func (f *fakeReviews) decide(action string, id uuid.UUID, in kanban.ReviewInput) (domain.Task, error) {
	f.calls++
	f.action, f.gotID, f.gotInput = action, id, in
	t := sampleTask(id)
	t.Version = in.Version + 1
	return t, f.err
}

func (f *fakeReviews) Approve(_ context.Context, _ authz.Principal, id uuid.UUID, in kanban.ReviewInput) (domain.Task, error) {
	return f.decide("approve", id, in)
}

func (f *fakeReviews) Reject(_ context.Context, _ authz.Principal, id uuid.UUID, in kanban.ReviewInput) (domain.Task, error) {
	return f.decide("reject", id, in)
}

func (f *fakeReviews) Reviews(_ context.Context, _ authz.Principal, id uuid.UUID) ([]domain.TaskReview, error) {
	f.calls++
	f.action, f.gotID = "list", id
	reason := "Thiếu test biên"
	return []domain.TaskReview{{
		ID: uuid.New(), TaskID: id, ReviewerID: uuid.New(), ReviewerName: "Quản lý A",
		Decision: domain.ReviewRejected, Reason: &reason, CreatedAt: time.Now(),
	}}, f.err
}

func newReviewAPI(t *testing.T, f *fakeReviews) testAPI {
	t.Helper()
	return newTestAPI(t, func(d *Deps) { d.Reviews = f })
}

func TestReviewAPI_ApproveAndReject_PassInput_CF09(t *testing.T) {
	for _, c := range []struct {
		action, body string
		want         kanban.ReviewInput
	}{
		{"approve", `{"version":3}`, kanban.ReviewInput{Version: 3}},
		{"reject", `{"version":2,"reason":"Thiếu test biên"}`, kanban.ReviewInput{Version: 2, Reason: "Thiếu test biên"}},
	} {
		t.Run(c.action, func(t *testing.T) {
			f := &fakeReviews{}
			tid := uuid.New()
			rec := newReviewAPI(t, f).do(t, http.MethodPost, "/api/v1/tasks/"+tid.String()+"/review/"+c.action,
				c.body, withToken("valid-manager"))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; body=%s", rec.Code, rec.Body)
			}
			if f.action != c.action || f.gotID != tid || f.gotInput != c.want {
				t.Errorf("gọi %s %v %+v", f.action, f.gotID, f.gotInput)
			}
			var body taskResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Version != c.want.Version+1 {
				t.Errorf("body = %+v %v", body, err)
			}
		})
	}
}

func TestReviewAPI_List_ReturnsHistory_CF09(t *testing.T) {
	f := &fakeReviews{}
	tid := uuid.New()
	rec := newReviewAPI(t, f).do(t, http.MethodGet, "/api/v1/tasks/"+tid.String()+"/reviews", "", withToken("valid-member"))
	if rec.Code != http.StatusOK || f.gotID != tid {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body)
	}
	var body itemsResponse[reviewResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0].Decision != "rejected" || body.Items[0].Reason == nil ||
		*body.Items[0].Reason != "Thiếu test biên" || body.Items[0].ReviewerName != "Quản lý A" {
		t.Errorf("items = %+v", body.Items)
	}
}

func TestReviewAPI_BadRequests_CF09(t *testing.T) {
	for name, c := range map[string]struct{ path, body string }{
		"thiếu version": {"/api/v1/tasks/" + uuid.NewString() + "/review/approve", `{}`},
		"version 0":     {"/api/v1/tasks/" + uuid.NewString() + "/review/reject", `{"version":0,"reason":"x"}`},
		"JSON hỏng":     {"/api/v1/tasks/" + uuid.NewString() + "/review/approve", `{`},
		"id không phải": {"/api/v1/tasks/abc/review/approve", `{"version":1}`},
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeReviews{}
			rec := newReviewAPI(t, f).do(t, http.MethodPost, c.path, c.body, withToken("valid-manager"))
			if rec.Code != http.StatusBadRequest || f.calls != 0 {
				t.Errorf("status = %d calls=%d; body=%s", rec.Code, f.calls, rec.Body)
			}
		})
	}
}

func TestReviewAPI_ErrorMapping_CF09(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrTaskNotInReview, http.StatusConflict, "TASK_NOT_IN_REVIEW"},
		{domain.ErrTaskVersionConflict, http.StatusConflict, "TASK_VERSION_CONFLICT"},
		{domain.ErrInvalidReviewReason, http.StatusUnprocessableEntity, "REVIEW_REASON_INVALID"},
		{domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{domain.ErrTaskNotFound, http.StatusNotFound, "TASK_NOT_FOUND"},
	}
	for _, c := range cases {
		t.Run(c.code, func(t *testing.T) {
			rec := newReviewAPI(t, &fakeReviews{err: c.err}).do(t, http.MethodPost,
				"/api/v1/tasks/"+uuid.NewString()+"/review/reject", `{"version":1,"reason":""}`, withToken("valid-manager"))
			if rec.Code != c.status || errCode(t, rec) != c.code {
				t.Errorf("got %d %s; want %d %s", rec.Code, errCode(t, rec), c.status, c.code)
			}
		})
	}
}

func TestReviewAPI_RequiresLogin_CF09(t *testing.T) {
	for _, p := range []struct{ method, path string }{
		{http.MethodPost, "/review/approve"}, {http.MethodPost, "/review/reject"}, {http.MethodGet, "/reviews"},
	} {
		f := &fakeReviews{}
		rec := newReviewAPI(t, f).do(t, p.method, "/api/v1/tasks/"+uuid.NewString()+p.path, `{"version":1}`)
		if rec.Code != http.StatusUnauthorized || f.calls != 0 {
			t.Errorf("%s: status = %d calls=%d", p.path, rec.Code, f.calls)
		}
	}
}
