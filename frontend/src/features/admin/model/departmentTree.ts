import type { Department } from './types'

export interface DepartmentRow extends Department {
  depth: number
}

/**
 * Trải cây phòng ban thành danh sách theo thứ tự duyệt sâu (cha trước con, anh em theo tên).
 * Phòng ban có parentId trỏ tới mục không tồn tại được coi là gốc để không bị mất.
 */
export function flattenDepartments(items: readonly Department[]): DepartmentRow[] {
  const ids = new Set(items.map((d) => d.id))
  const children = new Map<string | null, Department[]>()
  for (const d of items) {
    const key = d.parentId !== null && ids.has(d.parentId) ? d.parentId : null
    const list = children.get(key) ?? []
    list.push(d)
    children.set(key, list)
  }
  const out: DepartmentRow[] = []
  const seen = new Set<string>()
  const walk = (parent: string | null, depth: number) => {
    const list = [...(children.get(parent) ?? [])].sort((a, b) =>
      a.name.localeCompare(b.name, 'vi'),
    )
    for (const d of list) {
      if (seen.has(d.id)) continue
      seen.add(d.id)
      out.push({ ...d, depth })
      walk(d.id, depth + 1)
    }
  }
  walk(null, 0)
  return out
}

/** Mã phòng ban và mọi phòng ban con cháu — không được chọn làm cha khi sửa (tránh vòng). */
export function descendantsOf(items: readonly Department[], id: string): Set<string> {
  const out = new Set<string>([id])
  let grew = true
  while (grew) {
    grew = false
    for (const d of items) {
      if (d.parentId !== null && out.has(d.parentId) && !out.has(d.id)) {
        out.add(d.id)
        grew = true
      }
    }
  }
  return out
}

/** Nhãn ô chọn có thụt lề theo cấp (dùng chung cho bộ lọc và form). */
export function departmentOptions(
  rows: readonly DepartmentRow[],
): { value: string; label: string }[] {
  return rows.map((d) => ({ value: d.id, label: `${'  '.repeat(d.depth)}${d.name}` }))
}
