---
name: react-feature
description: Mẫu code frontend WorkTrack (React 19, TypeScript strict, TanStack Query, Zustand, Tailwind v4, Vitest + RTL + MSW). Dùng khi tạo màn hình, component, form, hook gọi API, bảng dữ liệu, Kanban kéo thả, dashboard hoặc test giao diện.
---

# React feature — mẫu chuẩn

Luôn đọc `docs/design/design-system.md` trước khi viết JSX. Cấu trúc thư mục: `frontend/CLAUDE.md`.

## 1. API + query key (features/<x>/api/)

```ts
// features/timesheet/api/timeLogs.ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { http } from "@/shared/api/http";
import { taskKeys } from "@/features/tasks";

export type LogTimeInput = { taskId: string; minutes: number; note: string };

export function useLogTime() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ taskId, ...body }: LogTimeInput) =>
      http.post<TimeLogDto>(`/api/v1/tasks/${taskId}/time-logs`, body),
    onSuccess: (_data, { taskId }) => {
      qc.invalidateQueries({ queryKey: taskKeys.detail(taskId) });
    },
  });
}
```

Query key gom một chỗ mỗi feature:

```ts
export const taskKeys = {
  all: ["tasks"] as const,
  byProject: (projectId: string) => [...taskKeys.all, "project", projectId] as const,
  detail: (id: string) => [...taskKeys.all, "detail", id] as const,
};
```

## 2. Lỗi API
`shared/api/http.ts` ném `ApiError { status, code, message }` lấy từ body `{"error":{code,message}}`. UI hiển thị `message` (đã là tiếng Việt từ backend); switch theo `code` khi cần hành vi riêng (vd. `TASK_WIP_LIMIT` → vô hiệu hóa nút "Nhận việc").

## 3. Form (react-hook-form + zod)

```tsx
const schema = z.object({
  hours: z.number({ invalid_type_error: "Nhập số giờ" }).positive("Số giờ phải lớn hơn 0").max(24),
  note: z.string().max(500),
});
```
Đổi giờ → phút **trước khi gửi**: `minutes: Math.round(hours * 60)`.

## 4. Component — đủ trạng thái

```tsx
export function MyTasksTable() {
  const { data, isPending, isError, refetch } = useMyTasks();
  if (isPending) return <TableSkeleton rows={8} />;
  if (isError) return <ErrorState title="Không tải được danh sách việc" onRetry={refetch} />;
  if (data.items.length === 0)
    return (
      <EmptyState
        title="Backlog đang trống"
        hint="Chưa có công việc nào để nhận. Quản lý dự án sẽ đưa công việc mới vào Backlog."
        action={{ label: "Xem công việc của tôi", to: "/my-tasks" }}
      />
    );
  return <DataTable rows={data.items} columns={columns} />;
}
```

## 4b. Số liệu và mức giờ

```ts
// shared/lib/format.ts
const nf1 = new Intl.NumberFormat("vi-VN", { maximumFractionDigits: 1 });
export const formatHours = (minutes: number) => nf1.format(minutes / 60);          // 870 → "14,5"
export const formatHoursPair = (actualMin: number, estMin: number) =>
  `${formatHours(actualMin)}/${formatHours(estMin)}h`;                             // "14,5/16h"
export const formatNumber = (n: number) => new Intl.NumberFormat("vi-VN").format(n); // 5480 → "5.480"

// shared/lib/hoursLevel.ts — phải khớp ngưỡng ở backend
export type HoursLevel = "ok" | "warn" | "bad";
export function getHoursLevel(actual: number, estimated: number): HoursLevel {
  if (estimated <= 0 || actual <= estimated) return "ok";
  return (actual - estimated) / estimated < 0.2 ? "warn" : "bad"; // ⚠ mốc đúng 20% chưa chốt
}
```
UI: `<HoursLevelMark level={…} />` vẽ vuông/tròn/tam giác — không chỉ đổi màu.

## 5. Phân quyền ở UI
`const can = useCan();` → `can("task.approve", project)`. Không có quyền → **ẩn**. Chỉ `disabled` khi có lý do hiển thị (tooltip).

## 6. Kanban
- Bám `docs/design/artboards/02-kanban-manager.html` (Manager) và `docs/design/artboards/04-member-board.html` (Member).
- dnd-kit (đề xuất). Kéo thả: cập nhật lạc quan bằng `onMutate` + rollback ở `onError`; rollback thì hiện `Banner` đỏ "Không lưu được thay đổi vị trí thẻ. Thẻ đã trở về cột cũ, vui lòng kéo lại."
- Vùng thả không hợp lệ (theo vai trò/luật chuyển/WIP) mờ đi kèm **lý do cụ thể** — lấy luật từ API, không hard-code.
- Thẻ luôn là `<TaskCard>` dùng chung; giữ ≥ 6 thẻ/cột ở 1366×768.
- Sự kiện WebSocket → `queryClient.setQueryData(taskKeys.byProject(id), …)`.

## 7. Test (Vitest + RTL + MSW)

```tsx
it("hiện lỗi khi log giờ âm", async () => {
  render(<LogTimeModal taskId="t1" />, { wrapper: TestProviders });
  await userEvent.type(screen.getByLabelText("Số giờ"), "-1");
  await userEvent.click(screen.getByRole("button", { name: "Log giờ" }));
  expect(await screen.findByText("Số giờ phải lớn hơn 0")).toBeInTheDocument();
});
```
Handler MSW đặt ở `tests/mocks/handlers.ts`, trả đúng shape lỗi backend. Truy vấn theo role/label tiếng Việt — đây cũng là kiểm tra khả năng truy cập.

## Checklist
- [ ] Không `any`, không hex trong component, không chuỗi tiếng Anh hiển thị.
- [ ] Đủ skeleton / empty / error / toast.
- [ ] Ẩn theo vai trò đúng `worktrack-domain`.
- [ ] `npx tsc --noEmit`, `npm run lint`, `npm run test` xanh.
