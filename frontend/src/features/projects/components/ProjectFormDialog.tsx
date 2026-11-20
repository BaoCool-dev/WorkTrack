import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { ApiError } from '@/shared/api/http'
import { cn } from '@/shared/lib/cn'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, Input, Select, useToast } from '@/shared/ui'
import { projectFormSchema, type ProjectFormInput, type ProjectFormValues } from '../model/schemas'
import { PROJECT_STATUSES, type Project, type ProjectInput } from '../model/types'

const t = messages.projects

const NAME_ERRORS = new Set(['PROJECT_NAME_TAKEN', 'PROJECT_NAME_INVALID'])

export interface ProjectFormDialogProps {
  /** Có thì là sửa; không thì tạo mới. */
  project?: Project
  save: (body: ProjectInput) => Promise<unknown>
  onClose: () => void
}

/** Tạo/sửa dự án (CF-03). Tạo mới luôn ở trạng thái "Mới tạo" nên không hiện ô trạng thái. */
export function ProjectFormDialog({ project, save, onClose }: ProjectFormDialogProps) {
  const formId = useId()
  const descId = useId()
  const toast = useToast()
  const isEdit = project !== undefined
  const [serverError, setServerError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<ProjectFormInput, unknown, ProjectFormValues>({
    resolver: zodResolver(projectFormSchema),
    defaultValues: {
      name: project?.name ?? '',
      description: project?.description ?? '',
      status: project?.status ?? 'new',
      startDate: project?.startDate ?? '',
      endDate: project?.endDate ?? '',
    },
  })

  const onSubmit = handleSubmit(async (v) => {
    setServerError(null)
    const body: ProjectInput = {
      name: v.name,
      description: v.description,
      startDate: v.startDate,
      endDate: v.endDate,
      ...(isEdit ? { status: v.status } : {}),
    }
    try {
      await save(body)
      toast.show(isEdit ? t.form.updated : t.form.created)
      onClose()
    } catch (err) {
      if (!(err instanceof ApiError)) setServerError(messages.errors.unknown)
      else if (NAME_ERRORS.has(err.code))
        setError('name', { message: err.message }, { shouldFocus: true })
      else if (err.code === 'PROJECT_DATES_INVALID') setError('endDate', { message: err.message })
      else if (err.code === 'PROJECT_DESCRIPTION_INVALID')
        setError('description', { message: err.message })
      else setServerError(err.message)
    }
  })

  const descError = errors.description?.message
  return (
    <Dialog
      open
      onClose={onClose}
      size="lg"
      title={isEdit ? t.form.editTitle : t.form.createTitle}
      description={isEdit ? undefined : t.form.createHint}
      body={
        <form
          id={formId}
          noValidate
          onSubmit={(e) => void onSubmit(e)}
          className="flex flex-col gap-3"
        >
          <FormAlert message={serverError} />
          <Input label={t.form.name} autoFocus error={errors.name?.message} {...register('name')} />
          <div className="flex flex-col gap-1">
            <label htmlFor={descId} className="text-sm font-semibold text-ink-muted">
              {t.form.description}
            </label>
            <textarea
              id={descId}
              rows={4}
              aria-invalid={descError ? true : false}
              aria-describedby={descError ? `${descId}-note` : undefined}
              className={cn(
                'rounded-md border bg-surface px-3 py-2 text-base text-ink placeholder:text-ink-faint',
                'focus:ring-1 focus:outline-none',
                descError
                  ? 'border-bad focus:border-bad focus:ring-bad'
                  : 'border-line-strong focus:border-accent focus:ring-accent',
              )}
              {...register('description')}
            />
            {descError && (
              <p id={`${descId}-note`} className="text-sm text-bad">
                {descError}
              </p>
            )}
          </div>
          {isEdit && (
            <Select
              label={t.form.status}
              options={PROJECT_STATUSES.map((s) => ({ value: s, label: t.status[s] }))}
              error={errors.status?.message}
              {...register('status')}
            />
          )}
          <div className="grid grid-cols-2 gap-3">
            <Input
              type="date"
              label={t.form.startDate}
              error={errors.startDate?.message}
              {...register('startDate')}
            />
            <Input
              type="date"
              label={t.form.endDate}
              error={errors.endDate?.message}
              {...register('endDate')}
            />
          </div>
        </form>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button type="submit" form={formId} loading={isSubmitting}>
        {isEdit ? t.form.save : t.form.create}
      </Button>
    </Dialog>
  )
}
