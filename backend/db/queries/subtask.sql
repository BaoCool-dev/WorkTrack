-- CF-07 — công việc con 1 cấp (không Story Point, không qua workflow_transition).

-- name: ListSubtasks :many
-- Con của một công việc cha, theo thứ tự tạo. Mỗi cha có ít việc con nên không phân trang.
SELECT * FROM task_view v
WHERE v.parent_task_id = @parent_task_id
ORDER BY v.created_at, v.id;

-- name: CreateSubtask :one
-- Con kế thừa dự án và người nhận của cha; story_point luôn NULL (ck_task_story_point_parent_only).
INSERT INTO task (project_id, column_id, parent_task_id, title, description, assignee_id, position, created_by)
VALUES (@project_id, @column_id, @parent_task_id, @title, @description, sqlc.narg(assignee_id),
        (SELECT COALESCE(max(t.position) + 1, 0) FROM task t WHERE t.parent_task_id = @parent_task_id),
        @created_by)
RETURNING id;

-- name: MoveSubtask :execrows
-- Đổi cột của con. Cột khác dự án → vi phạm fk_task_column.
UPDATE task SET column_id = @column_id, version = version + 1, updated_at = now()
WHERE id = @id AND parent_task_id IS NOT NULL;

-- name: DeleteSubtask :execrows
DELETE FROM task WHERE id = @id AND parent_task_id IS NOT NULL;
