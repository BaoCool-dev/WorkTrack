# Tích hợp bước 07/9 — `docs/manual-testing` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `53709ec` (cha: `83d3b66` = HEAD sau bước 06, tag `integ/06-tasks`; `7a851af` đầu nhánh). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 4 (`53c656c`, `684c178`, `1317830`, `7a851af`), cộng merge commit `53709ec` |
| Thay đổi | 5 file, +1151 dòng, **chỉ trong `docs/`** (không đổi code, cấu hình, migration) |
| Nội dung mang vào | `docs/testing/manual-test-guide.md` (~1000 dòng): hướng dẫn kiểm thử thủ công không Docker cho CF-01, CF-02, CF-03, CF-06 (cài đặt, đăng nhập 4 vai trò, kích hoạt qua Mailpit, từng chức năng kèm ca lỗi/quyền, SQL kiểm, curl có token, đặt lại dữ liệu, checklist); 1 dòng ở `docs/testing/README.md`; 4 giả định mục C ở `docs/decisions.md`; báo cáo gate `docs/gates/docs-manual-testing.md` (ĐẠT, CI PR #8); entry nhật ký `2026-11-21-01-docs-manual-testing.md` |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip | Độ phủ domain + application |
|---|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/07-manual-testing/after-merge`) | 831 | 831 | 0 | 0 | 90,4% |
| Sau dọn (`.ai-log/integration/07-manual-testing/after-cleanup`) | 831 | 831 | 0 | 0 | 90,4% |

Chi tiết (giống nhau ở cả hai lần): backend API 328, backend Unit 332, frontend Unit/UI 171. Số Pass không đổi so với bước 06 (831/831) — đúng kỳ vọng vì nhánh chỉ có tài liệu.
Không có test lỗi → không có commit `fix(integration)`, không tạo issue mới.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| Biến môi trường | — | Bước này không đổi code. Biến hướng dẫn nhắc tới (`DATABASE_URL`, `JWT_SECRET`, `CORS_ORIGIN`) đều có trong `backend/cmd/api/config.go`. Issue [#13](https://github.com/BaoCool-dev/WorkTrack/issues/13) (`.env.example` lệch SMTP/`APP_BASE_URL`) vẫn OPEN, không làm nặng thêm | — |
| `docker-compose.yml`, `.github/workflows`, `Makefile`, `package.json`, lockfile, `go.mod`, `.gitignore`, `.mcp.json`, `.claude/settings.json`, `scripts/` | — | Không file nào đổi (`git diff --stat integ/06-tasks..HEAD -- <các đường dẫn>` rỗng) | — |
| Lệnh `make` trong hướng dẫn (`dev-local`, `seed`, `migrate-up`, `migrate-down`) | — | Đều là target có thật trong `Makefile` | — |
| Đường dẫn trong hướng dẫn (`scripts/dev-local.sh`, `scripts/seed.sh`, `backend/cmd/api/config.go`, `router.go`, `docs/setup.md`, `traceability-matrix.md`, `frontend/src/shared/api/`) | — | Tồn tại hết | — |
| Endpoint trong hướng dẫn (`$API/auth/*`, `/users…`, `/departments…`, `/job-titles`, `/skills`, `/projects/{id}/{board,columns,column-order,transitions,members,member-candidates,tasks}`, `/tasks/{id}[/claim]`, `/me/tasks`, `/me/wip`, `/labels…`, `/settings`) | — | Đối chiếu `router.go`: khớp toàn bộ. `localhost:8025/api/v1/*` là API Mailpit, không phải WorkTrack | — |
| `go mod tidy`, `npm install --package-lock-only`, `sqlc generate` | — | `git status` sạch, không khác biệt | — |
| Goose: 4 migration, `goose -dir backend/db/migrations validate` | — | rc=0, tăng dần, không trùng | — |
| `.gitignore` | — | `git status` sạch sau tidy/sqlc/chạy test | — |
| `.claude/settings.json`, `.mcp.json` | — | JSON hợp lệ | — |
| `gitleaks detect` | — | no leaks found (hướng dẫn dùng chỗ giữ `<MK_ADMIN>`… thay mật khẩu thật) | — |
| `golangci-lint run ./...` | — | 0 issues | — |

Không có commit `fix(config)` ở bước này.

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| `deadcode.txt` | Rỗng | Không có gì | — |
| `knip.txt` (24 mục export/type: `adminKeys`, `taskKeys`, `Ref`, `useSession`, `useCanAccess`, `ForbiddenState`, `healthKeys`, `fetchHealth`, `useHealth`, `TOAST_DURATION_MS`, các `*Props`, `CurrentUser`, `HealthResponse`, `ProjectMember`) | Giống hệt bước 06 — nhánh này không đổi `frontend/` | Giữ (đã xét ở bước 03–06; `taskKeys` dùng ở bước sau trong `features/kanban`) | — |
| TODO/FIXME trong code | `git grep` chỉ trúng `ErrBoardNeedsTodoColumn`/`BOARD_NEEDS_TODO_COLUMN` (tên lỗi, không phải ghi chú) | Không phải rác | — |
| "TODO" trong `manual-test-guide.md` dòng 569–573 | Biến shell `TODO=$(jq …)` giữ id cột nhóm todo | Không phải rác | — |
| Tài liệu trỏ tới thứ không còn | Xem mục (c): mọi target, script, đường dẫn, endpoint, trường JSON (`statusGroup`, `fromColumnId`, `toColumnId`) đều có thật | Không có gì | — |
| `manual-test-guide.md` khác bản cuối `test/test-cases` (+125/−3 dòng) | `git diff --stat HEAD test/test-cases -- docs/testing/manual-test-guide.md` | Giữ — bản sau mở rộng tài liệu ở bước sau; không xóa tài liệu kiểm thử | — |

Không có commit `chore(cleanup)` → không có gì phải revert.

## (e) Việc người dùng tự kiểm

```bash
git diff integ/06-tasks..HEAD --stat                      # chỉ 5 file trong docs/
git diff integ/06-tasks..HEAD -- docs/testing             # đọc hướng dẫn mới
git diff integ/06-tasks..HEAD -- docs/decisions.md        # 4 giả định mục C
make migrate-up && make seed                              # ghi lại mật khẩu in ra → thay <MK_ADMIN>, <MK_CEO>, <MK_MANAGER>, <MK_MEMBER>
make dev-local
```
- Làm theo `docs/testing/manual-test-guide.md` từ §1 tới checklist §8 trên máy không Docker (Postgres + Mailpit qua Homebrew).
- **Admin**: tạo tài khoản → mở Mailpit `http://localhost:8025` → link `http://localhost:5173/activate` kích hoạt.
- **Manager**: các bước CF-03 (dự án, thành viên, cột, luật chuyển) và CF-06 (tạo/sửa việc, nhãn).
- **Member**: nhận việc, `/my-tasks`, thử gọi `PUT /tasks/{id}` bằng curl → bị chặn.
- **CEO**: đăng nhập, xem chỉ đọc.
- Chạy thử khối SQL §7.3 (dọn dữ liệu thử) — trong phiên tự chạy chỉ được kiểm bằng `BEGIN … ROLLBACK`.

## (f) Không chắc chắn — để người dùng quyết

1. **Khối §7.3/§7.4 chưa chạy thật** (giả định mục C của nhánh): dọn dữ liệu chỉ kiểm bằng `ROLLBACK`, `goose reset` chỉ kiểm cú pháp. Người dùng nên chạy thật một lần trên CSDL nháp trước khi dùng khi thuyết trình.
2. **`COLLATE "C"` trong các câu `LIKE` tiếng Việt** là cách né lỗi PostgreSQL trên macOS (`en_US.UTF-8`). AI không chắc lỗi này tái hiện ở mọi phiên bản PostgreSQL/macOS; nếu máy chấm dùng Linux thì không cần nhưng cũng vô hại.
3. Hướng dẫn chỉ phủ CF-01/02/03/06; các bước sau (kanban, subtasks, logtime…) chưa có. Bản cuối `test/test-cases` có thêm ~125 dòng — sẽ vào ở bước sau.
4. Issue #13 (`.env.example` lệch) vẫn mở; hướng dẫn §1 dựa vào biến môi trường, người làm theo có thể vấp đúng chỗ lệch SMTP/`APP_BASE_URL`. AI không đọc/sửa `.env*` (bị chặn).
