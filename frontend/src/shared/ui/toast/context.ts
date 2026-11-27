import { createContext } from 'react'

export interface ToastApi {
  /** Thông báo thành công một dòng, tự ẩn sau 3 giây. Không dùng cho lỗi cần đọc kỹ. */
  show: (message: string) => void
}

export const ToastContext = createContext<ToastApi | null>(null)
