package domain

import (
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// CF-02: cơ cấu tổ chức — phòng ban dạng cây, chức danh, danh mục kỹ năng, hồ sơ kỹ năng.

// OrgNameMaxRunes giới hạn độ dài tên phòng ban, chức danh, kỹ năng.
const OrgNameMaxRunes = 100

// MaxUserSkills giới hạn số kỹ năng trong một hồ sơ (chặn payload quá lớn).
const MaxUserSkills = 50

// Department là một nút trong cây phòng ban. ParentID nil = phòng ban gốc.
type Department struct {
	ID          uuid.UUID
	Name        string
	ParentID    *uuid.UUID
	MemberCount int
}

// JobTitle là chức danh (vd. "Lập trình viên").
type JobTitle struct {
	ID          uuid.UUID
	Name        string
	MemberCount int
}

// Skill là một mục trong danh mục kỹ năng. MemberCount = số nhân sự có kỹ năng này.
type Skill struct {
	ID          uuid.UUID
	Name        string
	MemberCount int
}

// SkillLevel là cấp độ kỹ năng của nhân sự.
type SkillLevel string

const (
	SkillBeginner     SkillLevel = "beginner"
	SkillIntermediate SkillLevel = "intermediate"
	SkillSenior       SkillLevel = "senior"
)

// ParseSkillLevel chỉ chấp nhận đúng ba mã (phân biệt hoa thường).
func ParseSkillLevel(s string) (SkillLevel, error) {
	l := SkillLevel(s)
	switch l {
	case SkillBeginner, SkillIntermediate, SkillSenior:
		return l, nil
	default:
		return "", ErrInvalidSkillLevel
	}
}

// UserSkill là một kỹ năng trong hồ sơ nhân sự.
type UserSkill struct {
	SkillID uuid.UUID
	Name    string
	Level   SkillLevel
}

// UserSkillInput là một dòng hồ sơ kỹ năng người dùng gửi lên (chưa kiểm tra).
type UserSkillInput struct {
	SkillID uuid.UUID
	Level   string
}

// NormalizeOrgName cắt và gộp khoảng trắng; tên phải có 1..100 ký tự.
func NormalizeOrgName(s string) (string, error) {
	name := strings.Join(strings.Fields(s), " ")
	if name == "" || utf8.RuneCountInString(name) > OrgNameMaxRunes {
		return "", ErrInvalidOrgName
	}
	return name, nil
}

// ValidateDepartmentParent kiểm tra đặt phòng ban id dưới parent không tạo vòng.
// parentOf là toàn bộ cây hiện tại: id → cha (nil nếu gốc). id có thể chưa có trong cây (tạo mới).
func ValidateDepartmentParent(id uuid.UUID, parent *uuid.UUID, parentOf map[uuid.UUID]*uuid.UUID) error {
	if parent == nil {
		return nil
	}
	if _, ok := parentOf[*parent]; !ok {
		return ErrDepartmentNotFound
	}
	seen := make(map[uuid.UUID]bool, len(parentOf))
	for cur := parent; cur != nil; cur = parentOf[*cur] {
		if *cur == id || seen[*cur] {
			return ErrDepartmentCycle
		}
		seen[*cur] = true
	}
	return nil
}

// ValidateUserSkills kiểm tra hồ sơ kỹ năng: cấp độ hợp lệ, không trùng kỹ năng, tối đa MaxUserSkills.
func ValidateUserSkills(in []UserSkillInput) ([]UserSkill, error) {
	if len(in) > MaxUserSkills {
		return nil, ErrTooManySkills
	}
	out := make([]UserSkill, 0, len(in))
	seen := make(map[uuid.UUID]bool, len(in))
	for _, x := range in {
		lvl, err := ParseSkillLevel(x.Level)
		if err != nil {
			return nil, err
		}
		if seen[x.SkillID] {
			return nil, ErrDuplicateSkill
		}
		seen[x.SkillID] = true
		out = append(out, UserSkill{SkillID: x.SkillID, Level: lvl})
	}
	return out, nil
}
