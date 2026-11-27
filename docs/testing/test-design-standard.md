# Chuẩn thiết kế và thực thi kiểm thử — WorkTrack

> Rút ra từ bài giảng **Kiểm thử phần mềm (SOTE431079, Khoa Hoàng — Khoa CNTT, HCM-UTE)**: Ch.02 Yêu cầu và yêu cầu kiểm thử, Ch.03 Tổng quan test case, Ch.03 Kỹ thuật hộp đen, Ch.04 Lỗi phần mềm và báo cáo lỗi, Ch.06 Kiểm thử tự động; và tài liệu **Test Techniques** (Day 6+7).
> Mọi bảng test case, bug report và báo cáo kiểm thử trong repo phải theo chuẩn này. Skill `test-case-design` là bản rút gọn cho Claude.

## 1. Quy trình kiểm thử và sản phẩm tương ứng trong repo

| Giai đoạn (Ch.03) | Đầu ra trong WorkTrack |
|---|---|
| Lập kế hoạch | `docs/testing/test-plan.md` — phạm vi (CF Lõi), cách tiếp cận, môi trường, **tiêu chí kết thúc** (§8) |
| Giám sát, điều khiển | Báo cáo gate từng nhánh `docs/gates/*.md` |
| Phân tích | **Điều kiện kiểm thử** (test condition) cho từng chức năng, truy vết về yêu cầu (§3) |
| Thiết kế | Test case mức cao, suy ra bằng **kỹ thuật cụ thể** (§4), có độ ưu tiên |
| Hiện thực | Test tự động (Go, Vitest, Playwright), dữ liệu kiểm thử (`make seed`), môi trường |
| Thực thi | `scripts/test-cases.sh` — chạy thật, so sánh thực tế với mong đợi, ghi log, báo lỗi |
| Kết thúc | `docs/testing/test-report.md` — tổng kết, lỗi còn mở, bài học (§8) |

Truy vết đầy đủ: **yêu cầu (CF/UC) → điều kiện kiểm thử → test case → test tự động → kết quả → bug**.

## 2. Thành phần bắt buộc của một test case (Ch.03)

| Trường | Yêu cầu |
|---|---|
| **Mã TC** | `TC_<MODULE>_<NNN>`, ví dụ `TC_AUTH_001`. Module: AUTH (CF-01), ORG (CF-02), PROJ (CF-03), KANBAN (CF-04/05), TASK (CF-06), SUB (CF-07), LOG (CF-08), REVIEW (CF-09), REPORT (CF-10/11), AI (CF-12). Mã đã cấp không đổi, không đánh số lại |
| **Mô tả** | Một câu nêu mục đích: "Xác minh …" |
| **Điều kiện kiểm thử** | Mã TCond mà ca này kiểm (§3) |
| **Kỹ thuật** | EP, BVA, Bảng quyết định, Chuyển trạng thái, Use case, Ràng buộc, Quan hệ dữ liệu–chức năng, Đoán lỗi |
| **Tiền điều kiện** | Trạng thái hệ thống/dữ liệu/vai trò cần có trước khi chạy |
| **Dữ liệu kiểm thử** | Giá trị cụ thể (không ghi "dữ liệu hợp lệ") |
| **Các bước** | Đánh số, mỗi bước một hành động cụ thể |
| **Kết quả mong đợi** | Quan sát được: màn hình/thông điệp/mã HTTP/dữ liệu trong CSDL |
| **Tiêu chí Pass/Fail** | Điều kiện để kết luận Pass; liệt kê các dấu hiệu Fail |
| **Kết quả thực tế** | Điền sau khi chạy — đúng như quan sát |
| **Trạng thái** | Pass / Fail / Chưa chạy / Bị chặn |
| **Ưu tiên** | Cao (Lõi, phân quyền, bảo mật, dữ liệu) / Trung bình / Thấp |
| **Loại** | Chức năng / Phi chức năng; tầng Unit / Integration / API / UI |
| **Hình thức** | Tự động (ghi tên test) / Thủ công |

**Ca tốt phải:** rõ ràng, cụ thể, lặp lại được (cùng dữ liệu → cùng kết quả), độc lập (không phụ thuộc thứ tự chạy), truy vết được.

## 3. Phân tích: điều kiện kiểm thử

Với mỗi chức năng, trước khi viết ca, lập bảng điều kiện kiểm thử `TCond_<MODULE>_<NN>` từ: yêu cầu (CF/UC/YCCN), quy tắc nghiệp vụ (`worktrack-domain`, `docs/decisions.md`), thiết kế (`design-system.md`), ràng buộc trong code (validator, CHECK CSDL). Mỗi điều kiện ghi nguồn. Yêu cầu nào mơ hồ (không đo được, thiếu ngưỡng) thì ghi vào mục "Câu hỏi cần làm rõ" thay vì tự đoán (Ch.02: yêu cầu phải cụ thể, đo được).

## 4. Thiết kế: chọn kỹ thuật theo loại đối tượng

| Đối tượng kiểm thử | Kỹ thuật bắt buộc |
|---|---|
| Trường nhập có miền giá trị, độ dài, số lượng | **EP** (1 giá trị đại diện/lớp, gồm lớp không hợp lệ) **+ BVA** (giá trị biên dưới/trên và ngay ngoài biên, cách 2 giá trị: `min-1, min, max, max+1`; trường quan trọng dùng 3 giá trị) |
| Tập giá trị rời rạc (Story Point Fibonacci, vai trò, trạng thái) | **EP** theo từng giá trị hợp lệ + giá trị ngoài tập |
| Nhiều điều kiện kết hợp ra kết quả khác nhau (phân quyền vai trò × hành động, luật chuyển cột vai trò × nhóm nguồn × nhóm đích, đăng nhập email × mật khẩu × khóa) | **Bảng quyết định**: liệt kê điều kiện, hành động, quy tắc (2ⁿ cột), gộp cột cùng kết quả; mỗi cột còn lại ≥ 1 ca |
| Đối tượng có trạng thái (tài khoản: chờ kích hoạt → hoạt động → bị khóa tạm; số lần đăng nhập sai; công việc: todo → in_progress → chờ nghiệm thu → done; dự án) | **Chuyển trạng thái**: sơ đồ trạng thái (Mermaid), ca cho mọi chuyển hợp lệ **và** chuyển không hợp lệ |
| Luồng nghiệp vụ end-to-end | **Use case**: luồng chính, luồng thay thế, luồng ngoại lệ |
| Ràng buộc giữa các trường (ngày bắt đầu ≤ ngày kết thúc, email duy nhất, WIP ≤ giới hạn) | **Phân tích ràng buộc** |
| Vai trò → hành vi khác nhau | **Quan hệ dữ liệu–chức năng** (mỗi vai trò một ca) |
| Lỗi thường gặp | **Đoán lỗi / checklist**: IDOR (truy cập ID của dự án khác), token hết hạn/sửa chữ ký/`alg: none`, XSS trong tiêu đề/bình luận, SQL injection trong ô tìm kiếm, bấm hai lần, hai người cùng sửa, mất mạng giữa chừng, chuỗi chỉ khoảng trắng, tiếng Việt có dấu, emoji |

Các miền giá trị đã biết (kiểm lại với code trước khi dùng): mật khẩu ≥ 8 ký tự có chữ và số; khóa 15 phút sau 5 lần sai; token kích hoạt 72 giờ; tiêu đề công việc ≤ 200 ký tự, mô tả ≤ 5000; ≤ 10 nhãn/việc; hệ số 1–40 giờ/SP; Story Point ∈ {1,2,3,5,8,13,21}; WIP 1–20 (mặc định 5); 2–8 cột/dự án; ≤ 50 kỹ năng/người; `pageSize` ≤ 100; giới hạn tần suất đăng nhập 10 lượt.

## 5. Phi chức năng

Theo yêu cầu phi chức năng định lượng của bản cam kết: hiệu năng (p95 ≤ 500 ms cho Log time và chuyển thẻ), đồng bộ tức thời (p95 ≤ 2 giây — KPI 1), bảo mật (phân quyền mọi API, 0 secret), truy cập (WCAG AA, 1366×768). Mỗi mục là một ca có số đo thật, minh chứng trong `docs/testing/evidence/`.

## 6. Báo cáo lỗi (Ch.04)

Mỗi ca **Fail** → một bug trên **GitHub Issues** (công cụ quản lý lỗi của dự án), dùng mẫu `.github/ISSUE_TEMPLATE/bug_report.md`:
- **Tiêu đề**: một câu nêu lỗi và vị trí xảy ra.
- **Các bước tái hiện** đánh số, tối thiểu; **kết quả mong đợi** và **kết quả thực tế**.
- **Môi trường**: hệ điều hành, trình duyệt, commit, vai trò, dữ liệu.
- **Severity** (Critical/Major/Minor/Trivial) và **Priority** (P1–P3).
- Mã TC liên quan, ảnh/log đính kèm.
- Một lỗi một issue; tìm trùng trước khi tạo; mô tả sự thật quan sát được, không đoán nguyên nhân; ngôn ngữ trung tính.

Vòng đời: New → Assigned → Open → Fixed → Retest → Verified → Closed (có thể Reopened/Rejected). Trên GitHub: issue mở = New/Open; PR sửa ghi `Fixes #n`; chạy lại `scripts/test-cases.sh` để Retest; Pass → đóng issue (Verified/Closed); còn lỗi → mở lại.

## 7. Kiểm thử tự động

Theo kim tự tháp kiểm thử: nhiều unit test (domain, application), ít hơn ở API/integration, ít nhất ở UI/E2E. Ca tự động hóa được thì tự động hóa (mục tiêu ≥ 70% — M29); ca cần mắt người (giao diện, email, realtime trên 2 trình duyệt) để thủ công. Bộ hồi quy = toàn bộ test tự động, chạy lại mỗi lần đổi code.

## 8. Báo cáo tổng kết kiểm thử và tiêu chí kết thúc

`docs/testing/test-report.md` gồm: phạm vi đã kiểm, môi trường, số ca theo chức năng/kỹ thuật/hình thức, Pass/Fail/Chưa chạy, độ phủ code, danh sách lỗi theo severity và trạng thái, đối chiếu **tiêu chí kết thúc** (bản cam kết): 100% chức năng Lõi có ca Pass, 0 lỗi Critical/Blocker còn mở, độ phủ gói cốt lõi ≥ 70%, 100% test tự động Pass ở lần chạy cuối; kết luận đạt/chưa đạt; bài học rút ra.
