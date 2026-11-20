package task_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// fakeProjects mô phỏng project.Service.Get: chỉ thành viên (hoặc CEO) xem được.
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

type fakeSettings struct{ hours int }

func (f fakeSettings) HoursPerStoryPoint(context.Context) (int, error) { return f.hours, nil }

type column struct {
	id    uuid.UUID
	name  string
	group domain.StatusGroup
}

// fakeStore là Store trong bộ nhớ; InTx khôi phục bản chụp khi fn lỗi.
type fakeStore struct {
	tasks      map[uuid.UUID]domain.Task
	columns    map[uuid.UUID][]column
	members    map[uuid.UUID]map[uuid.UUID]task.Assignable
	labels     map[uuid.UUID]string
	locks      int
	lastFilter task.Filter
	lastMine   task.MyFilter
}

func (f *fakeStore) InTx(_ context.Context, fn func(task.Store) error) error {
	snap := map[uuid.UUID]domain.Task{}
	for k, v := range f.tasks {
		snap[k] = v
	}
	if err := fn(f); err != nil {
		f.tasks = snap
		return err
	}
	return nil
}

func (f *fakeStore) List(_ context.Context, pid uuid.UUID, flt task.Filter, _ domain.PageRequest) ([]domain.Task, int, error) {
	f.lastFilter = flt
	var out []domain.Task
	for _, t := range f.tasks {
		if t.ProjectID == pid {
			out = append(out, t)
		}
	}
	return out, len(out), nil
}

func (f *fakeStore) ListMine(_ context.Context, uid uuid.UUID, flt task.MyFilter, _ domain.PageRequest) ([]domain.Task, int, error) {
	f.lastMine = flt
	var out []domain.Task
	for _, t := range f.tasks {
		if t.AssigneeID != nil && *t.AssigneeID == uid {
			out = append(out, t)
		}
	}
	return out, len(out), nil
}

func (f *fakeStore) Get(_ context.Context, id uuid.UUID) (domain.Task, error) {
	t, ok := f.tasks[id]
	if !ok {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	return t, nil
}

func (f *fakeStore) Lock(_ context.Context, id uuid.UUID) error {
	if _, ok := f.tasks[id]; !ok {
		return domain.ErrTaskNotFound
	}
	f.locks++
	return nil
}

func (f *fakeStore) FirstTodoColumn(_ context.Context, pid uuid.UUID) (uuid.UUID, error) {
	for _, c := range f.columns[pid] {
		if c.group == domain.GroupTodo {
			return c.id, nil
		}
	}
	return uuid.Nil, domain.ErrBoardNeedsTodoColumn
}

func (f *fakeStore) apply(t *domain.Task, in task.Fields) error {
	t.Labels = []domain.TaskLabel{}
	for _, id := range in.LabelIDs {
		name, ok := f.labels[id]
		if !ok {
			return domain.ErrLabelNotFound
		}
		t.Labels = append(t.Labels, domain.TaskLabel{ID: id, Name: name})
	}
	t.Title, t.Description, t.StoryPoint, t.EstimatedMinutes = in.Title, in.Description, in.StoryPoint, in.EstimatedMinutes
	t.AssigneeID, t.DueDate = in.AssigneeID, in.DueDate
	return nil
}

func (f *fakeStore) Create(_ context.Context, pid, cid, by uuid.UUID, in task.Fields) (uuid.UUID, error) {
	t := domain.Task{ID: uuid.New(), ProjectID: pid, ColumnID: cid, CreatedBy: by}
	for _, c := range f.columns[pid] {
		if c.id == cid {
			t.ColumnName, t.StatusGroup = c.name, c.group
		}
	}
	if err := f.apply(&t, in); err != nil {
		return uuid.Nil, err
	}
	f.tasks[t.ID] = t
	return t.ID, nil
}

func (f *fakeStore) Update(_ context.Context, id uuid.UUID, in task.Fields) error {
	t, ok := f.tasks[id]
	if !ok {
		return domain.ErrTaskNotFound
	}
	if err := f.apply(&t, in); err != nil {
		return err
	}
	f.tasks[id] = t
	return nil
}

func (f *fakeStore) SetAssignee(_ context.Context, id uuid.UUID, uid *uuid.UUID) error {
	t, ok := f.tasks[id]
	if !ok {
		return domain.ErrTaskNotFound
	}
	t.AssigneeID = uid
	f.tasks[id] = t
	return nil
}

func (f *fakeStore) GetAssignable(_ context.Context, pid, uid uuid.UUID) (task.Assignable, bool, error) {
	a, ok := f.members[pid][uid]
	return a, ok, nil
}

func (f *fakeStore) CountWIP(_ context.Context, uid uuid.UUID) (int, error) {
	n := 0
	for _, t := range f.tasks {
		if t.AssigneeID != nil && *t.AssigneeID == uid && t.ParentID == nil && t.StatusGroup == domain.GroupInProgress {
			n++
		}
	}
	return n, nil
}

type env struct {
	svc                       *task.Service
	st                        *fakeStore
	pr                        *fakeProjects
	project                   domain.Project
	mgr, member, other, admin authz.Principal
	ceo, disabled             authz.Principal
	backlog, doing            uuid.UUID
	label                     uuid.UUID
}

func setup(t *testing.T) *env {
	t.Helper()
	e := &env{
		mgr:      authz.Principal{UserID: uuid.New(), Role: domain.RoleManager},
		member:   authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		other:    authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		admin:    authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin},
		ceo:      authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO},
		disabled: authz.Principal{UserID: uuid.New(), Role: domain.RoleMember},
		backlog:  uuid.New(),
		doing:    uuid.New(),
		label:    uuid.New(),
	}
	e.project = domain.Project{ID: uuid.New(), Name: "Dự án A", ManagerID: e.mgr.UserID}
	pid := e.project.ID
	e.pr = &fakeProjects{
		projects: map[uuid.UUID]domain.Project{pid: e.project},
		members: map[uuid.UUID]map[uuid.UUID]bool{pid: {
			e.mgr.UserID: true, e.member.UserID: true, e.disabled.UserID: true,
		}},
	}
	e.st = &fakeStore{
		tasks: map[uuid.UUID]domain.Task{},
		columns: map[uuid.UUID][]column{pid: {
			{e.doing, "In Progress", domain.GroupInProgress}, // cột todo không đứng đầu vẫn phải chọn đúng todo
			{e.backlog, "Backlog", domain.GroupTodo},
		}},
		members: map[uuid.UUID]map[uuid.UUID]task.Assignable{pid: {
			e.mgr.UserID:      {Role: domain.RoleManager, Status: domain.UserStatusActive},
			e.member.UserID:   {Role: domain.RoleMember, Status: domain.UserStatusActive},
			e.disabled.UserID: {Role: domain.RoleMember, Status: domain.UserStatusDisabled},
		}},
		labels: map[uuid.UUID]string{e.label: "Backend"},
	}
	e.svc = task.NewService(e.st, e.st, e.pr, fakeSettings{hours: 4})
	return e
}

func ptr[T any](v T) *T { return &v }

func expectErr(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func (e *env) create(t *testing.T, in task.Input) domain.Task {
	t.Helper()
	got, err := e.svc.Create(context.Background(), e.mgr, e.project.ID, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return got
}

func TestCreate_ManagerPutsTaskInFirstTodoColumn_CF06(t *testing.T) {
	e := setup(t)
	due := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	got := e.create(t, task.Input{
		Title: "  Thiết kế   API  ", Description: " mô tả ", StoryPoint: ptr(3),
		AssigneeID: &e.member.UserID, DueDate: &due, LabelIDs: []uuid.UUID{e.label, e.label},
	})
	if got.ColumnID != e.backlog || got.StatusGroup != domain.GroupTodo {
		t.Fatalf("cột = %v (%s), muốn Backlog", got.ColumnID, got.StatusGroup)
	}
	if got.Title != "Thiết kế API" || got.Description != "mô tả" {
		t.Fatalf("chưa chuẩn hóa: %q %q", got.Title, got.Description)
	}
	if got.EstimatedMinutes == nil || *got.EstimatedMinutes != 3*4*60 {
		t.Fatalf("giờ ước tính B1 sai: %v", got.EstimatedMinutes)
	}
	if len(got.Labels) != 1 || got.CreatedBy != e.mgr.UserID {
		t.Fatalf("nhãn/người tạo sai: %+v", got)
	}
}

func TestCreate_NoStoryPointNoEstimate_CF06(t *testing.T) {
	e := setup(t)
	got := e.create(t, task.Input{Title: "Việc nhỏ"})
	if got.StoryPoint != nil || got.EstimatedMinutes != nil || got.AssigneeID != nil {
		t.Fatalf("muốn trống: %+v", got)
	}
	if got.Labels == nil {
		t.Fatal("nhãn phải là slice rỗng")
	}
}

func TestCreate_OnlyProjectManager_CF06(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	otherMgr := authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	e.pr.members[e.project.ID][otherMgr.UserID] = true
	cases := []struct {
		name  string
		actor authz.Principal
		want  error
	}{
		{"Member không giao việc (§5.3)", e.member, domain.ErrForbidden},
		{"Manager khác trong dự án", otherMgr, domain.ErrForbidden},
		{"CEO chỉ xem", e.ceo, domain.ErrForbidden},
		{"Admin không thấy dự án", e.admin, domain.ErrProjectNotFound},
		{"người ngoài dự án", e.other, domain.ErrProjectNotFound},
	}
	for _, c := range cases {
		_, err := e.svc.Create(ctx, c.actor, e.project.ID, task.Input{Title: "x"})
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	if len(e.st.tasks) != 0 {
		t.Fatal("không được tạo việc")
	}
}

func TestCreate_Validation_CF06(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	many := make([]uuid.UUID, domain.TaskLabelsMax+1)
	for i := range many {
		many[i] = uuid.New()
	}
	cases := []struct {
		name string
		in   task.Input
		want error
	}{
		{"tiêu đề trống", task.Input{Title: "   "}, domain.ErrInvalidTaskTitle},
		{"tiêu đề quá dài", task.Input{Title: strings.Repeat("a", 201)}, domain.ErrInvalidTaskTitle},
		{"mô tả quá dài", task.Input{Title: "x", Description: strings.Repeat("a", 5001)}, domain.ErrInvalidTaskDescription},
		{"SP không thuộc thang", task.Input{Title: "x", StoryPoint: ptr(4)}, domain.ErrInvalidStoryPoint},
		{"SP âm", task.Input{Title: "x", StoryPoint: ptr(-1)}, domain.ErrInvalidStoryPoint},
		{"quá nhiều nhãn", task.Input{Title: "x", LabelIDs: many}, domain.ErrTooManyLabels},
		{"nhãn không tồn tại", task.Input{Title: "x", LabelIDs: []uuid.UUID{uuid.New()}}, domain.ErrLabelNotFound},
		{"người nhận ngoài dự án", task.Input{Title: "x", AssigneeID: &e.other.UserID}, domain.ErrInvalidAssignee},
		{"người nhận đã khóa", task.Input{Title: "x", AssigneeID: &e.disabled.UserID}, domain.ErrInvalidAssignee},
	}
	for _, c := range cases {
		_, err := e.svc.Create(ctx, e.mgr, e.project.ID, c.in)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	if len(e.st.tasks) != 0 {
		t.Fatalf("lỗi phải rollback, còn %d việc", len(e.st.tasks))
	}
}

func TestCreate_NeedsTodoColumn_CF06(t *testing.T) {
	e := setup(t)
	e.st.columns[e.project.ID] = []column{{e.doing, "In Progress", domain.GroupInProgress}}
	_, err := e.svc.Create(context.Background(), e.mgr, e.project.ID, task.Input{Title: "x"})
	expectErr(t, err, domain.ErrBoardNeedsTodoColumn)
}

func TestCreate_ManagerCanAssignSelf_CF06(t *testing.T) {
	e := setup(t)
	got := e.create(t, task.Input{Title: "x", AssigneeID: &e.mgr.UserID})
	if got.AssigneeID == nil || *got.AssigneeID != e.mgr.UserID {
		t.Fatal("Manager phải tự giao được cho mình")
	}
}

func TestUpdate_ReplaceAndEstimateSnapshot_CF06_B1(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	created := e.create(t, task.Input{Title: "x", StoryPoint: ptr(2), LabelIDs: []uuid.UUID{e.label}})

	// Đổi hệ số sau khi tạo: giữ nguyên SP → giữ giờ ước tính đã chụp.
	e.svc = task.NewService(e.st, e.st, e.pr, fakeSettings{hours: 8})
	got, err := e.svc.Update(ctx, e.mgr, created.ID, task.Input{Title: "y", StoryPoint: ptr(2), AssigneeID: &e.member.UserID})
	if err != nil {
		t.Fatal(err)
	}
	if *got.EstimatedMinutes != 2*4*60 {
		t.Fatalf("SP không đổi phải giữ ước tính cũ, got %d", *got.EstimatedMinutes)
	}
	if got.Title != "y" || len(got.Labels) != 0 || got.AssigneeID == nil {
		t.Fatalf("PUT phải thay toàn bộ: %+v", got)
	}
	// Đổi SP → tính lại theo hệ số hiện hành.
	got, err = e.svc.Update(ctx, e.mgr, created.ID, task.Input{Title: "y", StoryPoint: ptr(5)})
	if err != nil {
		t.Fatal(err)
	}
	if *got.EstimatedMinutes != 5*8*60 || got.AssigneeID != nil {
		t.Fatalf("got %+v", got)
	}
	// Bỏ SP → bỏ ước tính.
	got, err = e.svc.Update(ctx, e.mgr, created.ID, task.Input{Title: "y"})
	if err != nil {
		t.Fatal(err)
	}
	if got.StoryPoint != nil || got.EstimatedMinutes != nil {
		t.Fatalf("got %+v", got)
	}
}

func TestUpdate_PermissionAndNotFound_CF06(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	created := e.create(t, task.Input{Title: "x"})
	_, err := e.svc.Update(ctx, e.member, created.ID, task.Input{Title: "hack"})
	expectErr(t, err, domain.ErrForbidden)
	_, err = e.svc.Update(ctx, e.other, created.ID, task.Input{Title: "hack"})
	expectErr(t, err, domain.ErrTaskNotFound) // IDOR: người ngoài dự án không biết việc tồn tại
	_, err = e.svc.Update(ctx, e.mgr, uuid.New(), task.Input{Title: "x"})
	expectErr(t, err, domain.ErrTaskNotFound)
	if e.st.tasks[created.ID].Title != "x" {
		t.Fatal("không được sửa")
	}
}

func TestUpdate_SubtaskNotHandledHere_CF06(t *testing.T) {
	e := setup(t)
	parent := e.create(t, task.Input{Title: "cha"})
	child := domain.Task{ID: uuid.New(), ProjectID: e.project.ID, ParentID: &parent.ID, ColumnID: e.backlog, StatusGroup: domain.GroupTodo}
	e.st.tasks[child.ID] = child
	_, err := e.svc.Update(context.Background(), e.mgr, child.ID, task.Input{Title: "x", StoryPoint: ptr(3)})
	expectErr(t, err, domain.ErrTaskNotFound)
}

func TestGet_Visibility_CF06(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	created := e.create(t, task.Input{Title: "x"})
	for _, a := range []authz.Principal{e.mgr, e.member, e.ceo} {
		if _, err := e.svc.Get(ctx, a, created.ID); err != nil {
			t.Errorf("%s: %v", a.Role, err)
		}
	}
	for _, a := range []authz.Principal{e.other, e.admin} {
		_, err := e.svc.Get(ctx, a, created.ID)
		expectErr(t, err, domain.ErrTaskNotFound)
	}
}

func TestClaim_MemberTakesUnassignedTodo_CF06(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	created := e.create(t, task.Input{Title: "x"})
	got, err := e.svc.Claim(ctx, e.member, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AssigneeID == nil || *got.AssigneeID != e.member.UserID || e.st.locks == 0 {
		t.Fatalf("chưa nhận / chưa khóa dòng: %+v", got)
	}
	if got.ColumnID != e.backlog {
		t.Fatal("nhận việc không được chuyển cột (CF-04)")
	}
}

func TestClaim_Rules_CF06(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	assigned := e.create(t, task.Input{Title: "đã giao", AssigneeID: &e.mgr.UserID})
	open := e.create(t, task.Input{Title: "mở"})
	doing := e.create(t, task.Input{Title: "đang làm"})
	d := e.st.tasks[doing.ID]
	d.ColumnID, d.StatusGroup = e.doing, domain.GroupInProgress
	e.st.tasks[doing.ID] = d

	cases := []struct {
		name  string
		actor authz.Principal
		id    uuid.UUID
		want  error
	}{
		{"việc đã có người nhận", e.member, assigned.ID, domain.ErrTaskAlreadyAssigned},
		{"việc không ở nhóm todo", e.member, doing.ID, domain.ErrTaskNotClaimable},
		{"Manager giao qua sửa việc, không nhận", e.mgr, open.ID, domain.ErrForbidden},
		{"CEO không nhận việc", e.ceo, open.ID, domain.ErrForbidden},
		{"người ngoài dự án", e.other, open.ID, domain.ErrTaskNotFound},
		{"tài khoản đã khóa", e.disabled, open.ID, domain.ErrForbidden},
		{"việc không tồn tại", e.member, uuid.New(), domain.ErrTaskNotFound},
	}
	for _, c := range cases {
		_, err := e.svc.Claim(ctx, c.actor, c.id)
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, err, c.want)
		}
	}
	if e.st.tasks[open.ID].AssigneeID != nil {
		t.Fatal("việc mở không được bị nhận")
	}
}

func TestList_FiltersAndVisibility_CF06(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.create(t, task.Input{Title: "a"})
	page, err := e.svc.List(ctx, e.member, e.project.ID, task.ListInput{
		Search: "  api ", StatusGroup: "todo", AssigneeID: &e.member.UserID, Unassigned: true,
	}, domain.NewPageRequest(1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("got %+v", page)
	}
	f := e.st.lastFilter
	if f.Search != "api" || f.StatusGroup == nil || *f.StatusGroup != domain.GroupTodo {
		t.Fatalf("filter chưa chuẩn hóa: %+v", f)
	}
	if !f.Unassigned || f.AssigneeID != nil {
		t.Fatal("unassigned=true phải bỏ qua assigneeId")
	}
	_, err = e.svc.List(ctx, e.member, e.project.ID, task.ListInput{StatusGroup: "xong"}, domain.NewPageRequest(1, 20))
	expectErr(t, err, domain.ErrInvalidStatusGroup)
	_, err = e.svc.List(ctx, e.member, e.project.ID, task.ListInput{Search: strings.Repeat("a", 101)}, domain.NewPageRequest(1, 20))
	expectErr(t, err, domain.ErrInvalidSearch)
	_, err = e.svc.List(ctx, e.other, e.project.ID, task.ListInput{}, domain.NewPageRequest(1, 20))
	expectErr(t, err, domain.ErrProjectNotFound)
}

func TestList_EmptyIsNotNil_CF06(t *testing.T) {
	e := setup(t)
	page, err := e.svc.List(context.Background(), e.mgr, e.project.ID, task.ListInput{}, domain.NewPageRequest(1, 20))
	if err != nil || page.Items == nil {
		t.Fatalf("got %v %v", page.Items, err)
	}
}

func TestListMine_AndWIP_CF06(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	a := e.create(t, task.Input{Title: "a", AssigneeID: &e.member.UserID})
	e.create(t, task.Input{Title: "b", AssigneeID: &e.member.UserID})
	ta := e.st.tasks[a.ID]
	ta.StatusGroup = domain.GroupInProgress
	e.st.tasks[a.ID] = ta

	page, err := e.svc.ListMine(ctx, e.member, task.MyListInput{StatusGroup: "in_progress"}, domain.NewPageRequest(1, 20))
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 2 || e.st.lastMine.StatusGroup == nil {
		t.Fatalf("got %+v / %+v", page, e.st.lastMine)
	}
	n, err := e.svc.MyWIP(ctx, e.member)
	if err != nil || n != 1 {
		t.Fatalf("WIP = %d, %v; muốn 1", n, err)
	}
	for _, a := range []authz.Principal{e.admin, e.ceo} {
		_, err := e.svc.ListMine(ctx, a, task.MyListInput{}, domain.NewPageRequest(1, 20))
		expectErr(t, err, domain.ErrForbidden)
		_, err = e.svc.MyWIP(ctx, a)
		expectErr(t, err, domain.ErrForbidden)
	}
	_, err = e.svc.ListMine(ctx, e.member, task.MyListInput{StatusGroup: "bad"}, domain.NewPageRequest(1, 20))
	expectErr(t, err, domain.ErrInvalidStatusGroup)
}
