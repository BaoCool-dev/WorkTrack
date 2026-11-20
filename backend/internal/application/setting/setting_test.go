package setting_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/setting"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

type fakeStore struct {
	vals map[string]int
	by   map[string]uuid.UUID
	at   time.Time
	err  error
	sets int
}

func newStore() *fakeStore {
	return &fakeStore{vals: map[string]int{domain.SettingStoryPointHours: 4}, by: map[string]uuid.UUID{}, at: time.Unix(1700000000, 0)}
}

func (f *fakeStore) GetInt(_ context.Context, key string) (int, time.Time, error) {
	if f.err != nil {
		return 0, time.Time{}, f.err
	}
	return f.vals[key], f.at, nil
}

func (f *fakeStore) SetInt(_ context.Context, key string, v int, by uuid.UUID) error {
	f.sets++
	f.vals[key] = v
	f.by[key] = by
	return nil
}

var (
	admin  = authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin}
	ceo    = authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO}
	mgr    = authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	member = authz.Principal{UserID: uuid.New(), Role: domain.RoleMember}
)

func TestGet_AnyRole_B1(t *testing.T) {
	s := setting.NewService(newStore())
	for _, p := range []authz.Principal{admin, ceo, mgr, member} {
		got, err := s.Get(context.Background(), p)
		if err != nil || got.HoursPerStoryPoint != 4 || got.UpdatedAt.IsZero() {
			t.Fatalf("%s: %+v %v", p.Role, got, err)
		}
	}
	if _, err := s.Get(context.Background(), authz.Principal{}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("principal rỗng: %v", err)
	}
}

func TestUpdate_OnlyAdminCEO_Validates_B1(t *testing.T) {
	ctx := context.Background()
	st := newStore()
	s := setting.NewService(st)
	for _, p := range []authz.Principal{mgr, member} {
		if _, err := s.Update(ctx, p, 8); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("%s: %v", p.Role, err)
		}
	}
	for _, v := range []int{0, -1, 41} {
		if _, err := s.Update(ctx, admin, v); !errors.Is(err, domain.ErrInvalidSetting) {
			t.Fatalf("%d: %v", v, err)
		}
	}
	if st.sets != 0 {
		t.Fatal("không được ghi khi bị từ chối/không hợp lệ")
	}
	got, err := s.Update(ctx, ceo, 8)
	if err != nil || got.HoursPerStoryPoint != 8 || st.by[domain.SettingStoryPointHours] != ceo.UserID {
		t.Fatalf("update: %+v %v", got, err)
	}
	h, err := s.HoursPerStoryPoint(ctx)
	if err != nil || h != 8 {
		t.Fatalf("hours: %d %v", h, err)
	}
}

func TestHoursPerStoryPoint_WrapsError_B1(t *testing.T) {
	st := newStore()
	st.err = errors.New("boom")
	if _, err := setting.NewService(st).HoursPerStoryPoint(context.Background()); !errors.Is(err, st.err) {
		t.Fatalf("got %v", err)
	}
}
