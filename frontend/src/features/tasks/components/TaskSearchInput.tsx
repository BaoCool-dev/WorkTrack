import { Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { messages } from '@/shared/lib/messages'

const SEARCH_DEBOUNCE_MS = 300

/** Ô tìm theo tên công việc; gọi onSearch sau 300ms ngừng gõ. `resetKey` đổi → xóa ô. */
export function TaskSearchInput({
  onSearch,
  resetKey,
}: {
  onSearch: (value: string) => void
  resetKey: number
}) {
  const [value, setValue] = useState('')
  const [lastReset, setLastReset] = useState(resetKey)
  if (lastReset !== resetKey) {
    setLastReset(resetKey)
    setValue('')
  }

  useEffect(() => {
    const id = setTimeout(() => {
      onSearch(value.trim())
    }, SEARCH_DEBOUNCE_MS)
    return () => {
      clearTimeout(id)
    }
  }, [value, onSearch])

  return (
    <div className="relative w-72">
      <Search
        size={16}
        strokeWidth={2}
        aria-hidden
        className="pointer-events-none absolute top-3 left-3 text-ink-faint"
      />
      <input
        type="search"
        aria-label={messages.tasks.list.searchLabel}
        placeholder={messages.tasks.list.search}
        value={value}
        onChange={(e) => {
          setValue(e.target.value)
        }}
        className="h-10 w-full rounded-md border border-line-strong bg-surface pr-3 pl-9 text-base text-ink placeholder:text-ink-faint focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none"
      />
    </div>
  )
}
