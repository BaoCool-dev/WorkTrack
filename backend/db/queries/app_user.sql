-- name: CreateUser :one
INSERT INTO app_user (email, full_name, role_id, activation_token_hash, activation_expires_at, created_by)
VALUES (
    @email,
    @full_name,
    (SELECT r.id FROM role r WHERE r.code = @role_code),
    @activation_token_hash,
    @activation_expires_at,
    sqlc.narg(created_by)
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
UPDATE app_user
SET failed_login_count = CASE WHEN failed_login_count + 1 >= @max_failures::int THEN 0
                              ELSE failed_login_count + 1 END,
    locked_until = CASE WHEN failed_login_count + 1 >= @max_failures::int THEN @lock_until::timestamptz
                        ELSE locked_until END,
    updated_at = now()
WHERE id = @id
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
-- B9: phân trang offset; mới tạo trước.
SELECT u.id, u.email, u.full_name, r.code AS role_code, u.status, u.last_login_at, u.created_at
FROM app_user u
JOIN role r ON r.id = u.role_id
ORDER BY u.created_at DESC, u.id
LIMIT @page_limit OFFSET @page_offset;

-- name: CountUsers :one
SELECT count(*) FROM app_user;
