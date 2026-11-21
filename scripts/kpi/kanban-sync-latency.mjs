#!/usr/bin/env node
// KPI 1 (NFR-05): độ trễ đồng bộ bảng Kanban giữa hai client.
//
// Cách đo: mở 2 kết nối WebSocket vào cùng một dự án — A (Manager, người kéo) và B (Member, người xem).
// Mỗi lượt A gửi POST /tasks/{id}/move, đồng hồ bắt đầu NGAY TRƯỚC khi gửi và dừng khi B nhận được
// sự kiện task.moved đúng thẻ, đúng version mới. Con số này gồm cả thời gian ghi CSDL của server —
// đúng thứ người dùng B cảm nhận. Mục tiêu: p95 ≤ 2000 ms trên ≥ 30 lượt.
//
// Không cần thư viện ngoài: Node ≥ 22 có sẵn fetch và WebSocket.
// Biến môi trường (mật khẩu KHÔNG ghi vào repo — lấy từ lần chạy `make seed`):
//   KPI_MANAGER_PASSWORD, KPI_MEMBER_PASSWORD  bắt buộc
//   KPI_API         mặc định http://localhost:8080
//   KPI_RUNS        mặc định 30
//   KPI_PROJECT     tên dự án, mặc định "Cổng thông tin nội bộ" (dự án mẫu của seed)
//   KPI_OUT         đường dẫn file kết quả JSON (tùy chọn)
// Ví dụ:  KPI_MANAGER_PASSWORD='…' KPI_MEMBER_PASSWORD='…' node scripts/kpi/kanban-sync-latency.mjs

import { writeFileSync } from 'node:fs'

const API = (process.env.KPI_API ?? 'http://localhost:8080').replace(/\/$/, '')
const RUNS = Number(process.env.KPI_RUNS ?? 30)
const PROJECT = process.env.KPI_PROJECT ?? 'Cổng thông tin nội bộ'
const TASK_TITLE = 'KPI1 — thẻ đo đồng bộ'
const EVENT_TIMEOUT_MS = 5000
const TARGET_P95_MS = 2000

function need(name) {
  const v = process.env[name]
  if (!v) throw new Error(`Thiếu biến môi trường ${name}`)
  return v
}

async function call(method, path, token, body) {
  const res = await fetch(`${API}/api/v1${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  const data = res.status === 204 ? null : await res.json()
  if (!res.ok) throw new Error(`${method} ${path} → ${res.status} ${JSON.stringify(data?.error ?? data)}`)
  return data
}

const login = async (role, password) =>
  (await call('POST', '/auth/login', null, { email: `${role}@worktrack.local`, password })).accessToken

/** Mở WebSocket của dự án; trả về socket và hàm chờ sự kiện thỏa điều kiện. */
function openSocket(projectId, token) {
  const wsBase = API.replace(/^http/, 'ws')
  const ws = new WebSocket(`${wsBase}/api/v1/projects/${projectId}/ws?access_token=${encodeURIComponent(token)}`)
  const waiters = new Set()
  ws.onmessage = (e) => {
    const at = performance.now()
    const ev = JSON.parse(e.data)
    for (const w of waiters) {
      if (w.match(ev)) {
        waiters.delete(w)
        w.resolve({ ev, at })
      }
    }
  }
  const ready = new Promise((resolve, reject) => {
    ws.onopen = resolve
    ws.onerror = () => reject(new Error('Không mở được WebSocket'))
  })
  const waitFor = (match) =>
    new Promise((resolve, reject) => {
      const w = { match, resolve }
      waiters.add(w)
      setTimeout(() => {
        if (waiters.delete(w)) reject(new Error(`Hết ${EVENT_TIMEOUT_MS} ms chưa nhận sự kiện`))
      }, EVENT_TIMEOUT_MS)
    })
  return { ws, ready, waitFor }
}

/**
 * Hai cột nhóm "Cần làm" mà Manager chuyển qua lại được — kéo giữa chúng không đụng WIP
 * và không vào nhóm Hoàn thành, nên đo bao nhiêu lượt cũng không làm sai dữ liệu nghiệp vụ.
 */
function pickColumns(board) {
  const todo = board.columns.filter((c) => c.statusGroup === 'todo')
  const allowed = (a, b) =>
    board.transitions.some((t) => t.role === 'manager' && t.fromColumnId === a.id && t.toColumnId === b.id)
  for (const a of todo)
    for (const b of todo) if (a.id !== b.id && allowed(a, b) && allowed(b, a)) return [a, b]
  throw new Error('Dự án không có hai cột nhóm Cần làm mà Manager chuyển qua lại được')
}

async function ensureTask(projectId, token, columnIds) {
  const page = await call('GET', `/projects/${projectId}/board/tasks`, token)
  const found = page.items.find((t) => t.title === TASK_TITLE && columnIds.includes(t.column.id))
  if (found) return found
  return call('POST', `/projects/${projectId}/tasks`, token, {
    title: TASK_TITLE,
    description: 'Thẻ do scripts/kpi/kanban-sync-latency.mjs tạo để đo KPI 1. Có thể xóa sau khi đo.',
  })
}

const percentile = (sorted, p) => sorted[Math.min(sorted.length - 1, Math.ceil((p / 100) * sorted.length) - 1)]
const round = (x) => Math.round(x * 10) / 10

async function main() {
  const managerToken = await login('manager', need('KPI_MANAGER_PASSWORD'))
  const memberToken = await login('member', need('KPI_MEMBER_PASSWORD'))

  const projects = await call('GET', '/projects?pageSize=100', managerToken)
  const project = projects.items.find((p) => p.name === PROJECT)
  if (!project) throw new Error(`Không thấy dự án "${PROJECT}" trong danh sách của Manager`)
  const board = await call('GET', `/projects/${project.id}/board`, managerToken)
  const [colA, colB] = pickColumns(board)
  let task = await ensureTask(project.id, managerToken, [colA.id, colB.id])

  const a = openSocket(project.id, managerToken)
  const b = openSocket(project.id, memberToken)
  await Promise.all([a.ready, b.ready])

  const samples = []
  for (let i = 0; i < RUNS; i++) {
    const to = task.column.id === colA.id ? colB : colA
    const nextVersion = task.version + 1
    const match = (ev) => ev.type === 'task.moved' && ev.taskId === task.id && ev.version === nextVersion
    const seenByB = b.waitFor(match)
    const seenByA = a.waitFor(match)

    const t0 = performance.now()
    const saved = await call('POST', `/tasks/${task.id}/move`, managerToken, {
      columnId: to.id,
      position: 0,
      version: task.version,
    })
    const tHttp = performance.now()
    const [{ at: tB, ev }, { at: tA }] = await Promise.all([seenByB, seenByA])

    samples.push({
      run: i + 1,
      to: to.name,
      endToEndMs: round(tB - t0), // A bấm thả → B thấy thẻ đổi cột
      httpMs: round(tHttp - t0), // A nhận phản hồi lưu
      clientSpreadMs: round(Math.abs(tB - tA)), // chênh giữa hai client nhận cùng sự kiện
      serverSentAt: ev.sentAt,
    })
    task = saved
    await new Promise((r) => setTimeout(r, 100)) // giống thao tác người, không dồn dập
  }
  a.ws.close()
  b.ws.close()

  const e2e = samples.map((s) => s.endToEndMs).sort((x, y) => x - y)
  const summary = {
    measuredAt: new Date().toISOString(),
    api: API,
    project: PROJECT,
    columns: [colA.name, colB.name],
    runs: samples.length,
    endToEndMs: {
      min: e2e[0],
      p50: percentile(e2e, 50),
      p95: percentile(e2e, 95),
      max: e2e.at(-1),
      mean: round(e2e.reduce((s, x) => s + x, 0) / e2e.length),
    },
    targetP95Ms: TARGET_P95_MS,
    pass: percentile(e2e, 95) <= TARGET_P95_MS,
  }

  console.table(samples.map(({ serverSentAt: _s, ...rest }) => rest))
  console.log(JSON.stringify(summary, null, 2))
  console.log(summary.pass ? `KPI 1 ĐẠT: p95 = ${summary.endToEndMs.p95} ms ≤ ${TARGET_P95_MS} ms` : 'KPI 1 CHƯA ĐẠT')
  if (process.env.KPI_OUT) writeFileSync(process.env.KPI_OUT, JSON.stringify({ summary, samples }, null, 2) + '\n')
  process.exit(summary.pass ? 0 : 1)
}

main().catch((err) => {
  console.error(`Lỗi: ${err.message}`)
  process.exit(2)
})
