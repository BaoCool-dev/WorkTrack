import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, type ReactNode } from 'react'
import { configureHttpAuth } from '@/shared/api/http'
import { SessionContext } from '../model/context'
import { session as appSession, type Session } from '../model/session'

interface Props {
  children: ReactNode
  /** Test truyền phiên riêng; ứng dụng dùng phiên dùng chung. */
  session?: Session
}

export function AuthProvider({ children, session = appSession }: Props) {
  const queryClient = useQueryClient()
  const prevStatus = useRef(session.getSnapshot().status)

  useEffect(() => {
    configureHttpAuth({ getToken: session.getToken, refresh: session.refresh })
    void session.start()
    // Rời phiên (đăng xuất hoặc hết phiên) → xóa cache để người sau không thấy dữ liệu cũ.
    const unsubscribe = session.subscribe(() => {
      const status = session.getSnapshot().status
      if (prevStatus.current === 'authenticated' && status === 'anonymous') queryClient.clear()
      prevStatus.current = status
    })
    return () => {
      unsubscribe()
      configureHttpAuth(null)
    }
  }, [session, queryClient])

  return <SessionContext value={session}>{children}</SessionContext>
}
