#!/usr/bin/env bash
# Chạy THẬT toàn bộ test tự động (Go unit/API, Go tích hợp với PostgreSQL, Vitest) + độ phủ,
# ghi kết quả máy đọc được vào <thư mục run> rồi in tóm tắt. Không gọi AI.
#   ./scripts/run-all-tests.sh [thư mục run]   (mặc định .ai-log/test-runs/<thời gian>)
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"
RUN="${1:-.ai-log/test-runs/$(date +%Y%m%d-%H%M)}"; mkdir -p "$RUN"; RUN="$(cd "$RUN" && pwd)"
echo "== Chạy test trên $(git branch --show-current) @ $(git rev-parse --short HEAD) → $RUN"
git rev-parse --short HEAD > "$RUN/commit.txt"

echo "-- go test (unit, API)"
( cd backend && go test -json -count=1 ./... ) > "$RUN/go-unit.json" 2> "$RUN/go-unit.err"

echo "-- độ phủ gói cốt lõi"
( cd backend && go test -count=1 -coverprofile="$RUN/coverage.out" ./internal/domain/... ./internal/application/... >/dev/null 2>&1 \
  && go tool cover -func="$RUN/coverage.out" ) > "$RUN/coverage.txt" 2>&1

IT_DB="worktrack_it_testcases"
DEV_URL="postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable"
if command -v psql >/dev/null && psql "$DEV_URL" -Atc "select 1" >/dev/null 2>&1; then
  echo "-- go test -tags=integration (CSDL riêng $IT_DB)"
  psql "$DEV_URL" -Atc "select 1 from pg_database where datname='$IT_DB'" | grep -q 1 \
    || psql "$DEV_URL" -qc "create database $IT_DB owner worktrack" >/dev/null
  ( cd backend && TEST_DATABASE_URL="postgres://worktrack:worktrack@localhost:5432/$IT_DB?sslmode=disable" \
      go test -tags=integration -json -count=1 ./... ) > "$RUN/go-integration.json" 2> "$RUN/go-integration.err"
else
  echo "⚠ Không kết nối được PostgreSQL ở localhost:5432 — bỏ qua test tích hợp (brew services start postgresql@16)."
  : > "$RUN/go-integration.json"
fi

echo "-- vitest"
( cd frontend && npx vitest run --reporter=json --outputFile="$RUN/vitest.json" ) > "$RUN/vitest.log" 2>&1

python3 scripts/test-summary.py "$RUN"
