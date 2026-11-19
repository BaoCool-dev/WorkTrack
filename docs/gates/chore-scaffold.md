# Gate: chore/scaffold — khung dự án (không có CF/UC nghiệp vụ)
**Ngày:** 2026-11-19 · **Commit cuối:** (commit scaffold đầu tiên trên nhánh) · **Kết luận:** ĐẠT (kèm ghi chú — xem "Còn tồn đọng")

Phạm vi: khung backend Go (Clean Architecture, chi, pgxpool, slog JSON + request ID, graceful shutdown, `/healthz`, `/readyz`),
khung frontend (Vite + React 19 + TS strict + Tailwind v4, theme theo `docs/design/design-system.md` §1, React Router + TanStack Query,
trang tạm hiển thị `/healthz`), `docker-compose.yml`, CI GitHub Actions. Không có tính năng nghiệp vụ.

## Tự động
| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint v2.15.0 + eslint + `tsc -b`) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: `cmd/api`, `transport/httpapi` pass · FE: 2 test pass |
| `go vet ./...` và `go vet -tags=integration ./...` | ✅ 0 cảnh báo |
| `npm run build`, `prettier --check` | ✅ |
| Coverage `internal/transport/httpapi` | ✅ 86,7% |
| `make cover` (gói cốt lõi domain/application) | ⚪ 0,0% — N/A: hai gói chưa có code ở nhánh scaffold |
| `gitleaks detect --no-banner` | ✅ no leaks found |
| `python3 scripts/ai-log/ai_log.py check` | ✅ OK (cảnh báo ở 3 entry cũ, không thuộc nhánh này) |
| `make test-integration` (testcontainers `/readyz` ↔ PostgreSQL thật) | ✅ pass trong CI (job Backend, PR #1); chưa chạy local (máy chưa có Docker) |
| `make dev` / `docker build` | ⏳ chưa chạy local (máy chưa có Docker) — job `docker` trong CI build cả 2 image |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Xác thực / phân quyền / IDOR | — | N/A | Nhánh chưa có endpoint cần đăng nhập (chỉ `/healthz`, `/readyz` công khai) |
| `/readyz` khi CSDL lỗi | Ping lỗi → 503, body `{"error":{"code","message"}}` tiếng Việt, không lộ chi tiết lỗi pgx | ✅ | `TestReadyz_DBDown_Returns503`, `TestReadyz_DBUp_ReturnsReady` |
| CORS | Origin được phép nhận header; origin lạ không nhận (preflight chưa có test riêng) | ✅ | `TestCORS_AllowedOrigin_SetsHeaders`, `TestCORS_OtherOrigin_NoHeaders` |
| Request ID + log | Response có `X-Request-Id`; log slog JSON gắn `request_id`; log truy cập chỉ ghi method/path/status/thời gian, không ghi header | ✅ | `TestHealthz_ReturnsOK` + đọc `middleware.go` |
| Cấu hình | Thiếu `DATABASE_URL` → lỗi khởi động rõ ràng; không secret cứng trong code | ✅ | `TestLoadConfig_MissingDatabaseURL_ReturnsError`, gitleaks |
| Container | API chạy user không phải root (uid 10001); `.dockerignore` loại `.env*` | ✅ | `backend/Dockerfile`, `*/.dockerignore` |
| CI | `permissions: contents: read`, không bước deploy, không push image | ✅ | `.github/workflows/ci.yml` |
| UX trang tạm | Đang tải (`role=status`), lỗi + nút Thử lại (`role=alert`), thành công | ✅ | `features/health/components/HealthStatus.test.tsx` |
| Validate input / artboard / agent `ui-reviewer` | — | N/A | Chưa có form hay màn hình nghiệp vụ |

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | `make lint` gọi `npx tsc --noEmit` trên tsconfig gốc chỉ có `references` → không kiểm tra file nào (luôn xanh giả) | Trung bình | commit scaffold (đổi sang `npm run typecheck` = `tsc -b`) |
| 2 | ESLint `no-misused-spread` khi spread `init.headers` trong `shared/api/http.ts` (header dạng `Headers` sẽ bị mất) | Thấp | commit scaffold (dùng `new Headers(init?.headers)`) |
| 3 | TypeScript 7 không tương thích peer của typescript-eslint (`<6.1`) | Thấp | commit scaffold (ghim `typescript ~6.0.3`) |
| 4 | Job CI gitleaks thiếu quyền `pull-requests: read` → đỏ trên PR | Trung bình | commit `fix(ci)` sau scaffold |

## Còn tồn đọng
- Chưa chạy `make dev`, `docker build`, test tích hợp trên máy local (chưa cài Docker) → người dùng cài Docker rồi chạy `make dev`, mở http://localhost:5173.
- `.env.example` thiếu `CORS_ORIGIN=http://localhost:5173` (Claude không được sửa `.env*`) → người dùng tự thêm.
- `make sqlc` báo "no queries" vì `db/queries/` còn trống — bình thường cho tới `feature/auth`.
- `make cover` 0% vì domain/application chưa có code — ngưỡng 70% áp dụng từ `feature/auth`.
- Nhật ký AI: entry 02, 03 và 04 (scaffold) chưa điền "Phần tôi đã sửa / hoàn thiện" → người dùng review rồi commit riêng với trailer `Human-Edit: yes` (mẫu hai commit).
- Giả định Claude tự chọn: `docs/decisions.md` mục C (5 dòng, nhánh chore/scaffold) → người dùng duyệt khi review PR.
