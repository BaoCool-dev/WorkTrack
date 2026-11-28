package kanban_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// fakeReviewStore thêm cột, nghiệm thu và thông báo lên fakeStore của bảng Kanban.
type fakeReviewStore struct {
	*fakeStore
	cols     []domain.BoardColumn
	reviews  []kanban.ReviewRecord
	notes    []domain.NewNotification
	failNote bool
}

func (f *fakeReviewStore) InTx(_ context.Context, fn func(kanban.ReviewStore) error) error {
	snap := map[uuid.UUID]domain.Task{}
	for k, v := range f.tasks {
		snap[k] = v
	}
	h, r, n := len(f.history), len(f.reviews), len(f.notes)
	if err := fn(f); err != nil {
		f.tasks, f.history, f.reviews, f.notes = snap, f.history[:h], f.reviews[:r], f.notes[:n]
		return err
	}
	return nil
}

func (f *fakeReviewStore) ListColumns(_ context.Context, _ uuid.UUID) ([]domain.BoardColumn, error) {
	return f.cols, nil
}

func (f *fakeReviewStore) InsertReview(_ context.Context, r kanban.ReviewRecord) error {
	f.reviews = append(f.reviews, r)
	return nil
}

func (f *fakeReviewStore) ListReviews(_ context.Context, taskID uuid.UUID) ([]domain.TaskReview, error) {
	out := []domain.TaskReview{}
	for _, r := range f.reviews {
		if r.TaskID == taskID {
			out = append(out, domain.TaskReview{TaskID: r.TaskID, ReviewerID: r.ReviewerID, Decision: r.Decision, Reason: r.Reason})
		}
	}
	return out, nil
}

func (f *fakeReviewStore) AddNotification(_ context.Context, n domain.NewNotification) error {
	if f.failNote {
		return errBoom
	}
	f.notes = append(f.notes, n)
	return nil
}

var errBoom = errors.New("boom")

type reviewEnv struct {
	*env
	rs  *fakeReviewStore
	svc *kanban.ReviewService
}

func setupReview(t *testing.T) *reviewEnv {
	t.Helper()
	e := setup(t)
	// Từ chối = Manager kéo Review → Doing; luật này có trong DefaultTransitions.
	e.st.rules[transition{e.review, e.doing, domain.TransitionManager}] = true
	rs := &fakeReviewStore{fakeStore: e.st, cols: []domain.BoardColumn{
		{ID: e.backlog, StatusGroup: domain.GroupTodo, Position: 0},
		{ID: e.doing, StatusGroup: domain.GroupInProgress, Position: 1},
		{ID: e.review, StatusGroup: domain.GroupInProgress, Position: 2},
		{ID: e.done, StatusGroup: domain.GroupDone, Position: 3},
	}}
	pr := &fakeProjects{
		projects: e.st.projects,
		members:  map[uuid.UUID]map[uuid.UUID]bool{e.project.ID: {e.mgr.UserID: true, e.member.UserID: true, e.other.UserID: true}},
	}
	return &reviewEnv{env: e, rs: rs, svc: kanban.NewReviewService(rs, rs, pr, e.pub)}
}

// addReview thêm công việc cha đang chờ nghiệm thu, người tạo là Manager.
func (e *reviewEnv) addReview() domain.Task {
	t := e.add(e.review, &e.member)
	t.CreatedBy = e.mgr.UserID
	e.st.tasks[t.ID] = t
	return t
}

func (e *reviewEnv) approve(actor authz.Principal, t domain.Task) (domain.Task, error) {
	return e.svc.Approve(context.Background(), actor, t.ID, kanban.ReviewInput{Version: t.Version})
}

func (e *reviewEnv) reject(actor authz.Principal, t domain.Task, reason string) (domain.Task, error) {
	return e.svc.Reject(context.Background(), actor, t.ID, kanban.ReviewInput{Version: t.Version, Reason: reason})
}

func TestApprove_ManagerMovesToDoneAndRecords_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()

	got, err := e.approve(e.mgr, tk)
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if got.ColumnID != e.done || got.StatusGroup != domain.GroupDone || got.Version != tk.Version+1 {
		t.Fatalf("got column %v group %s version %d", got.ColumnID, got.StatusGroup, got.Version)
	}
	want := kanban.ReviewRecord{TaskID: tk.ID, ProjectID: e.project.ID, ReviewerID: e.mgr.UserID,
		Decision: domain.ReviewApproved, FromColumnID: e.review, ToColumnID: e.done}
	if len(e.rs.reviews) != 1 || e.rs.reviews[0] != want {
		t.Fatalf("reviews = %+v", e.rs.reviews)
	}
	if len(e.st.history) != 1 || e.st.history[0].ToColumnID != e.done {
		t.Fatalf("move history = %+v", e.st.history)
	}
	if len(e.rs.notes) != 0 {
		t.Fatalf("approve must not notify, got %+v", e.rs.notes)
	}
	if len(e.pub.events) != 1 || e.pub.events[0].Type != domain.EventTaskMoved ||
		e.pub.events[0].FromColumnID != e.review || e.pub.events[0].ToColumnID != e.done {
		t.Fatalf("events = %+v", e.pub.events)
	}
}

func TestReject_MovesBackWithReasonAndNotifies_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()

	got, err := e.reject(e.mgr, tk, "  Thiếu test biên\nbổ sung giúp  ")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if got.ColumnID != e.doing || got.StatusGroup != domain.GroupInProgress || got.Position != 0 {
		t.Fatalf("got column %v group %s pos %d", got.ColumnID, got.StatusGroup, got.Position)
	}
	if len(e.rs.reviews) != 1 {
		t.Fatalf("reviews = %+v", e.rs.reviews)
	}
	r := e.rs.reviews[0]
	if r.Decision != domain.ReviewRejected || r.Reason == nil || *r.Reason != "Thiếu test biên\nbổ sung giúp" ||
		r.FromColumnID != e.review || r.ToColumnID != e.doing {
		t.Fatalf("review = %+v", r)
	}
	// Manager là người tạo nên chỉ người nhận việc được báo.
	if len(e.rs.notes) != 1 || e.rs.notes[0].Type != domain.NotificationReviewRejected ||
		len(e.rs.notes[0].UserIDs) != 1 || e.rs.notes[0].UserIDs[0] != e.member.UserID ||
		e.rs.notes[0].Message != "Thiếu test biên bổ sung giúp" {
		t.Fatalf("notes = %+v", e.rs.notes)
	}
}

func TestReject_InvalidReason_CF09(t *testing.T) {
	for name, reason := range map[string]string{
		"rỗng":         "",
		"khoảng trắng": "   \n\t",
		"501 ký tự":    strings.Repeat("ạ", domain.ReviewReasonMaxRunes+1),
	} {
		t.Run(name, func(t *testing.T) {
			e := setupReview(t)
			tk := e.addReview()
			_, err := e.reject(e.mgr, tk, reason)
			expectErr(t, err, domain.ErrInvalidReviewReason)
			if e.st.tasks[tk.ID].ColumnID != e.review || len(e.rs.reviews) != 0 {
				t.Fatalf("task must stay in review")
			}
		})
	}
	t.Run("đúng 500 ký tự", func(t *testing.T) {
		e := setupReview(t)
		if _, err := e.reject(e.mgr, e.addReview(), strings.Repeat("ạ", domain.ReviewReasonMaxRunes)); err != nil {
			t.Fatalf("reject 500: %v", err)
		}
	})
}

func TestReview_OnlyResponsibleManager_CF09(t *testing.T) {
	e := setupReview(t)
	otherMgr := authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	pr := &fakeProjects{projects: e.st.projects, members: map[uuid.UUID]map[uuid.UUID]bool{
		e.project.ID: {e.mgr.UserID: true, e.member.UserID: true, otherMgr.UserID: true},
	}}
	svc := kanban.NewReviewService(e.rs, e.rs, pr, e.pub)
	tk := e.addReview()

	cases := map[string]struct {
		actor authz.Principal
		want  error
	}{
		"member được giao":  {e.member, domain.ErrForbidden},
		"manager khác":      {otherMgr, domain.ErrForbidden},
		"ceo chỉ xem":       {e.ceo, domain.ErrForbidden},
		"người ngoài dự án": {e.outsider, domain.ErrTaskNotFound},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := svc.Approve(context.Background(), c.actor, tk.ID, kanban.ReviewInput{Version: tk.Version})
			expectErr(t, err, c.want)
			_, err = svc.Reject(context.Background(), c.actor, tk.ID, kanban.ReviewInput{Version: tk.Version, Reason: "x"})
			expectErr(t, err, c.want)
		})
	}
	if e.st.tasks[tk.ID].ColumnID != e.review || len(e.rs.reviews) != 0 {
		t.Fatalf("task must be unchanged")
	}
}

func TestReview_ManagerChangedBeforeLock_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()
	// fakeProjects đọc bản cũ; dòng dự án đã khóa cho thấy Manager phụ trách mới.
	p := e.st.projects[e.project.ID]
	e.st.projects[e.project.ID] = domain.Project{ID: p.ID, ManagerID: uuid.New(), WIPLimit: p.WIPLimit}
	pr := &fakeProjects{projects: map[uuid.UUID]domain.Project{p.ID: p},
		members: map[uuid.UUID]map[uuid.UUID]bool{p.ID: {e.mgr.UserID: true}}}
	svc := kanban.NewReviewService(e.rs, e.rs, pr, e.pub)

	_, err := svc.Approve(context.Background(), e.mgr, tk.ID, kanban.ReviewInput{Version: tk.Version})
	expectErr(t, err, domain.ErrForbidden)
}

func TestReview_TaskNotInReviewColumn_CF09(t *testing.T) {
	e := setupReview(t)
	for name, col := range map[string]uuid.UUID{"backlog": e.backlog, "đang làm": e.doing, "done": e.done} {
		t.Run(name, func(t *testing.T) {
			tk := e.add(col, &e.member)
			_, err := e.approve(e.mgr, tk)
			expectErr(t, err, domain.ErrTaskNotInReview)
			_, err = e.reject(e.mgr, tk, "lý do")
			expectErr(t, err, domain.ErrTaskNotInReview)
		})
	}
	if len(e.rs.reviews) != 0 {
		t.Fatalf("reviews = %+v", e.rs.reviews)
	}
}

func TestReview_BoardWithoutReviewStep_CF09(t *testing.T) {
	e := setupReview(t)
	e.rs.cols = e.rs.cols[:2] // chỉ Backlog + Doing: không có cột done/chờ nghiệm thu
	_, err := e.approve(e.mgr, e.addReview())
	expectErr(t, err, domain.ErrTaskNotInReview)
}

func TestReview_SubtaskNotFound_CF09(t *testing.T) {
	e := setupReview(t)
	parent := e.addReview()
	sub := e.add(e.review, &e.member)
	sub.ParentID = &parent.ID
	e.st.tasks[sub.ID] = sub
	_, err := e.approve(e.mgr, sub)
	expectErr(t, err, domain.ErrTaskNotFound)
}

func TestReview_VersionConflict_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()
	tk.Version-- // client đang thấy bản cũ
	_, err := e.approve(e.mgr, tk)
	expectErr(t, err, domain.ErrTaskVersionConflict)
	if len(e.rs.reviews) != 0 || len(e.pub.events) != 0 {
		t.Fatalf("nothing must be written on conflict")
	}
}

func TestReview_UsesKanbanTransitionRules_CF09(t *testing.T) {
	e := setupReview(t)
	delete(e.st.rules, transition{e.review, e.doing, domain.TransitionManager})
	tk := e.addReview()
	_, err := e.reject(e.mgr, tk, "lý do")
	expectErr(t, err, domain.ErrTransitionNotAllowed)
	if e.st.tasks[tk.ID].ColumnID != e.review || len(e.rs.reviews) != 0 || len(e.rs.notes) != 0 {
		t.Fatalf("must roll back")
	}
}

func TestReject_NotificationFailureRollsBack_CF09(t *testing.T) {
	e := setupReview(t)
	e.rs.failNote = true
	tk := e.addReview()
	if _, err := e.reject(e.mgr, tk, "lý do"); err == nil {
		t.Fatal("want error")
	}
	if e.st.tasks[tk.ID].ColumnID != e.review || len(e.rs.reviews) != 0 || len(e.st.history) != 0 {
		t.Fatalf("must roll back")
	}
}

func TestReviews_History_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()
	if _, err := e.reject(e.mgr, tk, "lý do"); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.Reviews(context.Background(), e.member, tk.ID)
	if err != nil || len(got) != 1 || got[0].Decision != domain.ReviewRejected {
		t.Fatalf("reviews = %+v, %v", got, err)
	}
	_, err = e.svc.Reviews(context.Background(), e.outsider, tk.ID)
	expectErr(t, err, domain.ErrTaskNotFound)
}

// ---- QA bổ sung (CF-09) ----

// Bảng quyết định vai trò: Admin không xem dự án → 404 (chống dò ID) cho cả Duyệt, Từ chối và lịch sử.
func TestReview_AdminNotFound_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()
	admin := authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin}

	_, err := e.approve(admin, tk)
	expectErr(t, err, domain.ErrTaskNotFound)
	_, err = e.reject(admin, tk, "lý do")
	expectErr(t, err, domain.ErrTaskNotFound)
	_, err = e.svc.Reviews(context.Background(), admin, tk.ID)
	expectErr(t, err, domain.ErrTaskNotFound)
	if e.st.tasks[tk.ID].ColumnID != e.review || len(e.rs.reviews) != 0 || len(e.pub.events) != 0 {
		t.Fatalf("Admin không được làm đổi dữ liệu")
	}
}

// CEO chỉ xem: đọc được lịch sử nghiệm thu nhưng không Duyệt/Từ chối (đã kiểm ở OnlyResponsibleManager).
func TestReviews_CEOCanReadHistory_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()
	if _, err := e.reject(e.mgr, tk, "lý do"); err != nil {
		t.Fatal(err)
	}
	got, err := e.svc.Reviews(context.Background(), e.ceo, tk.ID)
	if err != nil || len(got) != 1 {
		t.Fatalf("CEO xem lịch sử = %+v, %v", got, err)
	}
}

// Thông báo review_rejected: người nhận + người tạo (khi khác Manager), không trùng; không có ai → không ghi.
func TestReject_NotificationRecipients_CF09(t *testing.T) {
	t.Run("người tạo khác Manager cũng được báo", func(t *testing.T) {
		e := setupReview(t)
		tk := e.addReview()
		tk.CreatedBy = e.other.UserID
		e.st.tasks[tk.ID] = tk
		if _, err := e.reject(e.mgr, tk, "lý do"); err != nil {
			t.Fatal(err)
		}
		if len(e.rs.notes) != 1 {
			t.Fatalf("notes = %+v", e.rs.notes)
		}
		ids := e.rs.notes[0].UserIDs
		if len(ids) != 2 || ids[0] != e.member.UserID || ids[1] != e.other.UserID {
			t.Fatalf("người nhận thông báo = %v", ids)
		}
		if e.rs.notes[0].ActorID != e.mgr.UserID || e.rs.notes[0].TaskID != tk.ID || e.rs.notes[0].ProjectID != e.project.ID {
			t.Fatalf("note = %+v", e.rs.notes[0])
		}
	})
	t.Run("người nhận cũng là người tạo → một người", func(t *testing.T) {
		e := setupReview(t)
		tk := e.addReview()
		tk.CreatedBy = e.member.UserID
		e.st.tasks[tk.ID] = tk
		if _, err := e.reject(e.mgr, tk, "lý do"); err != nil {
			t.Fatal(err)
		}
		if len(e.rs.notes) != 1 || len(e.rs.notes[0].UserIDs) != 1 {
			t.Fatalf("notes = %+v", e.rs.notes)
		}
	})
	t.Run("việc chưa giao ai, Manager tạo → không thông báo, vẫn từ chối được", func(t *testing.T) {
		e := setupReview(t)
		tk := e.add(e.review, nil)
		tk.CreatedBy = e.mgr.UserID
		e.st.tasks[tk.ID] = tk
		if _, err := e.reject(e.mgr, tk, "lý do"); err != nil {
			t.Fatal(err)
		}
		if len(e.rs.notes) != 0 || len(e.rs.reviews) != 1 {
			t.Fatalf("notes = %+v reviews = %+v", e.rs.notes, e.rs.reviews)
		}
	})
}

// Biên WIP: người nhận đã đầy WIP (2/2). Từ chối (in_progress → in_progress) và Duyệt (→ done) không làm
// tăng số việc đang làm nên không bị chặn bởi WIP.
func TestReview_WIPFullDoesNotBlock_CF09(t *testing.T) {
	for _, action := range []string{"approve", "reject"} {
		t.Run(action, func(t *testing.T) {
			e := setupReview(t)
			tk := e.addReview()
			e.add(e.doing, &e.member) // việc thứ hai → WIP 2/2 (giới hạn 2)
			var err error
			if action == "approve" {
				_, err = e.approve(e.mgr, tk)
			} else {
				_, err = e.reject(e.mgr, tk, "lý do")
			}
			if err != nil {
				t.Fatalf("%s khi WIP đầy: %v", action, err)
			}
		})
	}
}

// Đoán lỗi "bấm hai lần": lần hai cùng version → 409 xung đột; version mới → không còn ở cột chờ nghiệm thu.
func TestApprove_DoubleSubmit_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()
	moved, err := e.approve(e.mgr, tk)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.approve(e.mgr, tk)
	expectErr(t, err, domain.ErrTaskVersionConflict)
	_, err = e.approve(e.mgr, moved)
	expectErr(t, err, domain.ErrTaskNotInReview)
	if len(e.rs.reviews) != 1 {
		t.Fatalf("chỉ được ghi 1 lần nghiệm thu, got %d", len(e.rs.reviews))
	}
}

// Chuyển trạng thái: Từ chối → làm lại → đưa lại vào chờ nghiệm thu → Duyệt; lịch sử ghi đủ hai lần.
func TestReview_RejectThenResubmitThenApprove_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()
	back, err := e.reject(e.mgr, tk, "lý do")
	if err != nil {
		t.Fatal(err)
	}
	// Member (người nhận) tự kéo lại vào chờ nghiệm thu bằng luật Kanban thường.
	again, err := e.move(e.member, back, e.review, 0)
	if err != nil {
		t.Fatalf("member đưa lại vào chờ nghiệm thu: %v", err)
	}
	if _, err := e.approve(e.mgr, again); err != nil {
		t.Fatal(err)
	}
	if len(e.rs.reviews) != 2 || e.rs.reviews[0].Decision != domain.ReviewRejected ||
		e.rs.reviews[1].Decision != domain.ReviewApproved {
		t.Fatalf("reviews = %+v", e.rs.reviews)
	}
}

// Quyền kiểm trước lý do: người không được nghiệm thu gửi lý do rỗng vẫn nhận lỗi quyền, không phải 422.
func TestReject_PermissionCheckedBeforeReason_CF09(t *testing.T) {
	e := setupReview(t)
	tk := e.addReview()
	admin := authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin}
	for name, c := range map[string]struct {
		actor authz.Principal
		want  error
	}{
		"member":      {e.member, domain.ErrForbidden},
		"ceo":         {e.ceo, domain.ErrForbidden},
		"admin":       {admin, domain.ErrTaskNotFound},
		"ngoài dự án": {e.outsider, domain.ErrTaskNotFound},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := e.reject(c.actor, tk, "   ")
			expectErr(t, err, c.want)
		})
	}
}
