# Nhật ký sử dụng AI — WorkTrack

Minh chứng cho bản cam kết **P7** và metric **M15–M18**. Hạn nộp trên hệ thống: **22:32 08/12/2026**.

## Cách hệ thống hoạt động

```
Bạn gõ prompt ──► hook UserPromptSubmit ──► .ai-log/prompts-YYYY-MM.jsonl   (nguyên văn, đã che secret, KHÔNG commit)
                                                   │
Claude code xong ──► skill ai-usage-log ──► docs/ai-log/entries/<ngày>-<số>-<tên>.md   (1 file / 1 việc)
                                                   │
git commit + trailer ── AI-Assisted: Claude Code (model)
                     └─ AI-Log: docs/ai-log/entries/....md      ◄── hook commit-msg chặn nếu thiếu
                                                   │
Bạn review & sửa ──► commit "Human-Edit: yes" ──► ai_log.py human-diff  (diff khách quan phần bạn sửa)
                                                   │
Cuối kỳ ──► ai_log.py check / form / export ──► 5 ô của form nộp bài + AI_USAGE_LOG.md/.docx
```

Vì sao thiết kế như vậy:
- **Prompt thật, không dựng lại**: hook ghi tự động mọi prompt; skill chỉ chọn và rút gọn.
- **Đối chiếu được với commit (M15)**: trailer `AI-Log` trên commit trỏ về entry; script đọc ngược từ `git log`, không cần ghi hash bằng tay.
- **Không quên ghi**: hook git `commit-msg` chặn commit có dấu hiệu AI mà thiếu `AI-Log`.
- **"Phần bạn đã sửa" có bằng chứng**: mẫu hai commit (AI rồi Human-Edit) cho ra `git diff` khách quan.
- **Mỗi việc một file**: không xung đột khi sửa song song, dễ đọc lại khi vấn đáp (M17).

## Cài đặt (một lần)
```bash
git config core.hooksPath .githooks      # bật hook commit-msg
python3 scripts/ai-log/ai_log.py check   # kiểm tra
```
Hook ghi prompt đã khai báo trong `.claude/settings.json` (chạy cả ở terminal và VS Code).

## Lệnh
| Lệnh | Việc |
|---|---|
| `ai_log.py new --slug <tên> --type code --scope "CF-08"` | Tạo entry từ `entry-template.md` |
| `ai_log.py prompts --since-last-ai-commit` | In prompt đã ghi từ commit AI gần nhất |
| `ai_log.py human-diff <entry>` | Chèn diff phần bạn sửa sau commit AI |
| `ai_log.py check` | Kiểm tra: mục thiếu, commit AI thiếu nhật ký, đếm lỗi/ảo giác (mục tiêu ≥ 5) |
| `ai_log.py form` | In khối 5 ô cho từng công cụ để dán vào form |
| `ai_log.py export --docx` | Xuất `dist/AI_USAGE_LOG.md` (+ `.docx` nếu có pandoc) |

Với Claude: chỉ cần nói **"commit kèm nhật ký AI"**.

## Quy trình kiểm soát đầu ra AI (M18)
<!-- policy:start -->
1. Đọc lại toàn bộ diff do AI sinh trước khi commit; tự giải thích được từng đoạn (chuẩn bị vấn đáp M17).
2. Đối chiếu API thư viện với tài liệu chính thức (Context7 / trang chủ) — không tin phiên bản hay hàm do AI tự nêu.
3. `make lint` và `make test` phải xanh; logic nghiệp vụ có test với ca biên và ca âm.
4. Kiểm tra giấy phép của mọi dependency mới do AI đề xuất.
5. Không đưa secret, dữ liệu người dùng thật hay dữ liệu nhạy cảm vào prompt; hook ghi prompt tự che chuỗi giống secret.
6. Mọi kết quả AI trong sản phẩm (Gemini) chỉ là bản xem trước, người dùng quyết định.
7. Mỗi lỗi/ảo giác phát hiện được ghi vào entry kèm nguyên nhân và cách sửa.
<!-- policy:end -->

## Công cụ AI đang dùng
| Công cụ | Model / phiên bản | Dùng cho |
|---|---|---|
| Claude (claude.ai, Claude Code) | [ĐIỀN model id thật] | Thiết lập dự án, viết code, test, review |
| Claude Design | — | Thiết kế giao diện (13 artboard) |
| [khác: ChatGPT, Copilot…] | | |
