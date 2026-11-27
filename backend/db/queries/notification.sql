-- CF-07 — thông báo trong hệ thống (chuông).

-- name: InsertNotifications :exec
-- Một thông báo cho mỗi người trong @user_ids (đã bỏ trùng, bỏ người gây ra ở tầng domain).
INSERT INTO notification (user_id, type, task_id, project_id, actor_id, message)
SELECT unnest(@user_ids::uuid[]), @type, @task_id, @project_id, sqlc.narg(actor_id), @message;

-- name: ListNotifications :many
-- Thông báo của chính @user_id, mới nhất trước; @unread_only chỉ lấy chưa đọc.
SELECT n.id, n.type, n.task_id, t.title AS task_title, n.project_id, p.name AS project_name,
       n.actor_id, COALESCE(a.full_name, '') AS actor_name, n.message, n.read_at, n.created_at
FROM notification n
JOIN task t ON t.id = n.task_id
JOIN project p ON p.id = n.project_id
LEFT JOIN app_user a ON a.id = n.actor_id
WHERE n.user_id = @user_id AND (NOT @unread_only::boolean OR n.read_at IS NULL)
ORDER BY n.created_at DESC, n.id DESC
LIMIT @page_limit OFFSET @page_offset;

-- name: CountNotifications :one
SELECT count(*) FROM notification
WHERE user_id = @user_id AND (NOT @unread_only::boolean OR read_at IS NULL);

-- name: CountUnreadNotifications :one
SELECT count(*) FROM notification WHERE user_id = @user_id AND read_at IS NULL;

-- name: MarkNotificationRead :execrows
-- Điều kiện user_id chống IDOR: thông báo của người khác → 0 dòng → 404.
UPDATE notification SET read_at = COALESCE(read_at, now())
WHERE id = @id AND user_id = @user_id;

-- name: MarkAllNotificationsRead :execrows
UPDATE notification SET read_at = now() WHERE user_id = @user_id AND read_at IS NULL;
