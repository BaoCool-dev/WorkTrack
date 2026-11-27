# Tích hợp bước 09/9 — `test/test-cases` → `release/v1`

- Ngày: 2026-11-27 · Chế độ tự chạy (CLAUDE.md §7c) · Thực hiện: Claude Code (claude-opus-5-5)
- Merge commit: `f43875d` (cha: `f17a33f` = HEAD sau bước 08, tag `integ/08-kanban`). Gộp không xung đột.

## (a) Nhánh gộp

| Mục | Giá trị |
|---|---|
| Số commit mang vào | 6 (`4d7eaf9` … `98ef587`), cộng merge commit `f43875d` |
| Thay đổi | 66 file, +2201 / −98 dòng |
| Chức năng mang vào | Không thêm mã ứng dụng. Kiểm thử theo chuẩn HCM-UTE cho **CF-01…CF-06**: bảng 226 test case (`docs/testing/test-cases.md`, `.csv`), báo cáo kiểm thử `docs/testing/test-report.md`, ma trận truy vết, minh chứng giao diện `docs/testing/evidence/ui/*.png`, `docs/testing/runs/` |
| Test mới | Backend: `internal/domain/tc_design_test.go`, `internal/application/authz/tc_design_test.go`, `internal/infrastructure/postgres/tc_constraints_integration_test.go` (tag `integration`). FE: `src/app/tc-design.test.tsx` |
| Migration / dependency mới | Không có |
| Script | `scripts/integrate.sh`, `scripts/day-run.sh` (mới); `auto-run.sh`, `run-all-tests.sh` (`REPO_ROOT`, bỏ qua khi chưa có backend/frontend, `npm ci` theo hash lockfile), `test-cases.sh` (token dài hạn từ Keychain) |
| Khác | `docs/integration/README.md`, mục chạy không người trông trong `docs/setup.md`, loại nhánh `release` trong CONTRIBUTING, 2 entry nhật ký AI (`2026-11-26-01-test-standard`, `2026-11-27-01-test-cases`), thêm quyền `allow` và thu hẹp `deny` trong `.claude/settings.json` |

## (b) Test

| Thời điểm | Tổng | Pass | Fail | Skip | Backend API | Backend Integration | Backend Unit | FE Unit/UI | Độ phủ domain + application |
|---|---|---|---|---|---|---|---|---|---|
| Sau gộp (`.ai-log/integration/09-test-cases/after-merge`) | 1150 | 1150 | 0 | 0 | 363 | 7 | 557 | 223 | 90,4% |
| Sau dọn (`.ai-log/integration/09-test-cases/after-cleanup`) | 1150 | 1150 | 0 | 0 | 363 | 7 | 557 | 223 | 90,4% |

So với bước 08 (974): thêm 176 test (Integration +7, Unit +166, FE +3). Test integration chạy trên CSDL riêng `worktrack_it_testcases`. Không có test nào lỗi nên không có commit `fix(integration)` và không tạo issue mới.

## (c) Cấu hình

| Vấn đề | Mức | Xử lý | Commit |
|---|---|---|---|
| Biến môi trường | — | Không thêm biến cho app. Issue [#13](https://github.com/BaoCool-dev/WorkTrack/issues/13) (`.env.example` thiếu SMTP/`APP_BASE_URL`) vẫn OPEN, bước này không làm nặng thêm | — |
| `docker-compose.yml`, `.github/workflows`, `Makefile`, `.gitignore`, `.mcp.json`, `go.mod`, `package.json` | — | Không đổi ở bước này | — |
| `go mod tidy`, `npm install --package-lock-only`, `sqlc generate` | — | `git status` sạch, không khác biệt | — |
| Goose: 5 migration, `goose -dir backend/db/migrations validate` | — | rc=0 | — |
| Script mới/sửa (`integrate.sh`, `day-run.sh`, `auto-run.sh`, `run-all-tests.sh`, `test-cases.sh`) | — | Đều 100755. Script mà `integrate.sh` sao chép (`test-summary.py`) có tồn tại. `day-run.sh` gọi `auto-run.sh --phase 2 --from feature/subtasks --base release/v1`: hợp lệ (`BASE_ARG` có, phase 2 có `feature/subtasks`) | — |
| Đường dẫn nhắc trong `docs/integration/README.md`, `docs/setup.md`, `docs/testing/test-report.md`, `integrate.sh`, `day-run.sh` | — | Đều tồn tại | — |
| `.claude/settings.json`, `.mcp.json` | — | JSON hợp lệ. Quyền mới: `go install`, `deadcode`, `npx knip`, `git merge --continue/--abort`, `git revert`, `git checkout --ours/--theirs`, `git tag -l`, `git push … release/*`, `git grep`, `goose -dir`. Không có quyền push `main`. `deny` đổi từ wildcard `.env*` sang danh sách cụ thể, xem (f) | — |
| `gitleaks detect` | — | no leaks found | — |
| `golangci-lint run ./...`, `tsc --noEmit`, `eslint` | — | 0 issues | — |

Không có commit `fix(config)` ở bước này.

## (d) Code rác

| Mục | Bằng chứng | Xử lý | Commit |
|---|---|---|---|
| `deadcode.txt` | Rỗng | Không có gì | — |
| knip | Giống bước 08 (`adminKeys`, `useSession`, `useCanAccess`, `ForbiddenState`, `healthKeys`, `fetchHealth`, `useHealth`, `backoffDelay`, `applyBoardEvent`, `UNASSIGNED`, `entersWip`, `useProject`, `afterTaskWrite`, `TOAST_DURATION_MS`, các type). Không có mục mới | Giữ (lý do như bước 03–08; còn ở bản cuối `test/test-cases`) | — |
| TODO/FIXME, `fmt.Println`, `console.log` trong diff bước này | Chỉ trúng dòng chữ trong prompt của script (`integrate.sh`) mô tả việc rà code rác, không phải lệnh gỡ lỗi | Giữ vì lý do khác | — |
| Code bị comment-out | Rà diff `backend`, `frontend`, `scripts`: không có | Không có gì | — |
| Test, migration, nhật ký AI, báo cáo gate, tài liệu kiểm thử | Thuộc loại cấm xóa | Giữ | — |

Không có commit `chore(cleanup)` nên không có gì phải revert.

## (e) Việc người dùng tự kiểm

```bash
git diff integ/08-kanban..HEAD --stat
git diff integ/08-kanban..HEAD -- backend frontend/src
git diff integ/08-kanban..HEAD -- scripts .claude/settings.json
git diff integ/08-kanban..HEAD -- docs/testing/test-cases.md docs/testing/traceability-matrix.md
make dev-local
bash scripts/test-cases.sh --no-ai     # chạy lại bảng test case, không gọi AI
```
- Mở `docs/testing/test-report.md`, đối chiếu số liệu với lần chạy mới. Xem vài ảnh trong `docs/testing/evidence/ui/` để chắc chúng khớp màn hình hiện tại.
- Thử `scripts/day-run.sh` trên máy (chỉ khi muốn chạy không người trông). Token đọc từ Keychain item `worktrack-claude-token` (`docs/setup.md` §chạy không người trông).

## (f) Không chắc chắn — để người dùng quyết

1. **`deny` trong `.claude/settings.json` bị thu hẹp.** Trước đây chặn `Read(./.env.*)` và `Edit(./.env*)`; nay chỉ chặn `.env`, `.env.local`, `.env.development`, `.env.production`, `.env.test`, `**/.env.*.local` để Claude đọc được `.env.example`. Các tên khác (vd. `.env.staging`) không còn bị chặn. Nên thêm lại nếu có dùng.
2. **Token dài hạn trong Keychain** (`claude setup-token`): token không nằm trong repo, nhưng có quyền rộng và hạn dài. Người dùng tự quyết định thu hồi sau đồ án.
3. **Quyền mới `git push … release/*`, `git revert`, `git checkout --ours/--theirs`** cho Claude tự chạy tích hợp. Xác nhận có muốn giữ sau khi xong `release/v1` không.
4. **`docs/testing/test-report.md` và `runs/`** là kết quả tại một thời điểm. Sau khi chỉnh sửa (commit `Human-Edit: yes`) cần chạy lại `test-cases.sh` để số liệu khớp.
5. Issue #13 (`.env.example`) vẫn mở. AI không đọc/sửa `.env*` (bị chặn).
