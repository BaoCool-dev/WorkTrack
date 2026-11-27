package notification_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/authz"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/application/notification"
	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

type row struct {
	n    domain.Notification
	user uuid.UUID
}

type fakeStore struct{ rows []*row }

func (f *fakeStore) List(_ context.Context, uid uuid.UUID, unread bool, _ domain.PageRequest) ([]domain.Notification, int, error) {
	var out []domain.Notification
	for _, r := range f.rows {
		if r.user == uid && (!unread || r.n.ReadAt == nil) {
			out = append(out, r.n)
		}
	}
	return out, len(out), nil
}

func (f *fakeStore) CountUnread(_ context.Context, uid uuid.UUID) (int, error) {
	n := 0
	for _, r := range f.rows {
		if r.user == uid && r.n.ReadAt == nil {
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) MarkRead(_ context.Context, id, uid uuid.UUID) (bool, error) {
	for _, r := range f.rows {
		if r.n.ID == id && r.user == uid {
			if r.n.ReadAt == nil {
				now := time.Now()
				r.n.ReadAt = &now
			}
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) MarkAllRead(_ context.Context, uid uuid.UUID) (int, error) {
	n := 0
	for _, r := range f.rows {
		if r.user == uid && r.n.ReadAt == nil {
			now := time.Now()
			r.n.ReadAt = &now
			n++
		}
	}
	return n, nil
}

func TestNotifications_CF07(t *testing.T) {
	ctx := context.Background()
	me := authz.Principal{UserID: uuid.New(), Role: domain.RoleMember}
	other := authz.Principal{UserID: uuid.New(), Role: domain.RoleMember}
	mine1, mine2, theirs := uuid.New(), uuid.New(), uuid.New()
	st := &fakeStore{rows: []*row{
		{domain.Notification{ID: mine1}, me.UserID},
		{domain.Notification{ID: mine2}, me.UserID},
		{domain.Notification{ID: theirs}, other.UserID},
	}}
	svc := notification.NewService(st)

	if n, _ := svc.UnreadCount(ctx, me); n != 2 {
		t.Fatalf("chưa đọc = %d, want 2", n)
	}
	if err := svc.MarkRead(ctx, me, theirs); !errors.Is(err, domain.ErrNotificationNotFound) {
		t.Fatalf("đánh dấu thông báo của người khác phải 404, got %v", err)
	}
	if err := svc.MarkRead(ctx, me, mine1); err != nil {
		t.Fatal(err)
	}
	if err := svc.MarkRead(ctx, me, mine1); err != nil {
		t.Fatalf("đánh dấu lại phải idempotent: %v", err)
	}
	page, err := svc.List(ctx, me, true, domain.NewPageRequest(1, 20))
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != mine2 {
		t.Fatalf("lọc chưa đọc sai: %+v %v", page, err)
	}
	if n, _ := svc.MarkAllRead(ctx, me); n != 1 {
		t.Fatalf("read-all = %d, want 1", n)
	}
	if n, _ := svc.UnreadCount(ctx, other); n != 1 {
		t.Fatal("không được đụng thông báo người khác")
	}
	empty, _ := svc.List(ctx, me, true, domain.NewPageRequest(1, 20))
	if empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("rỗng phải là slice khác nil: %+v", empty.Items)
	}
}
