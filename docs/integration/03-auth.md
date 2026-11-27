# Tích hợp bước 03/9 — `feature/auth` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `b6acf23` (cha: `1957583` = tag `integ/02-scaffold`, `a8ffe51` đầu nhánh). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 17 (`8276014` … `a8ffe51`), cộng merge commit `b6acf23` |
| Thay đổi | 133 file, +8655 / −103 dòng (`git diff --stat 1957583..b6acf23`), gồm ~750 dòng `package-lock.json`, ~600 dòng code sinh sqlc và 10 ảnh bằng chứng |
| Chức năng mang vào | **CF-01 Đăng nhập & tài khoản**: migration `create_auth_tables` (tài khoản, vai trò, refresh token); domain `User`, `RefreshToken`, phân trang; use case `auth` (đăng nhập, làm mới xoay vòng, đăng xuất, kích hoạt, khóa đăng nhập nguyên tử), `account` (quản lý tài khoản), `authz`; adapter bcrypt, JWT HS256, SMTP (Mailpit); API `/api/v1/auth/*`, `/me`, quản lý user, rate limit; `cmd/seed` (4 tài khoản theo vai trò, B11); FE: http client gắn Bearer + tự làm mới khi 401, `AuthProvider`, route guard, `AppLayout`/`Sidebar`/`Topbar` theo vai trò, trang Đăng nhập, Kích hoạt, `PlaceholderPage`; `shared/ui` (Button, Input, Badge, Avatar, Dialog, Toast, EmptyState, ErrorState, Skeleton); `make dev-local`, `make seed`, `make test-integration`, `format:check` trong `make lint`; service `mailpit` trong compose; báo cáo gate, ma trận truy vết, nhật ký AI |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip |
|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/03-auth/after-merge`) | 228 | 228 | 0 | 0 |
| Sau dọn (`.ai-log/integration/03-auth/after-cleanup`) | 228 | 228 | 0 | 0 |

Chi tiết (giống nhau ở hai lần): backend API 67, backend Unit 86, frontend Unit/UI 75. Độ phủ domain + application: 94,0%. Số Pass không giảm (bước 02: 9/10).

- Test tích hợp Postgres (`-tags=integration`) chạy được ở máy không Docker nhờ `TEST_DATABASE_URL` (commit `762f62c`): `TestNewPool_RealPostgres_Pings`, `TestUserRepo_*`, `TestRefreshTokenRepo_*` đều PASS.
- Lỗi của issue [#12](https://github.com/BaoCool-dev/WorkTrack/issues/12) (bước 02) đã hết → đã comment vào issue; **chưa đóng**, để người dùng đóng.
- Không có test lỗi → không có commit `fix(integration)`.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| `.env.example` lệch code: `SMTP_USER` ≠ `SMTP_USERNAME`, `MAIL_FROM` ≠ `SMTP_FROM` (code đọc) | Trung bình | **Không sửa được**: `.claude/settings.json` chặn `Edit(./.env*)` (CLAUDE.md quy tắc 10). Tạo issue [#13](https://github.com/BaoCool-dev/WorkTrack/issues/13) (đã tìm trùng: không có). Bản cuối `test/test-cases` còn cùng lệch | — |
| `.env.example` thiếu `APP_BASE_URL` (link kích hoạt B7), `TEST_DATABASE_URL` (B12), `POSTGRES_USER/PASSWORD/DB` (compose), `CONTEXT7_API_KEY`, `WORKTRACK_MCP_DATABASE_URI` (`.mcp.json`) | Thấp | Ghi vào issue #13 (đều có mặc định hoặc không bắt buộc) | — |
| `.env.example` thừa `REFRESH_TOKEN_TTL` (code dùng hằng `domain.RefreshTokenTTL` = 7 ngày, B4) | Thấp | Ghi vào issue #13 | — |
| `GEMINI_API_KEY`, `GEMINI_MODEL`, `AI_TIMEOUT` chưa ai đọc | — | Giữ — dùng ở `feature/ai` (CF-12) | — |
| Cách đối chiếu `.env.example` | Thông tin | Chỉ lấy **tên biến** bằng `git show HEAD:.env.example \| grep -oE '^[A-Z_]+='`, không xem giá trị; không đọc `.env` | — |
| `docker-compose.yml`: thêm `mailpit` (SMTP 1025, UI 8025), api nhận `JWT_SECRET` bắt buộc (`:?`), `APP_BASE_URL`, `SMTP_*` khớp `config.go`; `SMTP_PORT` 1025 khớp mặc định | — | Hợp lệ. `docker compose` sẽ báo lỗi nếu `.env` thiếu `JWT_SECRET` — đúng ý đồ | — |
| CI `ci.yml`: `go-version-file: backend/go.mod` (go 1.27 = `golang:1.27-alpine`), Node 24 khớp `engines >=22.22` và `node:24-alpine`; script `build/lint/typecheck/format:check/test` có trong `package.json`; integration test dùng testcontainers trên runner | — | Khớp, không sửa | — |
| `Makefile`: `dev-local` → `scripts/dev-local.sh`, `seed` → `scripts/seed.sh` (có thật, mode 100755); mọi `make <target>` nhắc trong tài liệu/script đều tồn tại; không script nào trỏ tới file không có | — | Không sửa | — |
| `go mod tidy` | — | Không tạo khác biệt | — |
| `npm install --package-lock-only` | — | Không tạo khác biệt lockfile | — |
| `sqlc generate` | — | Không tạo khác biệt `sqlcgen/` | — |
| Goose: 1 migration `20261119092421_create_auth_tables.sql`; `goose -dir backend/db/migrations validate` | — | Hợp lệ, không trùng | — |
| `.gitignore` | — | `git status` sạch sau build + 2 lần chạy test | — |
| `.claude/settings.json`, `.mcp.json` | — | JSON hợp lệ | — |
| `gitleaks detect` (lịch sử, gồm dải `1957583..HEAD`) | — | no leaks found | — |
| `gitleaks dir .` (cả file không track) | Thông tin | 20 phát hiện, **toàn bộ ở file không track/bị ignore**: `.env` (1), `.ai-log/runs/*.jsonl` (19 — JWT/token trong log thô của phiên test). Không vào repo; xem (f) | — |
| `golangci-lint run` (gồm `unused`), `eslint`, `prettier --check` | — | 0 issues / sạch | — |

Không có commit `fix(config)` ở bước này.

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| `deadcode` | `deadcode.txt` rỗng | Không có gì | — |
| knip: dependency, devDependency, file không dùng | `knip --include dependencies,files,devDependencies`: 0 | Không có gì | — |
| knip: `useCan` (export), type `Permission`, `BadgeTone` | Bản cuối dùng ở `features/admin/*`, `features/projects/*` | Giữ vì dùng ở bước sau | — |
| knip: `useSession`, `useCanAccess`, `ForbiddenState` (barrel), `healthKeys`, `fetchHealth`, `useHealth` (barrel), `TOAST_DURATION_MS`, type `CurrentUser` (barrel), `HealthResponse`, `ButtonProps`, `DialogProps`, `EmptyStateProps`, `ErrorStateProps`, `InputProps` | Đều được dùng **trong chính module**, chỉ `export` là thừa; bản cuối vẫn export y như vậy (`git grep -w <tên> test/test-cases -- frontend/src`) | Giữ vì lý do khác: API công khai của feature/ui-kit theo skill `react-feature`; bỏ `export` ở đây làm `release/v1` lệch bản cuối và dễ xung đột ở các bước gộp sau (các nhánh sau sửa `shared/ui`, `features/auth`) — xem (f) | — |
| `fmt.Println/Printf` trong `backend/cmd/seed/main.go` | Đầu ra CLI có chủ đích (B11: in tài khoản mẫu) | Giữ vì lý do khác (không phải gỡ lỗi) | — |
| Dòng seed "Dự án mẫu sẽ được thêm ở nhánh feature/projects." | Bản cuối thay bằng in số dự án mẫu | Giữ vì dùng ở bước sau (nhánh `feature/projects` thay) | — |
| `PlaceholderPage` (trang giữ chỗ cho mục menu chưa làm) | Bản cuối vẫn dùng làm fallback trong `routes.tsx` | Giữ vì dùng ở bước sau | — |
| TODO/FIXME/XXX/HACK, `console.log/debug` | `git grep` trong `backend frontend/src scripts Makefile docker-compose.yml .github` (trừ sqlcgen): 0 | — | — |
| Code bị comment-out | `git grep` mẫu `// func|if|return|const|import|export…`: 0 | — | — |
| Comment lỗi thời dòng 2 `Makefile` (từ bước 02) | Bản cuối giữ nguyên | Giữ vì lý do khác (chỉ là comment) | — |

Không có commit `chore(cleanup)` → không có gì phải revert.

## (e) Việc người dùng tự kiểm

```bash
git diff integ/02-scaffold..HEAD --stat
git diff integ/02-scaffold..HEAD -- backend/ ':!backend/internal/infrastructure/postgres/sqlcgen'
git diff integ/02-scaffold..HEAD -- frontend/src frontend/package.json
git diff integ/02-scaffold..HEAD -- docker-compose.yml Makefile scripts/ docs/decisions.md
make lint && make test && make test-integration   # integration cần TEST_DATABASE_URL trong môi trường
make migrate-up && make seed                      # in 4 tài khoản mẫu
make dev-local                                    # cần PostgreSQL + Mailpit (Homebrew) và .env có JWT_SECRET ≥ 32 byte
```
- Mở `http://localhost:5173/login`: đăng nhập đúng/sai mật khẩu (thông báo tiếng Việt), sai nhiều lần → bị khóa tạm.
- Thử **cả 4 vai trò** từ `make seed`: Admin (menu quản trị tài khoản), Manager, Member, CEO — menu Sidebar khác nhau; mở đường dẫn không có quyền → trang "không có quyền" (`ForbiddenState`).
- Admin tạo tài khoản → mở Mailpit `http://localhost:8025` → link kích hoạt → đặt mật khẩu → mở lại link → báo link đã dùng.
- Để tab mở > 15 phút (hoặc giảm `ACCESS_TOKEN_TTL=1m`) → thao tác vẫn chạy nhờ tự làm mới token; Đăng xuất → quay lại `/login`.
- Sửa `.env.example` theo issue #13 rồi đối chiếu với `.env` của mình (đổi `SMTP_USER`→`SMTP_USERNAME`, `MAIL_FROM`→`SMTP_FROM`, thêm `APP_BASE_URL`, `TEST_DATABASE_URL`).
- Đóng issue #12 nếu đồng ý.

## (f) Không chắc chắn — để người dùng quyết

1. **`.env.example` không được sửa** dù bước này yêu cầu: settings chặn Edit `.env*`; AI không vượt chặn bằng shell. Đã ghi đủ chênh lệch vào issue #13. Nên sửa ở nhánh chính để bản cuối cũng đúng. Nếu `.env` của người dùng đang dùng `SMTP_USER`/`MAIL_FROM` thì SMTP có xác thực và địa chỉ gửi hiện **không có hiệu lực** (Mailpit không cần xác thực nên dev vẫn chạy).
2. **`REFRESH_TOKEN_TTL` thừa**: AI đề xuất xóa khỏi `.env.example` (B4 chốt 7 ngày, code dùng hằng); hoặc cho `config.go` đọc biến này — người dùng chọn.
3. **Các export không dùng (knip)**: AI giữ vì bản cuối giữ nguyên. Nếu muốn knip sạch, bỏ `export` ở bản cuối hoặc cấu hình knip bỏ qua barrel `index.ts`.
4. **`.ai-log/runs/*.jsonl` chứa JWT/token** của phiên test (gitleaks dir). Đã ignore, không vào repo, nhưng là token dev thật — nên xóa log cũ hoặc đổi `JWT_SECRET` dev nếu từng chia sẻ thư mục này.
5. **Comment vào issue #12** (đầu ra hướng ngoài, nhẹ) để ghi nhận lỗi đã hết; AI không tự đóng issue.
6. `.github/ISSUE_TEMPLATE/` chưa có ở `release/v1` bước này; issue #13 dùng template lấy từ `test/test-cases` (giống bước 02).
