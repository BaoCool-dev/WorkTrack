# WorkTrack

Nền tảng web quản trị công việc và cộng tác nội bộ cho doanh nghiệp CNTT vừa và nhỏ: Kanban, ghi giờ thực tế trên từng đầu việc (Task-based Timesheet), báo cáo hiệu suất minh bạch và Trợ lý AI (Gemini) chỉ gợi ý.

Đồ án tốt nghiệp — Khoa Công nghệ Thông tin, Trường Đại học Công nghệ Kỹ thuật TP.HCM.

## Công nghệ
Go (chi, pgx, sqlc, goose) · PostgreSQL · WebSocket · React 19 + TypeScript + Vite + TailwindCSS v4 · Docker · GitHub Actions · Google Gemini API

## Trạng thái
Đang thiết lập dự án. Mã nguồn được phát triển theo từng nhánh tính năng (xem `CLAUDE.md` §7b).

## Tài liệu
| File | Nội dung |
|---|---|
| `SETUP-CLAUDE.md` | Cài công cụ và môi trường phát triển (terminal, VS Code) |
| `DEVELOPMENT.md` | Quy ước code, git, kiểm thử, bảo mật, Definition of Done |
| `DESIGN.md` | Hệ thống thiết kế giao diện |
| `docs/ai-log/` | Nhật ký sử dụng AI (AI Usage Log) |
| `docs/gates/` | Báo cáo kiểm tra từng tính năng trước khi merge |

## Quy ước code
Go theo Effective Go và Go Code Review Comments (`gofmt`, `golangci-lint`); TypeScript strict + ESLint + Prettier. Chi tiết: `DEVELOPMENT.md`.
