---
title: Chế độ tự chạy qua nhiều nhánh và file quyết định
date: 2026-11-19
tool: Claude (claude.ai, phiên có công cụ)
model: claude-opus-5-5
surface: claude.ai
type: docs
scope: quy trình phát triển, M23, M34
---

## Mục tiêu
Cho Claude Code tự làm lần lượt các nhánh tính năng mà không hỏi lại những điều đã chốt, để người dùng review PR sau.

## Prompt đã dùng
> Có cách nào để làm các nhánh feature rồi tôi kiểm tra sau được không, tôi muốn làm việc khác, thay vì ngồi lựa chọn những yếu tố mình đã chốt trước đó

## Nội dung AI tạo ra
- `docs/decisions.md`: mục A (đã chốt: thư viện, công cụ), B (11 mặc định tạm: SP 1:4, quyền CEO, JWT, ngưỡng 20%, mã lỗi, kích hoạt, mật khẩu, phân trang, múi giờ, dữ liệu mẫu), C (giả định Claude tự chọn khi chạy).
- `CLAUDE.md` §6 trỏ về decisions.md; §7c chế độ tự chạy: không hỏi, được/không được làm gì, khi nào dừng, PR draft nối tiếp.
- `.claude/settings.json`: mở quyền cần cho chạy không người trông (push nhánh tính năng, `gh pr create`, docker compose, migrate dev), chặn cứng push/merge `main`, force-push, xóa volume; bật MCP dự án.
- `scripts/auto-run.sh`: chạy `claude -p` lần lượt giai đoạn 1 (scaffold → auth → org → projects → tasks, dừng trước kanban) và giai đoạn 2 (subtasks → logtime → review → reports → ai); kiểm tra gate và push sau mỗi nhánh; giữ máy thức; thông báo macOS.
- `feature-gate`: sửa bước push/PR cho chế độ tự chạy.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Phạm vi nhánh tasks trong script chứa câu tự mâu thuẫn ("cung cấp API chuyển cột… nếu cần? KHÔNG") | AI viết luồng suy nghĩ thẳng vào chuỗi prompt | AI đọc lại script | Viết lại thành một quy tắc rõ: không tạo API chuyển cột |
| Script lấy nhánh hiện tại làm nhánh gốc → nhánh đầu tiên sẽ tạo PR với base là chính nó | Không xét trường hợp đang đứng sẵn trên nhánh đầu kế hoạch | Chạy `--dry-run` | Thêm `--base`, mặc định `main` |
| `--dry-run` vẫn kiểm tra đủ công cụ nên không chạy được ở máy thiếu `gh` | Đặt bước kiểm tra trước bước in kế hoạch | Chạy `--dry-run` | Đưa phần in kế hoạch lên trước |
| Lỗi backtick trong `feature-gate` từ lần đổi tên trước (`docs/gates/<loai>-<ten>.md (vd. ...)`) | Thay chuỗi tự động chèn vào trong cặp backtick | Đọc lại skill | Sửa định dạng |

## Kiểm chứng
- `bash -n scripts/auto-run.sh`; `--dry-run` cho giai đoạn 1 và 2 in đúng kế hoạch.
- `python3 -m json.tool .claude/settings.json`.
- Chưa chạy thật `claude -p` — chạy lần đầu trên máy người dùng.

## Bổ sung 19/11 16:15 — chạy không cần Docker, sửa hook trailer
**Prompt:**
> hiện tại chưa cần tới docker chạy trên local trước được không

**AI tạo ra:** quyết định B12 (PostgreSQL + Mailpit qua Homebrew, test tích hợp chạy ở CI, `make dev-local`); `auto-run.sh` không bắt buộc Docker, dùng `--permission-mode auto`; hook `commit-msg` đọc trailer bằng `git interpret-trailers`; skill `ai-usage-log` yêu cầu trailer liền với `Co-Authored-By`.

**Lỗi của AI:**
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Hook `commit-msg` cho qua 2 commit (`fix(ci)`, `docs(gate)`) mà git không nhận trailer `AI-Log`, vì `Co-Authored-By` nằm ở đoạn riêng phía sau | Hook tìm chữ "AI-Log:" ở bất kỳ dòng nào, trong khi git chỉ đọc trailer ở đoạn cuối | `ai_log.py check` báo 2 commit thiếu AI-Log | Hook dùng `git interpret-trailers --parse`; viết lại message 2 commit (gộp trailer vào đoạn cuối) |
| Script bắt buộc Docker dù người dùng chưa cần | Giả định môi trường dev luôn có Docker | Script dừng ở "Thiếu lệnh: docker" | Docker thành tùy chọn (B12) |
