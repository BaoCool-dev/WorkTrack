import { useId, type ReactNode } from 'react'
import { useSearchParams } from 'react-router'
import { cn } from '@/shared/lib/cn'
import { messages } from '@/shared/lib/messages'
import { CatalogPanel } from './CatalogPanel'
import { DepartmentsPanel } from './DepartmentsPanel'
import { UsersPanel } from './UsersPanel'

const t = messages.admin

const TABS = [
  { value: 'users', label: t.tabs.users },
  { value: 'departments', label: t.tabs.departments },
  { value: 'job-titles', label: t.tabs.jobTitles },
  { value: 'skills', label: t.tabs.skills },
] as const

/** Tab do feature khác cấp (vd. Nhãn, Cấu hình của tasks) — ghép ở trang để tránh phụ thuộc vòng. */
export interface ExtraAdminTab {
  value: string
  label: string
  render: () => ReactNode
}

export interface AdminOrgViewProps {
  extraTabs?: readonly ExtraAdminTab[]
}

/** Màn Nhân sự & phòng ban (artboard 10, CF-02). Tab lưu trong ?tab để quay lại đúng chỗ. */
export function AdminOrgView({ extraTabs = [] }: AdminOrgViewProps) {
  const [params, setParams] = useSearchParams()
  const tabs = [...TABS, ...extraTabs]
  const tab = tabs.find((x) => x.value === params.get('tab'))?.value ?? 'users'
  const extra = extraTabs.find((x) => x.value === tab)
  const baseId = useId()

  const select = (next: string) => {
    setParams(
      (p) => {
        const q = new URLSearchParams(p)
        if (next === 'users') q.delete('tab')
        else q.set('tab', next)
        return q
      },
      { replace: true },
    )
  }

  return (
    <div className="flex flex-col gap-4">
      <header className="flex items-baseline gap-3">
        <h1 className="text-lg font-semibold text-ink">{t.title}</h1>
        <p className="text-sm text-ink-muted">{t.meta}</p>
      </header>
      <div role="tablist" aria-label={t.tabsLabel} className="flex gap-1 border-b border-line">
        {tabs.map((x) => {
          const selected = x.value === tab
          return (
            <button
              key={x.value}
              type="button"
              role="tab"
              id={`${baseId}-${x.value}-tab`}
              aria-selected={selected}
              aria-controls={`${baseId}-panel`}
              onClick={() => {
                select(x.value)
              }}
              className={cn(
                '-mb-px border-b-2 px-3 py-2 text-base focus-visible:outline-2 focus-visible:outline-accent',
                selected
                  ? 'border-accent font-semibold text-accent'
                  : 'border-transparent text-ink-muted hover:text-ink',
              )}
            >
              {x.label}
            </button>
          )
        })}
      </div>
      <div role="tabpanel" id={`${baseId}-panel`} aria-labelledby={`${baseId}-${tab}-tab`}>
        {tab === 'users' && <UsersPanel />}
        {tab === 'departments' && <DepartmentsPanel />}
        {tab === 'job-titles' && <CatalogPanel kind="job-titles" />}
        {tab === 'skills' && <CatalogPanel key="skills" kind="skills" />}
        {extra?.render()}
      </div>
    </div>
  )
}
