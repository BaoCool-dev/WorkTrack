import { HealthStatus } from '@/features/health'
import { messages } from '@/shared/lib/messages'

// Trang tạm của chore/scaffold — thay bằng màn hình thật ở các nhánh tính năng.
export function HomePage() {
  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-4 p-8">
      <h1 className="text-lg font-semibold">{messages.appName}</h1>
      <section className="flex flex-col gap-2 rounded-md border border-line bg-surface p-4">
        <h2 className="font-semibold">{messages.health.title}</h2>
        <HealthStatus />
      </section>
    </main>
  )
}
