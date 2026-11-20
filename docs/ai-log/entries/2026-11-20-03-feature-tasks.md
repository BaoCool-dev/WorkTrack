---
title: feature tasks
date: 2026-11-20
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-06
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác
     Thêm dòng "commit: none" ở phần đầu nếu việc này không tạo commit (vd. thiết kế, soạn báo cáo ngoài repo). -->

## Mục tiêu
Làm CF-06 (công việc cha) theo lát dọc ở chế độ tự chạy: Manager tạo/sửa/giao việc, Member tự nhận việc ở nhóm todo, Story Point quy đổi giờ theo hệ số B1 cấu hình được, hạn chót, nhãn, API danh sách (lọc, phân trang) + chi tiết, panel chi tiết và trang "Công việc của tôi". Không làm Kanban/kéo thả/chuyển cột (feature/kanban do người dùng tự code).

## Prompt đã dùng
<!-- Dán NGUYÊN VĂN prompt chính (lấy từ: python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit).
     Mỗi prompt một dòng bắt đầu bằng ">". Dài quá thì rút gọn và ghi [rút gọn]. Không dán secret. -->
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: feature/tasks (mọc từ feature/projects). PR sẽ tạo với --base feature/projects --draft. [rút gọn]
> Phạm vi nhánh này: CF-06. Công việc cha: Manager tạo, sửa, giao việc; Member tự nhận việc từ Backlog (giới hạn WIP là Mở rộng — không chặn, chỉ hiển thị số đếm); Story Point và giờ ước tính quy đổi theo B1 (tham số cấu hình); hạn chót; nhãn; API danh sách công việc theo dự án (có lọc, phân trang) và chi tiết; panel chi tiết công việc (không gồm công việc con, bình luận, Log time — các nhánh sau); trang 'Công việc của tôi' dạng bảng. KHÔNG làm bảng Kanban, kéo thả, chuyển cột hay realtime và KHÔNG tạo API chuyển cột — tất cả thuộc feature/kanban do người dùng tự code. Công việc mới tạo nằm ở cột đầu tiên có status_group=todo. Mở rộng make seed với công việc mẫu.
> Quy trình bắt buộc: implement-ticket theo lát dọc, Context7, commit nhỏ qua ai-usage-log, feature-gate tối đa 3 vòng, báo cáo docs/gates/feature-tasks.md, push + PR draft nếu ĐẠT. Cuối cùng in đúng một dòng GATE=ĐẠT hoặc GATE=CHƯA_ĐẠT. [rút gọn]

## Nội dung AI tạo ra
<!-- Liệt kê theo file/phần: AI viết gì. Có thể kèm git diff --stat của commit AI. -->
- Commit 1 — CSDL + domain: migration `create_task_tables` (system_setting với hệ số B1 mặc định 4 giờ/SP, label + unique lower(name), task với FK kép cột/cha cùng dự án, CHECK Story Point chỉ ở cha, task_label, view `task_view`); query sqlc `task.sql` (danh sách theo dự án có lọc + phân trang, "của tôi", WIP, khóa dòng, cột todo đầu tiên, tạo/sửa/giao, nhãn), `label.sql`, `system_setting.sql`; `board_column.sql` đếm `task_count` thật; `domain/task.go` (chuẩn hóa tiêu đề/mô tả, thang SP Fibonacci, quy đổi phút, hệ số 1–40, bỏ trùng nhãn ≤ 10, điều kiện được nhận việc) + `task_test.go` viết trước; lỗi CF-06 trong `domain/errors.go`.
- Commit 2 — tầng application: quyền `settings.manage` (Admin, CEO) và `tasks.claim` (Member) trong `authz`; gói `task` (port Store/Transactor/Projects/Settings, service tạo/sửa/giao/nhận việc/danh sách/của tôi/WIP, kiểm IDOR trả ErrTaskNotFound, khóa dòng khi nhận việc và khi sửa, chụp giờ ước tính khi Story Point đổi); gói `label` (danh mục nhãn, mọi vai trò đọc, Admin/CEO ghi); gói `setting` (đọc/sửa hệ số B1). Test viết trước cho cả ba gói.
- Commit 3 — Postgres + HTTP: `task_repo.go` (repo công việc/nhãn/cấu hình, transaction `pgx.BeginFunc`, ánh xạ vi phạm FK/unique sang lỗi nghiệp vụ, nạp nhãn theo lô); `project_repo.go` xóa cột còn việc → `COLUMN_HAS_TASKS`; `task_handler.go` (danh sách theo dự án có lọc + phân trang, chi tiết, tạo, sửa, nhận việc, "của tôi", WIP, CRUD nhãn, GET/PUT cấu hình — không có API chuyển cột); 13 mã lỗi CF-06 tiếng Việt trong `errors.go`; route trong `router.go`; wiring `cmd/api/main.go`; `task_api_test.go` (đường đúng, truyền đầu vào, hình dạng JSON, cần đăng nhập, request sai, ánh xạ lỗi, chặn route chuyển cột); `task_integration_test.go` (luồng service thật trên Postgres).
- Commit 4 — seed: `cmd/seed/main.go` thêm 4 nhãn mẫu và 6 công việc cha mẫu cho 2 dự án mẫu (có việc giao sẵn cho Member, có việc để ở Backlog cho Member tự nhận, có/không hạn chót), tạo qua `task.Service` với vai trò Manager, idempotent theo tiêu đề; `main_test.go` kiểm tiêu đề, Story Point, tham chiếu dự án/nhãn hợp lệ.
- Commit 5 — frontend tab Công việc của dự án: `features/tasks` (API client, query key + hook TanStack Query, type/schema zod), `ProjectTasksPanel` (bảng lọc theo nhóm trạng thái/người nhận/nhãn, tìm kiếm, phân trang, skeleton/rỗng/lỗi), `TaskFormDialog` (tạo/sửa, Story Point → giờ ước tính theo B1, người nhận, hạn chót, nhãn; lỗi server gắn vào ô), `TaskDetailDialog` (chi tiết, nút Nhận việc cho Member, Sửa cho Manager); `shared/ui/LabelChip`, `formatHours`/`formatMinutesAsHours`; tab "Công việc" ghép vào `ProjectDetail` qua prop `renderTasks` để tránh phụ thuộc vòng; chuỗi tiếng Việt trong `messages.ts`; test RTL + MSW `ProjectTasksPanel.test.tsx`.
- Commit 6 — frontend Nhãn, Cấu hình, "Công việc của tôi": `LabelsPanel` + `LabelNameDialog` (danh mục nhãn, thêm/đổi tên/xóa cho Admin/CEO, báo nhãn đang dùng), `SettingsPanel` (hệ số B1 giờ/Story Point 1–40, chỉ xem nếu thiếu `settings.manage`), hai tab ghép vào trang Quản trị qua prop `AdminOrgView.extraTabs`; trang `/my-tasks` `MyTasksView` (bảng có cột dự án, chip WIP chỉ hiển thị, lọc nhóm trạng thái, tìm kiếm, phân trang, panel chi tiết chỉ xem); test RTL + MSW cho cả ba.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Query `ListMyTasks`/`CountWIP` sinh tham số `user_id` kiểu `uuid.NullUUID` | AI so sánh `@user_id` với cột nullable `assignee_id` mà không ép kiểu, sqlc suy ra kiểu nullable | Đọc code sinh sau `make sqlc` | Ép `@user_id::uuid`, chạy lại `make sqlc` |
| `TaskFormDialog` dùng `watch()` của react-hook-form, ESLint cảnh báo `react-hooks/incompatible-library` (React Compiler bỏ qua memo hóa component) | AI dùng mẫu cũ của react-hook-form, không tính tới React Compiler | `eslint src` báo warning | Đổi sang `useWatch({ control, name })` |
| Test API kỳ vọng mã lỗi `VALIDATION_ERROR` cho query sai | AI đoán tên mã thay vì đọc `errors.go` (mã thật là `VALIDATION_FAILED`) | `go test ./internal/transport/...` đỏ | Sửa kỳ vọng theo mã thật |
| Dùng class `text-md` cho tiêu đề panel Cấu hình | AI đoán token cỡ chữ không có trong `theme.css` (chỉ có `text-sm`/`text-base`/`text-lg`) | Tự soát lại theo design-system trước khi chạy test | Đổi sang `text-base` |
| Test "Công việc của tôi" tìm nút phân trang `/trang 2/i` | AI đoán giao diện `Pagination` thay vì đọc component (chỉ có nút Trang trước/Trang sau) | `vitest` đỏ: không tìm thấy nút | Dùng `messages.pagination.next` |
| Code frontend Commit 5–6 không qua Prettier, `make lint` đỏ (7 file) | AI chỉ chạy `eslint` + `tsc` riêng ở `frontend/`, không chạy `make lint` (có `prettier --check`) trước khi commit | `make lint` ở gốc repo báo `Code style issues found in 7 files` | `prettier --write` 7 file, chạy lại `make lint` |

## Kiểm chứng
<!-- Lệnh đã chạy và kết quả thật: make lint, make test, test nào, kiểm thử tay luồng nào. -->
- Commit 1: `make sqlc` → ok; `make migrate-up` trên Postgres Homebrew (B12) → OK `20261120095746_create_task_tables.sql`; `go test ./internal/domain/` → ok; `golangci-lint run ./internal/domain/...` → 0 issues.
- Commit 2: `go test ./internal/application/...` → ok (task, label, setting, authz); `golangci-lint run ./internal/application/...` → 0 issues. Trạng thái RED trước khi viết service: `undefined: task.Service`.
- Commit 3: `make lint` → 0 issues; `make test` → Go ok (gồm `TestTaskAPI_*`), Vitest 12 file / 133 test pass; `go vet -tags=integration ./internal/infrastructure/postgres/` → ok. Test tích hợp `TestTaskRepo_ServiceFlow_CF06` chưa chạy cục bộ (lệnh chạy trên CSDL dev bị chặn bởi quyền phiên tự chạy) — chạy trong CI.
- Commit 4: `go test ./cmd/seed/` → ok; `make seed` trên CSDL dev → "Công việc mẫu: 4 nhãn, 6 công việc (mới tạo 6)", chạy lại → "mới tạo 0" (idempotent); `make lint` → sạch; `make test` → Go ok, Vitest 133 pass.
- Commit 5: `npx tsc --noEmit` → sạch; `npx eslint src` → 0 problems (sau khi sửa `useWatch`); `npx vitest run` → 13 file / 149 test pass.
- Commit 6: `npx tsc --noEmit` → sạch; `npx eslint src` → 0 problems; `npx vitest run` → 16 file / 168 test pass (LabelsPanel, SettingsPanel, MyTasksView).
- Sửa định dạng: `npx prettier --write` 7 file → `make lint` sạch (golangci-lint + eslint + tsc + prettier); `make test` → Go ok, Vitest 16 file / 168 test pass.
