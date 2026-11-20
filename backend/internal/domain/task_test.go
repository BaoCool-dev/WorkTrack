package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestNormalizeTaskTitle_CF06(t *testing.T) {
	tests := []struct {
		name, in, want string
		err            error
	}{
		{"hợp lệ", "Thiết kế màn hình đăng nhập", "Thiết kế màn hình đăng nhập", nil},
		{"gộp khoảng trắng", "  API   danh sách\tviệc ", "API danh sách việc", nil},
		{"rỗng", "", "", domain.ErrInvalidTaskTitle},
		{"chỉ khoảng trắng", " \t ", "", domain.ErrInvalidTaskTitle},
		{"đúng 200 rune", strings.Repeat("ộ", 200), strings.Repeat("ộ", 200), nil},
		{"201 rune", strings.Repeat("a", 201), "", domain.ErrInvalidTaskTitle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NormalizeTaskTitle(tt.in)
			if !errors.Is(err, tt.err) || got != tt.want {
				t.Fatalf("NormalizeTaskTitle(%q) = %q, %v; want %q, %v", tt.in, got, err, tt.want, tt.err)
			}
		})
	}
}

func TestNormalizeTaskDescription_CF06(t *testing.T) {
	if got, err := domain.NormalizeTaskDescription("  mô tả\n  dòng 2  "); err != nil || got != "mô tả\n  dòng 2" {
		t.Fatalf("got %q, %v", got, err)
	}
	if got, err := domain.NormalizeTaskDescription(""); err != nil || got != "" {
		t.Fatalf("rỗng phải hợp lệ, got %q, %v", got, err)
	}
	if _, err := domain.NormalizeTaskDescription(strings.Repeat("ạ", 5000)); err != nil {
		t.Fatalf("5000 rune phải hợp lệ: %v", err)
	}
	if _, err := domain.NormalizeTaskDescription(strings.Repeat("a", 5001)); !errors.Is(err, domain.ErrInvalidTaskDescription) {
		t.Fatalf("5001 rune phải lỗi, got %v", err)
	}
}

func TestValidateStoryPoint_CF06(t *testing.T) {
	for _, sp := range []int{1, 2, 3, 5, 8, 13, 21} {
		if err := domain.ValidateStoryPoint(sp); err != nil {
			t.Fatalf("SP %d phải hợp lệ: %v", sp, err)
		}
	}
	for _, sp := range []int{-1, 0, 4, 6, 20, 34, 100} {
		if err := domain.ValidateStoryPoint(sp); !errors.Is(err, domain.ErrInvalidStoryPoint) {
			t.Fatalf("SP %d phải lỗi, got %v", sp, err)
		}
	}
}

func TestEstimateMinutes_CF06_B1(t *testing.T) {
	tests := []struct {
		sp, hours, want int
	}{
		{1, 4, 240}, // B1 mặc định: 1 SP = 4 giờ
		{3, 4, 720}, // 12 giờ
		{21, 4, 5040},
		{5, 1, 300},
		{8, 40, 19200},
	}
	for _, tt := range tests {
		if got := domain.EstimateMinutes(tt.sp, tt.hours); got != tt.want {
			t.Fatalf("EstimateMinutes(%d, %d) = %d; want %d", tt.sp, tt.hours, got, tt.want)
		}
	}
}

func TestValidateStoryPointHours_CF06_B1(t *testing.T) {
	for _, h := range []int{1, 4, 40} {
		if err := domain.ValidateStoryPointHours(h); err != nil {
			t.Fatalf("%d giờ phải hợp lệ: %v", h, err)
		}
	}
	for _, h := range []int{-4, 0, 41} {
		if err := domain.ValidateStoryPointHours(h); !errors.Is(err, domain.ErrInvalidSetting) {
			t.Fatalf("%d giờ phải lỗi, got %v", h, err)
		}
	}
}

func TestNormalizeLabelIDs_CF06(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	got, err := domain.NormalizeLabelIDs([]uuid.UUID{a, b, a})
	if err != nil || len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("bỏ trùng giữ thứ tự: got %v, %v", got, err)
	}
	if got, err := domain.NormalizeLabelIDs(nil); err != nil || len(got) != 0 || got == nil {
		t.Fatalf("nil → slice rỗng khác nil: got %#v, %v", got, err)
	}
	many := make([]uuid.UUID, domain.TaskLabelsMax+1)
	for i := range many {
		many[i] = uuid.New()
	}
	if _, err := domain.NormalizeLabelIDs(many[:domain.TaskLabelsMax]); err != nil {
		t.Fatalf("đúng %d nhãn phải hợp lệ: %v", domain.TaskLabelsMax, err)
	}
	if _, err := domain.NormalizeLabelIDs(many); !errors.Is(err, domain.ErrTooManyLabels) {
		t.Fatalf("quá %d nhãn phải lỗi, got %v", domain.TaskLabelsMax, err)
	}
	if _, err := domain.NormalizeLabelIDs([]uuid.UUID{uuid.Nil}); !errors.Is(err, domain.ErrLabelNotFound) {
		t.Fatalf("uuid rỗng phải lỗi, got %v", err)
	}
}

func TestCanBeAssigned_CF06(t *testing.T) {
	tests := []struct {
		role   domain.Role
		status domain.UserStatus
		want   bool
	}{
		{domain.RoleMember, domain.UserStatusActive, true},
		{domain.RoleManager, domain.UserStatusActive, true},
		{domain.RoleMember, domain.UserStatusPending, true},
		{domain.RoleMember, domain.UserStatusDisabled, false},
		{domain.RoleCEO, domain.UserStatusActive, false},
		{domain.RoleAdmin, domain.UserStatusActive, false},
	}
	for _, tt := range tests {
		if got := domain.CanBeAssigned(tt.role, tt.status); got != tt.want {
			t.Fatalf("CanBeAssigned(%s, %s) = %v; want %v", tt.role, tt.status, got, tt.want)
		}
	}
}
