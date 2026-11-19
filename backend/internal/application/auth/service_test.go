package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/auth"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/securetoken"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

var t0 = time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)

type env struct {
	svc    *auth.Service
	users  *fakeUsers
	tokens *fakeTokens
	hasher *fakeHasher
	now    *time.Time
}

func activeUser(role domain.Role) domain.User {
	return domain.User{
		ID: uuid.New(), Email: "an.nguyen@worktrack.local", FullName: "Nguyễn Văn An",
		Role: role, Status: domain.UserStatusActive, PasswordHash: fakeHash(testPassword),
	}
}

func newEnv(us ...domain.User) *env {
	e := &env{users: newFakeUsers(us...), tokens: newFakeTokens(), hasher: &fakeHasher{}}
	now := t0
	e.now = &now
	e.svc = auth.NewService(e.users, e.tokens, e.hasher, fakeIssuer{}, func() time.Time { return *e.now })
	return e
}

func TestLogin_Success_CF01(t *testing.T) {
	u := activeUser(domain.RoleManager)
	e := newEnv(u)
	s, err := e.svc.Login(context.Background(), "  AN.NGUYEN@worktrack.local ", "matkhau123")
	if err != nil {
		t.Fatal(err)
	}
	if s.User.ID != u.ID || !strings.HasSuffix(s.AccessToken, "-manager") || s.RefreshToken == "" {
		t.Fatalf("session sai: %+v", s)
	}
	if !s.AccessExpiresAt.Equal(t0.Add(15*time.Minute)) || !s.RefreshExpiresAt.Equal(t0.Add(7*24*time.Hour)) {
		t.Fatalf("thời hạn sai (B4): %v %v", s.AccessExpiresAt, s.RefreshExpiresAt)
	}
	if e.users.successAt == nil || e.tokens.activeCount() != 1 {
		t.Fatal("phải ghi đăng nhập thành công và lưu 1 refresh token")
	}
	for _, rt := range e.tokens.items {
		if string(rt.TokenHash) == s.RefreshToken {
			t.Fatal("CSDL phải lưu hash, không lưu token thô")
		}
	}
}

func TestLogin_UnknownEmail_GenericError_CF01(t *testing.T) {
	e := newEnv()
	_, err := e.svc.Login(context.Background(), "x@y.z", "matkhau123")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v", err)
	}
	if e.hasher.compares != 1 {
		t.Fatal("phải so khớp hash giả để chống dò email theo thời gian")
	}
}

func TestLogin_WrongPassword_FiveTimesLocks_CF01_B8(t *testing.T) {
	u := activeUser(domain.RoleMember)
	e := newEnv(u)
	ctx := context.Background()
	for i := 1; i <= 4; i++ {
		if _, err := e.svc.Login(ctx, u.Email, "sai-mat-khau1"); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Fatalf("lần %d: got %v", i, err)
		}
	}
	if _, err := e.svc.Login(ctx, u.Email, "sai-mat-khau1"); !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("lần 5 phải khóa: %v", err)
	}
	// Đang khóa: mật khẩu đúng vẫn bị chặn, không so khớp mật khẩu.
	before := e.hasher.compares
	if _, err := e.svc.Login(ctx, u.Email, "matkhau123"); !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("đang khóa: %v", err)
	}
	if e.hasher.compares != before {
		t.Fatal("không so khớp khi đang khóa")
	}
	// Sau 15 phút: đăng nhập lại được.
	*e.now = t0.Add(15 * time.Minute)
	if _, err := e.svc.Login(ctx, u.Email, "matkhau123"); err != nil {
		t.Fatalf("hết khóa: %v", err)
	}
}

func TestLogin_SuccessResetsFailureCount_CF01_B8(t *testing.T) {
	u := activeUser(domain.RoleMember)
	e := newEnv(u)
	ctx := context.Background()
	for range 4 {
		_, _ = e.svc.Login(ctx, u.Email, "sai-mat-khau1")
	}
	if _, err := e.svc.Login(ctx, u.Email, "matkhau123"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.Login(ctx, u.Email, "sai-mat-khau1"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("đếm phải reset sau thành công: %v", err)
	}
}

func TestLogin_PendingOrDisabled_CF01(t *testing.T) {
	pending := activeUser(domain.RoleMember)
	pending.Status, pending.PasswordHash = domain.UserStatusPending, ""
	disabled := activeUser(domain.RoleMember)
	disabled.Email, disabled.Status = "khoa@worktrack.local", domain.UserStatusDisabled
	e := newEnv(pending, disabled)
	if _, err := e.svc.Login(context.Background(), pending.Email, "matkhau123"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("pending: %v", err)
	}
	if _, err := e.svc.Login(context.Background(), disabled.Email, "matkhau123"); !errors.Is(err, domain.ErrAccountDisabled) {
		t.Fatalf("disabled: %v", err)
	}
	if _, err := e.svc.Login(context.Background(), disabled.Email, "sai-mat-khau1"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("disabled + sai mật khẩu không lộ trạng thái: %v", err)
	}
}

func TestLogin_StoreError_Wrapped(t *testing.T) {
	e := newEnv()
	e.users.err = errBoom
	if _, err := e.svc.Login(context.Background(), "a@b.c", "x"); !errors.Is(err, errBoom) {
		t.Fatalf("got %v", err)
	}
}

func TestLogin_IssuerError(t *testing.T) {
	u := activeUser(domain.RoleAdmin)
	e := newEnv(u)
	e.svc = auth.NewService(e.users, e.tokens, e.hasher, fakeIssuer{err: errBoom}, func() time.Time { return t0 })
	if _, err := e.svc.Login(context.Background(), u.Email, "matkhau123"); !errors.Is(err, errBoom) {
		t.Fatalf("got %v", err)
	}
}

func login(t *testing.T, e *env, u domain.User) auth.Session {
	t.Helper()
	s, err := e.svc.Login(context.Background(), u.Email, "matkhau123")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRefresh_RotatesToken_CF01_B4(t *testing.T) {
	u := activeUser(domain.RoleCEO)
	e := newEnv(u)
	s1 := login(t, e, u)
	*e.now = t0.Add(time.Hour)
	s2, err := e.svc.Refresh(context.Background(), s1.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if s2.RefreshToken == s1.RefreshToken || s2.AccessToken == "" {
		t.Fatal("phải phát refresh token mới")
	}
	if e.tokens.activeCount() != 1 {
		t.Fatalf("token cũ phải bị thu hồi, còn %d token hoạt động", e.tokens.activeCount())
	}
	if !s2.RefreshExpiresAt.Equal(t0.Add(time.Hour + 7*24*time.Hour)) {
		t.Fatal("hạn refresh mới tính từ lúc làm mới")
	}
}

func TestRefresh_ReuseRevokesFamily_CF01_B4(t *testing.T) {
	u := activeUser(domain.RoleMember)
	e := newEnv(u)
	s1 := login(t, e, u)
	s2, err := e.svc.Refresh(context.Background(), s1.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	// Kẻ gian dùng lại token cũ → 401 và token mới của người dùng cũng bị thu hồi.
	if _, err := e.svc.Refresh(context.Background(), s1.RefreshToken); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("reuse: %v", err)
	}
	if _, err := e.svc.Refresh(context.Background(), s2.RefreshToken); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("cả họ phải bị thu hồi: %v", err)
	}
}

func TestRefresh_ConcurrentUseRevokesFamily_CF01_B4(t *testing.T) {
	u := activeUser(domain.RoleMember)
	e := newEnv(u)
	s1 := login(t, e, u)
	e.tokens.raceOnce = true
	if _, err := e.svc.Refresh(context.Background(), s1.RefreshToken); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("got %v", err)
	}
	if e.tokens.familyCalls != 1 {
		t.Fatal("phải thu hồi cả họ")
	}
}

func TestRefresh_Invalid_CF01_B4(t *testing.T) {
	u := activeUser(domain.RoleMember)
	e := newEnv(u)
	s1 := login(t, e, u)
	ctx := context.Background()
	for name, tok := range map[string]string{"rỗng": "", "lạ": "khong-ton-tai"} {
		if _, err := e.svc.Refresh(ctx, tok); !errors.Is(err, domain.ErrUnauthenticated) {
			t.Errorf("%s: %v", name, err)
		}
	}
	*e.now = t0.Add(7 * 24 * time.Hour)
	if _, err := e.svc.Refresh(ctx, s1.RefreshToken); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("hết hạn: %v", err)
	}
}

func TestRefresh_UserDisabledOrDeleted_CF01(t *testing.T) {
	u := activeUser(domain.RoleMember)
	e := newEnv(u)
	s1 := login(t, e, u)
	e.users.byID[u.ID].Status = domain.UserStatusDisabled
	if _, err := e.svc.Refresh(context.Background(), s1.RefreshToken); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("disabled: %v", err)
	}
	e2 := newEnv(u)
	s2 := login(t, e2, u)
	delete(e2.users.byID, u.ID)
	if _, err := e2.svc.Refresh(context.Background(), s2.RefreshToken); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("deleted: %v", err)
	}
}

func TestRefresh_StoreError(t *testing.T) {
	e := newEnv()
	e.tokens.err = errBoom
	if _, err := e.svc.Refresh(context.Background(), "abc"); !errors.Is(err, errBoom) {
		t.Fatalf("got %v", err)
	}
	if err := e.svc.Logout(context.Background(), "abc"); !errors.Is(err, errBoom) {
		t.Fatalf("logout: %v", err)
	}
}

func TestLogout_RevokesFamily_Idempotent_CF01(t *testing.T) {
	u := activeUser(domain.RoleAdmin)
	e := newEnv(u)
	s1 := login(t, e, u)
	ctx := context.Background()
	if err := e.svc.Logout(ctx, s1.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if e.tokens.activeCount() != 0 {
		t.Fatal("phải thu hồi token")
	}
	if _, err := e.svc.Refresh(ctx, s1.RefreshToken); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatal("sau đăng xuất không làm mới được")
	}
	for _, tok := range []string{"", "khong-ton-tai", s1.RefreshToken} {
		if err := e.svc.Logout(ctx, tok); err != nil {
			t.Fatalf("logout(%q) phải idempotent: %v", tok, err)
		}
	}
}

func TestMe_CF01(t *testing.T) {
	u := activeUser(domain.RoleManager)
	e := newEnv(u)
	got, err := e.svc.Me(context.Background(), authz.Principal{UserID: u.ID, Role: u.Role})
	if err != nil || got.ID != u.ID {
		t.Fatalf("me: %v %v", got, err)
	}
	if _, err := e.svc.Me(context.Background(), authz.Principal{UserID: uuid.New()}); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("không tồn tại: %v", err)
	}
	e.users.byID[u.ID].Status = domain.UserStatusDisabled
	if _, err := e.svc.Me(context.Background(), authz.Principal{UserID: u.ID}); !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("bị vô hiệu: %v", err)
	}
	e.users.err = errBoom
	if _, err := e.svc.Me(context.Background(), authz.Principal{UserID: u.ID}); !errors.Is(err, errBoom) {
		t.Fatalf("store: %v", err)
	}
}

func pendingWithToken(e *env, exp time.Time) (domain.User, string) {
	u := domain.User{ID: uuid.New(), Email: "binh.tran@worktrack.local", FullName: "Trần Thị Bình",
		Role: domain.RoleMember, Status: domain.UserStatusPending, ActivationExpiresAt: &exp}
	raw, hash, _ := securetoken.New()
	e.users.byID[u.ID] = &u
	e.users.tokenHash[u.ID] = hash
	return u, raw
}

func TestActivate_SetsPasswordOnce_CF01_B7(t *testing.T) {
	e := newEnv()
	u, raw := pendingWithToken(e, t0.Add(72*time.Hour))
	ctx := context.Background()
	got, err := e.svc.InspectActivation(ctx, raw)
	if err != nil || got.Email != u.Email {
		t.Fatalf("inspect: %v", err)
	}
	if err := e.svc.Activate(ctx, raw, "matkhau123"); err != nil {
		t.Fatal(err)
	}
	if e.users.byID[u.ID].Status != domain.UserStatusActive || e.users.byID[u.ID].PasswordHash != "hash:matkhau123" {
		t.Fatal("phải kích hoạt và lưu hash mật khẩu")
	}
	if err := e.svc.Activate(ctx, raw, "matkhau456"); !errors.Is(err, domain.ErrActivationTokenInvalid) {
		t.Fatalf("token chỉ dùng một lần: %v", err)
	}
	if _, err := e.svc.Login(ctx, u.Email, "matkhau123"); err != nil {
		t.Fatalf("đăng nhập sau kích hoạt: %v", err)
	}
}

func TestActivate_Expired72h_CF01_B7(t *testing.T) {
	e := newEnv()
	_, raw := pendingWithToken(e, t0.Add(72*time.Hour))
	*e.now = t0.Add(72 * time.Hour)
	if err := e.svc.Activate(context.Background(), raw, "matkhau123"); !errors.Is(err, domain.ErrActivationTokenInvalid) {
		t.Fatalf("got %v", err)
	}
}

func TestActivate_WeakPassword_CF01_B8(t *testing.T) {
	e := newEnv()
	u, raw := pendingWithToken(e, t0.Add(time.Hour))
	if err := e.svc.Activate(context.Background(), raw, "chiconchu"); !errors.Is(err, domain.ErrPasswordTooWeak) {
		t.Fatalf("got %v", err)
	}
	if e.users.byID[u.ID].Status != domain.UserStatusPending {
		t.Fatal("không được kích hoạt khi mật khẩu yếu")
	}
}

func TestActivate_InvalidToken_CF01_B7(t *testing.T) {
	e := newEnv()
	ctx := context.Background()
	for _, tok := range []string{"", "khong-ton-tai"} {
		if err := e.svc.Activate(ctx, tok, "matkhau123"); !errors.Is(err, domain.ErrActivationTokenInvalid) {
			t.Errorf("%q: %v", tok, err)
		}
	}
	u, raw := pendingWithToken(e, t0.Add(time.Hour))
	e.users.byID[u.ID].Status = domain.UserStatusActive
	if _, err := e.svc.InspectActivation(ctx, raw); !errors.Is(err, domain.ErrActivationTokenInvalid) {
		t.Fatalf("đã active: %v", err)
	}
}

func TestActivate_StoreErrors(t *testing.T) {
	e := newEnv()
	_, raw := pendingWithToken(e, t0.Add(time.Hour))
	e.users.activateFn = func() error { return errBoom }
	if err := e.svc.Activate(context.Background(), raw, "matkhau123"); !errors.Is(err, errBoom) {
		t.Fatalf("got %v", err)
	}
	e.users.activateFn = func() error { return domain.ErrActivationTokenInvalid }
	if err := e.svc.Activate(context.Background(), raw, "matkhau123"); !errors.Is(err, domain.ErrActivationTokenInvalid) {
		t.Fatalf("race: %v", err)
	}
	e.users.err = errBoom
	if _, err := e.svc.InspectActivation(context.Background(), raw); !errors.Is(err, errBoom) {
		t.Fatalf("inspect: %v", err)
	}
}
