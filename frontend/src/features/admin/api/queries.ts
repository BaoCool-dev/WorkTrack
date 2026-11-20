import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { SkillLevel, UpdateUserInput } from '../model/types'
import { adminApi, type CatalogKind, type UserListParams } from './adminApi'

export const adminKeys = {
  all: ['admin'] as const,
  users: () => [...adminKeys.all, 'users'] as const,
  userList: (p: UserListParams) => [...adminKeys.users(), 'list', p] as const,
  departments: () => [...adminKeys.all, 'departments'] as const,
  catalog: (kind: CatalogKind) => [...adminKeys.all, kind] as const,
}

export function useUsers(params: UserListParams) {
  return useQuery({
    queryKey: adminKeys.userList(params),
    queryFn: () => adminApi.listUsers(params),
    // Giữ trang cũ khi đổi trang/bộ lọc để bảng không nháy về skeleton.
    placeholderData: keepPreviousData,
  })
}

export function useDepartments() {
  return useQuery({
    queryKey: adminKeys.departments(),
    queryFn: async () => (await adminApi.listDepartments()).items,
  })
}

export function useCatalog(kind: CatalogKind) {
  return useQuery({
    queryKey: adminKeys.catalog(kind),
    queryFn: async () => (await adminApi.listCatalog(kind)).items,
  })
}

/** Sau khi đổi nhân sự, số nhân sự ở phòng ban/chức danh/kỹ năng cũng đổi → làm mới cả nhóm admin. */
function useInvalidateAdmin() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: adminKeys.all })
}

export function useCreateUser() {
  const invalidate = useInvalidateAdmin()
  return useMutation({ mutationFn: adminApi.createUser, onSuccess: invalidate })
}

export function useUpdateUser() {
  const invalidate = useInvalidateAdmin()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateUserInput & { id: string }) =>
      adminApi.updateUser(id, body),
    onSuccess: invalidate,
  })
}

export function useReplaceSkills() {
  const invalidate = useInvalidateAdmin()
  return useMutation({
    mutationFn: ({
      id,
      skills,
    }: {
      id: string
      skills: { skillId: string; level: SkillLevel }[]
    }) => adminApi.replaceSkills(id, skills),
    onSuccess: invalidate,
  })
}

export function useResendActivation() {
  return useMutation({ mutationFn: adminApi.resendActivation })
}

export interface DepartmentInput {
  id?: string
  name: string
  parentId: string | null
}

export function useSaveDepartment() {
  const invalidate = useInvalidateAdmin()
  return useMutation({
    mutationFn: ({ id, ...body }: DepartmentInput) =>
      id ? adminApi.updateDepartment(id, body) : adminApi.createDepartment(body),
    onSuccess: invalidate,
  })
}

export function useDeleteDepartment() {
  const invalidate = useInvalidateAdmin()
  return useMutation({ mutationFn: adminApi.deleteDepartment, onSuccess: invalidate })
}

export function useSaveCatalog(kind: CatalogKind) {
  const invalidate = useInvalidateAdmin()
  return useMutation({
    mutationFn: ({ id, name }: { id?: string; name: string }) =>
      id ? adminApi.updateCatalog(kind, id, name) : adminApi.createCatalog(kind, name),
    onSuccess: invalidate,
  })
}

export function useDeleteCatalog(kind: CatalogKind) {
  const invalidate = useInvalidateAdmin()
  return useMutation({
    mutationFn: (id: string) => adminApi.deleteCatalog(kind, id),
    onSuccess: invalidate,
  })
}
