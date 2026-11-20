import { cn } from '@/shared/lib/cn'

const palette = [
  'bg-label-a-bg text-label-a-fg',
  'bg-label-b-bg text-label-b-fg',
  'bg-label-c-bg text-label-c-fg',
] as const

function paletteFor(name: string): string {
  let h = 0
  for (const ch of name.toLowerCase()) h = (h * 31 + (ch.codePointAt(0) ?? 0)) >>> 0
  return palette[h % palette.length] ?? palette[0]
}

/** §5 chip nhãn: 12/600, bo 4, padding 1/6, không viền; màu theo hash tên nhãn (§3). */
export function LabelChip({ name }: { name: string }) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-sm px-1.5 py-px text-sm font-semibold',
        paletteFor(name),
      )}
    >
      {name}
    </span>
  )
}
