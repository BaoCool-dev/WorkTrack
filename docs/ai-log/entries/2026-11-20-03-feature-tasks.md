---
title: feature tasks
date: 2026-11-20
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-06
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác
     Thêm dòng "commit: none" ở phần đầu nếu việc này không tạo commit (vd. thiết kế, soạn báo cáo ngoài repo). -->

## Mục tiêu
Làm CF-06 (công việc cha) theo lát dọc ở chế độ tự chạy: Manager tạo/sửa/giao việc, Member tự nhận việc ở nhóm todo, Story Point quy đổi giờ theo hệ số B1 cấu hình được, hạn chót, nhãn, API danh sách (lọc, phân trang) + chi tiết, panel chi tiết và trang "Công việc của tôi". Không làm Kanban/kéo thả/chuyển cột (feature/kanban do người dùng tự code).

## Prompt đã dùng
<!-- Dán NGUYÊN VĂN prompt chính (lấy từ: python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit).
     Mỗi prompt một dòng bắt đầu bằng ">". Dài quá thì rút gọn và ghi [rút gọn]. Không dán secret. -->
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: feature/tasks (mọc từ feature/projects). PR sẽ tạo với --base feature/projects --draft. [rút gọn]
> Phạm vi nhánh này: CF-06. Công việc cha: Manager tạo, sửa, giao việc; Member tự nhận việc từ Backlog (giới hạn WIP là Mở rộng — không chặn, chỉ hiển thị số đếm); Story Point và giờ ước tính quy đổi theo B1 (tham số cấu hình); hạn chót; nhãn; API danh sách công việc theo dự án (có lọc, phân trang) và chi tiết; panel chi tiết công việc (không gồm công việc con, bình luận, Log time — các nhánh sau); trang 'Công việc của tôi' dạng bảng. KHÔNG làm bảng Kanban, kéo thả, chuyển cột hay realtime và KHÔNG tạo API chuyển cột — tất cả thuộc feature/kanban do người dùng tự code. Công việc mới tạo nằm ở cột đầu tiên có status_group=todo. Mở rộng make seed với công việc mẫu.
> Quy trình bắt buộc: implement-ticket theo lát dọc, Context7, commit nhỏ qua ai-usage-log, feature-gate tối đa 3 vòng, báo cáo docs/gates/feature-tasks.md, push + PR draft nếu ĐẠT. Cuối cùng in đúng một dòng GATE=ĐẠT hoặc GATE=CHƯA_ĐẠT. [rút gọn]

## Nội dung AI tạo ra
<!-- Liệt kê theo file/phần: AI viết gì. Có thể kèm git diff --stat của commit AI. -->
- Commit 1 — CSDL + domain: migration `create_task_tables` (system_setting với hệ số B1 mặc định 4 giờ/SP, label + unique lower(name), task với FK kép cột/cha cùng dự án, CHECK Story Point chỉ ở cha, task_label, view `task_view`); query sqlc `task.sql` (danh sách theo dự án có lọc + phân trang, "của tôi", WIP, khóa dòng, cột todo đầu tiên, tạo/sửa/giao, nhãn), `label.sql`, `system_setting.sql`; `board_column.sql` đếm `task_count` thật; `domain/task.go` (chuẩn hóa tiêu đề/mô tả, thang SP Fibonacci, quy đổi phút, hệ số 1–40, bỏ trùng nhãn ≤ 10, điều kiện được nhận việc) + `task_test.go` viết trước; lỗi CF-06 trong `domain/errors.go`.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Query `ListMyTasks`/`CountWIP` sinh tham số `user_id` kiểu `uuid.NullUUID` | AI so sánh `@user_id` với cột nullable `assignee_id` mà không ép kiểu, sqlc suy ra kiểu nullable | Đọc code sinh sau `make sqlc` | Ép `@user_id::uuid`, chạy lại `make sqlc` |

## Kiểm chứng
<!-- Lệnh đã chạy và kết quả thật: make lint, make test, test nào, kiểm thử tay luồng nào. -->
- Commit 1: `make sqlc` → ok; `make migrate-up` trên Postgres Homebrew (B12) → OK `20261120095746_create_task_tables.sql`; `go test ./internal/domain/` → ok; `golangci-lint run ./internal/domain/...` → 0 issues.
