# Tích hợp bước 01/9 — `chore/ai-workflow` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `5bbb07b` (cha: `dca8911` khởi tạo repo, `1154364` đầu nhánh). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 3 (`ea92bd1`, `77c88be`, `1154364`) |
| Thay đổi | 46 file, +3004 / −9 dòng (`git diff --stat dca8911..5bbb07b`) |
| Chức năng mang vào | Quy trình AI: `.claude/` (9 skill, 3 agent, 3 hook, `settings.json`), `.githooks/commit-msg` (chặn commit AI thiếu nhật ký), `.mcp.json` (context7, gopls, postgres, playwright), `scripts/ai-log/ai_log.py`, `scripts/auto-run.sh`, `scripts/setup-dev.sh`, `Makefile`, `CLAUDE.md`, `backend/CLAUDE.md`, `frontend/CLAUDE.md`, `CONTRIBUTING.md`, `docs/` (decisions, design-system, ai-log, README các thư mục), `.github/pull_request_template.md` |
| Code sản phẩm | Không có (chưa có `backend/` Go hay `frontend/src` ở bước này) |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip |
|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/01-ai-workflow/after-merge`) | 0 | 0 | 0 | 0 |
| Sau dọn (`.ai-log/integration/01-ai-workflow/after-cleanup`) | 0 | 0 | 0 | 0 |

Chưa có backend/frontend nên không có test. Không có test lỗi → không cần phân loại, không tạo issue. Số Pass không giảm.
`deadcode.txt` và `knip.txt` không được sinh ở bước này (không có code Go/FE).

Kiểm tra bổ sung đã chạy:
- `bash -n` cho `scripts/*.sh`, `.claude/hooks/*.sh`, `.githooks/commit-msg`: OK.
- `python3 -m py_compile scripts/ai-log/ai_log.py .claude/hooks/capture_prompt.py`: OK.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| `.gitignore` không phủ `.playwright-mcp/` (log console do Playwright MCP sinh); máy này chỉ bỏ qua được nhờ `.git/info/exclude` cục bộ | Thấp | Thêm `.playwright-mcp/` (giống bản cuối `test/test-cases`) | `8712025` |
| `.claude/settings.json`, `.mcp.json`, `.vscode/settings.json`, `.vscode/extensions.json` | — | JSON hợp lệ, không sửa | — |
| `gitleaks detect` (85 commit, ~2,3 MB) | — | no leaks found | — |
| `Makefile`: target `tools` → `scripts/setup-dev.sh` | — | Có thật, có quyền thực thi | — |
| `Makefile`: `dev`, `test`, `lint`, `sqlc`, `migrate-*`, `cover` trỏ tới `backend/`, `frontend/` chưa có code | Thông tin | Giữ — comment đầu Makefile đã nói rõ; có ở bước sau (`chore/scaffold`) | — |
| `.github/workflows/` chưa có; `scripts/auto-run.sh` nhắc `.github/workflows/ci.yml` (trong mô tả việc cho nhánh scaffold) | Thông tin | Giữ — `ci.yml` có ở bản cuối, mang vào ở bước sau | — |
| `docker-compose.yml`, `go.mod`, `package.json`, `sqlc.yaml`, migration goose | — | Chưa tồn tại ở bước này → bỏ qua `go mod tidy`, `npm`, `sqlc generate`, `goose validate` | — |
| `.env.example` so với biến môi trường code đọc | — | Không có code đọc biến môi trường ở bước này. **Không đọc nội dung `.env.example`** vì `settings.json` chặn `Read(./.env.*)` và lệnh đọc bị từ chối (xem mục f) | — |
| Hook commit-msg: trailer phải nằm ở **đoạn cuối** commit message | Thấp | Commit `8712025` tách `Co-Authored-By` thành đoạn riêng nên `ai_log.py check` báo thiếu `AI-Log` (hook vẫn cho qua vì dùng grep). Không amend (cấm); các commit sau gộp mọi trailer vào một đoạn | — |

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| TODO/FIXME/XXX | `git grep -nE 'TODO\|FIXME\|XXX'` (trừ `docs/requirements`): 0 kết quả | Không có gì để dọn | — |
| `console.log` / `fmt.Print` gỡ lỗi | `git grep` trong `scripts .claude .githooks`: 0 kết quả | — | — |
| Hàm Go không tới được / export FE không dùng | Chưa có code; `deadcode.txt`, `knip.txt` không có | — | — |
| Tham chiếu tới `.github/workflows/ci.yml`, `docs/testing/traceability-matrix.md` | Không có ở bước này; `git cat-file -e test/test-cases:<file>` → có | Giữ vì dùng ở bước sau | — |
| Tham chiếu tới `docs/requirements/commitment.md`, `docs/testing/test-plan.md`, `docs/design/artboards/*.html` (CLAUDE.md, CONTRIBUTING.md, `docs/design/README.md`, skill `react-feature`) | Không có ở bước này **và cả bản cuối** `test/test-cases` | Giữ vì lý do khác: là tài liệu nguồn/thiết kế người dùng cần bổ sung, không phải rác — xem mục f | — |
| Ví dụ đường dẫn minh họa (`docs/ai-log/entries/2026-11-20-01-log-time.md`, `docs/gates/feature-kanban.md`) | Chỉ xuất hiện trong ví dụ của hook/skill/CONTRIBUTING | Giữ vì lý do khác: ví dụ, không phải tham chiếu thật | — |

Không có commit `chore(cleanup)` ở bước này.

## (e) Việc người dùng tự kiểm

```bash
git diff main..HEAD --stat
git diff main..HEAD -- .claude/ .githooks/ scripts/ .mcp.json Makefile .gitignore
git config core.hooksPath            # phải là .githooks
python3 scripts/ai-log/ai_log.py check
```
- `make dev-local` chưa có ở bước này (chưa có `backend`/`frontend`); không có màn hình hay vai trò để thử.
- Mở Claude Code ở gốc repo: kiểm `/mcp` thấy context7, gopls, playwright (postgres cần Docker); thử `Edit` một file trong `backend/internal/infrastructure/postgres/sqlcgen/` phải bị hook `protect-generated.sh` chặn.
- Thử commit có `AI-Assisted:` mà thiếu `AI-Log:` → hook phải chặn.

## (f) Không chắc chắn — để người dùng quyết

1. **`.env.example` chưa được rà.** Quyền trong `.claude/settings.json` chặn đọc `.env.*`, kể cả `.env.example`. Người dùng nên xem `.env.example` có đủ `DATABASE_URL`, `CONTEXT7_API_KEY`, `WORKTRACK_MCP_DATABASE_URI` (được `Makefile`/`.mcp.json` dùng). Có thể thêm `"Read(./.env.example)"` vào `allow` nếu muốn AI rà được — AI không tự đổi vì đây là quy tắc §5.10.
2. **Tài liệu nguồn chưa có trong repo kể cả bản cuối:** `docs/requirements/commitment.md` (CLAUDE.md §1 coi là phạm vi cam kết), `docs/testing/test-plan.md`, 13 artboard `docs/design/artboards/*.html`. AI không tự tạo vì là nội dung nghiệp vụ/thiết kế của người dùng.
3. Commit `8712025` bị `ai_log.py check` báo `✗ thiếu trailer AI-Log` do định dạng trailer. Không sửa lịch sử được (cấm amend/rebase). Người dùng quyết: chấp nhận cảnh báo, hoặc sửa `ai_log.py` dùng grep giống hook.
4. `.git/info/exclude` cục bộ đang chứa `.playwright-mcp/`; giờ trùng với `.gitignore`, có thể xóa dòng đó nếu muốn (không ảnh hưởng gì).
