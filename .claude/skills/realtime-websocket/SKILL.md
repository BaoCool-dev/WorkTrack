---
name: realtime-websocket
description: Cập nhật tức thời trên Kanban WorkTrack bằng WebSocket (coder/websocket ở Go, client ở React). Dùng khi làm hub/room theo dự án, xác thực kết nối, định dạng sự kiện, tự kết nối lại, hoặc đo độ trễ đồng bộ (KPI 1, NFR-05).
---

# Realtime WebSocket

## Thiết kế
- Endpoint: `GET /api/v1/projects/{projectID}/ws`.
- **Xác thực lúc nâng cấp kết nối:** kiểm tra JWT (ticket ngắn hạn hoặc cookie) **và** quyền thành viên dự án trước khi `websocket.Accept`. Không có quyền → 403, không nâng cấp (D16/D19).
- Hub theo dự án (room = projectID). Use case sau khi ghi CSDL thành công gọi port `EventPublisher.Publish(ctx, projectID, event)`; adapter `infrastructure/realtime` phát tới room.
- Server chỉ **đẩy**; mọi thay đổi vẫn đi qua REST. Client không gửi lệnh nghiệp vụ qua WebSocket.
- Mỗi kết nối có ping định kỳ và giới hạn kích thước message; client chậm → đóng kết nối thay vì chặn hub.

## Định dạng sự kiện

```json
{ "type": "task.moved", "projectId": "…", "version": 42, "sentAt": "2026-11-20T03:00:00Z",
  "payload": { "taskId": "…", "fromColumnId": "…", "toColumnId": "…", "position": 3, "actorId": "…" } }
```
Loại tối thiểu: `task.created`, `task.updated`, `task.moved`, `comment.created`, `presence.changed`, `notification.created`.
`version` tăng dần theo dự án để client bỏ sự kiện cũ/trùng.

## Client
- Một hook `useProjectSocket(projectId)`; cập nhật cache TanStack Query bằng `setQueryData`.
- Tự kết nối lại với backoff có giới hạn; khi kết nối lại → `invalidateQueries` để đồng bộ lại (M14: mất kết nối tự nối lại).
- Hiển thị chỉ báo "đang cập nhật tức thời" + avatar người cùng xem (`presence.changed`).

## Kiểm thử
- Test Go: `httptest.NewServer` + client `websocket.Dial`; ca từ chối khi không phải thành viên.
- Đo KPI 1: script mở 2 client, kéo thẻ ở A, đo thời điểm B nhận; ≥ 30 lần, báo p95 ≤ 2 giây. Lưu log đo vào `docs/testing/evidence/`.

Kiểm tra API `coder/websocket` (Accept, Dial, CloseRead, wsjson) bằng Context7 trước khi viết — không đoán chữ ký hàm.
