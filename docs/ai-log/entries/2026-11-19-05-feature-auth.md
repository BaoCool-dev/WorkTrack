---
title: feature auth
date: 2026-11-19
tool: Claude Code
model: claude-opus-5-5
surface: terminal
type: code
scope: CF-01
---
<!-- type: code | test | docs | design | review | debug | research
     surface: terminal | vscode | claude.ai | claude-design | khác
     Thêm dòng "commit: none" ở phần đầu nếu việc này không tạo commit (vd. thiết kế, soạn báo cáo ngoài repo). -->

## Mục tiêu
<!-- 1–2 câu: định làm gì, gắn mã yêu cầu nào -->
Làm trọn CF-01 (xác thực và phân quyền) ở chế độ tự chạy: bảng tài khoản/vai trò/refresh token, Admin tạo tài khoản → email kích hoạt (B7) → đặt mật khẩu (B8), đăng nhập JWT access + refresh xoay vòng (B4), đăng xuất, `/auth/me`, gói `authz` duy nhất quyết định quyền (B2), giới hạn tần suất; frontend đăng nhập, kích hoạt, AuthProvider, route guard, AppShell, shared/ui; `make seed`, `make dev-local`.

## Prompt đã dùng
<!-- Dán NGUYÊN VĂN prompt chính (lấy từ: python3 scripts/ai-log/ai_log.py prompts --since-last-ai-commit).
     Mỗi prompt một dòng bắt đầu bằng ">". Dài quá thì rút gọn và ghi [rút gọn]. Không dán secret. -->
> Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt. Nhánh hiện tại: feature/auth (mọc từ chore/scaffold). PR sẽ tạo với --base chore/scaffold --draft. Môi trường: Máy KHÔNG có Docker (B12)… Phạm vi nhánh này: CF-01. Backend: bảng app_user, role, refresh_token; Admin tạo tài khoản → email kích hoạt qua Mailpit (B7) → đặt mật khẩu (B8); đăng nhập; JWT access + refresh xoay vòng (B4); đăng xuất; GET /api/v1/auth/me; khóa đăng nhập (B8); module internal/application/authz…; Frontend: trang đăng nhập, trang kích hoạt, AuthProvider, route guard, AppShell, shared/ui… make seed (B11). Test đủ 401/403 cho mọi vai trò. Quy trình bắt buộc: 1… 6. … Cuối cùng in đúng một dòng: GATE=ĐẠT hoặc GATE=CHƯA_ĐẠT. [rút gọn — nguyên văn trong scripts/auto-run.sh]

## Nội dung AI tạo ra
<!-- Liệt kê theo file/phần: AI viết gì. Có thể kèm git diff --stat của commit AI. -->
- **Lát 1 — CSDL + domain:** migration `20261119092421_create_auth_tables.sql` (bảng `role` có sẵn 4 vai trò, `app_user` với ràng buộc trạng thái/mật khẩu, token kích hoạt lưu SHA-256, `refresh_token` theo họ `family_id`); truy vấn sqlc `app_user.sql`, `refresh_token.sql`; override uuid trong `sqlc.yaml`; `domain/user.go` (vai trò, chính sách mật khẩu B8, khóa 15 phút sau 5 lần sai, chuẩn hóa email), `domain/errors.go`, test bảng `user_test.go`.
- **Lát 2 — application:** `application/authz` (ma trận quyền theo vai trò, CEO kế thừa quyền Admin + xem mọi dự án/Dashboard công ty — B2; `Can/Require/PermissionsOf`), `application/securetoken` (token ngẫu nhiên 32 byte + SHA-256), `application/auth` (port + use case Login/Refresh xoay vòng phát hiện dùng lại → thu hồi cả họ/Logout/Me/InspectActivation/Activate), `domain/page.go`, `domain/refresh_token.go`; truy vấn `ListUsers`/`CountUsers`; test với port giả lập (auth 91,6%, authz 100%).

## Phần tôi đã sửa / hoàn thiện
[NGƯỜI DÙNG ĐIỀN]
<!-- Người dùng tự viết: đã đọc gì, sửa gì, quyết định gì khác với AI, vì sao.
     Bằng chứng khách quan: sau commit Human-Edit, chạy  ai_log.py human-diff <file này>  -->

## Lỗi / ảo giác của AI
| Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |
|---|---|---|---|
| Test dùng chuỗi `"hash:matkhau123"` làm PasswordHash bị gosec G101 báo “hardcoded credentials” | AI viết literal trông giống mật khẩu trong test | `golangci-lint run` | Đổi sang hằng `testPassword` + hàm `fakeHash()` |

## Kiểm chứng
<!-- Lệnh đã chạy và kết quả thật: make lint, make test, test nào, kiểm thử tay luồng nào. -->
