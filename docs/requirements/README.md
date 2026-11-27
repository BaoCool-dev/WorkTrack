# docs/requirements — nguồn sự thật nghiệp vụ

Claude đọc thư mục này khi làm một issue. Đặt vào đây (dạng Markdown để Claude đọc nhanh, không dùng .docx):

| File | Nội dung | Lấy từ |
|---|---|---|
| `commitment.md` | Bản cam kết: CF-01..CF-12 (Lõi), Mở rộng, Ngoài phạm vi, KPI, NFR | Bản cam kết sản phẩm và metric |
| `use-cases.md` | UC01..UC16, luồng chính/phụ, acceptance criteria | Chương 3 |
| `functional-requirements.md` | YCCN-01..YCCN-50, YCPCN, YCHT | Chương 3 mục 3.1 |
| `business-rules.md` | R01..R20, quyết định D01..D19 | Chương 3–4 |
| `data-model.md` | 30 bảng, cột, ràng buộc (ERD dạng bảng) | Chương 4 |
| `permissions.md` | Ma trận phân quyền 4 vai trò (khi đã chốt) | Chương 3 |
| `decisions.md` | Nhật ký quyết định đã chốt theo ngày | Project WORK TRACK |

Quy tắc: mã (UC, YCCN, R, D, CF) **không đổi số**. Bỏ yêu cầu → đánh dấu "đã bỏ", không đánh số lại.
