---
title: feature projects
date: 2026-11-20
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-03
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác
     Thêm dòng "commit: none" ở phần đầu nếu việc này không tạo commit (vd. thiết kế, soạn báo cáo ngoài repo). -->

## Mục tiêu
Làm CF-03 (quản lý dự án) theo lát dọc ở chế độ tự chạy: Manager tạo/sửa dự án, trạng thái, thành viên; cấu hình 2–8 cột Kanban theo status_group; lưu luật workflow_transition mặc định + API đọc/sửa cấu hình (áp dụng luật khi kéo thẻ để dành feature/kanban do người dùng tự code).

## Prompt đã dùng
<!-- Dán NGUYÊN VĂN prompt chính (lấy từ: python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit).
     Mỗi prompt một dòng bắt đầu bằng ">". Dài quá thì rút gọn và ghi [rút gọn]. Không dán secret. -->
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: feature/projects (mọc từ feature/org). PR sẽ tạo với --base feature/org --draft. [rút gọn]
> Phạm vi nhánh này: CF-03. Manager tạo dự án trống, sửa thông tin, trạng thái dự án (Mới tạo/Đang tiến hành/Hoàn thành/Tạm dừng/Đã hủy), thêm/bớt thành viên; cấu hình 2–8 cột Kanban (thêm, đổi tên, xóa, sắp xếp; status_group todo|in_progress|done; cảnh báo khi xóa cột có việc); bảng workflow_transition với luật mặc định (CHỈ lưu dữ liệu + API đọc/sửa cấu hình). Danh sách dự án theo vai trò. [rút gọn]
> Quy trình bắt buộc: implement-ticket theo lát dọc, Context7, commit nhỏ qua ai-usage-log, feature-gate tối đa 3 vòng, báo cáo docs/gates/feature-projects.md, push + PR draft nếu ĐẠT. Cuối cùng in đúng một dòng GATE=ĐẠT hoặc GATE=CHƯA_ĐẠT. [rút gọn]

## Nội dung AI tạo ra
<!-- Liệt kê theo file/phần: AI viết gì. Có thể kèm git diff --stat của commit AI. -->
- Commit 1 — CSDL + domain: migration `create_project_tables` (project, project_member, board_column với vị trí unique DEFERRABLE, workflow_transition khóa ngoại kép ON DELETE CASCADE); query sqlc `project.sql`, `board_column.sql` + code sinh; `domain/project.go` (chuẩn hóa tên, trạng thái, kiểm tra 2–8 cột, phải có cột done, luật chuyển mặc định, kiểm tra xóa cột) + `project_test.go`; lỗi CF-03 trong `domain/errors.go`.
- Commit 2 — application: `authz` thêm quyền `ViewOwnProjects` (Manager, Member) và `RequireProjectManager` (chỉ Manager phụ trách được sửa); `application/project` gồm port `Store`/`Transactor` và `Service` (tạo dự án kèm 5 cột + luật mặc định trong 1 transaction, danh sách theo vai trò, xem: không phải thành viên → 404, sửa thông tin/trạng thái, thêm/bớt thành viên, thêm/sửa/xóa/sắp xếp cột có khóa dự án, thay luật chuyển). Test bảng với fake Store mô phỏng rollback.
- Commit 3 — repository Postgres: `postgres/project_repo.go` cài đặt `project.Store` (ánh xạ ràng buộc unique/FK sang lỗi domain, ngày `pgtype.Date`) và `ProjectTx` cài `project.Transactor` bằng `pgx.BeginFunc`; test tích hợp `project_integration_test.go` chạy Service thật trên Postgres (sắp xếp đảo cột nhờ ràng buộc DEFERRABLE, cascade luật khi xóa cột, rollback khi lỗi giữa transaction).
- Commit 4 — HTTP API: `httpapi/project_handler.go` (port `ProjectService`, DTO, ngày `YYYY-MM-DD`, 14 endpoint dưới `/api/v1/projects`: danh sách/tạo/xem/sửa, thành viên + ứng viên, bảng, thêm/sửa/xóa cột, `/column-order`, `/transitions`); route trong `router.go` (chỉ yêu cầu đăng nhập, quyền kiểm trong service); ánh xạ lỗi CF-03 trong `errors.go`; `project_api_test.go` (đường thành công từng endpoint, chuyển đúng đầu vào, hình dạng JSON, 401, 400, bảng ánh xạ lỗi 403/404/409/422).
- Commit 5 — wiring + seed: `cmd/api/main.go` lắp `project.NewService` vào `Deps.Projects`; `cmd/seed` tạo 2 dự án mẫu (B11) qua `project.Service` (có sẵn cột + luật mặc định), idempotent theo tên, thêm Member mẫu làm thành viên; test seed kiểm tên/trạng thái hợp lệ.
- Commit 6 — giao diện: `features/projects` (types, schema zod, API client, hook TanStack Query, `ProjectList` lọc/tìm/phân trang + `ProjectFormDialog`, `ProjectDetail` 4 tab: thông tin, `MembersPanel`, `ColumnsPanel` + `ColumnFormDialog`, `TransitionsPanel` ma trận luật theo vai trò); trang `/projects`, `/projects/:projectId`; chuyển `OrgTable` → `shared/ui/DataTable`; `formatDate`/`formatTimestampDate` trong `shared/lib/format.ts`; chữ tiếng Việt trong `messages.ts`; test RTL + MSW `ProjectList.test.tsx`, `ProjectDetail.test.tsx` (23 test: nút theo vai trò, body POST/PUT, 404, thêm/bỏ thành viên, cảnh báo xóa cột có việc + lỗi `COLUMN_HAS_TASKS`, body `columnIds`, khóa ô Member→done, lưu luật). Ghi 13 giả định mục C vào `docs/decisions.md`.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Định dạng ngày bằng template string trong `features/projects/model/format.ts` | AI tự viết hàm format riêng, quên quy định chỉ format qua `shared/lib/format.ts` | ESLint `restrict-template-expressions` + đối chiếu `frontend/CLAUDE.md` | Chuyển sang `formatDate` dùng `Intl` trong `shared/lib/format.ts`, xóa file riêng |
| Hoán đổi `ids[index]`, `ids[j]` kiểu `string \| undefined` | AI chỉ chạy `tsc --noEmit` lẻ, không bắt `noUncheckedIndexedAccess` như `tsc -b` trong `make lint` | `make lint` đỏ tại `ColumnsPanel.tsx:62` | Lấy 2 cột, kiểm tra tồn tại rồi map lại danh sách id |

## Kiểm chứng
<!-- Lệnh đã chạy và kết quả thật: make lint, make test, test nào, kiểm thử tay luồng nào. -->
- Commit 1: `go test ./internal/domain/` → ok.
- Commit 2: `go test ./internal/application/...` → ok; `golangci-lint run ./internal/application/...` → 0 issues.
- Commit 3: `go build ./...`, `golangci-lint run ./internal/infrastructure/...` → 0 issues; `TEST_DATABASE_URL=… go test -tags=integration ./internal/infrastructure/postgres/` trên Postgres Homebrew (B12) → ok (gồm `TestProjectRepo_ServiceFlow_CF03`).
- Commit 4: `go test ./internal/transport/httpapi/` → ok (49 subtest dự án pass); `golangci-lint run ./...` → 0 issues.
- Commit 5: `go test ./cmd/...` → ok; `make migrate-up` + `make seed` hai lần trên Postgres Homebrew → lần 1 tạo 2 dự án, lần 2 tạo 0 (idempotent).
- Commit 6: `make lint` → 0 issues (golangci-lint, eslint, `tsc -b`, prettier); `make test` → Go ok, Vitest 12 file / 133 test pass.
