import type { BoardColumn } from '@/features/projects'

/**
 * Tìm cột "Chờ nghiệm thu" của bảng — PHẢI khớp domain.ReviewTargets ở backend (CF-09).
 *
 * Vì sao theo vị trí, không theo tên: Manager được đổi tên cột (CF-03), nên backend coi cột
 * in_progress đứng cuối là cột chờ nghiệm thu, và chỉ khi bảng có ≥ 2 cột in_progress + 1 cột done.
 * Trả null khi bảng không có bước nghiệm thu.
 */
export function reviewColumnId(columns: readonly BoardColumn[]): string | null {
  const inProgress = columns.filter((c) => c.statusGroup === 'in_progress')
  const hasDone = columns.some((c) => c.statusGroup === 'done')
  if (inProgress.length < 2 || !hasDone) return null
  const last = inProgress.reduce((a, b) => (b.position > a.position ? b : a))
  return last.id
}

/** Việc cha còn việc con chưa xong → chỉ cảnh báo, vẫn cho chuyển sang chờ nghiệm thu. */
export function hasPendingSubtasks(task: {
  parentId: string | null
  subtaskCount: number
  subtaskDoneCount: number
}): boolean {
  return task.parentId === null && task.subtaskDoneCount < task.subtaskCount
}
