import { LoaderCircle } from 'lucide-react'
import type { ButtonHTMLAttributes, ReactNode, Ref } from 'react'
import { cn } from '@/shared/lib/cn'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger' | 'ok' | 'bad'

const variants: Record<Variant, string> = {
  primary:
    'bg-accent text-white hover:bg-accent-hover border border-accent hover:border-accent-hover',
  secondary: 'bg-surface text-ink border border-line-strong hover:bg-subtle',
  ghost: 'bg-transparent text-ink-muted border border-transparent hover:bg-subtle',
  danger: 'bg-surface text-bad border border-bad-border hover:bg-bad-bg',
  /** §5 "Nghiệm thu": nền ok, chữ trắng. */
  ok: 'bg-ok text-white border border-ok hover:opacity-90',
  /** §7: nút xác nhận "Từ chối" trong hộp cảnh báo nghiệm thu: nền bad. */
  bad: 'bg-bad text-white border border-bad hover:opacity-90',
}

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  loading?: boolean
  icon?: ReactNode
  ref?: Ref<HTMLButtonElement>
}

/** §5: cao tối thiểu 40px, padding 9/14, bo 6, chữ 14/600. */
export function Button({
  variant = 'primary',
  loading = false,
  icon,
  disabled,
  type = 'button',
  className,
  children,
  ...rest
}: ButtonProps) {
  const isDisabled = disabled === true || loading
  return (
    <button
      type={type}
      disabled={isDisabled}
      aria-busy={loading || undefined}
      className={cn(
        'inline-flex min-h-10 items-center justify-center gap-2 rounded-md px-3.5 py-2 text-base font-semibold transition-colors duration-150',
        'focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent',
        isDisabled
          ? 'cursor-not-allowed border border-line bg-subtle text-disabled'
          : variants[variant],
        className,
      )}
      {...rest}
    >
      {loading ? (
        <LoaderCircle size={16} strokeWidth={2} className="animate-spin" aria-hidden />
      ) : (
        icon
      )}
      {children}
    </button>
  )
}
