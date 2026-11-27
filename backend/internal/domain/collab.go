package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// CF-07 — công việc con, bình luận, thông báo trong hệ thống.
const (
	CommentMaxRunes = 2000
	// NotificationExcerptRunes: độ dài trích đoạn bình luận / lý do lưu kèm thông báo.
	NotificationExcerptRunes = 200
)

// CheckSubtaskActor: ai được tạo / chuyển / xóa công việc con.
//   - Manager phụ trách dự án luôn được.
//   - Người nhận của việc (cha khi tạo, con khi chuyển/xóa) được — "Member tự tạo, tự chuyển" (§5.4).
//   - Member khác không đụng việc của người khác (§5.3). CEO chỉ xem (§5.2), Admin không tham gia dự án.
//
// Kiểm "có xem được dự án không" làm trước ở application (trả 404 chống dò ID).
func CheckSubtaskActor(actorID uuid.UUID, role Role, managerID uuid.UUID, assigneeID *uuid.UUID) error {
	if role != RoleManager && role != RoleMember {
		return ErrForbidden
	}
	if actorID == managerID {
		return nil
	}
	if assigneeID == nil || *assigneeID != actorID {
		return ErrSubtaskNotAllowed
	}
	return nil
}

// CheckSubtaskParent: con chỉ 1 cấp — cha phải là công việc gốc. CSDL chốt lần nữa bằng trigger.
func CheckSubtaskParent(parent Task) error {
	if parent.ParentID != nil {
		return ErrSubtaskParentNotRoot
	}
	return nil
}

// Comment là một bình luận trên công việc.
type Comment struct {
	ID         uuid.UUID
	TaskID     uuid.UUID
	AuthorID   uuid.UUID
	AuthorName string
	Body       string
	CreatedAt  time.Time
}

// NormalizeComment cắt khoảng trắng hai đầu, giữ xuống dòng; 1–2000 ký tự.
func NormalizeComment(s string) (string, error) {
	b := strings.TrimSpace(s)
	if b == "" || utf8.RuneCountInString(b) > CommentMaxRunes {
		return "", ErrInvalidComment
	}
	return b, nil
}

// CanComment: chỉ người làm việc trong dự án (Manager, Member) được viết bình luận; CEO/Admin chỉ đọc.
func CanComment(role Role) error {
	if role != RoleManager && role != RoleMember {
		return ErrForbidden
	}
	return nil
}

// NotificationType là loại thông báo (khớp ck_notification_type).
type NotificationType string

const (
	NotificationTaskAssigned   NotificationType = "task_assigned"
	NotificationTaskCommented  NotificationType = "task_commented"
	NotificationReviewRejected NotificationType = "review_rejected"
)

// Valid: loại thông báo có trong danh sách cho phép.
func (t NotificationType) Valid() bool {
	switch t {
	case NotificationTaskAssigned, NotificationTaskCommented, NotificationReviewRejected:
		return true
	}
	return false
}

// Notification là một thông báo trong chuông của một người dùng.
type Notification struct {
	ID          uuid.UUID
	Type        NotificationType
	TaskID      uuid.UUID
	TaskTitle   string
	ProjectID   uuid.UUID
	ProjectName string
	ActorID     *uuid.UUID
	ActorName   string
	Message     string
	ReadAt      *time.Time
	CreatedAt   time.Time
}

// NewNotification là thông báo sắp ghi: cùng nội dung gửi cho nhiều người.
type NewNotification struct {
	UserIDs   []uuid.UUID
	Type      NotificationType
	TaskID    uuid.UUID
	ProjectID uuid.UUID
	ActorID   uuid.UUID
	Message   string
}

// NotificationRecipients chọn người nhận: bỏ nil, bỏ người gây ra (không tự báo cho mình), bỏ trùng.
// Giữ thứ tự xuất hiện để kết quả ổn định khi test. Luôn trả slice khác nil.
func NotificationRecipients(actorID uuid.UUID, candidates ...*uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(candidates))
	seen := map[uuid.UUID]bool{actorID: true}
	for _, c := range candidates {
		if c == nil || seen[*c] {
			continue
		}
		seen[*c] = true
		out = append(out, *c)
	}
	return out
}

// AssignmentRecipient: ai cần nhận thông báo "được giao việc" khi người nhận đổi từ old sang new.
// Chỉ khi có người nhận mới, khác người cũ và không phải chính người thao tác (vd. Manager tự giao, Member nhận việc).
func AssignmentRecipient(actorID uuid.UUID, old, new *uuid.UUID) *uuid.UUID {
	if new == nil || *new == actorID || (old != nil && *old == *new) {
		return nil
	}
	return new
}

// NotificationExcerpt rút gọn bình luận / lý do thành một dòng tối đa 200 ký tự (thêm "…" nếu bị cắt).
func NotificationExcerpt(s string) string {
	c := collapse(s)
	if utf8.RuneCountInString(c) <= NotificationExcerptRunes {
		return c
	}
	return string([]rune(c)[:NotificationExcerptRunes]) + "…"
}
