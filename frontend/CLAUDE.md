# frontend/CLAUDE.md — WorkTrack Web (React + TypeScript)

@../docs/design/design-system.md

Đọc thêm skill `react-feature` (mẫu code).

## Stack
React 19 · TypeScript strict · Vite · TailwindCSS v4 · React Router

**Đề xuất (chờ người dùng xác nhận khi scaffold, không tự đổi sau khi đã chọn):**
TanStack Query v5 (server state) · Zustand (UI state thuần) · react-hook-form + zod (form) · dnd-kit (Kanban kéo thả) · lucide-react (icon) · Vitest + React Testing Library + MSW (test) · Playwright (E2E)

**Biểu đồ:** bản thiết kế chỉ dùng thanh ngang và đường đơn giản → tự dựng `shared/ui/charts` (`BarRows` bằng div, `LineChart` bằng SVG, có `role="img"` + `aria-label`). Chỉ thêm thư viện biểu đồ nếu cần tooltip/tương tác phức tạp và người dùng đồng ý.

## Thiết kế
- Mở artboard tương ứng trong `docs/design/` trước khi làm màn hình (bảng `docs/design/design-system.md` §9). Chép bố cục, khoảng cách, câu chữ; **không chép HTML inline style**.
- `shared/ui` dựng đúng danh sách component ở `docs/design/design-system.md` §5–§7 **trước** khi làm màn hình đầu tiên: `Button`, `Input`, `Badge`, `LabelChip`, `FilterChip`, `Kbd`, `DataTable`, `StatCard`, `ProgressBar`, `HoursBar` + `HoursLevelMark`, `TaskCard`, `Avatar`/`AvatarStack`, `Toast`, `Dialog`, `Banner`, `InfoBox`, `EmptyState`, `ErrorState`, `Skeleton`, `AiSuggestion`, `AppShell` (Sidebar + Topbar).
- `shared/lib/format.ts`: `formatHours`, `formatNumber`, `formatPercentPoint`, `formatDate`, `formatDateRange` — `Intl` với `vi-VN`. `shared/lib/hoursLevel.ts`: `getHoursLevel(actual, estimated)` → `"ok" | "warn" | "bad"`. Có unit test cho cả hai.

## Cấu trúc (feature-based)

```
src/
  app/          router, providers (QueryClient, auth), styles/theme.css
  shared/       api/ (http client, lỗi), ui/ (Button, Badge, Table, Modal, Skeleton, EmptyState…), lib/
  features/
    auth/ projects/ kanban/ tasks/ timesheet/ reports/ ai/ admin/
      api/          hàm gọi API + query key + hook useQuery/useMutation
      components/   component + *.test.tsx cạnh nhau
      hooks/  model/ (type, schema zod)
  pages/        ghép feature thành trang theo route
```

Feature không import sâu vào feature khác; chỉ qua `features/<x>/index.ts`.

## Quy tắc cứng
- 100% dữ liệu server qua TanStack Query. Zustand chỉ cho UI state (sidebar, bước form nhiều trang, bộ lọc).
- Access token chỉ giữ trong bộ nhớ (không `localStorage`). Refresh token nằm trong cookie HttpOnly do backend đặt.
- Không `any`. Type DTO khớp API; parse response bằng zod ở biên khi dữ liệu quan trọng (giờ, Story Point).
- Không hard-code màu/hex, cỡ chữ lẻ (`text-[13px]`), bo góc > 6px trong component — chỉ dùng token trong `theme.css`.
- Không định dạng số/ngày bằng tay — luôn qua `shared/lib/format.ts`.
- Mọi chữ hiển thị tiếng Việt, đặt trong `shared/lib/messages.ts` hoặc cạnh feature; không rải chuỗi lặp.
- Ẩn/vô hiệu hóa theo vai trò qua một hook `useCan(action, resource)` — nhưng nhớ: backend mới là nơi chặn thật.
- Mỗi màn hình: skeleton, empty state, error state, toast thành công.
- Kanban realtime: cập nhật cache TanStack Query từ sự kiện WebSocket (`queryClient.setQueryData`), tự kết nối lại khi mất mạng.

## Lệnh
```bash
npm run dev
npm run test         # vitest run
npm run lint         # eslint
npx tsc --noEmit
npx playwright test
```
