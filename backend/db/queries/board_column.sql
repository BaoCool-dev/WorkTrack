-- name: ListBoardColumns :many
-- CF-03, CF-06: task_count đếm cả việc cha và con nằm trong cột (cột còn việc không xóa được).
SELECT c.id, c.project_id, c.name, c.position, c.status_group,
       (SELECT count(*) FROM task t WHERE t.column_id = c.id AND t.project_id = c.project_id) AS task_count
FROM board_column c
WHERE c.project_id = @project_id
ORDER BY c.position;

-- name: CreateBoardColumn :one
INSERT INTO board_column (project_id, name, position, status_group)
VALUES (@project_id, @name, @position, @status_group)
RETURNING id;

-- name: UpdateBoardColumn :execrows
UPDATE board_column
SET name = @name, status_group = @status_group, updated_at = now()
WHERE id = @id AND project_id = @project_id;

-- name: SetBoardColumnPosition :execrows
-- Ràng buộc vị trí là DEFERRABLE nên đổi chỗ trong cùng transaction không vướng trùng tạm thời.
UPDATE board_column
SET position = @position, updated_at = now()
WHERE id = @id AND project_id = @project_id;

-- name: DeleteBoardColumn :execrows
DELETE FROM board_column WHERE id = @id AND project_id = @project_id;

-- name: ListWorkflowTransitions :many
SELECT project_id, from_column_id, to_column_id, role
FROM workflow_transition
WHERE project_id = @project_id
ORDER BY from_column_id, to_column_id, role;

-- name: CreateWorkflowTransition :exec
INSERT INTO workflow_transition (project_id, from_column_id, to_column_id, role)
VALUES (@project_id, @from_column_id, @to_column_id, @role)
ON CONFLICT DO NOTHING;

-- name: DeleteWorkflowTransitions :exec
DELETE FROM workflow_transition WHERE project_id = @project_id;
