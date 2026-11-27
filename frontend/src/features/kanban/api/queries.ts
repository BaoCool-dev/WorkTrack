import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { projectKeys } from '@/features/projects'
import { taskKeys, type Task } from '@/features/tasks'

import { moveTask } from '../model/boardState'
import type { MoveVars } from '../model/types'
import { kanbanApi } from './kanbanApi'

export function useBoardTasks(projectId: string) {
  return useQuery({
    queryKey: taskKeys.board(projectId),
    queryFn: () => kanbanApi.boardTasks(projectId),
  })
}

/** Cột đích kèm tên và nhóm — để thẻ đổi màu/nhóm ngay khi thả, trước khi server trả lời. */
export interface MoveArgs extends MoveVars {
  column: Task['column']
}

/**
 * Kéo thả với cập nhật lạc quan:
 * 1. onMutate: đổi cache ngay → thẻ nằm ở cột mới không chờ mạng.
 * 2. onError: trả cache về bản chụp trước khi thả → thẻ về cột cũ (component hiện dải lỗi).
 * 3. onSuccess: thay thẻ bằng bản server (version mới, cần cho lần kéo tiếp theo).
 */
export function useMoveTask(projectId: string) {
  const qc = useQueryClient()
  const key = taskKeys.board(projectId)

  return useMutation({
    // column chỉ dùng cho bản lạc quan; API chỉ cần id cột.
    mutationFn: ({ taskId, position, version, column }: MoveArgs) =>
      kanbanApi.move({ taskId, columnId: column.id, position, version }),
    onMutate: async (args) => {
      // Hủy lần tải đang chạy để nó không ghi đè bản lạc quan bằng dữ liệu cũ.
      await qc.cancelQueries({ queryKey: key })
      const prev = qc.getQueryData<Task[]>(key)
      if (prev) qc.setQueryData(key, moveTask(prev, args.taskId, args.column, args.position))
      return { prev }
    },
    onError: (_err, _args, ctx) => {
      if (ctx?.prev) qc.setQueryData(key, ctx.prev)
      // Lỗi xung đột/luật: dữ liệu trên máy có thể đã cũ → tải lại bản mới nhất.
      void qc.invalidateQueries({ queryKey: key })
    },
    onSuccess: (saved) => {
      qc.setQueryData<Task[]>(key, (cur) => cur?.map((t) => (t.id === saved.id ? saved : t)))
      qc.setQueryData(taskKeys.detail(saved.id), saved)
    },
    onSettled: () => {
      // Số thẻ trên tiêu đề cột và WIP "của tôi" đổi theo.
      void qc.invalidateQueries({ queryKey: projectKeys.board(projectId) })
      void qc.invalidateQueries({ queryKey: taskKeys.mine() })
    },
  })
}
