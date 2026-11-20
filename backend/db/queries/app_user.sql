-- name: CreateUser :one
INSERT INTO app_user (email, full_name, role_id, activation_token_hash, activation_expires_at, created_by,
                      department_id, job_title_id)
VALUES (
    @email,
    @full_name,
    (SELECT r.id FROM role r WHERE r.code = @role_code),
    @activation_token_hash,
    @activation_expires_at,
    sqlc.narg(created_by),
    sqlc.narg(department_id),
    sqlc.narg(job_title_id)
)
RETURNING id;

-- name: GetUserByID :one
SELECT u.id, u.email, u.full_name, r.code AS role_code, u.status, u.password_hash,
       u.failed_login_count, u.locked_until, u.activation_expires_at, u.last_login_at, u.created_at
FROM app_user u
JOIN role r ON r.id = u.role_id
WHERE u.id = @id;

-- name: GetUserByEmail :one
SELECT u.id, u.email, u.full_name, r.code AS role_code, u.status, u.password_hash,
       u.failed_login_count, u.locked_until, u.activation_expires_at, u.last_login_at, u.created_at
FROM app_user u
JOIN role r ON r.id = u.role_id
WHERE lower(u.email) = lower(@email);

-- name: GetUserByActivationTokenHash :one
SELECT u.id, u.email, u.full_name, r.code AS role_code, u.status, u.password_hash,
       u.failed_login_count, u.locked_until, u.activation_expires_at, u.last_login_at, u.created_at
FROM app_user u
JOIN role r ON r.id = u.role_id
WHERE u.activation_token_hash = @token_hash;

-- name: ActivateUser :execrows
-- Chỉ kích hoạt khi token còn đúng (tránh dùng một token hai lần đồng thời).
UPDATE app_user
SET status = 'active',
    password_hash = @password_hash,
    activation_token_hash = NULL,
    activation_expires_at = NULL,
    failed_login_count = 0,
    locked_until = NULL,
    updated_at = now()
WHERE id = @id AND activation_token_hash = @token_hash AND status = 'pending'
  AND activation_expires_at > @now;

-- name: SetActivationToken :execrows
UPDATE app_user
SET activation_token_hash = @token_hash,
    activation_expires_at = @expires_at,
    updated_at = now()
WHERE id = @id AND status = 'pending';

-- name: RegisterLoginFailure :one
-- B8: tăng bộ đếm NGUYÊN TỬ (không đọc–sửa–ghi ở Go) để request song song không vượt được khóa.
-- Lần sai thứ @max_failures khóa tới @lock_until và đặt lại bộ đếm. Vế phải SET đọc giá trị cũ của hàng.
-- Đang khóa tại @now → không cập nhật (0 dòng): lần sai trong lúc khóa không cộng dồn, không gia hạn khóa.
UPDATE app_user
SET failed_login_count = CASE WHEN failed_login_count + 1 >= @max_failures::int THEN 0
                              ELSE failed_login_count + 1 END,
    locked_until = CASE WHEN failed_login_count + 1 >= @max_failures::int THEN @lock_until::timestamptz
                        ELSE locked_until END,
    updated_at = now()
WHERE id = @id AND (locked_until IS NULL OR locked_until <= @now::timestamptz)
RETURNING locked_until;

-- name: RecordLoginSuccess :execrows
-- Không ghi nếu một request song song vừa khóa tài khoản (0 dòng → coi như đang khóa).
UPDATE app_user
SET failed_login_count = 0,
    locked_until = NULL,
    last_login_at = @logged_in_at,
    updated_at = now()
WHERE id = @id AND (locked_until IS NULL OR locked_until <= @logged_in_at);

-- name: UpsertSeedUser :one
-- Chỉ dùng cho make seed (môi trường dev).
INSERT INTO app_user (email, full_name, role_id, status, password_hash)
VALUES (@email, @full_name, (SELECT r.id FROM role r WHERE r.code = @role_code), 'active', @password_hash)
ON CONFLICT (lower(email)) DO UPDATE
SET full_name = EXCLUDED.full_name,
    role_id = EXCLUDED.role_id,
    status = 'active',
    password_hash = EXCLUDED.password_hash,
    failed_login_count = 0,
    locked_until = NULL,
    activation_token_hash = NULL,
    activation_expires_at = NULL,
    updated_at = now()
RETURNING id;

-- name: ListUsers :many
-- B9: phân trang offset; mới tạo trước. CF-02: lọc theo từ khóa (họ tên/email), vai trò, trạng thái,
-- chức danh và phòng ban (gồm cả phòng ban con). @search là mẫu ILIKE đã thoát ký tự ở Go.
WITH RECURSIVE dept_scope AS (
    SELECT d.id FROM department d WHERE d.id = sqlc.narg(department_id)::uuid
    UNION ALL
    SELECT c.id FROM department c JOIN dept_scope s ON c.parent_id = s.id
)
SELECT u.id, u.email, u.full_name, r.code AS role_code, u.status, u.last_login_at, u.created_at,
       u.department_id, d.name AS department_name, u.job_title_id, j.name AS job_title_name
FROM app_user u
JOIN role r ON r.id = u.role_id
LEFT JOIN department d ON d.id = u.department_id
LEFT JOIN job_title j ON j.id = u.job_title_id
WHERE (sqlc.narg(search)::text IS NULL
       OR u.full_name ILIKE sqlc.narg(search)::text ESCAPE '\'
       OR u.email ILIKE sqlc.narg(search)::text ESCAPE '\')
  AND (sqlc.narg(role_code)::text IS NULL OR r.code = sqlc.narg(role_code)::text)
  AND (sqlc.narg(status)::text IS NULL OR u.status = sqlc.narg(status)::text)
  AND (sqlc.narg(job_title_id)::uuid IS NULL OR u.job_title_id = sqlc.narg(job_title_id)::uuid)
  AND (sqlc.narg(department_id)::uuid IS NULL OR u.department_id IN (SELECT id FROM dept_scope))
ORDER BY u.created_at DESC, u.id
LIMIT @page_limit OFFSET @page_offset;

-- name: CountUsers :one
WITH RECURSIVE dept_scope AS (
    SELECT d.id FROM department d WHERE d.id = sqlc.narg(department_id)::uuid
    UNION ALL
    SELECT c.id FROM department c JOIN dept_scope s ON c.parent_id = s.id
)
SELECT count(*)
FROM app_user u
JOIN role r ON r.id = u.role_id
WHERE (sqlc.narg(search)::text IS NULL
       OR u.full_name ILIKE sqlc.narg(search)::text ESCAPE '\'
       OR u.email ILIKE sqlc.narg(search)::text ESCAPE '\')
  AND (sqlc.narg(role_code)::text IS NULL OR r.code = sqlc.narg(role_code)::text)
  AND (sqlc.narg(status)::text IS NULL OR u.status = sqlc.narg(status)::text)
  AND (sqlc.narg(job_title_id)::uuid IS NULL OR u.job_title_id = sqlc.narg(job_title_id)::uuid)
  AND (sqlc.narg(department_id)::uuid IS NULL OR u.department_id IN (SELECT id FROM dept_scope));

-- name: GetUserDetail :one
-- CF-02: một tài khoản kèm phòng ban/chức danh (cùng tập cột với ListUsers).
SELECT u.id, u.email, u.full_name, r.code AS role_code, u.status, u.last_login_at, u.created_at,
       u.department_id, d.name AS department_name, u.job_title_id, j.name AS job_title_name
FROM app_user u
JOIN role r ON r.id = u.role_id
LEFT JOIN department d ON d.id = u.department_id
LEFT JOIN job_title j ON j.id = u.job_title_id
WHERE u.id = @id;

-- name: UpdateUserProfile :execrows
-- CF-02: email không sửa được sau khi tạo (mục C).
UPDATE app_user
SET full_name = @full_name,
    role_id = (SELECT r.id FROM role r WHERE r.code = @role_code),
    department_id = sqlc.narg(department_id),
    job_title_id = sqlc.narg(job_title_id),
    updated_at = now()
WHERE app_user.id = @id;

-- name: SetSeedUserOrg :exec
-- Chỉ dùng cho make seed.
UPDATE app_user
SET department_id = sqlc.narg(department_id), job_title_id = sqlc.narg(job_title_id), updated_at = now()
WHERE id = @id;
