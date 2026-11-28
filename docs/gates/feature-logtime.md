# Gate: feature/logtime — CF-08 (Log time trên công việc cha và con)
**Ngày:** 2026-11-28 · **Commit cuối (code):** 33ecab2 · **Kết luận:** ĐẠT (kèm ghi chú — xem "Còn tồn đọng")

Phạm vi:
- **Backend:**
  - Bảng `time_log` (migration `20261127114535_add_time_log.sql`): CHECK phút 1–1440, ghi chú 1–500 ký tự. `task_view` thêm `logged_minutes`, `actual_minutes` (= giờ cha + giờ con), `subtask_count`, `subtask_done_count`.
  - Domain `timelog.go`: chuẩn hóa phút/ghi chú, ngày làm việc theo giờ Việt Nam, cửa sổ 24 giờ sau Done, quyền ghi/xóa.
  - Use case `application/timelog`: ghi/xem/xóa log. Quyền theo người nhận việc gốc hoặc Manager phụ trách; kiểm lại trong giao dịch đã khóa. Luật 24 giờ áp cho cả việc cha khi ghi vào con. Phát `task.updated` cho thẻ gốc sau commit.
  - 3 endpoint: `GET/POST /tasks/{taskID}/time-logs`, `DELETE /time-logs/{logID}`. Lỗi validate → 422, Done quá 24 giờ → 409 `TIME_LOG_CLOSED`.
- **Frontend:**
  - Thẻ Kanban: thanh so giờ theo `actualMinutes` + ký hiệu mức B5, `x/y con`.
  - Hộp chi tiết: mục giờ thực tế/ước tính, lịch sử log có phân trang, xóa log của mình (xác nhận ngay trong dòng).
  - Hộp "Log giờ": chọn việc cha hoặc một việc con, ngày làm, số giờ (nhận dấu phẩy), ghi chú; 409 → khối khóa thay form (design-system §7).
- **Không làm ở nhánh này:** khối "Giờ của tôi tuần này" ở sidebar (dời sang CF-10), sửa log (xóa rồi ghi lại).

Gate chạy 2 vòng:
- **Vòng 1:** chạy toàn bộ lệnh tự động, smoke test API trên app dev (`make dev-local`), go-reviewer, ui-reviewer. Không có mục Chặn. Đã sửa ở aa7e978, 477d8cc; ghi giả định mục C ở 66df1a5; mục nhỏ còn lại đưa vào #16, #17.
- **Vòng 2:** qa-tester đối chiếu tiêu chí CF-08, bổ sung test biên/quyền và ma trận truy vết (33ecab2). Chạy lại toàn bộ lệnh tự động ở 33ecab2 — tất cả xanh (sau khi sửa định dạng Prettier một file test).

## Tự động
Môi trường: máy không có Docker (B12). PostgreSQL chạy bằng Homebrew; integration test chạy trên CSDL `worktrack_gate_logtime` tạo mới. `docker build` và `make test-integration` (testcontainers) chỉ chạy trong CI.

| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint + eslint + `tsc -b` + prettier) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: tất cả gói ok · FE: 314/314 test pass (28 file) |
| `go test -count=1 -tags integration ./...` (PostgreSQL local) | ✅ ok, gồm `timelog_integration_test.go` (giờ con cộng lên cha, lịch sử cha gồm log con, CHECK phút, luật 24 giờ cho cha và con) và `timelog_boundary_integration_test.go` |
| `make cover` (gói cốt lõi domain + application) | ✅ 89,7% (`application/timelog` 86,1%) |
| `gitleaks detect --no-banner` | ✅ 120 commit, no leaks found |
| `go vet ./...` (có và không tag `integration`), golangci-lint `--build-tags=integration`, `tsc --noEmit` | ✅ 0 issues |
| `python3 scripts/ai-log/ai_log.py check` | ⚠️ Không có lỗi nào trên các commit của nhánh này. Còn 2 lỗi cũ từ `release/v1`, không thuộc nhánh: entry `2026-11-27-01-test-cases.md` và commit `8712025` thiếu trailer `AI-Log`. Có cảnh báo mục "Phần tôi đã sửa" chưa điền |
| `docker build`, `make test-integration` (CI) | ✅ xem mục CI bên dưới |

## CI (PR #18)
| Lần chạy | Commit | Backend (build → lint → test, gồm integration) | Frontend | gitleaks | Docker build |
|---|---|---|---|---|---|
| [46089016104](https://github.com/BaoCool-dev/WorkTrack/actions/runs/46089016104) | 8e7e61f | ✅ 1m56s | ✅ 56s | ✅ 6s | ✅ 1m57s |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Xác thực | Không token → 401 trên cả 3 endpoint | ✅ | `timelog_api_test.go` |
| Phân quyền | <ul><li>Ghi log: người nhận việc gốc (việc con → người nhận việc cha) hoặc Manager phụ trách dự án.</li><li>CEO, Admin, Member khác → 403; xem lịch sử vẫn được trong phạm vi quyền đọc.</li><li>Chỉ người ghi được xóa log của mình.</li></ul> | ✅ | `domain/timelog_test.go`, `application/timelog/service_test.go`, `timelog_api_test.go` |
| IDOR | Người ngoài dự án dùng ID việc/log hợp lệ → 404, không lộ dữ liệu | ✅ | `service_test.go`, `timelog_api_test.go` |
| SQL / lỗi 500 | Mọi truy vấn qua sqlc (`time_log.sql`), không nối chuỗi; lỗi 500 trả câu chung | ✅ | go-reviewer |
| Quy tắc nghiệp vụ | <ul><li>Phút > 0 (domain + CHECK CSDL).</li><li>Không log vào việc Done quá 24 giờ; luật xét cả việc cha khi ghi vào con và cả chính việc con.</li><li>Mốc Done chỉ tính lần vào nhóm done từ ngoài nhóm.</li><li>Giờ thực tế = giờ cha + giờ con.</li><li>Mức B5 của thanh so giờ khớp `getHoursLevel`.</li></ul> | ✅ | `domain/timelog_test.go`, `service_test.go`, `timelog_integration_test.go`, `timelog_boundary_integration_test.go`, `TaskCard.test.tsx` |
| Validate | <ul><li>Phút 0, âm, 1441; ghi chú rỗng/khoảng trắng/501 ký tự; tiếng Việt và emoji.</li><li>Ngày tương lai, trước ngày tạo việc, sai định dạng.</li><li>UUID sai, body sai JSON → 400; lỗi validate → 422.</li><li>FE: số giờ có dấu phẩy, làm tròn phút, giới hạn 1440 phút.</li></ul> | ✅ | `timelog_api_test.go`, `domain/timelog_test.go`, `timeLogForm.test.ts`, `TimeLog.test.tsx` |
| HTML/script trong text | Ghi chú lưu và trả nguyên văn; FE hiển thị dạng chữ, không dùng `dangerouslySetInnerHTML` | ✅ | `TimeLog.test.tsx` |
| UX | <ul><li>Luồng ghi/xem/xóa log cho Member, Manager; CEO chỉ xem.</li><li>Đủ trạng thái đang tải, rỗng (câu chữ theo vai trò), lỗi + Thử lại, toast thành công; khối khóa khi Done quá 24 giờ.</li><li>Gửi trùng: nút bị khóa khi đang gửi.</li></ul> | ✅ (mục nhỏ ở #16) | `TimeLog.test.tsx`, `TaskCard.test.tsx`; ui-reviewer 0 mục Chặn; smoke test API vòng 1 |

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | Chuyển việc giữa các cột trong nhóm done làm "mở lại" mốc Done, kéo dài cửa sổ 24 giờ (go-reviewer) | Trung bình | aa7e978 |
| 2 | Truy vấn lịch sử/đếm log của việc cha không dùng được index (go-reviewer) | Thấp | aa7e978 |
| 3 | Lỗi validate phút/ghi chú/ngày trả 400, không thống nhất 422 với các lỗi validate khác (go-reviewer) | Thấp | aa7e978 |
| 4 | Hộp Log giờ ở trạng thái khóa vẫn hiện nút "Hủy" thay vì "Đóng" (ui-reviewer) | Thấp | 477d8cc |
| 5 | Lịch sử rỗng mời người chỉ xem (CEO) log giờ (ui-reviewer, design-system §7) | Thấp | 477d8cc |

## Còn tồn đọng
- **Người dùng cần làm:**
  - Điền mục "Phần tôi đã sửa / hoàn thiện" trong `docs/ai-log/entries/2026-11-27-04-feature-logtime.md`.
  - Lỗi `ai_log.py check` từ `release/v1` (entry `2026-11-27-01-test-cases.md`, commit `8712025`) cần xử lý ở nhánh đó.
- **Issue [#16](https://github.com/BaoCool-dev/WorkTrack/issues/16)** (giao diện, không chặn): focus/Esc ở xác nhận xóa log; lý do cạnh ô "Ghi vào" bị vô hiệu; nút "Báo lỗi" ở ErrorState; hiển thị log rất ngắn (`0,1h`); skeleton đúng hình hàng; hằng 500 dùng chung; `formatHours` thay `formatNumber`.
- **Issue [#17](https://github.com/BaoCool-dev/WorkTrack/issues/17)** (backend, không chặn): đổi `status_group` cột làm lệch mốc Done; cân nhắc cột `done_at`; rà index thừa trên `time_log`; bọc lỗi thống nhất; dùng chung `Clock`.
- **Khoảng trống kiểm thử (qa-tester):**
  - Chưa có acceptance criteria chính thức cho CF-08 trong `docs/requirements/`; ma trận truy vết dùng 12 tiêu chí rút từ phạm vi nhánh.
  - Chưa có E2E Playwright tự động.
  - Lệch nhỏ FE/BE: `todayValue` ở FE dùng múi giờ máy, server dùng giờ Việt Nam; ghi chú chỉ gồm ký tự BOM (`U+FEFF`) bị JS cắt nhưng Go không cắt (server vẫn là nơi chặn cuối).
  - "Giờ của tôi tuần này" thuộc CF-10.
- **Giả định mục C:** 8 dòng trong `docs/decisions.md` (ngày 2026-11-28, nhánh `feature/logtime`):
  - ô số giờ nhận dấu phẩy, làm tròn phút, API nhận phút nguyên 1–1440;
  - người được log và quyền xóa log;
  - ngày làm việc theo giờ Việt Nam, không tương lai, không trước ngày tạo việc;
  - 409 `TIME_LOG_CLOSED` + khối khóa, áp cả khi ghi vào con của việc cha đã Done;
  - khối "Giờ của tôi tuần này" dời sang CF-10;
  - luật 24 giờ xét cả chính việc con;
  - mốc "Done lúc" lấy từ lần vào nhóm done gần nhất (dự phòng `updated_at`);
  - Manager không xóa log của người khác, chưa có sửa log.
