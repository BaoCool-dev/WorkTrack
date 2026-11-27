# docs/design — bản thiết kế đã chốt

Nguồn: Claude Design, "WorkTrack – Thiết kế giao diện" (cập nhật 18/11/2026).
Quy tắc rút ra từ bản thiết kế nằm ở `design-system.md` (cùng thư mục). Thư mục `artboards/` chứa **bản xuất** để Claude Code mở khi làm từng màn hình.

Đặt vào đây, đúng tên để khớp bảng `design-system.md` §9:

```
artboards/01-design-foundation.html          01 Nền tảng thiết kế          (Claude Design: Artboard-m0tz)
artboards/02-kanban-manager.html             02 Bảng Kanban (Manager)      (Kanban)
artboards/03-task-detail-logtime.html        03 Chi tiết công việc + Log giờ (TaskDetail)
artboards/04-member-board.html               04 Bảng của Member, WIP 3/5   (MemberBoard)
artboards/05-ai-decompose.html               05 AI phân rã yêu cầu         (AiDecompose)
artboards/06-ai-assign-risk.html             06 AI gợi ý nhân sự, cảnh báo trễ hạn (AiAssign)
artboards/07-dashboard-project.html          07 Dashboard dự án (Manager)  (DashboardTeam)
artboards/08-performance-reports.html        08 Báo cáo hiệu suất          (Reports)
artboards/09-dashboard-company-ceo.html      09 Dashboard toàn công ty, CEO chỉ xem (DashboardCEO)
artboards/10-admin-users-departments.html    10 Nhân sự & phòng ban (Admin) (AdminUsers)
artboards/11-import-employees.html           11 Import nhân sự từ Excel    (ImportExcel)
artboards/12-project-settings.html           12 Cấu hình dự án & Kanban    (ProjectSettings)
artboards/13-screen-states.html              13 Trạng thái màn hình & cảnh báo (States)
screenshots/NN-<cung-ten>.png                ảnh chụp từng artboard — xem nhanh, dùng cho Chương 4
```
Tên trong ngoặc là tên file gốc trong Claude Design — khi xuất, đổi sang tên bên trái.

Khi bản thiết kế đổi: xuất lại, ghi ngày ở đầu file này, rồi cập nhật `docs/design/design-system.md` nếu luật thay đổi.
