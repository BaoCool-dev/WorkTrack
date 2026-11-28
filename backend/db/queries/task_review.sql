-- CF-09 — nghiệm thu công việc cha.

-- name: InsertTaskReview :exec
-- Ghi một lần Duyệt/Từ chối, cùng transaction với việc chuyển cột.
INSERT INTO task_review (task_id, project_id, reviewer_id, decision, reason, from_column_id, to_column_id)
VALUES (@task_id, @project_id, @reviewer_id, @decision, sqlc.narg(reason), @from_column_id, @to_column_id);

-- name: ListTaskReviews :many
-- Lịch sử nghiệm thu của một công việc, mới nhất trước.
SELECT r.id, r.task_id, r.reviewer_id, u.full_name AS reviewer_name, r.decision, r.reason, r.created_at
FROM task_review r
JOIN app_user u ON u.id = r.reviewer_id
WHERE r.task_id = @task_id
ORDER BY r.created_at DESC, r.id DESC
LIMIT 100;
