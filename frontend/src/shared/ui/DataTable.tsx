import type { ReactNode } from 'react'
import { cn } from '@/shared/lib/cn'
import { Skeleton } from './Skeleton'

/** §5 Bảng: khung viền 1px bo 6; header nền subtle 12/600 ink-faint; hàng 40px, phân cách viền trên. */
export function DataTable({
  label,
  head,
  children,
}: {
  label: string
  head: { label: string; numeric?: boolean; className?: string }[]
  children: ReactNode
}) {
  return (
    <div className="overflow-x-auto rounded-md border border-line bg-surface">
      <table aria-label={label} className="w-full border-collapse text-base">
        <thead className="bg-subtle">
          <tr>
            {head.map((h) => (
              <th
                key={h.label}
                scope="col"
                className={cn(
                  'px-3 py-2 text-sm font-semibold text-ink-faint',
                  h.numeric ? 'text-right' : 'text-left',
                  h.className,
                )}
              >
                {h.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
    </div>
  )
}

export const cellClass = 'border-t border-line px-3 py-2 align-middle'

export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="flex flex-col gap-2" aria-busy="true">
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} className="h-10 w-full" />
      ))}
    </div>
  )
}
