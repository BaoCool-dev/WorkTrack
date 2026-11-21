-- +goose Up
-- CF-04, CF-05: giới hạn WIP theo dự án, chống ghi đè bằng version, lịch sử chuyển cột.

-- Giới hạn WIP của mỗi người trong một dự án: số công việc CHA nhóm in_progress tối đa (CLAUDE.md §5.6).
-- Mặc định 5; Manager phụ trách sửa trong cấu hình dự án.
ALTER TABLE project
    ADD COLUMN wip_limit smallint NOT NULL DEFAULT 5,
    ADD CONSTRAINT ck_project_wip_limit CHECK (wip_limit BETWEEN 1 AND 20);

-- version tăng mỗi lần công việc đổi (chuyển cột, sửa, nhận việc). Client gửi version đang thấy;
-- lệch nghĩa là người khác vừa sửa → 409 TASK_VERSION_CONFLICT thay vì ghi đè im lặng.
ALTER TABLE task
    ADD COLUMN version integer NOT NULL DEFAULT 1;

-- Lịch sử chuyển cột: mỗi lần công việc đổi cột ghi một dòng (đổi thứ tự trong cùng cột không ghi).
-- Không FK tới board_column: xóa/đổi cột về sau không được làm mất lịch sử.
CREATE TABLE task_move_history (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id        uuid NOT NULL,
    project_id     uuid NOT NULL,
    from_column_id uuid NOT NULL,
    to_column_id   uuid NOT NULL,
    moved_by       uuid NOT NULL,
    moved_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_task_move_history_task FOREIGN KEY (task_id) REFERENCES task (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_move_history_project FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_move_history_moved_by FOREIGN KEY (moved_by) REFERENCES app_user (id) ON DELETE RESTRICT,
    CONSTRAINT ck_task_move_history_columns CHECK (from_column_id <> to_column_id)
);

CREATE INDEX ix_task_move_history_task ON task_move_history (task_id, moved_at);

-- Thêm version vào view dùng chung (tạo lại view vì thêm cột).
DROP VIEW task_view;
CREATE VIEW task_view AS
SELECT t.id, t.project_id, p.name AS project_name,
       t.column_id, c.name AS column_name, c.status_group, c.position AS column_position,
       t.parent_task_id, t.title, t.description, t.story_point, t.estimated_minutes,
       t.assignee_id, COALESCE(a.full_name, '') AS assignee_name,
       t.due_date, t.position, t.created_by, cb.full_name AS created_by_name,
       t.created_at, t.updated_at, t.version
FROM task t
JOIN project p ON p.id = t.project_id
JOIN board_column c ON c.id = t.column_id
LEFT JOIN app_user a ON a.id = t.assignee_id
JOIN app_user cb ON cb.id = t.created_by;

-- +goose Down
DROP VIEW task_view;
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
DROP TABLE task_move_history;
ALTER TABLE task DROP COLUMN version;
ALTER TABLE project DROP CONSTRAINT ck_project_wip_limit, DROP COLUMN wip_limit;
