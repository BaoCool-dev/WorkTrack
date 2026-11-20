import type { BadgeTone } from '@/shared/ui'
import type { ProjectStatus, StatusGroup } from './types'

/** Màu badge trạng thái dự án theo bộ trạng thái của design-system §2. */
export const projectStatusTone: Record<ProjectStatus, BadgeTone> = {
  new: 'draft',
  in_progress: 'active',
  completed: 'ok',
  paused: 'warn',
  cancelled: 'bad',
}

export const statusGroupTone: Record<StatusGroup, BadgeTone> = {
  todo: 'draft',
  in_progress: 'active',
  done: 'ok',
}
