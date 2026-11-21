import { describe, expect, it } from 'vitest'

import { makeTask } from '@/features/tasks/test-data'

import { columnTasks, dropPosition, filterTasks, moveTask, wipByAssignee } from './boardState'

const backlog = { id: 'c-1', name: 'Backlog', statusGroup: 'todo' as const }
const doing = { id: 'c-2', name: 'In Progress', statusGroup: 'in_progress' as const }
const ha = { id: 'u-2', name: 'Đỗ Thu Hà' }

const tasks = [
  makeTask({ id: 'a', column: backlog, position: 1 }),
  makeTask({ id: 'b', column: backlog, position: 0 }),
  makeTask({ id: 'c', column: backlog, position: 2 }),
  makeTask({ id: 'x', column: doing, position: 0, assignee: ha }),
]

const ids = (ts: { id: string }[]) => ts.map((t) => t.id)

describe('columnTasks', () => {
  it('lọc theo cột và sắp theo position', () => {
    expect(ids(columnTasks(tasks, 'c-1'))).toEqual(['b', 'a', 'c'])
  })
})

describe('moveTask', () => {
  it('đổi thứ tự trong cột và đánh lại số 0..n-1', () => {
    const next = moveTask(tasks, 'c', backlog, 0)
    expect(ids(columnTasks(next, 'c-1'))).toEqual(['c', 'b', 'a'])
    expect(columnTasks(next, 'c-1').map((t) => t.position)).toEqual([0, 1, 2])
  })
  it('chuyển sang cột khác: đổi column, đánh lại cả hai cột', () => {
    const next = moveTask(tasks, 'b', doing, 1)
    expect(ids(columnTasks(next, 'c-1'))).toEqual(['a', 'c'])
    expect(ids(columnTasks(next, 'c-2'))).toEqual(['x', 'b'])
    expect(next.find((t) => t.id === 'b')?.column).toEqual(doing)
    expect(columnTasks(next, 'c-1').map((t) => t.position)).toEqual([0, 1])
  })
  it('vị trí vượt quá cuối cột thì đặt ở cuối (như server)', () => {
    expect(ids(columnTasks(moveTask(tasks, 'b', doing, 99), 'c-2'))).toEqual(['x', 'b'])
  })
  it('không đổi mảng gốc', () => {
    moveTask(tasks, 'b', doing, 0)
    expect(tasks[1]?.column).toEqual(backlog)
  })
})

describe('dropPosition', () => {
  it('thả lên một thẻ → chèn vào chỗ thẻ đó (tính sau khi bỏ thẻ đang kéo)', () => {
    expect(dropPosition(tasks, 'c', 'c-1', 'b')).toBe(0)
    expect(dropPosition(tasks, 'b', 'c-2', 'x')).toBe(0)
  })
  it('kéo xuống trong cùng cột → đứng sau thẻ được thả lên', () => {
    // thứ tự Backlog: b, a, c
    expect(dropPosition(tasks, 'b', 'c-1', 'a')).toBe(1)
    expect(dropPosition(tasks, 'b', 'c-1', 'c')).toBe(2)
    expect(dropPosition(tasks, 'a', 'c-1', 'b')).toBe(0)
  })
  it('thả vào vùng trống của cột → cuối cột', () => {
    expect(dropPosition(tasks, 'b', 'c-2', null)).toBe(1)
    expect(dropPosition(tasks, 'b', 'c-1', null)).toBe(2)
  })
})

describe('wipByAssignee', () => {
  it('đếm việc cha nhóm đang làm theo người nhận', () => {
    const more = [
      ...tasks,
      makeTask({ id: 'y', column: doing, assignee: ha }),
      makeTask({ id: 'z', column: doing }),
    ]
    expect(wipByAssignee(more).get('u-2')).toBe(2)
    expect(wipByAssignee(more).size).toBe(1)
  })
})

describe('filterTasks', () => {
  const l1 = { id: 'l-1', name: 'Frontend' }
  const l2 = { id: 'l-2', name: 'Backend' }
  const list = [
    makeTask({ id: 'p', labels: [l1], assignee: ha }),
    makeTask({ id: 'q', labels: [l2], assignee: null }),
    makeTask({ id: 'r', labels: [l1, l2], assignee: { id: 'u-3', name: 'Lê Văn C' } }),
  ]
  it('không chọn gì → giữ nguyên', () => {
    expect(ids(filterTasks(list, { labelIds: [], assigneeIds: [] }))).toEqual(['p', 'q', 'r'])
  })
  it('nhãn: có ít nhất một nhãn đã chọn', () => {
    expect(ids(filterTasks(list, { labelIds: ['l-1'], assigneeIds: [] }))).toEqual(['p', 'r'])
  })
  it('người nhận, kể cả "chưa giao"; kết hợp với nhãn bằng VÀ', () => {
    expect(ids(filterTasks(list, { labelIds: [], assigneeIds: ['none'] }))).toEqual(['q'])
    expect(ids(filterTasks(list, { labelIds: ['l-2'], assigneeIds: ['u-3'] }))).toEqual(['r'])
  })
})
