---
title: Tích hợp từng bước các nhánh vào release/v1
date: 2026-11-27
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: review
scope: Tích hợp release/v1 (bước 01..09)
---

## Mục tiêu
Gộp lần lượt các nhánh tính năng vào `release/v1` (không phải `main`). Mỗi bước: đọc kết quả test sau gộp, rà cấu hình, dọn code rác (chỉ khi bản cuối `test/test-cases` không còn dùng), chạy lại test, viết báo cáo `docs/integration/<NN>-<nhanh>.md`.

## Prompt đã dùng
### Bước 01 — chore/ai-workflow
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng. Đang TÍCH HỢP từng bước vào nhánh release/v1 (KHÔNG phải main). Bước 01/9: vừa gộp nhánh chore/ai-workflow … Việc phải làm: 1. TEST … 2. CẤU HÌNH … 3. CODE RÁC … 4. Chạy lại … 5. BÁO CÁO docs/integration/01-ai-workflow.md … 6. Commit báo cáo qua skill ai-usage-log … [rút gọn]

### Bước 02 — chore/scaffold
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng. Đang TÍCH HỢP từng bước vào nhánh release/v1 (KHÔNG phải main). Bước 02/9: vừa gộp nhánh chore/scaffold … Việc phải làm: 1. TEST … 2. CẤU HÌNH … 3. CODE RÁC … 4. Chạy lại … 5. BÁO CÁO docs/integration/02-scaffold.md … 6. Commit báo cáo qua skill ai-usage-log … [rút gọn]

### Bước 03 — feature/auth
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng. Đang TÍCH HỢP từng bước vào nhánh release/v1 (KHÔNG phải main). Bước 03/9: vừa gộp nhánh feature/auth … Việc phải làm: 1. TEST … 2. CẤU HÌNH … 3. CODE RÁC … 4. Chạy lại … 5. BÁO CÁO docs/integration/03-auth.md … 6. Commit báo cáo qua skill ai-usage-log … [rút gọn]

### Bước 05 — feature/projects
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng. Đang TÍCH HỢP từng bước vào nhánh release/v1 (KHÔNG phải main). Bước 05/9: vừa gộp nhánh feature/projects … Việc phải làm: 1. TEST … 2. CẤU HÌNH … 3. CODE RÁC … 4. Chạy lại … 5. BÁO CÁO docs/integration/05-projects.md … 6. Commit báo cáo qua skill ai-usage-log … [rút gọn]

## Nội dung AI tạo ra
### Bước 01 — chore/ai-workflow
- `.gitignore`: thêm `.playwright-mcp/` (trước đó chỉ được bỏ qua nhờ `.git/info/exclude` cục bộ).
- `docs/integration/01-ai-workflow.md`: báo cáo bước 01 (test, cấu hình, code rác, việc người dùng tự kiểm, điểm không chắc chắn).
- Entry nhật ký này.

### Bước 02 — chore/scaffold
- Phân loại test lỗi `TestNewPool_RealPostgres_Pings` là lỗi có sẵn (không Docker, B12) → tạo GitHub issue #12, không sửa.
- Rà cấu hình (env, compose, CI, Makefile, tidy, lockfile, sqlc, goose, gitignore, JSON, gitleaks, golangci-lint): không cần commit `fix(config)`.
- Rà code rác (deadcode, knip, grep): mọi mục giữ lại vì dùng ở bước sau hoặc bản cuối giữ nguyên; không có commit `chore(cleanup)`.
- `docs/integration/02-scaffold.md`: báo cáo bước 02.

### Bước 03 — feature/auth
- Đọc kết quả test sau gộp: 228/228 pass, không có lỗi do gộp → không commit `fix(integration)`. Ghi bình luận vào issue #12 (test Ping đã pass khi có `TEST_DATABASE_URL`).
- Rà cấu hình: `.env.example` lệch tên biến so với `backend/cmd/api/config.go` (SMTP_USER/MAIL_FROM, thiếu APP_BASE_URL…); không sửa được vì settings chặn `Edit(./.env*)` → tạo issue #13. Các mục khác sạch, không có commit `fix(config)`.
- Rà code rác (deadcode rỗng; knip chỉ báo export/type chưa dùng, đều còn dùng ở bước sau hoặc bản cuối) → không có commit `chore(cleanup)`.
- `docs/integration/03-auth.md`: báo cáo bước 03.

### Bước 04 — feature/org
- Đọc kết quả test sau gộp: 513/513 pass, không lỗi do gộp → không commit `fix(integration)`, không tạo issue.
- Rà cấu hình: bước này không đổi file cấu hình, không thêm biến môi trường; tidy/lockfile/sqlc/goose/gitleaks/golangci-lint sạch → không commit `fix(config)`.
- Rà code rác: deadcode rỗng; knip chỉ báo export/type (`adminKeys`, `Ref`, `PaginationProps`, `SelectProps`…) giữ nguyên ở bản cuối → không commit `chore(cleanup)`.
- Ghi điểm không chắc: CEO có quyền ghi cơ cấu tổ chức (`ManageOrg`).
- `docs/integration/04-org.md`: báo cáo bước 04.

### Bước 05 — feature/projects
- Đọc kết quả test sau gộp: 693/693 pass, không lỗi do gộp → không commit `fix(integration)`, không tạo issue.
- Rà cấu hình: bước này không đổi file cấu hình, không thêm biến môi trường; tidy/lockfile/sqlc/goose (3 migration)/gitleaks/golangci-lint sạch → không commit `fix(config)`.
- Rà code rác: deadcode rỗng; knip báo thêm `projectKeys` — bản cuối `test/test-cases` dùng ở `features/kanban` → giữ; không commit `chore(cleanup)`.
- Ghi điểm không chắc: Admin 403/404, Manager phụ trách cố định, độ phủ giảm còn 90,3%, tồn đọng issue #4.
- `docs/integration/05-projects.md`: báo cáo bước 05.

### Bước 06 — feature/tasks
- Đọc kết quả test sau gộp: 831/831 pass, không lỗi do gộp → không commit `fix(integration)`, không tạo issue.
- Rà cấu hình: không thêm biến môi trường; tidy/lockfile/sqlc/goose (4 migration)/gitleaks/golangci-lint sạch → không commit `fix(config)`.
- Rà code rác: deadcode báo `domain.StoryPointScale` — không ai gọi ở cả HEAD lẫn bản cuối `test/test-cases` (FE dùng hằng `STORY_POINTS` riêng) → xóa (`chore(cleanup)`). knip báo thêm `taskKeys`, `Ref` (tasks) — bản cuối dùng ở `features/kanban` → giữ.
- `docs/integration/06-tasks.md`: báo cáo bước 06.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Bước 01: commit `fix(config)` đầu tiên bị hook `commit-msg` chặn vì trailer `AI-Log` trỏ tới entry chưa tạo | Viết commit trước khi tạo entry nhật ký | Hook báo "file không tồn tại" | Tạo entry rồi commit lại kèm entry |
| Bước 01: commit `8712025` đặt `Co-Authored-By` thành đoạn riêng sau `AI-Log`, nên git chỉ coi đoạn cuối là trailer → `ai_log.py check` báo thiếu `AI-Log` | Không để ý cách git phân tích trailer (chỉ đoạn cuối) | Chạy `ai_log.py check` sau commit | Không amend (cấm); các commit sau gộp mọi trailer vào một đoạn; ghi ở báo cáo mục f |

## Kiểm chứng
### Bước 01
- Test sau gộp và sau dọn: 0 test (chưa có backend/frontend ở bước này).
- `bash -n` các script/hook, `py_compile` các file Python: OK.
- `gitleaks detect`: no leaks found (85 commit).
- JSON hợp lệ: `.claude/settings.json`, `.mcp.json`, `.vscode/*.json`.

### Bước 02
- Test sau gộp và sau dọn: 10 test, pass 9, fail 1 (cùng test, lỗi môi trường không Docker — issue #12).
- `go mod tidy`, `npm install --package-lock-only`: không khác biệt. `goose validate`: OK. `sqlc generate`: chưa có query (đúng ở bước này).
- `golangci-lint run`: 0 issues. `deadcode -test ./...`: rỗng. `gitleaks detect`: no leaks found (87 commit).

### Bước 03
- Test sau gộp và sau dọn: 228 test, pass 228 (API 67, Unit 86, FE 75); coverage 94.0%.
- `go mod tidy`, lockfile, `sqlc generate`: không khác biệt. `goose validate`: OK. `golangci-lint run`: 0 issues. `gitleaks detect`: no leaks found.

### Bước 04
- Test sau gộp và sau dọn: 513 test, pass 513 (API 216, Unit 190, FE 107); coverage 92.4%.
- `go mod tidy`, lockfile, `sqlc generate`: không khác biệt. `goose validate` (2 migration): OK. `golangci-lint run`: 0 issues. `gitleaks detect`: no leaks found.

### Bước 05
- Test sau gộp và sau dọn: 693 test, pass 693 (API 271, Unit 289, FE 133); coverage 90.3%.
- `go mod tidy`, lockfile, `sqlc generate`: không khác biệt. `goose validate` (3 migration): OK. `golangci-lint run`: 0 issues. `gitleaks detect`: no leaks found.

### Bước 06
- Test sau gộp: 831 test, pass 831 (API 328, Unit 332, FE 171); coverage 90.3%.
- `go mod tidy`, lockfile, `sqlc generate`: không khác biệt. `goose validate` (4 migration): OK. `golangci-lint run`: 0 issues. `gitleaks detect`: no leaks found.
