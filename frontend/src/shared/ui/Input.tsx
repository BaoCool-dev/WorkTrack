import { useId, type InputHTMLAttributes, type Ref } from 'react'
import { cn } from '@/shared/lib/cn'

export interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string
  error?: string
  hint?: string
  ref?: Ref<HTMLInputElement>
}

/** §5: nhãn 12/600 ở trên, ô cao 40px, focus viền 2px accent, lỗi viền bad + dòng lỗi 12px. */
export function Input({ label, error, hint, id, className, ...rest }: InputProps) {
  const autoId = useId()
  const inputId = id ?? autoId
  const noteId = `${inputId}-note`
  const note = error ?? hint
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={inputId} className="text-sm font-semibold text-ink-muted">
        {label}
      </label>
      <input
        id={inputId}
        aria-invalid={error ? true : false}
        aria-describedby={note ? noteId : undefined}
        className={cn(
          'h-10 rounded-md border bg-surface px-3 text-base text-ink placeholder:text-ink-faint',
          'focus:outline-none focus:ring-1',
          error
            ? 'border-bad focus:border-bad focus:ring-bad'
            : 'border-line-strong focus:border-accent focus:ring-accent',
          'disabled:cursor-not-allowed disabled:bg-subtle disabled:text-disabled',
          className,
        )}
        {...rest}
      />
      {note && (
        <p id={noteId} className={cn('text-sm', error ? 'text-bad' : 'text-ink-faint')}>
          {note}
        </p>
      )}
    </div>
  )
}
