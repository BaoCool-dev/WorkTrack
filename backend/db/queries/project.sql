-- name: ListProjects :many
-- CF-03, B9: phân trang offset, mới tạo trước. @visible_to NULL = xem tất cả (CEO);
-- có giá trị = chỉ dự án người đó là thành viên. @search là mẫu ILIKE đã thoát ký tự ở Go.
SELECT p.id, p.name, p.description, p.status, p.start_date, p.end_date, p.manager_id,
       m.full_name AS manager_name, p.created_at, p.updated_at,
       (SELECT count(*) FROM project_member pm WHERE pm.project_id = p.id) AS member_count, p.wip_limit
FROM project p
JOIN app_user m ON m.id = p.manager_id
WHERE (sqlc.narg(visible_to)::uuid IS NULL
       OR EXISTS (SELECT 1 FROM project_member pm
                  WHERE pm.project_id = p.id AND pm.user_id = sqlc.narg(visible_to)::uuid))
  AND (sqlc.narg(search)::text IS NULL OR p.name ILIKE sqlc.narg(search)::text ESCAPE '\')
  AND (sqlc.narg(status)::text IS NULL OR p.status = sqlc.narg(status)::text)
ORDER BY p.created_at DESC, p.id
LIMIT @page_limit OFFSET @page_offset;

-- name: CountProjects :one
SELECT count(*)
FROM project p
WHERE (sqlc.narg(visible_to)::uuid IS NULL
       OR EXISTS (SELECT 1 FROM project_member pm
                  WHERE pm.project_id = p.id AND pm.user_id = sqlc.narg(visible_to)::uuid))
  AND (sqlc.narg(search)::text IS NULL OR p.name ILIKE sqlc.narg(search)::text ESCAPE '\')
  AND (sqlc.narg(status)::text IS NULL OR p.status = sqlc.narg(status)::text);

-- name: GetProject :one
SELECT p.id, p.name, p.description, p.status, p.start_date, p.end_date, p.manager_id,
       m.full_name AS manager_name, p.created_at, p.updated_at,
       (SELECT count(*) FROM project_member pm WHERE pm.project_id = p.id) AS member_count, p.wip_limit
FROM project p
JOIN app_user m ON m.id = p.manager_id
WHERE p.id = @id;

-- name: LockProject :one
-- Khóa dòng dự án trong transaction để các thay đổi cấu hình Kanban chạy tuần tự (giữ ràng buộc 2–8 cột).
SELECT id FROM project WHERE id = @id FOR UPDATE;

-- name: LockProjectBoard :one
-- Khóa dòng dự án và trả người phụ trách + giới hạn WIP tại thời điểm khóa: kéo thả/nhận việc kiểm
-- quyền và WIP trên số đã khóa (CF-04, CF-05), không trên bản đọc trước transaction.
SELECT manager_id, wip_limit FROM project WHERE id = @id FOR UPDATE;

-- name: CreateProject :one
INSERT INTO project (name, description, status, start_date, end_date, manager_id, created_by)
VALUES (@name, @description, @status, sqlc.narg(start_date), sqlc.narg(end_date), @manager_id, @created_by)
RETURNING id;

-- name: UpdateProject :execrows
UPDATE project
SET name = @name, description = @description, status = @status,
    start_date = sqlc.narg(start_date), end_date = sqlc.narg(end_date), updated_at = now()
WHERE id = @id;

-- name: UpdateProjectWIPLimit :execrows
-- CF-05: giới hạn WIP mỗi người trong dự án (1–20, CHECK ở CSDL).
UPDATE project SET wip_limit = @wip_limit, updated_at = now() WHERE id = @id;

-- name: IsProjectMember :one
SELECT EXISTS (SELECT 1 FROM project_member WHERE project_id = @project_id AND user_id = @user_id);

-- name: ListProjectMembers :many
SELECT u.id, u.full_name, u.email, r.code AS role_code, u.status, pm.added_at
FROM project_member pm
JOIN app_user u ON u.id = pm.user_id
JOIN role r ON r.id = u.role_id
WHERE pm.project_id = @project_id
ORDER BY lower(u.full_name), u.id;

-- name: AddProjectMember :execrows
-- Bỏ qua nếu đã là thành viên (thêm lại là thao tác idempotent).
INSERT INTO project_member (project_id, user_id)
VALUES (@project_id, @user_id)
ON CONFLICT (project_id, user_id) DO NOTHING;

-- name: RemoveProjectMember :execrows
DELETE FROM project_member WHERE project_id = @project_id AND user_id = @user_id;

-- name: ListMemberCandidates :many
-- Nhân sự có thể thêm vào dự án: Manager/Member chưa bị khóa, chưa là thành viên.
SELECT u.id, u.full_name, u.email, r.code AS role_code, u.status
FROM app_user u
JOIN role r ON r.id = u.role_id
WHERE r.code IN ('manager', 'member')
  AND u.status <> 'disabled'
  AND NOT EXISTS (SELECT 1 FROM project_member pm WHERE pm.project_id = @project_id AND pm.user_id = u.id)
  AND (sqlc.narg(search)::text IS NULL
       OR u.full_name ILIKE sqlc.narg(search)::text ESCAPE '\'
       OR u.email ILIKE sqlc.narg(search)::text ESCAPE '\')
ORDER BY lower(u.full_name), u.id
LIMIT 20;

-- name: GetMemberCandidate :one
-- Kiểm tra một người cụ thể trước khi thêm vào dự án.
SELECT u.id, r.code AS role_code, u.status
FROM app_user u
JOIN role r ON r.id = u.role_id
WHERE u.id = @id;

-- name: GetSeedProjectIDByName :one
-- Chỉ dùng cho make seed.
SELECT id FROM project WHERE lower(name) = lower(@name);
