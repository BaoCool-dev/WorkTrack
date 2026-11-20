---
name: implement-ticket
description: Quy trình bắt buộc khi bắt đầu làm một issue, user story, use case hoặc tính năng mới của WorkTrack (backend Go hoặc frontend React). Dùng khi người dùng nói "làm issue #..", "làm UC..", "cài đặt CF-..", "code tính năng ..".
---

# Quy trình làm một việc

Làm đúng thứ tự. Không bỏ bước.

## 0. Đúng nhánh
- `git switch main && git pull`, rồi `git switch -c feature/<ten>` (một tính năng một nhánh — xem `CLAUDE.md` §7b).
- Nhánh `feature/kanban`: Claude viết bản đầu, code dễ đọc, nhật ký AI tối giản (xem `CLAUDE.md` §7b).
- Tính năng trước chưa qua `feature-gate` → dừng, nhắc người dùng.

## 1. Hiểu việc
1. Nếu có số issue: `gh issue view <số>` để đọc nội dung thật. Không đoán nội dung issue.
2. Tìm mã yêu cầu (CF-xx, UCxx, YCCN-xx, Rxx) và đọc đoạn tương ứng trong `docs/requirements/`.
3. Kiểm tra việc có thuộc **Lõi** (CF-01..CF-12) không. Nếu là **Mở rộng** hoặc **Ngoài phạm vi** → dừng, báo người dùng.
4. Đụng tới mục "CHƯA CHỐT" trong `CLAUDE.md` §6 → hỏi một lần, kèm phương án.

## 2. Lập kế hoạch ngắn (gửi người dùng trước khi code nếu việc > 3 file)
- Lát dọc: migration → query sqlc → domain → use case → handler → FE api → UI.
- Liệt kê file sẽ tạo/sửa, test sẽ viết (ghi mã AC), rủi ro.

## 3. Test trước
- Domain/use case: test table-driven ở trạng thái đỏ.
- API: test `httptest` cho 2xx, 400, 401, 403.
- FE: test RTL + MSW cho trạng thái tải/rỗng/lỗi/thành công.

## 4. Code
- Theo skill `go-backend`, `db-migration-sqlc`, `react-feature`, `worktrack-domain`.
- Tra API thư viện bằng Context7 khi không chắc (pgx, sqlc, chi, coder/websocket, TanStack Query, Tailwind v4).

## 5. Kiểm tra
- `make lint` và `make test` phải xanh. Sửa tới khi xanh, không tắt rule lint để qua.
- Việc lớn: gọi agent `go-reviewer` (backend) hoặc `ui-reviewer` (frontend) để soát.
- Đối chiếu DoD trong `CONTRIBUTING.md` §9.

## 6. Kết thúc
- Commit message Conventional Commits kèm mã: `feat(scope): mô tả [CF-xx][UCxx]`.
- Commit qua skill `ai-usage-log` (entry + trailer `AI-Log`). Không commit tay bỏ qua bước này.
- Tóm tắt cho người dùng: đã làm gì, test nào, việc còn lại. Không tự `git push`, không tự merge.
- Xong cả tính năng → chạy skill `feature-gate` trước khi mở PR.
