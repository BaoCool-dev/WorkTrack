-- +goose Up
-- Index cho khóa ngoại task_id/project_id của notification: ON DELETE CASCADE khi xóa task/dự án
-- không phải quét toàn bảng notification.
CREATE INDEX ix_notification_task ON notification (task_id);
CREATE INDEX ix_notification_project ON notification (project_id);

-- +goose Down
DROP INDEX ix_notification_project;
DROP INDEX ix_notification_task;
