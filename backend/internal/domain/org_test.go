package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestNormalizeOrgName_CF02(t *testing.T) {
	tests := []struct {
		name, in, want string
		err            error
	}{
		{"giữ nguyên tên hợp lệ", "Phòng Kỹ thuật", "Phòng Kỹ thuật", nil},
		{"cắt khoảng trắng hai đầu", "  Backend  ", "Backend", nil},
		{"gộp khoảng trắng giữa", "Kiểm   thử\tphần mềm", "Kiểm thử phần mềm", nil},
		{"rỗng", "", "", domain.ErrInvalidOrgName},
		{"chỉ khoảng trắng", "   ", "", domain.ErrInvalidOrgName},
		{"đúng 100 ký tự (tính theo rune)", strings.Repeat("ạ", 100), strings.Repeat("ạ", 100), nil},
		{"101 ký tự", strings.Repeat("a", 101), "", domain.ErrInvalidOrgName},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NormalizeOrgName(tt.in)
			if !errors.Is(err, tt.err) || got != tt.want {
				t.Fatalf("NormalizeOrgName(%q) = %q, %v; want %q, %v", tt.in, got, err, tt.want, tt.err)
			}
		})
	}
}

func TestParseSkillLevel_CF02(t *testing.T) {
	for _, s := range []string{"beginner", "intermediate", "senior"} {
		if l, err := domain.ParseSkillLevel(s); err != nil || string(l) != s {
			t.Fatalf("ParseSkillLevel(%q) = %q, %v", s, l, err)
		}
	}
	for _, s := range []string{"", "expert", "Senior"} {
		if _, err := domain.ParseSkillLevel(s); !errors.Is(err, domain.ErrInvalidSkillLevel) {
			t.Fatalf("ParseSkillLevel(%q) err = %v, want ErrInvalidSkillLevel", s, err)
		}
	}
}

func TestValidateDepartmentParent_CF02(t *testing.T) {
	// Cây: root → a → b; c độc lập.
	root, a, b, c := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	newID := uuid.New()
	tree := map[uuid.UUID]*uuid.UUID{root: nil, a: &root, b: &a, c: nil}
	ptr := func(id uuid.UUID) *uuid.UUID { return &id }

	tests := []struct {
		name   string
		id     uuid.UUID
		parent *uuid.UUID
		want   error
	}{
		{"không có cha (gốc)", a, nil, nil},
		{"phòng ban mới dưới a", newID, ptr(a), nil},
		{"chuyển b sang c", b, ptr(c), nil},
		{"chuyển a lên gốc khác", a, ptr(c), nil},
		{"chính nó làm cha", a, ptr(a), domain.ErrDepartmentCycle},
		{"con trực tiếp làm cha", a, ptr(b), domain.ErrDepartmentCycle},
		{"cháu làm cha của gốc", root, ptr(b), domain.ErrDepartmentCycle},
		{"cha không tồn tại", b, ptr(uuid.New()), domain.ErrDepartmentNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.ValidateDepartmentParent(tt.id, tt.parent, tree); !errors.Is(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateDepartmentParent_CorruptLoopTerminates_CF02(t *testing.T) {
	// Dữ liệu hỏng (x ↔ y) không được làm vòng lặp vô hạn.
	x, y, z := uuid.New(), uuid.New(), uuid.New()
	tree := map[uuid.UUID]*uuid.UUID{x: &y, y: &x, z: nil}
	if err := domain.ValidateDepartmentParent(z, &x, tree); !errors.Is(err, domain.ErrDepartmentCycle) {
		t.Fatalf("got %v, want ErrDepartmentCycle", err)
	}
}

func TestValidateUserSkills_CF02(t *testing.T) {
	s1, s2 := uuid.New(), uuid.New()
	ok := []domain.UserSkillInput{{SkillID: s1, Level: "beginner"}, {SkillID: s2, Level: "senior"}}
	if _, err := domain.ValidateUserSkills(ok); err != nil {
		t.Fatalf("hợp lệ nhưng lỗi: %v", err)
	}
	if got, _ := domain.ValidateUserSkills(nil); got == nil || len(got) != 0 {
		t.Fatalf("danh sách rỗng phải trả slice rỗng, got %v", got)
	}
	dup := []domain.UserSkillInput{{SkillID: s1, Level: "beginner"}, {SkillID: s1, Level: "senior"}}
	if _, err := domain.ValidateUserSkills(dup); !errors.Is(err, domain.ErrDuplicateSkill) {
		t.Fatalf("trùng kỹ năng: got %v", err)
	}
	bad := []domain.UserSkillInput{{SkillID: s1, Level: "expert"}}
	if _, err := domain.ValidateUserSkills(bad); !errors.Is(err, domain.ErrInvalidSkillLevel) {
		t.Fatalf("cấp độ sai: got %v", err)
	}
	tooMany := make([]domain.UserSkillInput, domain.MaxUserSkills+1)
	for i := range tooMany {
		tooMany[i] = domain.UserSkillInput{SkillID: uuid.New(), Level: "beginner"}
	}
	if _, err := domain.ValidateUserSkills(tooMany); !errors.Is(err, domain.ErrTooManySkills) {
		t.Fatalf("quá nhiều kỹ năng: got %v", err)
	}
}
