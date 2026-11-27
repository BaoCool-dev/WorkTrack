---
title: Tích hợp từng bước các nhánh vào release/v1
date: 2026-11-27
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: review
scope: Tích hợp release/v1 (bước 01..09)
---

## Mục tiêu
Gộp lần lượt các nhánh tính năng vào `release/v1` (không phải `main`). Mỗi bước: đọc kết quả test sau gộp, rà cấu hình, dọn code rác (chỉ khi bản cuối `test/test-cases` không còn dùng), chạy lại test, viết báo cáo `docs/integration/<NN>-<nhanh>.md`.

## Prompt đã dùng
### Bước 01 — chore/ai-workflow
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng. Đang TÍCH HỢP từng bước vào nhánh release/v1 (KHÔNG phải main). Bước 01/9: vừa gộp nhánh chore/ai-workflow … Việc phải làm: 1. TEST … 2. CẤU HÌNH … 3. CODE RÁC … 4. Chạy lại … 5. BÁO CÁO docs/integration/01-ai-workflow.md … 6. Commit báo cáo qua skill ai-usage-log … [rút gọn]

## Nội dung AI tạo ra
### Bước 01 — chore/ai-workflow
- `.gitignore`: thêm `.playwright-mcp/` (trước đó chỉ được bỏ qua nhờ `.git/info/exclude` cục bộ).
- `docs/integration/01-ai-workflow.md`: báo cáo bước 01 (test, cấu hình, code rác, việc người dùng tự kiểm, điểm không chắc chắn).
- Entry nhật ký này.

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Bước 01: commit `fix(config)` đầu tiên bị hook `commit-msg` chặn vì trailer `AI-Log` trỏ tới entry chưa tạo | Viết commit trước khi tạo entry nhật ký | Hook báo "file không tồn tại" | Tạo entry rồi commit lại kèm entry |

## Kiểm chứng
### Bước 01
- Test sau gộp và sau dọn: 0 test (chưa có backend/frontend ở bước này).
- `gitleaks detect`: no leaks found (85 commit).
- JSON hợp lệ: `.claude/settings.json`, `.mcp.json`, `.vscode/*.json`.
