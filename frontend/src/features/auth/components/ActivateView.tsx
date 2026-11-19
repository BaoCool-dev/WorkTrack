import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Link2Off } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, useSearchParams } from 'react-router'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, EmptyState, Input, Skeleton, useToast } from '@/shared/ui'
import { authApi } from '../api/auth'
import { activateSchema, type ActivateValues } from '../model/schemas'
import { AuthCard } from './AuthCard'
import { FormAlert } from './FormAlert'

const t = messages.auth

/**
 * /activate?token=… — token đọc một lần rồi xóa khỏi thanh địa chỉ
 * (không nằm lại trong lịch sử trình duyệt hay bị chia sẻ nhầm).
 */
export function ActivateView() {
  const [params, setParams] = useSearchParams()
  const [token] = useState(() => params.get('token') ?? '')

  useEffect(() => {
    if (params.has('token')) setParams({}, { replace: true })
  }, [params, setParams])

  return (
    <AuthCard title={t.activateTitle} subtitle={t.activateSubtitle}>
      {token ? <VerifiedActivation token={token} /> : <LinkProblem title={t.linkMissingTitle} />}
    </AuthCard>
  )
}

function LinkProblem({ title, description = t.linkMissing }: { title: string; description?: string }) {
  const navigate = useNavigate()
  return (
    <EmptyState
      role="alert"
      tone="error"
      icon={Link2Off}
      title={title}
      description={description}
      actions={
        <Button
          variant="secondary"
          onClick={() => {
            void navigate('/login')
          }}
        >
          {t.goToLogin}
        </Button>
      }
    />
  )
}

function VerifiedActivation({ token }: { token: string }) {
  const verify = useQuery({
    queryKey: ['auth', 'activation', token],
    queryFn: () => authApi.verifyActivation(token),
    retry: false,
    staleTime: Infinity,
    gcTime: 0,
  })

  if (verify.isPending) {
    return (
      <div role="status" aria-label={t.verifying} className="flex flex-col gap-4">
        <Skeleton className="h-5 w-48" />
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
      </div>
    )
  }
  if (verify.isError) {
    const err = verify.error
    return (
      <LinkProblem
        title={t.linkInvalidTitle}
        description={err instanceof ApiError ? err.message : messages.errors.unknown}
      />
    )
  }
  return <ActivateForm token={token} email={verify.data.email} fullName={verify.data.fullName} />
}

function ActivateForm({ token, email, fullName }: { token: string; email: string; fullName: string }) {
  const navigate = useNavigate()
  const toast = useToast()
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ActivateValues>({ resolver: zodResolver(activateSchema) })
  const activate = useMutation({
    mutationFn: (password: string) => authApi.activate(token, password),
    onSuccess: () => {
      toast.show(t.activated)
      void navigate('/login', { replace: true })
    },
  })

  const serverError = activate.error
    ? activate.error instanceof ApiError
      ? activate.error.message
      : messages.errors.unknown
    : null

  return (
    <form
      noValidate
      onSubmit={(e) =>
        void handleSubmit(({ password }) => {
          activate.mutate(password)
        })(e)
      }
      className="flex flex-col gap-4"
    >
      <p className="text-sm text-ink-muted">
        {t.activateFor}: <span className="font-semibold text-ink">{fullName}</span> · {email}
      </p>
      <FormAlert message={serverError} />
      {/* Trường ẩn giúp trình quản lý mật khẩu lưu đúng tài khoản. */}
      <input type="hidden" autoComplete="username" value={email} readOnly />
      <Input
        label={t.newPassword}
        type="password"
        autoComplete="new-password"
        autoFocus
        hint={t.passwordHint}
        error={errors.password?.message}
        {...register('password')}
      />
      <Input
        label={t.confirmPassword}
        type="password"
        autoComplete="new-password"
        error={errors.confirm?.message}
        {...register('confirm')}
      />
      <Button type="submit" loading={activate.isPending}>
        {activate.isPending ? t.submittingActivate : t.submitActivate}
      </Button>
    </form>
  )
}
