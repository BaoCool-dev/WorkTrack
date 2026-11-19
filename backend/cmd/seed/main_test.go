package main

import (
	"testing"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestNewPassword_SatisfiesPolicyAndIsRandom_CF01_B11(t *testing.T) {
	seen := map[string]bool{}
	for range 50 {
		pw, err := newPassword()
		if err != nil {
			t.Fatal(err)
		}
		if err := domain.ValidatePassword(pw); err != nil {
			t.Fatalf("mật khẩu sinh ra không đạt B8: %v", err)
		}
		if seen[pw] {
			t.Fatal("mật khẩu bị lặp")
		}
		seen[pw] = true
	}
}

func TestSeedUsers_OnePerRole_B11(t *testing.T) {
	roles := map[domain.Role]int{}
	for _, u := range seedUsers {
		roles[u.Role]++
	}
	for _, r := range []domain.Role{domain.RoleAdmin, domain.RoleCEO, domain.RoleManager, domain.RoleMember} {
		if roles[r] != 1 {
			t.Errorf("vai trò %s có %d tài khoản, want 1", r, roles[r])
		}
	}
}
