import { z } from 'zod'
import { messages } from '@/shared/lib/messages'
import {
  LABEL_NAME_MAX,
  SP_HOURS_MAX,
  SP_HOURS_MIN,
  STORY_POINTS,
  TASK_DESCRIPTION_MAX,
  TASK_LABELS_MAX,
  TASK_TITLE_MAX,
} from './types'

const t = messages.tasks.form

const runeLen = (v: string) => Array.from(v).length

// '' từ ô ngày để trống → null khi gửi.
const optionalDate = z
  .string()
  .regex(/^(\d{4}-\d{2}-\d{2})?$/)
  .transform((v) => (v === '' ? null : v))

const storyPoint = z
  .string()
  .refine(
    (v) => v === '' || (STORY_POINTS as readonly number[]).includes(Number(v)),
    t.storyPointInvalid,
  )
  .transform((v) => (v === '' ? null : Number(v)))

// Khớp domain/task.go: tiêu đề 1–200 ký tự, mô tả ≤ 5000, tối đa 10 nhãn, SP Fibonacci 1–21.
export const taskFormSchema = z.object({
  title: z
    .string()
    .trim()
    .min(1, t.titleRequired)
    .refine((v) => runeLen(v) <= TASK_TITLE_MAX, t.titleTooLong),
  description: z
    .string()
    .trim()
    .refine((v) => runeLen(v) <= TASK_DESCRIPTION_MAX, t.descriptionTooLong),
  storyPoint,
  assigneeId: z.string().transform((v) => (v === '' ? null : v)),
  dueDate: optionalDate,
  labelIds: z.array(z.string()).max(TASK_LABELS_MAX, t.tooManyLabels),
})

export type TaskFormInput = z.input<typeof taskFormSchema>
export type TaskFormOutput = z.output<typeof taskFormSchema>

const tl = messages.tasks.labels

export const labelFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, tl.nameRequired)
    .refine((v) => runeLen(v) <= LABEL_NAME_MAX, tl.nameTooLong),
})

export type LabelFormValues = z.infer<typeof labelFormSchema>

const ts = messages.tasks.settings

export const settingsFormSchema = z.object({
  storyPointHours: z
    .string()
    .regex(/^\d+$/, ts.invalid)
    .transform(Number)
    .refine((n) => n >= SP_HOURS_MIN && n <= SP_HOURS_MAX, ts.invalid),
})

export type SettingsFormInput = z.input<typeof settingsFormSchema>
export type SettingsFormOutput = z.output<typeof settingsFormSchema>
