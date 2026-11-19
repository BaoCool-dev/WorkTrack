import { FileQuestionMark } from 'lucide-react'
import { useNavigate } from 'react-router'
import { messages } from '@/shared/lib/messages'
import { Button, EmptyState } from '@/shared/ui'

export function NotFoundPage() {
  const navigate = useNavigate()
  return (
    <div className="flex flex-col px-5 py-6">
      <EmptyState
        icon={FileQuestionMark}
        title={messages.notFound.title}
        description={messages.notFound.description}
        actions={
          <Button variant="secondary" onClick={() => void navigate('/')}>
            {messages.notFound.backHome}
          </Button>
        }
      />
    </div>
  )
}
