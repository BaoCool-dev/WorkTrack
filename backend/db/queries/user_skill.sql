-- name: ListUserSkillsByUserIDs :many
-- Một truy vấn cho cả trang danh sách (tránh N+1).
SELECT us.user_id, us.skill_id, s.name, us.level
FROM user_skill us
JOIN skill s ON s.id = us.skill_id
WHERE us.user_id = ANY(@user_ids::uuid[])
ORDER BY us.user_id, lower(s.name);

-- name: ReplaceUserSkills :exec
-- Một câu lệnh (nguyên tử): xóa kỹ năng không còn trong danh sách, thêm/cập nhật phần còn lại.
WITH removed AS (
    DELETE FROM user_skill
    WHERE user_skill.user_id = @user_id AND user_skill.skill_id <> ALL(@skill_ids::uuid[])
)
INSERT INTO user_skill (user_id, skill_id, level)
SELECT @user_id, unnest(@skill_ids::uuid[]), unnest(@levels::text[])
ON CONFLICT (user_id, skill_id) DO UPDATE SET level = EXCLUDED.level;
