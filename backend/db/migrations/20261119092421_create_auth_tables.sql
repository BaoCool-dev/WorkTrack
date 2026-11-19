-- +goose Up
-- CF-01: tài khoản, vai trò và refresh token (B2, B4, B7, B8).

CREATE TABLE role (
    id   smallint PRIMARY KEY,
    code text NOT NULL,
    name text NOT NULL,
    CONSTRAINT uq_role_code UNIQUE (code),
    CONSTRAINT ck_role_code CHECK (code IN ('admin', 'manager', 'member', 'ceo'))
);

INSERT INTO role (id, code, name) VALUES
    (1, 'admin', 'Admin'),
    (2, 'manager', 'Manager'),
    (3, 'member', 'Member'),
    (4, 'ceo', 'CEO');

CREATE TABLE app_user (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email                 text NOT NULL,
    full_name             text NOT NULL,
    role_id               smallint NOT NULL REFERENCES role (id),
    status                text NOT NULL DEFAULT 'pending',
    password_hash         text,
    -- B7: token kích hoạt chỉ lưu SHA-256, hiệu lực 72 giờ.
    activation_token_hash bytea,
    activation_expires_at timestamptz,
    -- B8: khóa 15 phút sau 5 lần sai liên tiếp.
    failed_login_count    integer NOT NULL DEFAULT 0,
    locked_until          timestamptz,
    last_login_at         timestamptz,
    created_by            uuid REFERENCES app_user (id),
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_app_user_status CHECK (status IN ('pending', 'active', 'disabled')),
    CONSTRAINT ck_app_user_active_has_password CHECK (status <> 'active' OR password_hash IS NOT NULL),
    CONSTRAINT ck_app_user_failed_login_count CHECK (failed_login_count >= 0),
    CONSTRAINT ck_app_user_email_not_blank CHECK (length(trim(email)) > 0)
);

CREATE UNIQUE INDEX uq_app_user_email ON app_user (lower(email));
CREATE UNIQUE INDEX uq_app_user_activation_token ON app_user (activation_token_hash)
    WHERE activation_token_hash IS NOT NULL;

CREATE TABLE refresh_token (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES app_user (id) ON DELETE CASCADE,
    -- Mọi token sinh ra từ một lần đăng nhập chung family_id; dùng lại token cũ → thu hồi cả họ (B4).
    family_id   uuid NOT NULL,
    token_hash  bytea NOT NULL,
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_refresh_token_hash UNIQUE (token_hash)
);

CREATE INDEX ix_refresh_token_user ON refresh_token (user_id);
CREATE INDEX ix_refresh_token_family ON refresh_token (family_id);

-- +goose Down
DROP TABLE refresh_token;
DROP TABLE app_user;
DROP TABLE role;
