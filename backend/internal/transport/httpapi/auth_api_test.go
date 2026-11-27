package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/auth"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// Test API cho CF-01 (đăng nhập, phiên, quản lý tài khoản): đi qua router thật,
// use case được thay bằng fake để kiểm tra đúng ánh xạ HTTP, cookie và phân quyền.

var testNow = time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)

const testSecret = "Matkhau" + "2026" // không phải secret thật — chỉ dùng trong test

// ---- fakes ----

type fakeAuth struct {
	loginErr   error
	refreshErr error
	meErr      error
	activErr   error
	gotRefresh string
	gotLogout  string
	logoutN    int
}

func sessionFor(role domain.Role) auth.Session {
	return auth.Session{
		User:             domain.User{ID: uuid.New(), Email: "an@worktrack.local", FullName: "Nguyễn Văn An", Role: role, Status: domain.UserStatusActive, CreatedAt: testNow},
		AccessToken:      "access-" + string(role),
		AccessExpiresAt:  testNow.Add(15 * time.Minute),
		RefreshToken:     "refresh-raw",
		RefreshExpiresAt: testNow.Add(7 * 24 * time.Hour),
	}
}

func (f *fakeAuth) Login(_ context.Context, _, _ string) (auth.Session, error) {
	if f.loginErr != nil {
		return auth.Session{}, f.loginErr
	}
	return sessionFor(domain.RoleMember), nil
}

func (f *fakeAuth) Refresh(_ context.Context, raw string) (auth.Session, error) {
	f.gotRefresh = raw
	if f.refreshErr != nil {
		return auth.Session{}, f.refreshErr
	}
	return sessionFor(domain.RoleMember), nil
}

func (f *fakeAuth) Logout(_ context.Context, raw string) error {
	f.gotLogout = raw
	f.logoutN++
	return nil
}

func (f *fakeAuth) Me(_ context.Context, p authz.Principal) (domain.User, error) {
	if f.meErr != nil {
		return domain.User{}, f.meErr
	}
	return domain.User{ID: p.UserID, Email: "x@worktrack.local", FullName: "Trần Thị Bình", Role: p.Role, Status: domain.UserStatusActive}, nil
}

func (f *fakeAuth) InspectActivation(_ context.Context, raw string) (domain.User, error) {
	if raw != "good" {
		return domain.User{}, domain.ErrActivationTokenInvalid
	}
	return domain.User{Email: "moi@worktrack.local", FullName: "Lê Văn Cường"}, nil
}

func (f *fakeAuth) Activate(_ context.Context, _, _ string) error { return f.activErr }

type fakeAccounts struct {
	createErr error
	calls     int
}

func (f *fakeAccounts) CreateUser(_ context.Context, actor authz.Principal, in account.CreateUserInput) (account.CreateUserResult, error) {
	f.calls++
	if f.createErr != nil {
		return account.CreateUserResult{}, f.createErr
	}
	if in.Role == domain.RoleCEO && !authz.Can(actor.Role, authz.AssignCEORole) {
		return account.CreateUserResult{}, domain.ErrForbidden
	}
	return account.CreateUserResult{
		User:      domain.User{ID: uuid.New(), Email: in.Email, FullName: in.FullName, Role: in.Role, Status: domain.UserStatusPending},
		EmailSent: true,
	}, nil
}

func (f *fakeAccounts) ListUsers(_ context.Context, _ authz.Principal, p domain.PageRequest) (domain.Page[domain.User], error) {
	f.calls++
	return domain.Page[domain.User]{Items: []domain.User{{ID: uuid.New(), Role: domain.RoleMember}}, Page: p.Page, PageSize: p.PageSize, Total: 1}, nil
}

func (f *fakeAccounts) ResendActivation(_ context.Context, _ authz.Principal, _ uuid.UUID) error {
	f.calls++
	return nil
}

// fakeVerifier chấp nhận token dạng "valid-<role>".
type fakeVerifier struct{}

func (fakeVerifier) Verify(token string, _ time.Time) (authz.Principal, error) {
	role, ok := strings.CutPrefix(token, "valid-")
	if !ok {
		return authz.Principal{}, errors.New("bad token")
	}
	r, err := domain.ParseRole(role)
	if err != nil {
		return authz.Principal{}, err
	}
	return authz.Principal{UserID: uuid.New(), Role: r}, nil
}

type testAPI struct {
	h    http.Handler
	auth *fakeAuth
	acc  *fakeAccounts
}

func newTestAPI(t *testing.T, mutate ...func(*Deps)) testAPI {
	t.Helper()
	fa, fc := &fakeAuth{}, &fakeAccounts{}
	d := Deps{
		DB: fakePinger{}, Auth: fa, Accounts: fc, Tokens: fakeVerifier{},
		Now: func() time.Time { return testNow },
	}
	for _, m := range mutate {
		m(&d)
	}
	return testAPI{h: NewRouter(d), auth: fa, acc: fc}
}

type reqOpt func(*http.Request)

func withToken(tok string) reqOpt {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+tok) }
}

func withCookie(v string) reqOpt {
	return func(r *http.Request) { r.Header.Add("Cookie", refreshCookieName+"="+v) }
}

func (a testAPI) do(t *testing.T, method, path, body string, opts ...reqOpt) *httptest.ResponseRecorder {
	t.Helper()
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequestWithContext(t.Context(), method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	for _, o := range opts {
		o(req)
	}
	rec := httptest.NewRecorder()
	a.h.ServeHTTP(rec, req)
	return rec
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var b errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &b); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return b.Error.Code
}

func refreshCookie(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() { //nolint:bodyclose // recorder, không có body mạng
		if c.Name == refreshCookieName {
			return c
		}
	}
	return nil
}

// ---- đăng nhập ----

func TestLogin_Valid_ReturnsTokenAndSecureCookie_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodPost, "/api/v1/auth/login", `{"email":"an@worktrack.local","password":"`+testSecret+`"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
	}
	var body sessionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.AccessToken == "" || body.User.Role != domain.RoleMember || body.User.Permissions == nil {
		t.Errorf("body = %+v", body)
	}
	if strings.Contains(rec.Body.String(), "refresh-raw") {
		t.Error("refresh token lộ trong body JSON")
	}
	c := refreshCookie(rec)
	if c == nil {
		t.Fatal("thiếu cookie refresh")
	}
	if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteStrictMode || c.Path != "/api/v1/auth" {
		t.Errorf("cookie sai thuộc tính B4: %+v", c)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Error("thiếu Cache-Control: no-store")
	}
}

func TestLogin_Errors_MapToHTTP_CF01(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		svcErr   error
		wantCode int
		wantErr  string
	}{
		{"json hỏng", `{"email":`, nil, 400, "INVALID_JSON"},
		{"trường lạ", `{"email":"a@b.vn","password":"x","role":"admin"}`, nil, 400, "INVALID_JSON"},
		{"thiếu mật khẩu", `{"email":"a@b.vn"}`, nil, 400, "VALIDATION_FAILED"},
		{"email sai định dạng", `{"email":"abc","password":"x"}`, nil, 400, "VALIDATION_FAILED"},
		{"sai mật khẩu", `{"email":"a@b.vn","password":"x"}`, domain.ErrInvalidCredentials, 401, "AUTH_INVALID_CREDENTIALS"},
		{"bị khóa", `{"email":"a@b.vn","password":"x"}`, domain.ErrAccountLocked, 423, "AUTH_ACCOUNT_LOCKED"},
		{"bị vô hiệu hóa", `{"email":"a@b.vn","password":"x"}`, domain.ErrAccountDisabled, 403, "AUTH_ACCOUNT_DISABLED"},
		{"lỗi hạ tầng", `{"email":"a@b.vn","password":"x"}`, errors.New("db down"), 500, "INTERNAL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestAPI(t)
			a.auth.loginErr = tt.svcErr
			rec := a.do(t, http.MethodPost, "/api/v1/auth/login", tt.body)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantCode, rec.Body)
			}
			if got := errCode(t, rec); got != tt.wantErr {
				t.Errorf("code = %q, want %q", got, tt.wantErr)
			}
			if strings.Contains(rec.Body.String(), "db down") {
				t.Error("lộ chi tiết lỗi nội bộ")
			}
		})
	}
}

func TestLogin_TooManyRequests_Returns429WithRetryAfter_CF01(t *testing.T) {
	a := newTestAPI(t, func(d *Deps) { d.LoginLimit = RateLimit{Burst: 2, Every: time.Minute} })
	a.auth.loginErr = domain.ErrInvalidCredentials
	body := `{"email":"a@b.vn","password":"x"}`
	for i := range 2 {
		if rec := a.do(t, http.MethodPost, "/api/v1/auth/login", body); rec.Code != http.StatusUnauthorized {
			t.Fatalf("lần %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	rec := a.do(t, http.MethodPost, "/api/v1/auth/login", body)
	if rec.Code != http.StatusTooManyRequests || errCode(t, rec) != "RATE_LIMITED" {
		t.Fatalf("status = %d body=%s, want 429 RATE_LIMITED", rec.Code, rec.Body)
	}
	if rec.Header().Get("Retry-After") != "60" {
		t.Errorf("Retry-After = %q, want 60", rec.Header().Get("Retry-After"))
	}
}

// ---- làm mới / đăng xuất ----

func TestRefresh_WithCookie_RotatesCookie_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodPost, "/api/v1/auth/refresh", "", withCookie("old-raw"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if a.auth.gotRefresh != "old-raw" {
		t.Errorf("service nhận %q, want old-raw", a.auth.gotRefresh)
	}
	if c := refreshCookie(rec); c == nil || c.Value != "refresh-raw" {
		t.Errorf("cookie mới = %+v", c)
	}
}

func TestRefresh_Invalid_Returns401AndClearsCookie_CF01(t *testing.T) {
	a := newTestAPI(t)
	a.auth.refreshErr = domain.ErrUnauthenticated
	rec := a.do(t, http.MethodPost, "/api/v1/auth/refresh", "")
	if rec.Code != http.StatusUnauthorized || errCode(t, rec) != "UNAUTHENTICATED" {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body)
	}
	if c := refreshCookie(rec); c == nil || c.MaxAge >= 0 {
		t.Errorf("cookie chưa bị xóa: %+v", c)
	}
}

func TestLogout_RevokesAndClearsCookie_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodPost, "/api/v1/auth/logout", "", withCookie("raw-1"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if a.auth.gotLogout != "raw-1" {
		t.Errorf("logout nhận %q", a.auth.gotLogout)
	}
	if c := refreshCookie(rec); c == nil || c.MaxAge >= 0 {
		t.Errorf("cookie chưa bị xóa: %+v", c)
	}
}

func TestLogout_NoCookie_Returns204WithoutCallingService_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodPost, "/api/v1/auth/logout", "")
	if rec.Code != http.StatusNoContent || a.auth.logoutN != 0 {
		t.Fatalf("status = %d, logoutN = %d", rec.Code, a.auth.logoutN)
	}
}

// ---- /auth/me ----

func TestMe_AuthMatrix_CF01(t *testing.T) {
	tests := []struct {
		name     string
		opts     []reqOpt
		wantCode int
	}{
		{"không token", nil, 401},
		{"sai scheme", []reqOpt{func(r *http.Request) { r.Header.Set("Authorization", "Basic abc") }}, 401},
		{"token hỏng", []reqOpt{withToken("garbage")}, 401},
		{"admin", []reqOpt{withToken("valid-admin")}, 200},
		{"manager", []reqOpt{withToken("valid-manager")}, 200},
		{"member", []reqOpt{withToken("valid-member")}, 200},
		{"ceo", []reqOpt{withToken("valid-ceo")}, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestAPI(t)
			rec := a.do(t, http.MethodGet, "/api/v1/auth/me", "", tt.opts...)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantCode, rec.Body)
			}
		})
	}
}

func TestMe_ReturnsPermissionsOfRole_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodGet, "/api/v1/auth/me", "", withToken("valid-ceo"))
	var u userResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &u); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(permStrings(u.Permissions), ","), string(authz.ViewCompanyDashboard)) {
		t.Errorf("CEO thiếu quyền dashboard: %v", u.Permissions)
	}
}

func TestMe_AccountDisabledAfterLogin_Returns401_CF01(t *testing.T) {
	a := newTestAPI(t)
	a.auth.meErr = domain.ErrUnauthenticated
	rec := a.do(t, http.MethodGet, "/api/v1/auth/me", "", withToken("valid-member"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func permStrings(ps []authz.Permission) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = string(p)
	}
	return out
}

// ---- kích hoạt ----

func TestActivationVerify_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodPost, "/api/v1/auth/activation/verify", `{"token":"good"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "moi@worktrack.local") {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body)
	}
	rec = a.do(t, http.MethodPost, "/api/v1/auth/activation/verify", `{"token":"bad"}`)
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "ACTIVATION_TOKEN_INVALID" {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body)
	}
}

func TestActivate_CF01(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		svcErr   error
		wantCode int
	}{
		{"thành công", `{"token":"good","password":"` + testSecret + `"}`, nil, 204},
		{"thiếu token", `{"password":"x"}`, nil, 400},
		{"mật khẩu yếu", `{"token":"good","password":"abc"}`, domain.ErrPasswordTooWeak, 422},
		{"token hết hạn", `{"token":"old","password":"x"}`, domain.ErrActivationTokenInvalid, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestAPI(t)
			a.auth.activErr = tt.svcErr
			rec := a.do(t, http.MethodPost, "/api/v1/auth/activation", tt.body)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantCode, rec.Body)
			}
		})
	}
}

// ---- quản lý tài khoản: 401/403 cho mọi vai trò ----

func TestUsersEndpoints_RoleMatrix_CF01(t *testing.T) {
	endpoints := []struct {
		method, path, body string
		okCode             int
	}{
		{http.MethodGet, "/api/v1/users?page=1&pageSize=20", "", 200},
		{http.MethodPost, "/api/v1/users", `{"email":"moi@worktrack.local","fullName":"Phạm Thị Dung","role":"member"}`, 201},
		{http.MethodPost, "/api/v1/users/" + uuid.NewString() + "/activation-email", "", 204},
	}
	roles := []struct {
		token string
		allow bool
	}{
		{"", false}, // không đăng nhập → 401
		{"valid-admin", true},
		{"valid-ceo", true}, // B2: CEO có toàn bộ quyền Admin
		{"valid-manager", false},
		{"valid-member", false},
	}
	for _, ep := range endpoints {
		for _, rl := range roles {
			name := ep.method + " " + ep.path + " as " + rl.token
			t.Run(name, func(t *testing.T) {
				a := newTestAPI(t)
				var opts []reqOpt
				if rl.token != "" {
					opts = append(opts, withToken(rl.token))
				}
				rec := a.do(t, ep.method, ep.path, ep.body, opts...)
				want := ep.okCode
				switch {
				case rl.token == "":
					want = http.StatusUnauthorized
				case !rl.allow:
					want = http.StatusForbidden
				}
				if rec.Code != want {
					t.Fatalf("status = %d, want %d; body=%s", rec.Code, want, rec.Body)
				}
				if !rl.allow && a.acc.calls != 0 {
					t.Error("use case vẫn bị gọi khi không có quyền")
				}
			})
		}
	}
}

func TestCreateUser_Errors_CF01(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		body     string
		svcErr   error
		wantCode int
		wantErr  string
	}{
		{"vai trò lạ", "valid-admin", `{"email":"a@b.vn","fullName":"A","role":"boss"}`, nil, 400, "VALIDATION_FAILED"},
		{"thiếu họ tên", "valid-admin", `{"email":"a@b.vn","role":"member"}`, nil, 400, "VALIDATION_FAILED"},
		{"email trùng", "valid-admin", `{"email":"a@b.vn","fullName":"A","role":"member"}`, domain.ErrEmailTaken, 409, "USER_EMAIL_TAKEN"},
		{"admin tạo CEO", "valid-admin", `{"email":"a@b.vn","fullName":"A","role":"ceo"}`, nil, 403, "FORBIDDEN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newTestAPI(t)
			a.acc.createErr = tt.svcErr
			rec := a.do(t, http.MethodPost, "/api/v1/users", tt.body, withToken(tt.token))
			if rec.Code != tt.wantCode || errCode(t, rec) != tt.wantErr {
				t.Fatalf("status = %d body=%s, want %d %s", rec.Code, rec.Body, tt.wantCode, tt.wantErr)
			}
		})
	}
}

func TestCreateUser_CEOCreatesCEO_Returns201_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodPost, "/api/v1/users", `{"email":"a@b.vn","fullName":"A","role":"ceo"}`, withToken("valid-ceo"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body)
	}
}

func TestListUsers_BadPageParam_Returns400_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodGet, "/api/v1/users?page=abc", "", withToken("valid-admin"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestListUsers_ReturnsB9Shape_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodGet, "/api/v1/users?pageSize=500", "", withToken("valid-admin"))
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"items", "page", "pageSize", "total"} {
		if _, ok := body[k]; !ok {
			t.Errorf("thiếu khóa %q", k)
		}
	}
	if string(body["pageSize"]) != "100" {
		t.Errorf("pageSize = %s, want 100 (B9 tối đa)", body["pageSize"])
	}
}

func TestResendActivation_BadUUID_Returns400_CF01(t *testing.T) {
	a := newTestAPI(t)
	rec := a.do(t, http.MethodPost, "/api/v1/users/not-a-uuid/activation-email", "", withToken("valid-admin"))
	if rec.Code != http.StatusBadRequest || errCode(t, rec) != "USER_ID_INVALID" {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body)
	}
}
