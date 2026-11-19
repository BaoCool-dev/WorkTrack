// Package securetoken sinh token ngẫu nhiên (crypto/rand) và băm SHA-256 để lưu CSDL.
// Dùng cho token kích hoạt (B7) và refresh token (B4): CSDL chỉ giữ hash.
package securetoken

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// byteLen = 32 byte (256 bit) entropy.
const byteLen = 32

// New trả token dạng base64url không padding (an toàn cho URL và cookie) và hash của nó.
func New() (token string, hash []byte, err error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("securetoken: read random: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, Hash(token), nil
}

// Hash là SHA-256 của chuỗi token.
func Hash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
