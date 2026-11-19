import { messages } from '@/shared/lib/messages'

export function NotFoundPage() {
  return (
    <main className="p-8">
      <h1 className="text-lg font-semibold">{messages.notFound.title}</h1>
    </main>
  )
}
