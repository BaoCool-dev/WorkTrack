import { useState } from 'react'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, useToast } from '@/shared/ui'

const t = messages.admin.org

/** Xác nhận xóa; lỗi từ server (vd. còn phòng ban con) hiện ngay trong hộp thoại. */
export function ConfirmDeleteDialog({
  name,
  onConfirm,
  onClose,
}: {
  name: string
  onConfirm: () => Promise<unknown>
  onClose: () => void
}) {
  const toast = useToast()
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const confirm = async () => {
    setPending(true)
    setError(null)
    try {
      await onConfirm()
      toast.show(t.deleted)
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : messages.errors.unknown)
      setPending(false)
    }
  }
  return (
    <Dialog
      open
      onClose={onClose}
      title={t.confirmDeleteTitle(name)}
      description={t.confirmDelete}
      body={error ? <FormAlert message={error} /> : undefined}
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button variant="danger" loading={pending} onClick={() => void confirm()}>
        {t.remove}
      </Button>
    </Dialog>
  )
}
