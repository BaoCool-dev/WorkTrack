---
name: db-migration-sqlc
description: Thêm hoặc sửa bảng, cột, index, ràng buộc, truy vấn PostgreSQL cho WorkTrack bằng goose migration và sqlc (pgx/v5). Dùng khi tạo migration, viết query .sql, chạy sqlc generate, hoặc tối ưu truy vấn báo cáo.
---

# Migration (goose) + truy vấn (sqlc)

## Quy trình
1. Tạo migration: `make migrate-new name=add_time_log` → file `backend/db/migrations/<timestamp>_add_time_log.sql`.
2. Viết cả `Up` và `Down`. **Không sửa migration đã merge** — tạo migration mới để sửa.
3. Viết query vào `backend/db/queries/<bang>.sql` với annotation sqlc.
4. `make sqlc` → code sinh vào `internal/infrastructure/postgres/sqlcgen/` (không sửa tay — hook sẽ chặn).
5. Bọc trong repository (xem skill `go-backend`), test tích hợp với PostgreSQL thật.

## Mẫu migration

```sql
-- +goose Up
CREATE TABLE time_log (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id     uuid NOT NULL REFERENCES task(id) ON DELETE CASCADE,
    user_id     uuid NOT NULL REFERENCES app_user(id),
    minutes     integer NOT NULL CHECK (minutes > 0),
    note        text NOT NULL DEFAULT '',
    logged_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_time_log_task ON time_log(task_id);

-- +goose Down
DROP TABLE time_log;
```

Ràng buộc nghiệp vụ đặt ở CSDL khi có thể (CHECK, FK, UNIQUE) — đó là lớp bảo vệ thứ hai sau domain:
- `CHECK (parent_task_id IS NULL OR story_point IS NULL)` — con không có Story Point.
- Giờ lưu bằng phút `integer`, không `numeric`/float.
- Thời điểm dùng `timestamptz`.

## Mẫu truy vấn sqlc

```sql
-- name: GetTask :one
SELECT * FROM task WHERE id = $1;

-- name: CountParentInProgressByAssignee :one
-- WIP chỉ đếm công việc cha (R09, R20)
SELECT count(*) FROM task t
JOIN board_column c ON c.id = t.column_id
WHERE t.assignee_id = $1
  AND t.parent_task_id IS NULL
  AND c.status_group = 'in_progress';

-- name: InsertTimeLog :one
INSERT INTO time_log (task_id, user_id, minutes, note)
VALUES ($1, $2, $3, $4)
RETURNING *;
```

Truy vấn báo cáo: luôn lọc `parent_task_id IS NULL` cho năng suất/tuân thủ; cộng giờ con lên cha bằng `COALESCE(parent_task_id, id)`.

## sqlc.yaml (tham chiếu)

```yaml
version: "2"
sql:
  - engine: postgresql
    schema: db/migrations
    queries: db/queries
    gen:
      go:
        package: sqlcgen
        out: internal/infrastructure/postgres/sqlcgen
        sql_package: pgx/v5
        emit_json_tags: false
        emit_interface: true
        overrides:
          - db_type: uuid
            go_type: github.com/google/uuid.UUID
```

Kiểm tra cú pháp cấu hình sqlc bằng Context7 nếu báo lỗi — không đoán.

## Tối ưu
- Postgres MCP (chế độ chỉ đọc) để chạy `EXPLAIN ANALYZE` trên CSDL dev.
- Thêm index khi có truy vấn chậm đo được, ghi số đo vào mô tả PR.
