# Tích hợp bước 06/9 — `feature/tasks` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `588ea4a` (cha: `8f092e0` = tag `integ/05-projects`, `72a93c2` đầu nhánh). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 13 (`9813eb7` … `72a93c2`), cộng merge commit `588ea4a` |
| Thay đổi | 86 file, +6966 / −41 dòng (`git diff --stat 8f092e0..588ea4a`), gồm ~740 dòng code sinh sqlc, ~2500 dòng test và 11 ảnh bằng chứng |
| Chức năng mang vào | **CF-06 Công việc cha**: migration `20261120095746_create_task_tables` (công việc, nhãn, tham số hệ thống B1); query sqlc `task.sql`, `label.sql`, `system_setting.sql`; domain `task.go` (thang SP Fibonacci, giới hạn độ dài, B1 từ 1 đến 40); use case `application/task`, `label`, `setting`; `authz` thêm `ManageSettings` (Admin) và `ClaimTask` (Member); endpoint `/projects/{id}/tasks`, `/tasks/{id}` (xem, sửa, nhận việc), `/me/tasks`, `/me/wip`, `/labels`, `/settings`; xóa cột có việc → `ErrColumnHasTasks`; seed thêm nhãn và công việc mẫu; FE `features/tasks` (tab Công việc ở chi tiết dự án, form tạo/sửa, panel chi tiết, tab Nhãn + Cấu hình ở trang Quản trị, trang `/my-tasks`); báo cáo gate ĐẠT (CI PR #7 xanh) và 7 giả định mục C |
| Thay đổi quy trình đi kèm | `CLAUDE.md` §7b/§7c và skill `implement-ticket`: `feature/kanban` đổi từ "người dùng tự code" sang "Claude viết bản đầu, người dùng chỉnh sửa" (quyết định 21/11); `scripts/auto-run.sh`: giai đoạn 2 bắt đầu từ `feature/tasks`, thêm nhánh `docs/manual-testing` và `feature/kanban`, báo lỗi khi `--only`/`--from` không có trong kế hoạch |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip | Độ phủ domain + application |
|---|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/06-tasks/after-merge`) | 831 | 831 | 0 | 0 | 90,3% |
| Sau dọn (`.ai-log/integration/06-tasks/after-cleanup`) | 831 | 831 | 0 | 0 | 90,4% |

Chi tiết (giống nhau ở cả hai lần): backend API 328, backend Unit 332, frontend Unit/UI 171. Số Pass không giảm (bước 05: 693/693).

- Gồm cả test tích hợp Postgres trong `task_integration_test.go`.
- Không có test lỗi → không có commit `fix(integration)`, không tạo issue mới.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| Biến môi trường | — | Bước này **không** thêm biến nào (diff không có `Getenv`/`LookupEnv`/`import.meta.env` mới; `main.go` chỉ thêm wiring service task/label/setting). Issue [#13](https://github.com/BaoCool-dev/WorkTrack/issues/13) (`.env.example` lệch) vẫn OPEN | — |
| `docker-compose.yml`, `.github/workflows`, `Makefile`, `package.json`, lockfile, `go.mod`, `.gitignore`, `.mcp.json`, `.claude/settings.json` | — | Không file nào đổi ở bước này (`git diff --stat 8f092e0..HEAD -- <các đường dẫn>` rỗng) | — |
| `scripts/auto-run.sh` thay đổi | — | Chỉ đổi kế hoạch giai đoạn 2 và thêm bước kiểm `--only`/`--from`; không gọi script/target mới nào | — |
| `go mod tidy` | — | Không tạo khác biệt | — |
| `npm install --package-lock-only` | — | Không tạo khác biệt lockfile | — |
| `sqlc generate` | — | Không tạo khác biệt `sqlcgen/` | — |
| Goose: 4 migration (`20261119092421`, `20261119192122`, `20261120003549`, `20261120095746`); `goose -dir backend/db/migrations validate` | — | rc=0, tăng dần, không trùng | — |
| `.gitignore` | — | `git status` sạch sau sqlc, tidy và 2 lần chạy test | — |
| `.claude/settings.json`, `.mcp.json` | — | JSON hợp lệ | — |
| `gitleaks detect` | — | no leaks found | — |
| `golangci-lint run ./...` (gồm `unused`) | — | 0 issues | — |

Không có commit `fix(config)` ở bước này.

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| `domain.StoryPointScale` | `deadcode.txt`: unreachable; `git grep StoryPointScale` ở HEAD và ở `test/test-cases` chỉ có dòng định nghĩa (FE dùng hằng `STORY_POINTS` riêng trong `tasks/model/types.ts`) | Đã xóa | `8cfcf11` |
| knip mới: `taskKeys` (`tasks/api/queries.ts`) | Bản cuối `test/test-cases` import trong `features/kanban/api/queries.ts` | Giữ vì dùng ở bước sau | — |
| knip mới: `Ref` (`tasks/model/types.ts`) | Kiểu dùng nội bộ trong cùng file (`Task.assignee`…), chỉ "export thừa"; trùng mẫu `Ref` ở `admin` đã giữ từ bước 04 | Giữ vì lý do khác | — |
| knip còn lại (`adminKeys`, `Ref` admin, `useSession`, `useCanAccess`, `ForbiddenState`, `healthKeys`, `fetchHealth`, `useHealth`, `TOAST_DURATION_MS`, các `*Props`, `CurrentUser`, `HealthResponse`, `ProjectMember`) | Đã xét ở bước 03–05 | Giữ (như các bước trước) | — |
| knip: dependency, file không dùng | `knip.txt` chỉ có mục export/type | Không có gì | — |
| `fmt.Printf` "Công việc mẫu: …" trong `backend/cmd/seed/main.go` | Đầu ra CLI có chủ đích (B11), cùng kiểu các dòng in bước 04/05 | Giữ vì lý do khác | — |
| Prompt giai đoạn 1 trong `scripts/auto-run.sh` còn câu "feature/kanban do người dùng tự code" | Lịch sử kế hoạch giai đoạn 1 (đã chạy xong); quy tắc hiện hành đã đổi trong `CLAUDE.md` | Giữ vì lý do khác (xem mục f) | — |
| `taskCount` của cột (tồn đọng bước 05) | Bước này nối `ErrColumnHasTasks` khi xóa cột có việc | Không còn là mục giữ chỗ | — |
| TODO/FIXME/XXX/HACK, `console.log/debug` | `git grep` trong `backend frontend/src scripts Makefile docker-compose.yml .github` (trừ sqlcgen): 0 | — | — |
| Code bị comment-out | `git grep` mẫu `// func|if|return|const|import|export|let|var|for|await`: 0 | — | — |

Commit dọn `8cfcf11` không làm đỏ test (831/831) → không phải revert.

## (e) Việc người dùng tự kiểm

```bash
git diff integ/05-projects..HEAD --stat
git diff integ/05-projects..HEAD -- backend/ ':!backend/internal/infrastructure/postgres/sqlcgen'
git diff integ/05-projects..HEAD -- backend/db/migrations backend/db/queries
git diff integ/05-projects..HEAD -- backend/internal/application/authz
git diff integ/05-projects..HEAD -- frontend/src
git diff integ/05-projects..HEAD -- CLAUDE.md .claude/skills scripts/auto-run.sh   # đổi quy tắc feature/kanban
git diff integ/05-projects..HEAD -- docs/decisions.md                              # 7 giả định mục C của feature/tasks
make lint && make test && make test-integration
make migrate-up && make seed                     # in thêm "Công việc mẫu: … nhãn, … công việc"
make dev-local
```
- **Manager**: mở chi tiết dự án → tab Công việc: tạo việc (rơi vào cột todo đầu tiên), chọn SP (chỉ thang Fibonacci), giờ ước tính quy đổi theo B1, gắn tối đa 10 nhãn, giao cho thành viên, tìm/lọc/phân trang; mở panel chi tiết; thử xóa một cột Kanban đang có việc → bị chặn.
- **Member**: tab Công việc chỉ xem + nút "Nhận việc" với việc chưa có người ở cột todo; `/my-tasks` thấy việc của mình và chip `WIP n`; gọi thẳng `PUT /api/v1/tasks/{id}` → bị chặn.
- **Admin**: Quản trị → tab Nhãn (thêm/đổi tên/xóa; nhãn đang dùng bị khóa xóa) và Cấu hình (B1 trong 1–40); `/my-tasks` → màn cấm.
- **CEO**: sửa được Nhãn/Cấu hình như Admin (xem mục f.1); xem công việc mọi dự án ở chế độ chỉ xem.
- **Manager khác** mở URL công việc của dự án không phụ trách → 404.

## (f) Không chắc chắn — để người dùng quyết

1. **CEO sửa được nhãn và hệ số B1.** Đúng với giả định mục C và mã (`authz.go`: CEO nhận toàn bộ `adminPerms`, gồm `ManageSettings`). CLAUDE.md §5.2 chỉ quy định Dashboard tổng quan là chỉ xem, nên không vi phạm, nhưng "CEO ghi cấu hình" có thể bị hội đồng hỏi — người dùng cân nhắc có giữ hay không.
2. **Quy tắc `feature/kanban` đã đổi** (Claude viết bản đầu) theo quyết định 21/11. Prompt giai đoạn 1 trong `scripts/auto-run.sh` vẫn ghi "người dùng tự code" — để nguyên vì là lịch sử, nhưng nếu muốn tài liệu nhất quán thì có thể sửa câu chữ.
3. **Thang SP Fibonacci chỉ chặn ở domain**, CHECK CSDL vẫn 1–100 (issue [#6](https://github.com/BaoCool-dev/WorkTrack/issues/6)). Không sửa migration đã áp dụng; nếu muốn chặt hơn cần migration mới.
4. **Tồn đọng gate** (issue #6): dự án đã hủy/hoàn thành vẫn tạo việc được, chưa có quy tắc hạn chót, chưa test hai người cùng nhận việc, chưa E2E tự động — AI không xử lý ở bước tích hợp.
5. Gate ghi 5 giả định mục C nhưng `decisions.md` có 7 dòng cho `feature/tasks` (thêm 2 dòng về vị trí tab Nhãn/Cấu hình và `/my-tasks` chỉ xem). Không ảnh hưởng chức năng.
6. Issue #13 (`.env.example` lệch) vẫn mở; bước này không làm nặng thêm.
