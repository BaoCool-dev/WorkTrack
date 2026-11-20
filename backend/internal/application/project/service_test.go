package project_test

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// fakeStore là Store trong bộ nhớ. InTx chụp trạng thái và khôi phục khi fn lỗi (mô phỏng rollback).
type fakeStore struct {
	projects    map[uuid.UUID]domain.Project
	members     map[uuid.UUID]map[uuid.UUID]bool
	columns     map[uuid.UUID][]domain.BoardColumn
	transitions map[uuid.UUID][]domain.Transition
	users       map[uuid.UUID]domain.MemberCandidate
	locks       int
	lastFilter  project.Filter
}

func newFake() *fakeStore {
	return &fakeStore{
		projects:    map[uuid.UUID]domain.Project{},
		members:     map[uuid.UUID]map[uuid.UUID]bool{},
		columns:     map[uuid.UUID][]domain.BoardColumn{},
		transitions: map[uuid.UUID][]domain.Transition{},
		users:       map[uuid.UUID]domain.MemberCandidate{},
	}
}

func (f *fakeStore) snapshot() *fakeStore {
	c := newFake()
	for k, v := range f.projects {
		c.projects[k] = v
	}
	for k, v := range f.members {
		m := map[uuid.UUID]bool{}
		for u := range v {
			m[u] = true
		}
		c.members[k] = m
	}
	for k, v := range f.columns {
		c.columns[k] = append([]domain.BoardColumn(nil), v...)
	}
	for k, v := range f.transitions {
		c.transitions[k] = append([]domain.Transition(nil), v...)
	}
	c.users = f.users
	return c
}

func (f *fakeStore) InTx(_ context.Context, fn func(project.Store) error) error {
	snap := f.snapshot()
	if err := fn(f); err != nil {
		f.projects, f.members, f.columns, f.transitions = snap.projects, snap.members, snap.columns, snap.transitions
		return err
	}
	return nil
}

func (f *fakeStore) List(_ context.Context, flt project.Filter, p domain.PageRequest) ([]domain.Project, int, error) {
	f.lastFilter = flt
	var out []domain.Project
	for id, pr := range f.projects {
		if flt.VisibleTo != nil && !f.members[id][*flt.VisibleTo] {
			continue
		}
		if flt.Status != nil && pr.Status != *flt.Status {
			continue
		}
		if flt.Search != "" && !strings.Contains(strings.ToLower(pr.Name), strings.ToLower(flt.Search)) {
			continue
		}
		out = append(out, pr)
	}
	return out, len(out), nil
}

func (f *fakeStore) Get(_ context.Context, id uuid.UUID) (domain.Project, error) {
	p, ok := f.projects[id]
	if !ok {
		return domain.Project{}, domain.ErrProjectNotFound
	}
	return p, nil
}

func (f *fakeStore) Lock(_ context.Context, id uuid.UUID) error {
	if _, ok := f.projects[id]; !ok {
		return domain.ErrProjectNotFound
	}
	f.locks++
	return nil
}

func (f *fakeStore) Create(_ context.Context, in project.NewProject) (uuid.UUID, error) {
	for _, p := range f.projects {
		if strings.EqualFold(p.Name, in.Name) {
			return uuid.Nil, domain.ErrProjectNameTaken
		}
	}
	id := uuid.New()
	f.projects[id] = domain.Project{ID: id, Name: in.Name, Description: in.Description, Status: in.Status,
		StartDate: in.StartDate, EndDate: in.EndDate, ManagerID: in.ManagerID}
	return id, nil
}

func (f *fakeStore) Update(_ context.Context, id uuid.UUID, in project.Fields) error {
	p, ok := f.projects[id]
	if !ok {
		return domain.ErrProjectNotFound
	}
	p.Name, p.Description, p.Status, p.StartDate, p.EndDate = in.Name, in.Description, in.Status, in.StartDate, in.EndDate
	f.projects[id] = p
	return nil
}

func (f *fakeStore) IsMember(_ context.Context, pid, uid uuid.UUID) (bool, error) {
	return f.members[pid][uid], nil
}

func (f *fakeStore) ListMembers(_ context.Context, pid uuid.UUID) ([]domain.ProjectMember, error) {
	var out []domain.ProjectMember
	for u := range f.members[pid] {
		out = append(out, domain.ProjectMember{UserID: u})
	}
	return out, nil
}

func (f *fakeStore) AddMember(_ context.Context, pid, uid uuid.UUID) error {
	if f.members[pid] == nil {
		f.members[pid] = map[uuid.UUID]bool{}
	}
	f.members[pid][uid] = true
	return nil
}

func (f *fakeStore) RemoveMember(_ context.Context, pid, uid uuid.UUID) error {
	if !f.members[pid][uid] {
		return domain.ErrProjectMemberNotFound
	}
	delete(f.members[pid], uid)
	return nil
}

func (f *fakeStore) ListCandidates(_ context.Context, _ uuid.UUID, _ string) ([]domain.MemberCandidate, error) {
	return nil, nil
}

func (f *fakeStore) GetCandidate(_ context.Context, uid uuid.UUID) (domain.MemberCandidate, error) {
	c, ok := f.users[uid]
	if !ok {
		return domain.MemberCandidate{}, domain.ErrUserNotFound
	}
	return c, nil
}

func (f *fakeStore) ListColumns(_ context.Context, pid uuid.UUID) ([]domain.BoardColumn, error) {
	cols := append([]domain.BoardColumn(nil), f.columns[pid]...)
	sort.Slice(cols, func(i, j int) bool { return cols[i].Position < cols[j].Position })
	return cols, nil
}

func (f *fakeStore) CreateColumn(_ context.Context, pid uuid.UUID, c domain.ColumnSpec, pos int) (uuid.UUID, error) {
	id := uuid.New()
	f.columns[pid] = append(f.columns[pid], domain.BoardColumn{ID: id, ProjectID: pid, Name: c.Name, StatusGroup: c.StatusGroup, Position: pos})
	return id, nil
}

func (f *fakeStore) col(pid, id uuid.UUID) *domain.BoardColumn {
	for i := range f.columns[pid] {
		if f.columns[pid][i].ID == id {
			return &f.columns[pid][i]
		}
	}
	return nil
}

func (f *fakeStore) UpdateColumn(_ context.Context, pid, id uuid.UUID, c domain.ColumnSpec) error {
	col := f.col(pid, id)
	if col == nil {
		return domain.ErrColumnNotFound
	}
	col.Name, col.StatusGroup = c.Name, c.StatusGroup
	return nil
}

func (f *fakeStore) SetColumnPosition(_ context.Context, pid, id uuid.UUID, pos int) error {
	col := f.col(pid, id)
	if col == nil {
		return domain.ErrColumnNotFound
	}
	col.Position = pos
	return nil
}

func (f *fakeStore) DeleteColumn(_ context.Context, pid, id uuid.UUID) error {
	cols := f.columns[pid]
	for i, c := range cols {
		if c.ID == id {
			f.columns[pid] = append(cols[:i:i], cols[i+1:]...)
			// ON DELETE CASCADE
			var keep []domain.Transition
			for _, t := range f.transitions[pid] {
				if t.FromColumnID != id && t.ToColumnID != id {
					keep = append(keep, t)
				}
			}
			f.transitions[pid] = keep
			return nil
		}
	}
	return domain.ErrColumnNotFound
}

func (f *fakeStore) ListTransitions(_ context.Context, pid uuid.UUID) ([]domain.Transition, error) {
	return append([]domain.Transition(nil), f.transitions[pid]...), nil
}

func (f *fakeStore) ReplaceTransitions(_ context.Context, pid uuid.UUID, ts []domain.Transition) error {
	f.transitions[pid] = append([]domain.Transition(nil), ts...)
	return nil
}

// ---- helpers ----

var (
	mgrA   = authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	mgrB   = authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	member = authz.Principal{UserID: uuid.New(), Role: domain.RoleMember}
	ceo    = authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO}
	admin  = authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin}
)

func setup(t *testing.T) (*project.Service, *fakeStore, domain.Project) {
	t.Helper()
	st := newFake()
	svc := project.NewService(st, st)
	p, err := svc.Create(context.Background(), mgrA, project.Input{Name: "  Website   bán hàng "})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return svc, st, p
}

func expectErr(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func colID(t *testing.T, st *fakeStore, pid uuid.UUID, name string) uuid.UUID {
	t.Helper()
	for _, c := range st.columns[pid] {
		if c.Name == name {
			return c.ID
		}
	}
	t.Fatalf("column %q not found", name)
	return uuid.Nil
}

// ---- tạo dự án ----

func TestCreate_ManagerGetsDefaultsBoard_CF03(t *testing.T) {
	_, st, p := setup(t)
	if p.Name != "Website bán hàng" || p.Status != domain.ProjectNew || p.ManagerID != mgrA.UserID {
		t.Fatalf("project = %+v", p)
	}
	if !st.members[p.ID][mgrA.UserID] {
		t.Fatal("manager phải là thành viên dự án")
	}
	cols, _ := st.ListColumns(context.Background(), p.ID)
	if len(cols) != 5 || cols[0].Name != "Backlog" || cols[4].StatusGroup != domain.GroupDone {
		t.Fatalf("cols = %+v", cols)
	}
	if len(st.transitions[p.ID]) == 0 {
		t.Fatal("phải có luật chuyển mặc định")
	}
	if err := domain.ValidateTransitions(cols, st.transitions[p.ID]); err != nil {
		t.Fatalf("luật mặc định không hợp lệ: %v", err)
	}
}

func TestCreate_OnlyManager_CF03(t *testing.T) {
	st := newFake()
	svc := project.NewService(st, st)
	for _, a := range []authz.Principal{admin, ceo, member} {
		_, err := svc.Create(context.Background(), a, project.Input{Name: "X"})
		expectErr(t, err, domain.ErrForbidden)
	}
	if len(st.projects) != 0 {
		t.Fatal("không được tạo dự án")
	}
}

func TestCreate_Validation_CF03(t *testing.T) {
	st := newFake()
	svc := project.NewService(st, st)
	start := time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, -1)
	cases := []struct {
		name string
		in   project.Input
		want error
	}{
		{"tên rỗng", project.Input{Name: "   "}, domain.ErrInvalidProjectName},
		{"tên quá dài", project.Input{Name: strings.Repeat("a", 151)}, domain.ErrInvalidProjectName},
		{"mô tả quá dài", project.Input{Name: "A", Description: strings.Repeat("a", 2001)}, domain.ErrInvalidProjectDescription},
		{"trạng thái sai", project.Input{Name: "A", Status: "done"}, domain.ErrInvalidProjectStatus},
		{"ngày kết thúc trước ngày bắt đầu", project.Input{Name: "A", StartDate: &start, EndDate: &end}, domain.ErrInvalidProjectDates},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), mgrA, c.in)
			expectErr(t, err, c.want)
		})
	}
}

func TestCreate_DuplicateNameRollsBack_CF03(t *testing.T) {
	svc, st, _ := setup(t)
	_, err := svc.Create(context.Background(), mgrB, project.Input{Name: "website bán hàng"})
	expectErr(t, err, domain.ErrProjectNameTaken)
	if len(st.projects) != 1 {
		t.Fatalf("projects = %d", len(st.projects))
	}
}

// ---- danh sách / xem ----

func TestList_ByRole_CF03(t *testing.T) {
	svc, st, _ := setup(t)
	ctx := context.Background()
	if _, err := svc.Create(ctx, mgrB, project.Input{Name: "Dự án B"}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name  string
		actor authz.Principal
		want  int
	}{
		{"CEO thấy tất cả", ceo, 2},
		{"Manager A chỉ dự án của mình", mgrA, 1},
		{"Member chưa tham gia dự án nào", member, 0},
	}
	for _, c := range cases {
		pg, err := svc.List(ctx, c.actor, project.ListInput{}, domain.NewPageRequest(1, 20))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if pg.Total != c.want || pg.Items == nil {
			t.Errorf("%s: total = %d, want %d", c.name, pg.Total, c.want)
		}
	}
	if st.lastFilter.VisibleTo == nil || *st.lastFilter.VisibleTo != member.UserID {
		t.Fatal("Member phải bị lọc theo chính mình")
	}
	_, err := svc.List(ctx, admin, project.ListInput{}, domain.NewPageRequest(1, 20))
	expectErr(t, err, domain.ErrForbidden)
}

func TestList_FilterValidation_CF03(t *testing.T) {
	svc, _, _ := setup(t)
	ctx := context.Background()
	_, err := svc.List(ctx, ceo, project.ListInput{Search: strings.Repeat("x", 101)}, domain.NewPageRequest(1, 20))
	expectErr(t, err, domain.ErrInvalidSearch)
	_, err = svc.List(ctx, ceo, project.ListInput{Status: "xxx"}, domain.NewPageRequest(1, 20))
	expectErr(t, err, domain.ErrInvalidProjectStatus)
	pg, err := svc.List(ctx, ceo, project.ListInput{Status: "new", Search: " website "}, domain.NewPageRequest(1, 20))
	if err != nil || pg.Total != 1 {
		t.Fatalf("pg=%+v err=%v", pg, err)
	}
}

func TestGet_Visibility_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	for _, a := range []authz.Principal{mgrA, ceo} {
		if _, err := svc.Get(ctx, a, p.ID); err != nil {
			t.Fatalf("%v: %v", a.Role, err)
		}
	}
	// Không phải thành viên → 404 (không lộ sự tồn tại).
	for _, a := range []authz.Principal{mgrB, member, admin} {
		_, err := svc.Get(ctx, a, p.ID)
		expectErr(t, err, domain.ErrProjectNotFound)
	}
	st.members[p.ID][member.UserID] = true
	if _, err := svc.Get(ctx, member, p.ID); err != nil {
		t.Fatalf("member: %v", err)
	}
	_, err := svc.Get(ctx, ceo, uuid.New())
	expectErr(t, err, domain.ErrProjectNotFound)
}

// ---- sửa ----

func TestUpdate_OnlyProjectManager_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	st.members[p.ID][member.UserID] = true
	_, err := svc.Update(ctx, member, p.ID, project.Input{Name: "X"})
	expectErr(t, err, domain.ErrForbidden)
	_, err = svc.Update(ctx, ceo, p.ID, project.Input{Name: "X"})
	expectErr(t, err, domain.ErrForbidden)
	_, err = svc.Update(ctx, mgrB, p.ID, project.Input{Name: "X"})
	expectErr(t, err, domain.ErrProjectNotFound)

	got, err := svc.Update(ctx, mgrA, p.ID, project.Input{Name: "Mới", Status: "paused"})
	if err != nil || got.Name != "Mới" || got.Status != domain.ProjectPaused {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	// Không gửi trạng thái → giữ trạng thái cũ.
	got, err = svc.Update(ctx, mgrA, p.ID, project.Input{Name: "Mới"})
	if err != nil || got.Status != domain.ProjectPaused {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if got.ManagerID != mgrA.UserID {
		t.Fatal("không đổi Manager phụ trách")
	}
}

// ---- thành viên ----

func TestMembers_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	disabled := uuid.New()
	adminUser := uuid.New()
	st.users[member.UserID] = domain.MemberCandidate{UserID: member.UserID, Role: domain.RoleMember, Status: domain.UserStatusActive}
	st.users[mgrB.UserID] = domain.MemberCandidate{UserID: mgrB.UserID, Role: domain.RoleManager, Status: domain.UserStatusPending}
	st.users[disabled] = domain.MemberCandidate{UserID: disabled, Role: domain.RoleMember, Status: domain.UserStatusDisabled}
	st.users[adminUser] = domain.MemberCandidate{UserID: adminUser, Role: domain.RoleAdmin, Status: domain.UserStatusActive}

	if err := svc.AddMember(ctx, mgrA, p.ID, member.UserID); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddMember(ctx, mgrA, p.ID, member.UserID); err != nil {
		t.Fatalf("thêm lại phải idempotent: %v", err)
	}
	if err := svc.AddMember(ctx, mgrA, p.ID, mgrB.UserID); err != nil {
		t.Fatalf("manager chưa kích hoạt vẫn thêm được: %v", err)
	}
	for _, u := range []uuid.UUID{disabled, adminUser, uuid.New()} {
		expectErr(t, svc.AddMember(ctx, mgrA, p.ID, u), domain.ErrInvalidProjectMember)
	}
	// Member trong dự án xem được thành viên nhưng không thêm/bớt.
	ms, err := svc.ListMembers(ctx, member, p.ID)
	if err != nil || len(ms) != 3 {
		t.Fatalf("members=%d err=%v", len(ms), err)
	}
	expectErr(t, svc.AddMember(ctx, member, p.ID, member.UserID), domain.ErrForbidden)
	expectErr(t, svc.RemoveMember(ctx, member, p.ID, mgrB.UserID), domain.ErrForbidden)
	_, err = svc.ListCandidates(ctx, member, p.ID, "")
	expectErr(t, err, domain.ErrForbidden)
	_, err = svc.ListCandidates(ctx, mgrA, p.ID, strings.Repeat("x", 101))
	expectErr(t, err, domain.ErrInvalidSearch)

	expectErr(t, svc.RemoveMember(ctx, mgrA, p.ID, mgrA.UserID), domain.ErrCannotRemoveProjectManager)
	if err := svc.RemoveMember(ctx, mgrA, p.ID, member.UserID); err != nil {
		t.Fatal(err)
	}
	expectErr(t, svc.RemoveMember(ctx, mgrA, p.ID, member.UserID), domain.ErrProjectMemberNotFound)
	// Bị bỏ khỏi dự án → không còn thấy dự án.
	_, err = svc.Get(ctx, member, p.ID)
	expectErr(t, err, domain.ErrProjectNotFound)
}

// ---- cột Kanban ----

func TestAddColumn_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	b, err := svc.AddColumn(ctx, mgrA, p.ID, " Kiểm   thử ", "in_progress")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Columns) != 6 || b.Columns[5].Name != "Kiểm thử" || b.Columns[5].Position != 5 {
		t.Fatalf("cols = %+v", b.Columns)
	}
	if st.locks == 0 {
		t.Fatal("phải khóa dự án khi sửa cột")
	}
	newID := b.Columns[5].ID
	toNew := 0
	for _, tr := range b.Transitions {
		if tr.ToColumnID == newID {
			toNew++
		}
	}
	if toNew == 0 {
		t.Fatal("cột mới phải có luật chuyển mặc định")
	}
	if err := domain.ValidateTransitions(b.Columns, b.Transitions); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, col, group string
		want             error
	}{
		{"tên trùng (khác hoa thường)", "done", "done", domain.ErrColumnNameTaken},
		{"nhóm sai", "QA", "review", domain.ErrInvalidStatusGroup},
		{"tên rỗng", "  ", "todo", domain.ErrInvalidColumnName},
	}
	for _, c := range cases {
		_, err := svc.AddColumn(ctx, mgrA, p.ID, c.col, c.group)
		expectErr(t, err, c.want)
	}
	_, err = svc.AddColumn(ctx, mgrA, p.ID, "C7", "todo")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddColumn(ctx, mgrA, p.ID, "C8", "todo")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddColumn(ctx, mgrA, p.ID, "C9", "todo")
	expectErr(t, err, domain.ErrColumnCount)
	if len(st.columns[p.ID]) != 8 {
		t.Fatalf("cols = %d", len(st.columns[p.ID]))
	}
	_, err = svc.AddColumn(ctx, ceo, p.ID, "X", "todo")
	expectErr(t, err, domain.ErrForbidden)
}

func TestUpdateColumn_PrunesMemberIntoDone_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	review := colID(t, st, p.ID, "Chờ nghiệm thu")
	b, err := svc.UpdateColumn(ctx, mgrA, p.ID, review, "Đã xong", "done")
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range b.Transitions {
		if tr.ToColumnID == review && tr.Role == domain.TransitionMember {
			t.Fatal("Member không được có luật vào cột nhóm done")
		}
	}
	_, err = svc.UpdateColumn(ctx, mgrA, p.ID, review, "Backlog", "done")
	expectErr(t, err, domain.ErrColumnNameTaken)
	done := colID(t, st, p.ID, "Done")
	// Đổi cột done cuối cùng... vẫn còn "Đã xong" nên hợp lệ; đổi cả hai thì báo lỗi.
	if _, err := svc.UpdateColumn(ctx, mgrA, p.ID, done, "Done", "in_progress"); err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdateColumn(ctx, mgrA, p.ID, review, "Đã xong", "todo")
	expectErr(t, err, domain.ErrBoardNeedsDoneColumn)
	_, err = svc.UpdateColumn(ctx, mgrA, p.ID, uuid.New(), "Z", "todo")
	expectErr(t, err, domain.ErrColumnNotFound)
}

func TestDeleteColumn_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	todo := colID(t, st, p.ID, "To Do")
	b, err := svc.DeleteColumn(ctx, mgrA, p.ID, todo)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range b.Columns {
		if c.Position != i {
			t.Fatalf("vị trí phải liên tục: %+v", b.Columns)
		}
	}
	for _, tr := range b.Transitions {
		if tr.FromColumnID == todo || tr.ToColumnID == todo {
			t.Fatal("luật của cột đã xóa phải mất")
		}
	}
	// Cột còn việc → cảnh báo, không xóa.
	st.col(p.ID, colID(t, st, p.ID, "Backlog")).TaskCount = 2
	_, err = svc.DeleteColumn(ctx, mgrA, p.ID, colID(t, st, p.ID, "Backlog"))
	expectErr(t, err, domain.ErrColumnHasTasks)
	// Cột done cuối cùng.
	_, err = svc.DeleteColumn(ctx, mgrA, p.ID, colID(t, st, p.ID, "Done"))
	expectErr(t, err, domain.ErrBoardNeedsDoneColumn)
	// Tối thiểu 2 cột.
	if _, err := svc.DeleteColumn(ctx, mgrA, p.ID, colID(t, st, p.ID, "In Progress")); err != nil {
		t.Fatal(err)
	}
	st.col(p.ID, colID(t, st, p.ID, "Backlog")).TaskCount = 0
	_, err = svc.DeleteColumn(ctx, mgrA, p.ID, colID(t, st, p.ID, "Chờ nghiệm thu"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.DeleteColumn(ctx, mgrA, p.ID, colID(t, st, p.ID, "Backlog"))
	expectErr(t, err, domain.ErrColumnCount)
	st.members[p.ID][member.UserID] = true
	_, err = svc.DeleteColumn(ctx, member, p.ID, colID(t, st, p.ID, "Backlog"))
	expectErr(t, err, domain.ErrForbidden)
}

func TestReorderColumns_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	cols, _ := st.ListColumns(ctx, p.ID)
	ids := make([]uuid.UUID, 0, len(cols))
	for i := len(cols) - 1; i >= 0; i-- {
		ids = append(ids, cols[i].ID)
	}
	b, err := svc.ReorderColumns(ctx, mgrA, p.ID, ids)
	if err != nil {
		t.Fatal(err)
	}
	if b.Columns[0].Name != "Done" || b.Columns[4].Name != "Backlog" {
		t.Fatalf("cols = %+v", b.Columns)
	}
	_, err = svc.ReorderColumns(ctx, mgrA, p.ID, ids[:3])
	expectErr(t, err, domain.ErrInvalidColumnOrder)
	dup := append([]uuid.UUID{ids[0]}, ids[:4]...)
	_, err = svc.ReorderColumns(ctx, mgrA, p.ID, dup)
	expectErr(t, err, domain.ErrInvalidColumnOrder)
}

// ---- luật chuyển ----

func TestReplaceTransitions_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	backlog := colID(t, st, p.ID, "Backlog")
	done := colID(t, st, p.ID, "Done")
	ok := []domain.Transition{{FromColumnID: backlog, ToColumnID: done, Role: domain.TransitionManager}}
	b, err := svc.ReplaceTransitions(ctx, mgrA, p.ID, ok)
	if err != nil || len(b.Transitions) != 1 {
		t.Fatalf("b=%+v err=%v", b, err)
	}
	cases := []struct {
		name string
		ts   []domain.Transition
		want error
	}{
		{"Member vào Done", []domain.Transition{{FromColumnID: backlog, ToColumnID: done, Role: domain.TransitionMember}}, domain.ErrMemberTransitionToDone},
		{"cùng cột", []domain.Transition{{FromColumnID: backlog, ToColumnID: backlog, Role: domain.TransitionManager}}, domain.ErrInvalidTransition},
		{"cột lạ", []domain.Transition{{FromColumnID: backlog, ToColumnID: uuid.New(), Role: domain.TransitionManager}}, domain.ErrColumnNotFound},
		{"vai trò sai", []domain.Transition{{FromColumnID: backlog, ToColumnID: done, Role: "ceo"}}, domain.ErrInvalidTransitionRole},
		{"trùng", append(ok, ok...), domain.ErrDuplicateTransition},
	}
	for _, c := range cases {
		_, err := svc.ReplaceTransitions(ctx, mgrA, p.ID, c.ts)
		expectErr(t, err, c.want)
	}
	if len(st.transitions[p.ID]) != 1 {
		t.Fatal("luật không hợp lệ không được ghi")
	}
	_, err = svc.ReplaceTransitions(ctx, ceo, p.ID, ok)
	expectErr(t, err, domain.ErrForbidden)
	// Ai xem được dự án thì đọc được cấu hình.
	if _, err := svc.GetBoard(ctx, ceo, p.ID); err != nil {
		t.Fatal(err)
	}
	_, err = svc.GetBoard(ctx, member, p.ID)
	expectErr(t, err, domain.ErrProjectNotFound)
}

// ---- quyền ghi theo vai trò (CLAUDE.md §5.2: CEO không có thao tác ghi) ----

func TestWriteOps_ForbiddenForNonManager_CF03(t *testing.T) {
	svc, st, p := setup(t)
	ctx := context.Background()
	st.users[member.UserID] = domain.MemberCandidate{UserID: member.UserID, Role: domain.RoleMember, Status: domain.UserStatusActive}
	if err := svc.AddMember(ctx, mgrA, p.ID, member.UserID); err != nil {
		t.Fatal(err)
	}
	backlog := colID(t, st, p.ID, "Backlog")
	todo := colID(t, st, p.ID, "To Do")
	ids := make([]uuid.UUID, 0, len(st.columns[p.ID]))
	for _, c := range st.columns[p.ID] {
		ids = append(ids, c.ID)
	}
	ops := map[string]func(authz.Principal) error{
		"Update": func(a authz.Principal) error {
			_, err := svc.Update(ctx, a, p.ID, project.Input{Name: "X"})
			return err
		},
		"ListCandidates": func(a authz.Principal) error { _, err := svc.ListCandidates(ctx, a, p.ID, ""); return err },
		"AddMember":      func(a authz.Principal) error { return svc.AddMember(ctx, a, p.ID, uuid.New()) },
		"RemoveMember":   func(a authz.Principal) error { return svc.RemoveMember(ctx, a, p.ID, member.UserID) },
		"AddColumn": func(a authz.Principal) error {
			_, err := svc.AddColumn(ctx, a, p.ID, "Kiểm thử", "in_progress")
			return err
		},
		"UpdateColumn": func(a authz.Principal) error {
			_, err := svc.UpdateColumn(ctx, a, p.ID, todo, "Việc cần làm", "todo")
			return err
		},
		"DeleteColumn":   func(a authz.Principal) error { _, err := svc.DeleteColumn(ctx, a, p.ID, backlog); return err },
		"ReorderColumns": func(a authz.Principal) error { _, err := svc.ReorderColumns(ctx, a, p.ID, ids); return err },
		"ReplaceTransitions": func(a authz.Principal) error {
			_, err := svc.ReplaceTransitions(ctx, a, p.ID, nil)
			return err
		},
	}
	actors := []struct {
		name  string
		actor authz.Principal
		want  error
	}{
		{"CEO chỉ xem", ceo, domain.ErrForbidden},
		{"Member là thành viên", member, domain.ErrForbidden},
		{"Manager khác không thuộc dự án", mgrB, domain.ErrProjectNotFound},
		{"Admin", admin, domain.ErrProjectNotFound},
	}
	for _, a := range actors {
		for name, op := range ops {
			t.Run(a.name+"/"+name, func(t *testing.T) {
				expectErr(t, op(a.actor), a.want)
			})
		}
	}
	// Không thao tác nào ở trên được làm thay đổi bảng.
	if len(st.columns[p.ID]) != 5 {
		t.Fatalf("bảng bị đổi: %d cột", len(st.columns[p.ID]))
	}
}

// IDOR: dùng mã cột có thật của dự án B trên đường dẫn dự án A → không tìm thấy, dự án B không đổi.
func TestColumnOps_CrossProjectColumn_CF03(t *testing.T) {
	svc, st, pA := setup(t)
	ctx := context.Background()
	pB, err := svc.Create(ctx, mgrA, project.Input{Name: "Dự án B"})
	if err != nil {
		t.Fatal(err)
	}
	colB := colID(t, st, pB.ID, "To Do")
	backlogA := colID(t, st, pA.ID, "Backlog")

	_, err = svc.UpdateColumn(ctx, mgrA, pA.ID, colB, "Đổi tên", "todo")
	expectErr(t, err, domain.ErrColumnNotFound)
	_, err = svc.DeleteColumn(ctx, mgrA, pA.ID, colB)
	expectErr(t, err, domain.ErrColumnNotFound)
	_, err = svc.ReplaceTransitions(ctx, mgrA, pA.ID, []domain.Transition{{FromColumnID: backlogA, ToColumnID: colB, Role: domain.TransitionManager}})
	expectErr(t, err, domain.ErrColumnNotFound)
	_, err = svc.ReorderColumns(ctx, mgrA, pA.ID, []uuid.UUID{colB})
	expectErr(t, err, domain.ErrInvalidColumnOrder)

	if len(st.columns[pB.ID]) != 5 || st.col(pB.ID, colB).Name != "To Do" {
		t.Fatal("cột của dự án B bị thay đổi")
	}
}
