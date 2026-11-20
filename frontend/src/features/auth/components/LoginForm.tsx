import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, useSearchParams } from 'react-router'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, FormAlert, Input } from '@/shared/ui'
import { useAuth } from '../hooks/useAuth'
import { safeNext } from '../model/access'
import { loginSchema, type LoginValues } from '../model/schemas'

const t = messages.auth

export function LoginForm() {
  const { login } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const [serverError, setServerError] = useState<string | null>(null)
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginValues>({ resolver: zodResolver(loginSchema) })

  const onSubmit = handleSubmit(async ({ email, password }) => {
    setServerError(null)
    try {
      await login(email, password)
      void navigate(safeNext(params.get('next')), { replace: true })
    } catch (err) {
      // Backend đã trả câu tiếng Việt (sai mật khẩu, bị khóa, chưa kích hoạt, quá nhiều lần thử…).
      setServerError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  })

  return (
    <form noValidate onSubmit={(e) => void onSubmit(e)} className="flex flex-col gap-4">
      <FormAlert message={serverError} />
      <Input
        label={t.email}
        type="email"
        autoComplete="username"
        placeholder={t.emailPlaceholder}
        autoFocus
        error={errors.email?.message}
        {...register('email')}
      />
      <Input
        label={t.password}
        type="password"
        autoComplete="current-password"
        error={errors.password?.message}
        {...register('password')}
      />
      <Button type="submit" loading={isSubmitting}>
        {isSubmitting ? t.submittingLogin : t.submitLogin}
      </Button>
    </form>
  )
}
