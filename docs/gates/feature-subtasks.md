# Gate: feature/subtasks — CF-07 (việc con, bình luận, thông báo trong ứng dụng)
**Ngày:** 2026-11-27 · **Commit cuối (code):** fa04f0a · **Kết luận:** ĐẠT (kèm ghi chú — xem "Còn tồn đọng")

Phạm vi:
- **Backend:**
  - Việc con 1 cấp:
    - bảng `task` dùng `parent_task_id` có sẵn;
    - trigger `trg_task_one_level`: cha phải là việc gốc, việc đã có con không thể thành con;
    - CHECK `ck_task_story_point_parent_only` chặn Story Point ở việc con.
  - Bảng `task_comment` và `notification` (goose + sqlc), có index cho danh sách và số chưa đọc.
  - Use case:
    - `application/subtask`: tạo/liệt kê/chuyển/xóa con. Quyền đi theo người nhận việc cha. Chuyển con ghi `task_move_history` và không qua `workflow_transition`.
    - `application/comment`: đọc/viết bình luận, 1–2000 code point.
    - `application/notification`: `task_assigned`, `task_commented`, `review_rejected` (chỉ định nghĩa sẵn, CF-09 sẽ gọi); danh sách, số chưa đọc, đánh dấu đã đọc.
  - 10 endpoint mới:
    - việc con: `GET/POST /tasks/{id}/subtasks`, `POST /subtasks/{id}/move`, `DELETE /subtasks/{id}`;
    - bình luận: `GET/POST /tasks/{id}/comments`;
    - thông báo: `GET /me/notifications`, `GET /me/notifications/unread-count`, `POST /me/notifications/{id}/read`, `POST /me/notifications/read-all`.
- **Frontend:**
  - Mục "Việc con" trong hộp chi tiết công việc: thêm bằng Enter, chọn cột, tiến độ x/y, xóa có hàng xác nhận ngay trong danh sách (Esc hủy).
  - Mục "Bình luận".
  - Chuông thông báo ở thanh trên: số chưa đọc, danh sách, đánh dấu đã đọc từng cái hoặc tất cả, làm mới mỗi 30 giây.
- **Không làm ở nhánh này:**
  - nghiệm thu và gọi `review_rejected` (CF-09);
  - Log time (CF-08);
  - thông báo đẩy qua WebSocket.

Gate chạy 2 vòng:
- **Vòng 1:**
  - Chạy toàn bộ lệnh tự động, rồi go-reviewer, ui-reviewer, qa-tester.
  - Không có mục Chặn.
  - go-reviewer phát hiện lỗi #1–#3; đã sửa ở 946d5df.
  - qa-tester và kiểm tay phát hiện lỗi #4–#5; đã sửa ở fa04f0a. Mỗi lỗi có test đỏ trước khi sửa.
  - qa-tester bổ sung test API (401, 400, biên, HTML).
- **Vòng 2:**
  - Chạy lại toàn bộ lệnh tự động ở fa04f0a.
  - Kiểm UX bằng Playwright MCP ở 1366×768.
  - Tất cả xanh.

## Tự động
Môi trường: máy không có Docker (B12).
- PostgreSQL chạy bằng Homebrew.
- Integration test chạy trên CSDL `worktrack_gate_subtasks` tạo mới.
- `docker build` và `make test-integration` (testcontainers) chỉ chạy trong CI.

| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint + eslint + `tsc -b` + prettier) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: tất cả gói ok · FE: 247/247 test pass (25 file) |
| `go test -count=1 -tags integration ./internal/infrastructure/postgres/` (PostgreSQL local) | ✅ ok. Gồm `TestCollab_ServiceFlow_CF07`, `TestMoveHistory_CF07`, `TestKanbanRepo_CountProjectWIP_ExcludesSubtasks_CF05`; CHECK SP ở con; trigger 1 cấp |
| `make cover` (gói cốt lõi domain + application) | ✅ 89,8% (`application/subtask` 82,1%, `application/comment` 89,7%, `application/notification` 81,0%) |
| `gitleaks detect --no-banner` | ✅ no leaks found |
| `go vet ./...` (kể cả tag `integration`), `tsc --noEmit` | ✅ |
| `python3 scripts/ai-log/ai_log.py check` | ⚠️ Không có lỗi nào trên các commit của nhánh này. Có 2 lỗi cũ từ `release/v1`, không thuộc nhánh: entry `2026-11-27-01-test-cases.md` và commit `8712025` thiếu trailer `AI-Log`. Có cảnh báo mục "Phần tôi đã sửa" chưa điền |
| `docker build`, `make test-integration` (CI) | xem mục CI bên dưới |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Xác thực | Không token → 401 trên cả 10 endpoint mới | ✅ | `TestCollabAPI_AllEndpointsRequireLogin_CF07` |
| Phân quyền | <ul><li>Tạo/chuyển/xóa con: chỉ Manager phụ trách hoặc người nhận hiện tại của việc cha. Khi cha được giao lại, người cũ mất quyền.</li><li>Admin, CEO, Member khác → 403.</li><li>Bình luận: Admin, CEO chỉ đọc; Manager/Member thành viên dự án được viết.</li><li>Thông báo: chỉ người nhận đọc và đánh dấu được.</li></ul> | ✅ | `TestCheckSubtaskActor_CF07`, `TestSubtaskPermissionFollowsParent_CF07`, `TestCreate_CF07`, `TestMove_CF07`, `TestDelete_CF07`, `TestCanComment_CF07`, `TestCreateComment_CF07`, `TestNotifications_CF07`, `TestCollabAPI_ErrorMapping_CF07` |
| IDOR | <ul><li>Người ngoài dự án dùng ID việc/con hợp lệ → 404.</li><li>Đánh dấu thông báo của người khác → 404.</li><li>Mọi truy vấn có lọc theo dự án hoặc theo `user_id`.</li></ul> | ✅ | `TestList_CF07`, `TestListComments_CF07`, `TestCollab_ServiceFlow_CF07`, `TestCollabAPI_Notifications_CF07` |
| SQL / lỗi 500 | <ul><li>Mọi truy vấn qua sqlc, không nối chuỗi.</li><li>Lỗi 500 trả câu chung.</li></ul> | ✅ | `backend/db/queries/subtask.sql`, `comment.sql`, `notification.sql`; go-reviewer |
| Quy tắc nghiệp vụ | <ul><li>Con 1 cấp (trigger CSDL + domain).</li><li>Con không có Story Point (CHECK CSDL).</li><li>Con không lên Kanban và không tính WIP (§5.6).</li><li>Chuyển con không qua `workflow_transition`, có ghi lịch sử; chuyển vào cột hiện tại là no-op.</li><li>Thông báo bỏ tác giả và bỏ trùng.</li><li>Bình luận trên việc con báo cho người nhận việc cha.</li></ul> | ✅ | `TestCheckSubtaskParent_CF07`, `TestMove_SubtaskNotOnBoard_CF04`, `TestUpdate_SubtaskNotHandledHere_CF06`, `TestMoveHistory_CF07`, `TestNotificationRecipients_CF07`, `TestAssignmentRecipient_CF07`, `TestNotifyAssigned_CF07`, `TestCommentOnSubtaskNotifiesParentAssignee_CF07` |
| Validate | <ul><li>Tên con rỗng, chỉ khoảng trắng, 200/201 rune.</li><li>Bình luận rỗng hoặc 2001 code point; tiếng Việt có dấu và emoji đếm theo code point ở cả FE và BE.</li><li>UUID sai, body sai JSON, query sai → 400.</li><li>Loại thông báo không hợp lệ bị chặn.</li></ul> | ✅ | `TestNormalizeComment_CF07`, `TestNotificationExcerpt_CF07`, `TestNotificationTypeValid_CF07`, `TestCollabAPI_SubtaskTitleBoundary_CF07`, `TestCollabAPI_BadPathBodyQuery_CF07`, `TestCollabAPI_CommentErrorMapping_CF07`, `TaskCollab.test.tsx` |
| HTML/script trong text | <ul><li>Bình luận `<script>`/`<img onerror>` lưu nguyên văn, trả về nguyên văn.</li><li>FE hiển thị dạng chữ, không dùng `dangerouslySetInnerHTML`.</li><li>Đoạn trích trong chuông cũng hiển thị dạng chữ.</li></ul> | ✅ | `TestCollabAPI_CommentWithHTMLRoundTrip_CF07`, ca XSS trong `TaskCollab.test.tsx`, `NotificationBell.test.tsx` |
| UX — Member | <ul><li>Thêm con bằng Enter; chuyển con sang Hoàn thành, tiến độ 1/1.</li><li>Gửi bình luận có HTML và emoji, hiển thị dạng chữ.</li><li>Bấm "Xóa": focus vào "Hủy". Esc chỉ hủy xác nhận, hộp chi tiết vẫn mở.</li></ul> | ✅ | Playwright MCP 1366×768: `cf07-chi-tiet-member.png`, `cf07-xac-nhan-xoa.png` |
| UX — Manager | <ul><li>Chuông đọc "Thông báo, 1 chưa đọc", hiển thị thông báo bình luận với đoạn trích dạng chữ.</li><li>Bấm thông báo thì đánh dấu đã đọc.</li></ul> | ✅ (ghi chú: bấm thông báo chưa mở công việc, xem #14) | `cf07-chuong-manager.png` |
| UX — Admin / CEO | <ul><li>Ô bình luận chỉ hiện với Manager/Member (`CommentsSection.tsx`, `canWrite`).</li><li>Nút việc con chỉ hiện với người nhận việc cha hoặc Manager.</li><li>Server vẫn chặn khi gọi API trực tiếp.</li><li>Chưa có test FE riêng cho Admin/CEO và chưa kiểm bằng trình duyệt (ghi vào #14).</li></ul> | ✅ (server) | `TestCanComment_CF07`, `TestCheckSubtaskActor_CF07`, `TestCollabAPI_ErrorMapping_CF07`; `TaskCollab.test.tsx` "không phải người nhận: ẩn nút thêm" |
| Trạng thái giao diện | Đủ trạng thái đang tải, rỗng, lỗi + Thử lại, toast thành công. Thao tác thất bại thì UI trả về đúng trạng thái | ✅ | `TaskCollab.test.tsx`, `NotificationBell.test.tsx`; ui-reviewer 0 mục Chặn |

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | Quyền tạo/chuyển/xóa con dựa vào người nhận lưu trên việc con. Khi việc cha được giao lại, người cũ vẫn thao tác được và người mới bị chặn (go-reviewer) | Trung bình | 946d5df |
| 2 | Chuyển việc con không ghi `task_move_history` (go-reviewer) | Thấp | 946d5df |
| 3 | Thiếu index cho danh sách thông báo và số chưa đọc theo `user_id` (go-reviewer) | Thấp | 946d5df |
| 4 | Bình luận trên việc con gửi thông báo cho người nhận lưu trên con, không phải người nhận hiện tại của việc cha | Trung bình | fa04f0a |
| 5 | Esc trong hàng xác nhận xóa con đóng luôn cả hộp chi tiết; nút "Hủy" chưa nhận focus | Thấp | fa04f0a |

## Còn tồn đọng
- **Người dùng cần làm:**
  - Điền mục "Phần tôi đã sửa / hoàn thiện" trong `docs/ai-log/entries/2026-11-27-03-feature-subtasks.md`.
  - Lỗi `ai_log.py check` từ `release/v1` (entry `2026-11-27-01-test-cases.md`, commit `8712025`) cần xử lý ở nhánh đó.
- **Issue [#14](https://github.com/BaoCool-dev/WorkTrack/issues/14)** (mục nhỏ, không chặn):
  - Gom kiểm vai trò việc con và bình luận vào `authz`.
  - Test `pageSize` cho danh sách thông báo.
  - CHECK độ dài bình luận ở CSDL.
  - E2E tự động.
  - Gọi `review_rejected` ở CF-09.
  - Thiếu artboard cho việc con, bình luận và chuông.
  - Các mục "Nên sửa" của ui-reviewer.
  - Bấm thông báo chưa mở công việc liên quan.
  - Cấp heading của `EmptyState` trong hộp chi tiết.
  - Test FE cho chế độ chỉ xem của Admin/CEO (ô bình luận, nút việc con).
- **Giả định mục C:** 11 dòng trong `docs/decisions.md` (ngày 2026-11-27, nhánh `feature/subtasks`):
  - trigger 1 cấp;
  - quyền việc con đi theo người nhận việc cha;
  - con không phát realtime và không gửi thông báo giao việc;
  - quyền và độ dài bình luận;
  - người nhận thông báo;
  - chuông làm mới mỗi 30 giây;
  - xác nhận xóa ngay trong danh sách;
  - đếm theo code point;
  - thông báo cũ vẫn hiện sau khi bị gỡ khỏi dự án;
  - kiểm vai trò tạm ở domain;
  - thao tác con không phụ thuộc trạng thái việc cha.

## CI
Cập nhật sau khi push (xem bên dưới).
