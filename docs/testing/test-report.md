# Báo cáo kiểm thử — WorkTrack (CF-01 → CF-06)

Theo `docs/testing/test-design-standard.md` §8. Bảng ca chi tiết: `docs/testing/test-cases.md` (CSV: `test-cases.csv`).

## 1. Thông tin chung

| Mục | Giá trị |
|---|---|
| Ngày | 2026-11-27 |
| Nhánh | `test/test-cases` (tách từ `feature/kanban`) |
| Commit gốc | `13333ea` |
| Người thực hiện | Claude Code (chế độ tự chạy), người dùng soát lại |
| Lần chạy tự động | trước: `docs/testing/runs/20261126-2233-before.md`; sau: `docs/testing/runs/20261126-2233-after.md` |

## 2. Phạm vi

- **Trong phạm vi:** CF-01 (đăng nhập, phiên, tài khoản), CF-02 (tổ chức), CF-03 (dự án, cột, luật chuyển), CF-04/CF-05 (Kanban, WIP, realtime), CF-06 (công việc cha).
- **Ngoài phạm vi:** CF-07 → CF-12 (công việc con, Log time, nghiệm thu, báo cáo, AI) — chưa có code trên nhánh gốc. Kiểm thử hiệu năng tải lớn, trình duyệt ngoài Chromium.
- **Ràng buộc:** không sửa code sản phẩm, không sửa/xóa test có sẵn, không sửa lỗi trên nhánh này. Chỉ thêm 4 file test thiết kế mới (`tc_design_test.go` ×2, `tc_constraints_integration_test.go`, `tc-design.test.tsx`).

## 3. Môi trường

| Thành phần | Chi tiết |
|---|---|
| Hệ điều hành | macOS (Darwin 24.5) |
| Backend | Go, `go test ./...` (unit + API qua httptest) và integration với PostgreSQL trong Docker |
| Frontend | Vitest + React Testing Library + MSW |
| Ca thủ công | App thật chạy bằng `make dev` (postgres + api + web), dữ liệu seed B11; điều khiển trình duyệt Chromium bằng Playwright MCP, độ phân giải 1366×768 |
| Công cụ dựng bảng | `scripts/test-cases.sh` (summary.json) |

## 4. Số lượng ca

### 4.1 Theo chức năng

| Module | Mã CF | Số ca | Tự động | Thủ công | Pass | Fail | Chưa chạy |
|---|---|---|---|---|---|---|---|
| AUTH — Đăng nhập, phiên và tài khoản | CF-01 | 49 | 35 | 14 | 49 | 0 | 0 |
| ORG — Tổ chức: nhân sự, phòng ban, chức danh, kỹ năng | CF-02 | 38 | 33 | 5 | 38 | 0 | 0 |
| PROJ — Dự án, cột Kanban và luật chuyển | CF-03 | 43 | 39 | 4 | 43 | 0 | 0 |
| KANBAN — Bảng Kanban, luật chuyển cột và giới hạn WIP | CF-04, CF-05 | 51 | 47 | 4 | 51 | 0 | 0 |
| TASK — Công việc cha, Story Point, nhãn, người được giao | CF-06 | 45 | 41 | 4 | 45 | 0 | 0 |
| **Tổng** | | **226** | **195** | **31** | **226** | **0** | **0** |

### 4.2 Theo kỹ thuật thiết kế

Một ca có thể dùng nhiều kỹ thuật nên tổng cột lớn hơn số ca.

| Kỹ thuật | AUTH | ORG | PROJ | KANBAN | TASK | Tổng |
|---|---|---|---|---|---|---|
| EP | 19 | 13 | 9 | 2 | 9 | 52 |
| BVA | 12 | 14 | 16 | 17 | 19 | 78 |
| Bảng quyết định | 6 | 4 | 15 | 24 | 10 | 59 |
| Chuyển trạng thái | 10 | 1 | 1 | 6 | 1 | 19 |
| Use case | 5 | 2 | 2 | 3 | 5 | 17 |
| Đoán lỗi | 9 | 7 | 5 | 5 | 6 | 32 |
| Khác (ràng buộc CSDL, kiểm tra tĩnh, cấu hình, KPI) | 1 | 1 | 0 | 3 | 3 | 8 |

### 4.3 Theo hình thức và loại ca

| Hình thức | Số ca |
|---|---|
| Tự động (unit) | 147 |
| Thủ công (Playwright MCP) | 31 |
| Tự động (unit + API) | 8 |
| Tự động (API) | 8 |
| Tự động (UI) | 8 |
| Tự động (integration) | 7 |
| Tự động (unit + API + UI) | 6 |
| Tự động (API + UI) | 5 |
| Tự động (unit + UI) | 2 |
| Tự động (unit + integration + UI) | 2 |
| Tự động (unit + API + integration) | 1 |
| Kiểm tra tĩnh + tự động | 1 |

| Loại ca | Số ca |
|---|---|
| Dương | 68 |
| Âm | 63 |
| Biên | 75 |
| Bảo mật | 20 |

Tỷ lệ tự động hóa: **195/226 = 86.3%**.

## 5. Kết quả thực thi

| Chỉ số | Trước (`20261126-2233-before`) | Sau (`20261126-2233-after`) |
|---|---|---|
| Tổng test tự động | 974 | 1150 |
| Pass / Fail / Skip | 974 / 0 / 0 | 1150 / 0 / 0 |
| Backend Unit | 391 | 557 |
| Backend API | 363 | 363 |
| Backend Integration | 0 | 7 |
| Frontend Unit/UI | 220 | 223 |
| Độ phủ gói cốt lõi (domain + application) | 90.1% | 90.3% |

Bảng test case: **226 Pass, 0 Fail, 0 Chưa chạy** trên 226 ca (31 ca thủ công đều Pass).

## 6. Độ phủ

- **Yêu cầu:** mỗi điều kiện `TCond_*` (36 điều kiện) có ≥ 1 ca; ma trận `docs/testing/traceability-matrix.md` đã thêm cột **Mã TC** nối tiêu chí → ca.
- **Vai trò:** mỗi module có ca cho đủ Admin, Manager, Member, CEO (bảng "Phủ 4 vai trò" trong test-cases.md).
- **Ca âm và ca biên:** mỗi module có ca EP lớp không hợp lệ và BVA min−1/min/max/max+1.
- **Mã nguồn:** 90.3% gói cốt lõi (ngưỡng tối thiểu theo chuẩn: 70%).

## 7. Danh sách lỗi

Không phát hiện lỗi mới trong lần kiểm thử này.

| Mã | Tiêu đề | Severity | Priority | Trạng thái | Ca liên quan |
|---|---|---|---|---|---|
| #9 (có từ trước) | Kết nối WebSocket không bị đóng khi access token hết hạn | Major | P2 | Open (không sửa trên nhánh này) | TC_KANBAN_009 |

| Severity | Số lỗi mới |
|---|---|
| Critical | 0 |
| Major | 0 |
| Minor | 0 |
| Trivial | 0 |

## 8. Đối chiếu tiêu chí kết thúc

| Tiêu chí | Yêu cầu | Thực tế | Đạt |
|---|---|---|---|
| Chức năng Lõi trong phạm vi có ca Pass | 100% | CF-01 → CF-06: 6/6 (CF-07 → CF-12 chưa có code, ngoài phạm vi) | Đạt (trong phạm vi) |
| Lỗi Critical/Blocker còn mở | 0 | 0 (lỗi #9 có từ trước là Major) | Đạt |
| Độ phủ gói cốt lõi | ≥ 70% | 90.3% | Đạt |
| Test tự động Pass ở lần chạy cuối | 100% | 1150/1150 | Đạt |
| Tỷ lệ tự động hóa (M29) | ≥ 70% | 86.3% | Đạt |
| Mọi điều kiện kiểm thử có ca | 100% | 36/36 | Đạt |

## 9. Minh chứng giao diện

31 ca thủ công chạy trên app thật, ảnh lưu tại `docs/testing/evidence/ui/` (tên ảnh = mã TC, hậu tố `-1`, `-2` cho trước/sau). Ảnh không chứa mật khẩu hay token (token kích hoạt đã che trước khi chụp).

| Mã TC | Mô tả | Ảnh |
|---|---|---|
| TC_AUTH_101 | Admin đăng nhập, thấy menu quản trị | docs/testing/evidence/ui/TC_AUTH_101.png |
| TC_AUTH_102 | Manager đăng nhập, thấy menu dự án | docs/testing/evidence/ui/TC_AUTH_102.png |
| TC_AUTH_103 | Member đăng nhập, thấy Công việc của tôi | docs/testing/evidence/ui/TC_AUTH_103.png |
| TC_AUTH_104 | CEO đăng nhập, thấy Dashboard và Quản trị | docs/testing/evidence/ui/TC_AUTH_104.png |
| TC_AUTH_105 | Sai mật khẩu và email không tồn tại cùng một thông báo | docs/testing/evidence/ui/TC_AUTH_105-1.png; docs/testing/evidence/ui/TC_AUTH_105-2.png |
| TC_AUTH_106 | Bỏ trống/ sai định dạng bị chặn ở client | docs/testing/evidence/ui/TC_AUTH_106-1.png; docs/testing/evidence/ui/TC_AUTH_106-2.png |
| TC_AUTH_107 | Đăng xuất rồi Back hoặc gõ URL đều về /login | docs/testing/evidence/ui/TC_AUTH_107-1.png; docs/testing/evidence/ui/TC_AUTH_107-2.png |
| TC_AUTH_108 | F5 vẫn giữ phiên nhờ refresh cookie | docs/testing/evidence/ui/TC_AUTH_108.png |
| TC_AUTH_109 | Admin tạo tài khoản, email kích hoạt tới Mailpit | docs/testing/evidence/ui/TC_AUTH_109-1.png; docs/testing/evidence/ui/TC_AUTH_109-2.png |
| TC_AUTH_110 | Đặt mật khẩu chỉ có chữ bị báo lỗi | docs/testing/evidence/ui/TC_AUTH_110.png |
| TC_AUTH_111 | Kích hoạt thành công rồi đăng nhập được | docs/testing/evidence/ui/TC_AUTH_111-1.png; docs/testing/evidence/ui/TC_AUTH_111-2.png |
| TC_AUTH_112 | Dùng lại link kích hoạt đã dùng bị từ chối | docs/testing/evidence/ui/TC_AUTH_112.png |
| TC_AUTH_113 | Sai 5 lần bị khóa; lần 6 nhập đúng vẫn bị khóa | docs/testing/evidence/ui/TC_AUTH_113-1.png; docs/testing/evidence/ui/TC_AUTH_113-2.png |
| TC_AUTH_114 | Tài khoản chưa kích hoạt bị từ chối | docs/testing/evidence/ui/TC_AUTH_114.png |
| TC_ORG_101 | Admin xem danh sách nhân sự và các tab quản trị | docs/testing/evidence/ui/TC_ORG_101.png |
| TC_ORG_102 | Thêm nhân sự trùng email | docs/testing/evidence/ui/TC_ORG_102.png |
| TC_ORG_103 | Admin không thấy vai trò CEO trong form | docs/testing/evidence/ui/TC_ORG_103.png |
| TC_ORG_104 | Manager và Member gõ thẳng /admin/users bị chặn | docs/testing/evidence/ui/TC_ORG_104-1.png; docs/testing/evidence/ui/TC_ORG_104-2.png |
| TC_ORG_105 | Cây phòng ban hiển thị đúng cấp | docs/testing/evidence/ui/TC_ORG_105.png |
| TC_PROJ_101 | Manager tạo dự án; ngày sai bị chặn | docs/testing/evidence/ui/TC_PROJ_101-1.png; docs/testing/evidence/ui/TC_PROJ_101-2.png |
| TC_PROJ_102 | Admin gõ /projects bị chặn | docs/testing/evidence/ui/TC_PROJ_102.png |
| TC_PROJ_103 | CEO xem cấu hình dự án chỉ đọc | docs/testing/evidence/ui/TC_PROJ_103.png |
| TC_PROJ_104 | Member thấy luật chuyển nhưng không sửa được | docs/testing/evidence/ui/TC_PROJ_104.png |
| TC_KANBAN_101 | Manager và Member mở bảng Kanban | docs/testing/evidence/ui/TC_KANBAN_101.png |
| TC_KANBAN_102 | Member kéo thẻ sang Done bị chặn ở client | docs/testing/evidence/ui/TC_KANBAN_102-1.png; docs/testing/evidence/ui/TC_KANBAN_102-2.png |
| TC_KANBAN_103 | Cập nhật tức thời giữa hai tab | docs/testing/evidence/ui/TC_KANBAN_103-1.png; docs/testing/evidence/ui/TC_KANBAN_103-2.png |
| TC_KANBAN_104 | Đặt WIP 2, Member đủ 2/2 bị chặn nhận thêm | docs/testing/evidence/ui/TC_KANBAN_104-1.png; docs/testing/evidence/ui/TC_KANBAN_104-2.png |
| TC_TASK_101 | Manager tạo công việc: bỏ trống tên bị chặn; 5 SP quy đổi 20 giờ | docs/testing/evidence/ui/TC_TASK_101-1.png; docs/testing/evidence/ui/TC_TASK_101-2.png |
| TC_TASK_102 | Member xem Công việc của tôi | docs/testing/evidence/ui/TC_TASK_102.png |
| TC_TASK_103 | Admin quản lý Nhãn và hệ số giờ/SP | docs/testing/evidence/ui/TC_TASK_103-1.png; docs/testing/evidence/ui/TC_TASK_103-2.png |
| TC_TASK_104 | Admin và CEO gõ /my-tasks bị chặn | docs/testing/evidence/ui/TC_TASK_104-1.png; docs/testing/evidence/ui/TC_TASK_104-2.png |

## 10. Kết luận

Các chức năng CF-01 → CF-06 trên commit `13333ea` **đạt** tiêu chí kết thúc: 1150/1150 test tự động pass, 226/226 ca thiết kế pass, không lỗi mới. Còn mở lỗi #9 (WS không đóng khi token hết hạn) và các câu hỏi Q1–Q5 trong test-cases.md cần người dùng làm rõ.

## 11. Bài học

- Viết điều kiện kiểm thử trước rồi mới chọn kỹ thuật giúp lộ ra các biên bị bỏ sót (72 byte bcrypt với ký tự đa byte, mốc locked_until chính xác tới giây, nhãn trùng trước khi đếm).
- Ràng buộc CHECK/UNIQUE ở CSDL cần test integration riêng; test domain không chứng minh được CSDL chặn dữ liệu sai.
- Ảnh minh chứng dễ lộ token trong URL; phải che hoặc xóa token khỏi URL trước khi chụp.
- `docs/requirements/` còn thiếu UC/YCCN chính thức, nên nguồn điều kiện tạm truy về CF, CLAUDE.md §5 và decisions mục B.
