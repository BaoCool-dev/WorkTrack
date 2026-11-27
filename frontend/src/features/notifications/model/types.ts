export type NotificationType = 'task_assigned' | 'task_commented' | 'review_rejected'

interface Ref {
  id: string
  name: string
}

/** Khớp notificationResponse ở backend (GET /api/v1/me/notifications). */
export interface AppNotification {
  id: string
  type: NotificationType
  task: Ref
  project: Ref
  actor: Ref | null
  /** Trích đoạn bình luận hoặc lý do từ chối; rỗng với task_assigned. */
  message: string
  read: boolean
  createdAt: string
}

export const NOTIFICATION_PAGE_SIZE = 10
/** Chưa có đẩy realtime cho thông báo — chuông hỏi lại mỗi 30 giây (decisions.md mục C). */
export const UNREAD_POLL_MS = 30_000
