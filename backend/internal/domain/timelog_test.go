package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestNormalizeTimeLog_CF08(t *testing.T) {
	tests := []struct {
		name     string
		minutes  int
		note     string
		wantNote string
		want     error
	}{
		{"1 phút (biên dưới)", 1, "Viết API", "Viết API", nil},
		{"1440 phút (biên trên)", 1440, "Cả ngày", "Cả ngày", nil},
		{"0 phút", 0, "x", "", domain.ErrInvalidLogMinutes},
		{"phút âm", -30, "x", "", domain.ErrInvalidLogMinutes},
		{"1441 phút", 1441, "x", "", domain.ErrInvalidLogMinutes},
		{"cắt khoảng trắng", 30, "  Sửa lỗi  ", "Sửa lỗi", nil},
		{"mô tả rỗng", 30, "   ", "", domain.ErrInvalidLogNote},
		{"mô tả 500 ký tự có dấu", 30, strings.Repeat("ệ", 500), strings.Repeat("ệ", 500), nil},
		{"mô tả 501 ký tự", 30, strings.Repeat("a", 501), "", domain.ErrInvalidLogNote},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.NormalizeTimeLog(tt.minutes, tt.note)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, muốn %v", err, tt.want)
			}
			if got != tt.wantNote {
				t.Fatalf("note = %q, muốn %q", got, tt.wantNote)
			}
		})
	}
}

func TestCheckLogWindow_CF08(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		group  domain.StatusGroup
		doneAt time.Time
		want   error
	}{
		{"chưa Done thì luôn được log", domain.GroupInProgress, now.Add(-72 * time.Hour), nil},
		{"todo", domain.GroupTodo, time.Time{}, nil},
		{"Done 1 giờ trước", domain.GroupDone, now.Add(-time.Hour), nil},
		{"Done đúng 24 giờ (biên, vẫn được)", domain.GroupDone, now.Add(-24 * time.Hour), nil},
		{"Done 24 giờ 1 giây", domain.GroupDone, now.Add(-24*time.Hour - time.Second), domain.ErrTimeLogClosed},
		{"Done 3 ngày", domain.GroupDone, now.Add(-72 * time.Hour), domain.ErrTimeLogClosed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.CheckLogWindow(tt.group, tt.doneAt, now); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, muốn %v", err, tt.want)
			}
		})
	}
}

func TestResolveWorkDate_CF08(t *testing.T) {
	// 18:30 UTC ngày 26 = 01:30 sáng ngày 27 giờ Việt Nam.
	now := time.Date(2026, 11, 26, 18, 30, 0, 0, time.UTC)
	created := time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC)
	day := func(d int) *time.Time { v := time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC); return &v }

	got, err := domain.ResolveWorkDate(nil, created, now)
	if err != nil || !got.Equal(*day(27)) {
		t.Fatalf("mặc định phải là hôm nay theo giờ Việt Nam (27/11), được %v, %v", got, err)
	}
	tests := []struct {
		name string
		in   *time.Time
		want error
	}{
		{"hôm nay (giờ VN)", day(27), nil},
		{"hôm qua", day(26), nil},
		{"ngày tạo việc (biên)", day(20), nil},
		{"trước ngày tạo việc", day(19), domain.ErrInvalidLogDate},
		{"ngày mai", day(28), domain.ErrInvalidLogDate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := domain.ResolveWorkDate(tt.in, created, now); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, muốn %v", err, tt.want)
			}
		})
	}
}

func TestCheckTimeLogActor_CF08(t *testing.T) {
	pm, member, other := uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name     string
		actor    uuid.UUID
		role     domain.Role
		assignee *uuid.UUID
		want     error
	}{
		{"người nhận việc", member, domain.RoleMember, &member, nil},
		{"Manager phụ trách", pm, domain.RoleManager, &member, nil},
		{"Member khác", other, domain.RoleMember, &member, domain.ErrTimeLogNotAllowed},
		{"Member, việc chưa giao", member, domain.RoleMember, nil, domain.ErrTimeLogNotAllowed},
		{"Manager không phụ trách", other, domain.RoleManager, &member, domain.ErrTimeLogNotAllowed},
		{"CEO chỉ xem", other, domain.RoleCEO, &other, domain.ErrForbidden},
		{"Admin", other, domain.RoleAdmin, &other, domain.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.CheckTimeLogActor(tt.actor, tt.role, pm, tt.assignee); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, muốn %v", err, tt.want)
			}
		})
	}
}

func TestCheckTimeLogDelete_CF08(t *testing.T) {
	author, other := uuid.New(), uuid.New()
	log := domain.TimeLog{UserID: author}
	if err := domain.CheckTimeLogDelete(author, log); err != nil {
		t.Fatalf("tác giả phải xóa được: %v", err)
	}
	if err := domain.CheckTimeLogDelete(other, log); !errors.Is(err, domain.ErrTimeLogNotAllowed) {
		t.Fatalf("người khác xóa: err = %v", err)
	}
}
