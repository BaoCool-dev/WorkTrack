package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

const (
	issuer          = "worktrack"
	MinSecretLength = 32
)

// ErrInvalidToken: token sai chữ ký, hết hạn, sai thuật toán hoặc thiếu claim.
var ErrInvalidToken = errors.New("invalid access token")

type claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// JWT phát và kiểm tra access token HS256 (B4: 15 phút).
type JWT struct {
	secret []byte
	ttl    time.Duration
}

func NewJWT(secret []byte, ttl time.Duration) (*JWT, error) {
	if len(secret) < MinSecretLength {
		return nil, fmt.Errorf("JWT secret must be at least %d bytes", MinSecretLength)
	}
	return &JWT{secret: secret, ttl: ttl}, nil
}

// Issue cài đặt auth.AccessTokenIssuer.
func (j *JWT) Issue(p authz.Principal, now time.Time) (string, time.Time, error) {
	exp := now.Add(j.ttl)
	c := claims{
		Role: string(p.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   p.UserID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        uuid.NewString(),
		},
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(j.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}
	return s, exp, nil
}

// Verify trả Principal của token hợp lệ; mọi lỗi đều quy về ErrInvalidToken.
func (j *JWT) Verify(token string, now time.Time) (authz.Principal, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c,
		func(*jwt.Token) (any, error) { return j.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	if err != nil {
		return authz.Principal{}, ErrInvalidToken
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil {
		return authz.Principal{}, ErrInvalidToken
	}
	role, err := domain.ParseRole(c.Role)
	if err != nil {
		return authz.Principal{}, ErrInvalidToken
	}
	return authz.Principal{UserID: id, Role: role}, nil
}
