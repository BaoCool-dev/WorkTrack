//go:build integration

package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/project"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func newIntegrationUser(t *testing.T, users *UserRepo, email string, role domain.Role) uuid.UUID {
	t.Helper()
	id, err := users.Create(t.Context(), account.NewUser{
		Email: email, FullName: email, Role: role,
		ActivationTokenHash: []byte(email), ActivationExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// Chạy Service thật trên Postgres: transaction, ràng buộc DEFERRABLE vị trí cột, cascade luật chuyển.
func TestProjectRepo_ServiceFlow_CF03(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	users := NewUserRepo(pool)
	mgrID := newIntegrationUser(t, users, "mgr@worktrack.local", domain.RoleManager)
	memID := newIntegrationUser(t, users, "mem@worktrack.local", domain.RoleMember)
	adminID := newIntegrationUser(t, users, "admin@worktrack.local", domain.RoleAdmin)

	svc := project.NewService(NewProjectRepo(pool), NewProjectTx(pool))
	mgr := authz.Principal{UserID: mgrID, Role: domain.RoleManager}
	mem := authz.Principal{UserID: memID, Role: domain.RoleMember}
	ceo := authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO}

	start := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	p, err := svc.Create(ctx, mgr, project.Input{Name: "Cổng thông tin", StartDate: &start})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != domain.ProjectNew || p.StartDate == nil || !p.StartDate.Equal(start) || p.MemberCount != 1 {
		t.Fatalf("dự án vừa tạo sai: %+v", p)
	}
	if _, err := svc.Create(ctx, mgr, project.Input{Name: "cổng thông tin"}); !errors.Is(err, domain.ErrProjectNameTaken) {
		t.Fatalf("trùng tên không phân biệt hoa thường: %v", err)
	}

	b, err := svc.GetBoard(ctx, mgr, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Columns) != 5 || len(b.Transitions) == 0 {
		t.Fatalf("bảng mặc định: %d cột, %d luật", len(b.Columns), len(b.Transitions))
	}

	// Member chưa là thành viên → không thấy (404); CEO thấy tất cả.
	if _, err := svc.Get(ctx, mem, p.ID); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("member ngoài dự án: %v", err)
	}
	if page, err := svc.List(ctx, ceo, project.ListInput{}, domain.PageRequest{Page: 1, PageSize: 20}); err != nil || page.Total != 1 {
		t.Fatalf("CEO list: %+v %v", page, err)
	}
	if err := svc.AddMember(ctx, mgr, p.ID, adminID); !errors.Is(err, domain.ErrInvalidProjectMember) {
		t.Fatalf("thêm Admin vào dự án: %v", err)
	}
	if err := svc.AddMember(ctx, mgr, p.ID, memID); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddMember(ctx, mgr, p.ID, memID); err != nil {
		t.Fatalf("thêm lại phải idempotent: %v", err)
	}
	if page, err := svc.List(ctx, mem, project.ListInput{Search: "thông"}, domain.PageRequest{Page: 1, PageSize: 20}); err != nil || page.Total != 1 {
		t.Fatalf("member list sau khi thêm: %+v %v", page, err)
	}

	// Đảo ngược thứ tự cột: chỉ chạy được nhờ ràng buộc vị trí DEFERRABLE.
	ids := make([]uuid.UUID, len(b.Columns))
	for i, c := range b.Columns {
		ids[len(ids)-1-i] = c.ID
	}
	b, err = svc.ReorderColumns(ctx, mgr, p.ID, ids)
	if err != nil {
		t.Fatalf("sắp xếp cột: %v", err)
	}
	for i, c := range b.Columns {
		if c.ID != ids[i] || c.Position != i {
			t.Fatalf("vị trí sau sắp xếp sai tại %d: %+v", i, c)
		}
	}

	// Xóa cột: luật liên quan biến mất (cascade), vị trí đánh lại liên tục.
	victim := b.Columns[1]
	if victim.StatusGroup == domain.GroupDone {
		victim = b.Columns[2]
	}
	b, err = svc.DeleteColumn(ctx, mgr, p.ID, victim.ID)
	if err != nil {
		t.Fatalf("xóa cột: %v", err)
	}
	for i, c := range b.Columns {
		if c.Position != i {
			t.Fatalf("vị trí sau xóa không liên tục: %+v", b.Columns)
		}
	}
	for _, tr := range b.Transitions {
		if tr.FromColumnID == victim.ID || tr.ToColumnID == victim.ID {
			t.Fatal("luật chuyển của cột đã xóa vẫn còn")
		}
	}
	if _, err := svc.AddColumn(ctx, mgr, p.ID, b.Columns[0].Name, "todo"); !errors.Is(err, domain.ErrColumnNameTaken) {
		t.Fatalf("trùng tên cột: %v", err)
	}

	// Lỗi giữa transaction → rollback (luật chuyển tham chiếu cột lạ).
	before := len(b.Transitions)
	bad := []domain.Transition{{FromColumnID: b.Columns[0].ID, ToColumnID: uuid.New(), Role: domain.TransitionManager}}
	if _, err := svc.ReplaceTransitions(ctx, mgr, p.ID, bad); err == nil {
		t.Fatal("luật chuyển tới cột lạ phải bị từ chối")
	}
	if b2, _ := svc.GetBoard(ctx, mgr, p.ID); len(b2.Transitions) != before {
		t.Fatalf("không rollback: %d → %d luật", before, len(b2.Transitions))
	}

	// IDOR qua SQL thật: mã cột của dự án A gọi trên đường dẫn dự án B → không tìm thấy, A không đổi.
	p2, err := svc.Create(ctx, mgr, project.Input{Name: "Dự án thứ hai"})
	if err != nil {
		t.Fatal(err)
	}
	colA := b.Columns[0]
	if _, err := svc.UpdateColumn(ctx, mgr, p2.ID, colA.ID, "Bị đổi", "todo"); !errors.Is(err, domain.ErrColumnNotFound) {
		t.Fatalf("sửa cột dự án khác: %v", err)
	}
	if _, err := svc.DeleteColumn(ctx, mgr, p2.ID, colA.ID); !errors.Is(err, domain.ErrColumnNotFound) {
		t.Fatalf("xóa cột dự án khác: %v", err)
	}
	if b3, _ := svc.GetBoard(ctx, mgr, p.ID); b3.Columns[0].Name != colA.Name {
		t.Fatal("cột dự án A bị thay đổi qua dự án B")
	}

	if err := svc.RemoveMember(ctx, mgr, p.ID, mgrID); !errors.Is(err, domain.ErrCannotRemoveProjectManager) {
		t.Fatalf("bỏ Manager phụ trách: %v", err)
	}
	if err := svc.RemoveMember(ctx, mgr, p.ID, memID); err != nil {
		t.Fatal(err)
	}
}
