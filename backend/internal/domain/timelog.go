package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Giới hạn Log time (CF-08). FE dùng cùng số trong features/tasks/model/types.ts.
const (
	TimeLogMaxMinutes   = 1440 // một lần ghi tối đa 24 giờ
	TimeLogNoteMaxRunes = 500
	// TimeLogDoneWindow: việc đã Done quá khoảng này thì không ghi/xóa giờ nữa (CLAUDE.md §5.7).
	TimeLogDoneWindow = 24 * time.Hour
)

// vnZone là múi giờ hiển thị (B10). Việt Nam không có giờ mùa hè nên dùng múi cố định UTC+7,
// tránh phụ thuộc tzdata của máy chạy.
var vnZone = time.FixedZone("ICT", 7*3600)

// TimeLog là một lần ghi giờ thực tế trên một công việc (cha hoặc con).
type TimeLog struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	TaskTitle string
	ParentID  *uuid.UUID // khác nil → log ghi trên việc con
	ProjectID uuid.UUID
	UserID    uuid.UUID
	UserName  string
	Minutes   int
	Note      string
	WorkDate  time.Time // ngày làm việc (chỉ phần ngày, UTC 00:00)
	CreatedAt time.Time
}

// NormalizeTimeLog kiểm số phút (1–1440, không âm, không 0) và mô tả ngắn bắt buộc (1–500 ký tự).
func NormalizeTimeLog(minutes int, note string) (string, error) {
	if minutes < 1 || minutes > TimeLogMaxMinutes {
		return "", ErrInvalidLogMinutes
	}
	n := strings.TrimSpace(note)
	if n == "" || utf8.RuneCountInString(n) > TimeLogNoteMaxRunes {
		return "", ErrInvalidLogNote
	}
	return n, nil
}

// CheckLogWindow: việc đang ở nhóm done mà đã Done quá 24 giờ thì khóa giờ.
// Vì sao: báo cáo hiệu suất (độ lệch giờ) chốt theo việc đã xong; cho ghi thêm mãi về sau
// sẽ làm số liệu thay đổi sau nghiệm thu. Đúng 24 giờ vẫn được ghi.
// Việc bị kéo ngược khỏi Done (từ chối nghiệm thu) không còn ở nhóm done → mở lại.
func CheckLogWindow(group StatusGroup, doneAt, now time.Time) error {
	if group != GroupDone {
		return nil
	}
	if now.Sub(doneAt) > TimeLogDoneWindow {
		return ErrTimeLogClosed
	}
	return nil
}

// ResolveWorkDate trả ngày làm việc: không gửi → hôm nay theo giờ Việt Nam.
// Không được là ngày tương lai, cũng không trước ngày tạo công việc (đều theo giờ Việt Nam).
func ResolveWorkDate(in *time.Time, taskCreatedAt, now time.Time) (time.Time, error) {
	today := vnDate(now)
	if in == nil {
		return today, nil
	}
	d := time.Date(in.Year(), in.Month(), in.Day(), 0, 0, 0, 0, time.UTC)
	if d.After(today) || d.Before(vnDate(taskCreatedAt)) {
		return time.Time{}, ErrInvalidLogDate
	}
	return d, nil
}

// vnDate lấy ngày lịch theo giờ Việt Nam của một thời điểm, trả về dạng UTC 00:00 để so sánh.
func vnDate(t time.Time) time.Time {
	v := t.In(vnZone)
	return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC)
}

// CheckTimeLogActor: ai được ghi giờ lên một công việc.
//   - Người nhận việc (với việc con: người nhận của VIỆC CHA, cùng căn cứ với CF-07).
//   - Manager phụ trách dự án.
//   - CEO chỉ xem (§5.2), Admin không tham gia dự án → ErrForbidden.
func CheckTimeLogActor(actorID uuid.UUID, role Role, managerID uuid.UUID, assigneeID *uuid.UUID) error {
	if role != RoleManager && role != RoleMember {
		return ErrForbidden
	}
	if actorID == managerID {
		return nil
	}
	if assigneeID == nil || *assigneeID != actorID {
		return ErrTimeLogNotAllowed
	}
	return nil
}

// CheckTimeLogDelete: chỉ người đã ghi mới xóa được dòng log của mình (sửa sai khi nhập nhầm).
func CheckTimeLogDelete(actorID uuid.UUID, l TimeLog) error {
	if l.UserID != actorID {
		return ErrTimeLogNotAllowed
	}
	return nil
}
