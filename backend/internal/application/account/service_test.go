package account_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/account"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/securetoken"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

var (
	t0      = time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)
	errBoom = errors.New("boom")
)

type fakeStore struct {
	created  []account.NewUser
	users    map[uuid.UUID]domain.User
	hashes   map[uuid.UUID][]byte
	err      error
	listSeen domain.PageRequest
}

func newStore() *fakeStore {
	return &fakeStore{users: map[uuid.UUID]domain.User{}, hashes: map[uuid.UUID][]byte{}}
}

func (f *fakeStore) Create(_ context.Context, u account.NewUser) (uuid.UUID, error) {
	if f.err != nil {
		return uuid.Nil, f.err
	}
	for _, x := range f.users {
		if strings.EqualFold(x.Email, u.Email) {
			return uuid.Nil, domain.ErrEmailTaken
		}
	}
	id := uuid.New()
	f.created = append(f.created, u)
	f.users[id] = domain.User{ID: id, Email: u.Email, FullName: u.FullName, Role: u.Role, Status: domain.UserStatusPending}
	f.hashes[id] = u.ActivationTokenHash
	return id, nil
}

func (f *fakeStore) List(_ context.Context, p domain.PageRequest) ([]domain.User, int, error) {
	f.listSeen = p
	if f.err != nil {
		return nil, 0, f.err
	}
	return nil, len(f.users), nil
}

func (f *fakeStore) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	if f.err != nil {
		return domain.User{}, f.err
	}
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (f *fakeStore) SetActivationToken(_ context.Context, id uuid.UUID, h []byte, _ time.Time) error {
	f.hashes[id] = h
	return nil
}

type fakeMailer struct {
	sent []account.ActivationEmail
	err  error
}

func (m *fakeMailer) SendActivation(_ context.Context, e account.ActivationEmail) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, e)
	return nil
}

func newSvc() (*account.Service, *fakeStore, *fakeMailer) {
	st, m := newStore(), &fakeMailer{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return account.NewService(st, m, "http://localhost:5173/", func() time.Time { return t0 }, log), st, m
}

func actor(r domain.Role) authz.Principal { return authz.Principal{UserID: uuid.New(), Role: r} }

func validInput(r domain.Role) account.CreateUserInput {
	return account.CreateUserInput{Email: "  Chi.Le@WorkTrack.local ", FullName: " Lê Thị Chi ", Role: r}
}

func tokenFromLink(t *testing.T, link string) string {
	t.Helper()
	u, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	return u.Query().Get("token")
}

func TestCreateUser_AdminCreatesPendingAndSendsEmail_CF01_B7(t *testing.T) {
	svc, st, m := newSvc()
	res, err := svc.CreateUser(context.Background(), actor(domain.RoleAdmin), validInput(domain.RoleMember))
	if err != nil {
		t.Fatal(err)
	}
	if !res.EmailSent || res.User.Status != domain.UserStatusPending || res.User.Email != "chi.le@worktrack.local" || res.User.FullName != "Lê Thị Chi" {
		t.Fatalf("kết quả sai: %+v", res)
	}
	if len(m.sent) != 1 || !strings.HasPrefix(m.sent[0].Link, "http://localhost:5173/activate?token=") {
		t.Fatalf("email sai: %+v", m.sent)
	}
	if !m.sent[0].ExpiresAt.Equal(t0.Add(72 * time.Hour)) {
		t.Fatal("token kích hoạt hiệu lực 72 giờ (B7)")
	}
	raw := tokenFromLink(t, m.sent[0].Link)
	if string(st.created[0].ActivationTokenHash) != string(securetoken.Hash(raw)) {
		t.Fatal("CSDL chỉ lưu hash của token gửi trong email")
	}
}

func TestCreateUser_Forbidden_CF01_B2(t *testing.T) {
	cases := []struct {
		actor domain.Role
		role  domain.Role
		want  error
	}{
		{domain.RoleManager, domain.RoleMember, domain.ErrForbidden},
		{domain.RoleMember, domain.RoleMember, domain.ErrForbidden},
		{domain.RoleAdmin, domain.RoleCEO, domain.ErrForbidden},
		{domain.RoleCEO, domain.RoleCEO, nil},
		{domain.RoleCEO, domain.RoleAdmin, nil},
		{domain.RoleAdmin, domain.RoleManager, nil},
	}
	for _, c := range cases {
		t.Run(string(c.actor)+"->"+string(c.role), func(t *testing.T) {
			svc, st, _ := newSvc()
			_, err := svc.CreateUser(context.Background(), actor(c.actor), validInput(c.role))
			if !errors.Is(err, c.want) {
				t.Fatalf("got %v want %v", err, c.want)
			}
			if c.want != nil && len(st.created) != 0 {
				t.Fatal("không được ghi khi bị từ chối")
			}
		})
	}
}

func TestCreateUser_Validation_CF01(t *testing.T) {
	svc, _, _ := newSvc()
	ctx, a := context.Background(), actor(domain.RoleAdmin)
	in := validInput("boss")
	if _, err := svc.CreateUser(ctx, a, in); !errors.Is(err, domain.ErrInvalidRole) {
		t.Fatalf("role: %v", err)
	}
	for _, name := range []string{"   ", strings.Repeat("a", 101)} {
		in := validInput(domain.RoleMember)
		in.FullName = name
		if _, err := svc.CreateUser(ctx, a, in); !errors.Is(err, domain.ErrInvalidFullName) {
			t.Fatalf("name %q: %v", name, err)
		}
	}
}

func TestCreateUser_DuplicateEmail_CF01(t *testing.T) {
	svc, _, _ := newSvc()
	ctx, a := context.Background(), actor(domain.RoleAdmin)
	if _, err := svc.CreateUser(ctx, a, validInput(domain.RoleMember)); err != nil {
		t.Fatal(err)
	}
	in := validInput(domain.RoleManager)
	in.Email = "CHI.LE@worktrack.local"
	if _, err := svc.CreateUser(ctx, a, in); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("got %v", err)
	}
}

func TestCreateUser_MailFailureStillCreates_CF01(t *testing.T) {
	svc, st, m := newSvc()
	m.err = errBoom
	res, err := svc.CreateUser(context.Background(), actor(domain.RoleAdmin), validInput(domain.RoleMember))
	if err != nil || res.EmailSent || len(st.created) != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestListUsers_CF01(t *testing.T) {
	svc, st, _ := newSvc()
	p := domain.NewPageRequest(2, 10)
	got, err := svc.ListUsers(context.Background(), actor(domain.RoleCEO), p)
	if err != nil || got.Items == nil || got.Page != 2 || got.PageSize != 10 || st.listSeen != p {
		t.Fatalf("got %+v err %v", got, err)
	}
	if _, err := svc.ListUsers(context.Background(), actor(domain.RoleManager), p); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("manager: %v", err)
	}
	st.err = errBoom
	if _, err := svc.ListUsers(context.Background(), actor(domain.RoleAdmin), p); !errors.Is(err, errBoom) {
		t.Fatalf("store: %v", err)
	}
}

func TestResendActivation_NewTokenInvalidatesOld_CF01_B7(t *testing.T) {
	svc, st, m := newSvc()
	ctx, a := context.Background(), actor(domain.RoleAdmin)
	res, _ := svc.CreateUser(ctx, a, validInput(domain.RoleMember))
	if err := svc.ResendActivation(ctx, a, res.User.ID); err != nil {
		t.Fatal(err)
	}
	if len(m.sent) != 2 {
		t.Fatal("phải gửi lại email")
	}
	newTok := tokenFromLink(t, m.sent[1].Link)
	if newTok == tokenFromLink(t, m.sent[0].Link) || string(st.hashes[res.User.ID]) != string(securetoken.Hash(newTok)) {
		t.Fatal("phải thay token mới")
	}
}

func TestResendActivation_Errors_CF01(t *testing.T) {
	svc, st, m := newSvc()
	ctx, a := context.Background(), actor(domain.RoleAdmin)
	if err := svc.ResendActivation(ctx, actor(domain.RoleMember), uuid.New()); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("member: %v", err)
	}
	if err := svc.ResendActivation(ctx, a, uuid.New()); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("not found: %v", err)
	}
	ceoID, activeID, pendingID := uuid.New(), uuid.New(), uuid.New()
	st.users[ceoID] = domain.User{ID: ceoID, Role: domain.RoleCEO, Status: domain.UserStatusPending}
	st.users[activeID] = domain.User{ID: activeID, Role: domain.RoleMember, Status: domain.UserStatusActive}
	st.users[pendingID] = domain.User{ID: pendingID, Role: domain.RoleMember, Status: domain.UserStatusPending}
	if err := svc.ResendActivation(ctx, a, ceoID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("admin gửi lại cho CEO: %v", err)
	}
	if err := svc.ResendActivation(ctx, a, activeID); !errors.Is(err, domain.ErrUserAlreadyActive) {
		t.Fatalf("active: %v", err)
	}
	m.err = errBoom
	if err := svc.ResendActivation(ctx, a, pendingID); !errors.Is(err, domain.ErrEmailDeliveryFailed) {
		t.Fatalf("mail: %v", err)
	}
}
