-- name: ListLabels :many
-- CF-06: danh mục nhãn, kèm số công việc đang gắn nhãn.
SELECT l.id, l.name,
       (SELECT count(*) FROM task_label tl WHERE tl.label_id = l.id) AS task_count
FROM label l
ORDER BY lower(l.name), l.id;

-- name: CreateLabel :one
INSERT INTO label (name) VALUES (@name) RETURNING id;

-- name: UpdateLabel :execrows
UPDATE label SET name = @name WHERE id = @id;

-- name: DeleteLabel :execrows
DELETE FROM label WHERE id = @id;

-- name: UpsertSeedLabel :one
-- Chỉ dùng cho make seed.
INSERT INTO label (name) VALUES (@name)
ON CONFLICT (lower(name)) DO UPDATE SET name = EXCLUDED.name
RETURNING id;
