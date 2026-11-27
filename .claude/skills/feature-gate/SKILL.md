---
name: feature-gate
description: Cổng kiểm tra cuối một nhánh tính năng WorkTrack trước khi mở PR và trước khi sang tính năng tiếp theo — chạy test, soát bảo mật, phân quyền, validate dữ liệu, thao tác người dùng và trạng thái giao diện, rồi xuất báo cáo kiểm tra. Dùng khi người dùng nói "xong tính năng", "kiểm tra trước khi merge", "test hết lỗ hổng", "chạy gate", hoặc trước khi tạo nhánh tính năng mới.
---

# Cổng kiểm tra tính năng (feature gate)

Quy ước dự án: **một tính năng = một nhánh = một PR**. Chưa qua cổng này thì **không** mở PR và **không** tạo nhánh tính năng tiếp theo.

Kết quả là file `docs/gates/<loai>-<ten>.md` (vd. `feature/kanban` → `feature-kanban.md`) (minh chứng cho kiểm thử M27–M32 và vấn đáp), commit cùng nhánh; phần tóm tắt dán vào mô tả PR. Báo cáo **kết quả thật** của lệnh đã chạy — không viết "chắc là pass".

## 0. Chuẩn bị
```bash
git switch <nhánh tính năng> && git status --short     # phải sạch hoặc đã commit
git fetch origin && git log --oneline origin/main..HEAD # các commit của tính năng
git diff --stat origin/main...HEAD
```
Xác định mã yêu cầu (CF/UC/YCCN) và acceptance criteria của tính năng từ `docs/requirements/`.

## 1. Tự động (bắt buộc xanh)
```bash
make lint
make test
make test-integration        # nếu tính năng đụng CSDL
make cover                   # gói cốt lõi ≥ 70%
gitleaks detect --no-banner  # 0 secret
cd backend && go vet ./... && cd -
cd frontend && npx tsc --noEmit && cd -
```
Có lệnh đỏ → dừng, sửa, chạy lại. Không tắt rule lint hay bỏ qua test để qua cổng.

## 2. Bảo mật (theo OWASP Top 10 và `.claude/claude-security-guidance.md`)
Với **mỗi endpoint mới/đổi** trong nhánh, kiểm tra và có test chứng minh:
- [ ] Không token → 401. Token hết hạn/sai chữ ký/`alg: none` → 401.
- [ ] Sai vai trò → 403 (thử đủ Admin, Manager, Member, CEO theo ma trận trong `worktrack-domain`).
- [ ] **IDOR:** người không thuộc dự án truy cập bằng ID hợp lệ của dự án khác → 403/404, không lộ dữ liệu.
- [ ] Truy vấn đều qua sqlc, có lọc theo dự án/quyền; không nối chuỗi SQL.
- [ ] Không log token, mật khẩu, API key; lỗi 500 không lộ stack trace hay câu SQL.
- [ ] Endpoint nhạy cảm (đăng nhập, refresh, gọi AI) có giới hạn tần suất.
- [ ] Upload (nếu có): giới hạn kích thước, kiểm tra loại file ở server.
- [ ] WebSocket (nếu có): xác thực + kiểm tra thành viên trước khi nhận kết nối.

## 3. Validate dữ liệu
Với mỗi input (body, path, query) — có test cho từng ca:
- [ ] Thiếu trường bắt buộc, sai kiểu, chuỗi rỗng, chỉ khoảng trắng.
- [ ] Biên: 0, âm, rất lớn, vượt độ dài tối đa, ngày quá khứ/tương lai không hợp lệ.
- [ ] Ký tự đặc biệt và tiếng Việt có dấu, emoji; HTML/script trong text (hiển thị an toàn, không thực thi).
- [ ] ID không tồn tại, UUID sai định dạng.
- [ ] Quy tắc nghiệp vụ (WIP, Log time > 0 và ≤ 24h sau Done, chỉ Manager sang nhóm hoàn thành, con không có SP…) bị vi phạm → mã lỗi đúng + thông điệp tiếng Việt.
- [ ] FE và BE validate **cùng** quy tắc; BE vẫn chặn khi gọi API trực tiếp bỏ qua FE.

## 4. Thao tác người dùng (UX)
Chạy app (`make dev`) và đi qua từng luồng theo artboard trong `docs/design/artboards/` (dùng Playwright MCP nếu có, chụp ở 1366×768):
- [ ] Luồng chính hoàn thành được cho **từng vai trò** liên quan.
- [ ] Đủ trạng thái: đang tải (skeleton), rỗng, lỗi + Thử lại, thành công (toast), không có quyền.
- [ ] Nút không có quyền bị ẩn; nút tạm khóa có lý do hiển thị.
- [ ] Bấm nhanh 2 lần / gửi trùng không tạo bản ghi trùng; thao tác thất bại thì UI trở về đúng trạng thái.
- [ ] Mất mạng / API chậm / API trả 500: thông báo rõ, không treo, không mất dữ liệu đã nhập.
- [ ] Bàn phím: Tab đi đúng thứ tự, Enter gửi form, Esc đóng hộp thoại, focus ring nhìn thấy.
- [ ] Gọi agent `ui-reviewer` cho các file `.tsx` của nhánh.

## 5. Review mã
- [ ] Agent `go-reviewer` không còn mục **Chặn**.
- [ ] Agent `qa-tester`: mọi acceptance criteria của tính năng có test (ca dương + âm + biên), cập nhật `docs/testing/traceability-matrix.md`.
- [ ] Mọi commit có AI đều có trailer `AI-Log`; `python3 scripts/ai-log/ai_log.py check` không báo lỗi; mục "Phần tôi đã sửa" đã điền.

## 6. Báo cáo cổng → `docs/gates/<loai>-<ten>.md` (vd. `feature/kanban` → `feature-kanban.md`)
```markdown
# Gate: feature/<tên> — <CF-xx, UCxx>
**Ngày:** YYYY-MM-DD · **Commit cuối:** <hash ngắn> · **Kết luận:** ĐẠT / CHƯA ĐẠT

## Tự động
| Lệnh | Kết quả |
|---|---|
| make lint | ✅ 0 lỗi |
| make test | ✅ 48 pass |
| make cover | ✅ 76,2% gói cốt lõi |
| gitleaks | ✅ 0 secret |

## Bảo mật / Validate / UX
| Hạng mục | Ca đã kiểm | Kết quả | Test hoặc minh chứng |
|---|---|---|---|

## Lỗi tìm thấy và đã sửa
| # | Mô tả | Mức | Commit sửa |
|---|---|---|---|

## Còn tồn đọng (nếu có) → GitHub Issue
```
Lỗi Critical/Blocker còn tồn → **CHƯA ĐẠT**. Lỗi nhỏ còn tồn → mở GitHub Issue (`gh issue create`), ghi số issue vào báo cáo.

## 7. Sau khi ĐẠT
1. Commit báo cáo: `docs(gate): kết quả kiểm tra feature/<tên> [CF-xx]` (kèm `AI-Log` nếu AI soạn báo cáo).
2. `git push -u origin <nhánh>` → hỏi người dùng trước khi push. Ở **chế độ tự chạy** (`CLAUDE.md` §7c): push luôn, không hỏi.
3. Mở PR → hỏi người dùng trước (chế độ tự chạy: không hỏi, thêm `--draft`, `--base <nhánh trước>`): `gh pr create --base main --title "<tiêu đề>" --body-file <file tạm chứa: mục tiêu, mã CF/UC, bảng Tự động + Bảo mật/Validate/UX từ báo cáo gate, lỗi đã sửa>`.
4. **Người dùng** tự review và merge. Sau khi merge: `git switch main && git pull`, rồi mới tạo nhánh tính năng tiếp theo từ `main` mới.
