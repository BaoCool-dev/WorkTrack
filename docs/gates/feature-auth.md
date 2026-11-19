# Gate: feature/auth — CF-01 (đăng nhập, kích hoạt tài khoản, phân quyền 4 vai trò; quy tắc B2, B4, B7, B8, B9, B11)
**Ngày:** 2026-11-20 · **Commit cuối (code):** 9ecee98 · **Kết luận:** ĐẠT (kèm ghi chú — xem "Còn tồn đọng")

Phạm vi: bảng `app_user`, `refresh_token`, `activation_token` (goose + sqlc); authz một chỗ cho 4 vai trò (CEO kế thừa Admin);
use case đăng nhập / làm mới xoay vòng / đăng xuất / kích hoạt / tạo và liệt kê người dùng; adapter bcrypt, JWT HS256, SMTP;
API `/auth/*`, `/me`, `/users`; giới hạn tần suất; `make seed`. FE: http client tự làm mới token, AuthProvider (access token chỉ trong
bộ nhớ), trang đăng nhập + kích hoạt, route guard, AppShell và menu theo vai trò, `shared/ui` nền.

Gate chạy 3 vòng: vòng 1 (go-reviewer, ui-reviewer, qa-tester) → sửa ở 143ee32, 5a04d55, ceb77a5; vòng 2 (review lại) → sửa ở 9ecee98; vòng 3 chạy lại toàn bộ, không còn mục Chặn.

## Tự động
Môi trường: không có Docker. PostgreSQL và Mailpit chạy bằng Homebrew, app chạy bằng `make dev-local`. Integration test chạy
trên PostgreSQL local, `TEST_DATABASE_URL` trỏ tới CSDL dev; mỗi lần chạy dùng một schema tạm `it_*`.

| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint + eslint + `tsc -b`) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: 144 test pass ở mọi gói · FE: 75/75 test pass |
| `go test -tags integration ./internal/infrastructure/postgres/` (PostgreSQL local) | ✅ ok — có test 10 goroutine sai mật khẩu song song (B8) và test xoay vòng refresh token |
| `make cover` (gói cốt lõi domain + application) | ✅ 94,0% |
| `gitleaks detect --no-banner` | ✅ no leaks found (25 commit) |
| `python3 scripts/ai-log/ai_log.py check` | ✅ OK (6 entry, 30 lỗi/ảo giác đã ghi); cảnh báo còn lại: mục "Phần tôi đã sửa" chưa điền ở các entry |
| `docker build`, `make test-integration` (testcontainers) | ⏳ chạy trong CI của PR — xem mục "CI" |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Xác thực | Không token, token hết hạn, sai chữ ký, `alg: none` → 401; secret < 32 byte bị từ chối khi khởi động | ✅ | `TestMe_AuthMatrix_CF01`, `TestJWT_Verify_RejectsBadTokens_CF01_B4`, `TestNewJWT_ShortSecret_CF01_B4` |
| Phân quyền 4 vai trò (B2) | Ma trận Admin/Manager/Member/CEO trên `/users`; CEO kế thừa Admin; Admin không có quyền tạo dự án | ✅ | `TestUsersEndpoints_RoleMatrix_CF01`, `TestCan_Matrix_CF01_B2`, `TestCan_CEOInheritsAdmin_CF01_B2`, `TestCan_AdminCannotCreateProject_CF01_R` |
| IDOR | Chưa có tài nguyên theo dự án; `/me` chỉ trả dữ liệu của chủ token; tài khoản bị vô hiệu sau đăng nhập → 401 | ✅ / N/A | `TestMe_AccountDisabledAfterLogin_Returns401_CF01` |
| Refresh token (B4) | Cookie `HttpOnly; Secure; SameSite`; xoay vòng; dùng lại token cũ → thu hồi cả họ; 2 request song song chỉ 1 thắng | ✅ | `TestRefresh_ReuseRevokesFamily_CF01_B4`, `TestRefresh_ConcurrentUseRevokesFamily_CF01_B4`, `TestRefreshTokenRepo_Rotation_CF01_B4` |
| Khóa đăng nhập (B8) | 5 lần sai → khóa 15 phút (423); đếm nguyên tử khi song song; sai khi đang khóa không cộng dồn, không gia hạn | ✅ | `TestUserRepo_RegisterLoginFailure_Concurrent_CF01_B8`, `TestLogin_WrongPasswordWhileLockedConcurrently_ReturnsLocked_CF01_B8` |
| Không lộ email | Email không tồn tại, sai mật khẩu, user bị xóa giữa chừng → cùng một lỗi chung | ✅ | `TestLogin_UnknownEmail_GenericError_CF01`, `TestLogin_UserDeletedDuringWrongPassword_GenericError_CF01` |
| Giới hạn tần suất | Đăng nhập, kích hoạt, làm mới → 429 + `Retry-After` | ✅ | `TestLogin_TooManyRequests_Returns429WithRetryAfter_CF01`, `TestRateLimiter_RefillsOverTime_CF01` |
| Token kích hoạt (B7) | Lưu hash, 72 giờ, dùng một lần, gửi lại thì token cũ hết hiệu lực; token gửi trong body, xóa khỏi URL ở FE | ✅ | `TestActivate_Expired72h_CF01_B7`, `TestResendActivation_NewTokenInvalidatesOld_CF01_B7`, app "token hợp lệ → xóa token khỏi URL…" |
| SQL / log / lỗi 500 | Mọi truy vấn qua sqlc; log không ghi token hay mật khẩu; lỗi 500 trả câu chung, không lộ lỗi pgx | ✅ | đọc mã + go-reviewer; `errors.go` một chỗ |
| Validate | Thiếu trường, chuỗi rỗng, khoảng trắng, email sai, mật khẩu < 8 rune hoặc thiếu chữ/số, mật khẩu có dấu tiếng Việt, UUID sai → 400; pageSize > 100 / tham số sai → 400; trùng email → 409. FE và BE dùng cùng luật | ✅ | `TestValidatePassword_Policy_CF01_B8`, `TestCreateUser_Validation_CF01`, `TestListUsers_BadPageParam_Returns400_CF01`, `TestResendActivation_BadUUID_Returns400_CF01`, `schemas.test.ts` |
| UX — luồng theo vai trò | Đăng nhập Admin/Manager/Member/CEO → menu đúng vai trò; vào trang không có quyền → trạng thái "không có quyền"; kích hoạt: form, lỗi validate, thành công, liên kết đã dùng | ✅ | Playwright MCP ở 1366×768, ảnh `docs/testing/evidence/feature-auth/01…10-*.png`; console chỉ có các 401/400 dự kiến |
| UX — trạng thái | Skeleton khi kiểm tra phiên; lỗi server hiện ở `FormAlert`; nút khóa khi đang gửi (không gửi trùng); toast khi đăng xuất; 404 trong app shell; một `h1`; Enter gửi form, có focus ring | ✅ | `app.test.tsx`, ảnh minh chứng |
| Agent | go-reviewer: không còn mục Chặn · ui-reviewer: đã sửa các mục (AuthCard lồng, câu lặp, favicon) · qa-tester: ma trận truy vết | ✅ | `docs/testing/traceability-matrix.md` (21 dòng CF-01) |

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | Đếm lần sai mật khẩu kiểu đọc-sửa-ghi: nhiều request song song vượt quá 5 lần mà không khóa (B8) | Cao | 143ee32 |
| 2 | Lần sai đến khi tài khoản đang khóa vẫn cộng dồn bộ đếm và gia hạn `locked_until`; user bị xóa giữa chừng làm endpoint đăng nhập trả 404 (lộ thông tin) | Trung bình | 9ecee98 |
| 3 | Test integration để sót schema tạm; test ping cần Docker mới chạy được | Thấp | 762f62c |
| 4 | Trang đăng nhập/kích hoạt có `AuthCard` lồng nhau, câu chữ lặp, thiếu favicon (ui-reviewer) | Thấp | 5a04d55 |
| 5 | Giả định và việc hoãn sau review chưa ghi vào `docs/decisions.md` mục C | Thấp | ceb77a5 |

## Còn tồn đọng
- **Người dùng cần làm:**
  - Điền mục "Phần tôi đã sửa / hoàn thiện" trong `docs/ai-log/entries/2026-11-19-05-feature-auth.md` (đang để `[NGƯỜI DÙNG ĐIỀN]`).
  - Thêm biến mới vào `.env.example` / `.env`: `JWT_SECRET` (≥ 32 byte, sinh bằng `openssl rand -hex 32`), `ACCESS_TOKEN_TTL`, `APP_BASE_URL`, `SMTP_*` (Claude không được sửa `.env*`; xem mục C ngày 2026-11-19).
  - Đưa 13 artboard vào `docs/design/artboards/`. Nhánh này dựng theo `design-system.md` §4–§8 vì thư mục chưa có.
  - CSDL dev `worktrack` còn sót 4 schema `it_*` từ các lần chạy integration trước bản sửa 762f62c. Claude không xóa dữ liệu; nếu muốn dọn: `DROP SCHEMA it_… CASCADE`.
- **Đã hoãn và ghi ở `docs/decisions.md` mục C (2026-11-20):**
  - xoay vòng token chưa bọc `TxManager`;
  - chưa có thời gian ân hạn khi nhiều tab cùng làm mới token;
  - bẫy focus trong `Dialog`, `disabledReason`, nút "Hoàn tác";
  - SMTP chưa bắt buộc TLS;
  - chưa có job dọn `refresh_token`;
  - mã 423 cho phép đoán email có tồn tại;
  - rate limit chưa gom IPv6 theo /64 và chưa dọn bucket cũ;
  - màn tạo tài khoản thuộc CF-02.
- **Gợi ý nhỏ (go-reviewer, không chặn):**
  - nhánh sai mật khẩu ghi DB thêm một lần, nên thời gian phản hồi chênh nhẹ so với nhánh email không tồn tại (bcrypt vẫn chiếm phần lớn);
  - fake `Activate` trong test bỏ qua tham số `now`.
- **Khoảng trống test:** chưa có E2E Playwright tự động; luồng UI đã đi tay bằng Playwright MCP.

## CI
Sẽ cập nhật sau khi push và CI của PR chạy xong.
