---
title: DESIGN.md theo 13 artboard và hệ thống AI Usage Log
date: 2026-11-19
tool: Claude (claude.ai, phiên có công cụ)
model: claude-opus-5-5
surface: claude.ai
type: docs
scope: M15–M18, P7
---

## Mục tiêu
(1) Viết lại DESIGN.md bám bản thiết kế đã chốt trên Claude Design. (2) Dựng quy trình ghi Nhật ký sử dụng AI gắn với commit, khớp form nộp bài của trường.

## Prompt đã dùng
> Tôi vừa thêm cho bạn một số yêu cầu thiết kế giao diện và có giao diện sẳn để bạn tham khảo
> Ngoài ra tôi trong việc làm với AI tôi cần một Nhật ký sử dụng AI (AI Usage Log) rõ ràng. Nên cho tôi một skill trong việc comiit code sau này để tôi có một Nhật ký sử dụng AI rõ ràng. Bạn có sáng kiến gì không
> Tôi cảm thấy ổn rồi và tôi muốn bạn hướng dẫn từng bước chạy setup tren termal trước, và tôi sẽ gửi cho  bạn một link githusb để bạn đẩy code và chúng ta chốt lại là những tình năng khác kanban tôi sẽ vibe code chỉ tự code tính năng kanban để thuyết trình cho giảng viên. Mội lần commit và push code thì sẽ là từng nhánh feature tính năng, xong tính năng nào test mọi lỗ hỏng, thao tác người dùng, validate, bảo mật, .. mới bắt đầu tới tính năng tiếp theo. Bạn hiểu tôi chứ

## Nội dung AI tạo ra
- `DESIGN.md` bản 2: token đầy đủ, quy cách chữ/số kiểu Việt Nam, app shell, bảng component, thẻ công việc, trạng thái màn hình, bản đồ artboard → màn hình, bảng điểm lệch trong thiết kế.
- Cập nhật `frontend/CLAUDE.md`, skill `react-feature`, agent `ui-reviewer`, `docs/design/README.md`.
- Skill `feature-gate` (cổng kiểm tra bảo mật, validate, UX trước PR), quy tắc nhánh và quy tắc nhánh Kanban do người dùng tự code (`CLAUDE.md` §7b), `README.md`.
- Hệ thống nhật ký AI: skill `ai-usage-log`, hook `capture-prompt.py`, git hook `.githooks/commit-msg`, `scripts/ai-log/ai_log.py`, `docs/ai-log/` (README, template, entries).

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Hook ghi prompt bản đầu dùng `python3 - <<heredoc` nên heredoc chiếm stdin, dữ liệu JSON của hook bị mất — hook sẽ không ghi được gì | Không để ý stdin của script đã bị heredoc dùng | AI tự đọc lại trước khi chạy thử | Viết lại thành file `capture-prompt.py` riêng, đọc stdin trực tiếp |

## Kiểm chứng
- Chạy thử `ai_log.py` trên repo git tạm: new → commit có trailer → human-diff → check → form → export.
- Chạy thử git hook `commit-msg`: chặn commit có `AI-Assisted` mà thiếu `AI-Log`.
- Chạy thử hook ghi prompt với dữ liệu giả: ghi đúng, che được API key.
