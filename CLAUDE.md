# CLAUDE.md — WorkTrack

> Hướng dẫn cấp dự án cho Claude Code (terminal và VS Code dùng chung file này).
> Đọc file này trước mọi việc. Khi file này mâu thuẫn với một skill/plugin bên ngoài, **file này thắng**.
> Trả lời người dùng bằng **tiếng Việt**. Code, tên biến, commit message type giữ tiếng Anh.

## 1. Dự án là gì

**WorkTrack** — web app quản trị công việc và cộng tác nội bộ cho doanh nghiệp CNTT vừa và nhỏ (SME).
Ý tưởng cốt lõi: thay chấm công theo giờ có mặt bằng **ghi giờ thực tế trên từng đầu việc** (Task-based Timesheet),
dùng dữ liệu đó đo hiệu suất minh bạch, kèm **Trợ lý AI (Gemini)** chỉ gợi ý — quyết định luôn thuộc về con người.

- Đồ án tốt nghiệp, **1 sinh viên**, code từ tuần 10 đến tuần 15. Phạm vi nhỏ, chạy ổn định quan trọng hơn nhiều tính năng.
- 4 vai trò: **Admin, Manager, Member, CEO**. Desktop-first, tối thiểu 1366×768. Không làm mobile.
- Nguồn sự thật nghiệp vụ: `docs/requirements/` (UC, YCCN, quy tắc R.., quyết định D..). Tham chiếu bằng mã: `UC05`, `YCCN-12`, `R09`, `CF-04`. **Không tự đặt mã mới.**
- Phạm vi cam kết: `docs/requirements/commitment.md` — 12 chức năng Lõi `CF-01..CF-12`. Phần "Mở rộng" **không làm** cho tới khi Lõi xong.

## 2. Tech stack (đã chốt — không tự đổi)

| Lớp | Lựa chọn |
|---|---|
| Backend | Go (xem `backend/go.mod`), Clean Architecture, router **chi**, **pgx v5**, **sqlc** (pgx/v5), migration **goose**, **golang-jwt v5**, **go-playground/validator v10** |
| Realtime | WebSocket, **coder/websocket**; xác thực JWT lúc kết nối + kiểm tra quyền theo dự án |
| CSDL | PostgreSQL |
| AI | Google Gemini API, gọi qua một cổng (interface) trong tầng application |
| Frontend | React 19 + TypeScript strict + Vite + TailwindCSS v4 |
| Đóng gói | Docker, docker compose; CI GitHub Actions |

Thư viện frontend phụ (TanStack Query, Zustand, dnd-kit, react-hook-form + zod, Recharts, Vitest/RTL/MSW, Playwright) là **đề xuất** — xem `frontend/CLAUDE.md`. Thêm dependency mới phải ghi 1 dòng lý do trong PR.

**Bị cấm:** ORM (GORM, ent…), framework web khác (gin, echo, fiber), .NET/CQRS/MediatR (đã bỏ), Redux, axios-interceptor tự chế thay TanStack Query, CSS-in-JS, UI kit nặng (MUI, Ant Design).

## 3. Cấu trúc repo

```
.claude/                      skills, agents, hooks, settings của Claude Code (quy trình AI)
.githooks/                    hook git: commit-msg kiểm tra nhật ký AI
.github/                      PR template, workflows CI
backend/                      Go API — đọc backend/CLAUDE.md khi làm ở đây
frontend/                     React app — đọc frontend/CLAUDE.md khi làm ở đây
docs/
  ai-log/                     Nhật ký sử dụng AI, mỗi việc 1 file (rubric M15–M18)
  design/design-system.md     Hệ thống thiết kế UI — bắt buộc khi làm giao diện
  design/artboards/           13 artboard đã chốt (NN-ten-man-hinh.html)
  gates/                      Báo cáo cổng kiểm tra từng nhánh tính năng
  requirements/               UC, YCCN, business rules, bản cam kết (nguồn sự thật)
  testing/                    test-plan.md, traceability-matrix.md, evidence/
  setup.md                    Cài công cụ, plugin, MCP (terminal + VS Code)
scripts/                      setup-dev.sh, ai-log/ai_log.py
CLAUDE.md · CONTRIBUTING.md · README.md · Makefile · .env.example · .mcp.json
```
Quy ước đặt tên: `CONTRIBUTING.md` §10. Tạo file/thư mục mới phải theo đúng bảng đó.

## 4. Lệnh chuẩn (chạy từ gốc repo)

```bash
make dev            # docker compose: postgres + api + web
make test           # go test ./... + vitest run
make lint           # golangci-lint run + eslint + tsc --noEmit
make sqlc           # sinh lại code từ backend/db/queries/*.sql
make migrate-up     # goose up
make migrate-new name=<ten_migration>
make cover          # coverage các gói cốt lõi
```

Trước khi báo "xong" một việc: **`make lint` và `make test` phải xanh.** Không có ngoại lệ.

## 5. Quy tắc cứng — không phá nếu chưa được người dùng đồng ý

1. **Server là nguồn sự thật về quyền.** Mọi endpoint kiểm tra vai trò + quyền theo dự án ở backend. Ẩn nút ở UI không phải là phân quyền.
2. **CEO xem Dashboard tổng quan toàn công ty ở chế độ chỉ xem** — không endpoint ghi nào cho màn này.
3. **Admin không tạo dự án. Member không giao việc cho người khác.** Chỉ Manager tạo dự án và công việc cha.
4. **Công việc cha / con:** Story Point chỉ ở cha (CHECK `story_point IS NULL` ở con). Con 1 cấp, Member tự tạo, tự chuyển cột, không đi qua luật `workflow_transition`. Nghiệm thu chỉ ở cha.
5. **Chỉ Manager chuyển công việc sang nhóm hoàn thành.** Cột "Chờ nghiệm thu" thuộc nhóm `in_progress`.
6. **WIP chỉ đếm công việc cha** (`parent_task_id IS NULL`).
7. **Log time:** không âm, không log vào việc đã Done quá 24 giờ. Giờ của con cộng lên cha trong báo cáo.
8. **Năng suất và tỷ lệ tuân thủ chỉ tính trên công việc gốc.** Công thức trong skill `worktrack-domain`.
9. **Kết quả AI luôn là bản xem trước**, không tự ghi vào CSDL. Mỗi lần gọi Gemini ghi `ai_usage_log` kèm thời gian phản hồi. Có timeout.
10. **Không secret trong repo.** Cấu hình qua biến môi trường. Không đọc/sửa `.env*` (đã chặn trong settings).
11. **Không sửa code sinh tự động** (`backend/internal/infrastructure/postgres/sqlcgen/`). Sửa file `.sql` rồi `make sqlc`.
12. **Không sửa migration đã áp dụng.** Tạo migration mới.
13. **Toàn bộ chữ hiển thị cho người dùng bằng tiếng Việt**, giữ thuật ngữ quen thuộc (Kanban, Story Point, Log time, Dashboard).

## 6. Việc CHƯA CHỐT — phải hỏi, không được đoán

- Tỷ lệ quy đổi **Story Point ↔ giờ** (BRD có ghi chú 1 SP : 4 giờ nhưng chưa chốt).
- **Ma trận phân quyền hợp nhất 4 vai trò** (CEO kế thừa quyền Admin tới đâu).
- Tên gói tầng ứng dụng: mặc định `internal/application`, chờ xác nhận.
- Ngưỡng giờ **đúng bằng 20%** thuộc mức vàng hay đỏ (tạm: ≥ 20% là đỏ) — xem `docs/design/design-system.md` §10.
- Mã lỗi hiển thị cho người dùng: thiết kế ghi dạng "WT-5031", API đang dùng mã chữ (`TASK_WIP_LIMIT`). Cần chốt có ánh xạ sang mã số không.
- JWT (đang tạm chọn): Access 15 phút, Refresh 7 ngày, HS256; Access giữ trong bộ nhớ React, Refresh trong cookie HttpOnly + Secure + SameSite=Strict.

Gặp các điểm này: dừng, hỏi **một lần, kèm 2–3 phương án**, rồi làm theo câu trả lời.

## 7. Cách làm việc với Claude trong repo này

- **Mỗi phiên code bắt đầu bằng skill `implement-ticket`** (đọc issue → đọc UC/YCCN → lập kế hoạch → test trước → code → lint/test → ghi AI log).
- Làm theo **lát dọc**: đăng nhập → dự án → công việc → Log time → báo cáo → AI. Mỗi lát chạy được end-to-end rồi mới sang lát sau.
- Tra tài liệu thư viện bằng **Context7 MCP** thay vì đoán API (pgx, sqlc, chi, coder/websocket, TanStack Query, Tailwind v4 thay đổi nhanh).
- Tra code Go bằng **gopls** (MCP/LSP) trước khi grep thủ công.
- Không chắc về kiến thức → nói rõ là không chắc. Không bịa phiên bản, API, số liệu.
- **Mọi commit có AI tham gia đi qua skill `ai-usage-log`**: entry trong `docs/ai-log/entries/` + trailer `AI-Assisted` và `AI-Log` trong commit message. Hook git chặn commit thiếu nhật ký. Đây là tiêu chí chấm điểm.
- Commit theo Conventional Commits, kèm mã yêu cầu: `feat(kanban): chặn Member kéo sang Done [CF-04][UC05]`. Không tự `git push`.

## 7a. Những gì được commit lên GitHub

**Commit:** mã nguồn (`backend/`, `frontend/`), cấu hình build/chạy/CI, **và toàn bộ quy trình AI** (`.claude/`, `CLAUDE.md`, `.mcp.json`, `.githooks/`, `scripts/`, `docs/` gồm nhật ký AI và báo cáo gate) — để hội đồng thấy và người dùng giải thích được khi vấn đáp.
**Không bao giờ commit:** `.env*` (trừ `.env.example`), `.ai-log/` (prompt thô riêng tư), `dist/`, `.claude/settings.local.json`, `CLAUDE.local.md`. Đã có trong `.gitignore`.
Dùng `git add <đường dẫn cụ thể>`; xem `git status` trước khi add.

## 7b. Quy trình nhánh và tính năng Kanban

**Một tính năng = một nhánh = một PR.** Thứ tự: `chore/scaffold` → `feature/auth` (CF-01) → `feature/org` (CF-02) → `feature/projects` (CF-03) → `feature/tasks` (CF-06) → **`feature/kanban` (CF-04, CF-05)** → `feature/subtasks` (CF-07) → `feature/logtime` (CF-08) → `feature/review` (CF-09) → `feature/reports` (CF-10, CF-11) → `feature/ai` (CF-12).
- Luôn tạo nhánh mới từ `main` vừa `git pull`. Không làm hai tính năng trên một nhánh.
- **Chưa qua skill `feature-gate` thì không mở PR và không bắt đầu tính năng tiếp theo.** Người dùng tự merge PR.
- Hỏi trước khi `git push` và `gh pr create`.

**Nhánh `feature/kanban*` do người dùng TỰ CODE** (để thuyết trình với giảng viên). Trên các nhánh này Claude:
- **KHÔNG** viết, sửa hay sinh code sản phẩm (Go trong `backend/internal/`, TS/TSX trong `frontend/src/`), kể cả khi được nhờ "viết giúp một đoạn nhỏ". Nhắc lại quy tắc này và đề xuất cách hướng dẫn thay thế.
- **ĐƯỢC:** giải thích khái niệm và tài liệu thư viện, gợi ý hướng làm bằng lời hoặc mã giả, review code người dùng viết (chỉ ra lỗi + lý do), viết và chạy test, chạy `feature-gate`, trả lời câu hỏi chuẩn bị vấn đáp.
- Mọi hỗ trợ trên nhánh này vẫn ghi vào nhật ký AI với `type: review` hoặc `type: test`, để minh bạch phần nào người dùng tự làm.
- Kiểm tra nhánh hiện tại bằng `git branch --show-current` trước khi sửa file.

## 8. Bản đồ skill / agent

| Khi nào | Dùng |
|---|---|
| Bắt đầu một issue/tính năng | skill `implement-ticket` |
| Viết Go: handler, use case, repository | skill `go-backend` |
| Thêm bảng, cột, truy vấn | skill `db-migration-sqlc` |
| Màn hình, component, gọi API ở FE | skill `react-feature` + `docs/design/design-system.md` |
| Quy tắc nghiệp vụ (WIP, cha/con, Log time, nghiệm thu, báo cáo) | skill `worktrack-domain` |
| Kanban realtime, WebSocket | skill `realtime-websocket` |
| Gemini: phân rã, gợi ý nhân sự, cảnh báo trễ | skill `gemini-ai` |
| Commit bất kỳ thay đổi có AI tham gia; xuất nhật ký để nộp | skill `ai-usage-log` |
| Xong một tính năng, trước PR / trước tính năng tiếp theo | skill `feature-gate` |
| Review code Go trước PR | agent `go-reviewer` |
| Viết/chạy test, đối chiếu AC | agent `qa-tester` |
| Soát giao diện theo docs/design/design-system.md | agent `ui-reviewer` |
