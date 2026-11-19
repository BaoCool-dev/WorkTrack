package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestValidatePassword_Policy_CF01_B8(t *testing.T) {
	tests := []struct {
		name string
		pw   string
		want error
	}{
		{"đủ 8 ký tự có chữ và số", "matkhau1", nil},
		{"có ký tự đặc biệt và tiếng Việt", "Mật-khẩu 2026", nil},
		{"7 ký tự", "abc1234", domain.ErrPasswordTooWeak},
		{"chỉ có chữ", "abcdefgh", domain.ErrPasswordTooWeak},
		{"chỉ có số", "12345678", domain.ErrPasswordTooWeak},
		{"rỗng", "", domain.ErrPasswordTooWeak},
		{"8 ký tự nhưng dấu tính theo rune", "ạạạạạạa1", nil},
		{"quá 72 byte (giới hạn bcrypt)", strings.Repeat("a1", 37), domain.ErrPasswordTooWeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.ValidatePassword(tt.pw); !errors.Is(got, tt.want) {
				t.Fatalf("ValidatePassword(%q) = %v, want %v", tt.pw, got, tt.want)
			}
		})
	}
}

func TestParseRole_CF01(t *testing.T) {
	for _, code := range []string{"admin", "manager", "member", "ceo"} {
		r, err := domain.ParseRole(code)
		if err != nil || string(r) != code {
			t.Fatalf("ParseRole(%q) = %q, %v", code, r, err)
		}
	}
	if _, err := domain.ParseRole("root"); !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("ParseRole(root) err = %v, want ErrInvalidRole", err)
	}
}

func TestUserIsLocked_CF01_B8(t *testing.T) {
	now := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	future := now.Add(time.Minute)
	past := now.Add(-time.Second)
	tests := []struct {
		name        string
		lockedUntil *time.Time
		want        bool
	}{
		{"chưa từng khóa", nil, false},
		{"đang khóa", &future, true},
		{"hết hạn khóa", &past, false},
		{"đúng thời điểm hết khóa", &now, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := domain.User{LockedUntil: tt.lockedUntil}
			if got := u.IsLocked(now); got != tt.want {
				t.Fatalf("IsLocked = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserCanLogin_Status_CF01(t *testing.T) {
	tests := []struct {
		status domain.UserStatus
		want   error
	}{
		{domain.UserStatusActive, nil},
		{domain.UserStatusPending, domain.ErrAccountNotActive},
		{domain.UserStatusDisabled, domain.ErrAccountDisabled},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			u := domain.User{Status: tt.status}
			if got := u.CheckCanLogin(); !errors.Is(got, tt.want) {
				t.Fatalf("CheckCanLogin = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeEmail_CF01(t *testing.T) {
	if got := domain.NormalizeEmail("  An.Nguyen@WorkTrack.VN "); got != "an.nguyen@worktrack.vn" {
		t.Fatalf("NormalizeEmail = %q", got)
	}
}
