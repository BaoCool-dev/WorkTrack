-- name: ListJobTitles :many
SELECT j.id, j.name,
       (SELECT count(*) FROM app_user u WHERE u.job_title_id = j.id) AS member_count
FROM job_title j
ORDER BY lower(j.name), j.id;

-- name: CreateJobTitle :one
INSERT INTO job_title (name) VALUES (@name) RETURNING id;

-- name: UpdateJobTitle :execrows
UPDATE job_title SET name = @name WHERE id = @id;

-- name: DeleteJobTitle :execrows
DELETE FROM job_title WHERE id = @id;

-- name: UpsertSeedJobTitle :one
-- Chỉ dùng cho make seed.
INSERT INTO job_title (name) VALUES (@name)
ON CONFLICT (lower(name)) DO UPDATE SET name = EXCLUDED.name
RETURNING id;
