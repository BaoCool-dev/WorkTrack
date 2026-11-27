---
title: feature subtasks
date: 2026-11-27
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-07
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác
     Thêm dòng "commit: none" ở phần đầu nếu việc này không tạo commit (vd. thiết kế, soạn báo cáo ngoài repo). -->

## Mục tiêu
Làm CF-07: công việc con 1 cấp (Member tự tạo, tự chuyển, không Story Point), bình luận gắn công việc, thông báo trong hệ thống (được giao việc, có bình luận, bị từ chối nghiệm thu).

## Prompt đã dùng
<!-- Dán NGUYÊN VĂN prompt chính (lấy từ: python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit).
     Mỗi prompt một dòng bắt đầu bằng ">". Dài quá thì rút gọn và ghi [rút gọn]. Không dán secret. -->
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: feature/subtasks (mọc từ release/v1). PR sẽ tạo với --base release/v1 --draft. [...] Phạm vi nhánh này: CF-07. Công việc con 1 cấp do Member tạo (không Story Point, CHECK ở CSDL), Member tự chuyển trạng thái con; bình luận gắn công việc; thông báo trong hệ thống (chuông, đã đọc/chưa đọc) cho: được giao việc, có bình luận, bị từ chối nghiệm thu. [...] [rút gọn]

## Nội dung AI tạo ra
<!-- Liệt kê theo file/phần: AI viết gì. Có thể kèm git diff --stat của commit AI. -->
- CSDL: migration `20261127014848_add_subtask_comment_notification.sql` (trigger con 1 cấp, bảng `task_comment`, `notification`); query `subtask.sql`, `comment.sql`, `notification.sql`; sinh lại sqlcgen.
- Domain: `internal/domain/collab.go` (quyền thao tác việc con, cha phải là việc gốc, chuẩn hóa bình luận, chọn người nhận thông báo, trích đoạn) + `collab_test.go` viết trước (RED → GREEN); lỗi CF-07 trong `errors.go`.
- Application: gói `subtask`, `comment`, `notification` (port + service + test với fake store); `task.Service` gửi thông báo `task_assigned` khi tạo/đổi người nhận (cùng giao dịch); query `GetTaskComment`.
- Hạ tầng: `postgres/collab_repo.go` (SubtaskRepo, CommentRepo, NotificationRepo, ghi thông báo theo lô), `checkViolation` (23514) trong `convert.go`, test tích hợp `collab_integration_test.go` (chạy trong CI).
- HTTP: `httpapi/collab_handler.go` (route việc con, bình luận, `/me/notifications`), ánh xạ 4 lỗi CF-07 trong `errors.go`, `parentId` trong `taskResponse`, wiring `main.go`; `collab_api_test.go` viết trước với service giả.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|

## Kiểm chứng
<!-- Lệnh đã chạy và kết quả thật: make lint, make test, test nào, kiểm thử tay luồng nào. -->
