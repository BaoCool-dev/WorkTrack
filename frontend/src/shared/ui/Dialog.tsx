import { useEffect, useId, useRef, type ReactNode } from 'react'

export interface DialogProps {
  open: boolean
  onClose: () => void
  title: string
  description?: ReactNode
  icon?: ReactNode
  /** Nút ở góc phải-dưới: phụ rồi chính. */
  children?: ReactNode
}

/** §5: nền trắng, viền, bo 6, shadow-overlay, padding 18; Escape đóng; trả focus khi đóng. */
export function Dialog({ open, onClose, title, description, icon, children }: DialogProps) {
  const titleId = useId()
  const descId = useId()
  const panelRef = useRef<HTMLDivElement>(null)
  const onCloseRef = useRef(onClose)

  useEffect(() => {
    onCloseRef.current = onClose
  })

  useEffect(() => {
    if (!open) return
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null
    panelRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCloseRef.current()
    }
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('keydown', onKey)
      previous?.focus()
    }
  }, [open])

  if (!open) return null
  return (
    <div className="fixed inset-0 z-40 flex items-center justify-center bg-ink/40 p-4">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={description ? descId : undefined}
        tabIndex={-1}
        className="w-full max-w-110 rounded-md border border-line bg-surface p-4.5 shadow-overlay focus:outline-none"
      >
        <div className="flex gap-3">
          {icon}
          <div className="flex flex-col gap-1">
            <h2 id={titleId} className="text-base font-semibold text-ink">
              {title}
            </h2>
            {description && (
              <div id={descId} className="text-sm text-ink-muted">
                {description}
              </div>
            )}
          </div>
        </div>
        {children && <div className="mt-4 flex justify-end gap-2">{children}</div>}
      </div>
    </div>
  )
}
