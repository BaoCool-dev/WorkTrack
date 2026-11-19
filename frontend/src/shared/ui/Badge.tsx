import type { ReactNode } from 'react'
import { cn } from '@/shared/lib/cn'

export type BadgeTone = 'ok' | 'warn' | 'bad' | 'draft' | 'active' | 'accent'

const tones: Record<BadgeTone, string> = {
  ok: 'bg-ok-bg text-ok border-ok-border',
  warn: 'bg-warn-bg text-warn border-warn-border',
  bad: 'bg-bad-bg text-bad border-bad-border',
  draft: 'bg-draft-bg text-ink-muted border-draft-border',
  active: 'bg-active-bg text-active border-active-border',
  accent: 'bg-accent-subtle text-accent-hover border-accent-border',
}

/** §5: 12/600, bo 4, padding 2/8 — luôn có chữ, không chỉ dựa vào màu. */
export function Badge({ tone = 'draft', children }: { tone?: BadgeTone; children: ReactNode }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-sm border px-2 py-0.5 text-sm font-semibold',
        tones[tone],
      )}
    >
      {children}
    </span>
  )
}
