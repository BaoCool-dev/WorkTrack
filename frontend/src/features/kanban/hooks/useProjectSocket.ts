import { useQueryClient, type QueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'

import { projectKeys, type Board } from '@/features/projects'
import { taskKeys, type Task } from '@/features/tasks'
import { wsUrl } from '@/shared/api/http'

import { kanbanApi } from '../api/kanbanApi'
import { moveTask } from '../model/boardState'
import type { BoardEvent, SocketStatus } from '../model/types'

/** Thử lại sau 1s, 2s, 4s… tối đa 15s — không dồn dập server khi nó đang khởi động lại. */
export function backoffDelay(attempt: number): number {
  return Math.min(1000 * 2 ** attempt, 15_000)
}

/**
 * Áp một sự kiện vào cache TanStack Query.
 * - task.moved đủ thông tin và mới hơn bản đang có → dời thẻ tại chỗ (nhanh, không gọi API — KPI 1).
 * - Còn lại (tạo/sửa thẻ, hoặc thiếu dữ liệu) → đánh dấu cũ để tải lại bảng.
 * Vì sao so version: sự kiện của chính mình đến sau khi cache đã có bản server, bỏ qua để không nhảy thẻ.
 */
export function applyBoardEvent(qc: QueryClient, projectId: string, e: BoardEvent): void {
  const key = taskKeys.board(projectId)
  if (
    e.type === 'task.moved' &&
    e.taskId &&
    e.toColumnId &&
    e.position !== undefined &&
    e.version
  ) {
    const tasks = qc.getQueryData<Task[]>(key)
    const column = qc
      .getQueryData<Board>(projectKeys.board(projectId))
      ?.columns.find((c) => c.id === e.toColumnId)
    const task = tasks?.find((t) => t.id === e.taskId)
    if (tasks && column && task) {
      if (task.version >= e.version) return
      const to = { id: column.id, name: column.name, statusGroup: column.statusGroup }
      const moved = moveTask(tasks, task.id, to, e.position).map((t) =>
        t.id === task.id ? { ...t, version: e.version ?? t.version } : t,
      )
      qc.setQueryData(key, moved)
      return
    }
  }
  void qc.invalidateQueries({ queryKey: key })
  void qc.invalidateQueries({ queryKey: projectKeys.board(projectId) })
}

interface SocketState {
  status: SocketStatus
  /** id người đang mở bảng này (gồm cả mình). */
  presence: string[]
}

/**
 * Kênh realtime của bảng một dự án. Tự kết nối lại khi rớt; mỗi lần nối lại làm mới token
 * (token có thể đã hết hạn trong lúc mất mạng) và tải lại bảng vì có thể đã lỡ sự kiện.
 */
export function useProjectSocket(projectId: string): SocketState {
  const qc = useQueryClient()
  const [state, setState] = useState<SocketState>({ status: 'connecting', presence: [] })

  useEffect(() => {
    let disposed = false
    let socket: WebSocket | null = null
    let timer: ReturnType<typeof setTimeout> | undefined
    let attempt = 0

    const scheduleRetry = () => {
      if (disposed) return
      setState((s) => ({ ...s, status: 'lost' }))
      timer = setTimeout(() => void connect(), backoffDelay(attempt))
      attempt += 1
    }

    const connect = async () => {
      const url = await wsUrl(kanbanApi.socketPath(projectId), { refresh: attempt > 0 })
      if (disposed) return
      if (!url) {
        scheduleRetry()
        return
      }
      const ws = new WebSocket(url)
      socket = ws
      ws.onopen = () => {
        if (attempt > 0) void qc.invalidateQueries({ queryKey: taskKeys.board(projectId) })
        attempt = 0
        setState((s) => ({ ...s, status: 'live' }))
      }
      ws.onmessage = (msg: MessageEvent<string>) => {
        const e = parseEvent(msg.data)
        if (!e) return
        if (e.type === 'presence') setState((s) => ({ ...s, presence: e.userIds ?? [] }))
        else applyBoardEvent(qc, projectId, e)
      }
      ws.onclose = () => {
        if (socket === ws) socket = null
        scheduleRetry()
      }
    }

    void connect()
    return () => {
      disposed = true
      clearTimeout(timer)
      socket?.close(1000)
    }
  }, [projectId, qc])

  return state
}

function parseEvent(raw: string): BoardEvent | null {
  try {
    return JSON.parse(raw) as BoardEvent
  } catch {
    return null
  }
}
