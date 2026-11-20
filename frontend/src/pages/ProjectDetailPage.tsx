import { useParams } from 'react-router'
import { ProjectDetail } from '@/features/projects'

export function ProjectDetailPage() {
  const { projectId = '' } = useParams()
  return (
    <div className="px-5 py-3">
      {/* key: đổi dự án thì dựng lại toàn bộ (reset tab, nháp luật chuyển). */}
      <ProjectDetail key={projectId} projectId={projectId} />
    </div>
  )
}
