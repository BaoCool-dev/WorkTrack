import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { useProjectMembers } from '@/features/projects'
import { ApiError } from '@/shared/api/http'
import { cn } from '@/shared/lib/cn'
import { formatHours } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import {
  Button,
  Dialog,
  FormAlert,
  Input,
  Select,
  useToast,
  type SelectOption,
} from '@/shared/ui'
import { useLabels, useSettings } from '../api/queries'
import { taskFormSchema, type TaskFormInput, type TaskFormOutput } from '../model/schemas'
import { STORY_POINTS, type Task, type TaskInput } from '../model/types'

const t = messages.tasks.form

type FieldName = keyof TaskFormInput

/** Mã lỗi server → ô cần gắn lỗi; mã khác hiện ở FormAlert. */
const FIELD_ERRORS: Record<string, FieldName> = {
  TASK_TITLE_INVALID: 'title',
  TASK_DESCRIPTION_INVALID: 'description',
  TASK_STORY_POINT_INVALID: 'storyPoint',
  TASK_TOO_MANY_LABELS: 'labelIds',
  LABEL_NOT_FOUND: 'labelIds',
  TASK_ASSIGNEE_INVALID: 'assigneeId',
}

export interface TaskFormDialogProps {
  projectId: string
  /** Có thì là sửa; không thì tạo mới. */
  task?: Task
  save: (body: TaskInput) => Promise<unknown>
  onClose: () => void
}

/** Tạo/sửa công việc cha (CF-06) — chỉ Manager phụ trách dự án (server kiểm lại). */
export function TaskFormDialog({ projectId, task, save, onClose }: TaskFormDialogProps) {
  const formId = useId()
  const descId = useId()
  const labelsId = useId()
  const toast = useToast()
  const isEdit = task !== undefined
  const [serverError, setServerError] = useState<string | null>(null)
  const members = useProjectMembers(projectId)
  const labels = useLabels()
  const settings = useSettings()
  const {
    register,
    handleSubmit,
    setError,
    setValue,
    control,
    formState: { errors, isSubmitting },
  } = useForm<TaskFormInput, unknown, TaskFormOutput>({
    resolver: zodResolver(taskFormSchema),
    defaultValues: {
      title: task?.title ?? '',
      description: task?.description ?? '',
      storyPoint: task?.storyPoint === null || task === undefined ? '' : String(task.storyPoint),
      assigneeId: task?.assignee?.id ?? '',
      dueDate: task?.dueDate ?? '',
      labelIds: task?.labels.map((l) => l.id) ?? [],
    },
  })

  const onSubmit = handleSubmit(async (v) => {
    setServerError(null)
    try {
      await save(v)
      toast.show(isEdit ? t.updated : t.created)
      onClose()
    } catch (err) {
      if (!(err instanceof ApiError)) {
        setServerError(messages.errors.unknown)
        return
      }
      const field = FIELD_ERRORS[err.code]
      if (field) setError(field, { message: err.message }, { shouldFocus: true })
      else setServerError(err.message)
    }
  })

  // Người nhận: thành viên đang hoạt động (Manager/Member); giữ người đang nhận nếu đã rời dự án.
  const assigneeOptions: SelectOption[] = [{ value: '', label: t.noAssignee }]
  for (const m of members.data ?? []) {
    if (m.status === 'active' && (m.role === 'manager' || m.role === 'member'))
      assigneeOptions.push({ value: m.userId, label: m.fullName })
  }
  if (task?.assignee && !assigneeOptions.some((o) => o.value === task.assignee?.id))
    assigneeOptions.push({ value: task.assignee.id, label: task.assignee.name })

  const sp = useWatch({ control, name: 'storyPoint' })
  const perSp = settings.data?.storyPointHours
  let estimateHint: string | undefined
  if (perSp !== undefined)
    estimateHint =
      sp === ''
        ? t.estimateHintEmpty(formatHours(perSp))
        : t.estimateHint(formatHours(Number(sp) * perSp), formatHours(perSp))

  const selected = useWatch({ control, name: 'labelIds' })
  const toggleLabel = (id: string, on: boolean) => {
    setValue('labelIds', on ? [...selected, id] : selected.filter((x) => x !== id), {
      shouldValidate: true,
    })
  }

  const descError = errors.description?.message
  const labelsError = errors.labelIds?.message
  return (
    <Dialog
      open
      onClose={onClose}
      size="lg"
      title={isEdit ? t.editTitle : t.createTitle}
      description={isEdit ? undefined : t.createHint}
      body={
        <form
          id={formId}
          noValidate
          onSubmit={(e) => void onSubmit(e)}
          className="flex flex-col gap-3"
        >
          <FormAlert message={serverError} />
          <Input label={t.title} autoFocus error={errors.title?.message} {...register('title')} />
          <div className="flex flex-col gap-1">
            <label htmlFor={descId} className="text-sm font-semibold text-ink-muted">
              {t.description}
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
          <div className="grid grid-cols-2 gap-3">
            <Select
              label={t.storyPoint}
              options={[
                { value: '', label: t.noStoryPoint },
                ...STORY_POINTS.map((n) => ({ value: String(n), label: String(n) })),
              ]}
              error={errors.storyPoint?.message}
              hint={estimateHint}
              {...register('storyPoint')}
            />
            <Input
              type="date"
              label={t.dueDate}
              error={errors.dueDate?.message}
              {...register('dueDate')}
            />
          </div>
          <Select
            label={t.assignee}
            options={assigneeOptions}
            error={errors.assigneeId?.message}
            {...register('assigneeId')}
          />
          <fieldset
            className="flex flex-col gap-1"
            aria-describedby={`${labelsId}-note`}
            aria-invalid={labelsError ? true : false}
          >
            <legend className="mb-1 text-sm font-semibold text-ink-muted">{t.labels}</legend>
            {labels.data && labels.data.length === 0 ? (
              <p className="text-sm text-ink-faint">{t.noLabels}</p>
            ) : (
              <div className="flex flex-wrap gap-x-4 gap-y-2">
                {(labels.data ?? []).map((l) => (
                  <label key={l.id} className="flex min-h-10 items-center gap-2 text-base text-ink">
                    <input
                      type="checkbox"
                      className="size-4 accent-accent"
                      checked={selected.includes(l.id)}
                      onChange={(e) => {
                        toggleLabel(l.id, e.target.checked)
                      }}
                    />
                    {l.name}
                  </label>
                ))}
              </div>
            )}
            <p
              id={`${labelsId}-note`}
              className={cn('text-sm', labelsError ? 'text-bad' : 'text-ink-faint')}
            >
              {labelsError ?? t.labelsHint}
            </p>
          </fieldset>
        </form>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button type="submit" form={formId} loading={isSubmitting}>
        {isEdit ? t.save : t.create}
      </Button>
    </Dialog>
  )
}
