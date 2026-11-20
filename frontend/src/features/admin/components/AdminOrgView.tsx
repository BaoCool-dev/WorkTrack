import { useId } from 'react'
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
type Tab = (typeof TABS)[number]['value']

function parseTab(v: string | null): Tab {
  return TABS.find((x) => x.value === v)?.value ?? 'users'
}

/** Màn Nhân sự & phòng ban (artboard 10, CF-02). Tab lưu trong ?tab để quay lại đúng chỗ. */
export function AdminOrgView() {
  const [params, setParams] = useSearchParams()
  const tab = parseTab(params.get('tab'))
  const baseId = useId()

  const select = (next: Tab) => {
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
      <header className="flex flex-col gap-1">
        <h1 className="text-lg font-semibold text-ink">{t.title}</h1>
        <p className="text-sm text-ink-muted">{t.meta}</p>
      </header>
      <div role="tablist" aria-label={t.tabsLabel} className="flex gap-1 border-b border-line">
        {TABS.map((x) => {
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
                '-mb-px border-b-2 px-3 py-2 text-sm font-semibold focus-visible:outline-2 focus-visible:outline-accent',
                selected
                  ? 'border-accent text-ink'
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
      </div>
    </div>
  )
}
