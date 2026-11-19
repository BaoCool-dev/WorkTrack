# WorkTrack

Nền tảng web quản trị công việc và cộng tác nội bộ cho doanh nghiệp CNTT vừa và nhỏ: Kanban, ghi giờ thực tế trên từng đầu việc (Task-based Timesheet), báo cáo hiệu suất minh bạch và Trợ lý AI (Gemini) chỉ gợi ý.

Đồ án tốt nghiệp — Khoa Công nghệ Thông tin, Trường Đại học Công nghệ Kỹ thuật TP.HCM.

## Công nghệ
Go (chi, pgx, sqlc, goose) · PostgreSQL · WebSocket · React 19 + TypeScript + Vite + TailwindCSS v4 · Docker · GitHub Actions · Google Gemini API

## Trạng thái
Đang thiết lập dự án. Mã nguồn được phát triển theo từng nhánh tính năng (mỗi tính năng một nhánh, kiểm tra xong mới merge).

## Tài liệu
| Đường dẫn | Nội dung |
|---|---|
| `CONTRIBUTING.md` | Quy ước code, đặt tên, git, kiểm thử, bảo mật, Definition of Done |
| `docs/setup.md` | Cài môi trường phát triển (terminal, VS Code, Claude Code) |
| `docs/design/` | Hệ thống thiết kế và 13 artboard giao diện |
| `docs/ai-log/` | Nhật ký sử dụng AI — mỗi commit có AI hỗ trợ trỏ tới một entry qua trailer `AI-Log` |
| `docs/gates/` | Báo cáo kiểm tra (bảo mật, validate, UX) từng tính năng trước khi merge |
| `CLAUDE.md`, `.claude/` | Quy tắc và quy trình dùng AI (Claude Code) trong dự án |

## Quy ước code
Xem `CONTRIBUTING.md` (§10: quy ước đặt tên). Mỗi tính năng một nhánh `feature/<ten>`, merge vào `main` qua Pull Request sau khi qua kiểm tra.
