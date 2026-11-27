#!/usr/bin/env bash
# Chạy WorkTrack trên máy không có Docker (docs/decisions.md B12).
# Yêu cầu: PostgreSQL 16 và Mailpit chạy sẵn (Homebrew), file .env ở gốc repo.
# Script chỉ nạp .env vào môi trường tiến trình con, không in giá trị nào ra màn hình.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
else
  echo "⚠ Không thấy .env — dùng biến môi trường hiện có (xem .env.example)." >&2
fi

trap 'kill 0' EXIT INT TERM

(cd backend && go run ./cmd/api) &
(cd frontend && npm run dev) &
wait
