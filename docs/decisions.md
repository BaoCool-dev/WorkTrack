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
| 2026-11-19 | feature/auth | `shared/ui` ở nhánh này chỉ dựng component CF-01 dùng tới (Button, Input, Badge, Avatar, Toast, Dialog, EmptyState, ErrorState, Skeleton, AppShell); các component còn lại và `format.ts`/`hoursLevel.ts` dựng ở nhánh đầu tiên cần chúng | Một nhánh một tính năng; tránh code chưa dùng, chưa có màn hình để kiểm | |
| 2026-11-19 | feature/auth | `useCan(permission)` chỉ nhận mã quyền theo vai trò, chưa có tham số `resource` (quyền theo dự án) | Quyền theo dự án chưa tồn tại ở CF-01; thêm tham số khi làm feature/projects | |
| 2026-11-19 | feature/auth | Menu + route guard dùng chung một bảng `app/layout/menu.ts`; các mục chưa có màn hình (Dự án, Công việc của tôi, Báo cáo, Dashboard, Nhân sự) trỏ tới trang giữ chỗ | Sidebar và guard không lệch nhau; route có sẵn để kiểm phân quyền UI ngay từ CF-01 | |
| 2026-11-19 | feature/auth | Khối "Giờ của tôi tuần này" ở đáy sidebar dời sang feature/logtime (CF-08) | Chưa có dữ liệu Log time | |
| 2026-11-19 | feature/auth | Trang kích hoạt đọc token từ URL rồi xóa ngay khỏi thanh địa chỉ (`replace`) | Không để token ở lịch sử trình duyệt / Referer | |
| 2026-11-19 | feature/auth | Làm mới token lỗi mạng khi đang đăng nhập → giữ phiên, thử lại ở lần sau; lỗi khác (401…) → về trạng thái chưa đăng nhập | Mất mạng thoáng qua không đá người dùng ra | |
| 2026-11-19 | feature/auth | Thư mục `docs/design/artboards/` chưa có trong repo → app shell, đăng nhập, kích hoạt dựng theo `design-system.md` §4–§8 | Người dùng cần xuất artboard để soát lại | |
| 2026-11-20 | feature/auth | Xoay vòng refresh token (thu hồi cũ + chèn mới) chưa bọc transaction `TxManager`; lỗi giữa hai bước → người dùng phải đăng nhập lại | Hậu quả chỉ là mất phiên, không lộ quyền; thêm `TxManager` ở nhánh đầu tiên cần transaction nhiều bảng | |
| 2026-11-20 | feature/auth | Chưa có thời gian ân hạn khi nhiều tab cùng làm mới token (tab chậm dùng lại token cũ → thu hồi cả họ, phải đăng nhập lại) | Ưu tiên phát hiện dùng lại token (B4); desktop nội bộ ít mở nhiều tab | |
| 2026-11-20 | feature/auth | Hoãn: bẫy focus trong `Dialog`, `Button` có prop `disabledReason`, nút "Hoàn tác" trên Toast | CF-01 chưa có hộp thoại/nút vô hiệu/thao tác đảo được; làm ở nhánh đầu tiên dùng | |
| 2026-11-20 | feature/auth | SMTP chưa bắt buộc TLS; chưa có job dọn `refresh_token` hết hạn; mã 423 (khóa) cho phép đoán email có tồn tại; bộ giới hạn tần suất chưa gom IPv6 theo /64 và chưa dọn bucket cũ | Môi trường dev dùng Mailpit; quy mô SME một tiến trình; 423 là yêu cầu UX của B8. Ghi lại để xử lý trước khi triển khai thật | |
| 2026-11-20 | feature/auth | Màn tạo tài khoản (Admin) thuộc CF-02 → ở CF-01 kiểm luồng tạo tài khoản qua API, UI kiểm từ bước email kích hoạt trở đi | Một nhánh một tính năng | |
| 2026-11-20 | feature/org | Email chỉ đọc sau khi tạo tài khoản: `PUT /users/{id}` từ chối trường `email` (400) | Email là định danh đăng nhập và đích của link kích hoạt; đổi email cần luồng xác minh riêng (ngoài phạm vi Lõi) | |
| 2026-11-20 | feature/org | Đổi cha phòng ban: kiểm vòng lặp trên ảnh chụp cây đọc trong use case, chưa khóa bảng; hai Admin đổi cha chéo nhau cùng lúc có thể tạo vòng | Chưa có `TxManager`; SME ít Admin, thao tác hiếm. Thêm khóa/transaction khi có `TxManager` | |
| 2026-11-20 | feature/org | Lọc nhân sự theo phòng ban gồm cả phòng ban con (CTE đệ quy) | Phù hợp kỳ vọng "xem cả khối" khi chọn phòng ban cha | |
| 2026-11-20 | feature/org | Chỉ CEO sửa được tài khoản CEO; không ai tự đổi vai trò của chính mình (422 `USER_CANNOT_CHANGE_OWN_ROLE`) | Tránh Admin tự hạ/nâng quyền hoặc sửa tài khoản cấp cao hơn | |
| 2026-11-20 | feature/org | Endpoint đọc phòng ban/chức danh/kỹ năng yêu cầu quyền `ManageOrg` (Admin, CEO); Manager/Member chưa đọc được | CF-02 chỉ có màn Admin; mở quyền đọc khi nhánh sau cần (vd. gợi ý nhân sự CF-12) | |
| 2026-11-20 | feature/org | Hồ sơ kỹ năng tối đa 50 kỹ năng/người (422 `SKILL_TOO_MANY`); `PUT /users/{id}/skills` thay toàn bộ | Chặn payload quá lớn; thay toàn bộ đơn giản hơn PATCH từng dòng | |
| 2026-11-20 | feature/org | Thông báo lỗi trùng email đổi thành "Email đã tồn tại" (mã `EMAIL_TAKEN` giữ nguyên) | Theo đúng câu chữ trong yêu cầu CF-02 | |
| 2026-11-20 | feature/org | `docs/design/artboards/` vẫn chưa có → màn Nhân sự & phòng ban dựng theo `design-system.md` | Người dùng cần xuất artboard 10 để soát lại | |
| 2026-11-20 | feature/org | Xóa phòng ban/chức danh/kỹ năng bị chặn khi còn phòng ban con hoặc còn người dùng tham chiếu (409) thay vì xóa lan | Không làm mất dữ liệu nhân sự ngoài ý muốn | |
| 2026-11-20 | feature/org | Màn Nhân sự: tab lưu ở `?tab=`; ô tìm kiếm debounce 300 ms; bảng hiện tối đa 3 kỹ năng rồi `+n` | Quay lại đúng tab khi tải lại trang; giảm số request khi gõ; giữ bảng gọn ở 1366px | |
| 2026-11-20 | feature/projects | `docs/design/artboards/` vẫn chưa có → màn Dự án và Chi tiết dự án dựng theo `design-system.md` | Người dùng cần xuất artboard tương ứng để soát lại | |
| 2026-11-20 | feature/projects | Admin không xem dự án (danh sách 403, chi tiết 404); Member chỉ thấy dự án mình là thành viên; CEO thấy tất cả, chỉ xem | Admin không tạo/tham gia dự án (CLAUDE.md §5.3); CEO chỉ xem (§5.2) | |
| 2026-11-20 | feature/projects | Người ngoài dự án xem/sửa dự án nhận 404 `PROJECT_NOT_FOUND` thay vì 403 | Không lộ sự tồn tại của dự án | |
| 2026-11-20 | feature/projects | Manager phụ trách cố định là người tạo dự án; chỉ người này sửa cấu hình (chưa có chức năng đổi Manager) | CF-03 không yêu cầu bàn giao dự án; giữ phân quyền đơn giản | |
| 2026-11-20 | feature/projects | Dự án mới có sẵn 5 cột (Backlog, To Do, In Progress, Chờ nghiệm thu, Done) + luật chuyển mặc định, tạo trong 1 transaction qua port `Transactor` | Đúng yêu cầu "luật mặc định"; port giữ application không import pgx | |
| 2026-11-20 | feature/projects | `taskCount` của cột luôn 0 cho tới khi có bảng công việc (CF-06); xóa cột có việc trả 409 `COLUMN_HAS_TASKS` | Kiểm tra sẵn để CF-06 chỉ cần nối truy vấn đếm | |
| 2026-11-20 | feature/projects | Vị trí cột đánh lại liên tục 0..n-1 sau xóa; đổi thứ tự qua `PUT /projects/{id}/column-order` nhận đủ danh sách id (ràng buộc unique DEFERRABLE) | Một request, không trạng thái trung gian sai thứ tự | |
| 2026-11-20 | feature/projects | Đổi thứ tự cột bằng nút lên/xuống; luật chuyển sửa bằng ma trận ô đánh dấu theo vai trò, ô Member vào/ra nhóm done bị khóa (server cũng chặn `TRANSITION_MEMBER_TO_DONE`) | Không thêm dnd-kit ở nhánh này; kéo thả để dành feature/kanban | |
| 2026-11-20 | feature/projects | Thêm được người dùng đang "Chờ kích hoạt" vào dự án; chỉ Manager/Member, không thêm Admin/CEO | Manager chuẩn bị đội trước khi mọi người kích hoạt xong | |
| 2026-11-20 | feature/projects | Route `/api/v1/projects/*` chỉ yêu cầu đăng nhập, quyền kiểm trong service (`authz`); ngày gửi/nhận dạng `YYYY-MM-DD` | Quyền phụ thuộc thành viên dự án nên phải kiểm ở tầng application | |
| 2026-11-20 | feature/projects | Seed tạo 2 dự án mẫu (B11) cho Manager mẫu, thêm Member mẫu làm thành viên, idempotent theo tên | Có dữ liệu demo ngay sau `make seed` | |
| 2026-11-20 | feature/projects | FE: thêm quyền `projects.view_own`; badge trạng thái: Mới tạo=draft, Đang tiến hành=active, Hoàn thành=ok, Tạm dừng=warn, Đã hủy=bad; trang chi tiết là route con `/projects/:projectId` dưới cùng guard menu | Ánh xạ theo nghĩa màu trong design-system | |
| 2026-11-20 | feature/projects | Chuyển `OrgTable` thành `shared/ui/DataTable`; thêm `formatDate`/`formatTimestampDate` vào `shared/lib/format.ts` | Dùng chung giữa admin và projects; định dạng ngày chỉ qua format.ts (frontend/CLAUDE.md) | |
| 2026-11-20 | feature/tasks | Tab "Nhãn" và "Cấu hình" (hệ số B1) đặt trong trang Quản trị `/admin/users`, ghép qua prop `AdminOrgView.extraTabs` | Cùng nhóm quyền Admin/CEO; tránh feature admin import feature tasks | |
| 2026-11-20 | feature/tasks | Trang "Công việc của tôi" `/my-tasks` chỉ xem (panel chi tiết không có nút Sửa); Manager sửa công việc ở trang dự án | Giữ một nơi sửa, đủ ngữ cảnh thành viên dự án cho ô người nhận | |
| 2026-11-20 | feature/tasks | Story Point theo thang Fibonacci 1, 2, 3, 5, 8, 13, 21 (FE chọn từ danh sách; BE chặn ngoài thang bằng `TASK_STORY_POINT_INVALID`) | Thang phổ biến trong Scrum, nằm trong CHECK 1–100 của CSDL | |
| 2026-11-20 | feature/tasks | Member chỉ nhận việc cha chưa có người nhận ở cột thuộc nhóm `todo` (mọi cột todo, không riêng "Backlog"); nhận việc chỉ gán người, không đổi cột | Chuyển cột để dành feature/kanban; khóa dòng `FOR UPDATE` tránh hai người cùng nhận | |
| 2026-11-20 | feature/tasks | Giới hạn: tiêu đề ≤ 200 ký tự, mô tả ≤ 5000, tối đa 10 nhãn/việc, hệ số B1 từ 1 đến 40 giờ/SP (số nguyên) | Đủ dùng thực tế, chặn dữ liệu bất thường; khớp FE (zod) và BE (domain) | |
| 2026-11-20 | feature/tasks | Admin và CEO quản lý danh mục nhãn và hệ số B1 (quyền `org.manage`/`settings.manage`); mọi vai trò đã đăng nhập xem được | Cấu hình cấp công ty, cùng nhóm với Nhân sự & phòng ban | |
| 2026-11-20 | feature/tasks | Chip WIP ở "Công việc của tôi" chỉ hiển thị số việc cha đang làm (`WIP n`), chưa có giới hạn/chặn | Phạm vi CF-06: WIP không chặn; giới hạn và hộp thoại chặn làm ở feature/kanban | |
