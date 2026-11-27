-- +goose Up
-- CF-06: công việc cha (con thuộc CF-07 dùng chung bảng), nhãn, tham số hệ thống (B1: SP → giờ).

-- Tham số cấu hình hệ thống Admin sửa được. Hiện chỉ có story_point_hours (B1).
CREATE TABLE system_setting (
    key        text PRIMARY KEY,
    int_value  integer NOT NULL,
    updated_by uuid,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_system_setting_updated_by FOREIGN KEY (updated_by) REFERENCES app_user (id) ON DELETE SET NULL,
    CONSTRAINT ck_system_setting_story_point_hours CHECK (key <> 'story_point_hours' OR int_value BETWEEN 1 AND 40)
);

-- B1: 1 SP = 4 giờ (mặc định, Admin sửa được).
INSERT INTO system_setting (key, int_value) VALUES ('story_point_hours', 4);

-- Danh mục nhãn dùng chung toàn công ty (Admin, CEO quản lý).
CREATE TABLE label (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_label_name_not_blank CHECK (length(trim(name)) > 0)
);

CREATE UNIQUE INDEX uq_label_name ON label (lower(name));

-- Công việc. parent_task_id NULL = công việc cha (gốc); con 1 cấp thuộc CF-07.
CREATE TABLE task (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id        uuid NOT NULL,
    column_id         uuid NOT NULL,
    parent_task_id    uuid,
    title             text NOT NULL,
    description       text NOT NULL DEFAULT '',
    story_point       smallint,
    -- Giờ ước tính (phút) chụp lại lúc đặt Story Point = SP × hệ số B1 × 60; đổi hệ số không sửa việc cũ.
    estimated_minutes integer,
    assignee_id       uuid,
    due_date          date,
    position          integer NOT NULL DEFAULT 0,
    created_by        uuid NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_task_project FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    -- Cột phải thuộc cùng dự án; cột còn việc không xóa được (COLUMN_HAS_TASKS).
    CONSTRAINT fk_task_column FOREIGN KEY (project_id, column_id) REFERENCES board_column (project_id, id) ON DELETE RESTRICT,
    CONSTRAINT uq_task_project_id UNIQUE (project_id, id),
    -- Con phải cùng dự án với cha.
    CONSTRAINT fk_task_parent FOREIGN KEY (project_id, parent_task_id) REFERENCES task (project_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_task_assignee FOREIGN KEY (assignee_id) REFERENCES app_user (id) ON DELETE RESTRICT,
    CONSTRAINT fk_task_created_by FOREIGN KEY (created_by) REFERENCES app_user (id) ON DELETE RESTRICT,
    CONSTRAINT ck_task_title_not_blank CHECK (length(trim(title)) > 0),
    -- Story Point chỉ ở công việc cha (CLAUDE.md §5.4).
    CONSTRAINT ck_task_story_point_parent_only CHECK (parent_task_id IS NULL OR story_point IS NULL),
    CONSTRAINT ck_task_story_point_range CHECK (story_point IS NULL OR story_point BETWEEN 1 AND 100),
    CONSTRAINT ck_task_estimated_minutes CHECK (estimated_minutes IS NULL OR estimated_minutes >= 0)
);

CREATE INDEX ix_task_project_column ON task (project_id, column_id, position);
CREATE INDEX ix_task_assignee ON task (assignee_id) WHERE assignee_id IS NOT NULL;
CREATE INDEX ix_task_parent ON task (parent_task_id) WHERE parent_task_id IS NOT NULL;

-- Nhãn gắn vào công việc. Nhãn còn gắn vào việc không xóa được (LABEL_IN_USE).
CREATE TABLE task_label (
    task_id  uuid NOT NULL,
    label_id uuid NOT NULL,
    CONSTRAINT pk_task_label PRIMARY KEY (task_id, label_id),
    CONSTRAINT fk_task_label_task FOREIGN KEY (task_id) REFERENCES task (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_label_label FOREIGN KEY (label_id) REFERENCES label (id) ON DELETE RESTRICT
);

CREATE INDEX ix_task_label_label ON task_label (label_id);

-- Dòng công việc kèm tên dự án, cột, người nhận, người tạo — dùng chung cho danh sách và chi tiết.
CREATE VIEW task_view AS
SELECT t.id, t.project_id, p.name AS project_name,
       t.column_id, c.name AS column_name, c.status_group, c.position AS column_position,
       t.parent_task_id, t.title, t.description, t.story_point, t.estimated_minutes,
       t.assignee_id, COALESCE(a.full_name, '') AS assignee_name,
       t.due_date, t.position, t.created_by, cb.full_name AS created_by_name,
       t.created_at, t.updated_at
FROM task t
JOIN project p ON p.id = t.project_id
JOIN board_column c ON c.id = t.column_id
LEFT JOIN app_user a ON a.id = t.assignee_id
JOIN app_user cb ON cb.id = t.created_by;

-- +goose Down
DROP VIEW IF EXISTS task_view;
DROP TABLE IF EXISTS task_label;
DROP TABLE IF EXISTS task;
DROP TABLE IF EXISTS label;
DROP TABLE IF EXISTS system_setting;
