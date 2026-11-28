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
- Frontend API client: `tasks/model/types.ts` (kiểu `TimeLog`, `TimeLogInput`, giờ thực tế và số việc con trên `Task`), `tasksApi.ts` (3 lời gọi Log time), `queries.ts` (`useTimeLogs`, `useCreateTimeLog`, `useDeleteTimeLog`, làm mới chi tiết/bảng/danh sách sau khi ghi), `test-data.ts` (`makeTimeLog`).
- Thẻ Kanban: `TaskCard.tsx` dùng `actualMinutes` thay số 0 tạm, thêm `x/y con`; việc không có định mức chỉ hiện giờ đã log; `TaskCard.test.tsx` viết trước (mức B5, thanh tối đa 100%, việc con).
- Hạ tầng: `postgres/timelog_repo.go` (TimeLogRepo nhúng TaskRepo, TimeLogTx; ánh xạ CHECK/FK sang lỗi domain) + `timelog_integration_test.go` (giờ con cộng lên cha, lịch sử cha gồm log con, CHECK phút, luật 24 giờ cho cha và con).
- HTTP: `httpapi/timelog_handler.go` (`GET/POST /tasks/{taskID}/time-logs`, `DELETE /time-logs/{logID}`), ánh xạ 6 lỗi CF-08 trong `errors.go`, `taskResponse` thêm `loggedMinutes`, `actualMinutes`, `subtaskCount`, `subtaskDoneCount`; wiring `main.go`; `timelog_api_test.go` viết trước với service giả (201/200/204, 400, 401, 403, 404, 409).
- Giao diện Log time: `tasks/model/timeLogForm.ts` + `timeLogForm.test.ts` (viết trước: đọc số giờ có dấu phẩy, làm tròn phút, giới hạn 1440 phút, mô tả 1–500 ký tự, ngày không ở tương lai); `TimeLogSection.tsx` (giờ thực tế/ước tính kèm ký hiệu mức B5, nút Log giờ theo quyền, lịch sử có phân trang, xóa log của mình với xác nhận ngay trong dòng); `LogTimeDialog.tsx` (chọn việc cha hoặc một việc con, ngày làm, số giờ, mô tả; 409 `TIME_LOG_CLOSED` → khối khóa); `TaskDetailDialog.tsx` gắn mục giờ thực tế và chuyển sang hộp Log giờ; `useTask` thêm cờ `enabled` để tải việc cha khi xem việc con; câu chữ trong `messages.ts`.
- Tài liệu: `docs/testing/manual-test-guide.md` §4.6 CF-08 (các bước trên giao diện, curl ghi/xem/xóa và các ca bị chặn, SQL xem `time_log` và `task_view.actual_minutes` = cha + con, cách giả Done quá 24 giờ trên CSDL dev), dòng `time_log` ở §5.2, checklist CF-08 ở §8.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Test API cho rằng `?page=0` bị từ chối 400 | Đoán hành vi `pageFrom` thay vì đọc code (thực tế chuẩn hóa về trang 1) | Test `TestTimeLogAPI_BadInput_CF08/page` đỏ (200) | Đổi ca test sang `page=abc` (không phải số → 400) |
| Test UI cho CEO mở chi tiết từ `/my-tasks` | Quên guard route: CEO không có trang "Việc của tôi" (TC_TASK_042) | Test "CEO chỉ xem" đỏ: không tìm thấy nút mở chi tiết | CEO mở chi tiết từ tab Công việc của dự án |
| Test dùng `buttons[0]` (kiểu `HTMLElement \| undefined`) | Quên `noUncheckedIndexedAccess` của tsconfig strict | `tsc --noEmit` báo TS2345 | Dùng `getByRole` trả một phần tử |
| `timeLogForm.test.ts` (test bổ sung của QA) sai định dạng Prettier | Viết test mới mà không chạy `prettier --write` | `make lint` (bước `prettier --check`) đỏ | `npx prettier --write` rồi amend commit |

## Kiểm chứng
- `go test ./internal/domain/ ./internal/application/timelog/` xanh.
- `go test ./...` và `golangci-lint run` (backend): xanh, 0 issues.
- Test tích hợp `go test -tags=integration ./internal/infrastructure/postgres/` trên PostgreSQL Homebrew (B12, DB `worktrack_it_logtime`, schema tạm): ok; CI chạy lại.
- Sau review (go-reviewer, ui-reviewer) và QA bổ sung test (aa7e978, 477d8cc, 33ecab2): `make lint`, `make test` xanh lại; Frontend 314/314 (28 file); `make cover` 89,7% (gói timelog 86,1%); gitleaks 0 secret. Báo cáo cổng: `docs/gates/feature-logtime.md`.
- Lần chạy đầu — Frontend: `npx vitest run` 278/278 xanh; `eslint`, `tsc --noEmit`, `prettier --check` xanh (`make lint`, `make test` từ gốc repo).
- Hướng dẫn kiểm thử thủ công: câu chữ UI đối chiếu với `messages.ts`, đường dẫn/mã lỗi đối chiếu với `router.go` và `errors.go`; chưa chạy curl trực tiếp (app dev không chạy, `make seed` sẽ đặt lại mật khẩu dev).
