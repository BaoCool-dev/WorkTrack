package domain

import "github.com/google/uuid"

// CF-04 (kéo thả Kanban) và CF-05 (giới hạn WIP) — quy tắc thuần, không I/O.
// Tầng application nạp dữ liệu (nhóm cột, luật, số WIP) rồi hỏi các hàm dưới đây.

const (
	WIPLimitDefault = 5 // BRD: mặc định 5 việc cha đang làm mỗi người
	WIPLimitMin     = 1
	WIPLimitMax     = 20
)

// ValidateWIPLimit: giới hạn WIP của dự án từ 1 đến 20 (khớp CHECK ck_project_wip_limit).
func ValidateWIPLimit(n int) error {
	if n < WIPLimitMin || n > WIPLimitMax {
		return ErrInvalidWIPLimit
	}
	return nil
}

// BoardRole quyết định người kéo thẻ dùng bộ luật nào trong workflow_transition.
//
// Vì sao không dùng thẳng vai trò hệ thống: "manager" trong bảng luật nghĩa là Manager PHỤ TRÁCH dự án.
// Một Manager khác chỉ là thành viên dự án thì không có quyền quản lý bảng → dùng luật "member".
//   - Manager phụ trách: luật manager, kéo được mọi thẻ.
//   - Thành viên (Manager/Member): luật member, chỉ kéo thẻ đang giao cho chính mình
//     (Member không giao việc cho người khác — CLAUDE.md §5.3, nên cũng không đẩy việc của người khác).
//   - CEO chỉ xem (§5.2), Admin không tham gia dự án → cấm.
//
// Việc kiểm "có là thành viên dự án không" làm trước đó ở application (canView).
func BoardRole(actorID uuid.UUID, role Role, managerID uuid.UUID, assigneeID *uuid.UUID) (TransitionRole, error) {
	if role != RoleManager && role != RoleMember {
		return "", ErrForbidden
	}
	if actorID == managerID {
		return TransitionManager, nil
	}
	if assigneeID == nil || *assigneeID != actorID {
		return "", ErrTaskNotAssignee
	}
	return TransitionMember, nil
}

// Move mô tả một lần kéo thẻ đã nạp đủ dữ liệu để kiểm luật.
type Move struct {
	Role       TransitionRole
	From, To   StatusGroup
	SameColumn bool // thả lại trong cùng cột = chỉ đổi thứ tự
	Allowed    bool // có dòng (from, to, role) trong workflow_transition
}

// CheckMove áp dụng luật chuyển cột (CLAUDE.md §5.5).
// Thứ tự kiểm có chủ ý: lỗi "Member vào/ra nhóm done" báo trước lỗi "không có luật" để thông điệp
// nói đúng lý do nghiệp vụ, kể cả khi luật trong CSDL đã bị sửa tay.
func CheckMove(m Move) error {
	if m.SameColumn {
		return nil // đổi thứ tự trong cột không đổi trạng thái → không qua luật
	}
	if m.Role != TransitionManager && (m.From == GroupDone || m.To == GroupDone) {
		return ErrMemberTransitionToDone
	}
	if !m.Allowed {
		return ErrTransitionNotAllowed
	}
	return nil
}

// EntersWIP: việc chỉ làm tăng WIP khi đi TỪ ngoài nhóm in_progress VÀO nhóm in_progress.
// In Progress → Chờ nghiệm thu (cùng nhóm) vẫn là một việc đang làm, không đếm thêm.
func EntersWIP(from, to StatusGroup) bool {
	return to == GroupInProgress && from != GroupInProgress
}

// CheckWIP chặn khi người nhận đã có đủ (hoặc vượt, nếu Manager vừa giảm giới hạn) số việc đang làm.
func CheckWIP(current, limit int) error {
	if current >= limit {
		return ErrWIPLimitReached
	}
	return nil
}

// InsertAt trả thứ tự mới của một cột sau khi đặt id vào vị trí pos (0 = đầu cột).
// id có sẵn trong ids (đổi thứ tự cùng cột) sẽ được bỏ khỏi chỗ cũ trước; pos ngoài khoảng bị kẹp lại.
func InsertAt(ids []uuid.UUID, id uuid.UUID, pos int) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(ids)+1)
	for _, v := range ids {
		if v != id {
			out = append(out, v)
		}
	}
	pos = max(0, min(pos, len(out)))
	out = append(out, uuid.Nil)
	copy(out[pos+1:], out[pos:])
	out[pos] = id
	return out
}

// Loại sự kiện realtime trên bảng Kanban của một dự án.
const (
	EventTaskCreated = "task.created"
	EventTaskUpdated = "task.updated"
	EventTaskMoved   = "task.moved"
	EventPresence    = "presence"
)

// BoardEvent là một thay đổi trên bảng cần báo cho các client đang mở bảng của dự án.
// Application phát sau khi transaction commit; tầng realtime tự đổi sang định dạng gửi qua WebSocket.
// FromColumnID/ToColumnID/Position chỉ có ở task.moved.
type BoardEvent struct {
	Type         string
	ProjectID    uuid.UUID
	TaskID       uuid.UUID
	ActorID      uuid.UUID
	FromColumnID uuid.UUID
	ToColumnID   uuid.UUID
	Position     int
	Version      int
}
