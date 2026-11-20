-- +goose Up
-- CF-02: phòng ban dạng cây, chức danh, danh mục kỹ năng và hồ sơ kỹ năng nhân sự.

CREATE TABLE department (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    parent_id  uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_department_parent FOREIGN KEY (parent_id) REFERENCES department (id) ON DELETE RESTRICT,
    CONSTRAINT ck_department_not_self CHECK (parent_id IS NULL OR parent_id <> id),
    CONSTRAINT ck_department_name_not_blank CHECK (length(trim(name)) > 0)
);

-- Tên phòng ban duy nhất trong cùng một phòng ban cha (không phân biệt hoa thường).
CREATE UNIQUE INDEX uq_department_parent_name
    ON department (COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), lower(name));
CREATE INDEX ix_department_parent ON department (parent_id);

CREATE TABLE job_title (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_job_title_name_not_blank CHECK (length(trim(name)) > 0)
);

CREATE UNIQUE INDEX uq_job_title_name ON job_title (lower(name));

CREATE TABLE skill (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_skill_name_not_blank CHECK (length(trim(name)) > 0)
);

CREATE UNIQUE INDEX uq_skill_name ON skill (lower(name));

CREATE TABLE user_skill (
    user_id  uuid NOT NULL,
    skill_id uuid NOT NULL,
    level    text NOT NULL,
    CONSTRAINT pk_user_skill PRIMARY KEY (user_id, skill_id),
    CONSTRAINT fk_user_skill_user FOREIGN KEY (user_id) REFERENCES app_user (id) ON DELETE CASCADE,
    CONSTRAINT fk_user_skill_skill FOREIGN KEY (skill_id) REFERENCES skill (id) ON DELETE RESTRICT,
    CONSTRAINT ck_user_skill_level CHECK (level IN ('beginner', 'intermediate', 'senior'))
);

CREATE INDEX ix_user_skill_skill ON user_skill (skill_id);

ALTER TABLE app_user
    ADD COLUMN department_id uuid,
    ADD COLUMN job_title_id  uuid,
    ADD CONSTRAINT fk_app_user_department FOREIGN KEY (department_id) REFERENCES department (id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_app_user_job_title FOREIGN KEY (job_title_id) REFERENCES job_title (id) ON DELETE RESTRICT;

CREATE INDEX ix_app_user_department ON app_user (department_id);
CREATE INDEX ix_app_user_job_title ON app_user (job_title_id);

-- +goose Down
ALTER TABLE app_user
    DROP CONSTRAINT fk_app_user_job_title,
    DROP CONSTRAINT fk_app_user_department,
    DROP COLUMN job_title_id,
    DROP COLUMN department_id;
DROP TABLE user_skill;
DROP TABLE skill;
DROP TABLE job_title;
DROP TABLE department;
