package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// CF-06 — công việc cha, nhãn, tham số quy đổi Story Point (B1).
const (
	TaskTitleMaxRunes       = 200
	TaskDescriptionMaxRunes = 5000
	TaskLabelsMax           = 10
	// SettingStoryPointHours là khóa tham số B1: số giờ cho 1 Story Point.
	SettingStoryPointHours = "story_point_hours"
	StoryPointHoursMin     = 1
	StoryPointHoursMax     = 40
)

// storyPoints là thang Fibonacci cho phép (giả định mục C, 2026-11-20).
var storyPoints = map[int]bool{1: true, 2: true, 3: true, 5: true, 8: true, 13: true, 21: true}

// TaskLabel là nhãn gắn trên một công việc.
type TaskLabel struct {
	ID   uuid.UUID
	Name string
}

// Label là một mục trong danh mục nhãn, kèm số công việc đang gắn.
type Label struct {
	ID        uuid.UUID
	Name      string
	TaskCount int
}

// Task là công việc (CF-06 chỉ dùng công việc cha: ParentID == nil).
type Task struct {
	ID               uuid.UUID
	ProjectID        uuid.UUID
	ProjectName      string
	ParentID         *uuid.UUID
	ColumnID         uuid.UUID
	ColumnName       string
	StatusGroup      StatusGroup
	Title            string
	Description      string
	StoryPoint       *int
	EstimatedMinutes *int
	AssigneeID       *uuid.UUID
	AssigneeName     string
	DueDate          *time.Time
	Labels           []TaskLabel
	CreatedBy        uuid.UUID
	CreatedByName    string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	// Position: thứ tự trong cột (0 = trên cùng). Version tăng mỗi lần thay đổi — chống ghi đè (CF-04).
	Position int
	Version  int
	// Giờ thực tế (CF-08), tính sẵn trong task_view:
	// LoggedMinutes là log ghi trực tiếp trên việc này; ActualMinutes = LoggedMinutes + tổng log của các con.
	LoggedMinutes int
	ActualMinutes int
	// SubtaskCount / SubtaskDoneCount: số con và số con ở nhóm done (dòng "2/3 con" trên thẻ).
	SubtaskCount     int
	SubtaskDoneCount int
}

// NormalizeTaskTitle gộp khoảng trắng; 1–200 ký tự.
func NormalizeTaskTitle(s string) (string, error) {
	n := collapse(s)
	if n == "" || utf8.RuneCountInString(n) > TaskTitleMaxRunes {
		return "", ErrInvalidTaskTitle
	}
	return n, nil
}

// NormalizeTaskDescription cắt khoảng trắng hai đầu, giữ xuống dòng; được rỗng, tối đa 5000 ký tự.
func NormalizeTaskDescription(s string) (string, error) {
	d := strings.TrimSpace(s)
	if utf8.RuneCountInString(d) > TaskDescriptionMaxRunes {
		return "", ErrInvalidTaskDescription
	}
	return d, nil
}

// ValidateStoryPoint: Story Point phải thuộc thang Fibonacci 1–21.
func ValidateStoryPoint(sp int) error {
	if !storyPoints[sp] {
		return ErrInvalidStoryPoint
	}
	return nil
}

// EstimateMinutes quy đổi Story Point sang phút ước tính theo hệ số B1 (giờ / SP).
func EstimateMinutes(storyPoint, hoursPerPoint int) int { return storyPoint * hoursPerPoint * 60 }

// ValidateStoryPointHours: hệ số B1 trong khoảng 1–40 giờ.
func ValidateStoryPointHours(h int) error {
	if h < StoryPointHoursMin || h > StoryPointHoursMax {
		return ErrInvalidSetting
	}
	return nil
}

// NormalizeLabelIDs bỏ trùng (giữ thứ tự), tối đa 10 nhãn; luôn trả slice khác nil.
func NormalizeLabelIDs(ids []uuid.UUID) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(ids))
	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			return nil, ErrLabelNotFound
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	if len(out) > TaskLabelsMax {
		return nil, ErrTooManyLabels
	}
	return out, nil
}

// CanBeAssigned: chỉ Manager/Member chưa bị khóa được nhận công việc (cùng điều kiện làm thành viên dự án).
func CanBeAssigned(role Role, status UserStatus) bool {
	return (role == RoleManager || role == RoleMember) && status != UserStatusDisabled
}
