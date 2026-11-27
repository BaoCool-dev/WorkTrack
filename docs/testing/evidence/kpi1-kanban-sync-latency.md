# KPI 1 — Độ trễ đồng bộ bảng Kanban (NFR-05)

- **Mục tiêu:** p95 ≤ 2 giây, ≥ 30 lần đo.
- **Ngày đo:** 2026-11-21, máy dev (macOS, PostgreSQL Homebrew, API `make dev-local` ở `localhost:8080`).
- **Cách đo:** `scripts/kpi/kanban-sync-latency.mjs` mở 2 WebSocket vào dự án mẫu "Cổng thông tin nội bộ" —
  A = Manager (kéo), B = Member (xem). Mỗi lượt đo từ ngay trước `POST /tasks/{id}/move` đến khi B nhận
  `task.moved` đúng thẻ và đúng version mới. Thẻ "KPI1 — thẻ đo đồng bộ" kéo qua lại Backlog ↔ To Do.
- **Lệnh:** `KPI_MANAGER_PASSWORD=… KPI_MEMBER_PASSWORD=… KPI_RUNS=40 KPI_OUT=docs/testing/evidence/kpi1-kanban-sync-latency.json node scripts/kpi/kanban-sync-latency.mjs`

| Số lượt | min | p50 | **p95** | max | trung bình |
|---|---|---|---|---|---|
| 40 | 4,7 ms | 7,9 ms | **10 ms** | 11,4 ms | 7,8 ms |

**Kết luận: ĐẠT** (p95 = 10 ms ≤ 2000 ms). Dữ liệu từng lượt: `kpi1-kanban-sync-latency.json`.

Giới hạn: đo trên localhost nên không có độ trễ mạng thật; trên mạng LAN/Internet con số sẽ cao hơn
và nên đo lại trên môi trường demo trước khi bảo vệ.
