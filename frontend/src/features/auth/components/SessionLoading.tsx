import { messages } from '@/shared/lib/messages'
import { Skeleton } from '@/shared/ui'

/** Đang khôi phục phiên: giữ chỗ bằng khung sidebar + nội dung để bố cục không nhảy. */
export function SessionLoading() {
  return (
    <div role="status" aria-label={messages.auth.sessionLoading} className="flex h-screen bg-page">
      <div className="flex w-59 flex-col gap-3 border-r border-line bg-surface px-2.5 py-3">
        <Skeleton className="h-7 w-32" />
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
        <Skeleton className="h-10" />
      </div>
      <div className="flex flex-1 flex-col gap-3 p-5">
        <Skeleton className="h-7 w-64" />
        <Skeleton className="h-32" />
      </div>
    </div>
  )
}
