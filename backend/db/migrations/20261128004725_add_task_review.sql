-- +goose Up
-- CF-09: nghiệm thu công việc cha. Mỗi lần Manager Duyệt hoặc Từ chối ghi một dòng.
-- Vì sao cần bảng riêng thay vì chỉ dựa vào task_move_history: tỷ lệ làm lại
-- (= số việc bị từ chối ít nhất 1 lần ÷ tổng việc đã nghiệm thu) cần biết lần chuyển nào là "nghiệm thu"
-- và lý do từ chối; một lần kéo tay sang Done không phải là nghiệm thu.

CREATE TABLE task_review (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id        uuid NOT NULL,
    project_id     uuid NOT NULL,
    reviewer_id    uuid NOT NULL,
    decision       text NOT NULL,
    reason         text,
    from_column_id uuid NOT NULL,
    to_column_id   uuid NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_task_review_task FOREIGN KEY (task_id) REFERENCES task (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_review_project FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_review_reviewer FOREIGN KEY (reviewer_id) REFERENCES app_user (id) ON DELETE RESTRICT,
    CONSTRAINT fk_task_review_from_column FOREIGN KEY (from_column_id) REFERENCES board_column (id) ON DELETE CASCADE,
    CONSTRAINT fk_task_review_to_column FOREIGN KEY (to_column_id) REFERENCES board_column (id) ON DELETE CASCADE,
    CONSTRAINT ck_task_review_decision CHECK (decision IN ('approved', 'rejected')),
    -- Từ chối bắt buộc lý do 1–500 ký tự; Duyệt không có lý do.
    CONSTRAINT ck_task_review_reason CHECK (
        (decision = 'rejected' AND reason IS NOT NULL AND char_length(reason) BETWEEN 1 AND 500)
        OR (decision = 'approved' AND reason IS NULL)
    )
);

-- Lịch sử nghiệm thu của một công việc (mới nhất trước) và tỷ lệ làm lại theo dự án (CF-10, CF-11).
CREATE INDEX ix_task_review_task ON task_review (task_id, created_at);
CREATE INDEX ix_task_review_project ON task_review (project_id);
CREATE INDEX ix_task_review_reviewer ON task_review (reviewer_id);
CREATE INDEX ix_task_review_from_column ON task_review (from_column_id);
CREATE INDEX ix_task_review_to_column ON task_review (to_column_id);

-- +goose Down
DROP TABLE task_review;
