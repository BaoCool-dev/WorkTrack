# Gate: docs/manual-testing — hướng dẫn kiểm thử thủ công (CF-01, CF-02, CF-03, CF-06)
**Ngày:** 2026-11-21 · **Commit cuối (tài liệu):** 684c178 · **Kết luận:** ĐẠT

Phạm vi:
- **Thêm:** `docs/testing/manual-test-guide.md`, tiếng Việt, khoảng 1000 dòng, 8 mục:
  1. cài đặt không Docker (B12);
  2. đăng nhập, refresh, đăng xuất và khóa tài khoản cho 4 vai trò;
  3. Admin tạo tài khoản → Mailpit → kích hoạt;
  4. từng chức năng CF-02, CF-03, CF-06 theo vai trò, kèm ca lỗi, ca quyền và SQL kiểm;
  5. psql, pgAdmin và bảng nào chứa gì;
  6. curl có token;
  7. đặt lại dữ liệu an toàn;
  8. checklist.
- **Sửa nhẹ:**
  - `docs/testing/README.md`: thêm một dòng;
  - `docs/decisions.md`: thêm 4 giả định vào mục C.
- **Không sửa code sản phẩm.** `git diff --stat origin/feature/tasks...HEAD` chỉ có 4 file trong `docs/`.

Tiêu chí gate của nhánh tài liệu (theo yêu cầu phiên): **mọi lệnh và câu SQL trong tài liệu chạy được trên máy dev**. Không kiểm UI.
Các mục §2–§4 của skill `feature-gate` (endpoint, validate, UX) không áp dụng vì nhánh không có endpoint hay màn hình mới.

Gate chạy 1 vòng.

## Kiểm chứng tài liệu
**Môi trường:**
- macOS;
- PostgreSQL 16 và Mailpit cài bằng Homebrew;
- `make migrate-up`, `make seed`, `make dev-local` (API :8080, web :5173);
- CSDL dev `worktrack`.

**Cách làm:**
- Một script tạm ngoài repo tách mọi khối ```` ```bash ```` và ```` ```sql ```` trong tài liệu, rồi chạy tuần tự theo đúng thứ tự đọc.
- Script tự thay `<MK_…>` bằng mật khẩu `make seed` vừa sinh. Mật khẩu không ghi vào repo.
- Script chờ giữa các lần đăng nhập vì giới hạn tần suất.
- SQL chạy bằng `psql -v ON_ERROR_STOP=1`.

| Hạng mục | Kết quả |
|---|---|
| Tổng số khối lệnh | 86 |
| Bỏ qua có chủ đích (`verify:skip`) | 6 khối, không chạy:<br>• `createuser`/`createdb`<br>• `cp .env.example`<br>• `make dev-local` (đã chạy riêng)<br>• psql tương tác<br>• `goose reset` và migrate+seed đi kèm |
| Chạy và khớp kết quả mô tả | 80/80, sau khi sửa lỗi #1 (lần đầu 79/80) |
| Mã lỗi và thông điệp trong tài liệu | Khớp phản hồi thật, ví dụ:<br>• `UNAUTHENTICATED`<br>• `AUTH_ACCOUNT_LOCKED` ở lần sai thứ 5<br>• `PASSWORD_TOO_WEAK`<br>• `ACTIVATION_TOKEN_INVALID`<br>• `USER_ALREADY_ACTIVE`<br>• các mã CF-02/03/06 |
| Tên ràng buộc trích ở §5 | Có trong `\d task`:<br>• `ck_task_story_point_parent_only`<br>• `ck_task_story_point_range`<br>• `fk_task_parent` (CASCADE)<br>• `fk_task_assignee` (RESTRICT) |
| Dọn dữ liệu thử §7.3 | Chạy trong `BEGIN … ROLLBACK`, 3 lần liên tiếp, không lỗi; không dữ liệu nào bị xóa |
| Reset toàn bộ §7.4 | **Không chạy.** Phiên tự chạy không được xóa dữ liệu; cú pháp kiểm bằng `goose --help` |
| Bí mật trong tài liệu | Không có mật khẩu seed hay email thật (đã grep) |

## Tự động
| Lệnh | Kết quả |
|---|---|
| `make lint` | ✅ exit 0: golangci-lint, eslint, tsc, prettier sạch |
| `make test` | ✅ exit 0: mọi gói Go ok; Vitest 16 file / 171 test pass |
| `gitleaks detect --no-banner` | ✅ no leaks found (61 commit) |
| `python3 scripts/ai-log/ai_log.py check` | ✅ OK. Cảnh báo: mục "Phần tôi đã sửa" chờ người dùng điền |
| Trailer `AI-Assisted` + `AI-Log` | ✅ có ở mọi commit của nhánh |

`make test-integration` và `make cover` không chạy lại vì nhánh không đổi code. CI chạy integration (testcontainers) và Docker build.

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | Câu `DELETE FROM department WHERE name LIKE 'Khối thử %'` (§7.3) làm tiến trình PostgreSQL bị hủy: `Assertion failed … lookup_substsearch, collate.c` (libc macOS, collation `en_US.UTF-8`) → mất kết nối. Server tự phục hồi; transaction bị hủy nên không mất dữ liệu | Cao (người đọc làm theo sẽ gặp) | 53c656c: thêm `COLLATE "C"` cho các `LIKE` tiếng Việt và ghi chú lý do |
| 2 | Token kích hoạt đọc từ Mailpit dính `\r` | Trung bình | 53c656c (`tr -d '\r\n'`) |
| 3 | SQL hệ số B1 và đường dẫn jq viết theo đoán, sai với lược đồ/DTO thật | Trung bình | 53c656c |

## Quan sát về sản phẩm (không sửa ở nhánh này)
- Phản hồi `POST /users` có tên phòng ban và chức danh rỗng.
- `POST /auth/login` và `GET /auth/me` trả `department: null`.
- Khi tài khoản bị khóa, `failed_login_count` về 0 còn `locked_until` có giá trị. Tài liệu §2.3 mô tả đúng hành vi này.

Ba mục trên chỉ được ghi lại, không chặn gate. Người dùng quyết định có mở issue hay không.

## CI
PR #8 (draft, base `feature/tasks`) — run 44812248595, commit 1317830:
| Job | Kết quả |
|---|---|
| Backend (build → lint → test, gồm integration testcontainers) | ✅ pass (2m0s) |
| Frontend (build → lint → test) | ✅ pass (49s) |
| Secret scan (gitleaks) | ✅ pass (17s) |
| Docker build | ✅ pass (2m55s) |
