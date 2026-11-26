package domain_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func TestValidateWIPLimit_CF05(t *testing.T) {
	for _, n := range []int{1, 5, 20} {
		if err := domain.ValidateWIPLimit(n); err != nil {
			t.Fatalf("WIP %d phải hợp lệ: %v", n, err)
		}
	}
	for _, n := range []int{-1, 0, 21, 100} {
		if err := domain.ValidateWIPLimit(n); !errors.Is(err, domain.ErrInvalidWIPLimit) {
			t.Fatalf("WIP %d phải lỗi, got %v", n, err)
		}
	}
}

func TestBoardRole_CF04(t *testing.T) {
	pm, other, member := uuid.New(), uuid.New(), uuid.New()
	mine := &member
	tests := []struct {
		name     string
		actor    uuid.UUID
		role     domain.Role
		assignee *uuid.UUID
		want     domain.TransitionRole
		err      error
	}{
		{"Manager phụ trách kéo mọi thẻ", pm, domain.RoleManager, nil, domain.TransitionManager, nil},
		{"Manager phụ trách kéo thẻ của người khác", pm, domain.RoleManager, mine, domain.TransitionManager, nil},
		{"Member kéo thẻ của mình", member, domain.RoleMember, mine, domain.TransitionMember, nil},
		{"Member kéo thẻ chưa có người nhận", member, domain.RoleMember, nil, "", domain.ErrTaskNotAssignee},
		{"Member kéo thẻ của người khác", other, domain.RoleMember, mine, "", domain.ErrTaskNotAssignee},
		{"Manager không phụ trách dùng luật member", member, domain.RoleManager, mine, domain.TransitionMember, nil},
		{"CEO chỉ xem", other, domain.RoleCEO, nil, "", domain.ErrForbidden},
		{"Admin không thao tác bảng", other, domain.RoleAdmin, nil, "", domain.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.BoardRole(tt.actor, tt.role, pm, tt.assignee)
			if got != tt.want || !errors.Is(err, tt.err) {
				t.Fatalf("BoardRole = %q, %v; want %q, %v", got, err, tt.want, tt.err)
			}
		})
	}
}

func TestCheckMove_CF04(t *testing.T) {
	const (
		todo = domain.GroupTodo
		prog = domain.GroupInProgress
		done = domain.GroupDone
	)
	tests := []struct {
		name string
		mv   domain.Move
		err  error
	}{
		{"đổi thứ tự cùng cột không cần luật", domain.Move{Role: domain.TransitionMember, From: done, To: done, SameColumn: true}, nil},
		{"Member todo → in_progress có luật", domain.Move{Role: domain.TransitionMember, From: todo, To: prog, Allowed: true}, nil},
		{"Member không có luật", domain.Move{Role: domain.TransitionMember, From: todo, To: prog}, domain.ErrTransitionNotAllowed},
		{"Member vào nhóm done", domain.Move{Role: domain.TransitionMember, From: prog, To: done, Allowed: true}, domain.ErrMemberTransitionToDone},
		{"Member ra khỏi nhóm done", domain.Move{Role: domain.TransitionMember, From: done, To: prog, Allowed: true}, domain.ErrMemberTransitionToDone},
		{"Manager vào nhóm done", domain.Move{Role: domain.TransitionManager, From: prog, To: done, Allowed: true}, nil},
		{"Manager không có luật", domain.Move{Role: domain.TransitionManager, From: prog, To: done}, domain.ErrTransitionNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := domain.CheckMove(tt.mv); !errors.Is(err, tt.err) {
				t.Fatalf("CheckMove = %v; want %v", err, tt.err)
			}
		})
	}
}

func TestEntersWIP_CF05(t *testing.T) {
	tests := []struct {
		from, to domain.StatusGroup
		want     bool
	}{
		{domain.GroupTodo, domain.GroupInProgress, true},
		{domain.GroupDone, domain.GroupInProgress, true},
		{domain.GroupInProgress, domain.GroupInProgress, false}, // In Progress → Chờ nghiệm thu: vẫn 1 việc
		{domain.GroupInProgress, domain.GroupDone, false},
		{domain.GroupTodo, domain.GroupTodo, false},
	}
	for _, tt := range tests {
		if got := domain.EntersWIP(tt.from, tt.to); got != tt.want {
			t.Fatalf("EntersWIP(%s, %s) = %v; want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestCheckWIP_CF05(t *testing.T) {
	if err := domain.CheckWIP(4, 5); err != nil {
		t.Fatalf("4/5 phải còn chỗ: %v", err)
	}
	if err := domain.CheckWIP(5, 5); !errors.Is(err, domain.ErrWIPLimitReached) {
		t.Fatalf("5/5 phải chặn, got %v", err)
	}
	if err := domain.CheckWIP(7, 5); !errors.Is(err, domain.ErrWIPLimitReached) {
		t.Fatalf("7/5 (giảm giới hạn sau) phải chặn, got %v", err)
	}
}

func TestInsertAt_CF04(t *testing.T) {
	a, b, c, x := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name string
		ids  []uuid.UUID
		pos  int
		want []uuid.UUID
	}{
		{"chèn đầu", []uuid.UUID{a, b}, 0, []uuid.UUID{x, a, b}},
		{"chèn giữa", []uuid.UUID{a, b}, 1, []uuid.UUID{a, x, b}},
		{"vị trí quá lớn → cuối", []uuid.UUID{a, b}, 99, []uuid.UUID{a, b, x}},
		{"vị trí âm → đầu", []uuid.UUID{a}, -3, []uuid.UUID{x, a}},
		{"cột rỗng", nil, 0, []uuid.UUID{x}},
		{"đổi thứ tự cùng cột: bỏ chỗ cũ trước", []uuid.UUID{x, a, b, c}, 2, []uuid.UUID{a, b, x, c}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.InsertAt(tt.ids, x, tt.pos)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v; want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v; want %v", got, tt.want)
				}
			}
		})
	}
}
