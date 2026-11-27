-- name: ListProjectTasks :many
-- CF-06, B9: công việc cha của một dự án, sắp theo thứ tự cột rồi vị trí trong cột.
-- @search là mẫu ILIKE đã thoát ký tự ở Go; @unassigned = true chỉ lấy việc chưa có người nhận.
SELECT * FROM task_view v
WHERE v.project_id = @project_id
  AND v.parent_task_id IS NULL
  AND (sqlc.narg(search)::text IS NULL OR v.title ILIKE sqlc.narg(search)::text ESCAPE '\')
  AND (sqlc.narg(column_id)::uuid IS NULL OR v.column_id = sqlc.narg(column_id)::uuid)
  AND (sqlc.narg(status_group)::text IS NULL OR v.status_group = sqlc.narg(status_group)::text)
  AND (sqlc.narg(assignee_id)::uuid IS NULL OR v.assignee_id = sqlc.narg(assignee_id)::uuid)
  AND (NOT @unassigned::boolean OR v.assignee_id IS NULL)
  AND (sqlc.narg(label_id)::uuid IS NULL
       OR EXISTS (SELECT 1 FROM task_label tl WHERE tl.task_id = v.id AND tl.label_id = sqlc.narg(label_id)::uuid))
ORDER BY v.column_position, v.position, v.created_at, v.id
LIMIT @page_limit OFFSET @page_offset;

-- name: CountProjectTasks :one
SELECT count(*) FROM task_view v
WHERE v.project_id = @project_id
  AND v.parent_task_id IS NULL
  AND (sqlc.narg(search)::text IS NULL OR v.title ILIKE sqlc.narg(search)::text ESCAPE '\')
  AND (sqlc.narg(column_id)::uuid IS NULL OR v.column_id = sqlc.narg(column_id)::uuid)
  AND (sqlc.narg(status_group)::text IS NULL OR v.status_group = sqlc.narg(status_group)::text)
  AND (sqlc.narg(assignee_id)::uuid IS NULL OR v.assignee_id = sqlc.narg(assignee_id)::uuid)
  AND (NOT @unassigned::boolean OR v.assignee_id IS NULL)
  AND (sqlc.narg(label_id)::uuid IS NULL
       OR EXISTS (SELECT 1 FROM task_label tl WHERE tl.task_id = v.id AND tl.label_id = sqlc.narg(label_id)::uuid));

-- name: ListMyTasks :many
-- "Công việc của tôi": việc cha giao cho @user_id ở các dự án người đó còn là thành viên.
-- Việc chưa xong trước, hạn gần trước, không có hạn xếp cuối.
SELECT * FROM task_view v
WHERE v.assignee_id = @user_id::uuid
  AND v.parent_task_id IS NULL
  AND EXISTS (SELECT 1 FROM project_member pm WHERE pm.project_id = v.project_id AND pm.user_id = @user_id)
  AND (sqlc.narg(status_group)::text IS NULL OR v.status_group = sqlc.narg(status_group)::text)
  AND (sqlc.narg(search)::text IS NULL OR v.title ILIKE sqlc.narg(search)::text ESCAPE '\')
ORDER BY (v.status_group = 'done'), v.due_date NULLS LAST, v.created_at DESC, v.id
LIMIT @page_limit OFFSET @page_offset;

-- name: CountMyTasks :one
SELECT count(*) FROM task_view v
WHERE v.assignee_id = @user_id::uuid
  AND v.parent_task_id IS NULL
  AND EXISTS (SELECT 1 FROM project_member pm WHERE pm.project_id = v.project_id AND pm.user_id = @user_id)
  AND (sqlc.narg(status_group)::text IS NULL OR v.status_group = sqlc.narg(status_group)::text)
  AND (sqlc.narg(search)::text IS NULL OR v.title ILIKE sqlc.narg(search)::text ESCAPE '\');

-- name: CountWIP :one
-- WIP: số công việc cha nhóm in_progress của một người (CLAUDE.md §5.6). Chỉ hiển thị, không chặn.
SELECT count(*) FROM task t
JOIN board_column c ON c.id = t.column_id
JOIN project_member pm ON pm.project_id = t.project_id AND pm.user_id = t.assignee_id
WHERE t.assignee_id = @user_id::uuid AND t.parent_task_id IS NULL AND c.status_group = 'in_progress';

-- name: GetTask :one
SELECT * FROM task_view v WHERE v.id = @id;

-- name: LockTask :one
-- Khóa dòng công việc tới hết transaction (nhận việc / sửa chạy tuần tự).
SELECT id FROM task WHERE id = @id FOR UPDATE;

-- name: FirstTodoColumn :one
-- Công việc mới nằm ở cột đầu tiên thuộc nhóm todo.
SELECT id FROM board_column
WHERE project_id = @project_id AND status_group = 'todo'
ORDER BY position
LIMIT 1;

-- name: CreateTask :one
INSERT INTO task (project_id, column_id, title, description, story_point, estimated_minutes,
                  assignee_id, due_date, position, created_by)
VALUES (@project_id, @column_id, @title, @description, sqlc.narg(story_point), sqlc.narg(estimated_minutes),
        sqlc.narg(assignee_id), sqlc.narg(due_date),
        (SELECT COALESCE(max(t.position) + 1, 0) FROM task t WHERE t.project_id = @project_id AND t.column_id = @column_id),
        @created_by)
RETURNING id;

-- name: UpdateTask :execrows
UPDATE task
SET title = @title, description = @description, story_point = sqlc.narg(story_point),
    estimated_minutes = sqlc.narg(estimated_minutes), assignee_id = sqlc.narg(assignee_id),
    due_date = sqlc.narg(due_date), updated_at = now()
WHERE id = @id;

-- name: SetTaskAssignee :execrows
UPDATE task SET assignee_id = sqlc.narg(assignee_id), updated_at = now() WHERE id = @id;

-- name: GetAssignableMember :one
-- Vai trò và trạng thái của một thành viên dự án (kiểm người nhận việc / người tự nhận).
SELECT r.code AS role_code, u.status
FROM project_member pm
JOIN app_user u ON u.id = pm.user_id
JOIN role r ON r.id = u.role_id
WHERE pm.project_id = @project_id AND pm.user_id = @user_id;

-- name: ListTaskLabels :many
SELECT tl.task_id, l.id, l.name
FROM task_label tl
JOIN label l ON l.id = tl.label_id
WHERE tl.task_id = ANY(@task_ids::uuid[])
ORDER BY lower(l.name), l.id;

-- name: DeleteTaskLabels :exec
DELETE FROM task_label WHERE task_id = @task_id;

-- name: AddTaskLabels :exec
INSERT INTO task_label (task_id, label_id)
SELECT @task_id, unnest(@label_ids::uuid[])
ON CONFLICT DO NOTHING;

-- name: GetSeedTaskIDByTitle :one
-- Chỉ dùng cho make seed (idempotent theo tên trong dự án).
SELECT id FROM task WHERE project_id = @project_id AND title = @title AND parent_task_id IS NULL LIMIT 1;
