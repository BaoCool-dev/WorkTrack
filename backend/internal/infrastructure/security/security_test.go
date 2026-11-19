package security_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/security"
)

var testKey = []byte(strings.Repeat("k", security.MinSecretLength))

func TestBcryptHasher_HashCompare_CF01_B8(t *testing.T) {
	h := security.NewBcryptHasher(bcrypt.MinCost)
	pw := "abc12345" + "xyz"
	hash, err := h.Hash(pw)
	if err != nil || hash == pw {
		t.Fatalf("hash: %q %v", hash, err)
	}
	if !h.Compare(hash, pw) || h.Compare(hash, pw+"!") || h.Compare("rac", pw) {
		t.Fatal("compare sai")
	}
	if _, err := h.Hash(strings.Repeat("a", 73)); err == nil {
		t.Fatal("quá 72 byte phải lỗi")
	}
	if security.NewBcryptHasher(99) != security.NewBcryptHasher(bcrypt.DefaultCost) {
		t.Fatal("cost ngoài khoảng phải về mặc định")
	}
}

func TestNewJWT_ShortSecret_CF01_B4(t *testing.T) {
	if _, err := security.NewJWT([]byte("ngan"), time.Minute); err == nil {
		t.Fatal("secret ngắn phải lỗi")
	}
}

func TestJWT_IssueVerify_CF01_B4(t *testing.T) {
	j, err := security.NewJWT(testKey, domain.AccessTokenTTL)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Truncate(time.Second)
	p := authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	tok, exp, err := j.Issue(p, now)
	if err != nil || !exp.Equal(now.Add(15*time.Minute)) {
		t.Fatalf("issue: %v %v", exp, err)
	}
	got, err := j.Verify(tok, now.Add(14*time.Minute))
	if err != nil || got != p {
		t.Fatalf("verify: %+v %v", got, err)
	}
	if _, err := j.Verify(tok, now.Add(16*time.Minute)); !errors.Is(err, security.ErrInvalidToken) {
		t.Fatalf("hết hạn: %v", err)
	}
	other, _ := security.NewJWT([]byte(strings.Repeat("x", 32)), time.Minute)
	if _, err := other.Verify(tok, now); !errors.Is(err, security.ErrInvalidToken) {
		t.Fatalf("sai khóa: %v", err)
	}
	if _, err := j.Verify(tok[:len(tok)-2]+"xx", now); !errors.Is(err, security.ErrInvalidToken) {
		t.Fatalf("sửa chữ ký: %v", err)
	}
}

func TestJWT_Verify_RejectsBadTokens_CF01_B4(t *testing.T) {
	j, _ := security.NewJWT(testKey, time.Minute)
	now := time.Now()
	sign := func(m jwt.SigningMethod, key any, c jwt.MapClaims) string {
		s, err := jwt.NewWithClaims(m, c).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	exp := now.Add(time.Minute).Unix()
	good := jwt.MapClaims{"iss": "worktrack", "sub": uuid.NewString(), "role": "admin", "exp": exp}
	if _, err := j.Verify(sign(jwt.SigningMethodHS256, testKey, good), now); err != nil {
		t.Fatalf("claims hợp lệ: %v", err)
	}
	cases := map[string]string{
		"alg none":   sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, good),
		"HS512":      sign(jwt.SigningMethodHS512, testKey, good),
		"sai issuer": sign(jwt.SigningMethodHS256, testKey, jwt.MapClaims{"iss": "x", "sub": good["sub"], "role": "admin", "exp": exp}),
		"thiếu exp":  sign(jwt.SigningMethodHS256, testKey, jwt.MapClaims{"iss": "worktrack", "sub": good["sub"], "role": "admin"}),
		"sub lạ":     sign(jwt.SigningMethodHS256, testKey, jwt.MapClaims{"iss": "worktrack", "sub": "abc", "role": "admin", "exp": exp}),
		"role lạ":    sign(jwt.SigningMethodHS256, testKey, jwt.MapClaims{"iss": "worktrack", "sub": good["sub"], "role": "root", "exp": exp}),
		"rác":        "abc.def.ghi",
	}
	for name, tok := range cases {
		if _, err := j.Verify(tok, now); !errors.Is(err, security.ErrInvalidToken) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
