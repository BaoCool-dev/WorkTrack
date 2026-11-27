-- name: InsertRefreshToken :exec
INSERT INTO refresh_token (id, user_id, family_id, token_hash, expires_at, created_at)
VALUES (@id, @user_id, @family_id, @token_hash, @expires_at, @created_at);

-- name: GetRefreshTokenByHash :one
SELECT id, user_id, family_id, expires_at, revoked_at
FROM refresh_token
WHERE token_hash = @token_hash;

-- name: RevokeRefreshTokenIfActive :execrows
-- Thu hồi nguyên tử: 0 dòng nghĩa là token đã bị dùng (xoay vòng B4).
UPDATE refresh_token
SET revoked_at = @revoked_at
WHERE id = @id AND revoked_at IS NULL;

-- name: RevokeRefreshTokenFamily :exec
UPDATE refresh_token
SET revoked_at = @revoked_at
WHERE family_id = @family_id AND revoked_at IS NULL;
