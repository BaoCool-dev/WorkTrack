import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, Input, Select, useToast, type SelectOption } from '@/shared/ui'
import { nameSchema, type NameFormInput, type NameFormValues } from '../model/schemas'

const t = messages.admin

const PARENT_ERRORS = new Set(['DEPARTMENT_CYCLE', 'DEPARTMENT_NOT_FOUND'])

function isNameError(code: string): boolean {
  return code.endsWith('_NAME_TAKEN') || code === 'ORG_NAME_INVALID'
}

export interface NameDialogProps {
  title: string
  defaultValues: NameFormInput
  /** Có thì hiện ô chọn "Phòng ban cha". */
  parentOptions?: readonly SelectOption[]
  isEdit: boolean
  save: (values: NameFormValues) => Promise<unknown>
  onClose: () => void
}

/** Form một trường tên (+ phòng ban cha) dùng chung cho phòng ban, chức danh, kỹ năng. */
export function NameDialog({
  title,
  defaultValues,
  parentOptions,
  isEdit,
  save,
  onClose,
}: NameDialogProps) {
  const formId = useId()
  const toast = useToast()
  const [serverError, setServerError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<NameFormInput, unknown, NameFormValues>({
    resolver: zodResolver(nameSchema),
    defaultValues,
  })

  const onSubmit = handleSubmit(async (values) => {
    setServerError(null)
    try {
      await save(values)
      toast.show(isEdit ? t.org.updated : t.org.created)
      onClose()
    } catch (err) {
      if (!(err instanceof ApiError)) {
        setServerError(messages.errors.unknown)
      } else if (isNameError(err.code)) {
        setError('name', { message: err.message }, { shouldFocus: true })
      } else if (parentOptions && PARENT_ERRORS.has(err.code)) {
        setError('parentId', { message: err.message })
      } else {
        setServerError(err.message)
      }
    }
  })

  return (
    <Dialog
      open
      onClose={onClose}
      title={title}
      body={
        <form
          id={formId}
          noValidate
          onSubmit={(e) => void onSubmit(e)}
          className="flex flex-col gap-3"
        >
          <FormAlert message={serverError} />
          <Input label={t.org.name} autoFocus error={errors.name?.message} {...register('name')} />
          {parentOptions && (
            <Select
              label={t.departments.parent}
              options={parentOptions}
              error={errors.parentId?.message}
              {...register('parentId')}
            />
          )}
        </form>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button type="submit" form={formId} loading={isSubmitting}>
        {t.org.save}
      </Button>
    </Dialog>
  )
}
