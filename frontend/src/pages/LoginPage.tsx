import { Navigate, useSearchParams } from 'react-router'
import { AuthCard, LoginForm, SessionLoading, safeNext, useAuth } from '@/features/auth'
import { messages } from '@/shared/lib/messages'

export function LoginPage() {
  const { status } = useAuth()
  const [params] = useSearchParams()
  if (status === 'loading') return <SessionLoading />
  // Đã đăng nhập mà mở /login → về trang định tới (hoặc trang của tôi).
  if (status === 'authenticated') return <Navigate to={safeNext(params.get('next'))} replace />
  return (
    <AuthCard title={messages.auth.loginTitle} subtitle={messages.auth.loginSubtitle}>
      <LoginForm />
      <p className="text-center text-sm text-ink-faint">{messages.auth.noAccount}</p>
    </AuthCard>
  )
}
