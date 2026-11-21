package kanban_test

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// ---- giả lập ----

type fakeProjects struct {
	projects map[uuid.UUID]domain.Project
	members  map[uuid.UUID]map[uuid.UUID]bool
}

func (f *fakeProjects) Get(_ context.Context, actor authz.Principal, id uuid.UUID) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	if actor.Role == domain.RoleCEO || f.members[id][actor.UserID] {
		return p, nil
	}
	return domain.Project{}, domain.ErrProjectNotFound
}

type transition struct {
	from, to uuid.UUID
	role     domain.TransitionRole
}

type fakeStore struct {
	tasks       map[uuid.UUID]domain.Task
	groups      map[uuid.UUID]map[uuid.UUID]domain.StatusGroup // project → column → group
	rules       map[transition]bool
	history     []kanban.MoveRecord
	projectLock int
	projects    map[uuid.UUID]domain.Project // dùng chung map với fakeProjects
}

func (f *fakeStore) InTx(_ context.Context, fn func(kanban.Store) error) error {
	snap := map[uuid.UUID]domain.Task{}
	for k, v := range f.tasks {
		snap[k] = v
	}
	hist := len(f.history)
	if err := fn(f); err != nil {
		f.tasks, f.history = snap, f.history[:hist]
		return err
	}
	return nil
}

func (f *fakeStore) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return t, nil
}

func (f *fakeStore) ListBoard(_ context.Context, pid uuid.UUID, _ int) ([]domain.Task, error) {
	out := []domain.Task{}
	for _, t := range f.tasks {
		if t.ProjectID == pid && t.ParentID == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeStore) LockProject(_ context.Context, pid uuid.UUID) (domain.LockedProject, error) {
	f.projectLock++
	p, ok := f.projects[pid]
	if !ok {
		return domain.LockedProject{}, domain.ErrProjectNotFound
	}
	return domain.LockedProject{ManagerID: p.ManagerID, WIPLimit: p.WIPLimit}, nil
}

func (f *fakeStore) Lock(_ context.Context, id uuid.UUID) error {
	if _, ok := f.tasks[id]; !ok {
		return domain.ErrTaskNotFound
	}
	return nil
}

func (f *fakeStore) ColumnGroup(_ context.Context, pid, cid uuid.UUID) (domain.StatusGroup, error) {
	g, ok := f.groups[pid][cid]
	if !ok {
		return "", domain.ErrColumnNotFound
	}
	return g, nil
}

func (f *fakeStore) TransitionAllowed(_ context.Context, _ uuid.UUID, from, to uuid.UUID, role domain.TransitionRole) (bool, error) {
	return f.rules[transition{from, to, role}], nil
}

func (f *fakeStore) CountProjectWIP(_ context.Context, pid, uid uuid.UUID) (int, error) {
	n := 0
	for _, t := range f.tasks {
		if t.ProjectID == pid && t.ParentID == nil && t.AssigneeID != nil && *t.AssigneeID == uid &&
			t.StatusGroup == domain.GroupInProgress {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) MoveColumn(_ context.Context, id, cid uuid.UUID, version int) error {
	t := f.tasks[id]
	if t.Version != version {
		return domain.ErrTaskVersionConflict
	}
	t.ColumnID, t.StatusGroup, t.Version = cid, f.groups[t.ProjectID][cid], t.Version+1
	f.tasks[id] = t
	return nil
}

func (f *fakeStore) ColumnTaskIDs(_ context.Context, pid, cid uuid.UUID) ([]uuid.UUID, error) {
	var ts []domain.Task
	for _, t := range f.tasks {
		if t.ProjectID == pid && t.ColumnID == cid && t.ParentID == nil {
			ts = append(ts, t)
		}
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].Position < ts[j].Position })
	ids := make([]uuid.UUID, 0, len(ts))
	for _, t := range ts {
		ids = append(ids, t.ID)
	}
	return ids, nil
}

func (f *fakeStore) SetPositions(_ context.Context, ids []uuid.UUID) error {
	for i, id := range ids {
		t := f.tasks[id]
		t.Position = i
		f.tasks[id] = t
	}
	return nil
}

func (f *fakeStore) InsertMoveHistory(_ context.Context, r kanban.MoveRecord) error {
	f.history = append(f.history, r)
	return nil
}

type fakePublisher struct{ events []domain.BoardEvent }

func (f *fakePublisher) Publish(_ context.Context, e domain.BoardEvent) {
	f.events = append(f.events, e)
}

// ---- môi trường test ----

// env: dự án có 4 cột Backlog(todo) → Doing(in_progress) → Review(in_progress) → Done(done).
type env struct {
	svc                               *kanban.Service
	st                                *fakeStore
	pub                               *fakePublisher
	project                           domain.Project
	mgr, member, other, ceo, outsider authz.Principal
	backlog, doing, review, done      uuid.UUID
}

func setup(t *testing.T) *env {
	t.Helper()
	e := &env{
		mgr:      authz.Principal{UserID: uuid.New(), Role: domain.RoleManager},
		member:   authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		other:    authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		ceo:      authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO},
		outsider: authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		backlog:  uuid.New(), doing: uuid.New(), review: uuid.New(), done: uuid.New(),
	}
	e.project = domain.Project{ID: uuid.New(), Name: "Dự án A", ManagerID: e.mgr.UserID, WIPLimit: 2}
	pid := e.project.ID
	pr := &fakeProjects{
		projects: map[uuid.UUID]domain.Project{pid: e.project},
		members:  map[uuid.UUID]map[uuid.UUID]bool{pid: {e.mgr.UserID: true, e.member.UserID: true, e.other.UserID: true}},
	}
	m, mb := domain.TransitionManager, domain.TransitionMember
	e.st = &fakeStore{
		tasks: map[uuid.UUID]domain.Task{},
		groups: map[uuid.UUID]map[uuid.UUID]domain.StatusGroup{pid: {
			e.backlog: domain.GroupTodo, e.doing: domain.GroupInProgress,
			e.review: domain.GroupInProgress, e.done: domain.GroupDone,
		}},
		rules: map[transition]bool{
			{e.backlog, e.doing, m}: true, {e.doing, e.review, m}: true, {e.review, e.done, m}: true,
			{e.done, e.doing, m}: true, {e.review, e.backlog, m}: true,
			{e.backlog, e.doing, mb}: true, {e.doing, e.review, mb}: true,
			// luật member vào Done lỡ có trong bảng vẫn bị chặn ở domain (CLAUDE.md §5.5)
			{e.review, e.done, mb}: true,
		},
	}
	e.st.projects = pr.projects
	e.pub = &fakePublisher{}
	e.svc = kanban.NewService(e.st, e.st, pr, e.pub)
	return e
}

// add thêm công việc cha vào cuối cột.
func (e *env) add(col uuid.UUID, assignee *authz.Principal) domain.Task {
	t := domain.Task{ID: uuid.New(), ProjectID: e.project.ID, ColumnID: col,
		StatusGroup: e.st.groups[e.project.ID][col], Version: 1, Title: "Việc"}
	for _, x := range e.st.tasks {
		if x.ColumnID == col {
			t.Position++
		}
	}
	if assignee != nil {
		id := assignee.UserID
		t.AssigneeID = &id
	}
	e.st.tasks[t.ID] = t
	return t
}

func (e *env) move(actor authz.Principal, t domain.Task, col uuid.UUID, pos int) (domain.Task, error) {
	return e.svc.Move(context.Background(), actor, t.ID, kanban.MoveInput{ColumnID: col, Position: pos, Version: t.Version})
}

func expectErr(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

// ---- chuyển cột ----

func TestMove_ManagerMovesAndRecordsHistory_CF04(t *testing.T) {
	e := setup(t)
	tk := e.add(e.backlog, &e.member)

	got, err := e.move(e.mgr, tk, e.doing, 0)
	if err != nil {
		t.Fatalf("move: %v", err)
	}
	if got.ColumnID != e.doing || got.Version != 2 {
		t.Fatalf("got column %v version %d", got.ColumnID, got.Version)
	}
	if len(e.st.history) != 1 || e.st.history[0] != (kanban.MoveRecord{
		TaskID: tk.ID, ProjectID: e.project.ID, FromColumnID: e.backlog, ToColumnID: e.doing, MovedBy: e.mgr.UserID,
	}) {
		t.Fatalf("history = %+v", e.st.history)
	}
	if e.st.projectLock != 1 {
		t.Fatalf("project not locked")
	}
	if len(e.pub.events) != 1 {
		t.Fatalf("events = %+v", e.pub.events)
	}
	ev := e.pub.events[0]
	if ev.Type != domain.EventTaskMoved || ev.FromColumnID != e.backlog || ev.ToColumnID != e.doing ||
		ev.ActorID != e.mgr.UserID || ev.Version != 2 || ev.ProjectID != e.project.ID {
		t.Fatalf("event = %+v", ev)
	}
}

func TestMove_MemberMovesOwnTaskByMemberRules_CF04(t *testing.T) {
	e := setup(t)
	tk := e.add(e.backlog, &e.member)
	if _, err := e.move(e.member, tk, e.doing, 0); err != nil {
		t.Fatalf("move: %v", err)
	}
	// Không có luật member Doing → Backlog
	tk = e.st.tasks[tk.ID]
	_, err := e.move(e.member, tk, e.backlog, 0)
	expectErr(t, err, domain.ErrTransitionNotAllowed)
}

func TestMove_MemberCannotTouchDoneGroup_CF04(t *testing.T) {
	e := setup(t)
	review := e.add(e.review, &e.member)
	_, err := e.move(e.member, review, e.done, 0)
	expectErr(t, err, domain.ErrMemberTransitionToDone)

	finished := e.add(e.done, &e.member)
	_, err = e.move(e.member, finished, e.doing, 0)
	expectErr(t, err, domain.ErrMemberTransitionToDone)

	if _, err := e.move(e.mgr, review, e.done, 0); err != nil {
		t.Fatalf("manager to done: %v", err)
	}
}

func TestMove_WhoCanDrag_CF04(t *testing.T) {
	e := setup(t)
	tk := e.add(e.backlog, &e.member)
	unassigned := e.add(e.backlog, nil)

	_, err := e.move(e.other, tk, e.doing, 0)
	expectErr(t, err, domain.ErrTaskNotAssignee)
	_, err = e.move(e.other, unassigned, e.doing, 0)
	expectErr(t, err, domain.ErrTaskNotAssignee)
	_, err = e.move(e.ceo, tk, e.doing, 0)
	expectErr(t, err, domain.ErrForbidden)
	_, err = e.move(e.outsider, tk, e.doing, 0)
	expectErr(t, err, domain.ErrTaskNotFound)
	admin := authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin}
	_, err = e.move(admin, tk, e.doing, 0) // Admin không xem dự án → như việc không tồn tại
	expectErr(t, err, domain.ErrTaskNotFound)
	if e.st.tasks[tk.ID].ColumnID != e.backlog || len(e.pub.events) != 0 {
		t.Fatalf("task changed by forbidden actor")
	}
}

func TestMove_RuleMissingIsRejected_CF04(t *testing.T) {
	e := setup(t)
	tk := e.add(e.backlog, &e.member)
	_, err := e.move(e.mgr, tk, e.done, 0) // không có luật Backlog → Done
	expectErr(t, err, domain.ErrTransitionNotAllowed)
}

func TestMove_ColumnOfAnotherProject_CF04(t *testing.T) {
	e := setup(t)
	tk := e.add(e.backlog, &e.member)
	_, err := e.move(e.mgr, tk, uuid.New(), 0)
	expectErr(t, err, domain.ErrColumnNotFound)
}

func TestMove_SubtaskNotOnBoard_CF04(t *testing.T) {
	e := setup(t)
	parent := e.add(e.backlog, &e.member)
	child := e.add(e.backlog, &e.member)
	child.ParentID = &parent.ID
	e.st.tasks[child.ID] = child
	_, err := e.move(e.mgr, child, e.doing, 0)
	expectErr(t, err, domain.ErrTaskNotFound)
}

func TestMove_StaleVersionConflicts_CF04(t *testing.T) {
	e := setup(t)
	tk := e.add(e.backlog, &e.member)
	if _, err := e.move(e.mgr, tk, e.doing, 0); err != nil { // version 1 → 2
		t.Fatalf("first move: %v", err)
	}
	// Client thứ hai vẫn giữ version 1 → không được ghi đè.
	_, err := e.move(e.member, tk, e.review, 0)
	expectErr(t, err, domain.ErrTaskVersionConflict)
	if got := e.st.tasks[tk.ID]; got.ColumnID != e.doing || len(e.st.history) != 1 {
		t.Fatalf("stale move applied: %+v", got)
	}
}

// ---- thứ tự trong cột ----

func TestMove_ReorderWithinColumn_CF04(t *testing.T) {
	e := setup(t)
	a, b, c := e.add(e.backlog, &e.member), e.add(e.backlog, &e.member), e.add(e.backlog, &e.member)

	// Đổi thứ tự trong cùng cột không cần luật chuyển và không ghi lịch sử.
	if _, err := e.move(e.member, c, e.backlog, 0); err != nil {
		t.Fatalf("reorder: %v", err)
	}
	want := map[uuid.UUID]int{c.ID: 0, a.ID: 1, b.ID: 2}
	for id, pos := range want {
		if e.st.tasks[id].Position != pos {
			t.Fatalf("position %v = %d, want %d", id, e.st.tasks[id].Position, pos)
		}
	}
	if len(e.st.history) != 0 {
		t.Fatalf("reorder wrote history")
	}
	if e.pub.events[0].FromColumnID != e.backlog || e.pub.events[0].Position != 0 {
		t.Fatalf("event = %+v", e.pub.events[0])
	}
}

func TestMove_InsertAtPositionInTargetColumn_CF04(t *testing.T) {
	e := setup(t)
	x, y := e.add(e.doing, &e.other), e.add(e.doing, &e.other)
	tk := e.add(e.backlog, &e.member)
	if _, err := e.move(e.mgr, tk, e.doing, 1); err != nil {
		t.Fatalf("move: %v", err)
	}
	want := map[uuid.UUID]int{x.ID: 0, tk.ID: 1, y.ID: 2}
	for id, pos := range want {
		if e.st.tasks[id].Position != pos {
			t.Fatalf("position %v = %d, want %d", id, e.st.tasks[id].Position, pos)
		}
	}
}

// ---- giới hạn WIP (CF-05) ----

func TestMove_WIPLimitBlocksEnteringInProgress_CF05(t *testing.T) {
	e := setup(t) // WIPLimit = 2
	e.add(e.doing, &e.member)
	e.add(e.review, &e.member) // "Chờ nghiệm thu" cũng thuộc in_progress (§5.5)
	tk := e.add(e.backlog, &e.member)

	_, err := e.move(e.member, tk, e.doing, 0)
	expectErr(t, err, domain.ErrWIPLimitReached)
	_, err = e.move(e.mgr, tk, e.doing, 0) // Manager kéo hộ cũng bị chặn: giới hạn là của người nhận
	expectErr(t, err, domain.ErrWIPLimitReached)
	if e.st.tasks[tk.ID].ColumnID != e.backlog {
		t.Fatalf("task moved despite WIP")
	}
}

func TestMove_WIPCountsOnlyParentTasks_CF05(t *testing.T) {
	e := setup(t)
	parent := e.add(e.doing, &e.member)
	child := e.add(e.doing, &e.member)
	child.ParentID = &parent.ID
	e.st.tasks[child.ID] = child
	tk := e.add(e.backlog, &e.member)
	if _, err := e.move(e.member, tk, e.doing, 0); err != nil {
		t.Fatalf("child counted in WIP: %v", err)
	}
}

func TestMove_WIPNotCheckedInsideInProgressOrUnassigned_CF05(t *testing.T) {
	e := setup(t)
	a := e.add(e.doing, &e.member)
	e.add(e.doing, &e.member) // đã đủ 2/2
	if _, err := e.move(e.member, a, e.review, 0); err != nil {
		t.Fatalf("move inside in_progress: %v", err)
	}
	free := e.add(e.backlog, nil)
	if _, err := e.move(e.mgr, free, e.doing, 0); err != nil {
		t.Fatalf("unassigned task: %v", err)
	}
}

// ---- tải bảng ----

func TestBoard_Visibility_CF04(t *testing.T) {
	e := setup(t)
	e.add(e.backlog, &e.member)
	for _, a := range []authz.Principal{e.mgr, e.member, e.ceo} {
		got, err := e.svc.Board(context.Background(), a, e.project.ID)
		if err != nil || len(got) != 1 {
			t.Fatalf("board for %s: %v, %d", a.Role, err, len(got))
		}
	}
	_, err := e.svc.Board(context.Background(), e.outsider, e.project.ID)
	expectErr(t, err, domain.ErrProjectNotFound)
}

// Giới hạn WIP và Manager phụ trách lấy trên dòng dự án ĐÃ KHÓA, không phải bản đọc trước transaction:
// Manager vừa hạ giới hạn (hoặc dự án vừa đổi người phụ trách) thì lần kéo đang chờ khóa dùng số mới.
func TestMove_UsesProjectStateReadUnderLock_CF04_CF05(t *testing.T) {
	e := setup(t) // bản ngoài transaction: WIPLimit = 2, Manager = e.mgr
	e.add(e.doing, &e.member)
	tk := e.add(e.backlog, &e.member)

	newMgr := authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	e.st.projects = map[uuid.UUID]domain.Project{e.project.ID: {ID: e.project.ID, ManagerID: newMgr.UserID, WIPLimit: 1}}

	_, err := e.move(e.member, tk, e.doing, 0)
	expectErr(t, err, domain.ErrWIPLimitReached) // 1 việc đang làm, giới hạn mới là 1
	_, err = e.move(e.mgr, tk, e.review, 0)      // e.mgr không còn phụ trách khi đã khóa
	expectErr(t, err, domain.ErrTaskNotAssignee)
	if e.st.tasks[tk.ID].ColumnID != e.backlog {
		t.Fatalf("task moved on stale project state")
	}
}
