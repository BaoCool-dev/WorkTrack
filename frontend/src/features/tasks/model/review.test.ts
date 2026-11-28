import { describe, expect, it } from 'vitest'
import type { BoardColumn } from '@/features/projects'
import { hasPendingSubtasks, reviewColumnId } from './review'

const col = (
  id: string,
  position: number,
  statusGroup: BoardColumn['statusGroup'],
): BoardColumn => ({
  id,
  name: id,
  position,
  statusGroup,
  taskCount: 0,
})

describe('reviewColumnId', () => {
  it('chọn cột in_progress đứng cuối của bảng mặc định', () => {
    const cols = [
      col('backlog', 0, 'todo'),
      col('todo', 1, 'todo'),
      col('doing', 2, 'in_progress'),
      col('review', 3, 'in_progress'),
      col('done', 4, 'done'),
    ]
    expect(reviewColumnId(cols)).toBe('review')
  })

  it('không theo thứ tự mảng mà theo position', () => {
    const cols = [
      col('review', 3, 'in_progress'),
      col('doing', 2, 'in_progress'),
      col('done', 4, 'done'),
    ]
    expect(reviewColumnId(cols)).toBe('review')
  })

  it('chỉ một cột in_progress → không có bước nghiệm thu', () => {
    expect(reviewColumnId([col('doing', 0, 'in_progress'), col('done', 1, 'done')])).toBeNull()
  })

  it('không có cột done → null', () => {
    expect(reviewColumnId([col('a', 0, 'in_progress'), col('b', 1, 'in_progress')])).toBeNull()
  })
})

describe('hasPendingSubtasks', () => {
  it.each([
    [{ parentId: null, subtaskCount: 3, subtaskDoneCount: 2 }, true],
    [{ parentId: null, subtaskCount: 3, subtaskDoneCount: 3 }, false],
    [{ parentId: null, subtaskCount: 0, subtaskDoneCount: 0 }, false],
    [{ parentId: 'p', subtaskCount: 1, subtaskDoneCount: 0 }, false],
  ])('%o → %s', (task, want) => {
    expect(hasPendingSubtasks(task)).toBe(want)
  })
})
