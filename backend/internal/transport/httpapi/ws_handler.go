package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/BaoCool-dev/WorkTrack/backend/internal/domain"
)

// BoardHub là phòng realtime theo dự án (cài đặt: realtime.Hub).
// Khai báo ở nơi dùng để tầng transport không import infrastructure.
type BoardHub interface {
	// Join trả kênh nhận sự kiện đã mã hóa JSON và hàm rời phòng. Kênh bị đóng khi client quá chậm.
	Join(projectID, userID uuid.UUID) (<-chan []byte, func())
}

const (
	wsPingEvery    = 25 * time.Second // giữ kết nối qua proxy/NAT và phát hiện client chết
	wsWriteTimeout = 5 * time.Second
	wsReadLimit    = 512 // client không gửi dữ liệu nghiệp vụ, chỉ khung điều khiển
)

type wsHandler struct {
	hub      BoardHub
	projects ProjectService
	tokens   TokenVerifier
	now      func() time.Time
	origins  []string
}

// wsOriginPatterns: chỉ chấp nhận origin của frontend (cùng giá trị CORS). Rỗng → thư viện chỉ cho cùng host.
func wsOriginPatterns(corsOrigin string) []string {
	if corsOrigin == "" {
		return nil
	}
	u, err := url.Parse(corsOrigin)
	if err != nil || u.Host == "" {
		return nil
	}
	return []string{u.Host}
}

// serve mở kênh realtime của bảng Kanban một dự án.
//
// Vì sao token đi qua query (?access_token=): WebSocket của trình duyệt không cho đặt header Authorization.
// Mọi kiểm tra (token, quyền xem dự án) làm TRƯỚC khi nâng cấp kết nối, để lỗi trả về là HTTP 401/404
// bình thường và người không thuộc dự án không bao giờ nhận được sự kiện của dự án đó.
func (h *wsHandler) serve(w http.ResponseWriter, r *http.Request) {
	actor, err := h.tokens.Verify(r.URL.Query().Get("access_token"), h.now())
	if err != nil {
		writeError(w, r, domain.ErrUnauthenticated)
		return
	}
	pid, err := pathUUID(r, "projectID", errProjectIDInvalid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if _, err := h.projects.Get(r.Context(), actor, pid); err != nil {
		writeError(w, r, err) // không phải thành viên → 404 như API dự án (không lộ dự án tồn tại)
		return
	}

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: h.origins})
	if err != nil {
		return // Accept đã tự trả lỗi HTTP (sai origin, không phải yêu cầu WebSocket…)
	}
	defer func() { _ = c.CloseNow() }() // đã đóng bình thường thì lỗi "already closed" là vô hại
	c.SetReadLimit(wsReadLimit)
	// Kênh chỉ chiều server → client. CloseRead đọc và bỏ khung từ client (xử lý ping/pong/close)
	// và hủy ctx khi client ngắt.
	ctx := c.CloseRead(r.Context())

	msgs, leave := h.hub.Join(pid, actor.UserID)
	defer leave()

	if err := pump(ctx, c, msgs); err != nil && !isNormalClose(err) {
		slog.DebugContext(r.Context(), "ws closed", "project_id", pid, "err", err)
	}
}

// pump đẩy sự kiện xuống client và ping định kỳ cho tới khi client ngắt hoặc bị hub loại vì chậm.
func pump(ctx context.Context, c *websocket.Conn, msgs <-chan []byte) error {
	ping := time.NewTicker(wsPingEvery)
	defer ping.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-msgs:
			if !ok {
				// Hub đóng kênh khi bộ đệm đầy: client tự kết nối lại và tải lại bảng (không mất trạng thái).
				return c.Close(websocket.StatusPolicyViolation, "client quá chậm")
			}
			if err := write(ctx, c, msg); err != nil {
				return err
			}
		case <-ping.C:
			pctx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
			err := c.Ping(pctx)
			cancel()
			if err != nil {
				return err
			}
		}
	}
}

func write(ctx context.Context, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, wsWriteTimeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageText, msg)
}

func isNormalClose(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	s := websocket.CloseStatus(err)
	return s == websocket.StatusNormalClosure || s == websocket.StatusGoingAway
}
