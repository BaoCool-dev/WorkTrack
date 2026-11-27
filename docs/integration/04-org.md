# Tích hợp bước 04/9 — `feature/org` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `fc0ba3a` (cha: `3fb89cb` = tag `integ/03-auth`, `51e47a9` đầu nhánh). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 8 (`23dc7d0` … `51e47a9`), cộng merge commit `fc0ba3a` |
| Thay đổi | 73 file, +6278 / −77 dòng (`git diff --stat 3fb89cb..fc0ba3a`), gồm ~660 dòng code sinh sqlc, ~1170 dòng test và 2 ảnh bằng chứng |
| Chức năng mang vào | **CF-02 Cơ cấu tổ chức & hồ sơ nhân sự**: migration `20261119192122_create_org_tables` (phòng ban dạng cây, chức danh, kỹ năng, `user_skill`); query sqlc `department/job_title/skill/user_skill`, mở rộng `app_user` (lọc, phân trang, CTE đệ quy theo phòng ban con); domain `org.go`; use case `org` (CRUD phòng ban/chức danh/kỹ năng, chặn vòng lặp cây, chặn xóa khi còn tham chiếu — 409) và mở rộng `account` (hồ sơ, kỹ năng ≤ 50, không tự đổi vai trò, chỉ CEO sửa CEO); API `/api/v1/departments`, `/job-titles`, `/skills`, `GET/PUT /users/{id}`, `PUT /users/{id}/skills`; seed phòng ban/chức danh/kỹ năng mẫu; FE: màn **Nhân sự & phòng ban** (`features/admin`: bảng lọc/phân trang, form tạo/sửa, dialog kỹ năng, cây phòng ban, danh mục), `shared/ui` thêm `Select`, `Pagination`, chuyển `FormAlert` từ `features/auth` sang `shared/ui`, `shared/lib/messages.ts`, `format.ts`; báo cáo gate, ma trận truy vết, 10 giả định mục C trong `docs/decisions.md` |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip |
|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/04-org/after-merge`) | 513 | 513 | 0 | 0 |
| Sau dọn (`.ai-log/integration/04-org/after-cleanup`) | 513 | 513 | 0 | 0 |

Chi tiết (giống nhau ở hai lần): backend API 216, backend Unit 190, frontend Unit/UI 107. Độ phủ domain + application: 92,4% (bước 03: 94,0% — giảm nhẹ do thêm gói `org`, vẫn trên ngưỡng). Số Pass không giảm (bước 03: 228/228).

- Gồm cả test tích hợp Postgres (`-tags=integration`) của `org_integration_test.go`.
- Không có test lỗi → không có commit `fix(integration)`, không tạo issue mới.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| Biến môi trường | — | Bước này **không** thêm biến nào (không đổi `config.go`, `docker-compose.yml`; `main.go` chỉ thêm wiring `org.NewService`). Chênh lệch `.env.example` cũ vẫn mở ở issue [#13](https://github.com/BaoCool-dev/WorkTrack/issues/13) (OPEN), không có gì mới để ghi thêm | — |
| `docker-compose.yml`, `.github/workflows`, `Makefile`, `scripts/`, `package.json`, `go.mod`, `.gitignore`, `.claude/`, `.mcp.json` | — | Không file nào đổi ở bước này (`git diff --stat 3fb89cb..HEAD -- <các đường dẫn>` rỗng); kết luận bước 03 vẫn đúng | — |
| `go mod tidy` | — | Không tạo khác biệt | — |
| `npm install --package-lock-only` | — | Không tạo khác biệt lockfile | — |
| `sqlc generate` (`backend/sqlc.yaml`) | — | Không tạo khác biệt `sqlcgen/` | — |
| Goose: `20261119092421_create_auth_tables.sql`, `20261119192122_create_org_tables.sql`; `goose -dir backend/db/migrations validate` | — | rc=0, tên tăng dần, không trùng | — |
| `.gitignore` | — | `git status` sạch sau sqlc, tidy và 2 lần chạy test | — |
| `.claude/settings.json`, `.mcp.json` | — | JSON hợp lệ | — |
| `gitleaks detect` | — | no leaks found | — |
| `golangci-lint run ./...` (gồm `unused`) | — | 0 issues | — |

Không có commit `fix(config)` ở bước này.

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| `deadcode` | `deadcode.txt` rỗng | Không có gì | — |
| knip: dependency, devDependency, file không dùng | `knip.txt` chỉ có mục export/type | Không có gì | — |
| knip mới ở bước này: `adminKeys` (`admin/api/queries.ts`), interface `Ref` (`admin/model/types.ts`), type `PaginationProps`, `SelectProps` (`shared/ui/index.ts`) | Đều được dùng **trong chính module**; bản cuối `test/test-cases` giữ nguyên y hệt (`git grep -nw <tên> test/test-cases -- frontend/src`) | Giữ vì lý do khác: API công khai của feature/ui-kit (skill `react-feature`); bỏ `export` làm lệch bản cuối, dễ xung đột ở bước gộp sau | — |
| knip còn lại (`useSession`, `useCanAccess`, `ForbiddenState`, `healthKeys`, `fetchHealth`, `useHealth`, `TOAST_DURATION_MS`, `CurrentUser`, `HealthResponse`, `ButtonProps`, `DialogProps`, `EmptyStateProps`, `ErrorStateProps`, `InputProps`) | Đã xét ở bước 03 | Giữ (như bước 03) | — |
| `fmt.Println/Printf` trong `backend/cmd/seed/main.go` (thêm dòng in số phòng ban/chức danh/kỹ năng) | Đầu ra CLI có chủ đích (B11) | Giữ vì lý do khác (không phải gỡ lỗi) | — |
| Dòng seed "Dự án mẫu sẽ được thêm ở nhánh feature/projects." | Bản cuối thay bằng in số dự án mẫu | Giữ vì dùng ở bước sau | — |
| `PlaceholderPage` | Còn là fallback trong `routes.tsx` (bản cuối vẫn dùng) | Giữ vì dùng ở bước sau | — |
| `FormAlert` chuyển từ `features/auth/components` sang `shared/ui` | `git grep 'auth/components/FormAlert'`: 0 tham chiếu cũ | Không có gì | — |
| TODO/FIXME/XXX/HACK, `console.log/debug` | `git grep` trong `backend frontend/src scripts Makefile docker-compose.yml .github` (trừ sqlcgen): 0 | — | — |
| Code bị comment-out | `git grep` mẫu `// func|if|return|const|import|export|let|var|for|await…`: 0 | — | — |

Không có commit `chore(cleanup)` → không có gì phải revert.

## (e) Việc người dùng tự kiểm

```bash
git diff integ/03-auth..HEAD --stat
git diff integ/03-auth..HEAD -- backend/ ':!backend/internal/infrastructure/postgres/sqlcgen'
git diff integ/03-auth..HEAD -- backend/db/migrations backend/db/queries
git diff integ/03-auth..HEAD -- frontend/src
git diff integ/03-auth..HEAD -- docs/decisions.md      # 10 giả định mục C của feature/org
make lint && make test && make test-integration
make migrate-up && make seed                           # in thêm số phòng ban/chức danh/kỹ năng
make dev-local
```
- Đăng nhập **Admin** → mở menu Nhân sự & phòng ban: lọc theo phòng ban cha (phải gồm cả phòng ban con), tìm kiếm, phân trang; tạo nhân sự trùng email → "Email đã tồn tại"; sửa hồ sơ (email chỉ đọc); gán kỹ năng; đổi tab rồi tải lại trang (giữ `?tab=`).
- Tab phòng ban: tạo cây, đổi cha thành chính con của nó → bị chặn; xóa phòng ban còn con/còn người → bị chặn (409, thông báo tiếng Việt).
- Đăng nhập **CEO**: vào được màn này và **sửa được** (xem f.1). Thử sửa tài khoản CEO bằng Admin → bị chặn; tự đổi vai trò của mình → bị chặn.
- Đăng nhập **Manager**, **Member**: không thấy menu; gọi thẳng `GET /api/v1/departments` → 403.

## (f) Không chắc chắn — để người dùng quyết

1. **CEO có quyền ghi cơ cấu tổ chức** (`authz.ManageOrg` gồm Admin và CEO; route POST/PUT/DELETE `/departments`, `/job-titles`, `/skills`). CLAUDE.md quy tắc 2 chỉ nói Dashboard tổng quan của CEO là chỉ xem, nên AI không coi là vi phạm và không sửa; nhưng nếu ý đồ là "CEO chỉ xem" ở mọi nơi thì cần đổi ở nhánh nguồn.
2. **Manager/Member chưa đọc được phòng ban/chức danh/kỹ năng** (mục C 2026-11-20). Các nhánh sau (giao việc, gợi ý nhân sự CF-12) có thể cần mở quyền đọc — sẽ kiểm ở bước gộp tương ứng.
3. **Đổi cha phòng ban chưa có transaction/khóa** (mục C): hai Admin đổi chéo đồng thời có thể tạo vòng. Rủi ro thấp, để người dùng quyết có sửa trước bảo vệ không.
4. Độ phủ giảm 94,0% → 92,4%: vẫn đạt, chỉ ghi nhận.
5. Issue #13 (`.env.example` lệch) vẫn mở; bước này không làm nặng thêm.
6. Các export knip báo: giữ để khớp bản cuối (như bước 03).
