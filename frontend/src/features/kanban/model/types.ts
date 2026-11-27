/** Sự kiện realtime của bảng — khớp realtime.wireEvent ở backend. */
export type BoardEventType = 'task.created' | 'task.updated' | 'task.moved' | 'presence'

export interface BoardEvent {
  type: BoardEventType
  projectId: string
  taskId?: string
  actorId?: string
  fromColumnId?: string
  toColumnId?: string
  position?: number
  version?: number
  /** Chỉ có ở `presence`: những người đang mở bảng. */
  userIds?: string[]
  sentAt: string
}

/** connecting: lần đầu; live: đang nhận sự kiện; lost: rớt mạng, đang thử lại. */
export type SocketStatus = 'connecting' | 'live' | 'lost'

export interface MoveVars {
  taskId: string
  columnId: string
  position: number
  version: number
}
