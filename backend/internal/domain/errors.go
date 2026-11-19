package domain

import "errors"

// Lỗi nghiệp vụ dùng chung. Ánh xạ sang HTTP ở transport/httpapi/errors.go.
var (
	ErrForbidden       = errors.New("forbidden")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrInvalidRole     = errors.New("invalid role")

	// CF-01 — tài khoản và đăng nhập.
	ErrUserNotFound           = errors.New("user not found")
	ErrEmailTaken             = errors.New("email already taken")
	ErrPasswordTooWeak        = errors.New("password does not meet policy")
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrAccountLocked          = errors.New("account temporarily locked")
	ErrAccountNotActive       = errors.New("account not activated")
	ErrAccountDisabled        = errors.New("account disabled")
	ErrActivationTokenInvalid = errors.New("activation token invalid or expired")
	ErrUserAlreadyActive      = errors.New("user already active")
	ErrRefreshTokenNotFound   = errors.New("refresh token not found")
)
