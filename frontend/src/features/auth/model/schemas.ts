import { z } from 'zod'
import { messages } from '@/shared/lib/messages'

const t = messages.auth

export const loginSchema = z.object({
  email: z.string().trim().min(1, t.emailRequired).pipe(z.email(t.emailInvalid)),
  password: z.string().min(1, t.passwordRequired),
})
export type LoginValues = z.infer<typeof loginSchema>

const encoder = new TextEncoder()

// Khớp backend: ≥ 8 ký tự (rune), ≤ 72 byte (giới hạn bcrypt), có cả chữ và số.
export const passwordSchema = z
  .string()
  .refine((v) => Array.from(v).length >= 8, t.passwordTooShort)
  .refine((v) => encoder.encode(v).length <= 72, t.passwordTooLong)
  .refine((v) => /\p{L}/u.test(v) && /\p{N}/u.test(v), t.passwordNeedsLetterAndDigit)

export const activateSchema = z
  .object({ password: passwordSchema, confirm: z.string() })
  .refine((v) => v.password === v.confirm, { error: t.passwordMismatch, path: ['confirm'] })
export type ActivateValues = z.infer<typeof activateSchema>
