import { Search, UserPlus, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import {
  Avatar,
  Badge,
  Button,
  cellClass,
  DataTable,
  Dialog,
  EmptyState,
  ErrorState,
  FormAlert,
  Skeleton,
  TableSkeleton,
  useToast,
  type BadgeTone,
} from '@/shared/ui'
import {
  useAddMember,
  useMemberCandidates,
  useProjectMembers,
  useRemoveMember,
} from '../api/queries'
import type { MemberStatus, Project, ProjectMember } from '../model/types'

const t = messages.projects.members
const SEARCH_DEBOUNCE_MS = 300

const statusTone: Record<MemberStatus, BadgeTone> = {
  pending: 'warn',
  active: 'ok',
  disabled: 'draft',
}

function Person({ m }: { m: ProjectMember }) {
  return (
    <div className="flex items-center gap-2">
      <Avatar name={m.fullName} />
      <div className="flex min-w-0 flex-col">
        <span className="truncate font-semibold text-ink">{m.fullName}</span>
        <span className="truncate text-sm text-ink-muted">{m.email}</span>
      </div>
    </div>
  )
}

/** Thành viên dự án: Manager phụ trách thêm/bớt; người khác chỉ xem. */
export function MembersPanel({ project, canManage }: { project: Project; canManage: boolean }) {
  const members = useProjectMembers(project.id)
  const remove = useRemoveMember(project.id)
  const [adding, setAdding] = useState(false)
  const [removing, setRemoving] = useState<ProjectMember | null>(null)

  let content
  if (members.isError && !members.data)
    content = (
      <ErrorState
        title={t.loadError}
        errorRef={members.error instanceof ApiError ? members.error.code : undefined}
        onRetry={() => void members.refetch()}
      />
    )
  else if (!members.data) content = <TableSkeleton rows={4} />
  else if (members.data.length === 0) content = <EmptyState icon={Users} title={t.emptyTitle} />
  else
    content = (
      <DataTable
        label={t.tableLabel}
        head={[
          { label: t.colName },
          { label: t.colRole },
          { label: t.colStatus },
          ...(canManage ? [{ label: t.colActions }] : []),
        ]}
      >
        {members.data.map((m) => {
          const isManager = m.userId === project.manager.id
          return (
            <tr key={m.userId}>
              <td className={cellClass}>
                <Person m={m} />
              </td>
              <td className={cellClass}>
                <span className="inline-flex items-center gap-2">
                  {messages.roles[m.role]}
                  {isManager && <Badge tone="accent">{t.managerTag}</Badge>}
                </span>
              </td>
              <td className={cellClass}>
                <Badge tone={statusTone[m.status]}>{t.accountStatus[m.status]}</Badge>
              </td>
              {canManage && (
                <td className={cellClass}>
                  {!isManager && (
                    <Button
                      variant="ghost"
                      onClick={() => {
                        setRemoving(m)
                      }}
                    >
                      {t.remove}
                    </Button>
                  )}
                </td>
              )}
            </tr>
          )
        })}
      </DataTable>
    )

  return (
    <div className="flex flex-col gap-3">
      {canManage && (
        <div className="flex justify-end">
          <Button
            icon={<UserPlus size={16} strokeWidth={2} aria-hidden />}
            onClick={() => {
              setAdding(true)
            }}
          >
            {t.add}
          </Button>
        </div>
      )}
      {content}
      {adding && (
        <AddMemberDialog
          projectId={project.id}
          onClose={() => {
            setAdding(false)
          }}
        />
      )}
      {removing && (
        <ConfirmDialog
          title={t.removeTitle(removing.fullName)}
          description={t.removeHint}
          confirmLabel={t.remove}
          doneMessage={t.removed}
          onConfirm={() => remove.mutateAsync(removing.userId)}
          onClose={() => {
            setRemoving(null)
          }}
        />
      )}
    </div>
  )
}

function AddMemberDialog({ projectId, onClose }: { projectId: string; onClose: () => void }) {
  const toast = useToast()
  const [input, setInput] = useState('')
  const [search, setSearch] = useState('')
  const [error, setError] = useState<string | null>(null)
  const candidates = useMemberCandidates(projectId, search, true)
  const add = useAddMember(projectId)

  useEffect(() => {
    const id = setTimeout(() => {
      setSearch(input)
    }, SEARCH_DEBOUNCE_MS)
    return () => {
      clearTimeout(id)
    }
  }, [input])

  const choose = async (m: ProjectMember) => {
    setError(null)
    try {
      await add.mutateAsync(m.userId)
      toast.show(t.added)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  let list
  if (candidates.isError && !candidates.data)
    list = (
      <FormAlert
        message={
          candidates.error instanceof ApiError ? candidates.error.message : messages.errors.unknown
        }
      />
    )
  else if (!candidates.data)
    list = (
      <div className="flex flex-col gap-2" aria-busy="true">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    )
  else if (candidates.data.length === 0)
    list = <p className="py-4 text-center text-sm text-ink-muted">{t.candidateEmpty}</p>
  else
    list = (
      <ul aria-label={t.addTitle} className="flex max-h-72 flex-col overflow-y-auto">
        {candidates.data.map((m) => (
          <li
            key={m.userId}
            className="flex items-center justify-between gap-2 border-t border-line py-2 first:border-t-0"
          >
            <Person m={m} />
            <span className="flex items-center gap-2">
              <span className="text-sm text-ink-muted">{messages.roles[m.role]}</span>
              <Button
                variant="secondary"
                aria-label={`${t.choose} ${m.fullName}`}
                disabled={add.isPending}
                onClick={() => void choose(m)}
              >
                {t.choose}
              </Button>
            </span>
          </li>
        ))}
      </ul>
    )

  return (
    <Dialog
      open
      onClose={onClose}
      size="lg"
      title={t.addTitle}
      description={t.addHint}
      body={
        <div className="flex flex-col gap-3">
          <FormAlert message={error} />
          <div className="relative">
            <Search
              size={16}
              strokeWidth={2}
              aria-hidden
              className="pointer-events-none absolute top-3 left-3 text-ink-faint"
            />
            <input
              type="search"
              autoFocus
              aria-label={t.candidateSearchLabel}
              placeholder={t.candidateSearch}
              value={input}
              onChange={(e) => {
                setInput(e.target.value)
              }}
              className="h-10 w-full rounded-md border border-line-strong bg-surface pr-3 pl-9 text-base text-ink placeholder:text-ink-faint focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none"
            />
          </div>
          {list}
        </div>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.close}
      </Button>
    </Dialog>
  )
}

/** Xác nhận thao tác xóa; lỗi server hiện ngay trong hộp thoại. */
export function ConfirmDialog({
  title,
  description,
  confirmLabel,
  doneMessage,
  onConfirm,
  onClose,
}: {
  title: string
  description: string
  confirmLabel: string
  doneMessage: string
  onConfirm: () => Promise<unknown>
  onClose: () => void
}) {
  const toast = useToast()
  const [error, setError] = useState<string | null>(null)
  const [pending, setPending] = useState(false)
  const confirm = async () => {
    setPending(true)
    setError(null)
    try {
      await onConfirm()
      toast.show(doneMessage)
      onClose()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : messages.errors.unknown)
      setPending(false)
    }
  }
  return (
    <Dialog
      open
      onClose={onClose}
      title={title}
      description={description}
      body={error ? <FormAlert message={error} /> : undefined}
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button variant="danger" loading={pending} onClick={() => void confirm()}>
        {confirmLabel}
      </Button>
    </Dialog>
  )
}
