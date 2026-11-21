import { OctagonAlert } from 'lucide-react'

import { messages } from '@/shared/lib/messages'
import { Badge, Button, Dialog } from '@/shared/ui'

const t = messages.kanban.wip

interface Props {
  /** Người đã đủ WIP (người nhận thẻ), null = đóng hộp thoại. */
  person: { id: string; name: string } | null
  wipLimit: number
  onClose: () => void
  /** Lọc bảng theo người này để thấy những việc đang làm của họ. */
  onViewDoing: (personId: string) => void
}

/** Hộp thoại chặn khi đủ WIP (design-system §6): badge 5/5, ghi chú việc con, nút xem việc đang làm. */
export function WipFullDialog({ person, wipLimit, onClose, onViewDoing }: Props) {
  return (
    <Dialog
      open={person !== null}
      onClose={onClose}
      title={t.fullTitle}
      icon={<OctagonAlert size={20} strokeWidth={2} className="text-bad" aria-hidden />}
      description={person ? t.fullBody(person.name, wipLimit) : undefined}
      body={
        <div className="flex items-center gap-2 text-sm text-ink-muted">
          <Badge tone="bad">
            <span className="num">
              {wipLimit}/{wipLimit}
            </span>
          </Badge>
          <span>{t.subtaskNote}</span>
        </div>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {t.close}
      </Button>
      <Button
        onClick={() => {
          if (person) onViewDoing(person.id)
        }}
      >
        {t.viewDoing(wipLimit)}
      </Button>
    </Dialog>
  )
}
