# Tích hợp bước 08/9 — `feature/kanban` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `8c2a2c0` (cha: `f767da8` = HEAD sau bước 07, tag `integ/07-manual-testing`). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 16 (`b1e168b` … `13333ea`), cộng merge commit `8c2a2c0` |
| Thay đổi | 110 file, +6716 / −118 dòng |
| Chức năng mang vào | **CF-04** Kanban kéo thả (UC05): luật chuyển cột phía domain + client, chặn Member kéo sang nhóm hoàn thành, đổi thứ tự, kéo thả lạc quan có hoàn tác, version chống xung đột, lịch sử chuyển cột. **CF-05** WIP theo dự án: cấu hình giới hạn, kiểm WIP khi nhận việc/đổi người nhận/chuyển cột, hộp thoại WIP đầy. Realtime: hub WebSocket trong bộ nhớ (`coder/websocket`) theo dự án, client tự kết nối lại, dải mất kết nối; script KPI 1 (p95 = 10 ms) |
| Migration mới | `20261120204233_add_kanban_wip_version_history.sql` |
| Dependency mới | Go: `github.com/coder/websocket v1.8.15` (có trong CLAUDE.md §2). FE: `@dnd-kit/core ^6.3.1` (đề xuất trong `frontend/CLAUDE.md`) |
| Khác | Chuẩn kiểm thử HCM-UTE (`docs/testing/test-design-standard.md`, skill `test-case-design`), `scripts/run-all-tests.sh`, `scripts/test-cases.sh`, `scripts/test-summary.py`, mẫu `.github/ISSUE_TEMPLATE/bug_report.md`, nhánh `test/<mo-ta>` trong CONTRIBUTING, 3 quyền `allow` mới trong `.claude/settings.json`, báo cáo gate `docs/gates/feature-kanban.md` |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip | Backend API | Backend Unit | FE Unit/UI | Độ phủ domain + application |
|---|---|---|---|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/08-kanban/after-merge`) | 974 | 974 | 0 | 0 | 363 | 391 | 220 | 90,2% |
| Sau dọn (`.ai-log/integration/08-kanban/after-cleanup`) | 974 | 974 | 0 | 0 | 363 | 391 | 220 | 90,2% |

So với bước 07 (831): +143 test (API +35, Unit +59, FE +49). Không có test lỗi → không có commit `fix(integration)`, không tạo issue mới.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| Biến môi trường | — | Bước này không thêm biến nào cho app: `os.Getenv`/`LookupEnv` (backend) và `import.meta.env` (FE, chỉ `VITE_API_BASE_URL`) không đổi; URL WebSocket suy ra từ `VITE_API_BASE_URL`. Biến `KPI_*` chỉ dùng cho script đo KPI, đã ghi trong `manual-test-guide.md`. Issue [#13](https://github.com/BaoCool-dev/WorkTrack/issues/13) (`.env.example` lệch SMTP/`APP_BASE_URL`) vẫn OPEN, không làm nặng thêm | — |
| `docker-compose.yml`, `.github/workflows`, `Makefile`, `.gitignore`, `.mcp.json` | — | Không đổi ở bước này | — |
| CI: phiên bản Go/Node | — | `go-version-file: backend/go.mod` (go 1.27); `node-version: 24` thỏa `engines.node >=22.22` | — |
| `go mod tidy` (thêm `coder/websocket`), `npm install --package-lock-only` (thêm `@dnd-kit/core`), `sqlc generate` (query mới ở `task.sql`, `project.sql`) | — | `git status` sạch, không khác biệt | — |
| Goose: 5 migration, `goose -dir backend/db/migrations validate` | — | rc=0, tên tăng dần, không trùng | — |
| Script mới (`run-all-tests.sh`, `test-cases.sh`, `test-summary.py`, `kpi/kanban-sync-latency.mjs`) | — | Đều có bit thực thi (100755); script gọi nhau tồn tại; kết quả ghi vào `.ai-log/` (đã ignore) và `docs/testing/runs/` (cố ý commit) | — |
| Target `make` nhắc trong tài liệu mới (`cover`, `dev-local`, `lint`, `seed`, `test`, `test-integration`) | — | Có thật trong `Makefile` | — |
| `.claude/settings.json` (thêm `git push … test/:*`, `./scripts/run-all-tests.sh:*`), `.mcp.json` | — | JSON hợp lệ. Quyền push chỉ cho nhánh `test/*`, không chạm `main` | — |
| `gitleaks detect` | — | no leaks found | — |
| `golangci-lint run ./...` | — | 0 issues | — |

Không có commit `fix(config)` ở bước này.

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| `deadcode.txt` | Rỗng | Không có gì | — |
| knip — export mới của kanban: `backoffDelay`, `applyBoardEvent` (`useProjectSocket.ts`), `UNASSIGNED` (`boardState.ts`), `entersWip`, `DropBlockCode` (`dropRules.ts`), `BoardEventType` (`types.ts`), `KanbanBoardProps` (`kanban/index.ts`) | Đều được dùng trong chính file đó (vd. `useProjectSocket.ts:74`, `:96`; `boardState.ts:82`; `dropRules.ts:52`, `:79`); chỉ là `export` thừa. Bản cuối `test/test-cases` vẫn còn | Giữ — dùng ở bước sau / còn ở bản cuối; bỏ `export` là đổi API module, không phải xóa rác | — |
| knip — `afterTaskWrite` (`tasks/index.ts`), `useProject` (`projects/index.ts`) | Barrel export, dùng nội bộ (`tasks/api/queries.ts:52`, `ProjectDetail.tsx:53`); barrel giống hệt ở bản cuối | Giữ — còn ở bản cuối | — |
| knip — các mục cũ (`adminKeys`, `useSession`, `useCanAccess`, `ForbiddenState`, `healthKeys`, `fetchHealth`, `useHealth`, `TOAST_DURATION_MS`, `Ref`, `CurrentUser`, `HealthResponse`, `ProjectMember`, các `*Props` ở `shared/ui`) | Đã xét ở bước 03–07 | Giữ (lý do như các bước trước) | — |
| knip — dependency không dùng | knip không báo; `@dnd-kit/core` dùng trong `features/kanban` | Không có gì | — |
| TODO/FIXME, `fmt.Println` gỡ lỗi | `git grep` không trúng (trừ tên lỗi `BOARD_NEEDS_TODO_COLUMN`) | Không có gì | — |
| `console.log` ở `scripts/kpi/kanban-sync-latency.mjs:171-172` | In kết quả KPI ra stdout — là đầu ra của CLI | Giữ vì lý do khác: không phải gỡ lỗi | — |
| Code bị comment-out | Rà diff `backend`, `frontend/src`: không có | Không có gì | — |
| Tài liệu trỏ tới thứ không có: `docs/testing/test-report.md`, `test-plan.md`, `docs/requirements/{commitment,use-cases,…}.md` | `test-report.md` do `scripts/test-cases.sh` sinh ra; còn lại đã ghi ở bước 01 (tài liệu nguồn người dùng cần bổ sung) | Giữ vì lý do khác | — |

Không có commit `chore(cleanup)` → không có gì phải revert.

## (e) Việc người dùng tự kiểm

```bash
git diff integ/07-manual-testing..HEAD --stat
git diff integ/07-manual-testing..HEAD -- backend/internal/domain/kanban.go backend/internal/application/kanban
git diff integ/07-manual-testing..HEAD -- backend/internal/infrastructure/realtime backend/internal/transport/httpapi/ws_handler.go
git diff integ/07-manual-testing..HEAD -- backend/db/migrations backend/db/queries
git diff integ/07-manual-testing..HEAD -- frontend/src/features/kanban
git diff integ/07-manual-testing..HEAD -- .claude/settings.json docs/decisions.md
make migrate-up && make seed      # ghi lại mật khẩu in ra
make dev-local
```
Màn hình: chi tiết dự án → bảng Kanban (`/projects/<id>?tab=board`), cửa sổ 1366×768.
- **Manager**: kéo thẻ qua các cột theo luật chuyển; kéo sang "Hoàn thành" được; đổi giới hạn WIP trên thanh công cụ.
- **Member**: kéo việc của mình; kéo sang nhóm hoàn thành → bị chặn có lý do; nhận việc khi WIP đầy → hộp thoại WIP đầy; thẻ con không tính WIP.
- **Hai tab** (Manager + Member) cùng dự án: kéo ở tab này, tab kia cập nhật ngay; tắt API → dải mất kết nối, bật lại → tự kết nối.
- **CEO/Admin**: mở bảng, xác nhận không kéo được (AI chưa tự chạy trên trình duyệt ở bước này — đối chiếu `docs/gates/feature-kanban.md`); thử `POST /api/v1/tasks/{id}/move` bằng curl với Member không thuộc dự án → 404.
- Đo lại KPI 1: `KPI_MANAGER_PASSWORD=… KPI_MEMBER_PASSWORD=… node scripts/kpi/kanban-sync-latency.mjs`.

## (f) Không chắc chắn — để người dùng quyết

1. **Hub WebSocket trong bộ nhớ** (giả định mục C của nhánh): chỉ đúng khi chạy một tiến trình API. Đủ cho đồ án; nếu triển khai nhiều bản sao cần pub/sub (vd. PostgreSQL `LISTEN/NOTIFY`).
2. **`export` thừa knip báo** (`backoffDelay`, `applyBoardEvent`, `entersWip`, `UNASSIGNED`…): AI giữ nguyên vì bản cuối vẫn có; người dùng có thể bỏ `export` khi chỉnh sửa nhánh kanban (commit `Human-Edit: yes`, CLAUDE.md §7b).
3. **Quyền mới trong `.claude/settings.json`** cho phép Claude tự `git push` nhánh `test/*` — người dùng xác nhận có muốn giữ quyền này không.
4. KPI 1 đo trên localhost (p95 = 10 ms); chưa đo qua mạng thật hay Docker.
5. Issue #13 (`.env.example`) vẫn mở; AI không đọc/sửa `.env*` (bị chặn).
