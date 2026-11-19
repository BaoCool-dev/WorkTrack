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
| B12 | Môi trường dev trên máy | **Chưa dùng Docker ở máy.** PostgreSQL 16 và Mailpit cài bằng Homebrew (`localhost:5432`, SMTP `localhost:1025`, xem thư `http://localhost:8025`), cấu hình qua `.env`. Thêm `make dev-local` (API `go run` + web `npm run dev`, không cần Docker). Test tích hợp testcontainers và `docker build` chạy trong CI; trong báo cáo gate ghi "⏳ chạy trong CI" và kiểm tra job CI xanh thay cho chạy ở máy. `make dev`/`docker-compose.yml` giữ nguyên cho khi có Docker | auth trở đi |

## C. Giả định Claude tự chọn khi chạy
| Ngày | Nhánh | Giả định | Lý do | Người dùng duyệt |
|---|---|---|---|---|
| 2026-11-19 | chore/scaffold | Ghim `typescript ~6.0.3` thay vì bản 7.x mới nhất | typescript-eslint chỉ hỗ trợ peer `<6.1`; giữ ESLint theo mục A | |
| 2026-11-19 | chore/scaffold | Tra phiên bản qua registry (npm, proxy.golang.org, Docker Hub) và tài liệu chính thức thay cho Context7 | Phiên làm việc không có Context7 MCP; người dùng đã đồng ý trong phiên | |
| 2026-11-19 | chore/scaffold | Thêm biến môi trường `CORS_ORIGIN` (mặc định `http://localhost:5173`, một origin) | FE (5173) gọi API (8080) khác origin; Claude không sửa được `.env.example` → người dùng tự thêm dòng này | |
| 2026-11-19 | chore/scaffold | `make lint` gọi `npm run typecheck` (`tsc -b`) thay cho `tsc --noEmit` | tsconfig gốc chỉ có `references`, `tsc --noEmit` không kiểm tra file nào | |
| 2026-11-19 | chore/scaffold | Test tích hợp testcontainers và docker build chỉ chạy trong CI | Máy dev chưa cài Docker | |
| 2026-11-19 | feature/auth | Giới hạn tần suất bằng token bucket trong bộ nhớ, khóa theo IP lấy từ `RemoteAddr` (không tin `X-Forwarded-For`): đăng nhập/kích hoạt 10 lượt, hồi 1 lượt/6 giây; làm mới 30 lượt, hồi 1 lượt/2 giây; vượt → 429 `RATE_LIMITED` + `Retry-After` | Một tiến trình API, không cần Redis; tránh giả mạo IP qua header | |
| 2026-11-19 | feature/auth | Kiểm tra token kích hoạt bằng `POST /auth/activation/verify` (token trong body) thay vì GET có query | Không để token lọt vào URL, log truy cập, lịch sử trình duyệt | |
| 2026-11-19 | feature/auth | Cookie refresh luôn `Secure`, không có cờ tắt cho dev | Chrome/Firefox coi `http://localhost` là ngữ cảnh an toàn; giữ nguyên B4 | |
| 2026-11-19 | feature/auth | Biến môi trường mới: `JWT_SECRET` (bắt buộc, ≥ 32 byte), `ACCESS_TOKEN_TTL` (mặc định 15m), `APP_BASE_URL`, `SMTP_HOST/PORT/FROM/USERNAME/PASSWORD` (mặc định Mailpit `localhost:1025`) | Claude không sửa được `.env.example`/`.env` → người dùng tự thêm (sinh secret: `openssl rand -hex 32`) | |
| 2026-11-19 | feature/auth | `make seed` chỉ tạo 4 tài khoản; 2 dự án mẫu của B11 dời sang nhánh feature/projects | Bảng dự án chưa tồn tại ở CF-01 | |
| 2026-11-19 | feature/auth | Mật khẩu seed sinh ngẫu nhiên mỗi lần chạy (chạy lại = đặt lại mật khẩu) | Không có mật khẩu cố định nào nằm trong repo | |
| 2026-11-19 | feature/auth | Thêm `react-hook-form` 7.92 + `zod` 4.6 + `@hookform/resolvers` 7.0 (MIT) cho form đăng nhập/kích hoạt; `lucide-react` 1.76 (ISC) cho icon; `msw` 3.0 (MIT, dev) giả lập API trong test | Đều nằm trong danh sách đề xuất của `frontend/CLAUDE.md`; ISC tương đương MIT | |
| 2026-11-19 | feature/auth | Chưa cài `zustand` | CF-01 không có UI state dùng chung; phiên đăng nhập là store nhỏ tự viết dùng `useSyncExternalStore` | |
