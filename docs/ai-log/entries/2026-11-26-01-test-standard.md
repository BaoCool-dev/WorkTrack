---
title: Chuẩn kiểm thử theo bài giảng HCM-UTE và script kiểm thử toàn bộ chức năng
date: 2026-11-26
tool: Claude (claude.ai, phiên có công cụ)
model: claude-opus-5-5
surface: claude.ai
type: test
scope: M27–M32
---

## Mục tiêu
Một lệnh chạy kiểm thử toàn bộ chức năng và lập bảng test case; nâng chất lượng bộ test theo bài giảng Kiểm thử phần mềm (SOTE431079) và tài liệu Test Techniques.

## Prompt đã dùng
> Tôi muốn chạy một lệnh để nó chạy toàn bộ feature để tạo nên một bảng test case cho từng chức năng của dự án.
> Bạn đọc qua tài liệu để có một bộ test thật chất lượng hơn

## Nội dung AI tạo ra
- `docs/testing/test-design-standard.md`: quy trình 7 giai đoạn ánh xạ vào repo, 15 trường của test case, bảng chọn kỹ thuật (EP, BVA, bảng quyết định, chuyển trạng thái, use case, ràng buộc, đoán lỗi), phi chức năng, báo cáo lỗi và vòng đời lỗi trên GitHub Issues, báo cáo tổng kết và tiêu chí kết thúc.
- Skill `test-case-design`; agent `qa-tester` trỏ tới chuẩn; mẫu `.github/ISSUE_TEMPLATE/bug_report.md`.
- `scripts/run-all-tests.sh` (chạy thật Go unit/API, tích hợp PostgreSQL trên CSDL riêng, Vitest, độ phủ), `scripts/test-summary.py`, `scripts/test-cases.sh` (điều phối: chạy thật → Claude phân tích, thiết kế, viết test còn thiếu, chạy lại, báo lỗi, lập bảng và báo cáo).

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Bản đầu của script chỉ gom test có sẵn thành bảng, mã ca `TC-CF01-001`, không có bước phân tích điều kiện kiểm thử, không áp dụng kỹ thuật thiết kế, không có bug report và báo cáo tổng kết | Thiết kế theo thói quen chung, chưa đối chiếu chuẩn môn học | Đọc bài giảng người dùng cung cấp | Viết chuẩn kiểm thử, đổi mã ca theo `TC_<MODULE>_<NNN>`, thêm phân tích, kỹ thuật, bug report, báo cáo |
| Commit trước để lại file khóa trong `.git` | Phiên không còn quyền xóa nên git không tự dọn | Cảnh báo `unable to unlink` | Đổi tên file khóa, sau đó xin quyền xóa và dọn |

## Kiểm chứng
- `bash -n` cho script; `py_compile` cho `test-summary.py`; chạy bộ tóm tắt với dữ liệu mẫu định dạng Go và Vitest: đếm đúng, gắn mã CF đúng, không trùng.
- Chưa chạy thật trên máy người dùng.
