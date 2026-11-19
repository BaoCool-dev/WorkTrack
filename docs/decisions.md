# Quyết định của dự án WorkTrack

> Nguồn sự thật cho mọi lựa chọn kỹ thuật và nghiệp vụ đã chốt. Claude **không hỏi lại** những gì có trong file này.
> Mục **B** là mặc định tạm: Claude dùng luôn để không bị chặn, người dùng đổi khi review PR.
> Mục **C** là giả định Claude tự chọn trong lúc chạy — mỗi dòng ghi nhánh và lý do; người dùng duyệt khi review.

## A. Đã chốt

### Công nghệ
| Chủ đề | Quyết định |
|---|---|
| Backend | Go (phiên bản đang cài trên máy; `go.mod` ghi đúng phiên bản đó), module `github.com/BaoCool-dev/WorkTrack/backend` |
| HTTP / CSDL | chi, pgx v5 (pgxpool), sqlc (pgx/v5), goose, PostgreSQL 16 |
| Auth | golang-jwt v5, bcrypt (`golang.org/x/crypto/bcrypt`) |
| Validate | go-playground/validator v10 |
| Realtime | coder/websocket |
| ID | `github.com/google/uuid` |
| Test Go | thư viện chuẩn `testing` + `httptest`; testcontainers-go cho test tích hợp (build tag `integration`); unit test `/readyz` dùng interface `Pinger` giả lập |
| Lint Go | golangci-lint |
| Frontend | React 19, TypeScript strict, Vite, TailwindCSS v4 (`@tailwindcss/vite`), React Router |
| State | TanStack Query v5 (server state), Zustand (UI state) |
| Form | react-hook-form + zod |
| Kéo thả | dnd-kit |
| Icon | lucide-react |
| Font | `@fontsource/be-vietnam-pro`, chỉ import 400 và 600 |
| Test FE | Vitest, React Testing Library, `@testing-library/jest-dom`, `@testing-library/user-event`, jsdom, MSW; E2E Playwright |
| Lint FE | ESLint (typescript-eslint, eslint-plugin-react-hooks, eslint-plugin-react-refresh), Prettier |
| Biểu đồ | Tự dựng bằng SVG/div trong `shared/ui/charts`, không thêm thư viện |
| Hạ tầng | Docker, docker compose; CI GitHub Actions |
| Package manager FE | npm (`package-lock.json` được commit) |
| Phiên bản thư viện | Bản ổn định mới nhất tại thời điểm cài, tra bằng Context7; khóa phiên bản qua `go.sum` / `package-lock.json` |

### Nghiệp vụ
Theo skill `worktrack-domain` (vai trò, công việc cha/con, WIP, Log time, nghiệm thu, công thức báo cáo) và `docs/design/design-system.md`.

### Quy trình
- Thứ tự nhánh, quy tắc nhánh Kanban: `CLAUDE.md` §7b.
- Đặt tên: `CONTRIBUTING.md` §10.

## B. Mặc định tạm (dùng ngay, người dùng có thể đổi)
| # | Chủ đề | Mặc định | Ảnh hưởng nhánh |
|---|---|---|---|
| B1 | Quy đổi Story Point → giờ | **1 SP = 4 giờ**, là tham số cấu hình hệ thống (Admin sửa được), không hard-code | tasks, reports |
| B2 | Quyền CEO | CEO có **toàn bộ quyền của Admin** + xem mọi dự án; riêng Dashboard toàn công ty chỉ xem | auth, mọi nhánh |
| B3 | Tên gói tầng ứng dụng | `internal/application` | scaffold |
| B4 | JWT | Access 15 phút, Refresh 7 ngày, HS256, xoay vòng refresh token mỗi lần làm mới; Access trong bộ nhớ React; Refresh trong cookie `HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth` | auth |
| B5 | Ngưỡng màu giờ | Vượt `< 20%` vàng, `≥ 20%` đỏ | tasks, reports |
| B6 | Mã lỗi | API trả mã chữ (`TASK_WIP_LIMIT`) + thông điệp tiếng Việt; UI hiện thông điệp, mã chữ đặt trong phần "Chi tiết" để báo lỗi | mọi nhánh |
| B7 | Kích hoạt tài khoản | Token kích hoạt hiệu lực 72 giờ, lưu dạng hash; môi trường dev gửi email qua Mailpit (container trong docker compose) | auth |
| B8 | Mật khẩu | Tối thiểu 8 ký tự, có chữ và số; khóa đăng nhập 15 phút sau 5 lần sai liên tiếp | auth |
| B9 | Phân trang | `?page=1&pageSize=20`, tối đa 100; trả `{items, page, pageSize, total}` | mọi danh sách |
| B10 | Múi giờ | Lưu UTC, hiển thị `Asia/Ho_Chi_Minh` | mọi nhánh |
| B11 | Dữ liệu mẫu | `make seed` tạo 1 tài khoản mỗi vai trò + 2 dự án mẫu, tên người Việt giả; mật khẩu dev in ra màn hình, không commit | auth trở đi |

## C. Giả định Claude tự chọn khi chạy
| Ngày | Nhánh | Giả định | Lý do | Người dùng duyệt |
|---|---|---|---|---|
