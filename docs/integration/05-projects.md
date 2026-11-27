# Tích hợp bước 05/9 — `feature/projects` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `f1fb96d` (cha: `e3733e5` = tag `integ/04-org`, `cf3e12b` đầu nhánh). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 9 (`5938064` … `cf3e12b`), cộng merge commit `f1fb96d` |
| Thay đổi | 70 file, +6963 / −23 dòng (`git diff --stat e3733e5..f1fb96d`), gồm ~690 dòng code sinh sqlc, ~2050 dòng test và 12 ảnh bằng chứng |
| Chức năng mang vào | **CF-03 Quản lý dự án & cấu hình Kanban**: migration `20261120003549_create_project_tables` (`project`, `project_member`, `board_column` vị trí unique DEFERRABLE, `workflow_transition`); query sqlc `project.sql`, `board_column.sql`; domain `project.go` (5 trạng thái, 2–8 cột, luôn có cột nhóm done, luật chuyển mặc định, chặn Member vào/ra nhóm done); use case `application/project` với port `Store` + `Transactor`; `authz` thêm `ViewOwnProjects`, `RequireProjectManager`; 14 endpoint dưới `/api/v1/projects`; seed 2 dự án mẫu; FE `features/projects` (màn `/projects` lọc/tìm/phân trang + form; `/projects/:projectId` 4 tab Thông tin, Thành viên, Cột Kanban, Luật chuyển), `OrgTable` → `shared/ui/DataTable`, `formatDate`/`formatTimestampDate`; báo cáo gate (ĐẠT), ma trận truy vết, 13 giả định mục C |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip |
|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/05-projects/after-merge`) | 693 | 693 | 0 | 0 |
| Sau dọn (`.ai-log/integration/05-projects/after-cleanup`) | 693 | 693 | 0 | 0 |

Chi tiết (giống nhau ở hai lần): backend API 271, backend Unit 289, frontend Unit/UI 133. Độ phủ domain + application: 90,3% (bước 04: 92,4%). Số Pass không giảm (bước 04: 513/513).

- Gồm cả test tích hợp Postgres của `project_integration_test.go`.
- Không có test lỗi → không có commit `fix(integration)`, không tạo issue mới.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| Biến môi trường | — | Bước này **không** thêm biến nào (diff không có `Getenv`/`LookupEnv`/`import.meta.env` mới; `config.go` không đổi; `main.go` chỉ thêm wiring service dự án). Issue [#13](https://github.com/BaoCool-dev/WorkTrack/issues/13) (`.env.example` lệch) vẫn OPEN | — |
| `docker-compose.yml`, `.github/workflows`, `Makefile`, `scripts/`, `package.json`, lockfile, `go.mod`, `.gitignore`, `.claude/`, `.mcp.json` | — | Không file nào đổi ở bước này (`git diff --stat e3733e5..HEAD -- <các đường dẫn>` rỗng) | — |
| `go mod tidy` | — | Không tạo khác biệt | — |
| `npm install --package-lock-only` | — | Không tạo khác biệt lockfile | — |
| `sqlc generate` | — | Không tạo khác biệt `sqlcgen/` | — |
| Goose: 3 migration (`20261119092421`, `20261119192122`, `20261120003549`); `goose -dir backend/db/migrations validate` | — | rc=0, tăng dần, không trùng | — |
| `.gitignore` | — | `git status` sạch sau sqlc, tidy và 2 lần chạy test | — |
| `.claude/settings.json`, `.mcp.json` | — | JSON hợp lệ | — |
| `gitleaks detect` | — | no leaks found | — |
| `golangci-lint run ./...` (gồm `unused`) | — | 0 issues | — |

Không có commit `fix(config)` ở bước này.

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| `deadcode` | `deadcode.txt` rỗng | Không có gì | — |
| knip: dependency, file không dùng | `knip.txt` chỉ có mục export/type | Không có gì | — |
| knip mới: `projectKeys` (`projects/api/queries.ts`) | Bản cuối `test/test-cases` import từ `@/features/projects` trong `kanban/api/queries.ts`, `kanban/hooks/useProjectSocket.ts` | Giữ vì dùng ở bước sau | — |
| knip còn lại (`adminKeys`, `Ref`, `useSession`, `useCanAccess`, `ForbiddenState`, `healthKeys`, `fetchHealth`, `useHealth`, `TOAST_DURATION_MS`, các `*Props`, `CurrentUser`, `HealthResponse`) | Đã xét ở bước 03/04 | Giữ (như các bước trước) | — |
| `fmt.Printf` "Dự án mẫu: …" trong `backend/cmd/seed/main.go` | Đầu ra CLI có chủ đích (B11), thay dòng giữ chỗ bước 04 | Giữ vì lý do khác | — |
| `OrgTable` đổi tên thành `shared/ui/DataTable` | `git grep OrgTable -- frontend`: 0 tham chiếu code (chỉ còn trong nhật ký AI và `decisions.md`, là lịch sử) | Không có gì | — |
| `PlaceholderPage` | Vẫn là fallback trong `routes.tsx` | Giữ vì dùng ở bước sau | — |
| `taskCount` luôn 0 ở cột | Giả định mục C, chờ CF-06 nối truy vấn đếm | Giữ vì dùng ở bước sau | — |
| TODO/FIXME/XXX/HACK, `console.log/debug` | `git grep` trong `backend frontend/src scripts Makefile docker-compose.yml .github` (trừ sqlcgen): 0 | — | — |
| Code bị comment-out | `git grep` mẫu `// func|if|return|const|import|export|let|var|for|await`: 0 | — | — |

Không có commit `chore(cleanup)` → không có gì phải revert.

## (e) Việc người dùng tự kiểm

```bash
git diff integ/04-org..HEAD --stat
git diff integ/04-org..HEAD -- backend/ ':!backend/internal/infrastructure/postgres/sqlcgen'
git diff integ/04-org..HEAD -- backend/db/migrations backend/db/queries
git diff integ/04-org..HEAD -- backend/internal/application/authz
git diff integ/04-org..HEAD -- frontend/src
git diff integ/04-org..HEAD -- docs/decisions.md      # 13 giả định mục C của feature/projects
make lint && make test && make test-integration
make migrate-up && make seed                          # in "Dự án mẫu: 2 dự án"
make dev-local
```
- **Manager**: mở `/projects` → tạo dự án (có sẵn 5 cột + luật mặc định); mở chi tiết: sửa thông tin, thêm/bỏ thành viên (chỉ Manager/Member, có cả người "Chờ kích hoạt"), thêm/sửa/xóa/đổi thứ tự cột (giới hạn 2–8, luôn còn cột nhóm done), tab Luật chuyển: ô Member vào/ra nhóm done bị khóa.
- **Manager khác** (không phụ trách): mở dự án của người khác → 404, không sửa được.
- **Member**: chỉ thấy dự án mình là thành viên; không thấy nút sửa; mở URL dự án không tham gia → 404.
- **CEO**: thấy mọi dự án, chỉ xem (không nút ghi; gọi thẳng `PUT /api/v1/projects/{id}` → 403).
- **Admin**: không có menu Dự án; `GET /api/v1/projects` → 403.
- Màn Nhân sự & phòng ban (Admin) vẫn hiển thị bảng đúng sau khi `OrgTable` chuyển thành `DataTable`.

## (f) Không chắc chắn — để người dùng quyết

1. **Admin nhận 403 ở danh sách nhưng 404 ở chi tiết dự án** (mục C). Nhất quán với "không lộ sự tồn tại", nhưng nếu hội đồng hỏi thì cần giải thích được vì sao hai mã khác nhau.
2. **Manager phụ trách cố định là người tạo**, chưa có chức năng bàn giao dự án. Nếu Manager nghỉ, dự án không ai sửa được — chấp nhận cho phạm vi đồ án?
3. **Độ phủ giảm 92,4% → 90,3%** (`application/project` 83,9%). Vẫn đạt, chỉ ghi nhận.
4. **Tồn đọng của gate** (issue [#4](https://github.com/BaoCool-dev/WorkTrack/issues/4)): `taskCount` cố định 0, chưa test FE cho CEO ở màn chi tiết, chưa E2E Playwright tự động — AI không xử lý ở bước tích hợp.
5. Issue #13 (`.env.example` lệch) vẫn mở; bước này không làm nặng thêm.
6. Kiểm tra luật chuyển khi kéo thẻ chưa có — để dành `feature/kanban` (người dùng tự code); cần đảm bảo nhánh đó dùng `workflow_transition` của bước này.
