import { useAuth } from '@/features/auth'
import { HealthStatus } from '@/features/health'
import { messages } from '@/shared/lib/messages'

// "Trang của tôi" tối thiểu cho CF-01; nội dung theo vai trò thêm dần ở các nhánh sau.
export function HomePage() {
  const { user } = useAuth()
  if (!user) return null
  return (
    <div className="flex flex-col gap-4 px-5 py-3">
      <div className="flex items-baseline gap-3">
        <h1 className="text-lg font-semibold">{messages.home.greeting(user.fullName)}</h1>
        <p className="text-sm text-ink-faint">
          {messages.home.roleMeta(messages.roles[user.role])}
        </p>
      </div>
      <section className="flex max-w-xl flex-col gap-2 rounded-md border border-line bg-surface p-4">
        <h2 className="font-semibold">{messages.health.title}</h2>
        <HealthStatus />
      </section>
    </div>
  )
}
