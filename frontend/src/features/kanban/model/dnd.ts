/** Id kéo/thả có tiền tố để biết đang thả lên thẻ hay lên khoảng trống của cột. */
export const taskDndId = (id: string) => `task:${id}`
export const columnDndId = (id: string) => `col:${id}`

/** Trạng thái vùng thả của cột trong lúc kéo; null = không có thẻ nào đang kéo. */
export type DropState = { ok: true } | { ok: false; reason: string } | null
