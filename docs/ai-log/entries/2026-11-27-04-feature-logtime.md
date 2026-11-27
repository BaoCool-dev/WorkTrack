---
title: feature logtime
date: 2026-11-27
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-08
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác -->

## Mục tiêu
Làm CF-08: Log time trên công việc cha và con (phút > 0, không log vào việc đã Done quá 24 giờ), tổng giờ thực tế = giờ cha + giờ con, thanh so giờ trên thẻ (design-system §6, ngưỡng B5), modal Log giờ, lịch sử log.

## Prompt đã dùng
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: feature/logtime (mọc từ feature/subtasks). PR sẽ tạo với --base feature/subtasks --draft. [...] Phạm vi nhánh này: CF-08. Log time trên công việc cha và con (phút > 0, không log vào việc đã Done quá 24 giờ); tổng giờ thực tế = giờ cha + giờ con; thanh so giờ trên thẻ theo design-system §6 và ngưỡng B5; modal Log giờ; lịch sử log. [...] [rút gọn]

## Nội dung AI tạo ra
- CSDL: migration `20261127114535_add_time_log.sql` (bảng `time_log` với CHECK phút 1–1440 và mô tả 1–500 ký tự; `task_view` thêm `logged_minutes`, `actual_minutes` = cha + con, `subtask_count`, `subtask_done_count`); query `time_log.sql` (tạo, đọc, lịch sử gồm cả log của con, xóa, mốc Done cuối từ `task_move_history`); sinh lại sqlcgen.
- Domain: `internal/domain/timelog.go` + `timelog_test.go` viết trước (RED → GREEN): chuẩn hóa phút/mô tả, cửa sổ 24 giờ sau Done, ngày làm việc theo giờ Việt Nam, quyền ghi/xóa; lỗi CF-08 trong `errors.go`; `Task` thêm giờ thực tế và số việc con.
- Application: gói `timelog` (port + service + `service_test.go` viết trước với fake store): ghi/xem/xóa log; quyền theo người nhận của việc gốc, kiểm lại trong giao dịch đã khóa dự án → công việc; luật 24 giờ áp cho cả việc cha khi ghi vào con; phát `task.updated` cho thẻ việc gốc sau commit.
- Hạ tầng: `postgres/timelog_repo.go` (TimeLogRepo nhúng TaskRepo, TimeLogTx; ánh xạ CHECK/FK sang lỗi domain) + `timelog_integration_test.go` (giờ con cộng lên cha, lịch sử cha gồm log con, CHECK phút, luật 24 giờ cho cha và con).

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|

## Kiểm chứng
- `go test ./internal/domain/ ./internal/application/timelog/` xanh.
- Test tích hợp `go test -tags=integration ./internal/infrastructure/postgres/` trên PostgreSQL Homebrew (B12, DB `worktrack_it_logtime`, schema tạm): ok; CI chạy lại.
