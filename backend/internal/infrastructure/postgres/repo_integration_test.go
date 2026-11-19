//go:build integration

package postgres

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/securetoken"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestUserRepo_CreateActivateLogin_CF01(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	repo := NewUserRepo(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	_, hash, _ := securetoken.New()

	id, err := repo.Create(ctx, account.NewUser{
		Email: "an.nguyen@worktrack.local", FullName: "Nguyễn Văn An", Role: domain.RoleMember,
		ActivationTokenHash: hash, ActivationExpiresAt: now.Add(domain.ActivationTokenTTL),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Create(ctx, account.NewUser{
		Email: "AN.NGUYEN@worktrack.local", FullName: "X", Role: domain.RoleMember,
		ActivationTokenHash: []byte("khac"), ActivationExpiresAt: now,
	}); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("trùng email (không phân biệt hoa thường): %v", err)
	}

	u, err := repo.GetByActivationTokenHash(ctx, hash)
	if err != nil || u.ID != id || u.Status != domain.UserStatusPending || u.Role != domain.RoleMember {
		t.Fatalf("get by token: %+v %v", u, err)
	}
	if u.ActivationExpiresAt == nil || !u.ActivationExpiresAt.Equal(now.Add(domain.ActivationTokenTTL)) {
		t.Fatalf("hạn kích hoạt: %v", u.ActivationExpiresAt)
	}

	if err := repo.Activate(ctx, id, []byte("sai"), "bcrypt"); !errors.Is(err, domain.ErrActivationTokenInvalid) {
		t.Fatalf("token sai: %v", err)
	}
	if err := repo.Activate(ctx, id, hash, "bcrypt-hash"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Activate(ctx, id, hash, "bcrypt-hash"); !errors.Is(err, domain.ErrActivationTokenInvalid) {
		t.Fatalf("dùng lại token: %v", err)
	}
	if _, err := repo.GetByActivationTokenHash(ctx, hash); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("token phải bị xóa: %v", err)
	}
	if err := repo.SetActivationToken(ctx, id, []byte("moi"), now); !errors.Is(err, domain.ErrUserAlreadyActive) {
		t.Fatalf("đã active: %v", err)
	}

	lock := now.Add(domain.LoginLockDuration)
	if err := repo.UpdateLoginFailure(ctx, id, 5, &lock); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.GetByEmail(ctx, "An.Nguyen@WorkTrack.local")
	if u.FailedLoginCount != 5 || u.LockedUntil == nil || !u.LockedUntil.Equal(lock) || u.PasswordHash != "bcrypt-hash" {
		t.Fatalf("sau khi sai: %+v", u)
	}
	if err := repo.RecordLoginSuccess(ctx, id, now); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.GetByID(ctx, id)
	if u.FailedLoginCount != 0 || u.LockedUntil != nil || u.LastLoginAt == nil {
		t.Fatalf("sau khi thành công: %+v", u)
	}
	if _, err := repo.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("not found: %v", err)
	}

	page, total, err := repo.List(ctx, domain.NewPageRequest(1, 10))
	if err != nil || total != 1 || len(page) != 1 || page[0].ID != id {
		t.Fatalf("list: %v %d %v", page, total, err)
	}
	page, _, _ = repo.List(ctx, domain.NewPageRequest(2, 10))
	if len(page) != 0 {
		t.Fatal("trang 2 phải rỗng")
	}
}

func TestRefreshTokenRepo_Rotation_CF01_B4(t *testing.T) {
	pool := newTestPool(t)
	ctx := t.Context()
	users := NewUserRepo(pool)
	repo := NewRefreshTokenRepo(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	uid, err := users.Create(ctx, account.NewUser{
		Email: "binh@worktrack.local", FullName: "Trần Bình", Role: domain.RoleManager,
		ActivationTokenHash: []byte("t"), ActivationExpiresAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	fam := uuid.New()
	a := domain.RefreshToken{ID: uuid.New(), UserID: uid, FamilyID: fam, TokenHash: []byte("a"), ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	b := a
	b.ID, b.TokenHash = uuid.New(), []byte("b")
	for _, x := range []domain.RefreshToken{a, b} {
		if err := repo.Insert(ctx, x); err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.GetByHash(ctx, []byte("a"))
	if err != nil || got.ID != a.ID || got.FamilyID != fam || got.RevokedAt != nil || !got.ExpiresAt.Equal(a.ExpiresAt) {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := repo.GetByHash(ctx, []byte("zzz")); !errors.Is(err, domain.ErrRefreshTokenNotFound) {
		t.Fatalf("not found: %v", err)
	}
	ok, err := repo.RevokeIfActive(ctx, a.ID, now)
	if err != nil || !ok {
		t.Fatalf("revoke lần 1: %v %v", ok, err)
	}
	if ok, _ := repo.RevokeIfActive(ctx, a.ID, now); ok {
		t.Fatal("revoke lần 2 phải trả false")
	}
	if err := repo.RevokeFamily(ctx, fam, now); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.GetByHash(ctx, []byte("b"))
	if got.RevokedAt == nil {
		t.Fatal("cả họ phải bị thu hồi")
	}
}
