import { cn } from '@/shared/lib/cn'
import { initials } from '@/shared/lib/initials'

const palette = [
  'bg-label-a-bg text-label-a-fg',
  'bg-label-b-bg text-label-b-fg',
  'bg-label-c-bg text-label-c-fg',
] as const

const sizes = { 20: 'size-5', 24: 'size-6', 28: 'size-7' } as const

function paletteFor(name: string): string {
  let h = 0
  for (const ch of name) h = (h * 31 + (ch.codePointAt(0) ?? 0)) >>> 0
  return palette[h % palette.length] ?? palette[0]
}

/** Avatar chữ cái dùng bảng màu nhãn (§3). */
export function Avatar({ name, size = 28 }: { name: string; size?: keyof typeof sizes }) {
  return (
    <span
      role="img"
      aria-label={name}
      className={cn(
        'inline-flex shrink-0 items-center justify-center rounded-full text-sm font-semibold',
        sizes[size],
        paletteFor(name),
      )}
    >
      {initials(name)}
    </span>
  )
}
