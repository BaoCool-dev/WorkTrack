---
name: test-case-design
description: Chuẩn thiết kế test case, báo cáo lỗi và báo cáo kiểm thử của WorkTrack theo bài giảng Kiểm thử phần mềm HCM-UTE — điều kiện kiểm thử, EP, BVA, bảng quyết định, chuyển trạng thái, use case, đoán lỗi. Dùng khi viết test, lập bảng test case, chạy scripts/test-cases.sh, viết bug report, hoặc soát độ phủ kiểm thử của một chức năng.
---

# Thiết kế test case WorkTrack

Đọc đầy đủ `docs/testing/test-design-standard.md` trước. Tóm tắt bắt buộc:

1. **Phân tích trước:** lập bảng điều kiện kiểm thử `TCond_<MODULE>_<NN>` cho chức năng, mỗi điều kiện có nguồn (CF/UC, quy tắc, code). Yêu cầu mơ hồ → "Câu hỏi cần làm rõ", không đoán.
2. **Chọn kỹ thuật theo đối tượng** (bảng §4): trường có miền giá trị → EP + BVA (`min-1, min, max, max+1`); nhiều điều kiện → bảng quyết định (vẽ bảng, gộp cột); có trạng thái → sơ đồ chuyển trạng thái Mermaid + ca chuyển hợp lệ và không hợp lệ; luồng → use case (chính/thay thế/ngoại lệ); vai trò → một ca mỗi vai trò; checklist đoán lỗi bảo mật.
3. **Mỗi test case đủ trường** (§2): Mã `TC_<MODULE>_<NNN>` · Mô tả · Điều kiện kiểm thử · Kỹ thuật · Tiền điều kiện · Dữ liệu cụ thể · Các bước đánh số · Kết quả mong đợi · Tiêu chí Pass/Fail · Kết quả thực tế · Trạng thái · Ưu tiên · Loại · Hình thức (+ tên test tự động).
4. **Kết quả thực tế chỉ lấy từ lần chạy thật.** Không suy đoán Pass.
5. **Ca Fail → bug** trên GitHub Issues theo `.github/ISSUE_TEMPLATE/bug_report.md` (tìm trùng trước, một lỗi một issue, severity + priority, môi trường).
6. **Viết test tự động** cho ca thiết kế mới nếu tự động hóa được; tên test chứa mã TC: `TestX_..._TC_AUTH_004` / `it("... [TC_AUTH_004]")`.
