import { z } from 'zod'
import { messages } from '@/shared/lib/messages'
import { ROLES } from './types'

const t = messages.admin

const roleSchema = z.enum(ROLES as ['admin', 'manager', 'member', 'ceo'])

// '' từ ô chọn "Chưa chọn" → null khi gửi.
const optionalId = z.string().transform((v) => (v === '' ? null : v))

// Khớp backend: email required,email,max=254; fullName required,max=100 (đếm theo ký tự).
export const userFormSchema = z.object({
  email: z
    .string()
    .trim()
    .min(1, t.userForm.emailRequired)
    .max(254, t.userForm.emailTooLong)
    .pipe(z.email(t.userForm.emailInvalid)),
  fullName: z
    .string()
    .trim()
    .min(1, t.userForm.fullNameRequired)
    .refine((v) => Array.from(v).length <= 100, t.userForm.fullNameTooLong),
  role: roleSchema,
  departmentId: optionalId,
  jobTitleId: optionalId,
})
export type UserFormInput = z.input<typeof userFormSchema>
export type UserFormValues = z.output<typeof userFormSchema>

export const nameSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, t.org.nameRequired)
    .refine((v) => Array.from(v).length <= 200, t.org.nameTooLong),
  parentId: optionalId.optional(),
})
export type NameFormInput = z.input<typeof nameSchema>
export type NameFormValues = z.output<typeof nameSchema>
