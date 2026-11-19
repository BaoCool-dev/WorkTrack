package domain

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Role là vai trò hệ thống (CLAUDE.md §1). Quyền theo vai trò quyết định ở application/authz.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleMember  Role = "member"
	RoleCEO     Role = "ceo"
)

// Roles liệt kê mọi vai trò theo thứ tự hiển thị.
var Roles = []Role{RoleAdmin, RoleManager, RoleMember, RoleCEO}

func ParseRole(s string) (Role, error) {
	r := Role(s)
	switch r {
	case RoleAdmin, RoleManager, RoleMember, RoleCEO:
		return r, nil
	default:
		return "", ErrInvalidRole
	}
}

type UserStatus string

const (
	UserStatusPending  UserStatus = "pending"
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type User struct {
	ID                  uuid.UUID
	Email               string
	FullName            string
	Role                Role
	Status              UserStatus
	PasswordHash        string
	FailedLoginCount    int
	LockedUntil         *time.Time
	ActivationExpiresAt *time.Time
	LastLoginAt         *time.Time
	CreatedAt           time.Time
}

// Chính sách đăng nhập (B8).
const (
	MaxFailedLogins    = 5
	LoginLockDuration  = 15 * time.Minute
	PasswordMinRunes   = 8
	PasswordMaxBytes   = 72             // giới hạn của bcrypt
	ActivationTokenTTL = 72 * time.Hour // B7
)

// ValidatePassword: tối thiểu 8 ký tự, có chữ và số (B8); không quá 72 byte vì bcrypt cắt phần dư.
func ValidatePassword(pw string) error {
	if utf8.RuneCountInString(pw) < PasswordMinRunes || len(pw) > PasswordMaxBytes {
		return ErrPasswordTooWeak
	}
	var hasLetter, hasDigit bool
	for _, r := range pw {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return ErrPasswordTooWeak
	}
	return nil
}

// IsLocked: đang trong thời gian khóa đăng nhập.
func (u User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// CheckCanLogin kiểm tra trạng thái tài khoản (gọi sau khi mật khẩu đúng).
func (u User) CheckCanLogin() error {
	switch u.Status {
	case UserStatusActive:
		return nil
	case UserStatusDisabled:
		return ErrAccountDisabled
	default:
		return ErrAccountNotActive
	}
}

func NormalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
