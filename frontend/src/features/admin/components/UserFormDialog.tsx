import { zodResolver } from '@hookform/resolvers/zod'
import { useId, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useAuth, useCan } from '@/features/auth'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, Input, Select, Skeleton, useToast } from '@/shared/ui'
import { useCatalog, useCreateUser, useDepartments, useUpdateUser } from '../api/queries'
import { departmentOptions, flattenDepartments } from '../model/departmentTree'
import { userFormSchema, type UserFormInput, type UserFormValues } from '../model/schemas'
import { ROLES, type AdminUser, type CatalogItem, type Department } from '../model/types'

const t = messages.admin.userForm

const FIELD_ERRORS = {
  DEPARTMENT_NOT_FOUND: 'departmentId',
  JOB_TITLE_NOT_FOUND: 'jobTitleId',
  USER_CANNOT_CHANGE_OWN_ROLE: 'role',
} as const satisfies Record<string, keyof UserFormInput>

function fieldFor(code: string): keyof UserFormInput | undefined {
  return code in FIELD_ERRORS ? FIELD_ERRORS[code as keyof typeof FIELD_ERRORS] : undefined
}

export interface UserFormDialogProps {
  /** Không có = thêm mới. */
  user?: AdminUser
  onClose: () => void
}

/** Thêm/sửa nhân sự (CF-02). Chờ tải danh mục để giá trị mặc định của ô chọn khớp. */
export function UserFormDialog({ user, onClose }: UserFormDialogProps) {
  const departments = useDepartments()
  const jobTitles = useCatalog('job-titles')
  if (departments.data && jobTitles.data)
    return (
      <UserForm
        user={user}
        departments={departments.data}
        jobTitles={jobTitles.data}
        onClose={onClose}
      />
    )
  return (
    <Dialog
      open
      onClose={onClose}
      title={user ? t.editTitle : t.createTitle}
      body={
        <div className="flex flex-col gap-3" aria-busy="true">
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
          <Skeleton className="h-9 w-full" />
        </div>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
    </Dialog>
  )
}

function UserForm({
  user,
  departments,
  jobTitles,
  onClose,
}: {
  user?: AdminUser
  departments: Department[]
  jobTitles: CatalogItem[]
  onClose: () => void
}) {
  const formId = useId()
  const toast = useToast()
  const me = useAuth().user
  const canAssignCeo = useCan('users.assign_ceo')
  const create = useCreateUser()
  const update = useUpdateUser()
  const [serverError, setServerError] = useState<string | null>(null)
  const isSelf = user !== undefined && me?.id === user.id

  const deptOptions = useMemo(
    () => [
      { value: '', label: t.noneOption },
      ...departmentOptions(flattenDepartments(departments)),
    ],
    [departments],
  )
  const jobOptions = [
    { value: '', label: t.noneOption },
    ...jobTitles.map((j) => ({ value: j.id, label: j.name })),
  ]
  const roleOptions = ROLES.filter((r) => r !== 'ceo' || canAssignCeo || user?.role === 'ceo').map(
    (r) => ({ value: r, label: messages.roles[r] }),
  )

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<UserFormInput, unknown, UserFormValues>({
    resolver: zodResolver(userFormSchema),
    defaultValues: {
      email: user?.email ?? '',
      fullName: user?.fullName ?? '',
      role: user?.role ?? 'member',
      departmentId: user?.department?.id ?? '',
      jobTitleId: user?.jobTitle?.id ?? '',
    },
  })

  const onSubmit = handleSubmit(async (values) => {
    setServerError(null)
    try {
      if (user) {
        await update.mutateAsync({
          id: user.id,
          fullName: values.fullName,
          role: values.role,
          departmentId: values.departmentId,
          jobTitleId: values.jobTitleId,
        })
        toast.show(t.updated)
      } else {
        const res = await create.mutateAsync(values)
        toast.show(res.emailSent ? t.created : t.createdNoEmail)
      }
      onClose()
    } catch (err) {
      if (!(err instanceof ApiError)) {
        setServerError(messages.errors.unknown)
        return
      }
      if (err.code === 'EMAIL_TAKEN' && !user) {
        setError('email', { message: t.emailTaken }, { shouldFocus: true })
        return
      }
      const field = fieldFor(err.code)
      if (field) setError(field, { message: err.message })
      else setServerError(err.message)
    }
  })

  return (
    <Dialog
      open
      onClose={onClose}
      title={user ? t.editTitle : t.createTitle}
      body={
        <form
          id={formId}
          noValidate
          onSubmit={(e) => void onSubmit(e)}
          className="flex flex-col gap-3"
        >
          <FormAlert message={serverError} />
          {user ? (
            <Input label={t.email} value={user.email} disabled readOnly hint={t.emailReadonly} />
          ) : (
            <Input
              label={t.email}
              type="email"
              autoComplete="off"
              autoFocus
              error={errors.email?.message}
              {...register('email')}
            />
          )}
          <Input label={t.fullName} error={errors.fullName?.message} {...register('fullName')} />
          <Select
            label={t.role}
            options={roleOptions}
            disabled={isSelf}
            error={errors.role?.message}
            {...register('role')}
          />
          <Select
            label={t.department}
            options={deptOptions}
            error={errors.departmentId?.message}
            {...register('departmentId')}
          />
          <Select
            label={t.jobTitle}
            options={jobOptions}
            error={errors.jobTitleId?.message}
            {...register('jobTitleId')}
          />
        </form>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button type="submit" form={formId} loading={isSubmitting}>
        {user ? t.submitEdit : t.submitCreate}
      </Button>
    </Dialog>
  )
}
