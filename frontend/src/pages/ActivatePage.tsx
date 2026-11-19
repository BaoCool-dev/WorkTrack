import { ActivateView, AuthCard } from '@/features/auth'
import { messages } from '@/shared/lib/messages'

export function ActivatePage() {
  return (
    <AuthCard title={messages.auth.activateTitle} subtitle={messages.auth.activateSubtitle}>
      <ActivateView />
    </AuthCard>
  )
}
