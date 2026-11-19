# docs/setup.md — Cài và dùng bộ cấu hình Claude Code cho WorkTrack

Một bộ cấu hình, dùng chung cho **terminal** (`claude`) và **VS Code** (extension Claude Code): cả hai đọc cùng `CLAUDE.md`, `.claude/settings.json`, `.claude/skills/`, `.claude/agents/`, `.mcp.json`.

## 0. Bộ này gồm gì

| Thành phần | File | Vai trò |
|---|---|---|
| Bộ nhớ dự án | `CLAUDE.md`, `backend/CLAUDE.md`, `frontend/CLAUDE.md` | Bối cảnh, stack, quy tắc cứng. File con tự nạp khi Claude làm trong thư mục đó |
| Thiết kế | `docs/design/design-system.md` | Token màu/chữ, quy tắc UI. `frontend/CLAUDE.md` import file này |
| Quy tắc kỹ thuật | `CONTRIBUTING.md` | Git, test, CSDL, API, bảo mật, DoD — gắn mã metric bản cam kết |
| Skills dự án | `.claude/skills/*` | `feature-gate`, `implement-ticket`, `go-backend`, `db-migration-sqlc`, `react-feature`, `worktrack-domain`, `realtime-websocket`, `gemini-ai`, `ai-usage-log` |
| Subagents | `.claude/agents/*` | `go-reviewer`, `qa-tester`, `ui-reviewer` |
| Hooks | `.claude/hooks/*` | Tự `goimports`/`prettier` sau khi sửa; chặn sửa tay sqlcgen và migration đã commit; **ghi nguyên văn prompt** vào `.ai-log/` cho nhật ký AI |
| Nhật ký AI | `docs/ai-log/`, `scripts/ai-log/ai_log.py`, `.githooks/commit-msg` | Mỗi việc 1 entry, commit gắn trailer `AI-Log`, git chặn commit AI thiếu nhật ký, xuất đúng 5 ô form nộp bài — xem `docs/ai-log/README.md` |
| Quyền | `.claude/settings.json` | Cho phép lệnh build/test/lint; hỏi trước push/migrate; cấm đọc `.env`, force-push, `rm -rf` |
| Plugins | `.claude/settings.json` → `enabledPlugins` | Bật tự động cho repo (xem §2) |
| MCP | `.mcp.json` | context7, gopls, postgres (chỉ đọc), playwright |
| VS Code | `.vscode/` | Cấu hình Go/TS/Tailwind + extension gợi ý |

## 1. Cài công cụ (một lần)

```bash
chmod +x scripts/setup-dev.sh && ./scripts/setup-dev.sh
```
Cài: Go, Node, gopls, goimports, goose, sqlc, golangci-lint, gitleaks, k6, jq, typescript-language-server.
Plugin LSP **không chứa** language server — phải có `gopls` và `typescript-language-server` trong `PATH` trước.

Biến môi trường cho MCP (đặt trong `~/.zshrc`, **không** trong repo):
```bash
export CONTEXT7_API_KEY=...            # không bắt buộc, có thì ít bị giới hạn tần suất
export WORKTRACK_MCP_DATABASE_URI=postgresql://worktrack_ro:...@host.docker.internal:5432/worktrack
```
Postgres MCP chạy trong Docker nên dùng `host.docker.internal` thay cho `localhost`. Nên tạo user CSDL **chỉ đọc** cho MCP.

## 2. Plugins

Khai báo sẵn trong `.claude/settings.json`; lần đầu mở repo, Claude Code hỏi cài — chọn đồng ý.

| Plugin | Nguồn | Vì sao |
|---|---|---|
| `gopls-lsp` | claude-plugins-official | Claude thấy lỗi kiểu Go ngay sau mỗi lần sửa, nhảy định nghĩa/tham chiếu |
| `typescript-lsp` | claude-plugins-official | Như trên cho TS/TSX |
| `superpowers` | claude-plugins-official | Kỹ năng quy trình: TDD, debug có hệ thống, lập kế hoạch, review |
| `security-guidance` | claude-plugins-official | Tự soát lỗ hổng khi Claude sửa code; đọc thêm `.claude/claude-security-guidance.md`. Có gọi model riêng → tốn thêm lượt dùng; tắt bằng `/plugin disable` nếu cần |
| `commit-commands` | claude-plugins-official | Lệnh commit/PR nhanh |
| `cc-skills-golang` | `samber/cc` (cộng đồng) | Bộ skill Go chất lượng (code style, database, testing…) |

Cài tay (nếu không được hỏi), trong phiên `claude`:
```
/plugin marketplace add samber/cc
/plugin install gopls-lsp@claude-plugins-official
/plugin install typescript-lsp@claude-plugins-official
/plugin install superpowers@claude-plugins-official
/plugin install security-guidance@claude-plugins-official
/plugin install commit-commands@claude-plugins-official
/plugin install cc-skills-golang@samber
/reload-plugins
```

**Lưu ý xung đột:** skill cộng đồng có thể gợi ý thư viện khác stack đã chốt (vd. ORM, framework khác chi). `CLAUDE.md` ghi rõ "file này thắng" và có danh sách thư viện bị cấm. Nếu `cc-skills-golang` gây nhiễu, tắt nó và chỉ cài lẻ skill cần: `npx skills add https://github.com/samber/cc-skills-golang --skill golang-code-style`.

**Không bật `frontend-design`:** plugin này đẩy Claude tới phong cách "táo bạo, khác biệt", trái với `docs/design/design-system.md` đã chốt. Chỉ bật tạm khi muốn thử ý tưởng giao diện mới.

## 3. MCP servers (`.mcp.json`)

| Server | Dùng khi | Ghi chú |
|---|---|---|
| `context7` | Tra tài liệu đúng phiên bản: pgx, sqlc, chi, coder/websocket, Gemini SDK, TanStack Query, Tailwind v4 | Remote HTTP |
| `gopls` | Tìm symbol, tham chiếu, ảnh hưởng thay đổi trong code Go | `gopls mcp`, cần gopls mới |
| `postgres` | Xem schema, `EXPLAIN ANALYZE`, kiểm tra dữ liệu dev | Postgres MCP Pro, `--access-mode=restricted` (chỉ đọc); luôn hỏi trước khi dùng |
| `playwright` | `ui-reviewer` chụp màn hình 1366×768, chạy thử luồng UI | Cần app đang chạy |

GitHub: dùng `gh` CLI (đã cho phép `gh issue view`, `gh pr view`…), không cần MCP riêng. Chạy `gh auth login` một lần.

Kiểm tra: trong `claude` gõ `/mcp`; ngoài terminal: `claude mcp list`.

## 4. Terminal

```bash
cd worktrack
claude                       # phiên tương tác
claude --continue            # tiếp phiên trước
```
Lệnh hay dùng trong phiên: `/plugin`, `/mcp`, `/agents`, `/memory` (sửa CLAUDE.md), `/permissions`, `/compact`, `/clear` (sang việc mới).

## 5. VS Code

1. Mở thư mục gốc repo (không mở riêng `backend/`).
2. Cài extension gợi ý (VS Code tự hiện thông báo từ `.vscode/extensions.json`), trong đó có **Claude Code** (`anthropic.claude-code`).
3. Mở panel Claude Code → dùng giống terminal; plugin, MCP, skills, hooks là **cùng một bộ**.
4. Có thể chạy `claude` trong terminal tích hợp của VS Code — cả hai cách đều đọc cấu hình trong repo.

## 6. Kiểm tra bộ cấu hình hoạt động

Trong một phiên mới, lần lượt hỏi:

| Câu hỏi | Kết quả đúng |
|---|---|
| "Tóm tắt quy tắc cứng của dự án" | Liệt kê từ CLAUDE.md §5 |
| "Dùng context7 tra cách tạo pgxpool" | Có lời gọi tool context7 |
| "Sửa file trong sqlcgen/x.go" | Hook chặn, nhắc sửa `.sql` + `make sqlc` |
| "Đọc file .env" | Bị từ chối theo `permissions.deny` |
| "Hệ số Story Point sang giờ là bao nhiêu?" | Claude nói **chưa chốt** và hỏi lại, không tự đặt số |
| "Làm issue #1" | Claude dùng skill `implement-ticket`, gọi `gh issue view 1` |

## 7. Cách dùng hằng ngày

```text
Làm issue #12 (CF-08 Log time). Theo implement-ticket.
```
→ Claude đọc issue, đọc UC, lập kế hoạch, viết test trước, code, chạy lint/test, đề xuất commit, ghi AI log.

```text
Dùng agent go-reviewer soát nhánh hiện tại.
Dùng agent ui-reviewer soát màn Kanban.
Dùng agent qa-tester kiểm tra UC07 đã đủ test chưa.
```

## 8. Việc người dùng cần tự làm

- [ ] Commit đầu tiên của repo (bộ cấu hình này do AI soạn) phải kèm nhật ký:
  ```bash
  git add -A && git commit -m "chore: khởi tạo cấu hình Claude Code và hệ thống thiết kế" \
    -m "AI-Assisted: Claude (claude-opus-5-5)
  AI-Log: docs/ai-log/entries/2026-11-18-01-claude-code-setup.md
  AI-Log: docs/ai-log/entries/2026-11-19-01-design-md-ai-log.md"
  ```
- [ ] Điền mục "Phần tôi đã sửa / hoàn thiện" trong 2 entry trên, và thêm entry `commit: none` cho các lần dùng AI trước đây (Claude Design, soạn Chương 1–4, form khảo sát).

- [ ] Chép nội dung Chương 3–4 và bản cam kết sang Markdown trong `docs/requirements/` (xem README ở đó).
- [ ] Xuất 13 artboard từ Claude Design ("WorkTrack – Thiết kế giao diện") vào `docs/design/`, giữ đúng tên file (xem `docs/design/README.md`). Claude Code không mở được link claude.ai, nên phải có bản xuất trong repo.
- [ ] Chốt 2 điểm lệch thiết kế: mốc đúng 20% và cách hiển thị mã lỗi (`docs/design/design-system.md` §10).
- [ ] Chốt: hệ số Story Point ↔ giờ, ma trận phân quyền, tên gói `application`, cấu hình JWT → cập nhật `CLAUDE.md` §6.
- [ ] Xác nhận thư viện frontend đề xuất trong `frontend/CLAUDE.md`.
- [ ] Kiểm tra lại phiên bản Go và các thư viện tại trang chính thức trước khi `go mod init`.
- [ ] Bật branch protection cho `main` trên GitHub (bắt buộc PR + CI xanh).
