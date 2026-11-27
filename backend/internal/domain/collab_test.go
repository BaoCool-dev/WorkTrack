package domain_test

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestCheckSubtaskActor_CF07(t *testing.T) {
	pm, member, other := uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name     string
		actor    uuid.UUID
		role     domain.Role
		assignee *uuid.UUID
		want     error
	}{
		{"manager phụ trách", pm, domain.RoleManager, &member, nil},
		{"manager phụ trách, việc chưa giao", pm, domain.RoleManager, nil, nil},
		{"member là người nhận", member, domain.RoleMember, &member, nil},
		{"member không phải người nhận", other, domain.RoleMember, &member, domain.ErrSubtaskNotAllowed},
		{"member, việc chưa giao", member, domain.RoleMember, nil, domain.ErrSubtaskNotAllowed},
		{"manager khác dự án", other, domain.RoleManager, &member, domain.ErrSubtaskNotAllowed},
		{"CEO chỉ xem", pm, domain.RoleCEO, &member, domain.ErrForbidden},
		{"Admin", pm, domain.RoleAdmin, &member, domain.ErrForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := domain.CheckSubtaskActor(tc.actor, tc.role, pm, tc.assignee)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestCheckSubtaskParent_CF07(t *testing.T) {
	root := domain.Task{ID: uuid.New()}
	if err := domain.CheckSubtaskParent(root); err != nil {
		t.Fatalf("việc gốc phải làm cha được: %v", err)
	}
	pid := uuid.New()
	child := domain.Task{ID: uuid.New(), ParentID: &pid}
	if err := domain.CheckSubtaskParent(child); !errors.Is(err, domain.ErrSubtaskParentNotRoot) {
		t.Fatalf("con không được làm cha (1 cấp), got %v", err)
	}
}

func TestNormalizeComment_CF07(t *testing.T) {
	got, err := domain.NormalizeComment("  Xin chào\nnhóm  ")
	if err != nil || got != "Xin chào\nnhóm" {
		t.Fatalf("got %q, %v", got, err)
	}
	ok := strings.Repeat("ạ", domain.CommentMaxRunes)
	if _, err := domain.NormalizeComment(ok); err != nil {
		t.Fatalf("2000 ký tự phải hợp lệ: %v", err)
	}
	for _, bad := range []string{"", "   \n\t", ok + "a"} {
		if _, err := domain.NormalizeComment(bad); !errors.Is(err, domain.ErrInvalidComment) {
			t.Fatalf("%d ký tự phải lỗi, got %v", utf8.RuneCountInString(bad), err)
		}
	}
}

func TestCanComment_CF07(t *testing.T) {
	for _, r := range []domain.Role{domain.RoleManager, domain.RoleMember} {
		if err := domain.CanComment(r); err != nil {
			t.Fatalf("%s phải bình luận được: %v", r, err)
		}
	}
	for _, r := range []domain.Role{domain.RoleCEO, domain.RoleAdmin} {
		if err := domain.CanComment(r); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("%s chỉ xem, got %v", r, err)
		}
	}
}

func TestNotificationRecipients_CF07(t *testing.T) {
	actor, a, b := uuid.New(), uuid.New(), uuid.New()
	got := domain.NotificationRecipients(actor, &a, nil, &actor, &b, &a)
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("phải bỏ nil, bỏ người gây ra, bỏ trùng và giữ thứ tự: %v", got)
	}
	if got := domain.NotificationRecipients(actor, &actor, nil); got == nil || len(got) != 0 {
		t.Fatalf("không ai nhận → slice rỗng khác nil, got %v", got)
	}
}

func TestAssignmentRecipient_CF07(t *testing.T) {
	actor, a, b := uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name     string
		old, new *uuid.UUID
		want     *uuid.UUID
	}{
		{"giao lần đầu", nil, &a, &a},
		{"đổi người nhận", &a, &b, &b},
		{"giữ nguyên người nhận", &a, &a, nil},
		{"bỏ giao", &a, nil, nil},
		{"tự giao cho mình", nil, &actor, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := domain.AssignmentRecipient(actor, tc.old, tc.new)
			if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNotificationExcerpt_CF07(t *testing.T) {
	if got := domain.NotificationExcerpt("  a\n\nb  "); got != "a b" {
		t.Fatalf("phải gộp khoảng trắng, got %q", got)
	}
	long := strings.Repeat("đ", domain.NotificationExcerptRunes+10)
	got := domain.NotificationExcerpt(long)
	if utf8.RuneCountInString(got) != domain.NotificationExcerptRunes+1 || !strings.HasSuffix(got, "…") {
		t.Fatalf("phải cắt %d ký tự + dấu …, got %d", domain.NotificationExcerptRunes, utf8.RuneCountInString(got))
	}
}

func TestNotificationTypeValid_CF07(t *testing.T) {
	for _, ty := range []domain.NotificationType{
		domain.NotificationTaskAssigned, domain.NotificationTaskCommented, domain.NotificationReviewRejected,
	} {
		if !ty.Valid() {
			t.Fatalf("%s phải hợp lệ", ty)
		}
	}
	if domain.NotificationType("other").Valid() {
		t.Fatal("loại lạ phải không hợp lệ")
	}
}
