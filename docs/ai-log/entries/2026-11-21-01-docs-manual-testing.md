---
title: docs manual testing
date: 2026-11-21
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: docs
scope: CF-01, CF-02, CF-03, CF-06
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác
     Thêm dòng "commit: none" ở phần đầu nếu việc này không tạo commit (vd. thiết kế, soạn báo cáo ngoài repo). -->

## Mục tiêu
Viết `docs/testing/manual-test-guide.md` — hướng dẫn kiểm thử thủ công (không Docker, B12) cho các chức năng đã có: CF-01 (đăng nhập, kích hoạt), CF-02 (tổ chức, tài khoản), CF-03 (dự án, cột, luật chuyển), CF-06 (công việc cha, nhãn, cấu hình). Không sửa code sản phẩm. Nhật ký chế độ tối giản (một entry).

## Prompt đã dùng
<!-- Dán NGUYÊN VĂN prompt chính (lấy từ: python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit).
     Mỗi prompt một dòng bắt đầu bằng ">". Dài quá thì rút gọn và ghi [rút gọn]. Không dán secret. -->
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: docs/manual-testing (mọc từ feature/tasks). PR sẽ tạo với --base feature/tasks --draft. [rút gọn]
> Phạm vi nhánh này: KHÔNG sửa code sản phẩm. Viết docs/testing/manual-test-guide.md (tiếng Việt, bước đánh số, dựa trên code thật): cài không Docker, đăng nhập/đăng xuất/refresh/khóa cho 4 vai trò, Admin tạo tài khoản → Mailpit → kích hoạt, từng chức năng theo vai trò kèm ca lỗi/quyền và SQL kiểm, psql và pgAdmin, curl có token, đặt lại an toàn, checklist CF-01/02/03/06. [rút gọn]
> Chạy thử từng lệnh và câu SQL trong tài liệu trên máy để chắc chắn đúng. Gate cho nhánh tài liệu: mọi lệnh và SQL chạy được; không cần test UI. Nhật ký AI chế độ tối giản (một entry). [rút gọn]

## Nội dung AI tạo ra
<!-- Liệt kê theo file/phần: AI viết gì. Có thể kèm git diff --stat của commit AI. -->
- `docs/testing/manual-test-guide.md` (≈ 1000 dòng, 8 mục): cài đặt không Docker (biến môi trường lấy từ `backend/internal/config`, `make migrate-up`, `make seed`, `make dev-local`, URL, lỗi thường gặp); đăng nhập/refresh/đăng xuất/khóa tài khoản bằng curl + SQL; Admin tạo tài khoản → đọc thư trong Mailpit API → kích hoạt; ca đúng/lỗi/quyền cho CF-02, CF-03, CF-06 kèm mã lỗi thật lấy từ `errors.go` và SQL kiểm tra; psql/pgAdmin + bảng "bảng nào chứa gì"; curl có token; đặt lại dữ liệu (seed, mở khóa, dọn dữ liệu thử trong transaction ROLLBACK, reset toàn bộ — không chạy); checklist CF-01/02/03/06.
- Mật khẩu mẫu để dạng `<MK_ADMIN>`… vì `make seed` sinh mật khẩu mới mỗi lần.
- `docs/testing/README.md` thêm dòng; `docs/decisions.md` mục C thêm giả định của nhánh; báo cáo `docs/gates/docs-manual-testing.md`.
- Công cụ kiểm chứng tạm (ngoài repo, `/tmp`): script tách mọi khối bash/sql trong tài liệu, thay mật khẩu, chạy tuần tự trên máy dev.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Token kích hoạt lấy từ thư Mailpit dính `\r` cuối dòng → `ACTIVATION_TOKEN_INVALID` | AI tách token từ nội dung thư dạng text mà không tính dòng CRLF | Chạy thử khối lệnh kích hoạt | Thêm `tr -d '\r\n'` |
| Câu SQL hệ số B1 dùng sai tên cột/khóa của `system_setting` | AI đoán lược đồ thay vì đọc migration | psql báo lỗi khi chạy thử | Đọc migration, dùng `int_value` và `key='story_point_hours'` |
| Đường dẫn jq lấy tên cột của công việc sai (`.column.name`) | AI đoán hình dạng JSON | jq trả `null` khi chạy thử | Đối chiếu DTO trong `task_handler.go`, sửa đường dẫn |
| Chạy liền các lệnh đăng nhập bị 429 | AI quên giới hạn tần suất login (burst 10, 1 lần/6 giây) | Chạy thử liên tiếp | Ghi chú giới hạn trong tài liệu, script kiểm chứng chờ giữa các lần |
| Khối dọn dữ liệu §7.3 làm PostgreSQL sập (`Assertion failed … collate.c`) ở `LIKE 'Khối thử %'` | Lỗi libc macOS khi so sánh chuỗi có dấu dưới collation `en_US.UTF-8`; AI không lường trước | Chạy thử: mất kết nối, log Postgres có assertion | Thêm `COLLATE "C"` cho các `LIKE` có chữ tiếng Việt + ghi chú lý do; chạy lại 3 lần không lỗi |

## Kiểm chứng
<!-- Lệnh đã chạy và kết quả thật: make lint, make test, test nào, kiểm thử tay luồng nào. -->
- Môi trường: PostgreSQL 16 + Mailpit (Homebrew), `make dev-local`, CSDL dev `worktrack`.
- Chạy mọi khối bash/sql của tài liệu theo thứ tự (trừ các khối đánh dấu `verify:skip`: tạo user/CSDL, `cp .env.example`, `make dev-local`, psql tương tác, reset toàn bộ). Kết quả khớp mô tả trong tài liệu; khối §7.3 lỗi lần đầu (xem bảng trên), sau sửa chạy lại 3 lần → ROLLBACK sạch.
- `make seed` chạy riêng một lần trước (mật khẩu đổi mỗi lần chạy).
- Lệnh xóa dữ liệu (§7.4 `goose reset`) không chạy — phiên tự chạy không được xóa dữ liệu; đã kiểm cú pháp bằng `goose --help`.
- Không sửa code sản phẩm → `make lint`, `make test` chạy lại ở gate (xem `docs/gates/docs-manual-testing.md`).
