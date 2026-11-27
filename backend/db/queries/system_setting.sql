-- name: GetIntSetting :one
-- B1: tham số hệ thống dạng số nguyên (vd. story_point_hours).
SELECT int_value, updated_at FROM system_setting WHERE key = @key;

-- name: SetIntSetting :execrows
UPDATE system_setting
SET int_value = @int_value, updated_by = @updated_by, updated_at = now()
WHERE key = @key;
