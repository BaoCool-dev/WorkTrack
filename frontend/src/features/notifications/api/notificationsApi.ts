import type { Page } from '@/features/projects'
import { http } from '@/shared/api/http'
import { NOTIFICATION_PAGE_SIZE, type AppNotification } from '../model/types'

const base = '/api/v1/me/notifications'

export const notificationsApi = {
  list: () =>
    http<Page<AppNotification>>(`${base}?page=1&pageSize=${String(NOTIFICATION_PAGE_SIZE)}`),
  unreadCount: () => http<{ count: number }>(`${base}/unread-count`),
  markRead: (id: string) =>
    http<undefined>(`${base}/${encodeURIComponent(id)}/read`, { method: 'POST' }),
  markAllRead: () => http<{ count: number }>(`${base}/read-all`, { method: 'POST' }),
}
