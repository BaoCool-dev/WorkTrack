package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// CF-03: dự án, thành viên dự án, cột Kanban theo dự án và luật chuyển trạng thái.

const (
	// ProjectNameMaxRunes giới hạn độ dài tên dự án.
	ProjectNameMaxRunes = 150
	// ProjectDescriptionMaxRunes giới hạn độ dài mô tả dự án.
	ProjectDescriptionMaxRunes = 2000
	// ColumnNameMaxRunes giới hạn độ dài tên cột Kanban.
	ColumnNameMaxRunes = 50
	// MinBoardColumns / MaxBoardColumns: một bảng Kanban có 2–8 cột.
	MinBoardColumns = 2
	MaxBoardColumns = 8
)

// ProjectStatus là trạng thái dự án.
type ProjectStatus string

const (
	ProjectNew        ProjectStatus = "new"
	ProjectInProgress ProjectStatus = "in_progress"
	ProjectCompleted  ProjectStatus = "completed"
	ProjectPaused     ProjectStatus = "paused"
	ProjectCancelled  ProjectStatus = "cancelled"
)

// ParseProjectStatus chỉ chấp nhận đúng năm mã (phân biệt hoa thường).
func ParseProjectStatus(s string) (ProjectStatus, error) {
	st := ProjectStatus(s)
	switch st {
	case ProjectNew, ProjectInProgress, ProjectCompleted, ProjectPaused, ProjectCancelled:
		return st, nil
	default:
		return "", ErrInvalidProjectStatus
	}
}

// Project là một dự án. ManagerID là Manager phụ trách (người duy nhất được sửa cấu hình).
type Project struct {
	ID          uuid.UUID
	Name        string
	Description string
	Status      ProjectStatus
	StartDate   *time.Time
	EndDate     *time.Time
	ManagerID   uuid.UUID
	ManagerName string
	MemberCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProjectMember là một thành viên dự án.
type ProjectMember struct {
	UserID   uuid.UUID
	FullName string
	Email    string
	Role     Role
	Status   UserStatus
	AddedAt  time.Time
}

// MemberCandidate là nhân sự có thể thêm vào dự án.
type MemberCandidate struct {
	UserID   uuid.UUID
	FullName string
	Email    string
	Role     Role
	Status   UserStatus
}

// collapse gộp mọi khoảng trắng liên tiếp thành một dấu cách và cắt hai đầu.
func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }

// NormalizeProjectName gộp khoảng trắng; 1–150 ký tự.
func NormalizeProjectName(s string) (string, error) {
	n := collapse(s)
	if n == "" || utf8.RuneCountInString(n) > ProjectNameMaxRunes {
		return "", ErrInvalidProjectName
	}
	return n, nil
}

// NormalizeProjectDescription cắt khoảng trắng hai đầu; được rỗng, tối đa 2000 ký tự.
func NormalizeProjectDescription(s string) (string, error) {
	d := strings.TrimSpace(s)
	if utf8.RuneCountInString(d) > ProjectDescriptionMaxRunes {
		return "", ErrInvalidProjectDescription
	}
	return d, nil
}

// ValidateProjectDates: ngày kết thúc không trước ngày bắt đầu (cả hai đều tùy chọn).
func ValidateProjectDates(start, end *time.Time) error {
	if start != nil && end != nil && end.Before(*start) {
		return ErrInvalidProjectDates
	}
	return nil
}

// StatusGroup là nhóm trạng thái của một cột Kanban.
type StatusGroup string

const (
	GroupTodo       StatusGroup = "todo"
	GroupInProgress StatusGroup = "in_progress"
	GroupDone       StatusGroup = "done"
)

// ParseStatusGroup chỉ chấp nhận todo | in_progress | done.
func ParseStatusGroup(s string) (StatusGroup, error) {
	g := StatusGroup(s)
	if !g.valid() {
		return "", ErrInvalidStatusGroup
	}
	return g, nil
}

func (g StatusGroup) valid() bool {
	return g == GroupTodo || g == GroupInProgress || g == GroupDone
}

// ColumnSpec là tên + nhóm của một cột trước khi lưu.
type ColumnSpec struct {
	Name        string
	StatusGroup StatusGroup
}

// DefaultColumns là 5 cột mặc định khi tạo dự án. "Chờ nghiệm thu" thuộc nhóm in_progress.
func DefaultColumns() []ColumnSpec {
	return []ColumnSpec{
		{Name: "Backlog", StatusGroup: GroupTodo},
		{Name: "To Do", StatusGroup: GroupTodo},
		{Name: "In Progress", StatusGroup: GroupInProgress},
		{Name: "Chờ nghiệm thu", StatusGroup: GroupInProgress},
		{Name: "Done", StatusGroup: GroupDone},
	}
}

// NormalizeColumnName gộp khoảng trắng; 1–50 ký tự.
func NormalizeColumnName(s string) (string, error) {
	n := collapse(s)
	if n == "" || utf8.RuneCountInString(n) > ColumnNameMaxRunes {
		return "", ErrInvalidColumnName
	}
	return n, nil
}

// ValidateColumnSpecs kiểm tra toàn bộ cấu hình cột của một bảng: 2–8 cột, tên không trùng
// (không phân biệt hoa thường), nhóm hợp lệ, có ít nhất một cột nhóm done.
func ValidateColumnSpecs(cols []ColumnSpec) error {
	if len(cols) < MinBoardColumns || len(cols) > MaxBoardColumns {
		return ErrColumnCount
	}
	seen := make(map[string]struct{}, len(cols))
	hasDone := false
	for _, c := range cols {
		if !c.StatusGroup.valid() {
			return ErrInvalidStatusGroup
		}
		key := strings.ToLower(c.Name)
		if _, dup := seen[key]; dup {
			return ErrColumnNameTaken
		}
		seen[key] = struct{}{}
		if c.StatusGroup == GroupDone {
			hasDone = true
		}
	}
	if !hasDone {
		return ErrBoardNeedsDoneColumn
	}
	return nil
}

// BoardColumn là một cột Kanban đã lưu. TaskCount = số công việc đang nằm trong cột.
type BoardColumn struct {
	ID          uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Position    int
	StatusGroup StatusGroup
	TaskCount   int
}

// Specs trả tên + nhóm của các cột (để dùng chung ValidateColumnSpecs).
func Specs(cols []BoardColumn) []ColumnSpec {
	out := make([]ColumnSpec, len(cols))
	for i, c := range cols {
		out[i] = ColumnSpec{Name: c.Name, StatusGroup: c.StatusGroup}
	}
	return out
}

// TransitionRole là vai trò áp dụng một luật chuyển trạng thái.
type TransitionRole string

const (
	TransitionManager TransitionRole = "manager"
	TransitionMember  TransitionRole = "member"
)

// Transition: vai trò Role được kéo thẻ từ cột From sang cột To.
type Transition struct {
	FromColumnID uuid.UUID
	ToColumnID   uuid.UUID
	Role         TransitionRole
}

func groupsOf(cols []BoardColumn) map[uuid.UUID]StatusGroup {
	m := make(map[uuid.UUID]StatusGroup, len(cols))
	for _, c := range cols {
		m[c.ID] = c.StatusGroup
	}
	return m
}

// DefaultTransitions: Manager được chuyển giữa mọi cặp cột; Member được chuyển giữa mọi cặp
// cột chưa thuộc nhóm done (chỉ Manager đưa việc sang/khỏi nhóm hoàn thành theo mặc định).
func DefaultTransitions(cols []BoardColumn) []Transition {
	var out []Transition
	for _, from := range cols {
		for _, to := range cols {
			if from.ID == to.ID {
				continue
			}
			out = append(out, Transition{FromColumnID: from.ID, ToColumnID: to.ID, Role: TransitionManager})
			if from.StatusGroup != GroupDone && to.StatusGroup != GroupDone {
				out = append(out, Transition{FromColumnID: from.ID, ToColumnID: to.ID, Role: TransitionMember})
			}
		}
	}
	return out
}

// ValidateTransitions kiểm tra bộ luật do Manager gửi lên. Danh sách rỗng hợp lệ (khóa mọi chuyển).
func ValidateTransitions(cols []BoardColumn, ts []Transition) error {
	groups := groupsOf(cols)
	seen := make(map[Transition]struct{}, len(ts))
	for _, t := range ts {
		if t.FromColumnID == t.ToColumnID {
			return ErrInvalidTransition
		}
		fromGroup, okFrom := groups[t.FromColumnID]
		toGroup, okTo := groups[t.ToColumnID]
		if !okFrom || !okTo {
			return ErrColumnNotFound
		}
		if t.Role != TransitionManager && t.Role != TransitionMember {
			return ErrInvalidTransitionRole
		}
		// Chỉ Manager chuyển công việc vào/ra nhóm hoàn thành (CLAUDE.md §5.5, nghiệm thu thuộc Manager).
		if t.Role == TransitionMember && (toGroup == GroupDone || fromGroup == GroupDone) {
			return ErrMemberTransitionToDone
		}
		if _, dup := seen[t]; dup {
			return ErrDuplicateTransition
		}
		seen[t] = struct{}{}
	}
	return nil
}

// PruneTransitions bỏ luật trỏ tới cột không còn tồn tại và luật Member đi vào/ra cột nhóm done
// (dùng sau khi xóa cột hoặc đổi nhóm của cột).
func PruneTransitions(cols []BoardColumn, ts []Transition) []Transition {
	groups := groupsOf(cols)
	out := make([]Transition, 0, len(ts))
	for _, t := range ts {
		fromGroup, okFrom := groups[t.FromColumnID]
		toGroup, okTo := groups[t.ToColumnID]
		if !okFrom || !okTo {
			continue
		}
		if t.Role == TransitionMember && (toGroup == GroupDone || fromGroup == GroupDone) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// TransitionsForColumn sinh luật mặc định cho một cột vừa thêm: Manager hai chiều với mọi cột khác;
// Member hai chiều với mọi cột chưa xong, chỉ khi bản thân cột mới không thuộc nhóm done.
func TransitionsForColumn(cols []BoardColumn, newID uuid.UUID) []Transition {
	groups := groupsOf(cols)
	newGroup, ok := groups[newID]
	if !ok {
		return nil
	}
	var out []Transition
	for _, c := range cols {
		if c.ID == newID {
			continue
		}
		out = append(out,
			Transition{FromColumnID: newID, ToColumnID: c.ID, Role: TransitionManager},
			Transition{FromColumnID: c.ID, ToColumnID: newID, Role: TransitionManager},
		)
		if newGroup != GroupDone && c.StatusGroup != GroupDone {
			out = append(out,
				Transition{FromColumnID: newID, ToColumnID: c.ID, Role: TransitionMember},
				Transition{FromColumnID: c.ID, ToColumnID: newID, Role: TransitionMember},
			)
		}
	}
	return out
}

// ValidateColumnOrder: ids phải là hoán vị đúng của các cột hiện có.
func ValidateColumnOrder(cols []BoardColumn, ids []uuid.UUID) error {
	if len(ids) != len(cols) {
		return ErrInvalidColumnOrder
	}
	groups := groupsOf(cols)
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := groups[id]; !ok {
			return ErrInvalidColumnOrder
		}
		if _, dup := seen[id]; dup {
			return ErrInvalidColumnOrder
		}
		seen[id] = struct{}{}
	}
	return nil
}

// CheckColumnDeletable: còn tối thiểu 2 cột, còn ít nhất một cột done, cột không còn công việc.
func CheckColumnDeletable(cols []BoardColumn, id uuid.UUID) error {
	idx := -1
	doneCount := 0
	for i, c := range cols {
		if c.ID == id {
			idx = i
		}
		if c.StatusGroup == GroupDone {
			doneCount++
		}
	}
	if idx < 0 {
		return ErrColumnNotFound
	}
	if len(cols) <= MinBoardColumns {
		return ErrColumnCount
	}
	if cols[idx].StatusGroup == GroupDone && doneCount == 1 {
		return ErrBoardNeedsDoneColumn
	}
	if cols[idx].TaskCount > 0 {
		return ErrColumnHasTasks
	}
	return nil
}
