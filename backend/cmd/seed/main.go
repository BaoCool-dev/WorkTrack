// Command seed tạo dữ liệu mẫu cho môi trường dev (B11):
// mỗi vai trò một tài khoản đã kích hoạt, mật khẩu ngẫu nhiên in ra màn hình.
// Chạy lại sẽ đặt lại mật khẩu mới. Từ chối chạy khi APP_ENV=production.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/security"
)

type seedUser struct {
	Email    string
	FullName string
	Role     domain.Role
}

// Tên người Việt giả, email miền .local không gửi ra ngoài được.
var seedUsers = []seedUser{
	{"admin@worktrack.local", "Nguyễn Thị Hạnh", domain.RoleAdmin},
	{"ceo@worktrack.local", "Trần Quốc Bảo", domain.RoleCEO},
	{"manager@worktrack.local", "Lê Minh Khoa", domain.RoleManager},
	{"member@worktrack.local", "Phạm Thu Trang", domain.RoleMember},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		os.Exit(1)
	}
}

func run() error {
	if os.Getenv("APP_ENV") == "production" {
		return errors.New("không chạy seed khi APP_ENV=production")
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	q := sqlcgen.New(pool)
	hasher := security.NewBcryptHasher(bcrypt.DefaultCost)

	fmt.Println("Tài khoản mẫu (chỉ dùng cho môi trường dev):")
	for _, u := range seedUsers {
		pw, err := newPassword()
		if err != nil {
			return err
		}
		hash, err := hasher.Hash(pw)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		if _, err := q.UpsertSeedUser(ctx, sqlcgen.UpsertSeedUserParams{
			Email:        u.Email,
			FullName:     u.FullName,
			RoleCode:     string(u.Role),
			PasswordHash: pgtype.Text{String: hash, Valid: true},
		}); err != nil {
			return fmt.Errorf("upsert %s: %w", u.Email, err)
		}
		fmt.Printf("  %-8s %-26s %s\n", u.Role, u.Email, pw)
	}
	fmt.Println("Dự án mẫu sẽ được thêm ở nhánh feature/projects.")
	return nil
}

// newPassword sinh mật khẩu ngẫu nhiên 96 bit (16 ký tự base64url) kẹp giữa
// tiền tố chữ và hậu tố số để luôn thỏa B8 (có chữ và số).
func newPassword() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("random: %w", err)
	}
	pw := "Wt" + base64.RawURLEncoding.EncodeToString(b) + "9"
	if err := domain.ValidatePassword(pw); err != nil {
		return "", err
	}
	return pw, nil
}
