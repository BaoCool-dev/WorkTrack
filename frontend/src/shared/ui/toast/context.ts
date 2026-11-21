import { createContext } from 'react'

/** Nút trong toast — chỉ dùng khi thao tác đảo ngược được (vd. "Hoàn tác"). */
export interface ToastAction {
  label: string
  onClick: () => void
}

export interface ToastApi {
  /** Thông báo thành công một dòng, tự ẩn sau 3 giây. Không dùng cho lỗi cần đọc kỹ. */
  show: (message: string, action?: ToastAction) => void
}

export const ToastContext = createContext<ToastApi | null>(null)
