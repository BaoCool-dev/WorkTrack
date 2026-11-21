import type { Active, Over } from '@dnd-kit/core'

import { makeBoard } from '@/features/projects/test-data'
import { messages } from '@/shared/lib/messages'

import { boardTasks } from '../test-data'
import { boardAnnouncements } from './a11y'

const t = messages.kanban.a11y

function must<T>(v: T | undefined): T {
  if (v === undefined) throw new Error('dữ liệu mẫu thiếu phần tử')
  return v
}

describe('boardAnnouncements — câu đọc khi kéo thả bằng tiếng Việt', () => {
  const tasks = boardTasks()
  const columns = makeBoard().columns
  const a = boardAnnouncements(tasks, columns)
  const card = must(tasks[0])
  const target = must(columns[columns.length - 1])
  const active = { id: `task:${card.id}` } as Active
  const asOver = (id: string) => ({ id }) as Over

  it('nhấc, hủy thẻ: đọc tiêu đề thẻ', () => {
    expect(a.onDragStart({ active })).toBe(t.picked(card.title))
    expect(a.onDragCancel({ active, over: null })).toBe(t.cancelled(card.title))
  })

  it('trên/thả vào cột: đọc tên cột', () => {
    const over = asOver(`col:${target.id}`)
    expect(a.onDragOver({ active, over })).toBe(t.over(card.title, target.name))
    expect(a.onDragEnd({ active, over })).toBe(t.dropped(card.title, target.name))
  })

  it('thả lên thẻ khác = thả vào cột của thẻ đó; ra ngoài bảng thì báo giữ chỗ cũ', () => {
    const other = must(tasks[tasks.length - 1])
    const colName = columns.find((c) => c.id === other.column.id)?.name ?? ''
    expect(a.onDragEnd({ active, over: asOver(`task:${other.id}`) })).toBe(
      t.dropped(card.title, colName),
    )
    expect(a.onDragEnd({ active, over: null })).toBe(t.droppedOutside(card.title))
  })
})
