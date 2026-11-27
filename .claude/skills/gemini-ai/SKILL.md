---
name: gemini-ai
description: Tích hợp Google Gemini cho Trợ lý AI của WorkTrack — phân rã yêu cầu thành công việc cha, gợi ý Top 3 nhân sự, cảnh báo trễ hạn. Dùng khi viết port/adapter AI, prompt, parse JSON đầu ra, timeout, ghi ai_usage_log hoặc UI bản xem trước.
---

# Trợ lý AI (Gemini)

## Kiến trúc
- Port trong application: `type AIAssistant interface { Decompose(ctx, in) (DecomposeResult, error); SuggestAssignees(ctx, in) ([]Suggestion, error) }`.
- Adapter `infrastructure/gemini` là cài đặt duy nhất. Nhà cung cấp khác: **chỉ thiết kế cổng, không cài** (ngoài phạm vi).
- Dùng SDK Go chính thức của Google cho Gemini — tra tên module và cách gọi hiện hành bằng Context7/tài liệu ai.google.dev, **không đoán**. Model và API key đọc từ cấu hình (Admin chỉnh tham số AI), không hard-code.

## Quy tắc bắt buộc
1. **Bản xem trước:** use case trả kết quả cho UI; **không ghi** công việc vào CSDL. Chỉ khi người dùng bấm "Import vào Kanban" mới gọi use case tạo việc bình thường (vẫn qua phân quyền).
2. **Timeout** (vd. 15 giây) bằng `context.WithTimeout`. Hết giờ → lỗi `AI_TIMEOUT` với thông điệp tiếng Việt rõ ràng.
3. **Ghi `ai_usage_log` mỗi lần gọi** — kể cả lỗi: loại tác vụ, model, thời gian phản hồi (ms), thành công/thất bại, mã lỗi. Không lưu nội dung nhạy cảm.
4. **Đầu ra có cấu trúc:** yêu cầu JSON theo schema; parse + validate; sai schema → thử lại tối đa 1 lần rồi báo lỗi. Không tin số liệu AI trả mà không kiểm tra (Story Point phải là số dương hợp lệ).
5. **Không đưa dữ liệu nhạy cảm** (email, SĐT, mật khẩu) vào prompt. Gợi ý nhân sự gửi mã ẩn danh + kỹ năng + tải việc, ánh xạ lại tên ở server.
6. Giới hạn tần suất gọi theo người dùng.

## Phân rã yêu cầu
Đầu vào: văn bản yêu cầu. Đầu ra: danh sách **công việc cha** `{title, description, storyPoint}` — không sinh công việc con.

## Gợi ý nhân sự
Đầu vào: công việc + danh sách ứng viên (kỹ năng + mức, số việc cha đang làm, giờ trống). Đầu ra: Top 3 `{userRef, reason}`. Lý do phải dựa trên dữ liệu đã gửi.

## Kiểm thử
- Unit test adapter với HTTP giả (`httptest`) — không gọi Gemini thật trong CI.
- Test parse: JSON đúng, thiếu trường, Story Point âm, rác.
- KPI 4: đo ≥ 30 lần phân rã thật trên môi trường triển khai; xuất từ `ai_usage_log`, tính tỷ lệ < 10 giây.
