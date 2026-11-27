package label_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/label"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

type fakeStore struct {
	items map[uuid.UUID]domain.Label
	inUse map[uuid.UUID]bool
	calls int
	err   error
}

func newStore() *fakeStore {
	return &fakeStore{items: map[uuid.UUID]domain.Label{}, inUse: map[uuid.UUID]bool{}}
}

func (f *fakeStore) taken(id uuid.UUID, name string) bool {
	for _, l := range f.items {
		if l.ID != id && strings.EqualFold(l.Name, name) {
			return true
		}
	}
	return false
}

func (f *fakeStore) List(context.Context) ([]domain.Label, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	var out []domain.Label
	for _, l := range f.items {
		out = append(out, l)
	}
	return out, nil
}

func (f *fakeStore) Create(_ context.Context, name string) (uuid.UUID, error) {
	f.calls++
	if f.taken(uuid.Nil, name) {
		return uuid.Nil, domain.ErrLabelNameTaken
	}
	id := uuid.New()
	f.items[id] = domain.Label{ID: id, Name: name}
	return id, nil
}

func (f *fakeStore) Update(_ context.Context, id uuid.UUID, name string) error {
	f.calls++
	l, ok := f.items[id]
	if !ok {
		return domain.ErrLabelNotFound
	}
	if f.taken(id, name) {
		return domain.ErrLabelNameTaken
	}
	l.Name = name
	f.items[id] = l
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id uuid.UUID) error {
	f.calls++
	if _, ok := f.items[id]; !ok {
		return domain.ErrLabelNotFound
	}
	if f.inUse[id] {
		return domain.ErrLabelInUse
	}
	delete(f.items, id)
	return nil
}

var (
	admin   = authz.Principal{UserID: uuid.New(), Role: domain.RoleAdmin}
	ceo     = authz.Principal{UserID: uuid.New(), Role: domain.RoleCEO}
	manager = authz.Principal{UserID: uuid.New(), Role: domain.RoleManager}
	member  = authz.Principal{UserID: uuid.New(), Role: domain.RoleMember}
)

func TestList_AnyRole_EmptyNotNil_CF06(t *testing.T) {
	for _, p := range []authz.Principal{admin, ceo, manager, member} {
		xs, err := label.NewService(newStore()).List(context.Background(), p)
		if err != nil || xs == nil {
			t.Fatalf("%s: %v %v", p.Role, xs, err)
		}
	}
	if _, err := label.NewService(newStore()).List(context.Background(), authz.Principal{}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("principal rỗng: %v", err)
	}
}

func TestList_WrapsStoreError_CF06(t *testing.T) {
	st := newStore()
	st.err = errors.New("boom")
	if _, err := label.NewService(st).List(context.Background(), admin); !errors.Is(err, st.err) {
		t.Fatalf("got %v", err)
	}
}

func TestWrite_OnlyAdminAndCEO_CF06(t *testing.T) {
	ctx := context.Background()
	for _, p := range []authz.Principal{manager, member} {
		st := newStore()
		s := label.NewService(st)
		if _, err := s.Create(ctx, p, "Bug"); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("Create %s: %v", p.Role, err)
		}
		if _, err := s.Update(ctx, p, uuid.New(), "Bug"); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("Update %s: %v", p.Role, err)
		}
		if err := s.Delete(ctx, p, uuid.New()); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("Delete %s: %v", p.Role, err)
		}
		if st.calls != 0 {
			t.Fatalf("store bị gọi dù không có quyền")
		}
	}
	for _, p := range []authz.Principal{admin, ceo} {
		if _, err := label.NewService(newStore()).Create(ctx, p, "Bug"); err != nil {
			t.Fatalf("%s: %v", p.Role, err)
		}
	}
}

func TestCreateUpdateDelete_CF06(t *testing.T) {
	ctx := context.Background()
	st := newStore()
	s := label.NewService(st)
	l, err := s.Create(ctx, admin, "   Giao   diện ")
	if err != nil || l.Name != "Giao diện" {
		t.Fatalf("chuẩn hóa tên: %+v %v", l, err)
	}
	if _, err := s.Create(ctx, admin, "giao diện"); !errors.Is(err, domain.ErrLabelNameTaken) {
		t.Fatalf("trùng tên không phân biệt hoa thường: %v", err)
	}
	if _, err := s.Create(ctx, admin, "  "); !errors.Is(err, domain.ErrInvalidOrgName) {
		t.Fatalf("tên rỗng: %v", err)
	}
	u, err := s.Update(ctx, admin, l.ID, "UI")
	if err != nil || u.Name != "UI" {
		t.Fatalf("update: %+v %v", u, err)
	}
	if _, err := s.Update(ctx, admin, uuid.New(), "X"); !errors.Is(err, domain.ErrLabelNotFound) {
		t.Fatalf("update không tồn tại: %v", err)
	}
	st.inUse[l.ID] = true
	if err := s.Delete(ctx, admin, l.ID); !errors.Is(err, domain.ErrLabelInUse) {
		t.Fatalf("xóa nhãn đang dùng: %v", err)
	}
	st.inUse[l.ID] = false
	if err := s.Delete(ctx, admin, l.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
