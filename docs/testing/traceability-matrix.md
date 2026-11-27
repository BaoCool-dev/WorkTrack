# Ma trận truy vết — yêu cầu → test

> Cập nhật theo từng nhánh tính năng (feature-gate §5).
> `docs/requirements/` chưa có `commitment.md` / `use-cases.md` → tạm truy vết theo mã CF và các quy tắc
> mục B của `docs/decisions.md`. Khi có UC/YCCN chính thức, bổ sung cột mã UC, không đánh số lại.
>
> Viết tắt file test:
> - **api** = `backend/internal/transport/httpapi/auth_api_test.go`
> - **auth** = `backend/internal/application/auth/service_test.go`
> - **acc** = `backend/internal/application/account/service_test.go`
> - **authz** = `backend/internal/application/authz/authz_test.go`
> - **dom** = `backend/internal/domain/user_test.go`, `page_test.go`
> - **sec** = `backend/internal/infrastructure/security/security_test.go`
> - **tok** = `backend/internal/application/securetoken/securetoken_test.go`
> - **mail** = `backend/internal/infrastructure/email/smtp_test.go`
> - **rl** = `backend/internal/transport/httpapi/ratelimit_test.go`
> - **repo** = `backend/internal/infrastructure/postgres/repo_integration_test.go` (tag `integration`)
> - **seed** = `backend/cmd/seed/main_test.go`
> - **app** = `frontend/src/app/app.test.tsx`
> - **sess** = `frontend/src/features/auth/model/session.test.ts`
> - **sch** = `frontend/src/features/auth/model/schemas.test.ts`
> - **acs** = `frontend/src/features/auth/model/access.test.ts`
> - **http** = `frontend/src/shared/api/http.test.ts`

## CF-01 — Đăng nhập, kích hoạt tài khoản, phân quyền 4 vai trò (nhánh `feature/auth`)

| # | Tiêu chí | Loại ca | Test |
|---|---|---|---|
| 1 | Đăng nhập đúng → access token + cookie refresh `HttpOnly; Secure; SameSite` | dương | api `TestLogin_Valid_ReturnsTokenAndSecureCookie_CF01`; auth `TestLogin_Success_CF01`; app "đúng → về trang next và sidebar theo vai trò" |
| 2 | Sai mật khẩu / email không tồn tại → lỗi chung, không lộ email có tồn tại | âm | auth `TestLogin_UnknownEmail_GenericError_CF01`; api `TestLogin_Errors_MapToHTTP_CF01`; app "sai mật khẩu → hiện thông báo của máy chủ" |
| 3 | Tài khoản chờ kích hoạt / bị khóa không đăng nhập được | âm | auth `TestLogin_PendingOrDisabled_CF01`; dom `TestUserCanLogin_Status_CF01` |
| 4 | B8: 5 lần sai liên tiếp → khóa 15 phút (423); đếm nguyên tử khi song song | âm/biên | auth `TestLogin_WrongPassword_FiveTimesLocks_CF01_B8`, `TestLogin_LockedByConcurrentRequest_ReturnsLocked_CF01_B8`; dom `TestUserIsLocked_CF01_B8`; repo `TestUserRepo_RegisterLoginFailure_Concurrent_CF01_B8` |
| 5 | B8: đăng nhập đúng xóa bộ đếm sai | dương | auth `TestLogin_SuccessResetsFailureCount_CF01_B8` |
| 6 | Giới hạn tần suất đăng nhập → 429 + `Retry-After` | âm | api `TestLogin_TooManyRequests_Returns429WithRetryAfter_CF01`; rl `TestRateLimiter_RefillsOverTime_CF01` |
| 7 | Validate input đăng nhập (trống, khoảng trắng, email) ở FE và BE | âm/biên | app "bỏ trống → báo lỗi dưới ô, không gọi API"; sch "cắt khoảng trắng quanh email"; dom `TestNormalizeEmail_CF01`; api `TestLogin_Errors_MapToHTTP_CF01` |
| 8 | B4: JWT HS256, hết hạn/sai chữ ký/`alg: none` → 401; secret ngắn bị từ chối | âm | sec `TestJWT_IssueVerify_CF01_B4`, `TestJWT_Verify_RejectsBadTokens_CF01_B4`, `TestNewJWT_ShortSecret_CF01_B4`; api `TestMe_AuthMatrix_CF01` |
| 9 | B4: refresh xoay vòng; dùng lại token cũ → thu hồi cả họ; song song chỉ một thắng | dương/âm | auth `TestRefresh_RotatesToken_CF01_B4`, `TestRefresh_ReuseRevokesFamily_CF01_B4`, `TestRefresh_ConcurrentUseRevokesFamily_CF01_B4`, `TestRefresh_Invalid_CF01_B4`; repo `TestRefreshTokenRepo_Rotation_CF01_B4`; api `TestRefresh_WithCookie_RotatesCookie_CF01`, `TestRefresh_Invalid_Returns401AndClearsCookie_CF01` |
| 10 | B4: access token chỉ trong bộ nhớ, tự làm mới trước hạn, single-flight | dương | sess "refresh single-flight…", "tự làm mới trước khi access token hết hạn 60 giây", "start gọi nhiều lần (StrictMode)…"; http "401 → làm mới token một lần rồi gửi lại request", "401 và làm mới thất bại → … không lặp vô hạn" |
| 11 | Tài khoản bị vô hiệu sau khi đăng nhập → 401 ở `/me` và refresh | âm | api `TestMe_AccountDisabledAfterLogin_Returns401_CF01`; auth `TestRefresh_UserDisabledOrDeleted_CF01` |
| 12 | Đăng xuất thu hồi họ token, xóa cookie, idempotent | dương | auth `TestLogout_RevokesFamily_Idempotent_CF01`; api `TestLogout_RevokesAndClearsCookie_CF01`, `TestLogout_NoCookie_Returns204WithoutCallingService_CF01`; app "bấm Đăng xuất → gọi API, toast, về /login", "đang đăng xuất → nút bị khóa…" |
| 13 | B7: Admin/CEO tạo người dùng → `pending` + email kích hoạt; email lỗi vẫn tạo; trùng email → 409 | dương/âm | acc `TestCreateUser_AdminCreatesPendingAndSendsEmail_CF01_B7`, `TestCreateUser_MailFailureStillCreates_CF01`, `TestCreateUser_DuplicateEmail_CF01`; api `TestCreateUser_CEOCreatesCEO_Returns201_CF01`, `TestCreateUser_Validation_CF01`; mail `TestSMTPMailer_*` |
| 14 | B7: token kích hoạt 72 giờ, lưu hash, dùng một lần; gửi lại vô hiệu token cũ | dương/âm/biên | acc `TestActivate_Expired72h_CF01_B7`, `TestActivate_InvalidToken_CF01_B7`, `TestActivate_SetsPasswordOnce_CF01_B7`, `TestResendActivation_NewTokenInvalidatesOld_CF01_B7`; tok `TestNew_UniqueAndHashMatches_B7`; repo `TestUserRepo_CreateActivateLogin_CF01`; app "token hết hạn…", "thiếu token…", "token hợp lệ → xóa token khỏi URL…", "lỗi mạng khi kiểm tra liên kết…" |
| 15 | B8: chính sách mật khẩu ≥ 8 ký tự (đếm rune, có dấu), có chữ và số; FE và BE cùng luật | âm/biên | dom `TestValidatePassword_Policy_CF01_B8`; acc `TestActivate_WeakPassword_CF01_B8`; sch "chấp nhận mật khẩu có dấu tiếng Việt: 8 rune…", "hai mật khẩu không khớp…"; sec `TestBcryptHasher_HashCompare_CF01_B8` |
| 16 | B2: ma trận quyền 4 vai trò; CEO kế thừa Admin; Admin không tạo dự án | dương/âm | authz `TestCan_Matrix_CF01_B2`, `TestCan_CEOInheritsAdmin_CF01_B2`, `TestCan_AdminCannotCreateProject_CF01_R`, `TestCan_UnknownRoleOrPermission_Denied`; api `TestUsersEndpoints_RoleMatrix_CF01`, `TestMe_ReturnsPermissionsOfRole_CF01`; acc `TestCreateUser_Forbidden_CF01_B2` |
| 17 | Route guard FE: chưa đăng nhập → `/login?next=`; sai vai trò → trạng thái không có quyền; chặn open redirect | dương/âm | app "chưa đăng nhập vào trang trong…", "Manager gõ thẳng /admin/users…", "Admin vào /admin/users…", "next trỏ ra ngoài (//evil)…"; acs (safeNext) |
| 18 | B9: danh sách người dùng phân trang `{items,page,pageSize,total}`, pageSize ≤ 100, tham số sai → 400 | dương/biên | dom `TestNewPageRequest_Normalize_B9`; api `TestListUsers_ReturnsB9Shape_CF01`, `TestListUsers_BadPageParam_Returns400_CF01`; acc `TestListUsers_CF01` |
| 19 | Gửi lại kích hoạt: UUID sai → 400 | âm | api `TestResendActivation_BadUUID_Returns400_CF01`; acc `TestResendActivation_Errors_CF01` |
| 20 | B11: seed 1 tài khoản mỗi vai trò, mật khẩu ngẫu nhiên đạt chính sách | dương | seed `TestSeedUsers_OnePerRole_B11`; tok `TestNewPassword_SatisfiesPolicyAndIsRandom_CF01_B11` |
| 21 | Trạng thái giao diện: skeleton khi kiểm tra phiên, 404 trong app shell, một `h1` | UX | app "đang kiểm tra phiên → hiện skeleton…", "đường dẫn không tồn tại…", "chỉ có một tiêu đề trang…"; ảnh `docs/testing/evidence/feature-auth/` |

**Khoảng trống đã biết:** chưa có E2E Playwright tự động (đã đi tay bằng Playwright MCP, ảnh trong `evidence/feature-auth/`). Màn tạo người dùng thuộc CF-02 nên chỉ kiểm qua API (mục C, 2026-11-20).

## CF-02 — Phòng ban dạng cây, chức danh, kỹ năng, hồ sơ nhân sự (nhánh `feature/org`)

Viết tắt: **orgsvc** = `backend/internal/application/org/service_test.go` · **orgapi** = `backend/internal/transport/httpapi/org_api_test.go` ·
**prof** = `backend/internal/application/account/profile_test.go` · **odom** = `backend/internal/domain/org_test.go` ·
**orgit** = `backend/internal/infrastructure/postgres/org_integration_test.go` (tag `integration`) ·
**view** = `frontend/src/features/admin/components/AdminOrgView.test.tsx` · **fmodel** = `frontend/src/features/admin/model/model.test.ts`

| # | Tiêu chí | Loại ca | Test |
|---|---|---|---|
| 1 | Chỉ Admin/CEO quản lý phòng ban, chức danh, kỹ năng; Manager/Member → 403; không token → 401 | âm | orgsvc `TestService_OnlyAdminAndCEOManageOrg_CF02`; orgapi `TestOrgAPI_RoleMatrix_CF02` |
| 2 | Phòng ban dạng cây: tạo con dưới cha, cha không tồn tại → 404, trùng tên cùng cha → 409 | dương/âm | orgsvc `TestCreateDepartment_UnknownParent_CF02`, `TestCreateDepartment_DuplicateUnderSameParent_CF02`; orgit `TestOrgRepo_DepartmentTree_CF02`; view "cây phòng ban: con nằm dưới cha…" |
| 3 | Không tạo vòng: chọn chính nó hoặc con cháu làm cha bị chặn (FE và BE); dữ liệu hỏng có vòng vẫn dừng | âm/biên | odom `TestValidateDepartmentParent_CF02`, `TestValidateDepartmentParent_CorruptLoopTerminates_CF02`; orgsvc `TestUpdateDepartment_RejectsCycle_CF02`; view "sửa phòng ban: không chọn được chính nó…"; fmodel "con cháu gồm chính nó" |
| 4 | Xóa phòng ban còn con / còn nhân sự → lý do từ server, không đóng hộp thoại | âm | orgapi `TestOrgAPI_DomainErrorMapping_CF02`; view "xóa phòng ban còn phòng ban con…" |
| 5 | Tên: cắt khoảng trắng, rỗng/chỉ khoảng trắng/quá dài bị từ chối, tiếng Việt có dấu | biên | odom `TestNormalizeOrgName_CF02`; orgsvc `TestCreateDepartment_NormalizesName_CF02`, `TestJobTitleAndSkill_ValidateName_CF02`; orgapi `TestOrgAPI_BadInput_CF02`; fmodel "tên rỗng hoặc quá 200 ký tự…" |
| 6 | Chức danh, kỹ năng: thêm, trùng tên → 409, xóa sau xác nhận | dương/âm | orgit `TestOrgRepo_CatalogsAndSkills_CF02`; view "thêm chức danh; tên trùng (409)…", "xóa kỹ năng sau khi xác nhận" |
| 7 | Hồ sơ kỹ năng mức Beginner/Intermediate/Senior; mức lạ → 422, kỹ năng trùng/không tồn tại bị chặn | dương/âm | odom `TestParseSkillLevel_CF02`, `TestValidateUserSkills_CF02`; prof `TestSetUserSkills_CF02`; orgapi `TestUserAPI_SetSkills_CF02`; view "thêm kỹ năng, chọn mức độ rồi lưu…" |
| 8 | Danh sách nhân sự: tìm kiếm (cắt khoảng trắng), lọc vai trò/trạng thái/phòng ban (gồm phòng con)/chức danh, phân trang B9 | dương | prof `TestListUsers_PassesFilterAndLoadsSkillsOnce_CF02`, `TestListUsers_TrimsSearch_CF02`; orgapi `TestUserAPI_ListFilters_CF02`; view "tìm kiếm gửi search…", "phân trang: bấm Trang sau…" |
| 9 | Form thêm/sửa: validate client, email trùng (409) → "Email đã tồn tại" dưới ô email; email chỉ đọc khi sửa, BE từ chối đổi email | âm | prof `TestCreateUser_WithDepartmentAndJobTitle_CF02`, `TestUpdateUser_Rules_CF02`; orgapi `TestUserAPI_UpdateRejectsEmailChange_CF02`, `TestUserAPI_UpdateErrors_CF02`; view "validate phía client…", "email trùng (409)…", "sửa: email chỉ đọc…"; fmodel "họ tên đúng 100 ký tự có dấu…" |
| 10 | Admin không gán/sửa CEO, không tự đổi vai trò mình; CEO gán được CEO | âm | prof `TestUpdateUser_Rules_CF02`; view "Admin không thấy lựa chọn vai trò CEO…", "Admin không có nút Sửa ở dòng của CEO" |
| 11 | Trạng thái màn hình: rỗng / không khớp bộ lọc / lỗi + mã lỗi + Thử lại; lỗi gửi lại kích hoạt hiện dải lỗi (không toast) | UX | view "chưa có ai → trạng thái rỗng…", "lỗi tải danh sách…", "gửi lại kích hoạt thất bại…"; ảnh `docs/testing/evidence/cf02-*.png` |

**Khoảng trống đã biết:** chưa có E2E Playwright tự động (đi tay bằng Playwright MCP). Chưa có integration test cho `skills: []` (go-reviewer gợi ý). Không có artboard trong repo.

## CF-03 — Dự án, thành viên, cột Kanban, luật chuyển (nhánh `feature/projects`)

Viết tắt: **pdom** = `backend/internal/domain/project_test.go` · **psvc** = `backend/internal/application/project/service_test.go` ·
**papi** = `backend/internal/transport/httpapi/project_api_test.go` ·
**pit** = `backend/internal/infrastructure/postgres/project_integration_test.go` (tag `integration`) ·
**plist** = `frontend/src/features/projects/components/ProjectList.test.tsx` · **pdet** = `frontend/src/features/projects/components/ProjectDetail.test.tsx`

| # | Tiêu chí | Loại ca | Test |
|---|---|---|---|
| 1 | Chỉ Manager tạo dự án (trống, kèm bảng mặc định); Admin/Member/CEO → 403; trùng tên (không phân biệt hoa thường) → 409 và rollback; không token → 401 | dương/âm | psvc `TestCreate_ManagerGetsDefaultsBoard_CF03`, `TestCreate_OnlyManager_CF03`, `TestCreate_DuplicateNameRollsBack_CF03`; papi `TestProjectAPI_HappyPaths_CF03`, `TestProjectAPI_RequiresAuth`, `TestProjectAPI_ErrorMapping`; pit `TestProjectRepo_ServiceFlow_CF03`; plist "Manager tạo dự án: gửi đúng body…", "trùng tên từ server hiện dưới ô tên" |
| 2 | Tên/mô tả/ngày: cắt khoảng trắng, rỗng, đúng 150 rune, 151 rune, mô tả 2001 ký tự, ngày kết thúc trước ngày bắt đầu, cùng ngày; ngày sai định dạng → 400 | âm/biên | pdom `TestNormalizeProjectName_CF03`, `TestNormalizeProjectDescription_CF03`, `TestValidateProjectDates_CF03`; psvc `TestCreate_Validation_CF03`; papi `TestProjectAPI_BadRequests`; plist "ngày kết thúc trước ngày bắt đầu bị chặn ở form" |
| 3 | Sửa thông tin + 5 trạng thái (Mới tạo/Đang tiến hành/Hoàn thành/Tạm dừng/Đã hủy); chỉ Manager phụ trách; Member/CEO → 403, Manager khác → 404; không gửi trạng thái thì giữ nguyên | dương/âm | pdom `TestParseProjectStatus_CF03`; psvc `TestUpdate_OnlyProjectManager_CF03`; papi `TestProjectAPI_PassesInputToService`; pdet "hiện thông tin và nút Sửa cho Manager phụ trách", "sửa thông tin gửi PUT với trạng thái mới", "Member chỉ xem: không có nút sửa…" |
| 4 | Thêm/bỏ thành viên: chỉ vai trò Manager/Member; Admin, tài khoản bị khóa, người không tồn tại → 422; thêm lại idempotent; không bỏ được Manager phụ trách; Member không thêm/bỏ | dương/âm | psvc `TestMembers_CF03`; pit `TestProjectRepo_ServiceFlow_CF03`; papi `TestProjectAPI_HappyPaths_CF03`, `TestProjectAPI_ErrorMapping`; pdet "Manager thêm thành viên từ danh sách ứng viên", "Manager bỏ thành viên qua hộp xác nhận…", "Member không thấy nút thêm/bỏ" |
| 5 | Cột Kanban 2–8 cột: thêm, đổi tên, xóa, sắp xếp; nhóm `todo`/`in_progress`/`done`; luôn còn ≥ 1 cột done; trùng tên → 409; xóa cột còn việc → `COLUMN_HAS_TASKS` | dương/âm/biên | pdom `TestDefaultColumns_CF03`, `TestNormalizeColumnName_CF03`, `TestParseStatusGroup_CF03`, `TestValidateColumnSpecs_CF03`, `TestValidateColumnOrder_CF03`, `TestCheckColumnDeletable_CF03`; psvc `TestAddColumn_CF03`, `TestUpdateColumn_PrunesMemberIntoDone_CF03`, `TestDeleteColumn_CF03`, `TestReorderColumns_CF03`; pit `TestProjectRepo_ServiceFlow_CF03`; pdet "đổi thứ tự gửi columnIds mới", "xóa cột đang có việc: cảnh báo…", "thêm cột gửi tên và nhóm trạng thái" |
| 6 | Luật chuyển `workflow_transition`: mặc định khi tạo dự án/thêm cột, đọc + cập nhật qua API; Member không vào nhóm done; cùng cột/cột lạ/vai trò sai/trùng → 422; CEO không sửa | dương/âm | pdom `TestDefaultTransitions_CF03`, `TestValidateTransitions_CF03`, `TestPruneTransitions_CF03`, `TestTransitionsForNewColumn_CF03`; psvc `TestReplaceTransitions_CF03`; pit `TestProjectRepo_ServiceFlow_CF03`; papi `TestProjectAPI_PassesInputToService`; pdet "ô Member vào/ra nhóm Hoàn thành bị khóa", "lưu luật chuyển gửi danh sách mới", "Member xem luật chuyển ở chế độ chỉ đọc" |
| 7 | Danh sách theo vai trò: Member chỉ dự án mình tham gia, Manager dự án mình phụ trách, CEO tất cả (chỉ xem), Admin → 403; không phải thành viên → 404; lọc/tìm kiếm, phân trang B9 | dương/âm/biên | psvc `TestList_ByRole_CF03`, `TestList_FilterValidation_CF03`, `TestGet_Visibility_CF03`; papi `TestProjectAPI_PassesInputToService`, `TestProjectAPI_ResponseShape`, `TestProjectAPI_BadRequests`; pit `TestProjectRepo_ServiceFlow_CF03`; plist "Manager thấy bảng dự án và nút Tạo dự án", "trạng thái rỗng theo vai trò", "lỗi tải có nút Thử lại", "lọc theo trạng thái gửi status lên server", "Admin không vào được màn dự án"; pdet "404 hiện trạng thái không tìm thấy" |

**Khoảng trống đã biết:** (1) ~~Lệch tiêu chí 6~~ — đã sửa ở gate vòng 1: server chặn Member chuyển cả *vào* và *ra* nhóm done (pdom "Member mở lại từ done về todo bị chặn"); thêm psvc `TestWriteOps_ForbiddenForNonManager_CF03`, `TestColumnOps_CrossProjectColumn_CF03` (IDOR mã cột dự án khác). (2) `pit` chạy cục bộ trên Postgres Homebrew (B12) qua `TEST_DATABASE_URL` → ok; CI chạy theo workflow. (3) Chưa có test FE cho CEO chỉ xem ở màn chi tiết; chưa có ca thêm CEO làm thành viên. (4) Chưa có E2E Playwright tự động (đã kiểm tay bằng Playwright MCP, ảnh ở `docs/testing/evidence/cf03-*.png`).
