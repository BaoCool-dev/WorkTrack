-- +goose Up
-- CF-03: dự án, thành viên dự án, cột Kanban theo dự án và luật chuyển trạng thái theo vai trò.

CREATE TABLE project (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'new',
    start_date  date,
    end_date    date,
    -- Manager phụ trách: người duy nhất được sửa thông tin, thành viên và cấu hình Kanban.
    manager_id  uuid NOT NULL,
    created_by  uuid NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_project_manager FOREIGN KEY (manager_id) REFERENCES app_user (id) ON DELETE RESTRICT,
    CONSTRAINT fk_project_created_by FOREIGN KEY (created_by) REFERENCES app_user (id) ON DELETE RESTRICT,
    CONSTRAINT ck_project_status CHECK (status IN ('new', 'in_progress', 'completed', 'paused', 'cancelled')),
    CONSTRAINT ck_project_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT ck_project_dates CHECK (start_date IS NULL OR end_date IS NULL OR end_date >= start_date)
);

CREATE UNIQUE INDEX uq_project_name ON project (lower(name));
CREATE INDEX ix_project_manager ON project (manager_id);
CREATE INDEX ix_project_status ON project (status);

-- Thành viên dự án, gồm cả Manager phụ trách (thêm tự động khi tạo dự án).
CREATE TABLE project_member (
    project_id uuid NOT NULL,
    user_id    uuid NOT NULL,
    added_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT pk_project_member PRIMARY KEY (project_id, user_id),
    CONSTRAINT fk_project_member_project FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    CONSTRAINT fk_project_member_user FOREIGN KEY (user_id) REFERENCES app_user (id) ON DELETE RESTRICT
);

CREATE INDEX ix_project_member_user ON project_member (user_id);

-- Cột Kanban: 2–8 cột mỗi dự án (kiểm ở use case trong transaction), mỗi cột thuộc một nhóm trạng thái.
CREATE TABLE board_column (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   uuid NOT NULL,
    name         text NOT NULL,
    position     smallint NOT NULL,
    status_group text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_board_column_project FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    CONSTRAINT ck_board_column_status_group CHECK (status_group IN ('todo', 'in_progress', 'done')),
    CONSTRAINT ck_board_column_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT ck_board_column_position CHECK (position >= 0 AND position < 8),
    -- Hoãn tới cuối transaction để sắp xếp lại thứ tự cột mà không vướng trùng tạm thời.
    CONSTRAINT uq_board_column_position UNIQUE (project_id, position) DEFERRABLE INITIALLY DEFERRED,
    -- Khóa phụ để workflow_transition tham chiếu cột cùng dự án.
    CONSTRAINT uq_board_column_project_id UNIQUE (project_id, id)
);

CREATE UNIQUE INDEX uq_board_column_name ON board_column (project_id, lower(name));

-- Luật chuyển trạng thái theo vai trò: có dòng = vai trò đó được kéo thẻ từ cột from sang cột to.
-- Chỉ lưu cấu hình; việc áp dụng luật khi kéo thẻ thuộc feature/kanban (CF-04).
CREATE TABLE workflow_transition (
    project_id     uuid NOT NULL,
    from_column_id uuid NOT NULL,
    to_column_id   uuid NOT NULL,
    role           text NOT NULL,
    CONSTRAINT pk_workflow_transition PRIMARY KEY (project_id, from_column_id, to_column_id, role),
    CONSTRAINT fk_workflow_transition_from FOREIGN KEY (project_id, from_column_id)
        REFERENCES board_column (project_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_workflow_transition_to FOREIGN KEY (project_id, to_column_id)
        REFERENCES board_column (project_id, id) ON DELETE CASCADE,
    CONSTRAINT ck_workflow_transition_role CHECK (role IN ('manager', 'member')),
    CONSTRAINT ck_workflow_transition_not_self CHECK (from_column_id <> to_column_id)
);

CREATE INDEX ix_workflow_transition_to ON workflow_transition (project_id, to_column_id);

-- +goose Down
DROP TABLE workflow_transition;
DROP TABLE board_column;
DROP TABLE project_member;
DROP TABLE project;
