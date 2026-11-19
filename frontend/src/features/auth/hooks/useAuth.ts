import { use, useSyncExternalStore } from 'react'
import { SessionContext } from '../model/context'

export function useSession() {
  const session = use(SessionContext)
  if (!session) throw new Error('useAuth phải nằm trong <AuthProvider>')
  return session
}

export function useAuth() {
  const session = useSession()
  const state = useSyncExternalStore(session.subscribe, session.getSnapshot)
  return { ...state, login: session.login, logout: session.logout }
}
