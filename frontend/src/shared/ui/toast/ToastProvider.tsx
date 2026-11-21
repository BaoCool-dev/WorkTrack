import { Check } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { ToastContext, type ToastAction, type ToastApi } from './context'

export const TOAST_DURATION_MS = 3000

interface ToastItem {
  id: number
  message: string
  action?: ToastAction
}

/** §5: nền ink, chữ trắng 12px, icon ✓, góc dưới phải, 3 giây, một dòng. */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([])
  const nextId = useRef(0)
  const timers = useRef(new Set<ReturnType<typeof setTimeout>>())

  const dismiss = useCallback((id: number) => {
    setItems((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const show = useCallback(
    (message: string, action?: ToastAction) => {
      const id = ++nextId.current
      setItems((prev) => [...prev, { id, message, action }])
      const timer = setTimeout(() => {
        timers.current.delete(timer)
        dismiss(id)
      }, TOAST_DURATION_MS)
      timers.current.add(timer)
    },
    [dismiss],
  )

  useEffect(() => {
    const pending = timers.current
    return () => {
      pending.forEach(clearTimeout)
    }
  }, [])

  const api = useMemo<ToastApi>(() => ({ show }), [show])

  return (
    <ToastContext value={api}>
      {children}
      <div
        role="status"
        aria-live="polite"
        className="pointer-events-none fixed right-5 bottom-5 z-50 flex flex-col items-end gap-2"
      >
        {items.map((t) => (
          <div
            key={t.id}
            className="flex max-w-90 items-center gap-2 truncate rounded-md bg-ink px-3 py-2 text-sm text-white shadow-overlay"
          >
            <Check size={14} strokeWidth={2} aria-hidden className="shrink-0" />
            <span className="truncate">{t.message}</span>
            {t.action && (
              <button
                type="button"
                className="pointer-events-auto ml-1 shrink-0 font-semibold text-accent-border hover:underline"
                onClick={() => {
                  t.action?.onClick()
                  dismiss(t.id)
                }}
              >
                {t.action.label}
              </button>
            )}
          </div>
        ))}
      </div>
    </ToastContext>
  )
}
