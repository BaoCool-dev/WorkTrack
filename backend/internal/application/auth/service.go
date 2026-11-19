// Package auth là use case xác thực CF-01: đăng nhập (B8), làm mới xoay vòng (B4),
// đăng xuất, kích hoạt tài khoản (B7) và /auth/me.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/securetoken"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Session là kết quả đăng nhập/làm mới. RefreshToken là chuỗi thô, chỉ gửi qua cookie.
type Session struct {
	User             domain.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type Service struct {
	users     UserStore
	tokens    RefreshTokenStore
	hasher    PasswordHasher
	issuer    AccessTokenIssuer
	now       func() time.Time
	dummyHash string
}

func NewService(users UserStore, tokens RefreshTokenStore, hasher PasswordHasher, issuer AccessTokenIssuer, now func() time.Time) *Service {
	// Hash giả để so khớp khi email không tồn tại → thời gian phản hồi tương đương, chống dò email.
	dummy, _ := hasher.Hash("worktrack-dummy-password-1")
	return &Service{users: users, tokens: tokens, hasher: hasher, issuer: issuer, now: now, dummyHash: dummy}
}

// Login kiểm tra email/mật khẩu; sai 5 lần liên tiếp → khóa 15 phút (B8).
func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	now := s.now().UTC()
	u, err := s.users.GetByEmail(ctx, domain.NormalizeEmail(email))
	if errors.Is(err, domain.ErrUserNotFound) {
		s.hasher.Compare(s.dummyHash, password)
		return Session{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, fmt.Errorf("auth.Login: get user: %w", err)
	}
	if u.IsLocked(now) {
		s.hasher.Compare(s.dummyHash, password) // giữ thời gian phản hồi ngang các nhánh khác
		return Session{}, domain.ErrAccountLocked
	}
	if u.PasswordHash == "" {
		// Tài khoản chưa đặt mật khẩu: trả lỗi chung để không lộ trạng thái tài khoản.
		s.hasher.Compare(s.dummyHash, password)
		return Session{}, domain.ErrInvalidCredentials
	}
	if !s.hasher.Compare(u.PasswordHash, password) {
		lockedUntil, err := s.users.RegisterLoginFailure(ctx, u.ID, now, domain.MaxFailedLogins, domain.LoginLockDuration)
		switch {
		case errors.Is(err, domain.ErrAccountLocked): // request song song vừa khóa
			return Session{}, err
		case errors.Is(err, domain.ErrUserNotFound): // bị xóa giữa chừng: vẫn lỗi chung, không lộ trạng thái
			return Session{}, domain.ErrInvalidCredentials
		case err != nil:
			return Session{}, fmt.Errorf("auth.Login: register failure: %w", err)
		}
		if lockedUntil != nil && now.Before(*lockedUntil) {
			return Session{}, domain.ErrAccountLocked
		}
		return Session{}, domain.ErrInvalidCredentials
	}
	if err := u.CheckCanLogin(); err != nil {
		return Session{}, err
	}
	if err := s.users.RecordLoginSuccess(ctx, u.ID, now); err != nil {
		if errors.Is(err, domain.ErrAccountLocked) {
			return Session{}, err
		}
		return Session{}, fmt.Errorf("auth.Login: record success: %w", err)
	}
	u.FailedLoginCount, u.LockedUntil, u.LastLoginAt = 0, nil, &now
	return s.newSession(ctx, u, uuid.New(), now)
}

// Refresh xoay vòng refresh token (B4). Token đã dùng bị dùng lại → thu hồi cả họ.
func (s *Service) Refresh(ctx context.Context, raw string) (Session, error) {
	if raw == "" {
		return Session{}, domain.ErrUnauthenticated
	}
	now := s.now().UTC()
	t, err := s.tokens.GetByHash(ctx, securetoken.Hash(raw))
	if errors.Is(err, domain.ErrRefreshTokenNotFound) {
		return Session{}, domain.ErrUnauthenticated
	}
	if err != nil {
		return Session{}, fmt.Errorf("auth.Refresh: get token: %w", err)
	}
	if t.IsRevoked() {
		return Session{}, s.revokeFamily(ctx, t.FamilyID, now)
	}
	if t.IsExpired(now) {
		return Session{}, domain.ErrUnauthenticated
	}
	ok, err := s.tokens.RevokeIfActive(ctx, t.ID, now)
	if err != nil {
		return Session{}, fmt.Errorf("auth.Refresh: revoke: %w", err)
	}
	if !ok { // một request khác vừa dùng token này
		return Session{}, s.revokeFamily(ctx, t.FamilyID, now)
	}
	u, err := s.users.GetByID(ctx, t.UserID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return Session{}, s.revokeFamily(ctx, t.FamilyID, now)
	}
	if err != nil {
		return Session{}, fmt.Errorf("auth.Refresh: get user: %w", err)
	}
	if u.CheckCanLogin() != nil {
		return Session{}, s.revokeFamily(ctx, t.FamilyID, now)
	}
	return s.newSession(ctx, u, t.FamilyID, now)
}

// Logout thu hồi cả họ token của phiên. Không có token / token lạ → vẫn thành công (idempotent).
func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	t, err := s.tokens.GetByHash(ctx, securetoken.Hash(raw))
	if errors.Is(err, domain.ErrRefreshTokenNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("auth.Logout: get token: %w", err)
	}
	if err := s.tokens.RevokeFamily(ctx, t.FamilyID, s.now().UTC()); err != nil {
		return fmt.Errorf("auth.Logout: revoke family: %w", err)
	}
	return nil
}

// Me trả hồ sơ người đang đăng nhập; tài khoản bị khóa/xóa sau khi cấp token → ErrUnauthenticated.
func (s *Service) Me(ctx context.Context, p authz.Principal) (domain.User, error) {
	u, err := s.users.GetByID(ctx, p.UserID)
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, domain.ErrUnauthenticated
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("auth.Me: %w", err)
	}
	if u.CheckCanLogin() != nil {
		return domain.User{}, domain.ErrUnauthenticated
	}
	return u, nil
}

// InspectActivation kiểm tra token kích hoạt còn hiệu lực (B7) để trang đặt mật khẩu hiện tên/email.
func (s *Service) InspectActivation(ctx context.Context, raw string) (domain.User, error) {
	if raw == "" {
		return domain.User{}, domain.ErrActivationTokenInvalid
	}
	u, err := s.users.GetByActivationTokenHash(ctx, securetoken.Hash(raw))
	if errors.Is(err, domain.ErrUserNotFound) {
		return domain.User{}, domain.ErrActivationTokenInvalid
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("auth.InspectActivation: %w", err)
	}
	now := s.now().UTC()
	if u.Status != domain.UserStatusPending || u.ActivationExpiresAt == nil || !now.Before(*u.ActivationExpiresAt) {
		return domain.User{}, domain.ErrActivationTokenInvalid
	}
	return u, nil
}

// Activate đặt mật khẩu lần đầu (B8) và kích hoạt tài khoản. Token chỉ dùng được một lần.
func (s *Service) Activate(ctx context.Context, raw, password string) error {
	u, err := s.InspectActivation(ctx, raw)
	if err != nil {
		return err
	}
	if err := domain.ValidatePassword(password); err != nil {
		return err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return fmt.Errorf("auth.Activate: hash: %w", err)
	}
	if err := s.users.Activate(ctx, u.ID, securetoken.Hash(raw), hash, s.now().UTC()); err != nil {
		if errors.Is(err, domain.ErrActivationTokenInvalid) {
			return err
		}
		return fmt.Errorf("auth.Activate: %w", err)
	}
	return nil
}

func (s *Service) newSession(ctx context.Context, u domain.User, familyID uuid.UUID, now time.Time) (Session, error) {
	access, accessExp, err := s.issuer.Issue(authz.Principal{UserID: u.ID, Role: u.Role}, now)
	if err != nil {
		return Session{}, fmt.Errorf("auth: issue access token: %w", err)
	}
	raw, hash, err := securetoken.New()
	if err != nil {
		return Session{}, err
	}
	rt := domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    u.ID,
		FamilyID:  familyID,
		TokenHash: hash,
		ExpiresAt: now.Add(domain.RefreshTokenTTL),
		CreatedAt: now,
	}
	if err := s.tokens.Insert(ctx, rt); err != nil {
		return Session{}, fmt.Errorf("auth: insert refresh token: %w", err)
	}
	return Session{User: u, AccessToken: access, AccessExpiresAt: accessExp, RefreshToken: raw, RefreshExpiresAt: rt.ExpiresAt}, nil
}

func (s *Service) revokeFamily(ctx context.Context, familyID uuid.UUID, now time.Time) error {
	if err := s.tokens.RevokeFamily(ctx, familyID, now); err != nil {
		return fmt.Errorf("auth: revoke family: %w", err)
	}
	return domain.ErrUnauthenticated
}
