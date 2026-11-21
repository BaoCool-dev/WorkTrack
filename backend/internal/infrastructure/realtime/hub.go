// Package realtime phát sự kiện bảng Kanban tới các client WebSocket theo từng dự án (CF-04).
package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// SendBuffer là số tin tối đa chờ gửi cho một client. Đầy = client quá chậm → ngắt kết nối,
// client tự kết nối lại và tải lại bảng. Vì sao: một client chậm không được làm chậm Publish của
// cả dự án (Publish chạy ngay trong request HTTP của người kéo thẻ).
const SendBuffer = 32

// Hub giữ các "phòng" theo dự án. Mỗi kết nối WebSocket là một subscriber trong phòng.
// Chỉ chạy trong một tiến trình API (đồ án 1 máy chủ); nhiều máy chủ thì cần pub/sub ngoài — chưa làm.
type Hub struct {
	mu    sync.Mutex
	rooms map[uuid.UUID]map[*subscriber]struct{}
	now   func() time.Time
}

type subscriber struct {
	userID uuid.UUID
	send   chan []byte
}

// NewHub tạo Hub rỗng.
func NewHub() *Hub {
	return &Hub{rooms: map[uuid.UUID]map[*subscriber]struct{}{}, now: time.Now}
}

// wireEvent là định dạng JSON gửi cho client. Trường rỗng bị lược để tin nhỏ gọn.
type wireEvent struct {
	Type         string      `json:"type"`
	ProjectID    uuid.UUID   `json:"projectId"`
	TaskID       *uuid.UUID  `json:"taskId,omitempty"`
	ActorID      *uuid.UUID  `json:"actorId,omitempty"`
	FromColumnID *uuid.UUID  `json:"fromColumnId,omitempty"`
	ToColumnID   *uuid.UUID  `json:"toColumnId,omitempty"`
	Position     *int        `json:"position,omitempty"`
	Version      *int        `json:"version,omitempty"`
	UserIDs      []uuid.UUID `json:"userIds,omitempty"`
	// SentAt: thời điểm máy chủ phát — dùng đo độ trễ đồng bộ (KPI 1).
	SentAt time.Time `json:"sentAt"`
}

func ptr[T comparable](v T) *T {
	var zero T
	if v == zero {
		return nil
	}
	return &v
}

// Publish cài đặt kanban.Publisher và task.Publisher. Không bao giờ chặn: gửi non-blocking.
func (h *Hub) Publish(ctx context.Context, e domain.BoardEvent) {
	w := wireEvent{
		Type: e.Type, ProjectID: e.ProjectID, TaskID: ptr(e.TaskID), ActorID: ptr(e.ActorID),
		FromColumnID: ptr(e.FromColumnID), ToColumnID: ptr(e.ToColumnID), Version: ptr(e.Version),
		SentAt: h.now().UTC(),
	}
	if e.Type == domain.EventTaskMoved {
		w.Position = &e.Position // vị trí 0 là hợp lệ → luôn gửi với task.moved
	}
	h.broadcast(ctx, e.ProjectID, w)
}

// Join thêm một subscriber vào phòng dự án và báo presence mới cho cả phòng.
// msgs bị đóng khi Hub ngắt subscriber chậm; leave() gọi khi kết nối đóng (gọi nhiều lần vẫn an toàn).
func (h *Hub) Join(projectID, userID uuid.UUID) (msgs <-chan []byte, leave func()) {
	s := &subscriber{userID: userID, send: make(chan []byte, SendBuffer)}
	h.mu.Lock()
	room := h.rooms[projectID]
	if room == nil {
		room = map[*subscriber]struct{}{}
		h.rooms[projectID] = room
	}
	room[s] = struct{}{}
	h.mu.Unlock()
	h.broadcastPresence(projectID)

	var once sync.Once
	return s.send, func() {
		once.Do(func() {
			if h.remove(projectID, s) {
				h.broadcastPresence(projectID)
			}
		})
	}
}

// remove bỏ subscriber khỏi phòng và đóng kênh; trả false nếu đã bị bỏ trước đó.
// Gọi khi đang KHÔNG giữ h.mu.
func (h *Hub) remove(projectID uuid.UUID, s *subscriber) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.removeLocked(projectID, s)
}

func (h *Hub) removeLocked(projectID uuid.UUID, s *subscriber) bool {
	room := h.rooms[projectID]
	if _, ok := room[s]; !ok {
		return false
	}
	delete(room, s)
	close(s.send)
	if len(room) == 0 {
		delete(h.rooms, projectID)
	}
	return true
}

// broadcastPresence gửi danh sách người đang mở bảng (mỗi người một lần dù mở nhiều tab).
func (h *Hub) broadcastPresence(projectID uuid.UUID) {
	h.mu.Lock()
	seen := map[uuid.UUID]struct{}{}
	for s := range h.rooms[projectID] {
		seen[s.userID] = struct{}{}
	}
	h.mu.Unlock()
	ids := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	h.broadcast(context.Background(), projectID, wireEvent{
		Type: domain.EventPresence, ProjectID: projectID, UserIDs: ids, SentAt: h.now().UTC(),
	})
}

func (h *Hub) broadcast(ctx context.Context, projectID uuid.UUID, w wireEvent) {
	msg, err := json.Marshal(w)
	if err != nil {
		slog.ErrorContext(ctx, "realtime marshal", "err", err)
		return
	}
	var dropped int
	h.mu.Lock()
	for s := range h.rooms[projectID] {
		select {
		case s.send <- msg:
		default:
			h.removeLocked(projectID, s) // client chậm: đóng kênh → handler đóng kết nối
			dropped++
		}
	}
	h.mu.Unlock()
	if dropped > 0 {
		slog.WarnContext(ctx, "realtime dropped slow subscribers", "project_id", projectID, "count", dropped)
		if w.Type != domain.EventPresence { // tránh đệ quy presence → presence
			h.broadcastPresence(projectID)
		}
	}
}
