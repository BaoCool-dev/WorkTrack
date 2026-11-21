import { useParams } from 'react-router'
import { KanbanBoard } from '@/features/kanban'
import { ProjectDetail } from '@/features/projects'
import { ProjectTasksPanel } from '@/features/tasks'

export function ProjectDetailPage() {
  const { projectId = '' } = useParams()
  return (
    <div className="px-5 py-3">
      {/* key: đổi dự án thì dựng lại toàn bộ (reset tab, nháp luật chuyển). */}
      <ProjectDetail
        key={projectId}
        projectId={projectId}
        renderBoard={(p, canManage) => <KanbanBoard project={p} canManage={canManage} />}
        renderTasks={(p, canManage) => <ProjectTasksPanel project={p} canManage={canManage} />}
      />
    </div>
  )
}
