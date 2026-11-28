package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func col(pos int, g domain.StatusGroup) domain.BoardColumn {
	return domain.BoardColumn{ID: uuid.New(), Position: pos, StatusGroup: g}
}

func TestReviewTargets_CF09(t *testing.T) {
	backlog := col(0, domain.GroupTodo)
	doing := col(2, domain.GroupInProgress)
	review := col(3, domain.GroupInProgress)
	done := col(4, domain.GroupDone)
	archived := col(5, domain.GroupDone)

	t.Run("bảng mặc định: cột cuối nhóm in_progress là cột chờ nghiệm thu", func(t *testing.T) {
		// Truyền lộn thứ tự để chắc hàm dựa vào Position, không dựa vào thứ tự slice.
		got, err := domain.ReviewTargets([]domain.BoardColumn{archived, review, backlog, done, doing})
		if err != nil {
			t.Fatalf("lỗi: %v", err)
		}
		if got.Review != review.ID || got.Approve != done.ID || got.Reject != doing.ID {
			t.Fatalf("sai cột: %+v", got)
		}
	})

	tests := []struct {
		name string
		cols []domain.BoardColumn
	}{
		{"chỉ một cột in_progress → không có bước nghiệm thu", []domain.BoardColumn{backlog, doing, done}},
		{"không có cột in_progress", []domain.BoardColumn{backlog, done}},
		{"không có cột done", []domain.BoardColumn{backlog, doing, review}},
		{"bảng rỗng", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := domain.ReviewTargets(tc.cols); !errors.Is(err, domain.ErrTaskNotInReview) {
				t.Fatalf("muốn ErrTaskNotInReview, got %v", err)
			}
		})
	}
}

func TestNormalizeReviewReason_CF09(t *testing.T) {
	tests := []struct {
		name, in, want string
		err            error
	}{
		{"lý do hợp lệ, cắt khoảng trắng đầu cuối", "  Thiếu test đăng nhập  ", "Thiếu test đăng nhập", nil},
		{"giữ xuống dòng bên trong", "Dòng 1\nDòng 2", "Dòng 1\nDòng 2", nil},
		{"đúng 1 ký tự (biên dưới)", "x", "x", nil},
		{"đúng 500 ký tự (biên trên)", strings.Repeat("ạ", 500), strings.Repeat("ạ", 500), nil},
		{"rỗng", "", "", domain.ErrInvalidReviewReason},
		{"chỉ khoảng trắng", " \n\t ", "", domain.ErrInvalidReviewReason},
		{"501 ký tự (vượt biên)", strings.Repeat("a", 501), "", domain.ErrInvalidReviewReason},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := domain.NormalizeReviewReason(tc.in)
			if !errors.Is(err, tc.err) {
				t.Fatalf("err = %v, muốn %v", err, tc.err)
			}
			if got != tc.want {
				t.Fatalf("got %q, muốn %q", got, tc.want)
			}
		})
	}
}

func TestCanReview_CF09(t *testing.T) {
	pm, other := uuid.New(), uuid.New()
	tests := []struct {
		name  string
		actor uuid.UUID
		role  domain.Role
		err   error
	}{
		{"Manager phụ trách", pm, domain.RoleManager, nil},
		{"Manager khác (thành viên dự án)", other, domain.RoleManager, domain.ErrForbidden},
		{"Member", other, domain.RoleMember, domain.ErrForbidden},
		{"CEO chỉ xem", other, domain.RoleCEO, domain.ErrForbidden},
		{"Admin", other, domain.RoleAdmin, domain.ErrForbidden},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := domain.CanReview(tc.actor, tc.role, pm); !errors.Is(err, tc.err) {
				t.Fatalf("err = %v, muốn %v", err, tc.err)
			}
		})
	}
}
