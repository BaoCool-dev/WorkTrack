# backend/CLAUDE.md — WorkTrack API (Go)

Đọc thêm skill `go-backend` (chi tiết mẫu code) và `db-migration-sqlc` (CSDL).

## Stack
Go · chi · pgx v5 (pgxpool) · sqlc · goose · golang-jwt v5 · validator v10 · coder/websocket · log/slog

## Cấu trúc (Clean Architecture — phụ thuộc chỉ hướng vào trong)

```
cmd/api/main.go                 wiring: config → pool → repo → use case → router. Không logic nghiệp vụ.
internal/
  domain/                       entity, value object, lỗi nghiệp vụ, quy tắc thuần (không I/O, không import pgx/chi)
  application/                  use case + port (interface) + authz. Chỉ import domain.
    authz/                      MỘT chỗ duy nhất quyết định "ai được làm gì"
  infrastructure/
    postgres/                   repository cài đặt port; sqlcgen/ là code sinh (KHÔNG SỬA TAY)
    gemini/  email/  realtime/  adapter ra ngoài
  transport/httpapi/               handler chi, middleware, DTO request/response, ánh xạ lỗi → HTTP
db/migrations/                  goose (*.sql)
db/queries/                     sqlc (*.sql)
```

Hướng phụ thuộc: `transport → application → domain` · `infrastructure → application (port) + domain`.
`domain` và `application` **không bao giờ** import `pgx`, `chi`, `net/http`, `sqlcgen`.

## Quy tắc cứng
- Mọi hàm I/O nhận `ctx context.Context` là tham số đầu.
- Lỗi: bọc bằng `fmt.Errorf("…: %w", err)`; lỗi nghiệp vụ là giá trị trong `domain` (`ErrWIPLimit`, …), so sánh bằng `errors.Is/As`. Không `panic` cho luồng nghiệp vụ.
- Ánh xạ lỗi → HTTP status + mã lỗi ở **một** file `transport/httpapi/errors.go`.
- Không biến toàn cục cho dependency; inject qua struct constructor `NewXxx(deps…)`.
- Interface khai báo ở nơi **dùng** (application), không ở nơi cài đặt.
- Transaction: use case nhận `TxManager` port; repository không tự mở transaction.
- Không ORM, không reflection-heavy lib, không framework web khác chi.
- `sqlcgen/` và migration đã merge: **không sửa tay**.

## Lệnh
```bash
go test ./...                       # unit + api
go test -tags=integration ./...     # cần Docker (testcontainers)
golangci-lint run
sqlc generate                       # hoặc make sqlc
goose -dir db/migrations postgres "$DATABASE_URL" up
```
