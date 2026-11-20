package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestParseProjectStatus_CF03(t *testing.T) {
	for _, s := range []string{"new", "in_progress", "completed", "paused", "cancelled"} {
		if got, err := domain.ParseProjectStatus(s); err != nil || string(got) != s {
			t.Fatalf("ParseProjectStatus(%q) = %q, %v", s, got, err)
		}
	}
	for _, s := range []string{"", "done", "NEW", "Đang tiến hành"} {
		if _, err := domain.ParseProjectStatus(s); !errors.Is(err, domain.ErrInvalidProjectStatus) {
			t.Fatalf("ParseProjectStatus(%q) err = %v, muốn ErrInvalidProjectStatus", s, err)
		}
	}
}

func TestNormalizeProjectName_CF03(t *testing.T) {
	tests := []struct {
		name, in, want string
		err            error
	}{
		{"hợp lệ", "Cổng thông tin khách hàng", "Cổng thông tin khách hàng", nil},
		{"gộp khoảng trắng", "  Tích hợp   VNPay ", "Tích hợp VNPay", nil},
		{"rỗng", "   ", "", domain.ErrInvalidProjectName},
		{"đúng 150 rune", strings.Repeat("ạ", 150), strings.Repeat("ạ", 150), nil},
		{"151 rune", strings.Repeat("a", 151), "", domain.ErrInvalidProjectName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NormalizeProjectName(tt.in)
			if !errors.Is(err, tt.err) || got != tt.want {
				t.Fatalf("got %q, %v; muốn %q, %v", got, err, tt.want, tt.err)
			}
		})
	}
}

func TestNormalizeProjectDescription_CF03(t *testing.T) {
	if got, err := domain.NormalizeProjectDescription("  Mô tả  \n"); err != nil || got != "Mô tả" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := domain.NormalizeProjectDescription(""); err != nil || got != "" {
		t.Fatalf("mô tả rỗng phải hợp lệ: %q, %v", got, err)
	}
	if _, err := domain.NormalizeProjectDescription(strings.Repeat("ạ", 2001)); !errors.Is(err, domain.ErrInvalidProjectDescription) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateProjectDates_CF03(t *testing.T) {
	d := func(s string) *time.Time {
		v, _ := time.Parse(time.DateOnly, s)
		return &v
	}
	tests := []struct {
		name       string
		start, end *time.Time
		err        error
	}{
		{"không có ngày", nil, nil, nil},
		{"chỉ ngày bắt đầu", d("2026-11-01"), nil, nil},
		{"chỉ ngày kết thúc", nil, d("2026-12-01"), nil},
		{"cùng ngày", d("2026-11-01"), d("2026-11-01"), nil},
		{"kết thúc trước bắt đầu", d("2026-11-02"), d("2026-11-01"), domain.ErrInvalidProjectDates},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.ValidateProjectDates(tt.start, tt.end); !errors.Is(err, tt.err) {
				t.Fatalf("err = %v, muốn %v", err, tt.err)
			}
		})
	}
}

func TestParseStatusGroup_CF03(t *testing.T) {
	for _, s := range []string{"todo", "in_progress", "done"} {
		if _, err := domain.ParseStatusGroup(s); err != nil {
			t.Fatalf("ParseStatusGroup(%q): %v", s, err)
		}
	}
	if _, err := domain.ParseStatusGroup("review"); !errors.Is(err, domain.ErrInvalidStatusGroup) {
		t.Fatalf("err = %v", err)
	}
}

func TestDefaultColumns_CF03(t *testing.T) {
	cols := domain.DefaultColumns()
	want := []struct {
		name  string
		group domain.StatusGroup
	}{
		{"Backlog", domain.GroupTodo},
		{"To Do", domain.GroupTodo},
		{"In Progress", domain.GroupInProgress},
		{"Chờ nghiệm thu", domain.GroupInProgress}, // "Chờ nghiệm thu" thuộc nhóm in_progress (CLAUDE.md §5.5)
		{"Done", domain.GroupDone},
	}
	if len(cols) != len(want) {
		t.Fatalf("len = %d", len(cols))
	}
	for i, w := range want {
		if cols[i].Name != w.name || cols[i].StatusGroup != w.group {
			t.Fatalf("cột %d = %+v, muốn %s/%s", i, cols[i], w.name, w.group)
		}
	}
	if err := domain.ValidateColumnSpecs(cols); err != nil {
		t.Fatalf("cột mặc định phải hợp lệ: %v", err)
	}
}

func TestNormalizeColumnName_CF03(t *testing.T) {
	if got, err := domain.NormalizeColumnName("  Đang   làm "); err != nil || got != "Đang làm" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, in := range []string{"", "  ", strings.Repeat("a", 51)} {
		if _, err := domain.NormalizeColumnName(in); !errors.Is(err, domain.ErrInvalidColumnName) {
			t.Fatalf("NormalizeColumnName(%q) err = %v", in, err)
		}
	}
}

func spec(name string, g domain.StatusGroup) domain.ColumnSpec {
	return domain.ColumnSpec{Name: name, StatusGroup: g}
}

func TestValidateColumnSpecs_CF03(t *testing.T) {
	eight := make([]domain.ColumnSpec, 0, 9)
	for i := range 7 {
		eight = append(eight, spec(strings.Repeat("c", i+1), domain.GroupTodo))
	}
	eight = append(eight, spec("Done", domain.GroupDone))
	nine := append(append([]domain.ColumnSpec{}, eight...), spec("Thêm", domain.GroupTodo))

	tests := []struct {
		name string
		cols []domain.ColumnSpec
		err  error
	}{
		{"2 cột tối thiểu", []domain.ColumnSpec{spec("Làm", domain.GroupTodo), spec("Xong", domain.GroupDone)}, nil},
		{"8 cột tối đa", eight, nil},
		{"1 cột", []domain.ColumnSpec{spec("Xong", domain.GroupDone)}, domain.ErrColumnCount},
		{"9 cột", nine, domain.ErrColumnCount},
		{"trùng tên không phân biệt hoa thường", []domain.ColumnSpec{spec("Done", domain.GroupTodo), spec("done", domain.GroupDone)}, domain.ErrColumnNameTaken},
		{"không có cột nhóm done", []domain.ColumnSpec{spec("A", domain.GroupTodo), spec("B", domain.GroupInProgress)}, domain.ErrBoardNeedsDoneColumn},
		{"nhóm sai", []domain.ColumnSpec{spec("A", "review"), spec("B", domain.GroupDone)}, domain.ErrInvalidStatusGroup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.ValidateColumnSpecs(tt.cols); !errors.Is(err, tt.err) {
				t.Fatalf("err = %v, muốn %v", err, tt.err)
			}
		})
	}
}

// board dựng 5 cột mặc định có ID để thử luật chuyển trạng thái.
func board() []domain.BoardColumn {
	specs := domain.DefaultColumns()
	cols := make([]domain.BoardColumn, len(specs))
	for i, s := range specs {
		cols[i] = domain.BoardColumn{ID: uuid.New(), Name: s.Name, Position: i, StatusGroup: s.StatusGroup}
	}
	return cols
}

func TestDefaultTransitions_CF03(t *testing.T) {
	cols := board()
	ts := domain.DefaultTransitions(cols)
	group := map[uuid.UUID]domain.StatusGroup{}
	for _, c := range cols {
		group[c.ID] = c.StatusGroup
	}
	var manager, member int
	for _, tr := range ts {
		if tr.FromColumnID == tr.ToColumnID {
			t.Fatal("không được có luật chuyển cột vào chính nó")
		}
		switch tr.Role {
		case domain.TransitionManager:
			manager++
		case domain.TransitionMember:
			member++
			// Chỉ Manager chuyển sang nhóm done (CLAUDE.md §5.5); Member không mở lại việc đã xong.
			if group[tr.ToColumnID] == domain.GroupDone || group[tr.FromColumnID] == domain.GroupDone {
				t.Fatalf("Member không được có luật chạm cột done: %+v", tr)
			}
		default:
			t.Fatalf("vai trò lạ %q", tr.Role)
		}
	}
	if manager != 5*4 {
		t.Fatalf("Manager có %d luật, muốn 20 (mọi cặp cột)", manager)
	}
	if member != 4*3 {
		t.Fatalf("Member có %d luật, muốn 12 (mọi cặp giữa 4 cột chưa xong)", member)
	}
	if err := domain.ValidateTransitions(cols, ts); err != nil {
		t.Fatalf("luật mặc định phải hợp lệ: %v", err)
	}
}

func TestValidateTransitions_CF03(t *testing.T) {
	cols := board()
	backlog, todo, done := cols[0].ID, cols[1].ID, cols[4].ID
	tr := func(from, to uuid.UUID, r domain.TransitionRole) domain.Transition {
		return domain.Transition{FromColumnID: from, ToColumnID: to, Role: r}
	}
	tests := []struct {
		name string
		ts   []domain.Transition
		err  error
	}{
		{"rỗng hợp lệ (khóa hết)", nil, nil},
		{"Manager sang done", []domain.Transition{tr(todo, done, domain.TransitionManager)}, nil},
		{"Member mở lại từ done về todo", []domain.Transition{tr(done, todo, domain.TransitionMember)}, nil},
		{"Member sang done bị chặn", []domain.Transition{tr(todo, done, domain.TransitionMember)}, domain.ErrMemberTransitionToDone},
		{"cùng cột", []domain.Transition{tr(todo, todo, domain.TransitionManager)}, domain.ErrInvalidTransition},
		{"cột không thuộc dự án", []domain.Transition{tr(todo, uuid.New(), domain.TransitionManager)}, domain.ErrColumnNotFound},
		{"vai trò sai", []domain.Transition{tr(backlog, todo, "ceo")}, domain.ErrInvalidTransitionRole},
		{"trùng luật", []domain.Transition{tr(backlog, todo, domain.TransitionMember), tr(backlog, todo, domain.TransitionMember)}, domain.ErrDuplicateTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.ValidateTransitions(cols, tt.ts); !errors.Is(err, tt.err) {
				t.Fatalf("err = %v, muốn %v", err, tt.err)
			}
		})
	}
}

func TestPruneTransitions_CF03(t *testing.T) {
	cols := board()
	todo, review := cols[1].ID, cols[3].ID
	ts := []domain.Transition{
		{FromColumnID: todo, ToColumnID: review, Role: domain.TransitionMember},
		{FromColumnID: todo, ToColumnID: review, Role: domain.TransitionManager},
		{FromColumnID: todo, ToColumnID: uuid.New(), Role: domain.TransitionManager}, // cột đã xóa
	}
	// Đổi "Chờ nghiệm thu" sang nhóm done → luật Member vào cột đó bị bỏ, luật cột không tồn tại cũng bỏ.
	cols[3].StatusGroup = domain.GroupDone
	got := domain.PruneTransitions(cols, ts)
	if len(got) != 1 || got[0].Role != domain.TransitionManager || got[0].ToColumnID != review {
		t.Fatalf("got %+v", got)
	}
}

func TestTransitionsForNewColumn_CF03(t *testing.T) {
	cols := board()
	added := domain.BoardColumn{ID: uuid.New(), Name: "Kiểm thử", Position: 5, StatusGroup: domain.GroupInProgress}
	cols = append(cols, added)
	got := domain.TransitionsForColumn(cols, added.ID)
	var manager, member int
	for _, tr := range got {
		if tr.FromColumnID != added.ID && tr.ToColumnID != added.ID {
			t.Fatalf("luật không chạm cột mới: %+v", tr)
		}
		if tr.Role == domain.TransitionManager {
			manager++
		} else {
			member++
		}
	}
	// 5 cột cũ × 2 chiều cho Manager; Member: 4 cột chưa xong × 2 chiều.
	if manager != 10 || member != 8 {
		t.Fatalf("manager=%d member=%d, muốn 10/8", manager, member)
	}
}

func TestValidateColumnOrder_CF03(t *testing.T) {
	cols := board()
	ids := make([]uuid.UUID, len(cols))
	for i, c := range cols {
		ids[len(cols)-1-i] = c.ID
	}
	if err := domain.ValidateColumnOrder(cols, ids); err != nil {
		t.Fatalf("đảo ngược phải hợp lệ: %v", err)
	}
	bad := [][]uuid.UUID{
		ids[:4],
		append(append([]uuid.UUID{}, ids[:4]...), ids[0]),
		append(append([]uuid.UUID{}, ids[:4]...), uuid.New()),
	}
	for i, b := range bad {
		if err := domain.ValidateColumnOrder(cols, b); !errors.Is(err, domain.ErrInvalidColumnOrder) {
			t.Fatalf("ca %d err = %v", i, err)
		}
	}
}

func TestCheckColumnDeletable_CF03(t *testing.T) {
	cols := board()
	if err := domain.CheckColumnDeletable(cols, cols[0].ID); err != nil {
		t.Fatalf("xóa Backlog phải được: %v", err)
	}
	if err := domain.CheckColumnDeletable(cols, uuid.New()); !errors.Is(err, domain.ErrColumnNotFound) {
		t.Fatalf("err = %v", err)
	}
	if err := domain.CheckColumnDeletable(cols, cols[4].ID); !errors.Is(err, domain.ErrBoardNeedsDoneColumn) {
		t.Fatalf("xóa cột done duy nhất: err = %v", err)
	}
	withTasks := board()
	withTasks[1].TaskCount = 3
	if err := domain.CheckColumnDeletable(withTasks, withTasks[1].ID); !errors.Is(err, domain.ErrColumnHasTasks) {
		t.Fatalf("cột có việc: err = %v", err)
	}
	two := []domain.BoardColumn{
		{ID: uuid.New(), Name: "A", StatusGroup: domain.GroupTodo},
		{ID: uuid.New(), Name: "B", StatusGroup: domain.GroupDone},
	}
	if err := domain.CheckColumnDeletable(two, two[0].ID); !errors.Is(err, domain.ErrColumnCount) {
		t.Fatalf("còn 2 cột: err = %v", err)
	}
}
