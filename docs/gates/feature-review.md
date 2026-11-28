# Gate: feature/review — CF-09 (Nghiệm thu công việc cha)
**Ngày:** 2026-11-28 · **Commit cuối (code):** eb19930 · **Kết luận:** ĐẠT (kèm ghi chú — xem "Còn tồn đọng")

Phạm vi:
- **Backend:**
  - Bảng `task_review` (migration `20261128004725_add_task_review.sql`): quyết định `approved`/`rejected`, CHECK lý do 1–500 ký tự khi Từ chối, NULL khi Duyệt. Migration `20261128102034_restrict_task_review_column_fks.sql` đổi FK cột sang ON DELETE RESTRICT.
  - Domain `review.go`: `ReviewTargets` (cột chờ nghiệm thu = cột cuối nhóm `in_progress`; Duyệt → cột done đầu tiên; Từ chối → cột in_progress đầu tiên), `NormalizeReviewReason`, `CanReview` (chỉ Manager phụ trách).
  - Use case `kanban.ReviewService`: dùng lại `Service.moveLocked` (luật `workflow_transition`, WIP, version, lịch sử chuyển cột) trong giao dịch đã khóa; ghi `task_review`; Từ chối gửi thông báo `review_rejected`; phát `task.moved` sau commit.
  - 3 endpoint: `POST /tasks/{id}/review/approve`, `POST /tasks/{id}/review/reject`, `GET /tasks/{id}/reviews`. Lỗi: 409 `TASK_NOT_IN_REVIEW`, 422 `REVIEW_REASON_INVALID`, 409 `COLUMN_HAS_REVIEWS` khi xóa cột có lịch sử nghiệm thu.
- **Frontend:**
  - Khối Nghiệm thu trong chi tiết công việc (`ReviewSection.tsx`): nút "Nghiệm thu" / "Từ chối (kèm lý do)" chỉ cho Manager phụ trách khi việc ở cột chờ nghiệm thu; dòng nhắc việc con chưa xong; lịch sử nghiệm thu.
  - Hộp thoại Từ chối (`RejectReviewDialog.tsx`): lý do bắt buộc, tối đa 500 ký tự.
  - Kanban: kéo việc cha vào cột chờ nghiệm thu khi còn việc con chưa xong → toast cảnh báo, vẫn cho chuyển (cả lần Hoàn tác).
- **Không làm ở nhánh này:** tính tỷ lệ làm lại (CF-10/11 đọc từ `task_review`).

Gate chạy 1 vòng sửa: chạy toàn bộ lệnh tự động, go-reviewer, ui-reviewer, qa-tester. Không có mục Chặn. Các mục "Nên sửa" đã sửa ở 5646712, 1073e5f, 83bb41a; giả định và ma trận truy vết ở eb19930; mục nhỏ còn lại đưa vào #19. Chạy lại toàn bộ lệnh tự động sau khi sửa — tất cả xanh.

## Tự động
Môi trường: máy không có Docker (B12). PostgreSQL chạy bằng Homebrew; integration test chạy trên CSDL `worktrack_gate_review` tạo mới. `docker build` và `make test-integration` (testcontainers) chỉ chạy trong CI.

| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint + eslint + `tsc -b` + prettier) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: tất cả gói ok · FE: 347/347 test pass |
| `go test -count=1 -tags=integration ./...` (PostgreSQL local) | ✅ ok, gồm `review_integration_test.go` (Từ chối rồi Duyệt, `task_review` + lịch sử chuyển cột, xóa cột có nghiệm thu → `ErrColumnHasReviews`) và `review_qa_integration_test.go` |
| `make cover` (gói cốt lõi domain + application) | ✅ 89,9% |
| `gitleaks detect --no-banner` | ✅ no leaks found |
| `go vet ./...`, `tsc --noEmit` | ✅ 0 issues |
| `python3 scripts/ai-log/ai_log.py check` | ⚠️ Không có lỗi nào trên các commit của nhánh này. Còn 2 lỗi cũ từ `release/v1`, không thuộc nhánh: entry `2026-11-27-01-test-cases.md` và commit `8712025` thiếu trailer `AI-Log`. Có cảnh báo mục "Phần tôi đã sửa" chưa điền |
| `docker build`, `make test-integration` (CI) | xem mục CI bên dưới |

## CI
Cập nhật sau khi push (xem mục này ở commit kế tiếp).

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Xác thực | Không token → 401 trên cả 3 endpoint | ✅ | `review_api_test.go` |
| Phân quyền | <ul><li>Chỉ Manager phụ trách dự án được Duyệt/Từ chối.</li><li>Manager khác, Member, CEO, Admin → 403; kiểm quyền trước khi kiểm lý do (Member gửi lý do rỗng nhận 403, không phải 422).</li><li>Kiểm lại quyền trên dòng project đã khóa trong giao dịch.</li><li>Xem lịch sử nghiệm thu theo quyền đọc công việc.</li></ul> | ✅ | `domain/review_test.go`, `application/kanban/review_test.go`, `review_api_test.go` |
| IDOR | Người ngoài dự án dùng ID việc hợp lệ → 404, không lộ dữ liệu | ✅ | `review_test.go`, `review_api_test.go` |
| SQL / lỗi 500 | Mọi truy vấn qua sqlc (`task_review.sql`), không nối chuỗi; lỗi 500 trả câu chung | ✅ | go-reviewer |
| Quy tắc nghiệp vụ | <ul><li>Chỉ nghiệm thu việc đang ở cột chờ nghiệm thu (409 `TASK_NOT_IN_REVIEW`).</li><li>Duyệt → nhóm done; Từ chối → nhóm in_progress, qua đúng luật chuyển cột Kanban, WIP, version (409 khi lệch).</li><li>Việc con chưa xong: cảnh báo, không chặn.</li><li>Từ chối ghi `task_review` (dữ liệu tỷ lệ làm lại) và báo người nhận việc/người tạo.</li><li>Xóa cột có lịch sử nghiệm thu → 409 `COLUMN_HAS_REVIEWS`, không mất dữ liệu.</li></ul> | ✅ | `review_test.go`, `review_integration_test.go`, `review_qa_integration_test.go`, `useBoardDrag.test.tsx` |
| Validate | <ul><li>Lý do rỗng, chỉ khoảng trắng, 501 ký tự → 422; 500 ký tự, tiếng Việt, emoji → ok (CHECK ở CSDL).</li><li>UUID sai, body sai JSON → 400; thiếu `version` → 400.</li><li>FE: lý do bắt buộc, đếm ký tự tối đa 500.</li></ul> | ✅ | `domain/review_test.go`, `review_api_test.go`, `Review.test.tsx`, `review.test.ts` |
| HTML/script trong text | Lý do lưu và trả nguyên văn; FE hiển thị dạng chữ, không dùng `dangerouslySetInnerHTML` | ✅ | `Review.test.tsx` |
| UX | <ul><li>Luồng Duyệt/Từ chối cho Manager; Member/CEO không thấy nút.</li><li>Đang tải / lỗi tải bảng: báo rõ thay vì ẩn nút im lặng.</li><li>Lỗi 409 hiện thông báo tiếng Việt; toast thành công; nút khóa khi đang gửi.</li><li>Hộp thoại Từ chối: Esc đóng, focus vào ô lý do.</li></ul> | ✅ (mục nhỏ ở #19) | `Review.test.tsx`, `useBoardDrag.test.tsx`; ui-reviewer 0 mục Chặn |

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | FK `task_review` → cột dùng ON DELETE CASCADE: xóa cột làm mất lịch sử nghiệm thu (go-reviewer) | Trung bình | 5646712 |
| 2 | Kiểm lý do trước kiểm quyền: Member gửi lý do rỗng nhận 422 thay vì 403 (qa-tester) | Thấp | 1073e5f |
| 3 | Hoàn tác đưa việc cha về cột chờ nghiệm thu không hiện cảnh báo việc con (ui-reviewer) | Thấp | 83bb41a |
| 4 | Chi tiết công việc ẩn nút Nghiệm thu im lặng khi bảng đang tải hoặc lỗi tải (ui-reviewer) | Thấp | 83bb41a |

## Còn tồn đọng
- **Người dùng cần làm:**
  - Điền mục "Phần tôi đã sửa / hoàn thiện" trong `docs/ai-log/entries/2026-11-28-01-feature-review.md`.
  - Lỗi `ai_log.py check` từ `release/v1` (entry `2026-11-27-01-test-cases.md`, commit `8712025`) cần xử lý ở nhánh đó.
  - Chạy thử luồng curl §4.7 trong `docs/testing/manual-test-guide.md` trên app dev (cần mật khẩu tài khoản mẫu trong `.env`, Claude không đọc được).
- **Issue [#19](https://github.com/BaoCool-dev/WorkTrack/issues/19)** (không chặn): test API/FE cho `COLUMN_HAS_REVIEWS`; E2E Playwright luồng nghiệm thu; test WebSocket sau nghiệm thu; tỷ lệ làm lại ở CF-10/11.
- **Giả định mục C:** 8 dòng trong `docs/decisions.md` (ngày 2026-11-28, nhánh `feature/review`):
  - cột chờ nghiệm thu xác định theo vị trí (cột cuối nhóm `in_progress`), đích Duyệt/Từ chối theo vị trí;
  - Duyệt/Từ chối dùng lại luật chuyển cột Kanban; chỉ Manager phụ trách được nghiệm thu;
  - lý do từ chối 1–500 ký tự, Duyệt không lưu lý do;
  - việc con chưa xong chỉ cảnh báo, không chặn;
  - tỷ lệ làm lại do CF-10/11 tính từ `task_review`;
  - FK cột của `task_review` là RESTRICT, xóa cột có nghiệm thu → 409;
  - bỏ checkbox "Trả thẻ về cột Đang làm…" của artboard 13;
  - toast cảnh báo thay hộp thoại "Để sau / Vẫn chuyển" của artboard 13.
