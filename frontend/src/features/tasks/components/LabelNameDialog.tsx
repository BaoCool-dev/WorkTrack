import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, Input, useToast } from '@/shared/ui'
import { labelFormSchema, type LabelFormValues } from '../model/schemas'

const t = messages.tasks.labels

/** Mã lỗi server gắn vào ô tên; mã khác hiện ở FormAlert. */
const NAME_ERRORS = new Set(['LABEL_NAME_TAKEN', 'VALIDATION_FAILED'])

export interface LabelNameDialogProps {
  isEdit: boolean
  defaultName: string
  save: (name: string) => Promise<unknown>
  onClose: () => void
}

export function LabelNameDialog({ isEdit, defaultName, save, onClose }: LabelNameDialogProps) {
  const formId = useId()
  const toast = useToast()
  const [serverError, setServerError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<LabelFormValues>({
    resolver: zodResolver(labelFormSchema),
    defaultValues: { name: defaultName },
  })

  const onSubmit = handleSubmit(async ({ name }) => {
    setServerError(null)
    try {
      await save(name)
      toast.show(isEdit ? t.updated : t.created)
      onClose()
    } catch (err) {
      if (!(err instanceof ApiError)) setServerError(messages.errors.unknown)
      else if (NAME_ERRORS.has(err.code))
        setError('name', { message: err.message }, { shouldFocus: true })
      else setServerError(err.message)
    }
  })

  return (
    <Dialog
      open
      onClose={onClose}
      title={isEdit ? t.editTitle : t.createTitle}
      body={
        <form
          id={formId}
          noValidate
          onSubmit={(e) => void onSubmit(e)}
          className="flex flex-col gap-3"
        >
          <FormAlert message={serverError} />
          <Input label={t.name} autoFocus error={errors.name?.message} {...register('name')} />
        </form>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button type="submit" form={formId} loading={isSubmitting}>
        {t.save}
      </Button>
    </Dialog>
  )
}
