package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/subtask"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// fakeCollab giả cả ba service CF-07 để test tầng HTTP (giải mã, ánh xạ lỗi, định dạng JSON).
type fakeCollab struct {
	err        error
	gotID      uuid.UUID
	gotCol     uuid.UUID
	gotIn      subtask.Input
	gotBody    string
	gotUnread  bool
	gotPage    domain.PageRequest
	deleted    bool
	markedAll  bool
	markedRead uuid.UUID
}

func (f *fakeCollab) List(_ context.Context, _ authz.Principal, parentID uuid.UUID) ([]domain.Task, error) {
	f.gotID = parentID
	c := sampleTask(uuid.New())
	c.ParentID = &parentID
	return []domain.Task{c}, f.err
}

func (f *fakeCollab) Create(_ context.Context, _ authz.Principal, parentID uuid.UUID, in subtask.Input) (domain.Task, error) {
	f.gotID, f.gotIn = parentID, in
	c := sampleTask(uuid.New())
	c.ParentID, c.Title = &parentID, in.Title
	return c, f.err
}

func (f *fakeCollab) Move(_ context.Context, _ authz.Principal, id, columnID uuid.UUID) (domain.Task, error) {
	f.gotID, f.gotCol = id, columnID
	c := sampleTask(id)
	c.ColumnID = columnID
	return c, f.err
}

func (f *fakeCollab) Delete(_ context.Context, _ authz.Principal, id uuid.UUID) error {
	f.gotID, f.deleted = id, f.err == nil
	return f.err
}

type fakeComments struct{ *fakeCollab }

func (f fakeComments) List(_ context.Context, _ authz.Principal, taskID uuid.UUID, p domain.PageRequest) (domain.Page[domain.Comment], error) {
	f.gotID, f.gotPage = taskID, p
	c := domain.Comment{ID: uuid.New(), TaskID: taskID, AuthorID: uuid.New(), AuthorName: "An", Body: "Xin chào", CreatedAt: testNow}
	return domain.Page[domain.Comment]{Items: []domain.Comment{c}, Page: p.Page, PageSize: p.PageSize, Total: 1}, f.err
}

func (f fakeComments) Create(_ context.Context, _ authz.Principal, taskID uuid.UUID, body string) (domain.Comment, error) {
	f.gotID, f.gotBody = taskID, body
	return domain.Comment{ID: uuid.New(), TaskID: taskID, AuthorName: "An", Body: body, CreatedAt: testNow}, f.err
}

type fakeNotifs struct{ *fakeCollab }

func (f fakeNotifs) List(_ context.Context, _ authz.Principal, unreadOnly bool, p domain.PageRequest) (domain.Page[domain.Notification], error) {
	f.gotUnread, f.gotPage = unreadOnly, p
	actor := uuid.New()
	n := domain.Notification{
		ID: uuid.New(), Type: domain.NotificationTaskCommented, TaskID: uuid.New(), TaskTitle: "Việc A",
		ProjectID: uuid.New(), ProjectName: "Dự án", ActorID: &actor, ActorName: "An", Message: "Xin chào", CreatedAt: testNow,
	}
	return domain.Page[domain.Notification]{Items: []domain.Notification{n}, Page: p.Page, PageSize: p.PageSize, Total: 1}, f.err
}

func (f fakeNotifs) UnreadCount(_ context.Context, _ authz.Principal) (int, error) { return 3, f.err }

func (f fakeNotifs) MarkRead(_ context.Context, _ authz.Principal, id uuid.UUID) error {
	f.markedRead = id
	return f.err
}

func (f fakeNotifs) MarkAllRead(_ context.Context, _ authz.Principal) (int, error) {
	f.markedAll = true
	return 2, f.err
}

func newCollabAPI(t *testing.T, f *fakeCollab) testAPI {
	t.Helper()
	return newTestAPI(t, func(d *Deps) {
		d.Subtasks, d.Comments, d.Notifications = f, fakeComments{f}, fakeNotifs{f}
	})
}

func TestCollabAPI_Subtasks_CF07(t *testing.T) {
	f := &fakeCollab{}
	a := newCollabAPI(t, f)
	pid := uuid.New()

	rec := a.do(t, http.MethodPost, "/api/v1/tasks/"+pid.String()+"/subtasks", `{"title":"Viết test","description":"x"}`, withToken("valid-member"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d; body=%s", rec.Code, rec.Body)
	}
	var created taskResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if f.gotID != pid || f.gotIn.Title != "Viết test" || created.ParentID == nil || *created.ParentID != pid.String() {
		t.Errorf("create: id=%v in=%+v parent=%v", f.gotID, f.gotIn, created.ParentID)
	}
	// Story Point là trường lạ với việc con → 400 (DisallowUnknownFields), không lén ghi được.
	rec = a.do(t, http.MethodPost, "/api/v1/tasks/"+pid.String()+"/subtasks", `{"title":"A","storyPoint":3}`, withToken("valid-member"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("storyPoint trên con: %d", rec.Code)
	}

	rec = a.do(t, http.MethodGet, "/api/v1/tasks/"+pid.String()+"/subtasks", "", withToken("valid-member"))
	var list itemsResponse[taskResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || rec.Code != http.StatusOK || len(list.Items) != 1 {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}

	sid, col := uuid.New(), uuid.New()
	rec = a.do(t, http.MethodPost, "/api/v1/subtasks/"+sid.String()+"/move", `{"columnId":"`+col.String()+`"}`, withToken("valid-member"))
	if rec.Code != http.StatusOK || f.gotID != sid || f.gotCol != col {
		t.Errorf("move: %d id=%v col=%v", rec.Code, f.gotID, f.gotCol)
	}
	rec = a.do(t, http.MethodPost, "/api/v1/subtasks/"+sid.String()+"/move", `{}`, withToken("valid-member"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("move thiếu columnId: %d", rec.Code)
	}

	rec = a.do(t, http.MethodDelete, "/api/v1/subtasks/"+sid.String(), "", withToken("valid-member"))
	if rec.Code != http.StatusNoContent || !f.deleted {
		t.Errorf("delete: %d", rec.Code)
	}
	rec = a.do(t, http.MethodDelete, "/api/v1/subtasks/khong-phai-uuid", "", withToken("valid-member"))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("id sai: %d", rec.Code)
	}
}

func TestCollabAPI_ErrorMapping_CF07(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"không phải người nhận", domain.ErrSubtaskNotAllowed, http.StatusForbidden, "SUBTASK_NOT_ALLOWED"},
		{"cha là việc con", domain.ErrSubtaskParentNotRoot, http.StatusUnprocessableEntity, "SUBTASK_PARENT_NOT_ROOT"},
		{"ngoài dự án", domain.ErrTaskNotFound, http.StatusNotFound, "TASK_NOT_FOUND"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := newCollabAPI(t, &fakeCollab{err: tt.err}).do(t, http.MethodPost,
				"/api/v1/tasks/"+uuid.NewString()+"/subtasks", `{"title":"A"}`, withToken("valid-member"))
			if rec.Code != tt.status {
				t.Fatalf("status = %d; body=%s", rec.Code, rec.Body)
			}
			var body errorBody
			_ = json.Unmarshal(rec.Body.Bytes(), &body)
			if body.Error.Code != tt.code {
				t.Errorf("code = %q", body.Error.Code)
			}
		})
	}
	rec := newCollabAPI(t, &fakeCollab{err: domain.ErrInvalidComment}).do(t, http.MethodPost,
		"/api/v1/tasks/"+uuid.NewString()+"/comments", `{"body":"   "}`, withToken("valid-member"))
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("bình luận rỗng: %d", rec.Code)
	}
	rec = newCollabAPI(t, &fakeCollab{err: domain.ErrNotificationNotFound}).do(t, http.MethodPost,
		"/api/v1/me/notifications/"+uuid.NewString()+"/read", "", withToken("valid-member"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("thông báo của người khác: %d", rec.Code)
	}
}

func TestCollabAPI_Comments_CF07(t *testing.T) {
	f := &fakeCollab{}
	a := newCollabAPI(t, f)
	tid := uuid.New()

	rec := a.do(t, http.MethodPost, "/api/v1/tasks/"+tid.String()+"/comments", `{"body":"Đã xong phần API"}`, withToken("valid-member"))
	if rec.Code != http.StatusCreated || f.gotBody != "Đã xong phần API" {
		t.Fatalf("create: %d %s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodGet, "/api/v1/tasks/"+tid.String()+"/comments?page=2&pageSize=5", "", withToken("valid-ceo"))
	var page pageResponse[commentResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if f.gotPage.Page != 2 || f.gotPage.PageSize != 5 || page.Total != 1 || page.Items[0].Author.Name != "An" {
		t.Errorf("page = %+v body=%+v", f.gotPage, page)
	}
}

func TestCollabAPI_Notifications_CF07(t *testing.T) {
	f := &fakeCollab{}
	a := newCollabAPI(t, f)

	rec := a.do(t, http.MethodGet, "/api/v1/me/notifications?unread=true", "", withToken("valid-member"))
	var page pageResponse[notificationResponse]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("list: %d %s", rec.Code, rec.Body)
	}
	if !f.gotUnread || page.Items[0].Type != "task_commented" || page.Items[0].Read || page.Items[0].Task.Name != "Việc A" {
		t.Errorf("list = %+v", page.Items[0])
	}

	rec = a.do(t, http.MethodGet, "/api/v1/me/notifications/unread-count", "", withToken("valid-member"))
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"count\":3}\n" {
		t.Errorf("unread-count: %d %s", rec.Code, rec.Body)
	}

	nid := uuid.New()
	rec = a.do(t, http.MethodPost, "/api/v1/me/notifications/"+nid.String()+"/read", "", withToken("valid-member"))
	if rec.Code != http.StatusNoContent || f.markedRead != nid {
		t.Errorf("read: %d", rec.Code)
	}
	rec = a.do(t, http.MethodPost, "/api/v1/me/notifications/read-all", "", withToken("valid-member"))
	if rec.Code != http.StatusOK || !f.markedAll || rec.Body.String() != "{\"count\":2}\n" {
		t.Errorf("read-all: %d %s", rec.Code, rec.Body)
	}

	// Không đăng nhập → 401 (chuông là dữ liệu riêng của từng người).
	rec = a.do(t, http.MethodGet, "/api/v1/me/notifications", "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("không token: %d", rec.Code)
	}
}

// untouched: service giả chưa hề được gọi (tầng HTTP chặn trước khi tới use case).
func (f *fakeCollab) untouched() bool {
	return f.gotID == uuid.Nil && f.gotCol == uuid.Nil && f.gotBody == "" &&
		!f.deleted && !f.markedAll && f.markedRead == uuid.Nil && !f.gotUnread
}

// Đoán lỗi / EP lớp "chưa đăng nhập": mọi endpoint CF-07 trả 401 và không chạm service.
func TestCollabAPI_AllEndpointsRequireLogin_CF07(t *testing.T) {
	tid, sid, nid := uuid.NewString(), uuid.NewString(), uuid.NewString()
	for name, req := range map[string][3]string{
		"list subtasks":  {http.MethodGet, "/api/v1/tasks/" + tid + "/subtasks", ""},
		"create subtask": {http.MethodPost, "/api/v1/tasks/" + tid + "/subtasks", `{"title":"A"}`},
		"move subtask":   {http.MethodPost, "/api/v1/subtasks/" + sid + "/move", `{"columnId":"` + uuid.NewString() + `"}`},
		"delete subtask": {http.MethodDelete, "/api/v1/subtasks/" + sid, ""},
		"list comments":  {http.MethodGet, "/api/v1/tasks/" + tid + "/comments", ""},
		"create comment": {http.MethodPost, "/api/v1/tasks/" + tid + "/comments", `{"body":"Xin chào"}`},
		"list notifs":    {http.MethodGet, "/api/v1/me/notifications", ""},
		"unread-count":   {http.MethodGet, "/api/v1/me/notifications/unread-count", ""},
		"mark read":      {http.MethodPost, "/api/v1/me/notifications/" + nid + "/read", ""},
		"mark all read":  {http.MethodPost, "/api/v1/me/notifications/read-all", ""},
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeCollab{}
			rec := newCollabAPI(t, f).do(t, req[0], req[1], req[2])
			if rec.Code != http.StatusUnauthorized || !f.untouched() {
				t.Errorf("status = %d untouched=%v body=%s", rec.Code, f.untouched(), rec.Body)
			}
		})
	}
}

// EP lớp không hợp lệ cho tham số đường dẫn, thân JSON và query: 400, không chạm service.
func TestCollabAPI_BadPathBodyQuery_CF07(t *testing.T) {
	col := uuid.NewString()
	cases := map[string]struct {
		method, path, body, code string
	}{
		"list subtasks id sai":  {http.MethodGet, "/api/v1/tasks/khong-phai-uuid/subtasks", "", "TASK_ID_INVALID"},
		"create subtask id sai": {http.MethodPost, "/api/v1/tasks/123/subtasks", `{"title":"A"}`, "TASK_ID_INVALID"},
		"move id sai":           {http.MethodPost, "/api/v1/subtasks/abc/move", `{"columnId":"` + col + `"}`, "SUBTASK_ID_INVALID"},
		"move cột sai UUID":     {http.MethodPost, "/api/v1/subtasks/" + uuid.NewString() + "/move", `{"columnId":"abc"}`, ""},
		"list comments id sai":  {http.MethodGet, "/api/v1/tasks/xyz/comments", "", "TASK_ID_INVALID"},
		"create comment id sai": {http.MethodPost, "/api/v1/tasks/xyz/comments", `{"body":"A"}`, "TASK_ID_INVALID"},
		"comment JSON hỏng":     {http.MethodPost, "/api/v1/tasks/" + uuid.NewString() + "/comments", `{"body":`, ""},
		"comment trường lạ":     {http.MethodPost, "/api/v1/tasks/" + uuid.NewString() + "/comments", `{"body":"A","authorId":"x"}`, ""},
		"comment sai kiểu":      {http.MethodPost, "/api/v1/tasks/" + uuid.NewString() + "/comments", `{"body":123}`, ""},
		"mark read id sai":      {http.MethodPost, "/api/v1/me/notifications/khong-phai-uuid/read", "", "NOTIFICATION_ID_INVALID"},
		"unread ngoài tập":      {http.MethodGet, "/api/v1/me/notifications?unread=yes", "", "QUERY_INVALID"},
		"subtask tiêu đề rỗng":  {http.MethodPost, "/api/v1/tasks/" + uuid.NewString() + "/subtasks", `{"title":""}`, ""},
		"subtask tiêu đề 201":   {http.MethodPost, "/api/v1/tasks/" + uuid.NewString() + "/subtasks", `{"title":"` + strings.Repeat("a", 201) + `"}`, ""},
		"subtask mô tả 5001":    {http.MethodPost, "/api/v1/tasks/" + uuid.NewString() + "/subtasks", `{"title":"A","description":"` + strings.Repeat("a", 5001) + `"}`, ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			f := &fakeCollab{}
			rec := newCollabAPI(t, f).do(t, c.method, c.path, c.body, withToken("valid-member"))
			if rec.Code != http.StatusBadRequest || !f.untouched() {
				t.Fatalf("status = %d untouched=%v body=%s", rec.Code, f.untouched(), rec.Body)
			}
			if c.code != "" && errCode(t, rec) != c.code {
				t.Errorf("code = %q, muốn %q", errCode(t, rec), c.code)
			}
		})
	}
}

// BVA tiêu đề việc con: 200 ký tự (biên trên, kể cả tiếng Việt có dấu) vẫn hợp lệ.
func TestCollabAPI_SubtaskTitleBoundary_CF07(t *testing.T) {
	for name, title := range map[string]string{
		"1 ký tự":          "A",
		"200 ASCII":        strings.Repeat("a", 200),
		"200 ký tự có dấu": strings.Repeat("ệ", 200),
	} {
		t.Run(name, func(t *testing.T) {
			f := &fakeCollab{}
			rec := newCollabAPI(t, f).do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/subtasks",
				`{"title":"`+title+`"}`, withToken("valid-member"))
			if rec.Code != http.StatusCreated || f.gotIn.Title != title {
				t.Errorf("status = %d body=%s", rec.Code, rec.Body)
			}
		})
	}
}

// Bảng quyết định vai trò × bình luận: service trả ErrForbidden (CEO/Admin chỉ đọc) → 403,
// người ngoài dự án → 404 (không lộ công việc tồn tại).
func TestCollabAPI_CommentErrorMapping_CF07(t *testing.T) {
	for name, tt := range map[string]struct {
		err    error
		status int
		code   string
	}{
		"CEO/Admin viết": {domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		"ngoài dự án":    {domain.ErrTaskNotFound, http.StatusNotFound, "TASK_NOT_FOUND"},
		"quá 2000 ký tự": {domain.ErrInvalidComment, http.StatusUnprocessableEntity, "COMMENT_INVALID"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := newCollabAPI(t, &fakeCollab{err: tt.err}).do(t, http.MethodPost,
				"/api/v1/tasks/"+uuid.NewString()+"/comments", `{"body":"Xin chào"}`, withToken("valid-ceo"))
			if rec.Code != tt.status || errCode(t, rec) != tt.code {
				t.Errorf("status = %d body=%s", rec.Code, rec.Body)
			}
		})
	}
	rec := newCollabAPI(t, &fakeCollab{err: domain.ErrTaskNotFound}).do(t, http.MethodGet,
		"/api/v1/tasks/"+uuid.NewString()+"/comments", "", withToken("valid-member"))
	if rec.Code != http.StatusNotFound {
		t.Errorf("đọc bình luận ngoài dự án: %d", rec.Code)
	}
}

// Đoán lỗi XSS: API lưu và trả nguyên văn bình luận chứa HTML (không tự chèn/biến đổi),
// JSON mã hóa "<" ">" thành < > nên không thể bị trình duyệt hiểu là thẻ nếu
// phản hồi bị mở nhầm như HTML. Việc hiển thị an toàn do FE (React escape) — xem collab.
func TestCollabAPI_CommentWithHTMLRoundTrip_CF07(t *testing.T) {
	f := &fakeCollab{}
	payload := `<img src=x onerror=alert(1)><script>alert("xss")</script>`
	reqBody, _ := json.Marshal(map[string]string{"body": payload})
	rec := newCollabAPI(t, f).do(t, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/comments", string(reqBody), withToken("valid-member"))
	if rec.Code != http.StatusCreated || f.gotBody != payload {
		t.Fatalf("status = %d got=%q", rec.Code, f.gotBody)
	}
	if strings.Contains(rec.Body.String(), "<script>") {
		t.Errorf("phản hồi chứa thẻ thô: %s", rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q", ct)
	}
	var out commentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || out.Body != payload {
		t.Errorf("body = %q err=%v", out.Body, err)
	}
}
