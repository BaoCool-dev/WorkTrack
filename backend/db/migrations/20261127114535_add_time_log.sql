-- +goose Up
-- CF-08: Log time trên công việc cha và con. Đơn vị lưu là phút (CLAUDE.md §5.7).

CREATE TABLE time_log (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    uuid NOT NULL,
    project_id uuid NOT NULL,
    user_id    uuid NOT NULL,
    minutes    integer NOT NULL,
    note       text NOT NULL,
    work_date  date NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_time_log_task FOREIGN KEY (task_id) REFERENCES task (id) ON DELETE CASCADE,
    CONSTRAINT fk_time_log_project FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    CONSTRAINT fk_time_log_user FOREIGN KEY (user_id) REFERENCES app_user (id) ON DELETE RESTRICT,
    -- Phút > 0 (không âm, không 0); tối đa 24 giờ cho một lần ghi.
    CONSTRAINT ck_time_log_minutes CHECK (minutes BETWEEN 1 AND 1440),
    CONSTRAINT ck_time_log_note CHECK (char_length(note) BETWEEN 1 AND 500)
);

-- Lịch sử log của một công việc (mới nhất trước) và báo cáo theo người/ngày (CF-10, CF-11).
CREATE INDEX ix_time_log_task ON time_log (task_id, created_at);
CREATE INDEX ix_time_log_user ON time_log (user_id, work_date);
CREATE INDEX ix_time_log_project ON time_log (project_id);

-- Thêm giờ và số việc con vào view dùng chung để thẻ Kanban hiện "14,5/16h · 2/3 con" mà không N+1.
--   logged_minutes : giờ ghi trên chính công việc này.
--   actual_minutes : giờ thực tế = giờ của nó + tổng giờ các con (con chỉ 1 cấp nên một lần cộng là đủ).
DROP VIEW task_view;
CREATE VIEW task_view AS
SELECT t.id, t.project_id, p.name AS project_name,
       t.column_id, c.name AS column_name, c.status_group, c.position AS column_position,
       t.parent_task_id, t.title, t.description, t.story_point, t.estimated_minutes,
       t.assignee_id, COALESCE(a.full_name, '') AS assignee_name,
       t.due_date, t.position, t.created_by, cb.full_name AS created_by_name,
       t.created_at, t.updated_at, t.version,
       COALESCE(own.minutes, 0)::integer AS logged_minutes,
       (COALESCE(own.minutes, 0) + COALESCE(sub.minutes, 0))::integer AS actual_minutes,
       COALESCE(sub.total, 0)::integer AS subtask_count,
       COALESCE(sub.done, 0)::integer AS subtask_done_count
FROM task t
JOIN project p ON p.id = t.project_id
JOIN board_column c ON c.id = t.column_id
LEFT JOIN app_user a ON a.id = t.assignee_id
JOIN app_user cb ON cb.id = t.created_by
LEFT JOIN LATERAL (
    SELECT sum(l.minutes) AS minutes FROM time_log l WHERE l.task_id = t.id
) own ON true
LEFT JOIN LATERAL (
    SELECT count(*) AS total,
           count(*) FILTER (WHERE sc.status_group = 'done') AS done,
           (SELECT sum(l.minutes) FROM time_log l JOIN task s2 ON s2.id = l.task_id
            WHERE s2.parent_task_id = t.id) AS minutes
    FROM task s
    JOIN board_column sc ON sc.id = s.column_id
    WHERE s.parent_task_id = t.id
) sub ON true;

-- +goose Down
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
DROP TABLE time_log;
