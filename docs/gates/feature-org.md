# Gate: feature/org — CF-02 (phòng ban dạng cây, chức danh, danh mục kỹ năng, hồ sơ kỹ năng nhân sự; màn Nhân sự & phòng ban)
**Ngày:** 2026-11-20 · **Commit cuối (code):** 801c3f8 · **Kết luận:** ĐẠT (kèm ghi chú — xem "Còn tồn đọng")

Phạm vi:
- **Backend:**
  - bảng `department` (cây, `parent_id`), `job_title`, `skill`, `user_skill` (mức Beginner/Intermediate/Senior);
  - cột `department_id`, `job_title_id` trên `app_user` (goose + sqlc);
  - quy tắc thuần ở `domain/org.go` (chuẩn hóa tên, chặn vòng, mức kỹ năng);
  - use case `application/org` và phần hồ sơ trong `application/account`;
  - API `/departments`, `/job-titles`, `/skills`, `GET/PUT /users/{id}`, `PUT /users/{id}/skills`, `GET /users` có lọc và phân trang (B9);
  - `make seed` thêm phòng ban, chức danh, kỹ năng mẫu.
- **Frontend:**
  - màn Nhân sự & phòng ban cho Admin/CEO, gồm các tab Nhân sự, Phòng ban, Chức danh, Kỹ năng;
  - tìm kiếm, lọc, phân trang;
  - form thêm/sửa có validate và báo "Email đã tồn tại";
  - hộp thoại hồ sơ kỹ năng;
  - component dùng chung `Select`, `Pagination`, `FormAlert` và hàm `format.ts`.
- **Không làm:** nhập/xuất Excel (thuộc phần Mở rộng).

Gate chạy 2 vòng:
- **Vòng 1:** go-reviewer, ui-reviewer, test tự động. Các lỗi tìm thấy đã sửa ở 801c3f8.
- **Vòng 2:** chạy lại toàn bộ lệnh tự động. Không còn mục Chặn.

## Tự động
Môi trường: máy không có Docker (B12).
- PostgreSQL và Mailpit chạy bằng Homebrew.
- Integration test chạy trên PostgreSQL local, CSDL `worktrack_gate`, mỗi lần dùng một schema tạm `it_*`.
- `docker build` và `make test-integration` (testcontainers) chỉ chạy trong CI.

| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint + eslint + `tsc -b` + prettier) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: 350 test/subtest pass ở mọi gói · FE: 107/107 test pass (10 file) |
| `go test -tags integration ./internal/infrastructure/postgres/` (PostgreSQL local) | ✅ ok 0,704s — gồm `TestOrgRepo_DepartmentTree_CF02`, `TestOrgRepo_CatalogsAndSkills_CF02` |
| `make cover` (gói cốt lõi domain + application) | ✅ 92,4% (gói `application/org` 91,1%) |
| `gitleaks detect --no-banner` | ✅ no leaks found |
| `go vet ./...` (kể cả tag `integration`) | ✅ |
| `python3 scripts/ai-log/ai_log.py check` | ✅ OK; cảnh báo còn lại: mục "Phần tôi đã sửa" chưa điền |
| `docker build`, `make test-integration` (CI) | xem mục CI bên dưới |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Xác thực | Không token → 401 trên mọi endpoint mới | ✅ | `TestOrgAPI_RoleMatrix_CF02` |
| Phân quyền | Admin/CEO quản lý phòng ban, chức danh, kỹ năng, hồ sơ; Manager/Member → 403 (cả đọc, mục C); Admin không sửa/gán CEO; không ai tự đổi vai trò của mình | ✅ | `TestService_OnlyAdminAndCEOManageOrg_CF02`, `TestOrgAPI_RoleMatrix_CF02`, `TestUpdateUser_Rules_CF02`; view "Admin không có nút Sửa ở dòng của CEO" |
| IDOR | Chưa có tài nguyên theo dự án; dữ liệu tổ chức chỉ mở cho Admin/CEO; ID không tồn tại → 404 | ✅ / N/A | `TestOrgAPI_DomainErrorMapping_CF02`, `TestUserAPI_UpdateErrors_CF02` |
| SQL / lỗi 500 | Mọi truy vấn qua sqlc, tham số hóa (tìm kiếm ILIKE qua tham số); CTE đệ quy dùng `UNION` nên tự dừng nếu cây có vòng; lỗi 500 trả câu chung | ✅ | `backend/db/queries/*.sql`; go-reviewer |
| Toàn vẹn cây | Chọn chính nó hoặc con cháu làm cha → 422 `DEPARTMENT_CYCLE`; dữ liệu hỏng có vòng vẫn dừng; xóa phòng ban còn con/còn người → 409 (không xóa lan) | ✅ | `TestValidateDepartmentParent_CF02`, `TestValidateDepartmentParent_CorruptLoopTerminates_CF02`, `TestUpdateDepartment_RejectsCycle_CF02` |
| Validate | Tên rỗng, chỉ khoảng trắng, quá dài, có dấu tiếng Việt; trùng tên → 409; mức kỹ năng lạ → 422; quá 50 kỹ năng; UUID sai → 400; trường `email` trong PUT → 400; FE dùng cùng luật (zod) | ✅ | `TestNormalizeOrgName_CF02`, `TestParseSkillLevel_CF02`, `TestValidateUserSkills_CF02`, `TestOrgAPI_BadInput_CF02`, `TestUserAPI_UpdateRejectsEmailChange_CF02`, `model.test.ts` |
| HTML/script trong text | Hiển thị qua React (escape mặc định), không dùng `dangerouslySetInnerHTML` | ✅ | đọc mã |
| UX — luồng Admin | Danh sách nhân sự, lọc, phân trang, thêm tài khoản; email trùng (`ADMIN@worktrack.local`) → "Email đã tồn tại" dưới ô email (API 409) | ✅ | Playwright MCP ở 1366×768: `docs/testing/evidence/cf02-users-list.png`, `cf02-duplicate-email.png`; console chỉ có 401 refresh trước đăng nhập và 409 dự kiến |
| UX — trạng thái | Skeleton, rỗng, không khớp bộ lọc (gợi ý xóa bộ lọc), lỗi + mã lỗi + Thử lại, toast thành công, lỗi gửi lại kích hoạt ở dải lỗi (không toast), ô vai trò bị khóa có lý do | ✅ | `AdminOrgView.test.tsx` (20 test) |
| Gửi trùng | Nút gửi khóa khi đang gửi; `PUT /skills` thay toàn bộ (lặp lại không tạo bản ghi trùng) | ✅ | `TestSetUserSkills_CF02`; view "thêm kỹ năng… → PUT danh sách đầy đủ" |
| Agent | go-reviewer: không có mục Chặn · ui-reviewer: 1 mục Chặn, đã sửa · ma trận truy vết đã cập nhật | ✅ | `docs/testing/traceability-matrix.md` (11 dòng CF-02) |

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | Lỗi "Gửi lại email kích hoạt" hiện bằng toast 3 giây, trái design-system §5 (ui-reviewer: Chặn) | Trung bình | 801c3f8 |
| 2 | CTE `dept_scope` dùng `UNION ALL` nên lặp vô hạn nếu cây phòng ban lỡ có vòng | Trung bình | 801c3f8 |
| 3 | Ô vai trò bị khóa khi tự sửa mình nhưng không hiện lý do (ui-reviewer) | Thấp | 801c3f8 |
| 4 | Tab và đầu trang lệch design-system §4: tab chọn chưa `accent` 600, h1 và meta không cùng hàng (ui-reviewer) | Thấp | 801c3f8 |
| 5 | Lỗi biên dịch test integration và fake sau khi mở rộng interface; 27 lỗi eslint; thiếu import chỉ `tsc -b` bắt được (trong quá trình làm, xem AI log) | Thấp | 23dc7d0, 8a378b3 |

## Còn tồn đọng
- **Người dùng cần làm:**
  - Điền mục "Phần tôi đã sửa / hoàn thiện" trong `docs/ai-log/entries/2026-11-20-01-feature-org.md` (đang để `[NGƯỜI DÙNG ĐIỀN]`).
  - Đưa artboard 10 (`10-admin-users-departments.html`) vào `docs/design/artboards/` rồi soát lại màn. Màn hiện dựng theo `design-system.md` vì thư mục chưa có artboard.
- **Đã hoãn và ghi ở `docs/decisions.md` mục C:**
  - Đổi cha phòng ban: kiểm vòng trên ảnh chụp cây, chưa có khóa hay `TxManager` (TOCTOU khi hai Admin đổi chéo cùng lúc). CTE dùng `UNION` nên truy vấn vẫn không treo.
  - Endpoint đọc dữ liệu tổ chức chưa mở cho Manager/Member.
- **go-reviewer — nên sửa, không chặn:**
  - Kiểm tra CEO giữa `targetUser` và `UpdateProfile` chưa nằm trong một transaction. Chung nguyên nhân với mục `TxManager` ở trên.
  - `user_handler` trả lỗi chung `VALIDATION` cho vai trò/trạng thái lạ, thay vì một mã riêng.
  - Chưa có integration test cho `skills: []`.
  - Gợi ý thêm:
    - tag `max=200` lệch với giới hạn 100 rune của họ tên (BE vẫn kiểm theo rune);
    - ILIKE phụ thuộc locale;
    - thêm ca API;
    - thêm mã UC vào tên test khi có UC chính thức;
    - bọc lỗi chưa đồng nhất.
- **ui-reviewer — nên sửa, không chặn:**
  - cảnh báo `createdNoEmail` (tạo được tài khoản nhưng gửi email lỗi) vẫn hiện bằng toast;
  - hàng bảng cao hơn 40px khi có kỹ năng;
  - 3 nút thao tác trên một hàng có thể tràn;
  - `ConfirmDeleteDialog` thiếu icon mức độ;
  - kích thước skeleton chưa khớp hàng thật;
  - `ErrorState` thiếu nút "Báo lỗi";
  - `Dialog` chưa bẫy focus (đã hoãn từ feature/auth);
  - tên kỹ năng hiển thị `text-sm`.
  - Gợi ý thêm:
    - phím mũi tên cho tab;
    - `aria-level` trên cây;
    - tổng số hiện hai lần;
    - nhãn mức kỹ năng tiếng Anh (Beginner/Intermediate/Senior, giữ theo yêu cầu CF-02).
- **Khoảng trống test:** chưa có E2E Playwright tự động; đã đi tay luồng bằng Playwright MCP.
