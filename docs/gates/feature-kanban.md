# Gate: feature/kanban — CF-04, CF-05 (UC05: bảng Kanban kéo thả, giới hạn WIP, cập nhật tức thời)
**Ngày:** 2026-11-21 · **Commit cuối (code):** aa5f2fd · **Kết luận:** ĐẠT (kèm ghi chú, xem "Còn tồn đọng", issue #9)

Phạm vi:
- **Backend:**
  - migration `20261120204233_add_kanban_wip_version_history.sql`: `project.wip_limit` (1–20, mặc định 5), `task.version`, bảng `task_move_history`;
  - quy tắc thuần `domain/kanban.go`: `CheckMove` (luật `workflow_transition` theo vai trò), `EntersWIP` / `CheckWIP` (chỉ việc cha), `InsertAt`;
  - use case `application/kanban`: khóa dự án → khóa việc → so version → kiểm luật → kiểm WIP trên số đã khóa → ghi → lịch sử → phát sự kiện sau commit;
  - endpoint: `GET /projects/{id}/board/tasks`, `POST /tasks/{id}/move`, `PUT /projects/{id}/wip-limit`, WebSocket `/projects/{id}/ws`;
  - hub realtime (`coder/websocket`) theo phòng dự án, có presence, bỏ client chậm.
- **Frontend (`features/kanban/`):** bảng theo vai trò (Manager / Member / chỉ xem), chặn có lý do trước khi gửi, cập nhật lạc quan + rollback, toast "Hoàn tác", hộp thoại WIP đầy, dải mất kết nối, chỉ báo người đang xem, thông báo trợ năng tiếng Việt.
- **Không làm ở nhánh này:** việc con trên bảng (CF-07), Log time (CF-08, thanh giờ đang hiện 0 giờ), nghiệm thu (CF-09).

Gate chạy 2 vòng:
- **Vòng 1:** toàn bộ lệnh tự động, go-reviewer, qa-tester, ui-reviewer.
  - go-reviewer: 0 Chặn. Phát hiện kiểm quyền/WIP trên bản đọc trước transaction → sửa ở 63be5b1.
  - qa-tester: thiếu ca 401, IDOR, body/path sai, ma trận vai trò đổi WIP → bổ sung ở 801a826.
  - ui-reviewer: 3 Chặn (thông báo trợ năng tiếng Anh, cột chưa đủ 6 thẻ ở 1366×768, trạng thái rỗng không có nút) → sửa ở aa5f2fd.
- **Vòng 2:** chạy lại toàn bộ lệnh tự động ở aa5f2fd, kiểm lại UX bằng Playwright MCP ở 1366×768. Tất cả xanh.

## Tự động
Môi trường: máy không có Docker (B12). PostgreSQL và Mailpit chạy bằng Homebrew; integration test chạy cục bộ qua `TEST_DATABASE_URL` trên CSDL riêng `worktrack_it_kanban`. `docker build` và `make test-integration` (testcontainers) chạy trong CI — kết quả ở mục "CI".

| Lệnh | Kết quả |
|---|---|
| `make lint` (golangci-lint + eslint + `tsc -b` + prettier) | ✅ 0 lỗi |
| `make test` (go test + vitest) | ✅ Go: 684 test/subtest pass · FE: 220/220 test pass (22 file) |
| `go test -tags=integration ./internal/infrastructure/postgres/` (cục bộ) | ✅ pass, gồm 5 ca `TestKanbanRepo_*` |
| `make cover` (gói cốt lõi) | ✅ 90,1% (ngưỡng 70%) |
| `gitleaks detect --no-banner` | ✅ 0 secret |
| `go vet ./...`, `go vet -tags=integration ./...` | ✅ sạch |
| `npx tsc --noEmit` | ✅ 0 lỗi |
| `python3 scripts/ai-log/ai_log.py check` | ✅ OK |
| KPI 1 — độ trễ đồng bộ 2 client (`scripts/kpi/kanban-sync-latency.mjs`) | ✅ 40 lượt, p95 = 10 ms (ngưỡng 2000 ms), `docs/testing/evidence/kpi1-kanban-sync-latency.md` |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|
| Không token → 401 | 3 endpoint REST + WS | ✅ | `TestKanbanAPI_AllEndpointsRequireLogin_CF04_CF05`, `TestKanbanAPI_RequiresLogin_CF04`, `TestWS_RejectsBeforeUpgrade_CF04` |
| Sai vai trò → 403/404 | Member kéo việc người khác, Member sang nhóm Done, Admin/CEO kéo, Admin xem bảng, đổi WIP với Admin/CEO/Member/Manager khác | ✅ | `TestMove_WhoCanDrag_CF04`, `TestMove_MemberCannotTouchDoneGroup_CF04`, `TestBoard_Visibility_CF04`, `TestSetWIPLimit_ManagerOnly_CF05`, `TestKanbanAPI_SetWIPLimit_ForbiddenMapping_CF05` |
| IDOR | Người ngoài dự án dùng ID hợp lệ; cột của dự án khác | ✅ 404, không lộ dữ liệu | `TestKanbanAPI_OutsiderGetsNotFound_CF04`, `TestMove_ColumnOfAnotherProject_CF04` |
| WebSocket | Xác thực + kiểm thành viên trước khi nâng cấp; chỉ nhận sự kiện của dự án mình | ✅ | `TestWS_RejectsBeforeUpgrade_CF04`, `TestWS_MemberReceivesProjectEvents_CF04`, `TestHub_PublishOnlyToSameProject` |
| SQL | Toàn bộ qua sqlc, lọc theo dự án; khóa `FOR UPDATE` dự án → việc | ✅ | `LockProjectBoard`, `TestMove_UsesProjectStateReadUnderLock_CF04_CF05` |
| Ghi đè đồng thời | Version cũ → 409; hai lần kéo song song không vượt WIP | ✅ | `TestMove_StaleVersionConflicts_CF04`, `TestKanbanRepo_MoveTaskColumn_StaleVersionNoRows_CF04`, `TestKanbanRepo_ConcurrentMoves_WIPNotExceeded_CF05` |
| Validate input | UUID sai, JSON hỏng, sai kiểu, position âm/> 10000, WIP ngoài 1–20 | ✅ 400 + thông điệp tiếng Việt | `TestKanbanAPI_BadPathAndBody_CF04_CF05`, `TestKanbanAPI_Move_BadRequests_CF04`, `TestSetWIPLimit_Range_CF05`, `TestValidateWIPLimit_CF05` |
| Quy tắc nghiệp vụ | Luật chuyển cột thiếu → chặn; WIP chỉ đếm việc cha; việc con không lên bảng; đổi người nhận / nhận việc cũng kiểm WIP | ✅ | `TestMove_RuleMissingIsRejected_CF04`, `TestMove_WIPCountsOnlyParentTasks_CF05`, `TestKanbanRepo_CountProjectWIP_ExcludesSubtasks_CF05`, `TestMove_SubtaskNotOnBoard_CF04`, `TestClaim_BlockedWhenWIPFull_CF05`, `TestUpdate_ReassignInProgressChecksNewAssigneeWIP_CF05` |
| FE và BE cùng luật | `dropRules` mô phỏng luật server; BE vẫn chặn khi gọi thẳng API | ✅ | `dropRules.test.ts`, `TestKanbanAPI_Move_ErrorMapping_CF04_CF05` |
| UX theo vai trò | Manager: kéo mọi thẻ, nút "+", ô WIP · Member: chỉ kéo thẻ của mình, chip WIP · CEO: chỉ xem | ✅ | `KanbanBoard.test.tsx`; ảnh `cf04-manager-board.png`, `cf04-member-board.png` |
| Chặn có lý do | Member kéo sang nhóm hoàn thành → cột mờ + câu lý do; WIP đầy → hộp thoại | ✅ | `useBoardDrag.test.tsx`; ảnh `cf04-member-drag-done-blocked.png`, `cf04-manager-dragging.png` |
| Trạng thái | Skeleton, rỗng (có nút theo vai trò), lỗi + Thử lại, toast + Hoàn tác, mất kết nối | ✅ | `KanbanBoard.test.tsx`; ảnh `cf04-manager-moved-toast.png` |
| Thất bại khi lưu | 409 WIP, 409 version, lỗi mạng/5xx → thẻ về cột cũ + dải lỗi | ✅ | `useBoardDrag.test.tsx` |
| Bàn phím / trợ năng | Kéo bằng phím cách + mũi tên; câu đọc cho trình đọc màn hình tiếng Việt | ✅ | `a11y.test.ts`, `KanbanBoard.test.tsx` |
| Mật độ 1366×768 | ≥ 6 thẻ/cột | ✅ 6 thẻ trọn vẹn + một phần thẻ thứ 7 | ảnh `cf04-manager-board.png` |

Ảnh minh chứng: `docs/testing/evidence/cf04-*.png` (chụp bằng Playwright MCP, 1366×768, dữ liệu `make seed`).

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|
| 1 | Kiểm quyền Manager và giới hạn WIP dựa trên bản đọc trước transaction; đổi giới hạn WIP song song có thể lọt | Cao | 63be5b1 |
| 2 | Thiếu test 401 / IDOR / body sai / ma trận vai trò đổi WIP | Trung bình | 801a826 |
| 3 | Câu thông báo kéo thả cho trình đọc màn hình là tiếng Anh (mặc định dnd-kit) | Chặn (UI) | aa5f2fd |
| 4 | Cột chỉ hiện ~4 thẻ ở 1366×768 (tiêu đề xuống 2 dòng, chỗ trống khi kéo lệch cỡ) | Chặn (UI) | aa5f2fd |
| 5 | Trạng thái rỗng không có nút hành động | Chặn (UI) | aa5f2fd |

## Còn tồn đọng → GitHub Issue #9
Không có lỗi Critical/Blocker. Các mục nhỏ gom ở issue #9:
- WebSocket chưa tự đóng khi token hết hạn hoặc bị gỡ khỏi dự án; token đi qua query string (ghi ở `docs/decisions.md` mục C).
- Gợi ý Go: `SetTaskPositions` lọc thêm `project_id`, đọc lại thẻ trong transaction, test CHECK `ck_task_story_point_parent_only`.
- UI Nên sửa: `aria-disabled` kèm lý do, `coordinateGetter` cho bàn phím, dải lỗi, `WipLimitControl` 40px, nút đóng dải 40px, tên cột mặc định tiếng Anh.
- Test `useProjectSocket`, `WipLimitControl`; E2E Playwright.
- UC05 chưa có acceptance criteria chính thức.

## CI
Đang chờ — cập nhật sau khi push.
