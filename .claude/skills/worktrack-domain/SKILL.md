---
name: worktrack-domain
description: Quy tắc nghiệp vụ của WorkTrack — vai trò và quyền, công việc cha/con, Story Point, WIP, luồng Kanban và nghiệm thu, Log time, công thức báo cáo hiệu suất. Dùng khi viết logic, test hoặc UI liên quan tới các quy tắc này, hoặc khi không chắc một hành vi có đúng nghiệp vụ không.
---

# Quy tắc nghiệp vụ WorkTrack

Nguồn: các quyết định đã chốt đến 17/11/2026. Mã R.., D.., UC.. tham chiếu `docs/requirements/`.
Điều gì không có ở đây và không có trong `docs/requirements/` → **hỏi**, không tự đặt quy tắc.

## Vai trò
| Vai trò | Được | Không được |
|---|---|---|
| Admin | Tài khoản (kích hoạt qua email), phòng ban dạng cây, chức danh, kỹ năng, nhãn, mẫu dự án, cấu hình AI, nhập/xuất nhân sự | Tạo dự án |
| Manager | Tạo dự án, thêm thành viên, cấu hình Kanban, tạo công việc cha, giao việc, nghiệm thu, dashboard team, dùng AI | — |
| Member | Tự nhận việc từ Backlog, tạo công việc con, kéo thả việc của mình, Log time, bình luận | Giao việc cho người khác; chuyển sang nhóm hoàn thành |
| CEO | Quyền cao nhất | Dashboard tổng quan toàn công ty **chỉ xem** |

⚠ Ma trận hợp nhất (CEO kế thừa Admin tới đâu) **chưa chốt** → hỏi khi gặp.

## Công việc cha / con
- **Cha:** Manager tạo; có Story Point, hạn chót, nhãn, người nhận; đi qua luật chuyển trạng thái; được nghiệm thu.
- **Con:** Member tự tạo, **1 cấp**, **không có Story Point**, nằm trong một cột, Member tự chuyển, không qua `workflow_transition`, không nghiệm thu.
- Giờ thực tế ghi ở cả cha và con; báo cáo **cộng giờ con lên cha**.
- Cha được chuyển sang "Chờ nghiệm thu" khi còn con chưa xong → **cảnh báo, vẫn cho phép**; Manager có thể từ chối.
- Manager xem được con (giờ, trạng thái) qua quyền xem dự án.

## Kanban
- 2–8 cột cấu hình theo dự án. Mặc định: Backlog → To Do → In Progress → Chờ nghiệm thu → Done.
- Mỗi cột thuộc một nhóm trạng thái: `todo` | `in_progress` | `done`. "Chờ nghiệm thu" thuộc `in_progress`.
- Luật chuyển trạng thái theo vai trò (bảng `workflow_transition`). **Chỉ Manager** chuyển sang nhóm `done`.
- Nghiệm thu: Manager **Duyệt** (→ done) hoặc **Từ chối** (bắt buộc lý do → quay lại in_progress, tính vào tỷ lệ làm lại).
- Thay đổi trạng thái đẩy realtime tới người đang mở cùng dự án (skill `realtime-websocket`).

## WIP
- Chỉ đếm **công việc cha** ở nhóm `in_progress` của một người. Giới hạn mặc định 5 (BRD). Vượt → chặn nhận việc, UI hiện "3/5".
- WIP là chức năng **Mở rộng** trong bản cam kết — chỉ làm khi Lõi xong.

## Log time
- Đơn vị lưu: **phút**, > 0. Có mô tả ngắn.
- Không log vào việc đã Done quá **24 giờ**.
- Log được trên cả cha và con.

## Story Point
- Chỉ ở cha. Quy đổi giờ ước tính = SP × hệ số. ⚠ **Hệ số chưa chốt** (BRD ghi chú 1 SP : 4 giờ) → để là cấu hình, hỏi giá trị trước khi hiển thị số cụ thể.

## Công thức báo cáo (chỉ tính công việc gốc: `parent_task_id IS NULL`)
- **Tuân thủ thời hạn** = số việc Done đúng hoặc trước hạn ÷ tổng việc Done × 100%.
- **Độ lệch giờ** = Actual − Estimated, trong đó Actual = giờ cha + tổng giờ con.
- **Màu hiệu suất giờ:** Xanh: Actual ≤ Estimated · Vàng: vượt < 20% · Đỏ: vượt ≥ 20%.
- **Năng suất theo chu kỳ** = tổng Story Point của việc Done trong chu kỳ (Mở rộng).
- **Tỷ lệ làm lại** = số việc bị từ chối nghiệm thu ít nhất 1 lần ÷ tổng việc đã nghiệm thu (Mở rộng).
- KPI 3: Dashboard phải **khớp 100%** với số tính tay trên bộ dữ liệu mẫu ≥ 30 công việc → viết test so sánh với bảng tính tay cố định.

## AI (Gemini)
- Phân rã yêu cầu → danh sách **công việc cha** có Story Point (không tạo con). Là **bản xem trước**, người dùng chọn/sửa rồi mới tạo.
- Gợi ý nhân sự → Top 3 kèm lý do (kỹ năng, tải việc hiện tại, thời gian trống).
- Cảnh báo trễ hạn → quét hằng đêm (Mở rộng).
- Mỗi lần gọi ghi `ai_usage_log` (thời gian phản hồi). KPI 4: ≥ 95% lần phân rã < 10 giây.

## Ngoài phạm vi — từ chối nếu được yêu cầu
Chấm công check-in/vân tay/FaceID, lương/thưởng/BHXH/thuế, nghỉ phép, tuyển dụng, nhắn tin trực tiếp, app di động, đa doanh nghiệp, AI tóm tắt tiến độ, AI ngoài Gemini.
