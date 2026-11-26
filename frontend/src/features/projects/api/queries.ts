import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { Board, ProjectInput, Transition } from '../model/types'
import { projectsApi, type ColumnInput, type ProjectListParams } from './projectsApi'

export const projectKeys = {
  all: ['projects'] as const,
  lists: () => [...projectKeys.all, 'list'] as const,
  list: (p: ProjectListParams) => [...projectKeys.lists(), p] as const,
  detail: (id: string) => [...projectKeys.all, 'detail', id] as const,
  members: (id: string) => [...projectKeys.detail(id), 'members'] as const,
  candidates: (id: string, search: string) =>
    [...projectKeys.detail(id), 'candidates', search] as const,
  board: (id: string) => [...projectKeys.detail(id), 'board'] as const,
}

export function useProjects(params: ProjectListParams) {
  return useQuery({
    queryKey: projectKeys.list(params),
    queryFn: () => projectsApi.list(params),
    placeholderData: keepPreviousData,
  })
}

export function useProject(id: string) {
  return useQuery({ queryKey: projectKeys.detail(id), queryFn: () => projectsApi.get(id) })
}

export function useProjectMembers(id: string) {
  return useQuery({
    queryKey: projectKeys.members(id),
    queryFn: async () => (await projectsApi.members(id)).items,
  })
}

export function useMemberCandidates(id: string, search: string, enabled: boolean) {
  return useQuery({
    queryKey: projectKeys.candidates(id, search),
    queryFn: async () => (await projectsApi.candidates(id, search)).items,
    enabled,
    placeholderData: keepPreviousData,
  })
}

export function useBoard(id: string) {
  return useQuery({ queryKey: projectKeys.board(id), queryFn: () => projectsApi.board(id) })
}

export function useCreateProject() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: projectsApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: projectKeys.lists() }),
  })
}

export function useUpdateProject(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: ProjectInput) => projectsApi.update(id, body),
    onSuccess: (p) => {
      qc.setQueryData(projectKeys.detail(id), p)
      return qc.invalidateQueries({ queryKey: projectKeys.lists() })
    },
  })
}

/** Manager phụ trách đổi giới hạn WIP (CF-05). */
export function useSetWipLimit(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (wipLimit: number) => projectsApi.setWipLimit(id, wipLimit),
    onSuccess: (p) => {
      qc.setQueryData(projectKeys.detail(id), p)
    },
  })
}

/** Thêm/bớt thành viên đổi số thành viên ở danh sách và chi tiết → làm mới cả nhóm dự án. */
export function useAddMember(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => projectsApi.addMember(id, userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: projectKeys.all }),
  })
}

export function useRemoveMember(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => projectsApi.removeMember(id, userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: projectKeys.all }),
  })
}

/** Mọi thao tác cột/luật trả về bảng mới nhất → ghi thẳng vào cache, không cần gọi lại. */
function useBoardMutation<V>(id: string, fn: (v: V) => Promise<Board>) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSuccess: (b) => {
      qc.setQueryData(projectKeys.board(id), b)
    },
  })
}

export function useAddColumn(id: string) {
  return useBoardMutation(id, (c: ColumnInput) => projectsApi.addColumn(id, c))
}

export function useUpdateColumn(id: string) {
  return useBoardMutation(id, ({ columnId, ...c }: ColumnInput & { columnId: string }) =>
    projectsApi.updateColumn(id, columnId, c),
  )
}

export function useDeleteColumn(id: string) {
  return useBoardMutation(id, (columnId: string) => projectsApi.deleteColumn(id, columnId))
}

export function useReorderColumns(id: string) {
  return useBoardMutation(id, (ids: string[]) => projectsApi.reorderColumns(id, ids))
}

export function useReplaceTransitions(id: string) {
  return useBoardMutation(id, (ts: Transition[]) => projectsApi.replaceTransitions(id, ts))
}
