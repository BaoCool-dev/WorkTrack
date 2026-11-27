import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
} from '@tanstack/react-query'
import { projectKeys } from '@/features/projects'
import type { Task, TaskInput } from '../model/types'
import { tasksApi, type MyTaskListParams, type TaskListParams } from './tasksApi'

export const taskKeys = {
  all: ['tasks'] as const,
  project: (projectId: string) => [...taskKeys.all, 'project', projectId] as const,
  list: (projectId: string, p: TaskListParams) => [...taskKeys.project(projectId), p] as const,
  /** Thẻ trên bảng Kanban — nằm dưới project(...) để mọi lần ghi công việc cũng làm mới bảng. */
  board: (projectId: string) => [...taskKeys.project(projectId), 'board'] as const,
  detail: (id: string) => [...taskKeys.all, 'detail', id] as const,
  mine: () => [...taskKeys.all, 'mine'] as const,
  mineList: (p: MyTaskListParams) => [...taskKeys.mine(), 'list', p] as const,
  wip: () => [...taskKeys.mine(), 'wip'] as const,
  labels: ['labels'] as const,
  settings: ['settings'] as const,
}

/** Sau khi ghi một công việc: cập nhật chi tiết, làm mới danh sách liên quan và số việc trên cột. */
export function afterTaskWrite(qc: QueryClient, task: Task) {
  qc.setQueryData(taskKeys.detail(task.id), task)
  void qc.invalidateQueries({ queryKey: taskKeys.project(task.projectId) })
  void qc.invalidateQueries({ queryKey: taskKeys.mine() })
  void qc.invalidateQueries({ queryKey: projectKeys.board(task.projectId) })
  void qc.invalidateQueries({ queryKey: taskKeys.labels })
}

export function useProjectTasks(projectId: string, params: TaskListParams) {
  return useQuery({
    queryKey: taskKeys.list(projectId, params),
    queryFn: () => tasksApi.list(projectId, params),
    placeholderData: keepPreviousData,
  })
}

export function useTask(id: string) {
  return useQuery({ queryKey: taskKeys.detail(id), queryFn: () => tasksApi.get(id) })
}

export function useCreateTask(projectId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: TaskInput) => tasksApi.create(projectId, body),
    onSuccess: (t) => {
      afterTaskWrite(qc, t)
    },
  })
}

export function useUpdateTask(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: TaskInput) => tasksApi.update(id, body),
    onSuccess: (t) => {
      afterTaskWrite(qc, t)
    },
  })
}

export function useClaimTask() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => tasksApi.claim(id),
    onSuccess: (t) => {
      afterTaskWrite(qc, t)
    },
  })
}

export function useMyTasks(params: MyTaskListParams) {
  return useQuery({
    queryKey: taskKeys.mineList(params),
    queryFn: () => tasksApi.mine(params),
    placeholderData: keepPreviousData,
  })
}

export function useMyWip() {
  return useQuery({
    queryKey: taskKeys.wip(),
    queryFn: async () => (await tasksApi.myWip()).count,
  })
}

export function useLabels() {
  return useQuery({
    queryKey: taskKeys.labels,
    queryFn: async () => (await tasksApi.labels()).items,
  })
}

export function useSaveLabel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id?: string; name: string }) =>
      id ? tasksApi.renameLabel(id, name) : tasksApi.createLabel(name),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: taskKeys.labels })
      // Tên nhãn hiển thị trong công việc.
      void qc.invalidateQueries({ queryKey: taskKeys.all })
    },
  })
}

export function useDeleteLabel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => tasksApi.deleteLabel(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: taskKeys.labels }),
  })
}

export function useSettings() {
  return useQuery({ queryKey: taskKeys.settings, queryFn: tasksApi.settings })
}

export function useUpdateSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: tasksApi.updateSettings,
    onSuccess: (s) => {
      qc.setQueryData(taskKeys.settings, s)
    },
  })
}
