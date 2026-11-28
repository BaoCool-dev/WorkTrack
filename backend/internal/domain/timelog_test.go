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

// QA bổ sung — BVA 3 giá trị cho số phút và mô tả, đoán lỗi (khoảng trắng lạ, emoji, HTML) [CF-08].
// Đếm "ký tự" theo code point, khớp CHECK char_length(note) của bảng time_log.
func TestNormalizeTimeLog_BVAAndErrorGuessing_CF08(t *testing.T) {
	family := "👨‍👩‍👧" // 5 code point (3 emoji + 2 ZWJ)
	tests := []struct {
		name     string
		minutes  int
		note     string
		wantNote string
		want     error
	}{
		{"2 phút (biên dưới + 1)", 2, "x", "x", nil},
		{"1439 phút (biên trên − 1)", 1439, "x", "x", nil},
		{"phút rất lớn", 1 << 30, "x", "", domain.ErrInvalidLogMinutes},
		{"mô tả chuỗi rỗng tuyệt đối", 30, "", "", domain.ErrInvalidLogNote},
		{"mô tả chỉ tab/xuống dòng", 30, "\t\n \r\n", "", domain.ErrInvalidLogNote},
		{"mô tả 1 ký tự (biên dưới)", 30, "a", "a", nil},
		{"mô tả 499 ký tự", 30, strings.Repeat("a", 499), strings.Repeat("a", 499), nil},
		{"500 emoji 4 byte (500 code point)", 30, strings.Repeat("😀", 500), strings.Repeat("😀", 500), nil},
		{"501 emoji 4 byte", 30, strings.Repeat("😀", 501), "", domain.ErrInvalidLogNote},
		{"emoji ghép ZWJ đếm theo code point: 100 × 5 = 500", 30, strings.Repeat(family, 100), strings.Repeat(family, 100), nil},
		{"emoji ghép ZWJ 100 × 5 + 1 = 501", 30, strings.Repeat(family, 100) + "a", "", domain.ErrInvalidLogNote},
		{"500 ký tự + khoảng trắng hai đầu: đếm sau khi cắt", 30, "  " + strings.Repeat("ệ", 500) + "\n", strings.Repeat("ệ", 500), nil},
		{"tiếng Việt + emoji giữ nguyên", 45, "Sửa lỗi đăng nhập 🐛✅", "Sửa lỗi đăng nhập 🐛✅", nil},
		{"HTML giữ nguyên văn bản (thoát ở tầng hiển thị)", 15, "<script>alert(1)</script>", "<script>alert(1)</script>", nil},
		{"xuống dòng ở giữa giữ nguyên", 15, "Dòng 1\nDòng 2", "Dòng 1\nDòng 2", nil},
		{"phút sai và mô tả rỗng → báo phút trước", 0, "", "", domain.ErrInvalidLogMinutes},
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

// QA bổ sung — biên 24 giờ sau Done ở mức giây/nano giây, lệch đồng hồ, mở lại sau từ chối [CF-08].
func TestCheckLogWindow_Boundaries_CF08(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		group  domain.StatusGroup
		doneAt time.Time
		want   error
	}{
		{"Done 23 giờ 59 phút 59 giây", domain.GroupDone, now.Add(-24*time.Hour + time.Second), nil},
		{"Done 24 giờ + 1 nano giây", domain.GroupDone, now.Add(-24*time.Hour - time.Nanosecond), domain.ErrTimeLogClosed},
		{"vừa Done (0 giây)", domain.GroupDone, now, nil},
		{"mốc Done ở tương lai (lệch đồng hồ) vẫn được", domain.GroupDone, now.Add(time.Minute), nil},
		{"bị từ chối nghiệm thu, quay lại in_progress sau 30 ngày → mở lại", domain.GroupInProgress, now.Add(-30 * 24 * time.Hour), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.CheckLogWindow(tt.group, tt.doneAt, now); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, muốn %v", err, tt.want)
			}
		})
	}
}

// QA bổ sung — biên nửa đêm giờ Việt Nam (UTC+7) cho "ngày tương lai" và "trước ngày tạo việc" [CF-08].
func TestResolveWorkDate_VNMidnightBoundary_CF08(t *testing.T) {
	day := func(d int) *time.Time { v := time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC); return &v }
	// Việc tạo lúc 00:30 ngày 20 giờ VN = 17:30 UTC ngày 19.
	created := time.Date(2026, 11, 19, 17, 30, 0, 0, time.UTC)
	tests := []struct {
		name string
		now  time.Time
		in   *time.Time
		want error
	}{
		{"23:59:59 VN ngày 26: ngày 27 là tương lai", time.Date(2026, 11, 26, 16, 59, 59, 0, time.UTC), day(27), domain.ErrInvalidLogDate},
		{"23:59:59 VN ngày 26: ngày 26 là hôm nay", time.Date(2026, 11, 26, 16, 59, 59, 0, time.UTC), day(26), nil},
		{"00:00 VN ngày 27: ngày 27 đã là hôm nay", time.Date(2026, 11, 26, 17, 0, 0, 0, time.UTC), day(27), nil},
		{"tạo việc 00:30 VN ngày 20: ngày 19 (theo UTC) bị chặn", time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC), day(19), domain.ErrInvalidLogDate},
		{"tạo việc 00:30 VN ngày 20: ngày 20 được", time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC), day(20), nil},
		{"tạo việc và log cùng ngày (hôm nay = ngày tạo)", time.Date(2026, 11, 19, 18, 0, 0, 0, time.UTC), day(20), nil},
		{"ngày rất xa trong tương lai", time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC), func() *time.Time { v := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC); return &v }(), domain.ErrInvalidLogDate},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := domain.ResolveWorkDate(tt.in, created, tt.now); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, muốn %v", err, tt.want)
			}
		})
	}
	// Mặc định (không gửi ngày) lúc 00:00 VN ngày 27 phải là ngày 27, không phải 26 theo UTC.
	got, err := domain.ResolveWorkDate(nil, created, time.Date(2026, 11, 26, 17, 0, 0, 0, time.UTC))
	if err != nil || !got.Equal(*day(27)) {
		t.Fatalf("mặc định = %v, %v; muốn 27/11", got, err)
	}
}

// QA bổ sung — bảng quyết định vai trò × (là Manager phụ trách, là người nhận) [CF-08].
func TestCheckTimeLogActor_DecisionTable_CF08(t *testing.T) {
	pm, member := uuid.New(), uuid.New()
	tests := []struct {
		name     string
		actor    uuid.UUID
		role     domain.Role
		assignee *uuid.UUID
		want     error
	}{
		{"Manager phụ trách, việc chưa giao", pm, domain.RoleManager, nil, nil},
		{"Manager phụ trách tự nhận việc", pm, domain.RoleManager, &pm, nil},
		{"CEO trùng ID Manager phụ trách vẫn bị chặn (chỉ xem)", pm, domain.RoleCEO, &member, domain.ErrForbidden},
		{"Admin là người nhận vẫn bị chặn", member, domain.RoleAdmin, &member, domain.ErrForbidden},
		{"vai trò rỗng/lạ", member, domain.Role(""), &member, domain.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.CheckTimeLogActor(tt.actor, tt.role, pm, tt.assignee); !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, muốn %v", err, tt.want)
			}
		})
	}
}
