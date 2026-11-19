---
title: feature auth
date: 2026-11-19
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-01
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác
     Thêm dòng "commit: none" ở phần đầu nếu việc này không tạo commit (vd. thiết kế, soạn báo cáo ngoài repo). -->

## Mục tiêu
<!-- 1–2 câu: định làm gì, gắn mã yêu cầu nào -->
Làm trọn CF-01 (xác thực và phân quyền) ở chế độ tự chạy: bảng tài khoản/vai trò/refresh token, Admin tạo tài khoản → email kích hoạt (B7) → đặt mật khẩu (B8), đăng nhập JWT access + refresh xoay vòng (B4), đăng xuất, `/auth/me`, gói `authz` duy nhất quyết định quyền (B2), giới hạn tần suất; frontend đăng nhập, kích hoạt, AuthProvider, route guard, AppShell, shared/ui; `make seed`, `make dev-local`.

## Prompt đã dùng
<!-- Dán NGUYÊN VĂN prompt chính (lấy từ: python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit).
     Mỗi prompt một dòng bắt đầu bằng ">". Dài quá thì rút gọn và ghi [rút gọn]. Không dán secret. -->
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: feature/auth (mọc từ chore/scaffold). PR sẽ tạo với --base chore/scaffold --draft. Môi trường: Máy KHÔNG có Docker (B12)… Phạm vi nhánh này: CF-01. Backend: bảng app_user, role, refresh_token; Admin tạo tài khoản → email kích hoạt qua Mailpit (B7) → đặt mật khẩu (B8); đăng nhập; JWT access + refresh xoay vòng (B4); đăng xuất; GET /api/v1/auth/me; khóa đăng nhập (B8); module internal/application/authz…; Frontend: trang đăng nhập, trang kích hoạt, AuthProvider, route guard, AppShell, shared/ui… make seed (B11). Test đủ 401/403 cho mọi vai trò. Quy trình bắt buộc: 1… 6. … Cuối cùng in đúng một dòng: GATE=ĐẠT hoặc GATE=CHƯA_ĐẠT. [rút gọn — nguyên văn trong scripts/auto-run.sh]

## Nội dung AI tạo ra
<!-- Liệt kê theo file/phần: AI viết gì. Có thể kèm git diff --stat của commit AI. -->
- **Lát 1 — CSDL + domain:** migration `20261119092421_create_auth_tables.sql` (bảng `role` có sẵn 4 vai trò, `app_user` với ràng buộc trạng thái/mật khẩu, token kích hoạt lưu SHA-256, `refresh_token` theo họ `family_id`); truy vấn sqlc `app_user.sql`, `refresh_token.sql`; override uuid trong `sqlc.yaml`; `domain/user.go` (vai trò, chính sách mật khẩu B8, khóa 15 phút sau 5 lần sai, chuẩn hóa email), `domain/errors.go`, test bảng `user_test.go`.
- **Lát 2 — application:** `application/authz` (ma trận quyền theo vai trò, CEO kế thừa quyền Admin + xem mọi dự án/Dashboard công ty — B2; `Can/Require/PermissionsOf`), `application/securetoken` (token ngẫu nhiên 32 byte + SHA-256), `application/auth` (port + use case Login/Refresh xoay vòng phát hiện dùng lại → thu hồi cả họ/Logout/Me/InspectActivation/Activate), `domain/page.go`, `domain/refresh_token.go`; truy vấn `ListUsers`/`CountUsers`; test với port giả lập (auth 91,6%, authz 100%).
- **Lát 3 — quản lý tài khoản + repository:** `application/account` (Admin/CEO tạo tài khoản, liệt kê có phân trang, gửi lại email kích hoạt; quyền `users.assign_ceo` chỉ CEO có; lỗi gửi mail khi tạo không làm hỏng việc tạo), chặn trang tối đa `MaxPage` để offset không tràn int32; `infrastructure/postgres` (`UserRepo`, `RefreshTokenRepo`, ánh xạ unique violation → `ErrEmailTaken`); integration test (tag `integration`, dùng `TEST_DATABASE_URL` hoặc testcontainers, mỗi test một schema tạm).
- **Lát 4 — adapter hạ tầng:** `infrastructure/security` (bcrypt, JWT HS256 kiểm tra độ dài secret ≥ 32 byte), `infrastructure/email` (SMTP gửi email kích hoạt, giờ hiển thị `Asia/Ho_Chi_Minh`) kèm test.
- **Lát 5 — HTTP API:** `transport/httpapi`: `POST /auth/login|refresh|logout|activation/verify|activation`, `GET /auth/me`, `GET|POST /users`, `POST /users/{id}/activation-email`; middleware `requireAuth` (Bearer) và `requirePermission` (gọi `authz.Require`); giới hạn tần suất token bucket theo IP (`RemoteAddr`) trả 429 + `Retry-After`; cookie refresh `wt_refresh` luôn `Secure; HttpOnly; SameSite=Strict; Path=/api/v1/auth`; bảng ánh xạ lỗi domain → HTTP duy nhất ở `writeError`; decode JSON chặn trường lạ, body > 1 MB, dữ liệu thừa. Test API `auth_api_test.go` (ma trận 401/403 đủ 4 vai trò × 3 endpoint `/users`, 7 trường hợp `/auth/me`, cookie, 423, 429).
- **Lát 6 — wiring + công cụ dev:** `cmd/api/config.go` (JWT_SECRET bắt buộc ≥ 32 byte, ACCESS_TOKEN_TTL, APP_BASE_URL, SMTP_*), `main.go` lắp repo/bcrypt/JWT/mailer vào use case; `cmd/seed` (4 tài khoản tên Việt giả, mật khẩu ngẫu nhiên 96 bit in ra màn hình, từ chối khi `APP_ENV=production`); `scripts/dev-local.sh`, `scripts/seed.sh`, target `make dev-local`, `make seed`; thêm dịch vụ Mailpit và biến auth vào `docker-compose.yml`.
- **Lát 7 — frontend nền:** `shared/api/http.ts` thêm `configureHttpAuth` (token chỉ trong bộ nhớ), tùy chọn `json`/`auth`, gắn Bearer, 401 → làm mới một lần rồi gửi lại, 204 → `undefined`, lỗi mạng → `ApiError` `NETWORK`; MSW (`src/test/server.ts`, `setup.ts` chặn request không có handler); chuỗi tiếng Việt cho đăng nhập/kích hoạt/menu trong `messages.ts`; test `http.test.ts` (7 ca).

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Test dùng chuỗi `"hash:matkhau123"` làm PasswordHash bị gosec G101 báo “hardcoded credentials” | AI viết literal trông giống mật khẩu trong test | `golangci-lint run` | Đổi sang hằng `testPassword` + hàm `fakeHash()` |
| Cookie refresh đặt `Secure: h.secureCookie` (tắt được qua cấu hình) bị gosec G124 báo | AI thêm cờ cấu hình "cho tiện dev" làm yếu quyết định B4 | `golangci-lint run` | Bỏ cờ, luôn `Secure: true` (trình duyệt coi `http://localhost` là ngữ cảnh an toàn); test dựng cookie bằng header thay cho `http.Cookie` |
| File `dto.go`, `auth_handler.go` chưa gofmt sau khi sửa bằng sed | Sửa máy móc không chạy formatter | `golangci-lint run` (gofmt) | `gofmt -w` |
| Gọi `server.listen({ onUnhandledRequest: 'error' })` theo API MSW v2 trong khi dự án dùng MSW 3.0.7 (đã đổi tên thành `onUnhandledFrame`) | AI mặc định API phiên bản cũ, chưa đối chiếu file `.d.ts` của bản cài | `tsc -b --noEmit` báo TS2353 | Đọc `node_modules/msw/lib/_chunks/shared-options.d.ts`, đổi sang `onUnhandledFrame: 'error'` |

## Kiểm chứng
<!-- Lệnh đã chạy và kết quả thật: make lint, make test, test nào, kiểm thử tay luồng nào. -->
- `golangci-lint run ./...` → 0 issues; `go test ./...` → tất cả gói ok (backend).
- Smoke test tay trên PostgreSQL + Mailpit Homebrew (B12), API chạy `:18080` với JWT secret sinh tạm: `make migrate-up`, `make seed` (4 tài khoản); đăng nhập admin 200 + cookie `Secure` path `/api/v1/auth`; `/auth/me` 200; `GET /users` trả `{items,page,pageSize,total}`; tạo tài khoản 201 và Mailpit nhận thư "Kích hoạt tài khoản WorkTrack"; refresh 200; member gọi `/users` → 403 `FORBIDDEN`; sai mật khẩu 5 lần → lần 5, 6 trả 423; logout 204; log API không chứa mật khẩu.
