import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useCan } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import { formatTimestampDate } from '@/shared/lib/format'
import { messages } from '@/shared/lib/messages'
import { Button, ErrorState, FormAlert, Input, Skeleton, useToast } from '@/shared/ui'
import { useSettings, useUpdateSettings } from '../api/queries'
import {
  settingsFormSchema,
  type SettingsFormInput,
  type SettingsFormOutput,
} from '../model/schemas'
import { SP_HOURS_MAX, SP_HOURS_MIN, type Settings } from '../model/types'

const t = messages.tasks.settings

/** Tab "Cấu hình" ở trang Quản trị: hệ số quy đổi Story Point → giờ (B1, CF-06). */
export function SettingsPanel() {
  const query = useSettings()
  if (query.isPending)
    return (
      <div className="flex max-w-md flex-col gap-3">
        <Skeleton className="h-6 w-48" />
        <Skeleton className="h-10 w-full" />
      </div>
    )
  if (query.isError)
    return (
      <ErrorState
        title={t.loadError}
        errorRef={query.error instanceof ApiError ? query.error.code : undefined}
        onRetry={() => void query.refetch()}
      />
    )
  return <SettingsForm settings={query.data} />
}

function SettingsForm({ settings }: { settings: Settings }) {
  const canManage = useCan('settings.manage')
  const update = useUpdateSettings()
  const toast = useToast()
  const titleId = useId()
  const [serverError, setServerError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    setError,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<SettingsFormInput, unknown, SettingsFormOutput>({
    resolver: zodResolver(settingsFormSchema),
    defaultValues: { storyPointHours: String(settings.storyPointHours) },
  })

  const onSubmit = handleSubmit(async (v) => {
    setServerError(null)
    try {
      const saved = await update.mutateAsync(v)
      reset({ storyPointHours: String(saved.storyPointHours) })
      toast.show(t.saved)
    } catch (err) {
      if (!(err instanceof ApiError)) setServerError(messages.errors.unknown)
      else if (err.code === 'SETTING_INVALID' || err.code === 'VALIDATION_FAILED')
        setError('storyPointHours', { message: err.message }, { shouldFocus: true })
      else setServerError(err.message)
    }
  })

  return (
    <form
      aria-labelledby={titleId}
      noValidate
      onSubmit={(e) => void onSubmit(e)}
      className="flex max-w-md flex-col gap-3"
    >
      <h2 id={titleId} className="text-base font-semibold text-ink">
        {t.title}
      </h2>
      <p className="text-base text-ink-muted">{t.intro}</p>
      <FormAlert message={serverError} />
      <Input
        type="number"
        inputMode="numeric"
        min={SP_HOURS_MIN}
        max={SP_HOURS_MAX}
        step={1}
        label={t.hours}
        hint={canManage ? t.hint : t.readOnly}
        error={errors.storyPointHours?.message}
        disabled={!canManage}
        {...register('storyPointHours')}
      />
      <p className="num text-sm text-ink-faint">
        {t.updatedAt(formatTimestampDate(settings.updatedAt))}
      </p>
      {canManage && (
        <div>
          <Button type="submit" loading={isSubmitting}>
            {t.save}
          </Button>
        </div>
      )}
    </form>
  )
}
