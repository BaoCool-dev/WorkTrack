-- +goose Up
-- CF-09: đổi FK cột Kanban của task_review từ CASCADE sang RESTRICT.
-- Vì sao: task_review là bằng chứng nghiệm thu, dùng để tính tỷ lệ làm lại (CF-10, CF-11).
-- Với CASCADE, Manager xóa một cột Kanban sẽ lặng lẽ xóa luôn lịch sử Duyệt/Từ chối và làm sai báo cáo.
-- RESTRICT buộc ứng dụng chặn việc xóa cột đã có lịch sử nghiệm thu thay vì mất dữ liệu.
ALTER TABLE task_review DROP CONSTRAINT fk_task_review_from_column;
ALTER TABLE task_review DROP CONSTRAINT fk_task_review_to_column;
ALTER TABLE task_review
    ADD CONSTRAINT fk_task_review_from_column FOREIGN KEY (from_column_id) REFERENCES board_column (id) ON DELETE RESTRICT,
    ADD CONSTRAINT fk_task_review_to_column FOREIGN KEY (to_column_id) REFERENCES board_column (id) ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE task_review DROP CONSTRAINT fk_task_review_from_column;
ALTER TABLE task_review DROP CONSTRAINT fk_task_review_to_column;
ALTER TABLE task_review
    ADD CONSTRAINT fk_task_review_from_column FOREIGN KEY (from_column_id) REFERENCES board_column (id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_task_review_to_column FOREIGN KEY (to_column_id) REFERENCES board_column (id) ON DELETE CASCADE;
