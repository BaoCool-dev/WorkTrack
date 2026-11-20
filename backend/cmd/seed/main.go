// Command seed tạo dữ liệu mẫu cho môi trường dev (B11):
// mỗi vai trò một tài khoản đã kích hoạt, mật khẩu ngẫu nhiên in ra màn hình;
// cây phòng ban, chức danh, danh mục kỹ năng và hồ sơ kỹ năng mẫu (CF-02).
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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/security"
)

type seedUser struct {
	Email      string
	FullName   string
	Role       domain.Role
	Department string // khóa trong seedDepartments
	JobTitle   string
	Skills     map[string]domain.SkillLevel
}

// Tên người Việt giả, email miền .local không gửi ra ngoài được.
var seedUsers = []seedUser{
	{"admin@worktrack.local", "Nguyễn Thị Hạnh", domain.RoleAdmin, "Hành chính - Nhân sự", "Chuyên viên nhân sự", nil},
	{"ceo@worktrack.local", "Trần Quốc Bảo", domain.RoleCEO, "Ban Giám đốc", "Giám đốc điều hành", nil},
	{"manager@worktrack.local", "Lê Minh Khoa", domain.RoleManager, "Khối Kỹ thuật", "Quản lý dự án",
		map[string]domain.SkillLevel{"Go": domain.SkillSenior, "PostgreSQL": domain.SkillIntermediate, "Scrum": domain.SkillSenior}},
	{"member@worktrack.local", "Phạm Thu Trang", domain.RoleMember, "Nhóm Frontend", "Lập trình viên",
		map[string]domain.SkillLevel{"React": domain.SkillIntermediate, "TypeScript": domain.SkillIntermediate, "Go": domain.SkillBeginner}},
}

// seedDepartments: tên → tên phòng ban cha ("" là gốc). Cha đứng trước con.
var seedDepartments = []struct{ Name, Parent string }{
	{"Ban Giám đốc", ""},
	{"Hành chính - Nhân sự", ""},
	{"Khối Kỹ thuật", ""},
	{"Nhóm Backend", "Khối Kỹ thuật"},
	{"Nhóm Frontend", "Khối Kỹ thuật"},
	{"Nhóm Kiểm thử", "Khối Kỹ thuật"},
}

var seedJobTitles = []string{
	"Giám đốc điều hành", "Chuyên viên nhân sự", "Quản lý dự án", "Lập trình viên", "Kiểm thử viên",
}

var seedSkills = []string{
	"Go", "PostgreSQL", "React", "TypeScript", "Docker", "Kiểm thử tự động", "Scrum", "UI/UX",
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
	org, err := seedOrg(ctx, q)
	if err != nil {
		return err
	}
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
		id, err := q.UpsertSeedUser(ctx, sqlcgen.UpsertSeedUserParams{
			Email:        u.Email,
			FullName:     u.FullName,
			RoleCode:     string(u.Role),
			PasswordHash: pgtype.Text{String: hash, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("upsert %s: %w", u.Email, err)
		}
		if err := org.assign(ctx, q, id, u); err != nil {
			return fmt.Errorf("org %s: %w", u.Email, err)
		}
		fmt.Printf("  %-8s %-26s %s\n", u.Role, u.Email, pw)
	}
	fmt.Printf("Cơ cấu tổ chức: %d phòng ban, %d chức danh, %d kỹ năng.\n",
		len(org.departments), len(org.jobTitles), len(org.skills))
	fmt.Println("Dự án mẫu sẽ được thêm ở nhánh feature/projects.")
	return nil
}

type seededOrg struct {
	departments, jobTitles, skills map[string]uuid.UUID
}

// seedOrg tạo (idempotent) cây phòng ban, chức danh và danh mục kỹ năng.
func seedOrg(ctx context.Context, q *sqlcgen.Queries) (seededOrg, error) {
	o := seededOrg{map[string]uuid.UUID{}, map[string]uuid.UUID{}, map[string]uuid.UUID{}}
	for _, d := range seedDepartments {
		var parent uuid.NullUUID
		if d.Parent != "" {
			parent = uuid.NullUUID{UUID: o.departments[d.Parent], Valid: true}
		}
		id, err := q.UpsertSeedDepartment(ctx, sqlcgen.UpsertSeedDepartmentParams{Name: d.Name, ParentID: parent})
		if err != nil {
			return o, fmt.Errorf("department %s: %w", d.Name, err)
		}
		o.departments[d.Name] = id
	}
	for _, n := range seedJobTitles {
		id, err := q.UpsertSeedJobTitle(ctx, n)
		if err != nil {
			return o, fmt.Errorf("job title %s: %w", n, err)
		}
		o.jobTitles[n] = id
	}
	for _, n := range seedSkills {
		id, err := q.UpsertSeedSkill(ctx, n)
		if err != nil {
			return o, fmt.Errorf("skill %s: %w", n, err)
		}
		o.skills[n] = id
	}
	return o, nil
}

func (o seededOrg) assign(ctx context.Context, q *sqlcgen.Queries, userID uuid.UUID, u seedUser) error {
	if err := q.SetSeedUserOrg(ctx, sqlcgen.SetSeedUserOrgParams{
		ID:           userID,
		DepartmentID: uuid.NullUUID{UUID: o.departments[u.Department], Valid: true},
		JobTitleID:   uuid.NullUUID{UUID: o.jobTitles[u.JobTitle], Valid: true},
	}); err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(u.Skills))
	levels := make([]string, 0, len(u.Skills))
	for name, lv := range u.Skills {
		ids = append(ids, o.skills[name])
		levels = append(levels, string(lv))
	}
	return q.ReplaceUserSkills(ctx, sqlcgen.ReplaceUserSkillsParams{UserID: userID, SkillIds: ids, Levels: levels})
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
