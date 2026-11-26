// Command seed tạo dữ liệu mẫu cho môi trường dev (B11):
// mỗi vai trò một tài khoản đã kích hoạt, mật khẩu ngẫu nhiên in ra màn hình;
// cây phòng ban, chức danh, danh mục kỹ năng và hồ sơ kỹ năng mẫu (CF-02);
// 2 dự án mẫu do Manager phụ trách, Member là thành viên (CF-03).
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/task"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/postgres/sqlcgen"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/infrastructure/realtime"
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

// seedProjects: dự án mẫu (B11), tạo qua project.Service để có sẵn 5 cột + luật chuyển mặc định.
var seedProjects = []project.Input{
	{Name: "Cổng thông tin nội bộ", Description: "Xây dựng cổng thông tin nội bộ cho nhân viên.", Status: string(domain.ProjectInProgress)},
	{Name: "Ứng dụng đặt lịch họp", Description: "Ứng dụng web đặt phòng và lịch họp.", Status: string(domain.ProjectNew)},
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
	ids := map[domain.Role]uuid.UUID{}
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
		ids[u.Role] = id
		fmt.Printf("  %-8s %-26s %s\n", u.Role, u.Email, pw)
	}
	fmt.Printf("Cơ cấu tổ chức: %d phòng ban, %d chức danh, %d kỹ năng.\n",
		len(org.departments), len(org.jobTitles), len(org.skills))
	n, err := seedProjectData(ctx, pool, q, ids[domain.RoleManager], ids[domain.RoleMember])
	if err != nil {
		return err
	}
	fmt.Printf("Dự án mẫu: %d dự án (mới tạo %d).\n", len(seedProjects), n)
	nt, err := seedTaskData(ctx, pool, q, ids[domain.RoleManager], ids[domain.RoleMember])
	if err != nil {
		return err
	}
	fmt.Printf("Công việc mẫu: %d nhãn, %d công việc (mới tạo %d).\n", len(seedLabels), len(seedTasks), nt)
	return nil
}

// seedProjectData tạo dự án mẫu còn thiếu (idempotent theo tên) và thêm Member vào mọi dự án mẫu.
func seedProjectData(ctx context.Context, pool *pgxpool.Pool, q *sqlcgen.Queries, managerID, memberID uuid.UUID) (int, error) {
	svc := project.NewService(postgres.NewProjectRepo(pool), postgres.NewProjectTx(pool))
	mgr := authz.Principal{UserID: managerID, Role: domain.RoleManager}
	created := 0
	for _, in := range seedProjects {
		id, err := q.GetSeedProjectIDByName(ctx, in.Name)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			p, err := svc.Create(ctx, mgr, in)
			if err != nil {
				return created, fmt.Errorf("project %s: %w", in.Name, err)
			}
			id = p.ID
			created++
		case err != nil:
			return created, fmt.Errorf("project %s: %w", in.Name, err)
		}
		if err := svc.AddMember(ctx, mgr, id, memberID); err != nil {
			return created, fmt.Errorf("project member %s: %w", in.Name, err)
		}
	}
	return created, nil
}

// seedLabels: danh mục nhãn mẫu (CF-06).
var seedLabels = []string{"Backend", "Frontend", "Thiết kế", "Kiểm thử"}

type seedTask struct {
	Project    string
	Title      string
	StoryPoint int
	Labels     []string
	ToMember   bool // giao sẵn cho Member; false = để ở Backlog cho Member tự nhận
	DueInDays  int  // 0 = không có hạn chót
}

// seedTasks: công việc cha mẫu, tạo qua task.Service nên nằm ở cột todo đầu tiên của dự án.
var seedTasks = []seedTask{
	{"Cổng thông tin nội bộ", "Thiết kế trang chủ cổng thông tin", 3, []string{"Thiết kế", "Frontend"}, true, 7},
	{"Cổng thông tin nội bộ", "API đăng tin thông báo nội bộ", 5, []string{"Backend"}, true, 14},
	{"Cổng thông tin nội bộ", "Tìm kiếm bài viết theo từ khóa", 8, []string{"Backend", "Frontend"}, false, 21},
	{"Cổng thông tin nội bộ", "Viết kịch bản kiểm thử đăng nhập", 2, []string{"Kiểm thử"}, false, 0},
	{"Ứng dụng đặt lịch họp", "Mô hình dữ liệu phòng họp", 3, []string{"Backend"}, false, 10},
	{"Ứng dụng đặt lịch họp", "Màn hình lịch tuần", 5, []string{"Frontend", "Thiết kế"}, false, 0},
}

// seedTaskData tạo nhãn mẫu và công việc mẫu còn thiếu (idempotent theo tiêu đề trong dự án).
func seedTaskData(ctx context.Context, pool *pgxpool.Pool, q *sqlcgen.Queries, managerID, memberID uuid.UUID) (int, error) {
	labels := map[string]uuid.UUID{}
	for _, n := range seedLabels {
		id, err := q.UpsertSeedLabel(ctx, n)
		if err != nil {
			return 0, fmt.Errorf("label %s: %w", n, err)
		}
		labels[n] = id
	}
	projects := project.NewService(postgres.NewProjectRepo(pool), postgres.NewProjectTx(pool))
	settings := setting.NewService(postgres.NewSettingRepo(pool))
	svc := task.NewService(postgres.NewTaskRepo(pool), postgres.NewTaskTx(pool), projects, settings, realtime.NewHub())
	mgr := authz.Principal{UserID: managerID, Role: domain.RoleManager}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	created := 0
	for _, st := range seedTasks {
		pid, err := q.GetSeedProjectIDByName(ctx, st.Project)
		if err != nil {
			return created, fmt.Errorf("task %s: project %s: %w", st.Title, st.Project, err)
		}
		_, err = q.GetSeedTaskIDByTitle(ctx, sqlcgen.GetSeedTaskIDByTitleParams{ProjectID: pid, Title: st.Title})
		if err == nil {
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return created, fmt.Errorf("task %s: %w", st.Title, err)
		}
		in := task.Input{Title: st.Title, StoryPoint: &st.StoryPoint}
		for _, l := range st.Labels {
			in.LabelIDs = append(in.LabelIDs, labels[l])
		}
		if st.ToMember {
			in.AssigneeID = &memberID
		}
		if st.DueInDays > 0 {
			d := today.AddDate(0, 0, st.DueInDays)
			in.DueDate = &d
		}
		if _, err := svc.Create(ctx, mgr, pid, in); err != nil {
			return created, fmt.Errorf("task %s: %w", st.Title, err)
		}
		created++
	}
	return created, nil
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
