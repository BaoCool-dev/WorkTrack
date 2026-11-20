-- name: ListDepartments :many
-- CF-02: toàn bộ cây (dữ liệu nhỏ), kèm số nhân sự trực tiếp của từng phòng ban.
SELECT d.id, d.name, d.parent_id,
       (SELECT count(*) FROM app_user u WHERE u.department_id = d.id) AS member_count
FROM department d
ORDER BY lower(d.name), d.id;

-- name: CreateDepartment :one
INSERT INTO department (name, parent_id)
VALUES (@name, sqlc.narg(parent_id))
RETURNING id;

-- name: UpdateDepartment :execrows
UPDATE department
SET name = @name, parent_id = sqlc.narg(parent_id), updated_at = now()
WHERE id = @id;

-- name: DeleteDepartment :execrows
DELETE FROM department WHERE id = @id;

-- name: UpsertSeedDepartment :one
-- Chỉ dùng cho make seed.
INSERT INTO department (name, parent_id)
VALUES (@name, sqlc.narg(parent_id))
ON CONFLICT (COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(name))
DO UPDATE SET updated_at = now()
RETURNING id;
