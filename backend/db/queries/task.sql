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
-- WIP: số công việc cha nhóm in_progress của một người trên mọi dự án (CLAUDE.md §5.6). Chỉ để hiển thị;
-- việc chặn dùng CountProjectWIP (giới hạn tính theo từng dự án).
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
    due_date = sqlc.narg(due_date), version = version + 1, updated_at = now()
WHERE id = @id;

-- name: SetTaskAssignee :execrows
UPDATE task SET assignee_id = sqlc.narg(assignee_id), version = version + 1, updated_at = now() WHERE id = @id;

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

-- CF-04, CF-05 — Kanban.

-- name: CountProjectWIP :one
-- WIP của một người trong MỘT dự án: chỉ đếm công việc cha (parent_task_id IS NULL, CLAUDE.md §5.6)
-- đang ở cột nhóm in_progress. Gọi sau khi đã khóa dòng dự án để hai lần kéo song song không cùng lọt.
SELECT count(*) FROM task t
JOIN board_column c ON c.id = t.column_id
WHERE t.project_id = @project_id AND t.assignee_id = @assignee_id::uuid
  AND t.parent_task_id IS NULL AND c.status_group = 'in_progress';

-- name: ListBoardTasks :many
-- Toàn bộ công việc cha của bảng Kanban (lọc nhãn/người nhận làm ở client). Giới hạn an toàn @max_rows.
SELECT * FROM task_view v
WHERE v.project_id = @project_id AND v.parent_task_id IS NULL
ORDER BY v.column_position, v.position, v.created_at, v.id
LIMIT @max_rows;

-- name: ListColumnTaskIDs :many
-- Thứ tự hiện tại các công việc cha trong một cột (để chèn thẻ vào vị trí mới).
SELECT id FROM task
WHERE project_id = @project_id AND column_id = @column_id AND parent_task_id IS NULL
ORDER BY position, created_at, id;

-- name: SetTaskPositions :exec
-- Đánh lại vị trí 0..n-1 theo thứ tự mảng @ids (một câu lệnh, không vòng lặp).
UPDATE task SET position = x.ord - 1
FROM unnest(@ids::uuid[]) WITH ORDINALITY AS x(id, ord)
WHERE task.id = x.id;

-- name: MoveTaskColumn :execrows
-- Đổi cột và tăng version. Điều kiện version = @version là chốt chặn cuối chống ghi đè.
UPDATE task SET column_id = @column_id, version = version + 1, updated_at = now()
WHERE id = @id AND version = @version;

-- name: InsertTaskMoveHistory :exec
INSERT INTO task_move_history (task_id, project_id, from_column_id, to_column_id, moved_by)
VALUES (@task_id, @project_id, @from_column_id, @to_column_id, @moved_by);

-- name: GetColumnGroup :one
-- Nhóm trạng thái của một cột trong dự án (cột khác dự án → không có dòng).
SELECT status_group FROM board_column WHERE project_id = @project_id AND id = @id;

-- name: TransitionAllowed :one
SELECT EXISTS (
    SELECT 1 FROM workflow_transition
    WHERE project_id = @project_id AND from_column_id = @from_column_id
      AND to_column_id = @to_column_id AND role = @role
);
