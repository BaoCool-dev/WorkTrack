import type { ReactNode } from 'react'

import type { Task } from '@/features/tasks'
import { cn } from '@/shared/lib/cn'
import { messages } from '@/shared/lib/messages'
import { Avatar, HoursLegend } from '@/shared/ui'

import { filterOptions, type BoardFilters } from '../model/boardState'
import type { SocketStatus } from '../model/types'

const t = messages.kanban

const MAX_FACES = 4

function toggle(list: string[], id: string): string[] {
  return list.includes(id) ? list.filter((x) => x !== id) : [...list, id]
}

function Chip({
  pressed,
  onClick,
  children,
}: {
  pressed: boolean
  onClick: () => void
  children: ReactNode
}) {
  return (
    <button
      type="button"
      aria-pressed={pressed}
      onClick={onClick}
      className={cn(
        'inline-flex h-7 items-center gap-1 rounded-sm border px-2 text-sm font-semibold',
        pressed
          ? 'border-accent-border bg-accent-subtle text-accent-hover'
          : 'border-line-strong bg-surface text-ink-muted hover:bg-subtle',
      )}
    >
      {children}
    </button>
  )
}

interface FilterBarProps {
  tasks: readonly Task[]
  filters: BoardFilters
  onChange: (f: BoardFilters) => void
  /** Member: hiện chip "Việc của tôi" thay vì cả danh sách người nhận. */
  mineId?: string
}

export function FilterBar({ tasks, filters, onChange, mineId }: FilterBarProps) {
  const { labels, people } = filterOptions(tasks)
  const active = filters.labelIds.length + filters.assigneeIds.length > 0
  return (
    <div role="group" aria-label={t.filters.label} className="flex flex-wrap items-center gap-1.5">
      {mineId !== undefined && (
        <Chip
          pressed={filters.assigneeIds.length === 1 && filters.assigneeIds[0] === mineId}
          onClick={() => {
            const on = filters.assigneeIds.length === 1 && filters.assigneeIds[0] === mineId
            onChange({ ...filters, assigneeIds: on ? [] : [mineId] })
          }}
        >
          {t.filters.mine}
        </Chip>
      )}
      {labels.length > 0 && <span className="text-sm text-ink-faint">{t.filters.labels}</span>}
      {labels.map((l) => (
        <Chip
          key={l.id}
          pressed={filters.labelIds.includes(l.id)}
          onClick={() => {
            onChange({ ...filters, labelIds: toggle(filters.labelIds, l.id) })
          }}
        >
          {l.name}
        </Chip>
      ))}
      {mineId === undefined && people.length > 0 && (
        <span className="text-sm text-ink-faint">{t.filters.assignees}</span>
      )}
      {mineId === undefined &&
        people.map((p) => (
          <Chip
            key={p.id}
            pressed={filters.assigneeIds.includes(p.id)}
            onClick={() => {
              onChange({ ...filters, assigneeIds: toggle(filters.assigneeIds, p.id) })
            }}
          >
            {p.name}
          </Chip>
        ))}
      {active && (
        <button
          type="button"
          onClick={() => {
            onChange({ labelIds: [], assigneeIds: [] })
          }}
          className="h-7 px-1.5 text-sm font-semibold text-accent hover:underline"
        >
          {t.filters.clear}
        </button>
      )}
    </div>
  )
}

/** Chip WIP của người xem: viền line-strong; đủ giới hạn chuyển màu bad (design-system §6). */
export function WipChip({ current, limit }: { current: number; limit: number }) {
  const full = current >= limit
  return (
    <span
      title={t.wip.chipTitle}
      className={cn(
        'num inline-flex h-7 items-center rounded-sm border px-2 text-sm font-semibold',
        full ? 'border-bad-border bg-bad-bg text-bad' : 'border-line-strong text-ink-muted',
      )}
    >
      {t.wip.chip(current, limit)}
    </span>
  )
}

/** Chấm trạng thái realtime + chồng avatar người đang xem (24px, chồng −7px). */
export function LiveIndicator({
  status,
  viewers,
}: {
  status: SocketStatus
  viewers: readonly string[]
}) {
  const shown = viewers.slice(0, MAX_FACES)
  const extra = viewers.length - shown.length
  return (
    <div className="flex items-center gap-2">
      {status !== 'lost' && (
        <span className="inline-flex items-center gap-1.5 text-sm text-ink-muted">
          <span
            aria-hidden
            className={cn('size-2 rounded-full', status === 'live' ? 'bg-ok' : 'bg-disabled')}
          />
          {status === 'live' ? t.realtime.live : t.realtime.connecting}
        </span>
      )}
      {viewers.length > 0 && (
        <div
          className="flex items-center"
          role="group"
          aria-label={t.realtime.presence(viewers.length)}
        >
          {shown.map((name, i) => (
            <span
              key={`${name}-${String(i)}`}
              className={cn('rounded-full ring-2 ring-surface', i > 0 && '-ml-1.75')}
            >
              <Avatar name={name} size={24} />
            </span>
          ))}
          {extra > 0 && (
            <span className="num -ml-1.75 inline-flex size-6 items-center justify-center rounded-full bg-subtle text-sm font-semibold text-ink-muted ring-2 ring-surface">
              {t.realtime.more(extra)}
            </span>
          )}
        </div>
      )}
    </div>
  )
}

export { HoursLegend }
