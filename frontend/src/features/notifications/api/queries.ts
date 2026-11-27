import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { UNREAD_POLL_MS } from '../model/types'
import { notificationsApi } from './notificationsApi'

export const notificationKeys = {
  all: ['notifications'] as const,
  list: () => [...notificationKeys.all, 'list'] as const,
  unread: () => [...notificationKeys.all, 'unread'] as const,
}

export function useUnreadCount() {
  return useQuery({
    queryKey: notificationKeys.unread(),
    queryFn: notificationsApi.unreadCount,
    select: (d) => d.count,
    refetchInterval: UNREAD_POLL_MS,
  })
}

/** Danh sách chỉ tải khi mở chuông. */
export function useNotifications(enabled: boolean) {
  return useQuery({
    queryKey: notificationKeys.list(),
    queryFn: notificationsApi.list,
    enabled,
  })
}

function useInvalidateAll() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: notificationKeys.all })
}

export function useMarkRead() {
  const invalidate = useInvalidateAll()
  return useMutation({ mutationFn: notificationsApi.markRead, onSuccess: invalidate })
}

export function useMarkAllRead() {
  const invalidate = useInvalidateAll()
  return useMutation({ mutationFn: notificationsApi.markAllRead, onSuccess: invalidate })
}
