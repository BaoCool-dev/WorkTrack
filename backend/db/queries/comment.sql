-- CF-07 — bình luận gắn công việc.

-- name: ListTaskComments :many
-- Bình luận cũ trước, mới sau (đọc như hội thoại).
SELECT c.id, c.task_id, c.author_id, u.full_name AS author_name, c.body, c.created_at
FROM task_comment c
JOIN app_user u ON u.id = c.author_id
WHERE c.task_id = @task_id
ORDER BY c.created_at, c.id
LIMIT @page_limit OFFSET @page_offset;

-- name: CountTaskComments :one
SELECT count(*) FROM task_comment WHERE task_id = @task_id;

-- name: CreateTaskComment :one
INSERT INTO task_comment (task_id, author_id, body)
VALUES (@task_id, @author_id, @body)
RETURNING id, created_at;

-- name: GetTaskComment :one
SELECT c.id, c.task_id, c.author_id, u.full_name AS author_name, c.body, c.created_at
FROM task_comment c
JOIN app_user u ON u.id = c.author_id
WHERE c.id = @id;
