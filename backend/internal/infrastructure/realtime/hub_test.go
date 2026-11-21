package realtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

func recv(t *testing.T, ch <-chan []byte) wireEvent {
	t.Helper()
	select {
	case b, ok := <-ch:
		if !ok {
			t.Fatal("channel closed")
		}
		var w wireEvent
		if err := json.Unmarshal(b, &w); err != nil {
			t.Fatal(err)
		}
		return w
	default:
		t.Fatal("no message")
	}
	return wireEvent{}
}

func TestHub_PublishOnlyToSameProject(t *testing.T) {
	h := NewHub()
	p1, p2, u1, u2 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	a, leaveA := h.Join(p1, u1)
	defer leaveA()
	b, leaveB := h.Join(p2, u2)
	defer leaveB()
	recv(t, a) // presence của chính mình
	recv(t, b)

	task := uuid.New()
	h.Publish(context.Background(), domain.BoardEvent{Type: domain.EventTaskMoved, ProjectID: p1, TaskID: task, Position: 0, Version: 3})
	got := recv(t, a)
	if got.Type != domain.EventTaskMoved || *got.TaskID != task || got.Position == nil || *got.Position != 0 || *got.Version != 3 {
		t.Fatalf("bad event %+v", got)
	}
	select {
	case m := <-b:
		t.Fatalf("project 2 must not receive project 1 events: %s", m)
	default:
	}
}

func TestHub_PresenceJoinLeave(t *testing.T) {
	h := NewHub()
	p, u1, u2 := uuid.New(), uuid.New(), uuid.New()
	a, leaveA := h.Join(p, u1)
	recv(t, a)
	_, leaveB := h.Join(p, u2)
	if got := recv(t, a); got.Type != domain.EventPresence || len(got.UserIDs) != 2 {
		t.Fatalf("want 2 online, got %+v", got)
	}
	_, leaveB2 := h.Join(p, u2) // tab thứ hai của cùng người → vẫn 2 người
	if got := recv(t, a); len(got.UserIDs) != 2 {
		t.Fatalf("same user twice should count once, got %+v", got.UserIDs)
	}
	leaveB()
	leaveB() // gọi lại an toàn
	recv(t, a)
	leaveB2()
	if got := recv(t, a); len(got.UserIDs) != 1 || got.UserIDs[0] != u1 {
		t.Fatalf("want only u1, got %+v", got.UserIDs)
	}
	leaveA()
	if len(h.rooms) != 0 {
		t.Fatal("empty room should be removed")
	}
}

func TestHub_SlowSubscriberIsDropped(t *testing.T) {
	h := NewHub()
	p := uuid.New()
	slow, leave := h.Join(p, uuid.New())
	defer leave()
	for range SendBuffer + 1 {
		h.Publish(context.Background(), domain.BoardEvent{Type: domain.EventTaskUpdated, ProjectID: p, TaskID: uuid.New()})
	}
	n := 0
	for range slow {
		n++
	}
	if n != SendBuffer {
		t.Fatalf("want %d buffered then closed, got %d", SendBuffer, n)
	}
}
