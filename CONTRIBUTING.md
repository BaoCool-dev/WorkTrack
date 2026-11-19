# CONTRIBUTING.md — Quy tắc kỹ thuật WorkTrack

> Đi cùng `CLAUDE.md`. Khi mâu thuẫn với "Quy tắc cứng" trong `CLAUDE.md` §5, quy tắc cứng thắng.
> Nhiều mục gắn trực tiếp với metric trong bản cam kết (M19–M38) — ghi mã metric để biết vì sao có quy tắc.

## 0. Nguyên tắc

- **Nhàm chán là tốt.** Dùng mẫu quen thuộc của Go/React, không khéo léo.
- **Lát dọc nhỏ, luôn chạy được.** Mỗi PR là một lát có test, không để nhánh dài.
- **Server là nguồn sự thật** về thời gian, danh tính, quyền, số giờ.

## 1. Git và PR (M22, M23, M24)

- `main` được bảo vệ, chỉ merge qua PR, CI phải xanh.
- Nhánh: `feature/<ten-tinh-nang>`, `fix/<mo-ta>`, `chore/<mo-ta>`, `docs/<mo-ta>` — chữ thường, kebab-case, không dấu (xem §10).
- Conventional Commits, kèm mã yêu cầu: `feat(timesheet): chặn log giờ âm [CF-08][YCCN-31]`.
- Mỗi tuần **ít nhất 1 commit**. Không sửa ngày commit.
- PR dùng template `.github/pull_request_template.md` (checklist tự review + mã yêu cầu + cách test + mục AI log).
- Squash merge; tiêu đề PR là tiêu đề commit.

## 2. Kiểm thử (M27–M32)

| Tầng | Công cụ | Bắt buộc cho |
|---|---|---|
| Unit | `go test`, table-driven; Vitest + RTL | domain (WIP, Log time, nghiệm thu, công thức báo cáo), phân quyền, hook/component có logic |
| Integration (CSDL thật) | `go test` + testcontainers-go (đề xuất) chạy PostgreSQL | repository, migration, truy vấn báo cáo |
| API (HTTP) | `net/http/httptest` qua router chi thật | mọi endpoint Lõi: 200, 400, 401, 403, 404 |
| E2E | Playwright | 3 hành trình chính: đăng nhập → Kanban → Log time → nghiệm thu → báo cáo |
| Tải | k6 | NFR-01 (p95 ≤ 500ms), NFR-02 (200 người dùng đồng thời) |

- Coverage gói cốt lõi ≥ 70% (`make cover`). Danh sách gói cốt lõi ghi trong `docs/testing/test-plan.md`.
- Mỗi test case ghi mã AC/UC trong tên hoặc comment để làm ma trận truy vết: `TestLogTime_RejectsNegative_UC07_AC2`.
- Sửa bug = thêm test tái hiện lỗi trước (đỏ), sửa (xanh).
- Không dùng dữ liệu thật; tên người Việt giả.

## 3. Thời gian và giờ làm

- Lưu `timestamptz` (UTC). Hiển thị theo `Asia/Ho_Chi_Minh`.
- Giờ làm lưu bằng **phút (integer)**, không dùng float. Hiển thị dạng `6.5h` hoặc `6h30`.
- Hạn chót là `date` khi chỉ cần ngày; dùng `timestamptz` khi cần giờ.
- API trả thời gian ISO 8601 có offset.

## 4. CSDL và migration

- Mọi thay đổi schema qua goose, tiến một chiều. Không sửa migration đã merge.
- Mọi truy vấn qua sqlc (tham số hóa). Nối chuỗi SQL là lỗi chặn PR.
- Dữ liệu cô lập ở mức dự án: truy vấn công việc/giờ luôn lọc `project_id` và quyền thành viên.
- Index theo truy vấn thật (`EXPLAIN ANALYZE`, có thể dùng Postgres MCP ở chế độ chỉ đọc).
- Không N+1 trong danh sách và Dashboard. Danh sách có thể lớn thì phân trang.

## 5. API

- REST, JSON, prefix `/api/v1`. Xác thực mặc định bật; danh sách endpoint công khai ghi rõ trong router.
- Lỗi trả **mã ổn định + thông điệp tiếng Việt**: `{"error":{"code":"TASK_WIP_LIMIT","message":"Đã đủ 5 việc đang làm"}}`.
- Kiểm tra dữ liệu vào ở biên (validator v10) trước khi tới use case.
- Không đưa thông tin cá nhân lên query string.

## 6. Bảo mật (NFR-03, M24)

- Mật khẩu băm bằng bcrypt hoặc argon2id. Token kích hoạt lưu dạng hash.
- Phân quyền tập trung một chỗ (`internal/application/authz`); handler gọi vào, không tự viết lại.
- gitleaks trong CI; `.env*` không commit, có `.env.example`.
- Giới hạn tần suất cho đăng nhập, làm mới token, gọi AI.
- Không đưa dữ liệu nhạy cảm của người dùng vào prompt Gemini (rubric M18).

## 7. Log và vận hành (M38)

- `log/slog` dạng JSON, có request ID xuyên suốt.
- `/healthz` (sống) và `/readyz` (kết nối CSDL).
- Không log mật khẩu, token, API key.

## 8. Chất lượng mã (M19–M21, M26)

- `gofmt`/`goimports` tự động (hook), `golangci-lint` 0 lỗi, trùng lặp ≤ 3%.
- Go: theo Effective Go và Go Code Review Comments. TS: strict, không `any`, ESLint + Prettier.

## 9. Định nghĩa Hoàn thành (DoD)

- [ ] Có mã CF/UC/YCCN trong commit và PR.
- [ ] Test unit + API cho logic mới; test tích hợp nếu đụng CSDL.
- [ ] `make lint` và `make test` xanh; CI xanh.
- [ ] Chữ hiển thị tiếng Việt; UI đúng `docs/design/design-system.md`, đủ trạng thái tải/rỗng/lỗi.
- [ ] Phân quyền kiểm tra ở backend, có test 403.
- [ ] Commit có trailer `AI-Log` trỏ tới entry trong `docs/ai-log/entries/`; `python3 scripts/ai-log/ai_log.py check` không báo lỗi.
- [ ] Tài liệu (README, sơ đồ, docs/requirements) cập nhật nếu hành vi đổi.

## 10. Quy ước đặt tên

Nguyên tắc chung: **tiếng Anh, không dấu** cho mọi tên file, thư mục, định danh trong code. Tiếng Việt chỉ dùng cho nội dung hiển thị và tài liệu.

**Repo**
| Loại | Quy ước | Ví dụ |
|---|---|---|
| File gốc có tên cố định (công cụ/GitHub tự tìm) | Đúng tên chuẩn, CHỮ HOA | `README.md`, `CONTRIBUTING.md`, `CLAUDE.md`, `Makefile` |
| Thư mục, tài liệu trong `docs/` | kebab-case, chữ thường | `docs/design/design-system.md`, `docs/testing/test-plan.md` |
| File danh mục trong thư mục | `README.md` | `docs/ai-log/README.md` |
| Có thứ tự | tiền tố số 2 chữ số | `artboards/02-kanban-manager.html` |
| Có ngày | ISO `YYYY-MM-DD-` đứng đầu | `docs/ai-log/entries/2026-11-20-01-log-time.md` |
| Script shell | kebab-case `.sh` | `scripts/setup-dev.sh` |
| Script Python | snake_case `.py` (PEP 8) | `scripts/ai-log/ai_log.py`, `.claude/hooks/capture_prompt.py` |
| Skill, agent Claude | kebab-case | `.claude/skills/feature-gate/SKILL.md`, `.claude/agents/go-reviewer.md` |
| Báo cáo gate | `<loai>-<ten-nhanh>.md` (thay `/` bằng `-`) | `docs/gates/feature-kanban.md` |

**Backend (Go)**
| Loại | Quy ước | Ví dụ |
|---|---|---|
| Thư mục = tên package | một từ, chữ thường, không `_`/`-`; không trùng thư viện chuẩn | `timesheet`, `authz`, `httpapi` (không dùng `http`) |
| File | snake_case `.go` | `log_time.go`, `task_repo.go`, `errors.go` |
| Test | `<file>_test.go`; test tích hợp có build tag `integration` | `log_time_test.go` |
| Tên test | `Test<Hàm>_<TìnhHuống>_<UC>_<AC>` | `TestLogTime_RejectsNegative_UC07_AC2` |
| Kiểu, hàm export | PascalCase; viết tắt giữ hoa | `TaskRepo`, `NewLogTime`, `UserID`, `HTTPError` |
| Biến, hàm nội bộ | camelCase | `taskID`, `writeError` |
| Lỗi nghiệp vụ | `Err` + PascalCase | `ErrWIPLimitReached` |
| Migration | `YYYYMMDDHHMMSS_snake_case.sql` (goose tạo) | `20261120083000_create_time_log.sql` |
| File query sqlc | `<ten_bang>.sql` | `db/queries/time_log.sql` |
| Truy vấn sqlc | PascalCase động từ trước | `GetTask`, `CountParentInProgressByAssignee` |

**CSDL và API**
| Loại | Quy ước | Ví dụ |
|---|---|---|
| Bảng, cột | snake_case, bảng số ít (theo Chương 4) | `time_log`, `parent_task_id` |
| Index, ràng buộc | `ix_<bang>_<cot>`, `fk_…`, `ck_…`, `uq_…` | `ix_task_parent` |
| Đường dẫn API | `/api/v1/`, danh từ số nhiều, kebab-case | `/api/v1/tasks/{taskID}/time-logs` |
| Trường JSON | camelCase | `storyPoint`, `parentTaskId` |
| Mã lỗi API | UPPER_SNAKE_CASE | `TASK_WIP_LIMIT` |
| Biến môi trường | UPPER_SNAKE_CASE | `DATABASE_URL`, `GEMINI_API_KEY` |
| Sự kiện WebSocket | `<tai-nguyen>.<dong-tu-qua-khu>` | `task.moved`, `comment.created` |

**Frontend (React + TypeScript)**
| Loại | Quy ước | Ví dụ |
|---|---|---|
| Thư mục | kebab-case | `features/kanban`, `shared/ui` |
| Component | PascalCase `.tsx`, 1 component chính / file | `TaskCard.tsx`, `LogTimeModal.tsx` |
| Hook | `use` + PascalCase `.ts` | `useLogTime.ts`, `useProjectSocket.ts` |
| Module khác (api, lib, store, type) | camelCase `.ts` | `timeLogs.ts`, `format.ts`, `hoursLevel.ts` |
| File gom export | `index.ts` | `features/tasks/index.ts` |
| Test đơn vị | cạnh file, `.test.ts(x)` | `TaskCard.test.tsx` |
| Test E2E | `frontend/e2e/<luong>.spec.ts` | `e2e/kanban-drag.spec.ts` |
| Type, interface | PascalCase, không tiền tố `I` | `TaskDto`, `HoursLevel` |
| Hằng số | UPPER_SNAKE_CASE | `MAX_WIP` |
| Query key | `<tai-nguyen>Keys` | `taskKeys.detail(id)` |

**Git**
| Loại | Quy ước | Ví dụ |
|---|---|---|
| Nhánh | `<loai>/<ten>` kebab-case | `feature/kanban`, `fix/logtime-negative`, `chore/scaffold` |
| Commit | Conventional Commits + mã yêu cầu | `feat(kanban): chặn Member kéo sang Done [CF-04][UC05]` |
| Tag phiên bản | SemVer có `v` | `v0.1.0` |
