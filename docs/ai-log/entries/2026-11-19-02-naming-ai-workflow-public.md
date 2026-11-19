---
title: Chuẩn hóa tên file/thư mục và đưa quy trình AI lên GitHub
date: 2026-11-19
tool: Claude (claude.ai, phiên có công cụ)
model: claude-opus-5-5
surface: claude.ai
type: docs
scope: M15, M18, M25, M26
---

## Mục tiêu
Đưa quy trình AI (CLAUDE.md, .claude/, nhật ký AI, hook) lên GitHub để giải thích được khi vấn đáp; rà soát và chuẩn hóa tên file, thư mục theo quy ước phổ biến.

## Prompt đã dùng
> Sau một lúc suy nghĩ thì tôi nghĩ vẫn nên để quytrinhf AI lên để khi giảng viên hỏi tôi vẫn giải thích được. Ngoài ra tôi cảm thấy viết đặt tên file/ fordel chưa đúng chuẩn lắm bạn ra soát lại rồi cập nhập giúp tôi

## Nội dung AI tạo ra
- Đổi tên: `DEVELOPMENT.md` → `CONTRIBUTING.md`; `DESIGN.md` → `docs/design/design-system.md`; `SETUP-CLAUDE.md` → `docs/setup.md`; `capture-prompt.py` → `capture_prompt.py`; `_template.md` → `entry-template.md`; 13 artboard → `docs/design/artboards/NN-<ten>.html`; gói Go `transport/http` → `transport/httpapi`; thư mục kiểm thử `docs/testing/`.
- Thêm `CONTRIBUTING.md` §10 Quy ước đặt tên (repo, Go, CSDL/API, React/TS, Git), `docs/README.md`, `docs/testing/README.md`.
- Cập nhật mọi tham chiếu trong CLAUDE.md, skill, agent, hook, script, README; `.gitignore` chỉ còn bỏ qua secret, `.ai-log/`, `dist/`, cấu hình cá nhân.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Quy ước nhánh mâu thuẫn: DEVELOPMENT.md ghi `feat/<module>-<mo-ta>`, CLAUDE.md ghi `feature/<ten>` | Viết hai file ở hai thời điểm, không đối chiếu chéo | Rà soát đặt tên theo yêu cầu người dùng | Thống nhất `feature/<ten>` trong CONTRIBUTING.md §1 và §10 |
| Đặt gói Go là `transport/http` — tên package `http` trùng `net/http`, buộc phải đặt alias khi import | Đặt theo tên tầng kiến trúc, bỏ qua quy ước tránh trùng thư viện chuẩn | Rà soát quy ước Go | Đổi thành `transport/httpapi` |
| Đặt tên artboard theo tên gốc của Claude Design (`Artboard-m0tz.dc.html`, PascalCase, có mã ngẫu nhiên) | Giữ nguyên tên công cụ xuất ra | Rà soát đặt tên | Đổi sang `NN-kebab-case.html`, ghi bảng đối chiếu tên gốc |

## Kiểm chứng
- Tìm toàn repo không còn tham chiếu tên cũ (trừ nhật ký lịch sử).
- `python3 -m json.tool .claude/settings.json`, `bash -n` cho hook/script, `py_compile` cho script Python.
- Chạy thử commit trên repo tạm: hook `commit-msg` chấp nhận khi entry được stage, chặn khi thiếu; `ai_log.py check` chạy được.
