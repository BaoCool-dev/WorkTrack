package main

import (
	"testing"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestNewPassword_SatisfiesPolicyAndIsRandom_CF01_B11(t *testing.T) {
	seen := map[string]bool{}
	for range 50 {
		pw, err := newPassword()
		if err != nil {
			t.Fatal(err)
		}
		if err := domain.ValidatePassword(pw); err != nil {
			t.Fatalf("mật khẩu sinh ra không đạt B8: %v", err)
		}
		if seen[pw] {
			t.Fatal("mật khẩu bị lặp")
		}
		seen[pw] = true
	}
}

func TestSeedUsers_OnePerRole_B11(t *testing.T) {
	roles := map[domain.Role]int{}
	for _, u := range seedUsers {
		roles[u.Role]++
	}
	for _, r := range []domain.Role{domain.RoleAdmin, domain.RoleCEO, domain.RoleManager, domain.RoleMember} {
		if roles[r] != 1 {
			t.Errorf("vai trò %s có %d tài khoản, want 1", r, roles[r])
		}
	}
}

func TestSeedProjects_TwoValidProjects_B11(t *testing.T) {
	if len(seedProjects) != 2 {
		t.Fatalf("B11 cần 2 dự án mẫu, có %d", len(seedProjects))
	}
	for _, p := range seedProjects {
		if _, err := domain.NormalizeProjectName(p.Name); err != nil {
			t.Errorf("tên %q: %v", p.Name, err)
		}
		if _, err := domain.ParseProjectStatus(p.Status); err != nil {
			t.Errorf("trạng thái %q: %v", p.Status, err)
		}
	}
}

func TestSeedTasks_ValidAndKnownRefs_CF06(t *testing.T) {
	projects := map[string]bool{}
	for _, p := range seedProjects {
		projects[p.Name] = true
	}
	labels := map[string]bool{}
	for _, l := range seedLabels {
		labels[l] = true
	}
	titles := map[string]bool{}
	backlog := 0
	for _, st := range seedTasks {
		if !projects[st.Project] {
			t.Errorf("%s: dự án %q không có trong seedProjects", st.Title, st.Project)
		}
		if _, err := domain.NormalizeTaskTitle(st.Title); err != nil {
			t.Errorf("%s: tiêu đề sai: %v", st.Title, err)
		}
		if err := domain.ValidateStoryPoint(st.StoryPoint); err != nil {
			t.Errorf("%s: Story Point %d sai: %v", st.Title, st.StoryPoint, err)
		}
		for _, l := range st.Labels {
			if !labels[l] {
				t.Errorf("%s: nhãn %q không có trong seedLabels", st.Title, l)
			}
		}
		key := st.Project + "/" + st.Title
		if titles[key] {
			t.Errorf("trùng tiêu đề trong dự án: %s", key)
		}
		titles[key] = true
		if !st.ToMember {
			backlog++
		}
	}
	if backlog == 0 {
		t.Error("cần ít nhất một việc chưa giao để Member thử nhận việc")
	}
}
