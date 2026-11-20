import { ChevronDown } from 'lucide-react'
import { useId, type Ref, type SelectHTMLAttributes } from 'react'
import { cn } from '@/shared/lib/cn'

export interface SelectOption {
  value: string
  label: string
  disabled?: boolean
}

export interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  label: string
  options: readonly SelectOption[]
  error?: string
  hint?: string
  /** Ẩn nhãn khỏi giao diện nhưng giữ cho trình đọc màn hình (ô trong bảng). */
  hideLabel?: boolean
  ref?: Ref<HTMLSelectElement>
}

/** §5: cùng khung với Input — nhãn 12/600, cao 40px, viền line-strong, focus accent. */
export function Select({
  label,
  options,
  error,
  hint,
  hideLabel = false,
  id,
  className,
  ...rest
}: SelectProps) {
  const autoId = useId()
  const selectId = id ?? autoId
  const noteId = `${selectId}-note`
  const note = error ?? hint
  return (
    <div className="flex flex-col gap-1">
      <label
        htmlFor={selectId}
        className={cn('text-sm font-semibold text-ink-muted', hideLabel && 'sr-only')}
      >
        {label}
      </label>
      <div className="relative">
        <select
          id={selectId}
          aria-invalid={error ? true : false}
          aria-describedby={note ? noteId : undefined}
          className={cn(
            'w-full appearance-none rounded-md border bg-surface pr-8 pl-3 text-ink',
            'h-10 text-base',
            'focus:outline-none focus:ring-1',
            error
              ? 'border-bad focus:border-bad focus:ring-bad'
              : 'border-line-strong focus:border-accent focus:ring-accent',
            'disabled:cursor-not-allowed disabled:bg-subtle disabled:text-disabled',
            className,
          )}
          {...rest}
        >
          {options.map((o) => (
            <option key={o.value} value={o.value} disabled={o.disabled}>
              {o.label}
            </option>
          ))}
        </select>
        <ChevronDown
          size={16}
          strokeWidth={2}
          aria-hidden
          className="pointer-events-none absolute top-1/2 right-2.5 -translate-y-1/2 text-ink-faint"
        />
      </div>
      {note && (
        <p id={noteId} className={cn('text-sm', error ? 'text-bad' : 'text-ink-faint')}>
          {note}
        </p>
      )}
    </div>
  )
}
