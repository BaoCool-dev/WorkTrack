import type { Board, Project, ProjectMember } from './model/types'

export function makeProject(over: Partial<Project> = {}): Project {
  return {
    id: 'p-1',
    name: 'Cổng thông tin khách hàng',
    description: 'Xây cổng tự phục vụ cho khách hàng doanh nghiệp.',
    status: 'in_progress',
    startDate: '2026-12-01',
    endDate: '2027-03-31',
    manager: { id: 'u-manager', name: 'Trần Bình' },
    memberCount: 2,
    wipLimit: 5,
    createdAt: '2026-11-20T02:00:00Z',
    updatedAt: '2026-11-20T02:00:00Z',
    ...over,
  }
}

export const MEMBERS: ProjectMember[] = [
  {
    userId: 'u-manager',
    fullName: 'Trần Bình',
    email: 'binh@worktrack.local',
    role: 'manager',
    status: 'active',
  },
  {
    userId: 'u-2',
    fullName: 'Đỗ Thu Hà',
    email: 'ha@worktrack.local',
    role: 'member',
    status: 'active',
  },
]

export function makeBoard(): Board {
  const columns = [
    { id: 'c-1', name: 'Backlog', position: 0, statusGroup: 'todo' as const, taskCount: 0 },
    {
      id: 'c-2',
      name: 'In Progress',
      position: 1,
      statusGroup: 'in_progress' as const,
      taskCount: 3,
    },
    { id: 'c-3', name: 'Done', position: 2, statusGroup: 'done' as const, taskCount: 0 },
  ]
  return {
    columns,
    transitions: [
      { fromColumnId: 'c-1', toColumnId: 'c-2', role: 'manager' },
      { fromColumnId: 'c-2', toColumnId: 'c-3', role: 'manager' },
      { fromColumnId: 'c-1', toColumnId: 'c-2', role: 'member' },
    ],
  }
}
