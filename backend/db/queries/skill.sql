-- name: ListSkills :many
SELECT s.id, s.name,
       (SELECT count(*) FROM user_skill us WHERE us.skill_id = s.id) AS member_count
FROM skill s
ORDER BY lower(s.name), s.id;

-- name: CreateSkill :one
INSERT INTO skill (name) VALUES (@name) RETURNING id;

-- name: UpdateSkill :execrows
UPDATE skill SET name = @name WHERE id = @id;

-- name: DeleteSkill :execrows
DELETE FROM skill WHERE id = @id;

-- name: UpsertSeedSkill :one
-- Chỉ dùng cho make seed.
INSERT INTO skill (name) VALUES (@name)
ON CONFLICT (lower(name)) DO UPDATE SET name = EXCLUDED.name
RETURNING id;
