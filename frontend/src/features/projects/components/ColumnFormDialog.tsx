import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, Input, Select, useToast } from '@/shared/ui'
import { columnFormSchema, type ColumnFormInput, type ColumnFormValues } from '../model/schemas'
import { STATUS_GROUPS, type BoardColumn } from '../model/types'

const t = messages.projects

const NAME_ERRORS = new Set(['COLUMN_NAME_TAKEN', 'COLUMN_NAME_INVALID'])
const GROUP_ERRORS = new Set(['COLUMN_STATUS_GROUP_INVALID', 'BOARD_NEEDS_DONE_COLUMN'])

/** Thêm/sửa một cột Kanban. */
export function ColumnFormDialog({
  column,
  save,
  onClose,
}: {
  column?: BoardColumn
  save: (v: ColumnFormValues) => Promise<unknown>
  onClose: () => void
}) {
  const formId = useId()
  const toast = useToast()
  const isEdit = column !== undefined
  const [serverError, setServerError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ColumnFormInput, unknown, ColumnFormValues>({
    resolver: zodResolver(columnFormSchema),
    defaultValues: { name: column?.name ?? '', statusGroup: column?.statusGroup ?? 'todo' },
  })

  const onSubmit = handleSubmit(async (v) => {
    setServerError(null)
    try {
      await save(v)
      toast.show(isEdit ? t.columns.updated : t.columns.created)
      onClose()
    } catch (err) {
      if (!(err instanceof ApiError)) setServerError(messages.errors.unknown)
      else if (NAME_ERRORS.has(err.code))
        setError('name', { message: err.message }, { shouldFocus: true })
      else if (GROUP_ERRORS.has(err.code)) setError('statusGroup', { message: err.message })
      else setServerError(err.message)
    }
  })

  return (
    <Dialog
      open
      onClose={onClose}
      title={isEdit ? t.columns.editTitle : t.columns.createTitle}
      body={
        <form
          id={formId}
          noValidate
          onSubmit={(e) => void onSubmit(e)}
          className="flex flex-col gap-3"
        >
          <FormAlert message={serverError} />
          <Input
            label={t.columns.name}
            autoFocus
            error={errors.name?.message}
            {...register('name')}
          />
          <Select
            label={t.columns.group}
            hint={t.columns.groupHint}
            options={STATUS_GROUPS.map((g) => ({ value: g, label: t.groups[g] }))}
            error={errors.statusGroup?.message}
            {...register('statusGroup')}
          />
        </form>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button type="submit" form={formId} loading={isSubmitting}>
        {t.columns.save}
      </Button>
    </Dialog>
  )
}
