---
title: feature review
date: 2026-11-28
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-09
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác -->

## Mục tiêu
Làm CF-09: luồng nghiệm thu công việc cha. Member đưa việc cha sang cột "Chờ nghiệm thu" (cảnh báo nếu còn việc con chưa xong, vẫn cho phép). Manager Duyệt (sang nhóm done) hoặc Từ chối với lý do bắt buộc (về in_progress, tính vào tỷ lệ làm lại). Dùng lại luật chuyển cột của feature/kanban.

## Prompt đã dùng
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: feature/review (mọc từ feature/logtime). PR sẽ tạo với --base feature/logtime --draft. [...] Phạm vi nhánh này: CF-09. Luồng nghiệm thu: Member đưa công việc cha sang cột Chờ nghiệm thu (cảnh báo nếu còn con chưa xong, vẫn cho phép); Manager Duyệt (sang nhóm done) hoặc Từ chối bắt buộc lý do (về in_progress, tính tỷ lệ làm lại); dùng luật chuyển cột của feature/kanban, không viết lại. [...] [rút gọn]

## Nội dung AI tạo ra
- CSDL: migration `20261128004725_add_task_review.sql` tạo bảng `task_review`:
  - quyết định `approved`/`rejected`;
  - CHECK: bị từ chối thì lý do dài 1–500 ký tự, được duyệt thì lý do NULL;
  - FK và index trên các cột tham chiếu.
- Query `task_review.sql` (`InsertTaskReview`, `ListTaskReviews`); sinh lại sqlcgen.
- Domain: `internal/domain/review.go` và `review_test.go` (viết trước):
  - `ReviewTargets`: cột "Chờ nghiệm thu" là cột cuối của nhóm in_progress; Duyệt đi tới cột done đầu tiên; Từ chối đi tới cột in_progress đầu tiên.
  - `NormalizeReviewReason` chuẩn hóa lý do từ chối.
  - `CanReview`: chỉ Manager phụ trách dự án được nghiệm thu.
  - Thêm lỗi `ErrTaskNotInReview` và `ErrInvalidReviewReason`.
- Application: `internal/application/kanban/review.go` + `review_test.go` (viết trước):
  - `ReviewService.Approve/Reject/Reviews`; khóa project → task, kiểm lại quyền Manager phụ trách trên dòng đã khóa;
  - chuyển cột bằng `Service.moveLocked` có sẵn (luật workflow_transition, WIP, version, lịch sử chuyển cột);
  - ghi `task_review`, gửi thông báo `review_rejected` khi Từ chối, phát `task.moved` sau commit.
- Postgres: `review_repo.go` (`ReviewRepo`, `ReviewTx`) và `review_integration_test.go` (Từ chối rồi Duyệt trên CSDL thật, kiểm `task_review` + lịch sử chuyển cột).
- HTTP: `review_handler.go` gồm `POST /tasks/{id}/review/approve`, `POST /tasks/{id}/review/reject` và `GET /tasks/{id}/reviews`, cùng `review_api_test.go` (200/400/401/403/404/409/422). Bổ sung ánh xạ lỗi `TASK_NOT_IN_REVIEW` (409) và `REVIEW_REASON_INVALID` (422), wiring trong `main.go` và `router.go`.
- Giao diện (frontend):
  - `features/tasks/model/review.ts` (+ test): `reviewColumnId` (cùng quy tắc vị trí với `ReviewTargets`), `hasPendingSubtasks`.
  - `ReviewSection.tsx`: nút "Nghiệm thu" / "Từ chối (kèm lý do)" chỉ cho Manager phụ trách khi việc ở cột chờ nghiệm thu, cảnh báo còn việc con chưa xong (không chặn), lịch sử nghiệm thu có badge + lý do.
  - `RejectReviewDialog.tsx`: textarea lý do bắt buộc (tối đa 500 ký tự), nút "Từ chối" màu `bad`.
  - API + hook: `approveReview`, `rejectReview`, `useTaskReviews`; `useBoard(id, enabled)`; biến thể `ok`/`bad` cho `Button`; chuỗi tiếng Việt trong `messages.ts`.
  - Kanban: kéo việc cha vào cột chờ nghiệm thu khi còn việc con chưa xong → toast cảnh báo thay cho toast thường.
  - Test: `Review.test.tsx` (11 test: nút theo vai trò, Duyệt/Từ chối, lỗi 409, lịch sử), 2 test mới trong `useBoardDrag.test.tsx`.
- Tài liệu: `docs/testing/manual-test-guide.md` thêm §4.7 CF-09 (thao tác giao diện, curl Duyệt/Từ chối/lịch sử, các mã lỗi 400/403/404/409/422, SQL xem `task_review` và `task_move_history`), dòng `task_review` ở §5.2, dòng nghiệm thu trong ma trận quyền §4, checklist §8. `docs/decisions.md` mục C thêm 5 giả định của nhánh.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Test Từ chối kỳ vọng nội dung thông báo giữ nguyên xuống dòng | Không đọc lại `NotificationExcerpt` (gộp khoảng trắng thành một dấu cách) | Test đỏ khi chạy | Sửa kỳ vọng theo hành vi có sẵn của excerpt |
| Tìm `subtask_repo.go` để xem mẫu repository (file không tồn tại) | Đoán tên file theo tên gói application | `ls` không thấy | Code việc con nằm trong `collab_repo.go` |
| Viết `review_handler.go` lệch định dạng gofmt (căn hàng hai hàm một dòng) | Viết tay bằng heredoc, không chạy gofmt trước lint | `make lint` báo gofmt | Chạy `gofmt -w` |
| Tìm tầng HTTP ở `internal/infrastructure/httpapi` (không tồn tại) | Đoán theo cấu trúc thường gặp thay vì liệt kê thư mục | `ls` báo không có thư mục | Liệt kê `internal/`: tầng HTTP nằm ở `internal/transport/httpapi` |
| Bản kế hoạch đầu nhận diện cột "Chờ nghiệm thu" theo tên cột | Không để ý tên cột do Manager tự đặt (CF-03) | Tự rà khi đọc lại mô hình bảng | Đổi sang suy ra theo vị trí + nhóm trạng thái (`ReviewTargets`, `reviewColumnId`) |
| `useTaskReviews` trả nguyên `{items}` thay vì mảng | Quên bóc lớp bọc của API danh sách | `npm run typecheck` (`tsc -b`) báo lỗi kiểu; `tsc -p .` không bắt được | Bóc `items` trong hàm gọi API |
| `Review.test.tsx` dùng khóa thông điệp không tồn tại (`tr.historyTitle`, `tr.confirm`) | Viết test theo trí nhớ, không đối chiếu `messages.ts` | Typecheck báo lỗi | Đổi sang `tr.history`, `tr.confirmReject` |
| Ba file FE lệch định dạng Prettier | Không chạy Prettier trước `make lint` | `format:check` đỏ | `npx prettier --write` |

## Kiểm chứng
- `go test ./internal/domain/ -run CF09`: ok.
- `go test ./internal/application/kanban/`: ok; `make lint`: 0 issues.
- Lát API: `make lint` xanh; `make test` xanh (Go mọi gói ok, Vitest 314/314); `go test -tags=integration -run 'Review|Kanban' ./internal/infrastructure/postgres/` trên Postgres Homebrew: ok.
- Lát giao diện: `make lint` xanh (sau Prettier); `make test` xanh (Go mọi gói ok, Vitest 335/335).
- Tài liệu: kiểm lại tên cột, mã lỗi và chuỗi giao diện trong §4.7 với `review_handler.go`, `review_api_test.go`, `messages.ts`; `make lint` xanh. Luồng curl §4.7 chưa chạy trên API dev (cần mật khẩu tài khoản mẫu trong `.env`); các ca tương ứng đã được phủ bởi test API và test tích hợp.
