import { z } from 'zod'
import { messages } from '@/shared/lib/messages'
import {
  COLUMN_NAME_MAX,
  PROJECT_DESCRIPTION_MAX,
  PROJECT_NAME_MAX,
  PROJECT_STATUSES,
  STATUS_GROUPS,
  type ProjectStatus,
  type StatusGroup,
} from './types'

const t = messages.projects

const runeLen = (v: string) => Array.from(v).length

// '' từ ô ngày để trống → null khi gửi (backend nhận "YYYY-MM-DD" hoặc null).
const optionalDate = z
  .string()
  .regex(/^(\d{4}-\d{2}-\d{2})?$/)
  .transform((v) => (v === '' ? null : v))

// Khớp domain/project.go: tên 1–150 ký tự (gộp khoảng trắng ở server), mô tả ≤ 2000.
export const projectFormSchema = z
  .object({
    name: z
      .string()
      .trim()
      .min(1, t.form.nameRequired)
      .refine((v) => runeLen(v) <= PROJECT_NAME_MAX, t.form.nameTooLong),
    description: z
      .string()
      .trim()
      .refine((v) => runeLen(v) <= PROJECT_DESCRIPTION_MAX, t.form.descriptionTooLong),
    status: z.enum(PROJECT_STATUSES as [ProjectStatus, ...ProjectStatus[]]),
    startDate: optionalDate,
    endDate: optionalDate,
  })
  .refine((v) => !v.startDate || !v.endDate || v.endDate >= v.startDate, {
    message: t.form.datesInvalid,
    path: ['endDate'],
  })
export type ProjectFormInput = z.input<typeof projectFormSchema>
export type ProjectFormValues = z.output<typeof projectFormSchema>

export const columnFormSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, t.columns.nameRequired)
    .refine((v) => runeLen(v) <= COLUMN_NAME_MAX, t.columns.nameTooLong),
  statusGroup: z.enum(STATUS_GROUPS as [StatusGroup, ...StatusGroup[]]),
})
export type ColumnFormInput = z.input<typeof columnFormSchema>
export type ColumnFormValues = z.output<typeof columnFormSchema>
