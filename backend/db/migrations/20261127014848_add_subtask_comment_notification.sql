-- +goose Up
-- CF-07: công việc con 1 cấp, bình luận gắn công việc, thông báo trong hệ thống.

-- Con chỉ 1 cấp (CLAUDE.md §5.4). CHECK không tham chiếu được dòng khác nên dùng trigger:
-- (1) cha phải là công việc gốc; (2) việc đã có con thì không được trở thành con.
-- +goose StatementBegin
CREATE FUNCTION fn_task_one_level() RETURNS trigger AS $$
BEGIN
    IF NEW.parent_task_id IS NULL THEN
        RETURN NEW;
    END IF;
    IF EXISTS (SELECT 1 FROM task WHERE id = NEW.parent_task_id AND parent_task_id IS NOT NULL) THEN
        RAISE EXCEPTION 'subtask parent must be a root task'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'ck_task_one_level';
    END IF;
    IF EXISTS (SELECT 1 FROM task WHERE parent_task_id = NEW.id) THEN
        RAISE EXCEPTION 'task with subtasks cannot become a subtask'
            USING ERRCODE = 'check_violation', CONSTRAINT = 'ck_task_one_level';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_task_one_level
    BEFORE INSERT OR UPDATE OF parent_task_id ON task
    FOR EACH ROW EXECUTE FUNCTION fn_task_one_level();

-- Bình luận trên công việc (cha hoặc con). Xóa công việc thì xóa bình luận.
CREATE TABLE task_comment (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    uuid NOT NULL,
    author_id  uuid NOT NULL,
    body       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_task_comment_task FOREIGN KEY (task_id) REFERENCES task (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_comment_author FOREIGN KEY (author_id) REFERENCES app_user (id) ON DELETE RESTRICT,
    CONSTRAINT ck_task_comment_body CHECK (length(trim(body)) BETWEEN 1 AND 2000)
);

CREATE INDEX ix_task_comment_task ON task_comment (task_id, created_at);

-- Thông báo trong hệ thống (chuông). read_at NULL = chưa đọc.
-- message: trích đoạn bình luận hoặc lý do từ chối nghiệm thu (để chuông hiển thị không cần join thêm).
CREATE TABLE notification (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL,
    type       text NOT NULL,
    task_id    uuid NOT NULL,
    project_id uuid NOT NULL,
    actor_id   uuid,
    message    text NOT NULL DEFAULT '',
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_notification_user FOREIGN KEY (user_id) REFERENCES app_user (id) ON DELETE CASCADE,
    CONSTRAINT fk_notification_task FOREIGN KEY (task_id) REFERENCES task (id) ON DELETE CASCADE,
    CONSTRAINT fk_notification_project FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    CONSTRAINT fk_notification_actor FOREIGN KEY (actor_id) REFERENCES app_user (id) ON DELETE SET NULL,
    CONSTRAINT ck_notification_type CHECK (type IN ('task_assigned', 'task_commented', 'review_rejected'))
);

CREATE INDEX ix_notification_user_created ON notification (user_id, created_at DESC);
CREATE INDEX ix_notification_user_unread ON notification (user_id) WHERE read_at IS NULL;

-- +goose Down
DROP TABLE notification;
DROP TABLE task_comment;
DROP TRIGGER trg_task_one_level ON task;
DROP FUNCTION fn_task_one_level();
