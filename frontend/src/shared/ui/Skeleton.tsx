import { cn } from '@/shared/lib/cn'

/** Khối giữ chỗ khi tải — truyền kích thước thật qua className để bố cục không nhảy. */
export function Skeleton({ className }: { className?: string }) {
  return <div aria-hidden className={cn('animate-pulse rounded-md bg-line', className)} />
}
