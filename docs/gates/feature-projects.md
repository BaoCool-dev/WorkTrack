# Gate: feature/projects — CF-03 (quản lý dự án, thành viên, cột Kanban, luật chuyển)
**Ngày:** 2026-11-20 · **Commit cuối (code):** 4cfd063 · **Kết luận:** ĐẠT (kèm ghi chú — xem "Còn tồn đọng")

Phạm vi:
- **Backend:**
  - bảng `project`, `project_member`, `board_column` (vị trí unique DEFERRABLE), `workflow_transition` (khóa ngoại kép `(project_id, id)`, ON DELETE CASCADE) — goose + sqlc;
  - quy tắc thuần ở `domain/project.go`: chuẩn hóa tên, 5 trạng thái, 2–8 cột, luôn có cột nhóm done, luật chuyển mặc định, chặn Member chuyển vào/ra nhóm done;
  - use case `application/project` với port `Store` + `Transactor` (tạo dự án kèm 5 cột + luật mặc định trong một transaction);
  - 14 endpoint dưới `/api/v1/projects`: danh sách theo vai trò, tạo/xem/sửa, thành viên + ứng viên, bảng, thêm/sửa/xóa cột, `/column-order`, `/transitions`;
  - `make seed` thêm 2 dự án mẫu (B11), idempotent.
- **Frontend:** màn `/projects` (lọc, tìm, phân trang, form tạo/sửa) và `/projects/:projectId` với 4 tab Thông tin, Thành viên, Cột Kanban, Luật chuyển; nút ẩn theo vai trò; `shared/ui/DataTable`.
- **Không làm ở nhánh này:** áp dụng luật chuyển khi kéo thẻ (để dành `feature/kanban`, người dùng tự code); số việc thật trong cột (chờ CF-06).

Gate chạy 2 vòng:
- **Vòng 1:** go-reviewer, ui-reviewer, qa-tester và toàn bộ lệnh tự động. go-reviewer và ui-reviewer không có mục Chặn; qa-tester phát hiện lệch quy tắc Member ra khỏi nhóm done (lỗi #1). Đã sửa ở 4cfd063.
- **Vòng 2:** chạy lại toàn bộ lệnh tự động và kiểm tra UX bằng Playwright MCP. Tất cả xanh.

## Tự động
Môi trường: máy không có Docker (B12). PostgreSQL chạy bằng Homebrew; integration test chạy trên CSDL `worktrack_gate` tạo mới. `docker build` và `make test-integration` (testcontainers) chỉ chạy trong CI.

| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint + eslint + `tsc -b` + prettier) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: 503 test/subtest pass ở mọi gói · FE: 133/133 test pass (12 file) |
| `go test -count=1 -tags integration ./internal/infrastructure/postgres/` (PostgreSQL local) | ✅ ok — gồm `TestProjectRepo_ServiceFlow_CF03` (transaction, đảo thứ tự cột nhờ DEFERRABLE, cascade luật, rollback, IDOR cột qua SQL thật) |
| `make cover` (gói cốt lõi domain + application) | ✅ 90,3% (gói `application/project` 83,9%) |
| `gitleaks detect --no-banner` | ✅ no leaks found |
| `go vet ./...` (kể cả tag `integration`) | ✅ |
| `python3 scripts/ai-log/ai_log.py check` | ✅ OK; cảnh báo còn lại: mục "Phần tôi đã sửa" chưa điền |
| `docker build`, `make test-integration` (CI) | ✅ xem mục CI bên dưới |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Xác thực | Không token → 401 trên mọi endpoint dự án | ✅ | `TestProjectAPI_RequiresAuth` |
| Phân quyền | Chỉ Manager tạo dự án (Admin/Member/CEO → 403); chỉ Manager phụ trách sửa thông tin, thành viên, cột, luật; CEO/Member trong dự án → 403 trên 9 thao tác ghi; Admin → 404 | ✅ | `TestCreate_OnlyManager_CF03`, `TestUpdate_OnlyProjectManager_CF03`, `TestWriteOps_ForbiddenForNonManager_CF03`, `TestProjectAPI_ErrorMapping` |
| IDOR | Người ngoài dự án mở dự án bằng ID hợp lệ → 404 (không lộ tồn tại); Manager khác → 404; mã cột của dự án B gọi trên đường dẫn dự án A → `COLUMN_NOT_FOUND`/`INVALID_COLUMN_ORDER`, B không đổi | ✅ | `TestGet_Visibility_CF03`, `TestColumnOps_CrossProjectColumn_CF03`, `TestProjectRepo_ServiceFlow_CF03`; Playwright: `cf03-member-not-member-404.png` |
| SQL / lỗi 500 | Mọi truy vấn qua sqlc, có `project_id` trong điều kiện cột/luật; ràng buộc unique/FK ánh xạ sang lỗi domain; lỗi 500 trả câu chung | ✅ | `backend/db/queries/project.sql`, `board_column.sql`; go-reviewer |
| Quy tắc nghiệp vụ | 2–8 cột; luôn ≥ 1 cột done; xóa cột còn việc → 409 `COLUMN_HAS_TASKS`; Member không chuyển vào/ra nhóm done (CLAUDE.md §5.5); không bỏ được Manager phụ trách; Admin/tài khoản khóa không làm thành viên | ✅ | `TestValidateColumnSpecs_CF03`, `TestCheckColumnDeletable_CF03`, `TestValidateTransitions_CF03`, `TestPruneTransitions_CF03`, `TestMembers_CF03` |
| Validate | Tên rỗng/khoảng trắng/150–151 rune/tiếng Việt; trùng tên không phân biệt hoa thường → 409 + rollback; mô tả 2001 ký tự; ngày kết thúc trước ngày bắt đầu; ngày/UUID sai định dạng → 400; trạng thái/nhóm lạ → 422; FE dùng cùng luật (zod) | ✅ | `TestNormalizeProjectName_CF03`, `TestValidateProjectDates_CF03`, `TestCreate_Validation_CF03`, `TestProjectAPI_BadRequests`; plist "ngày kết thúc trước ngày bắt đầu bị chặn ở form" |
| HTML/script trong text | Hiển thị qua React (escape mặc định), không dùng `dangerouslySetInnerHTML` | ✅ | đọc mã |
| UX — Manager | Danh sách, tạo dự án → chuyển thẳng sang chi tiết, 4 tab, bảng 5 cột mặc định, hộp xác nhận xóa cột (Esc đóng), ma trận luật khóa 8/40 ô Member↔done | ✅ | Playwright MCP 1366×768: `cf03-manager-list.png`, `cf03-manager-create.png`, `cf03-manager-detail.png`, `cf03-manager-columns.png`, `cf03-manager-column-delete.png`, `cf03-manager-transitions.png` |
| UX — Member / CEO / Admin | Member: chỉ dự án mình tham gia, không có nút tạo/sửa/thêm cột; dự án không tham gia → "Không tìm thấy dự án". CEO: thấy tất cả, chỉ xem. Admin: "Bạn không có quyền xem trang này" | ✅ | `cf03-member-list.png`, `cf03-member-detail.png`, `cf03-member-not-member-404.png`, `cf03-ceo-list.png`, `cf03-ceo-detail.png`, `cf03-admin-denied.png` |
| UX — trạng thái | Skeleton, rỗng theo vai trò, không khớp bộ lọc, lỗi + Thử lại, 404, toast thành công, lỗi 409 dưới ô tên, cảnh báo xóa cột có việc | ✅ | `ProjectList.test.tsx`, `ProjectDetail.test.tsx` (23 test) |
| Gửi trùng | Nút gửi khóa khi đang gửi; thêm thành viên lần 2 idempotent; `PUT /transitions` và `/column-order` thay toàn bộ | ✅ | `TestMembers_CF03`, `TestReorderColumns_CF03`; pit |
| Agent | go-reviewer: 0 Chặn · ui-reviewer: 0 Chặn · qa-tester: 1 lệch quy tắc, đã sửa · ma trận truy vết đã cập nhật (7 dòng CF-03) | ✅ | `docs/testing/traceability-matrix.md` |

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | Server chấp nhận luật Member chuyển *ra khỏi* nhóm done (FE khóa cả hai chiều, `docs/decisions.md` mục C ghi "vào/ra"); test domain còn khẳng định hành vi sai (qa-tester) | Trung bình | 4cfd063 |
| 2 | Thiếu test chứng minh CEO/Member/Manager khác/Admin bị chặn trên từng thao tác ghi và IDOR mã cột chéo dự án (qa-tester) | Trung bình | 4cfd063 |
| 3 | `ProjectTx.InTx` trả lỗi transaction không bọc ngữ cảnh (go-reviewer) | Thấp | 4cfd063 |
| 4 | Bảng cột rỗng dùng nhầm chuỗi "Không tải được cấu hình cột"; cột số chưa canh phải; khoảng ngày dùng mũi tên (ui-reviewer) | Thấp | 4cfd063 |
| 5 | Định dạng ngày bằng template string ngoài `shared/lib/format.ts`; `tsc -b` bắt `noUncheckedIndexedAccess` ở `ColumnsPanel` (trong quá trình làm, xem AI log) | Thấp | fae6650 |

## Còn tồn đọng
- **Người dùng cần làm:** điền mục "Phần tôi đã sửa / hoàn thiện" trong `docs/ai-log/entries/2026-11-20-02-feature-projects.md` (đang để `[NGƯỜI DÙNG ĐIỀN]`).
- **Issue [#4](https://github.com/BaoCool-dev/WorkTrack/issues/4)** (mục nhỏ, không chặn):
  - `taskCount` của cột đang trả 0 cố định tới khi có bảng `task` (CF-06); luồng cảnh báo xóa cột có việc mới kiểm bằng test RTL + mã lỗi API, chưa kiểm với dữ liệu thật.
  - Chưa có test FE cho CEO ở màn chi tiết; chưa có ca thêm CEO làm thành viên.
  - Chưa có E2E Playwright tự động (đã đi tay bằng Playwright MCP).
  - ui-reviewer gợi ý: khóa nút xác nhận khi xóa cột đang có việc; câu trạng thái rỗng riêng cho CEO.
- **Giả định mục C:** 13 dòng trong `docs/decisions.md` (ngày 2026-11-20, nhánh `feature/projects`).

## CI (PR #5)
| Lần chạy | Commit | Backend (build → lint → test + integration testcontainers, `-race`) | Frontend (build → lint → test) | gitleaks | Docker build |
|---|---|---|---|---|---|
| [44681601735](https://github.com/BaoCool-dev/WorkTrack/actions/runs/44681601735) | f83fcff | ✅ 1m40s | ✅ 49s | ✅ 7s | ✅ 2m21s |
