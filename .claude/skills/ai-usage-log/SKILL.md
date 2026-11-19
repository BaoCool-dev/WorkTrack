---
name: ai-usage-log
description: Commit code kèm Nhật ký sử dụng AI (AI Usage Log) của WorkTrack — tạo/cập nhật entry trong docs/ai-log/entries, gắn trailer AI-Assisted và AI-Log vào commit, ghi lỗi/ảo giác AI, và xuất nhật ký để nộp. Dùng MỖI KHI commit thay đổi có AI tham gia, khi người dùng nói "commit", "commit kèm nhật ký AI", "ghi AI log", "xuất nhật ký AI", hoặc khi Claude vừa sai rồi tự sửa.
---

# Commit kèm Nhật ký sử dụng AI

Nhật ký này được GVHD/GVPB chấm (rubric M15–M18, bản cam kết P7) qua form có 5 ô: **Công cụ AI · Phiên bản/model · Prompt đã dùng · Nội dung AI tạo ra · Phần bạn đã sửa/hoàn thiện**. Mọi thứ dưới đây phục vụ việc điền 5 ô đó trung thực và chứng minh được bằng git.

## Nguyên tắc trung thực (không được phá)
1. **Prompt là nguyên văn** từ `.ai-log/` (hook ghi tự động), không dựng lại theo trí nhớ. Chỉ được rút gọn, ghi `[rút gọn]`.
2. **Không bao giờ tự điền "Phần tôi đã sửa / hoàn thiện"** thay người dùng. Giữ `[NGƯỜI DÙNG ĐIỀN]` hoặc hỏi người dùng rồi ghi lại đúng lời họ.
3. **Ghi lỗi của chính Claude.** Trong phiên Claude đã đề xuất API không tồn tại, sai phiên bản, sai nghiệp vụ, test sai, lệnh hỏng… → một dòng trong bảng "Lỗi / ảo giác". Không giấu, không làm nhẹ đi.
4. Model ghi đúng chuỗi model id đang chạy (vd. `claude-opus-5-5`). Không chắc → hỏi.

## Quy trình khi commit

### 1. Xem thay đổi
```bash
git status --short
git diff --stat            # và git diff --cached --stat nếu đã stage
```
Tách commit nếu thay đổi gồm nhiều việc không liên quan (mỗi việc một entry).

### 2. Tạo hoặc chọn entry
- Việc mới: `python3 scripts/ai-log/ai_log.py new --slug <ten-ngan> --type <code|test|docs|design|review|debug|research> --scope "CF-xx, UCxx" --model <model-id> --surface <terminal|vscode>`
- Việc đang làm dở (cùng tính năng, nhiều commit): mở lại entry cũ, bổ sung.

### 3. Điền entry
| Mục | Lấy từ đâu |
|---|---|
| Mục tiêu | Issue / yêu cầu của người dùng, kèm mã CF/UC/YCCN |
| Prompt đã dùng | `python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit` → chọn prompt chính, mỗi dòng bắt đầu bằng `>` |
| Nội dung AI tạo ra | Những gì Claude viết trong phiên: theo file/phần, ngắn gọn, cụ thể (hàm, test, migration) |
| Phần tôi đã sửa / hoàn thiện | **Hỏi người dùng** (bước 4). Không tự viết |
| Lỗi / ảo giác của AI | Rà lại phiên: chỗ nào Claude sai rồi sửa, test đỏ vì code AI, người dùng bác bỏ đề xuất. Mỗi lỗi: mô tả · nguyên nhân · cách phát hiện · cách sửa |
| Kiểm chứng | Lệnh đã chạy thật và kết quả (`make lint`, `make test`, test cụ thể) |

### 4. Hỏi người dùng đúng một câu
> "Trước khi commit: bạn đã đọc/sửa gì trong phần này? (vd. đổi tên hàm, sửa điều kiện, viết lại test, bác bỏ đề xuất X). Nếu chưa review, tôi sẽ commit phần AI trước để bạn sửa và commit riêng."

- Người dùng trả lời → ghi **nguyên ý** của họ vào mục "Phần tôi đã sửa / hoàn thiện".
- Chưa review → giữ `[NGƯỜI DÙNG ĐIỀN]`, dùng **mẫu hai commit** (bước 6).

### 5. Commit với trailer
```bash
git add <các file thay đổi> docs/ai-log/entries/<entry>.md
git commit -m "feat(timesheet): chặn log giờ âm [CF-08][UC07]" \
  -m "AI-Assisted: Claude Code (claude-opus-5-5)
AI-Log: docs/ai-log/entries/2026-11-20-01-log-time.md"
```
- Tiêu đề: Conventional Commits + mã yêu cầu.
- Trailer `AI-Assisted` + `AI-Log` ở **đoạn cuối** commit message, **liền với** `Co-Authored-By` (nếu có) — không có dòng trống xen giữa, vì git chỉ đọc trailer ở đoạn cuối. Ví dụ đoạn cuối đúng:
  ```
  AI-Assisted: Claude Code (claude-opus-5-5)
  AI-Log: docs/ai-log/entries/2026-11-20-01-log-time.md
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  ``` Hook git `commit-msg` sẽ chặn nếu thiếu `AI-Log` hoặc file nhật ký chưa được commit/stage.
- **Không ghi hash vào entry** — liên kết đi theo chiều commit → entry, script đọc ngược từ `git log`.
- Nhật ký (`docs/ai-log/entries/`) đi cùng commit lên GitHub. Prompt thô (`.ai-log/`) thì **không** — chỉ trích prompt chính vào entry.
- Không `git push`, không `--no-verify`.

### 6. Mẫu hai commit (khuyến nghị cho code quan trọng)
Cho bằng chứng khách quan về "phần bạn đã sửa":
1. Commit A — phần AI viết, có `AI-Assisted` + `AI-Log`.
2. Người dùng review, sửa tay, rồi commit B:
   ```bash
   git commit -m "refactor(timesheet): hoàn thiện sau review AI [CF-08]" \
     -m "Human-Edit: yes
   AI-Log: docs/ai-log/entries/2026-11-20-01-log-time.md"
   ```
3. Chạy `python3 scripts/ai-log/ai_log.py human-diff docs/ai-log/entries/<entry>.md` → tự chèn `git diff --stat A..B` vào mục "Phần tôi đã sửa". Người dùng bổ sung lời giải thích, commit cùng lần sửa kế tiếp.

### 7. Kiểm tra
```bash
python3 scripts/ai-log/ai_log.py check
```
Báo cho người dùng: số entry, số lỗi/ảo giác đã ghi so với mục tiêu ≥ 5, các cảnh báo còn lại.

## Việc không có commit
Dùng AI ngoài repo (Claude Design, soạn Chương 1–4, khảo sát, slide): vẫn tạo entry, thêm `commit: none` ở phần đầu, `surface: claude.ai` hoặc `claude-design`. Commit các entry này trong một commit `docs(ai-log): …` (không có trailer `AI-Assisted`).

## Xuất để nộp (hạn nộp trên hệ thống: 22:32 08/12/2026)
```bash
python3 scripts/ai-log/ai_log.py check
python3 scripts/ai-log/ai_log.py form                       # 5 ô cho từng công cụ → dán vào các thẻ "Thêm công cụ AI"
python3 scripts/ai-log/ai_log.py export --out dist/AI_USAGE_LOG.md --docx
```
- Ô "Nội dung (hỗ trợ Markdown)": dán phần 1 + 3 của `AI_USAGE_LOG.md` (tổng quan + bảng lỗi/ảo giác).
- Tệp đính kèm: `AI_USAGE_LOG.docx` (hoặc nén `.md` thành `.zip`), giới hạn 5MB.
- Đọc lại toàn bộ trước khi nộp; phần do script gom là bản nháp, người dùng chịu trách nhiệm nội dung cuối.
