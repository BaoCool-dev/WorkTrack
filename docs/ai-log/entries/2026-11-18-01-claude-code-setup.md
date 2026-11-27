---
title: Bộ cấu hình Claude Code cho WorkTrack
date: 2026-11-18
tool: Claude (claude.ai, phiên có công cụ)
model: claude-opus-5-5
surface: claude.ai
type: docs
scope: M15–M26, P2
---

## Mục tiêu
Dựng bộ cấu hình hỗ trợ lập trình (CLAUDE.md, skills, MCP, plugin, hook, VS Code) cho stack Go + TypeScript, dựa trên bộ mẫu của dự án Better Attendance (C# + TypeScript).

## Prompt đã dùng
> Hiện tại mọi thiết kế, UI đã được cơ bản chốt, hiện tại tôi đang muốn tào một hệ thống setup để hỗ trợ việc code gồm claude.mc, skill.md, design.md, mcp và pugin nếu cần các skill phù hợp trên mã nguồn mỡ github. Và có thể điểu chỉnh trên cả vscode và terminal. Tôi có gửi ban một setup cho một dự án khác bằng C# và typescript. Hiện tại tôi sử dụng Go + Typescript.

## Nội dung AI tạo ra
- `CLAUDE.md`, `backend/CLAUDE.md`, `frontend/CLAUDE.md`: bối cảnh, stack đã chốt, 13 quy tắc cứng, danh sách việc chưa chốt.
- `DESIGN.md` (bản 1), `DEVELOPMENT.md`: quy tắc kỹ thuật gắn mã metric M19–M38.
- 8 skill trong `.claude/skills/`, 3 agent trong `.claude/agents/`, 2 hook (format, chặn sửa sqlcgen/migration).
- `.claude/settings.json` (quyền, hook, plugin), `.mcp.json` (context7, gopls, postgres, playwright), `.vscode/`, `Makefile`, `scripts/setup-dev.sh`, PR template.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Bật plugin `frontend-design` trong cấu hình dự án dù plugin này hướng tới phong cách "táo bạo, khác biệt", trái với `DESIGN.md` đã chốt | Chọn plugin theo tên/độ phổ biến trước khi đọc mô tả của nó | AI tự rà mô tả plugin trước khi chốt | Bỏ khỏi `enabledPlugins`, ghi chú trong SETUP-CLAUDE.md |
| `DESIGN.md` bản 1 viết từ prompt gửi Claude Design chứ không từ bản thiết kế thật → thiếu token (màu viền ô nhập, nền khối AI, bóng đổ), thiếu quy cách app shell | Chưa có bản thiết kế trong repo; AI suy ra từ tài liệu gián tiếp | Người dùng báo đã có thiết kế; đối chiếu 13 artboard | Viết lại `DESIGN.md` ngày 19/11 (xem entry 2026-11-19-01) |

## Kiểm chứng
- Kiểm tra JSON hợp lệ (`python3 -m json.tool`) cho settings, .mcp.json, .vscode.
- `bash -n` cho các script; chạy thử hook: sửa file trong `sqlcgen/` bị chặn (exit 2), file Go được `gofmt`.
- Chưa chạy trong một phiên Claude Code thật — cần làm theo SETUP-CLAUDE.md §6.
