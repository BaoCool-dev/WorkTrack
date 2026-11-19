## Mục đích
<!-- Một câu: PR này làm gì -->

**Mã yêu cầu:** CF-.. / UC.. / YCCN-..
**Issue:** closes #

## Thay đổi chính
-

## Cách kiểm thử
- [ ] `make lint` xanh
- [ ] `make test` xanh
- [ ] Test tích hợp (nếu đụng CSDL): `make test-integration`
- [ ] Kiểm thử tay luồng chính: <mô tả>

## Checklist tự review (thay cho reviewer thứ hai — bản cam kết M23)
- [ ] Phân quyền kiểm tra ở backend, có test 403
- [ ] Đúng quy tắc nghiệp vụ (`worktrack-domain`)
- [ ] Không sửa tay sqlcgen / migration đã merge
- [ ] Không secret, không log token/mật khẩu
- [ ] UI đúng `docs/design/design-system.md`, đủ trạng thái tải/rỗng/lỗi, chữ tiếng Việt
- [ ] Dependency mới (nếu có) kèm lý do: …
- [ ] Đã review bằng agent `go-reviewer` / `ui-reviewer` (dán tóm tắt bên dưới nếu có)

## AI Usage Log
- [ ] Mọi commit có AI trong PR đều có trailer `AI-Log`; `python3 scripts/ai-log/ai_log.py check` sạch
- [ ] Đã điền "Phần tôi đã sửa / hoàn thiện" trong entry (không để `[NGƯỜI DÙNG ĐIỀN]`)
