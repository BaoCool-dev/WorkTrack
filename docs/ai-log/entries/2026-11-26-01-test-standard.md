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

## Bổ sung 27/11 — script tích hợp từng bước
**Prompt:**
> Hiện tại sau khi kiểm thử tôi muốn claude code từng bước gộp các nhánh lại những là tạo thành một nhánh mới chứ chưa được push lên main. ... Mỗi lần gộp là phải test lại, xem có config gì không, coi thử có code rác nào mà vẫn còn mà không có tác dụng gì trong hệ thống. Cứ làm từng bước đó rồi tôi sẽ là người tự kiểm tra lại.

**AI tạo ra:** `scripts/integrate.sh` (gộp 9 nhánh lần lượt vào `release/v1`, sau mỗi bước: test thật, deadcode, knip, Claude rà cấu hình và dọn code rác theo quy tắc "chỉ xóa khi bản cuối không dùng", chạy lại test, báo cáo `docs/integration/NN-*.md`, tag `integ/NN-*`); `run-all-tests.sh` chạy được ở bước chưa có backend/frontend và tự `npm ci` khi lockfile đổi; quyền `.env`: chỉ chặn file bí mật, cho phép đọc/sửa `.env.example`.

**Lỗi của AI:**
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Bản đầu của `integrate.sh` gọi `./scripts/run-all-tests.sh` trong repo, nhưng khi nhánh tích hợp còn ở bước đầu (tạo từ `main`) thì file này chưa tồn tại, và chính script đang chạy cũng bị thay khi đổi nhánh | Quên rằng chuyển nhánh làm thay đổi thư mục `scripts/` | AI tự rà luồng chạy trước khi giao | Script tự chép công cụ ra thư mục tạm rồi chạy từ đó |
| Luật `deny` `.env*` chặn luôn `.env.example`, khiến Agent ở các nhánh trước không cập nhật được file mẫu cấu hình và phải nhờ người dùng | Mẫu glob quá rộng | Rà quyền khi thêm bước kiểm cấu hình | Chỉ chặn `.env`, `.env.local`, `.env.development`, `.env.production`, `.env.test` |

## Bổ sung 27/11 07:45 — chạy cả ngày không người trông
**Prompt:**
> Hồm nay tôi không có thời gian tối tui mới rãnh tôi muốn bạn từ làm trước trong ngày tôi tui coi lại sau, vì từng nhánh nên tối dễ dàng kiểm tra và backup lại nên bạn từ gộp từng chức năng rồi test tồi đẩy lên nhánh mới lần lượt như thế đến xong dự án

**AI tạo ra:** `scripts/day-run.sh` (gộp nhánh cũ → làm 5 chức năng còn lại trên nền `release/v1` → gộp từng chức năng mới); `integrate.sh` tự thêm các nhánh giai đoạn 2 nếu đã có, đẩy nhánh sao lưu `integration/NN-*` và tag mỗi bước; `auto-run.sh` cho `--base` thắng khi dùng cùng `--from`, `--dry-run` hiển thị đúng các nhánh sẽ chạy.

**Lỗi của AI:**
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| `auto-run.sh --from X --base Y` bỏ qua `--base`: nhánh đầu tiên mọc từ nhánh đứng trước X trong kế hoạch | Vòng lặp gán BASE theo các nhánh bị bỏ qua | Đọc lại logic khi thêm day-run | `--base` thắng ở nhánh bắt đầu |
| `--dry-run` liệt kê cả nhánh sẽ bị bỏ qua bởi `--from` | Phần in kế hoạch không xét `--from` | Chạy thử `--dry-run` | In đúng các nhánh sẽ chạy |
