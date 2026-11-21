---
title: feature kanban
date: 2026-11-21
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-04, CF-05, UC05
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác
     Thêm dòng "commit: none" ở phần đầu nếu việc này không tạo commit (vd. thiết kế, soạn báo cáo ngoài repo). -->

## Mục tiêu
Bảng Kanban kéo thả realtime (CF-04, UC05) và giới hạn WIP theo dự án (CF-05): API chuyển cột/đổi thứ tự theo luật `workflow_transition`, chống ghi đè bằng version, lịch sử chuyển cột, WebSocket theo dự án, giao diện dnd-kit, script đo KPI 1. Chế độ nhật ký tối giản (CLAUDE.md §7b).

## Prompt đã dùng
<!-- Dán NGUYÊN VĂN prompt chính (lấy từ: python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit).
     Mỗi prompt một dòng bắt đầu bằng ">". Dài quá thì rút gọn và ghi [rút gọn]. Không dán secret. -->
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. [rút gọn]
> Phạm vi nhánh này: CF-04, CF-05. Claude viết bản đầu, người dùng sẽ chỉnh sửa — code dễ đọc, comment giải thích vì sao ở chỗ quan trọng (CLAUDE.md §7b). Backend: API di chuyển công việc cha giữa các cột và đổi thứ tự trong cột, áp dụng luật workflow_transition theo vai trò (dữ liệu đã có từ feature/projects); Member không đưa việc vào hoặc ra nhóm done; giới hạn WIP theo dự án (mặc định 5, chỉ đếm việc cha ở nhóm in_progress, Manager chỉnh trong cấu hình dự án) áp dụng cả khi kéo vào cột in_progress và khi nhận việc; chống ghi đè khi hai người cùng kéo (version); lưu lịch sử chuyển cột. Realtime theo skill realtime-websocket (coder/websocket) […] Frontend: bảng Kanban dnd-kit theo design-system §6 […] Viết script đo KPI 1 (độ trễ đồng bộ giữa 2 client, tối thiểu 30 lần, p95) […] [rút gọn]

## Nội dung AI tạo ra
<!-- Liệt kê theo file/phần: AI viết gì. Có thể kèm git diff --stat của commit AI. -->
- **CSDL:** migration `20261120204233_add_kanban_wip_version_history.sql` (`project.wip_limit` 1–20 mặc định 5, `task.version`, bảng `task_move_history`, dựng lại `task_view`); query sqlc `CountProjectWIP`, `ListBoardTasks`, `SetTaskPositions`, `MoveTaskColumn` (WHERE version), `InsertTaskMoveHistory`, `TransitionAllowed`, `UpdateProjectWIPLimit`.
- **Domain:** `domain/kanban.go` — `BoardRole`, `CheckMove`, `EntersWIP`, `CheckWIP`, `InsertAt`, kiểu sự kiện; lỗi nghiệp vụ mới; test bảng.
- **Application:** gói `application/kanban` (Board, Move: khóa dự án → khóa việc → version → luật → WIP → ghi → lịch sử → phát sự kiện sau commit); `task.Service` kiểm WIP khi nhận việc/đổi người nhận và phát `task.created`/`task.updated`; `project.Service.SetWIPLimit`.
- **Infrastructure:** `postgres/kanban_repo.go`, `realtime/hub.go` (phòng theo dự án, presence, bỏ client chậm).
- **Transport:** `kanban_handler.go` (`GET /projects/{id}/board/tasks`, `POST /tasks/{id}/move`, `PUT /projects/{id}/wip-limit`), `ws_handler.go` (xác thực token + thành viên trước khi nâng cấp), ánh xạ lỗi, test API + WebSocket.
- **Frontend (`features/kanban/`):** model thuần (`dropRules` mô phỏng luật server để chặn có lý do trước khi gửi, `boardState` tính vị trí thả/WIP/bộ lọc), `useMoveTask` (cập nhật lạc quan + rollback), `useBoardDrag` (kiểm luật → gửi → toast "Hoàn tác" hoặc dải lỗi), `useProjectSocket` (áp sự kiện vào cache, tự kết nối lại với backoff, presence), các component `KanbanBoard`, `KanbanColumn`, `TaskCard`, `BoardToolbar`, `WipFullDialog`, `WipLimitControl`, `Banners`; tab "Kanban" trong chi tiết dự án; test Vitest/RTL/MSW (bảng theo 3 vai trò, hook kéo thả, socket).

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Thêm method `SetWIPLimit` vào interface `ProjectService` nhưng quên cập nhật fake trong test httpapi | Sửa interface mà không rà mọi chỗ cài đặt | `go vet` báo `*fakeProjects does not implement ProjectService` | Thêm `fakeProjects.SetWIPLimit` |
| Để lại test ranh giới của nhánh trước (`TestTaskAPI_NoColumnMoveEndpoint_CF04Boundary`) khẳng định không có route `/move` | Không rà test cũ mâu thuẫn với phạm vi mới | `go test` đỏ (400 thay vì 404) | Xóa test ranh giới, thay bằng test API Kanban |
| Thả thẻ xuống dưới trong cùng cột bị lệch một vị trí | Tính chỉ số chèn trên danh sách còn chứa chính thẻ đang kéo | Test `dropPosition` đỏ | Bỏ thẻ đang kéo khỏi danh sách trước khi tính, chèn sau thẻ đích khi kéo xuống |
| Thẻ đang nhấc dùng `shadow-overlay` | Không đối chiếu design-system §6 (bóng chỉ cho modal/dropdown/toast) | Rà design-system | Đổi sang `border-accent` |
| Test "Hoàn tác" chờ nút mà fixture không có luật quay lại c-2→c-1 | Quên rằng hook chỉ hứa hoàn tác khi đường quay lại hợp lệ | Test đỏ | Thêm luật quay lại vào test và test riêng trường hợp không có nút |
| Gọi `defer c.CloseNow()` và `websocket.Dial` không đóng `resp.Body` | Bỏ qua giá trị lỗi trả về | `golangci-lint` (errcheck, bodyclose) | Bọc `_ =` và đóng body |

## Kiểm chứng
<!-- Lệnh đã chạy và kết quả thật: make lint, make test, test nào, kiểm thử tay luồng nào. -->
- Backend: `go vet ./...`, `go vet -tags=integration ./...`, `golangci-lint run ./...` (0 issues), `go test ./...` xanh.
- Frontend: `tsc -b --noEmit`, `eslint src`, `prettier --check`, `vitest run` (21 file, 215 test) xanh; `make lint`, `make test` xanh.
