import { useContext } from 'react'
import { ToastContext, type ToastApi } from './context'

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast phải nằm trong <ToastProvider>')
  return ctx
}
