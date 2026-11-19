import { Lock } from 'lucide-react'
import { useNavigate } from 'react-router'
import { messages } from '@/shared/lib/messages'
import { Button, EmptyState } from '@/shared/ui'
import { useAuth } from '../hooks/useAuth'

/** §7 "Không có quyền": icon khóa, lý do theo vai trò, nút về trang của tôi. */
export function ForbiddenState() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const role = user ? messages.roles[user.role] : ''
  return (
    <div className="flex flex-col px-5 py-6">
      <EmptyState
        icon={Lock}
        title={messages.access.forbiddenTitle}
        description={messages.access.forbiddenReason(role)}
        actions={
          <Button
            variant="secondary"
            onClick={() => {
              void navigate('/')
            }}
          >
            {messages.access.backToMine}
          </Button>
        }
      />
    </div>
  )
}
