package auth_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

var errBoom = errors.New("boom")

type fakeUsers struct {
	byID       map[uuid.UUID]*domain.User
	tokenHash  map[uuid.UUID][]byte
	err        error
	failCalls  int
	successAt  *time.Time
	activateFn func() error
}

func newFakeUsers(us ...domain.User) *fakeUsers {
	f := &fakeUsers{byID: map[uuid.UUID]*domain.User{}, tokenHash: map[uuid.UUID][]byte{}}
	for i := range us {
		u := us[i]
		f.byID[u.ID] = &u
	}
	return f
}

func (f *fakeUsers) GetByEmail(_ context.Context, email string) (domain.User, error) {
	if f.err != nil {
		return domain.User{}, f.err
	}
	for _, u := range f.byID {
		if strings.EqualFold(u.Email, email) {
			return *u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (f *fakeUsers) GetByID(_ context.Context, id uuid.UUID) (domain.User, error) {
	if f.err != nil {
		return domain.User{}, f.err
	}
	if u, ok := f.byID[id]; ok {
		return *u, nil
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (f *fakeUsers) GetByActivationTokenHash(_ context.Context, h []byte) (domain.User, error) {
	if f.err != nil {
		return domain.User{}, f.err
	}
	for id, th := range f.tokenHash {
		if bytes.Equal(th, h) {
			return *f.byID[id], nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func (f *fakeUsers) Activate(_ context.Context, id uuid.UUID, h []byte, pw string) error {
	if f.activateFn != nil {
		return f.activateFn()
	}
	if !bytes.Equal(f.tokenHash[id], h) {
		return domain.ErrActivationTokenInvalid
	}
	u := f.byID[id]
	u.Status, u.PasswordHash = domain.UserStatusActive, pw
	delete(f.tokenHash, id)
	return nil
}

func (f *fakeUsers) UpdateLoginFailure(_ context.Context, id uuid.UUID, count int, lockedUntil *time.Time) error {
	f.failCalls++
	u := f.byID[id]
	u.FailedLoginCount, u.LockedUntil = count, lockedUntil
	return nil
}

func (f *fakeUsers) RecordLoginSuccess(_ context.Context, id uuid.UUID, at time.Time) error {
	f.successAt = &at
	u := f.byID[id]
	u.FailedLoginCount, u.LockedUntil = 0, nil
	return nil
}

type fakeTokens struct {
	items       map[uuid.UUID]*domain.RefreshToken
	err         error
	raceOnce    bool // RevokeIfActive trả false một lần (giả lập request song song)
	familyCalls int
}

func newFakeTokens() *fakeTokens { return &fakeTokens{items: map[uuid.UUID]*domain.RefreshToken{}} }

func (f *fakeTokens) Insert(_ context.Context, t domain.RefreshToken) error {
	if f.err != nil {
		return f.err
	}
	f.items[t.ID] = &t
	return nil
}

func (f *fakeTokens) GetByHash(_ context.Context, h []byte) (domain.RefreshToken, error) {
	if f.err != nil {
		return domain.RefreshToken{}, f.err
	}
	for _, t := range f.items {
		if bytes.Equal(t.TokenHash, h) {
			return *t, nil
		}
	}
	return domain.RefreshToken{}, domain.ErrRefreshTokenNotFound
}

func (f *fakeTokens) RevokeIfActive(_ context.Context, id uuid.UUID, at time.Time) (bool, error) {
	if f.raceOnce {
		f.raceOnce = false
		return false, nil
	}
	t := f.items[id]
	if t.RevokedAt != nil {
		return false, nil
	}
	t.RevokedAt = &at
	return true, nil
}

func (f *fakeTokens) RevokeFamily(_ context.Context, fam uuid.UUID, at time.Time) error {
	f.familyCalls++
	for _, t := range f.items {
		if t.FamilyID == fam && t.RevokedAt == nil {
			t.RevokedAt = &at
		}
	}
	return nil
}

func (f *fakeTokens) activeCount() int {
	n := 0
	for _, t := range f.items {
		if t.RevokedAt == nil {
			n++
		}
	}
	return n
}

type fakeHasher struct{ compares int }

func (h *fakeHasher) Hash(pw string) (string, error) { return fakeHash(pw), nil }
func (h *fakeHasher) Compare(hash, pw string) bool {
	h.compares++
	return hash == fakeHash(pw)
}

type fakeIssuer struct{ err error }

func (i fakeIssuer) Issue(p authz.Principal, now time.Time) (string, time.Time, error) {
	if i.err != nil {
		return "", time.Time{}, i.err
	}
	return "access-" + p.UserID.String() + "-" + string(p.Role), now.Add(domain.AccessTokenTTL), nil
}

const testPassword = "matkhau123"

func fakeHash(pw string) string { return "hash:" + pw }
