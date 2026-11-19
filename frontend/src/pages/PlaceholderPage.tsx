import { Construction } from 'lucide-react'
import { messages } from '@/shared/lib/messages'
import { EmptyState } from '@/shared/ui'

/** Giữ chỗ cho route đã có trong menu nhưng thuộc nhánh tính năng sau. */
export function PlaceholderPage({ title }: { title: string }) {
  return (
    <div className="flex flex-col gap-4 px-5 py-3">
      <h1 className="text-lg font-semibold">{title}</h1>
      <EmptyState
        icon={Construction}
        title={messages.placeholder.title}
        description={messages.placeholder.description}
      />
    </div>
  )
}
