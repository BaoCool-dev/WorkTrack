import { Plus, X } from 'lucide-react'
import { useState } from 'react'
import { ApiError } from '@/shared/api/http'
import { messages } from '@/shared/lib/messages'
import { Button, Dialog, FormAlert, Select, Skeleton, useToast } from '@/shared/ui'
import { useCatalog, useReplaceSkills } from '../api/queries'
import {
  MAX_SKILLS,
  SKILL_LEVELS,
  type AdminUser,
  type SkillLevel,
  type UserSkill,
} from '../model/types'

const t = messages.admin.skillForm
const levelOptions = SKILL_LEVELS.map((l) => ({ value: l, label: messages.admin.levels[l] }))

function isLevel(v: string): v is SkillLevel {
  return (SKILL_LEVELS as readonly string[]).includes(v)
}

/** Hồ sơ kỹ năng một nhân sự: chọn kỹ năng + mức Beginner/Intermediate/Senior (CF-02). */
export function SkillsDialog({ user, onClose }: { user: AdminUser; onClose: () => void }) {
  const toast = useToast()
  const catalog = useCatalog('skills')
  const replace = useReplaceSkills()
  const [rows, setRows] = useState<UserSkill[]>(user.skills)
  const [picked, setPicked] = useState('')
  const [serverError, setServerError] = useState<string | null>(null)

  const chosen = new Set(rows.map((r) => r.skillId))
  const available = (catalog.data ?? []).filter((s) => !chosen.has(s.id))
  const full = rows.length >= MAX_SKILLS

  const add = () => {
    const skill = catalog.data?.find((s) => s.id === picked)
    if (!skill || full) return
    setRows((r) => [...r, { skillId: skill.id, name: skill.name, level: 'beginner' }])
    setPicked('')
  }

  const save = async () => {
    setServerError(null)
    try {
      await replace.mutateAsync({
        id: user.id,
        skills: rows.map(({ skillId, level }) => ({ skillId, level })),
      })
      toast.show(t.saved)
      onClose()
    } catch (err) {
      setServerError(err instanceof ApiError ? err.message : messages.errors.unknown)
    }
  }

  let picker
  if (catalog.isPending) picker = <Skeleton className="h-9 w-full" />
  else if (catalog.isError) picker = <FormAlert message={messages.admin.org.loadError} />
  else if (catalog.data.length === 0)
    picker = <p className="text-sm text-ink-muted">{t.noCatalog}</p>
  else
    picker = (
      <div className="flex items-end gap-2">
        <div className="flex-1">
          <Select
            label={t.pick}
            value={picked}
            onChange={(e) => {
              setPicked(e.target.value)
            }}
            options={[
              { value: '', label: messages.admin.users.none },
              ...available.map((s) => ({ value: s.id, label: s.name })),
            ]}
            hint={full ? t.tooMany : undefined}
          />
        </div>
        <Button
          variant="secondary"
          icon={<Plus size={16} strokeWidth={2} aria-hidden />}
          disabled={!picked || full}
          onClick={add}
        >
          {t.add}
        </Button>
      </div>
    )

  return (
    <Dialog
      open
      onClose={onClose}
      size="lg"
      title={t.title(user.fullName)}
      description={t.description}
      body={
        <div className="flex flex-col gap-3">
          <FormAlert message={serverError} />
          {picker}
          {rows.length === 0 ? (
            <p className="text-sm text-ink-muted">{t.empty}</p>
          ) : (
            <ul className="flex flex-col divide-y divide-line rounded-md border border-line">
              {rows.map((r) => (
                <li key={r.skillId} className="flex items-center gap-3 px-3 py-2">
                  <span className="flex-1 text-sm text-ink">{r.name}</span>
                  <div className="w-44">
                    <Select
                      label={`${t.level} ${r.name}`}
                      hideLabel
                      value={r.level}
                      options={levelOptions}
                      onChange={(e) => {
                        const level = e.target.value
                        if (isLevel(level))
                          setRows((all) =>
                            all.map((x) => (x.skillId === r.skillId ? { ...x, level } : x)),
                          )
                      }}
                    />
                  </div>
                  <Button
                    variant="ghost"
                    aria-label={t.remove(r.name)}
                    icon={<X size={16} strokeWidth={2} aria-hidden />}
                    onClick={() => {
                      setRows((all) => all.filter((x) => x.skillId !== r.skillId))
                    }}
                  />
                </li>
              ))}
            </ul>
          )}
        </div>
      }
    >
      <Button variant="secondary" onClick={onClose}>
        {messages.common.cancel}
      </Button>
      <Button loading={replace.isPending} onClick={() => void save()}>
        {t.save}
      </Button>
    </Dialog>
  )
}
