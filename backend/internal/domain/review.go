package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// CF-09 — nghiệm thu công việc cha. Quy tắc thuần, không I/O.

// ReviewReasonMaxRunes: lý do từ chối tối đa 500 ký tự (khớp ck_task_review_reason).
const ReviewReasonMaxRunes = 500

// ReviewDecision là kết quả một lần nghiệm thu (khớp ck_task_review_decision).
type ReviewDecision string

const (
	ReviewApproved ReviewDecision = "approved"
	ReviewRejected ReviewDecision = "rejected"
)

// TaskReview là một dòng lịch sử nghiệm thu hiển thị trong chi tiết công việc.
type TaskReview struct {
	ID           uuid.UUID
	TaskID       uuid.UUID
	ReviewerID   uuid.UUID
	ReviewerName string
	Decision     ReviewDecision
	Reason       *string // chỉ có khi Từ chối
	CreatedAt    time.Time
}

// ReviewColumns là ba cột tham gia nghiệm thu của một bảng.
type ReviewColumns struct {
	Review  uuid.UUID // cột "Chờ nghiệm thu"
	Approve uuid.UUID // Duyệt → cột này (nhóm done)
	Reject  uuid.UUID // Từ chối → cột này (nhóm in_progress, "Đang làm")
}

// ReviewTargets tìm các cột nghiệm thu từ cấu hình bảng (decisions.md mục C, CF-09).
//
// Vì sao suy ra từ vị trí thay vì đánh dấu cột: bảng chỉ lưu tên + nhóm trạng thái (CF-03), mà tên do
// Manager tự đặt. "Chờ nghiệm thu" thuộc nhóm in_progress (CLAUDE.md §5.5) và đứng ngay trước Done
// trong cấu hình mặc định, nên:
//   - Cột chờ nghiệm thu = cột in_progress có vị trí lớn nhất, chỉ khi bảng có ít nhất 2 cột in_progress
//     (bảng chỉ có một cột "đang làm" thì không có bước nghiệm thu).
//   - Duyệt → cột done đứng đầu.
//   - Từ chối → cột in_progress đứng đầu (khác cột chờ nghiệm thu).
//
// Bảng không đủ điều kiện → ErrTaskNotInReview: không công việc nào đang "chờ nghiệm thu".
func ReviewTargets(cols []BoardColumn) (ReviewColumns, error) {
	var first, last, done *BoardColumn
	inProgress := 0
	for i := range cols {
		c := &cols[i]
		switch c.StatusGroup {
		case GroupInProgress:
			inProgress++
			if first == nil || c.Position < first.Position {
				first = c
			}
			if last == nil || c.Position > last.Position {
				last = c
			}
		case GroupDone:
			if done == nil || c.Position < done.Position {
				done = c
			}
		case GroupTodo:
		}
	}
	if inProgress < 2 || done == nil {
		return ReviewColumns{}, ErrTaskNotInReview
	}
	return ReviewColumns{Review: last.ID, Approve: done.ID, Reject: first.ID}, nil
}

// NormalizeReviewReason cắt khoảng trắng đầu cuối, giữ xuống dòng bên trong; bắt buộc 1–500 ký tự.
func NormalizeReviewReason(s string) (string, error) {
	r := strings.TrimSpace(s)
	if r == "" || utf8.RuneCountInString(r) > ReviewReasonMaxRunes {
		return "", ErrInvalidReviewReason
	}
	return r, nil
}

// CanReview: chỉ Manager PHỤ TRÁCH dự án được Duyệt / Từ chối (CLAUDE.md §5.5).
// Manager khác chỉ là thành viên dự án, Member, CEO (chỉ xem), Admin → ErrForbidden.
func CanReview(actorID uuid.UUID, role Role, managerID uuid.UUID) error {
	if role != RoleManager || actorID != managerID {
		return ErrForbidden
	}
	return nil
}
