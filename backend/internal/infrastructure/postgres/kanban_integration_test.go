//go:build integration

package postgres

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/kanban"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
)

// kanbanEnv dựng Service thật (project, task, kanban) trên Postgres với một dự án 5 cột mặc định.
type kanbanEnv struct {
	db       *pgxpool.Pool // truy vấn SQL thô để dựng dữ liệu và kiểm bảng lịch sử
	repo     *TaskRepo
	kanban   *kanban.Service
	tasks    *task.Service
	projects *project.Service
	mgr, mem authz.Principal
	p        domain.Project
	cols     map[string]uuid.UUID // tên cột → id
}

func newKanbanEnv(t *testing.T) *kanbanEnv {
	t.Helper()
	pool := newTestPool(t)
	ctx := t.Context()
	users := NewUserRepo(pool)
	e := &kanbanEnv{
		db:   pool,
		repo: NewTaskRepo(pool),
		mgr:  authz.Principal{UserID: newIntegrationUser(t, users, "mgr@worktrack.local", domain.RoleManager), Role: domain.RoleManager},
		mem:  authz.Principal{UserID: newIntegrationUser(t, users, "mem@worktrack.local", domain.RoleMember), Role: domain.RoleMember},
		cols: map[string]uuid.UUID{},
	}
	hub := realtime.NewHub()
	e.projects = project.NewService(NewProjectRepo(pool), NewProjectTx(pool))
	e.tasks = task.NewService(e.repo, NewTaskTx(pool), e.projects, setting.NewService(NewSettingRepo(pool)), hub)
	e.kanban = kanban.NewService(e.repo, NewKanbanTx(pool), e.projects, hub)

	var err error
	if e.p, err = e.projects.Create(ctx, e.mgr, project.Input{Name: "Bảng thử"}); err != nil {
		t.Fatal(err)
	}
	if err := e.projects.AddMember(ctx, e.mgr, e.p.ID, e.mem.UserID); err != nil {
		t.Fatal(err)
	}
	board, err := e.projects.GetBoard(ctx, e.mgr, e.p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range board.Columns {
		e.cols[c.Name] = c.ID
	}
	return e
}

// newTask tạo công việc cha (rơi vào cột todo đầu tiên) giao cho Member.
func (e *kanbanEnv) newTask(t *testing.T, title string) domain.Task {
	t.Helper()
	uid := e.mem.UserID
	tk, err := e.tasks.Create(t.Context(), e.mgr, e.p.ID, task.Input{Title: title, AssigneeID: &uid})
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func (e *kanbanEnv) move(t *testing.T, actor authz.Principal, id uuid.UUID, col string, pos int) (domain.Task, error) {
	t.Helper()
	cur, err := e.repo.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	return e.kanban.Move(t.Context(), actor, id, kanban.MoveInput{ColumnID: e.cols[col], Position: pos, Version: cur.Version})
}

// WIP chỉ đếm công việc cha nhóm in_progress (CLAUDE.md §5.6) — kiểm câu SQL CountProjectWIP thật.
func TestKanbanRepo_CountProjectWIP_ExcludesSubtasks_CF05(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	parent := e.newTask(t, "Việc cha")
	if _, err := e.move(t, e.mgr, parent.ID, "In Progress", 0); err != nil {
		t.Fatal(err)
	}
	// Việc con cùng người nhận, cùng cột in_progress: không được tính.
	if _, err := e.db.Exec(ctx, `INSERT INTO task (project_id, column_id, parent_task_id, title, assignee_id, created_by)
		VALUES ($1, $2, $3, 'Việc con', $4, $4)`, e.p.ID, e.cols["In Progress"], parent.ID, e.mem.UserID); err != nil {
		t.Fatal(err)
	}
	n, err := e.repo.CountProjectWIP(ctx, e.p.ID, e.mem.UserID)
	if err != nil || n != 1 {
		t.Fatalf("WIP = %d %v, want 1 (chỉ việc cha)", n, err)
	}
	// "Chờ nghiệm thu" vẫn thuộc nhóm in_progress (§5.5) → vẫn đếm.
	if _, err := e.move(t, e.mgr, parent.ID, "Chờ nghiệm thu", 0); err != nil {
		t.Fatal(err)
	}
	if n, _ := e.repo.CountProjectWIP(ctx, e.p.ID, e.mem.UserID); n != 1 {
		t.Fatalf("WIP ở Chờ nghiệm thu = %d, want 1", n)
	}
}

// MoveTaskColumn chỉ ghi khi version khớp; version cũ → 0 dòng → ErrTaskVersionConflict.
func TestKanbanRepo_MoveTaskColumn_StaleVersionNoRows_CF04(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	tk := e.newTask(t, "Xung đột")
	err := e.repo.MoveColumn(ctx, tk.ID, e.cols["In Progress"], tk.Version+1)
	if !errors.Is(err, domain.ErrTaskVersionConflict) {
		t.Fatalf("version lệch: %v", err)
	}
	if err := e.repo.MoveColumn(ctx, tk.ID, e.cols["In Progress"], tk.Version); err != nil {
		t.Fatal(err)
	}
	got, _ := e.repo.Get(ctx, tk.ID)
	if got.Version != tk.Version+1 || got.ColumnID != e.cols["In Progress"] {
		t.Fatalf("sau khi chuyển: version %d cột %v", got.Version, got.ColumnID)
	}
	// Qua service: client giữ version cũ nhận 409, không ghi đè.
	_, err = e.kanban.Move(ctx, e.mgr, tk.ID, kanban.MoveInput{ColumnID: e.cols["To Do"], Position: 0, Version: tk.Version})
	if !errors.Is(err, domain.ErrTaskVersionConflict) {
		t.Fatalf("service với version cũ: %v", err)
	}
}

// Luồng kéo thật: vị trí đánh lại 0..n-1 ở cột đích, lịch sử chuyển cột chỉ ghi khi đổi cột.
func TestKanbanRepo_MoveFlow_PositionsAndHistory_CF04(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	a, b, c := e.newTask(t, "A"), e.newTask(t, "B"), e.newTask(t, "C")
	for _, tk := range []domain.Task{a, b} {
		if _, err := e.move(t, e.mgr, tk.ID, "To Do", 99); err != nil { // 99 → cuối cột
			t.Fatal(err)
		}
	}
	// C chen lên đầu cột To Do.
	if _, err := e.move(t, e.mgr, c.ID, "To Do", 0); err != nil {
		t.Fatal(err)
	}
	ids, err := e.repo.ColumnTaskIDs(ctx, e.p.ID, e.cols["To Do"])
	if err != nil {
		t.Fatal(err)
	}
	want := []uuid.UUID{c.ID, a.ID, b.ID}
	if len(ids) != len(want) {
		t.Fatalf("cột To Do có %d thẻ, want %d", len(ids), len(want))
	}
	for i, id := range want {
		got, _ := e.repo.Get(ctx, id)
		if ids[i] != id || got.Position != i {
			t.Fatalf("vị trí %d: id %v pos %d", i, ids[i], got.Position)
		}
	}
	// Đổi thứ tự trong cùng cột: không thêm lịch sử chuyển cột.
	if _, err := e.move(t, e.mem, b.ID, "To Do", 0); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := e.db.QueryRow(ctx, `SELECT count(*) FROM task_move_history WHERE project_id = $1`, e.p.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("lịch sử = %d dòng, want 3 (chỉ lần đổi cột)", n)
	}
	var from, to, by uuid.UUID
	if err := e.db.QueryRow(ctx, `SELECT from_column_id, to_column_id, moved_by FROM task_move_history
		WHERE task_id = $1`, c.ID).Scan(&from, &to, &by); err != nil {
		t.Fatal(err)
	}
	if from != e.cols["Backlog"] || to != e.cols["To Do"] || by != e.mgr.UserID {
		t.Fatalf("lịch sử của C sai: %v → %v bởi %v", from, to, by)
	}
	// Member kéo vào Done bị chặn, dữ liệu giữ nguyên.
	if _, err := e.move(t, e.mem, a.ID, "Done", 0); err == nil {
		t.Fatal("Member kéo vào Done phải bị chặn")
	}
}

// Hai lần kéo song song vào in_progress khi chỉ còn 1 chỗ WIP: khóa dòng dự án bắt chúng xếp hàng,
// nên đúng một lần thành công và lần kia nhận ErrWIPLimitReached.
func TestKanbanRepo_ConcurrentMoves_WIPNotExceeded_CF05(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	if _, err := e.projects.SetWIPLimit(ctx, e.mgr, e.p.ID, 1); err != nil {
		t.Fatal(err)
	}
	x, y := e.newTask(t, "X"), e.newTask(t, "Y")

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, tk := range []domain.Task{x, y} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[i] = e.kanban.Move(ctx, e.mem, tk.ID,
				kanban.MoveInput{ColumnID: e.cols["In Progress"], Position: 0, Version: tk.Version})
		}()
	}
	wg.Wait()

	ok, blocked := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, domain.ErrWIPLimitReached):
			blocked++
		default:
			t.Fatalf("lỗi lạ: %v", err)
		}
	}
	if ok != 1 || blocked != 1 {
		t.Fatalf("thành công %d, bị chặn %d; want 1 và 1", ok, blocked)
	}
	if n, _ := e.repo.CountProjectWIP(ctx, e.p.ID, e.mem.UserID); n != 1 {
		t.Fatalf("WIP sau hai lần kéo = %d, want 1", n)
	}
}

// LockProject đọc giới hạn WIP trên dòng đã khóa: đổi giới hạn rồi khóa trong transaction thấy số mới.
func TestKanbanRepo_LockProjectReturnsCurrentLimit_CF05(t *testing.T) {
	e := newKanbanEnv(t)
	ctx := t.Context()
	if _, err := e.projects.SetWIPLimit(ctx, e.mgr, e.p.ID, 3); err != nil {
		t.Fatal(err)
	}
	lp, err := e.repo.LockProject(ctx, e.p.ID) // ngoài transaction FOR UPDATE vẫn chạy được
	if err != nil || lp.WIPLimit != 3 || lp.ManagerID != e.mgr.UserID {
		t.Fatalf("LockProject = %+v %v", lp, err)
	}
	if _, err := e.repo.LockProject(ctx, uuid.New()); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("dự án không tồn tại: %v", err)
	}
}
