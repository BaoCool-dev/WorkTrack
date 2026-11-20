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
	ErrEmailDeliveryFailed    = errors.New("email delivery failed")
	ErrInvalidFullName        = errors.New("full name is empty or too long")

	// CF-02 — cơ cấu tổ chức.
	ErrInvalidOrgName        = errors.New("org name is empty or too long")
	ErrDepartmentNotFound    = errors.New("department not found")
	ErrDepartmentNameTaken   = errors.New("department name already used under the same parent")
	ErrDepartmentCycle       = errors.New("department parent would create a cycle")
	ErrDepartmentHasChildren = errors.New("department has sub-departments")
	ErrDepartmentInUse       = errors.New("department has members")
	ErrJobTitleNotFound      = errors.New("job title not found")
	ErrJobTitleNameTaken     = errors.New("job title name already used")
	ErrJobTitleInUse         = errors.New("job title is assigned to users")
	ErrSkillNotFound         = errors.New("skill not found")
	ErrSkillNameTaken        = errors.New("skill name already used")
	ErrSkillInUse            = errors.New("skill is used in user profiles")
	ErrInvalidSkillLevel     = errors.New("invalid skill level")
	ErrDuplicateSkill        = errors.New("duplicate skill in profile")
	ErrTooManySkills         = errors.New("too many skills in profile")
	ErrInvalidSearch         = errors.New("search keyword too long")
	ErrInvalidUserStatus     = errors.New("invalid user status")
	ErrCannotChangeOwnRole   = errors.New("cannot change own role")
)
