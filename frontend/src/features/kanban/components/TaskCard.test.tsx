import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { makeSubtask, makeTask } from '@/features/tasks/test-data'

import { TaskCardView } from './TaskCard'

// Thẻ Kanban hiện giờ thực tế (cha + con) so với dự kiến — design-system §6, ngưỡng B5 [CF-08].
describe('Thẻ Kanban — thanh so giờ (CF-08)', () => {
  it('giờ thực tế lấy từ actualMinutes, kèm số việc con xong/tổng', () => {
    render(
      <TaskCardView
        task={makeTask({
          estimatedMinutes: 960,
          actualMinutes: 870,
          subtaskCount: 3,
          subtaskDoneCount: 2,
        })}
      />,
    )
    expect(screen.getByText('14,5/16h')).toBeInTheDocument()
    expect(screen.getByText('2/3 con')).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Trong định mức' })).toBeInTheDocument()
  })

  it('vượt < 20% là "Vượt nhẹ", ≥ 20% là "Vượt nặng" (B5)', () => {
    const { rerender } = render(
      <TaskCardView task={makeTask({ estimatedMinutes: 960, actualMinutes: 1100 })} />,
    )
    expect(screen.getByRole('img', { name: 'Vượt nhẹ' })).toBeInTheDocument()
    rerender(<TaskCardView task={makeTask({ estimatedMinutes: 960, actualMinutes: 1152 })} />)
    expect(screen.getByRole('img', { name: 'Vượt nặng' })).toBeInTheDocument()
  })

  it('thanh giờ đầy tối đa 100% khi vượt dự kiến', () => {
    const { container } = render(
      <TaskCardView task={makeTask({ estimatedMinutes: 600, actualMinutes: 1200 })} />,
    )
    const fill = container.querySelector('[data-testid="hours-fill"]')
    expect(fill).toHaveStyle({ width: '100%' })
  })

  it('việc con không có SP và dự kiến: chỉ hiện giờ đã log, không gắn mức vượt', () => {
    render(<TaskCardView task={makeSubtask({ actualMinutes: 90 })} />)
    expect(screen.getByText('1,5h')).toBeInTheDocument()
    expect(screen.queryByText(/SP$/)).not.toBeInTheDocument()
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })

  it('không có việc con thì không hiện "x/y con"', () => {
    render(<TaskCardView task={makeTask()} />)
    expect(screen.queryByText(/con$/)).not.toBeInTheDocument()
  })
})

// QA bổ sung — BVA quanh ngưỡng định mức và 20% (dự kiến 960 phút → ngưỡng 1152) [CF-08].
describe('Thẻ Kanban — biên mức giờ (QA) [CF-08]', () => {
  it.each([
    [960, 'Trong định mức'], // đúng bằng dự kiến [TC_LOG_215]
    [961, 'Vượt nhẹ'], // vượt 1 phút
    [1151, 'Vượt nhẹ'], // ngay dưới 20% [TC_LOG_216]
    [1153, 'Vượt nặng'],
  ])('thực tế %i/960 phút → %s', (actualMinutes, label) => {
    render(<TaskCardView task={makeTask({ estimatedMinutes: 960, actualMinutes })} />)
    expect(screen.getByRole('img', { name: label })).toBeInTheDocument()
  })
})
