# Tích hợp bước 02/9 — `chore/scaffold` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `7c8ac37` (cha: `52fd18c` = tag `integ/01-ai-workflow`, `00aa4fe` đầu nhánh). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 9 (`ad793e3` … `00aa4fe`), cộng merge commit `7c8ac37` |
| Thay đổi | 59 file, +6056 / −18 dòng (`git diff --stat 52fd18c..7c8ac37`), trong đó 4285 dòng là `frontend/package-lock.json` |
| Chức năng mang vào | Khung backend Go (chi, pgx pool, `/healthz` + `/readyz`, middleware request-id/log/recover/CORS, định dạng lỗi JSON, `config.go` đọc env), khung frontend React 19 + Vite + Tailwind v4 (theme theo design system, TanStack Query provider, router, trang Home có `HealthStatus`, trang 404), `docker-compose.yml` (postgres/api/web), 2 Dockerfile, CI GitHub Actions (backend, frontend, gitleaks, docker build), `.golangci.yml`, `sqlc.yaml`, thư mục `db/migrations`, `db/queries` rỗng; quyết định B12 (máy không Docker) và sửa nhỏ ở `scripts/auto-run.sh`, `.githooks/commit-msg`, `.claude/settings.json` |
| Chức năng nghiệp vụ CF | Chưa có (nền tảng) |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip |
|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/02-scaffold/after-merge`) | 10 | 9 | 1 | 0 |
| Sau dọn (`.ai-log/integration/02-scaffold/after-cleanup`) | 10 | 9 | 1 | 0 |

Chi tiết theo tầng (giống nhau ở hai lần chạy): backend API 5/5, backend Unit 2/3, frontend Unit/UI 2/2. Độ phủ domain + application: 0,0% (hai gói chỉ có `doc.go`). Số Pass không giảm.

Test lỗi: `internal/infrastructure/postgres :: TestNewPool_RealPostgres_Pings` (build tag `integration`)
- Lỗi: `rootless Docker not found, failed to create Docker provider` — test chỉ dùng testcontainers.
- Phân loại: **lỗi có sẵn ở `chore/scaffold`, không do gộp** (bước 01 không có Go; file test không đổi khi gộp). Theo B12 máy dev không có Docker, test này chỉ chạy được trong CI.
- Bản cuối `test/test-cases` đã sửa bằng `testDSN` (`TEST_DATABASE_URL` khi có) ở commit `762f62c` (`feature/auth`).
- Xử lý: **không sửa**; tạo issue [#12](https://github.com/BaoCool-dev/WorkTrack/issues/12) (đã tìm trùng: không có). Template `.github/ISSUE_TEMPLATE/bug_report.md` chưa có ở `release/v1` bước này → dùng nội dung template từ `test/test-cases`.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| Biến môi trường code đọc: `APP_ENV`, `HTTP_ADDR`, `DATABASE_URL` (bắt buộc), `CORS_ORIGIN` (backend); `VITE_API_BASE_URL` (frontend); compose thêm `POSTGRES_USER/PASSWORD/DB` | — | Đã liệt kê. **Không đối chiếu được với `.env.example`** vì quyền đọc `.env.*` bị chặn (xem f) | — |
| `docker-compose.yml`: biến khớp `config.go`, healthcheck api dùng `/readyz` (có route), web dùng `wget` (có trong `nginx:alpine`), api image có cài `wget` | — | Hợp lệ, không sửa | — |
| CI `ci.yml`: `go-version-file: backend/go.mod` (go 1.27 = Dockerfile `golang:1.27`); Node 24 khớp `engines.node >=22.22` và Dockerfile `node:24`; script `build`, `lint`, `typecheck`, `format:check`, `test` đều có trong `package.json` | — | Khớp, không sửa | — |
| CI chạy `go test -tags=integration` cần Docker | Thông tin | Đúng thiết kế (runner GitHub có Docker) | — |
| `Makefile`: target `tools` → `scripts/setup-dev.sh` có thật; các target khác trỏ `backend/`, `frontend/` đã tồn tại | — | Không sửa | — |
| `Makefile` chưa có `make dev-local` mà B12 và `scripts/auto-run.sh` nhắc | Thấp | Giữ — B12 ghi áp dụng "auth trở đi"; bản cuối có `dev-local` → `scripts/dev-local.sh` (dùng ở bước sau) | — |
| `make sqlc`: `sqlc generate` báo `no queries contained in paths .../db/queries` | Thông tin | Đúng ở bước này (`db/queries` chỉ có `.gitkeep`); query có ở bước sau. Không có code sinh nên không có khác biệt | — |
| `go mod tidy` | — | Không tạo khác biệt `go.mod`/`go.sum` | — |
| `npm install --package-lock-only` | — | Không tạo khác biệt `package-lock.json` | — |
| Migration goose | — | Chưa có migration; `goose -dir backend/db/migrations validate` không lỗi | — |
| `.gitignore` | — | Phủ `backend/coverage.out`, `frontend/dist/`, `node_modules/`, `*.tsbuildinfo`, `frontend/coverage/`, Playwright; `git status` sạch sau build/test | — |
| `.claude/settings.json`, `.mcp.json` | — | JSON hợp lệ | — |
| `gitleaks detect` (87 commit, ~2,3 MB) | — | no leaks found | — |
| `golangci-lint run` (gồm `unused`) | — | 0 issues | — |

Không có commit `fix(config)` ở bước này.

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| `deadcode -test ./...` | Kết quả rỗng (`deadcode.txt` rỗng, chạy lại cũng rỗng) | Không có gì | — |
| devDependency `@testing-library/user-event` (knip: unused) | `git grep user-event test/test-cases` → dùng ở `app.test.tsx`, `AdminOrgView.test.tsx`, `KanbanBoard.test.tsx`… | Giữ vì dùng ở bước sau | — |
| Export `ApiError` (`shared/api/http.ts`) | Bản cuối dùng ở nhiều component `features/admin/*` | Giữ vì dùng ở bước sau | — |
| Export `healthKeys`, `fetchHealth`, `useHealth`, type `HealthResponse` (`features/health`) | `git diff HEAD test/test-cases -- frontend/src/features/health` rỗng: bản cuối giống hệt, vẫn export như vậy; `useHealth` dùng trong `HealthStatus.tsx` | Giữ vì lý do khác: mẫu API công khai của feature (query key để invalidate) theo skill `react-feature`, bản cuối giữ nguyên — xóa chỉ tạo khác biệt vô ích (xem f) | — |
| TODO/FIXME/XXX/HACK, `console.log`, `fmt.Print*` | `git grep` trong `backend frontend/src scripts Makefile docker-compose.yml .github`: 0 kết quả | — | — |
| Code bị comment-out | `git grep` mẫu `// func|if|return|…`: 0 kết quả | — | — |
| `internal/domain/doc.go`, `internal/application/doc.go` (gói rỗng) | Chỉ có comment package | Giữ vì dùng ở bước sau (chứa entity/use case từ `feature/auth`) | — |
| Trang giữ chỗ `HomePage.tsx` (comment "Trang tạm của chore/scaffold") | Bản cuối vẫn có `HomePage` trong `routes.tsx`, vẫn hiện `HealthStatus` | Giữ vì dùng ở bước sau | — |
| Comment đầu `Makefile` "thư mục backend/ và frontend/ sẽ có khi scaffold" đã lỗi thời | Bản cuối vẫn giữ nguyên dòng này | Giữ vì lý do khác: chỉ là comment, bản cuối không đổi; sửa ở đây không có lợi (xem f) | — |
| `scripts/auto-run.sh` nhắc `make dev-local` chưa có | Bản cuối có target `dev-local` | Giữ vì dùng ở bước sau | — |

Không có commit `chore(cleanup)` ở bước này → không có gì phải revert.

## (e) Việc người dùng tự kiểm

```bash
git diff integ/01-ai-workflow..HEAD --stat
git diff integ/01-ai-workflow..HEAD -- backend/
git diff integ/01-ai-workflow..HEAD -- frontend/src frontend/package.json
git diff integ/01-ai-workflow..HEAD -- docker-compose.yml .github/ Makefile .claude/ .githooks/ scripts/
make lint && make test
# PostgreSQL Homebrew đang chạy, DATABASE_URL trong .env:
cd backend && DATABASE_URL=postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable go run ./cmd/api
cd frontend && npm run dev
```
- `make dev-local` **chưa có ở bước này** (có từ `feature/auth`) → chạy API và web bằng hai lệnh trên.
- Mở `http://localhost:5173/`: trang Home hiện tên app và khối trạng thái `/healthz` (xanh khi API chạy, báo lỗi tiếng Việt khi tắt API). Mở một đường dẫn bất kỳ không tồn tại → trang 404.
- `curl -i localhost:8080/healthz` → 200; `curl -i localhost:8080/readyz` → 200 khi Postgres chạy, 503 khi tắt Postgres.
- Chưa có đăng nhập nên chưa có vai trò để thử.
- Xem issue #12 và đóng khi bước tích hợp `feature/auth` gộp xong và test ping xanh.

## (f) Không chắc chắn — để người dùng quyết

1. **`.env.example` vẫn chưa được rà** (giống bước 01): quyền chặn đọc `.env.*`. Người dùng kiểm `.env.example` có đủ `APP_ENV`, `HTTP_ADDR`, `DATABASE_URL`, `CORS_ORIGIN`, `VITE_API_BASE_URL`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` (và `CONTEXT7_API_KEY`, `WORKTRACK_MCP_DATABASE_URI` từ bước 01). Mặc định trong code: `HTTP_ADDR=:8080`, `CORS_ORIGIN=http://localhost:5173`, `VITE_API_BASE_URL=http://localhost:8080`.
2. **Tạo issue #12 cho lỗi đã được sửa ở nhánh sau.** Làm theo đúng quy trình (lỗi có sẵn → issue, không sửa), nhưng có thể coi là nhiễu; người dùng có thể đóng ngay hoặc giữ tới khi gộp `feature/auth`.
3. **Các export health không dùng** (`healthKeys`, `fetchHealth`, `HealthResponse`, `useHealth` qua barrel): AI giữ vì bản cuối giữ nguyên. Nếu muốn knip sạch, có thể bỏ `export` ở bản cuối thay vì ở bước này.
4. Comment lỗi thời ở dòng 2 `Makefile`: nên sửa ở nhánh chính (bản cuối) nếu muốn, không sửa ở bước tích hợp.
5. `.github/ISSUE_TEMPLATE/` chưa có ở bước này; issue #12 dùng template lấy từ `test/test-cases`.
