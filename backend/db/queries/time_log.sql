-- CF-08 — Log time trên công việc cha và con.

-- name: CreateTimeLog :one
INSERT INTO time_log (task_id, project_id, user_id, minutes, note, work_date)
VALUES (@task_id, @project_id, @user_id, @minutes, @note, @work_date)
RETURNING id;

-- name: GetTimeLog :one
SELECT l.id, l.task_id, t.title AS task_title, t.parent_task_id, l.project_id,
       l.user_id, u.full_name AS user_name, l.minutes, l.note, l.work_date, l.created_at
FROM time_log l
JOIN task t ON t.id = l.task_id
JOIN app_user u ON u.id = l.user_id
WHERE l.id = @id;

-- name: ListTaskTimeLogs :many
-- Lịch sử log của một công việc; với việc cha gồm cả log của các con (giờ con cộng lên cha). Mới nhất trước.
SELECT l.id, l.task_id, t.title AS task_title, t.parent_task_id, l.project_id,
       l.user_id, u.full_name AS user_name, l.minutes, l.note, l.work_date, l.created_at
FROM time_log l
JOIN task t ON t.id = l.task_id
JOIN app_user u ON u.id = l.user_id
-- IN (subquery) thay cho OR qua hai bảng để planner dùng được ix_task_parent + ix_time_log_task.
WHERE l.task_id IN (SELECT s.id FROM task s WHERE s.id = @task_id OR s.parent_task_id = @task_id)
ORDER BY l.created_at DESC, l.id DESC
LIMIT @max_rows OFFSET @skip_rows;

-- name: CountTaskTimeLogs :one
SELECT count(*) FROM time_log l
JOIN task t ON t.id = l.task_id
-- IN (subquery) thay cho OR qua hai bảng để planner dùng được ix_task_parent + ix_time_log_task.
WHERE l.task_id IN (SELECT s.id FROM task s WHERE s.id = @task_id OR s.parent_task_id = @task_id);

-- name: DeleteTimeLog :execrows
DELETE FROM time_log WHERE id = @id;

-- name: LastDoneAt :one
-- Mốc "đã Done" của công việc: lần gần nhất được chuyển TỪ cột ngoài nhóm done VÀO một cột nhóm done.
-- Đổi cột trong nội bộ nhóm done (vd. "Done" → "Đã đóng") không tính, để không mở lại cửa sổ 24 giờ.
-- Cột nguồn không còn tồn tại (đã xóa) coi như ngoài nhóm done.
-- NULL khi chưa có lịch sử (dữ liệu cũ) → tầng trên dùng updated_at.
SELECT max(h.moved_at)::timestamptz AS done_at
FROM task_move_history h
JOIN board_column c ON c.id = h.to_column_id
LEFT JOIN board_column f ON f.id = h.from_column_id
WHERE h.task_id = @task_id AND c.status_group = 'done'
  AND f.status_group IS DISTINCT FROM 'done';
