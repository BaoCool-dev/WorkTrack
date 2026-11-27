// Package security cài đặt port băm mật khẩu (bcrypt) và access token (JWT HS256, B4).
package security

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher cài đặt auth.PasswordHasher.
type BcryptHasher struct{ cost int }

// NewBcryptHasher nhận cost; giá trị ngoài [MinCost, MaxCost] dùng DefaultCost.
func NewBcryptHasher(cost int) BcryptHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return BcryptHasher{cost: cost}
}

func (h BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(b), nil
}

func (h BcryptHasher) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
