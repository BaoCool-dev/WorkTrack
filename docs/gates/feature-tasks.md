# Gate: feature/tasks — CF-06 (công việc cha, nhận việc, Story Point quy đổi giờ, nhãn, Công việc của tôi)
**Ngày:** 2026-11-20 · **Commit cuối (code):** c63dcce · **Kết luận:** ĐẠT (kèm ghi chú — xem "Còn tồn đọng")

Phạm vi:
- **Backend:**
  - bảng `task` (khóa ngoại kép `(project_id, parent_task_id)`, CHECK `ck_task_story_point_parent_only` và `ck_task_story_point_range`), `label`, `task_label`, `app_setting` (tham số B1 `hours_per_story_point`) — goose + sqlc;
  - quy tắc thuần ở `domain/task.go`: chuẩn hóa tên (≤ 200 rune) và mô tả (≤ 5000), Story Point thang Fibonacci, ≤ 10 nhãn, hệ số 1–40 giờ/SP, chụp giờ ước tính lúc đặt SP;
  - use case `application/task`, `application/label`, `application/setting`. Nhận việc dùng `SELECT … FOR UPDATE` trong transaction;
  - endpoint mới:
    - `GET/POST /projects/{id}/tasks` (lọc nhóm trạng thái, người nhận, chưa giao, nhãn, tìm; phân trang);
    - `GET/PUT /tasks/{id}`, `POST /tasks/{id}/claim`;
    - `GET /me/tasks`, `GET /me/wip`;
    - `GET/POST/PUT/DELETE /labels`, `GET/PUT /settings`;
  - `make seed` thêm 4 nhãn và công việc mẫu, idempotent.
- **Frontend:**
  - tab "Công việc" trong chi tiết dự án: bảng, bộ lọc, form tạo/sửa, hộp chi tiết, nút "Nhận việc";
  - trang `/my-tasks` có chip WIP;
  - tab "Nhãn" và "Cấu hình" trong `/admin/users`.
- **Không làm ở nhánh này:** Kanban, kéo thả, chuyển cột, realtime, API chuyển cột (CF-04, `feature/kanban` do người dùng tự code); việc con (CF-07); bình luận; Log time (CF-08).

Gate chạy 2 vòng:
- **Vòng 1:**
  - Chạy go-reviewer, qa-tester, ui-reviewer và toàn bộ lệnh tự động.
  - go-reviewer và qa-tester: 0 mục Chặn.
  - ui-reviewer: 1 mục Chặn (lỗi #1). Đã sửa cùng các gợi ý đáng làm ở c63dcce.
- **Vòng 2:**
  - Chạy lại toàn bộ lệnh tự động ở c63dcce.
  - Kiểm UX bằng Playwright MCP ở 1366×768 cho Manager, Member, Admin, CEO.
  - Tất cả xanh.

## Tự động
Môi trường: máy không có Docker (B12).
- PostgreSQL chạy bằng Homebrew.
- Integration test chạy trên CSDL `worktrack_gate_tasks` tạo mới.
- `docker build` và `make test-integration` (testcontainers) chỉ chạy trong CI.

| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint + eslint + `tsc -b` + prettier) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: 595 test/subtest pass · FE: 171/171 test pass (16 file) |
| `go test -count=1 -tags integration ./internal/infrastructure/postgres/` (PostgreSQL local) | ✅ ok. Gồm `TestTaskRepo_ServiceFlow_CF06`: cột todo đầu tiên, nhãn, chụp giờ B1, nhận việc hai lần, lọc + phân trang, IDOR, WIP chỉ đếm việc cha, CHECK chặn việc con có SP, nhãn đang dùng không xóa được |
| `make cover` (gói cốt lõi domain + application) | ✅ 90,3% (`application/task` 87,6%, `application/setting` 88,9%) |
| `gitleaks detect --no-banner` | ✅ no leaks found |
| `go vet ./...` (kể cả tag `integration`) | ✅ |
| `python3 scripts/ai-log/ai_log.py check` | ✅ OK. Cảnh báo còn lại: mục "Phần tôi đã sửa" chưa điền |
| `docker build`, `make test-integration` (CI) | xem mục CI bên dưới |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Xác thực | Không token → 401 trên mọi endpoint công việc, nhãn, cấu hình | ✅ | `TestTaskAPI_RequiresAuth` |
| Phân quyền | <ul><li>Chỉ Manager phụ trách dự án tạo/sửa việc cha. Admin, CEO, Member, Manager khác → 403/404.</li><li>Chỉ Member nhận việc.</li><li>Chỉ Admin/CEO ghi nhãn và cấu hình; mọi vai trò được đọc.</li><li>Không có endpoint chuyển cột (ranh giới CF-04).</li></ul> | ✅ | `TestCreate_OnlyProjectManager_CF06`, `TestUpdate_PermissionAndNotFound_CF06`, `TestClaim_Rules_CF06`, `TestWrite_OnlyAdminAndCEO_CF06`, `TestTaskAPI_ForbiddenWrites_CF06`, `TestTaskAPI_NoColumnMoveEndpoint_CF04Boundary` |
| IDOR | <ul><li>Member ngoài dự án mở việc bằng ID hợp lệ → 404.</li><li>Gán người ngoài dự án → `TASK_ASSIGNEE_INVALID`.</li><li>Nhãn không tồn tại → `LABEL_NOT_FOUND`.</li></ul> | ✅ | `TestGet_Visibility_CF06`, `TestList_FiltersAndVisibility_CF06`, `TestTaskRepo_ServiceFlow_CF06` |
| SQL / lỗi 500 | <ul><li>Mọi truy vấn qua sqlc, có lọc theo `project_id` và quyền.</li><li>Tìm kiếm dùng tham số, không nối chuỗi.</li><li>Lỗi 500 trả câu chung.</li></ul> | ✅ | `backend/db/queries/task.sql`, `label.sql`; go-reviewer |
| Quy tắc nghiệp vụ | <ul><li>Việc mới vào cột đầu tiên nhóm todo.</li><li>SP chỉ ở việc cha (CHECK DB).</li><li>WIP chỉ đếm việc cha ở nhóm in_progress (CLAUDE.md §5.6).</li><li>Nhận việc chỉ với việc cha chưa giao ở nhóm todo; nhận lần hai → 409 `TASK_ALREADY_ASSIGNED`.</li><li>Đổi hệ số B1 không sửa giờ đã chụp.</li><li>Nhãn đang dùng không xóa được → 409.</li></ul> | ✅ | `TestCreate_ManagerPutsTaskInFirstTodoColumn_CF06`, `TestClaim_MemberTakesUnassignedTodo_CF06`, `TestListMine_AndWIP_CF06`, `TestUpdate_ReplaceAndEstimateSnapshot_CF06_B1`, `TestEstimateMinutes_CF06_B1`, `TestTaskRepo_ServiceFlow_CF06` |
| Validate | <ul><li>Tên rỗng, chỉ khoảng trắng, 200/201 rune, có tiếng Việt và emoji.</li><li>Mô tả 5001 ký tự.</li><li>SP ngoài thang Fibonacci; 11 nhãn; nhãn trùng.</li><li>Hệ số 0/41.</li><li>Ngày, UUID, số trang sai → 400.</li><li>FE dùng cùng luật (zod).</li></ul> | ✅ | `TestNormalizeTaskTitle_CF06`, `TestNormalizeTaskDescription_CF06`, `TestValidateStoryPoint_CF06`, `TestNormalizeLabelIDs_CF06`, `TestValidateStoryPointHours_CF06_B1`, `TestCreate_Validation_CF06`, `TestTaskAPI_BadRequests`, `ProjectTasksPanel.test.tsx` |
| HTML/script trong text | Hiển thị qua React (escape mặc định), không dùng `dangerouslySetInnerHTML` | ✅ | đọc mã |
| UX — Manager | <ul><li>Tab Công việc: bảng 7 cột, bộ lọc.</li><li>Form tạo: gợi ý "Giờ ước tính: 20h (1 SP = 4h)".</li><li>Toast "Đã tạo công việc".</li><li>Bấm đúp nút gửi chỉ tạo 1 bản ghi (4 → 5 công việc).</li></ul> | ✅ | Playwright MCP 1366×768: `cf06-manager-tasks.png`, `cf06-manager-create.png`, `cf06-manager-created-toast.png` |
| UX — Member | <ul><li>Không thấy nút "Tạo công việc".</li><li>Mở việc chưa giao → "Nhận việc" → toast "Đã nhận việc", người nhận cập nhật.</li><li>`/my-tasks` liệt kê 3 việc, chip `WIP 0` kèm giải thích "Việc con không tính vào WIP".</li></ul> | ✅ | `cf06-member-tasks.png`, `cf06-member-detail.png`, `cf06-member-claim.png`, `cf06-member-my-tasks.png` |
| UX — Admin / CEO | <ul><li>Tab Nhãn có số công việc dùng nhãn.</li><li>Xóa nhãn đang gắn 4 việc: nút "Xóa" bị khóa kèm lý do; Esc đóng hộp thoại.</li><li>Tab Cấu hình có công thức quy đổi.</li><li>`/my-tasks` → "Bạn không có quyền xem trang này" (Admin, CEO).</li></ul> | ✅ | `cf06-admin-labels.png`, `cf06-admin-label-delete-locked.png`, `cf06-admin-settings.png`, `cf06-admin-my-tasks-forbidden.png` |
| UX — trạng thái | Skeleton, rỗng theo vai trò, không khớp bộ lọc, lỗi + Thử lại, toast, lỗi 409/422 gắn đúng ô | ✅ | `ProjectTasksPanel.test.tsx`, `MyTasksView.test.tsx`, `LabelsPanel.test.tsx`, `SettingsPanel.test.tsx` |
| Gửi trùng | <ul><li>Nút gửi khóa khi đang gửi.</li><li>Nhận việc khóa bằng `FOR UPDATE`; lần hai → 409.</li></ul> | ✅ | Playwright (bấm đúp), `TestTaskRepo_ServiceFlow_CF06` |
| Agent | <ul><li>go-reviewer: 0 Chặn.</li><li>qa-tester: 0 Chặn, ma trận truy vết đã cập nhật.</li><li>ui-reviewer: 1 Chặn, đã sửa.</li></ul> | ✅ | `docs/testing/traceability-matrix.md` |

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | Nút "Xóa" nhãn đang được dùng chưa bị khóa. Người dùng chỉ biết sau khi server trả 409 (ui-reviewer, mục Chặn) | Trung bình | c63dcce |
| 2 | Chip WIP dùng `Badge` trạng thái thay vì chip viền `line-strong` theo design-system §6 (ui-reviewer) | Thấp | c63dcce |
| 3 | Thiếu test chứng minh từng vai trò bị chặn trên thao tác ghi; chưa có test WIP bỏ qua việc con và CHECK SP ở việc con (qa-tester, go-reviewer) | Trung bình | c63dcce |
| 4 | Test tích hợp kỳ vọng nhận việc chuyển cột → WIP = 1, sai với phạm vi (CF-06 không chuyển cột) | Thấp | 071c130 |
| 5 | File frontend chưa định dạng Prettier, `make lint` đỏ | Thấp | 42004e4 |

## Còn tồn đọng
- **Người dùng cần làm:** điền mục "Phần tôi đã sửa / hoàn thiện" trong `docs/ai-log/entries/2026-11-20-03-feature-tasks.md` (đang để `[NGƯỜI DÙNG ĐIỀN]`).
- **Issue [#6](https://github.com/BaoCool-dev/WorkTrack/issues/6)** (mục nhỏ, không chặn):
  - CHECK SP ở CSDL (1–100) lỏng hơn thang Fibonacci.
  - Dự án đã hủy hoặc hoàn thành vẫn tạo việc được.
  - Chưa có quy tắc cho hạn chót.
  - Chưa có test hai người cùng nhận một việc.
  - Chưa có test FE cho `/my-tasks` với Admin/CEO và cho chế độ chỉ xem của Nhãn/Cấu hình.
  - Chưa có E2E tự động.
  - Các mục nhỏ khác.
- **Giả định mục C:** 5 dòng trong `docs/decisions.md` (ngày 2026-11-20, nhánh `feature/tasks`):
  - thang Fibonacci;
  - quy tắc nhận việc;
  - giới hạn độ dài và số nhãn;
  - quyền nhãn/cấu hình;
  - chip WIP chỉ hiển thị số.

## CI (PR #7)
| Lần chạy | Commit | Backend (build → lint → test + integration testcontainers, `-race`) | Frontend (build → lint → test) | gitleaks | Docker build |
|---|---|---|---|---|---|
| [44761736025](https://github.com/BaoCool-dev/WorkTrack/actions/runs/44761736025) | 2c2663a | ✅ 1m55s | ✅ 52s | ✅ 16s | ✅ 2m5s |
