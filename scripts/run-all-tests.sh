#!/usr/bin/env bash
# Chạy THẬT toàn bộ test tự động (Go unit/API, Go tích hợp với PostgreSQL, Vitest) + độ phủ,
# ghi kết quả máy đọc được vào <thư mục run> rồi in tóm tắt. Không gọi AI.
#   ./scripts/run-all-tests.sh [thư mục run]   (mặc định .ai-log/test-runs/<thời gian>)
set -uo pipefail
HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="${REPO_ROOT:-$(cd "$HERE/.." && pwd)}"; cd "$ROOT"
RUN="${1:-.ai-log/test-runs/$(date +%Y%m%d-%H%M)}"; mkdir -p "$RUN"; RUN="$(cd "$RUN" && pwd)"
echo "== Chạy test trên $(git branch --show-current) @ $(git rev-parse --short HEAD) → $RUN"
git rev-parse --short HEAD > "$RUN/commit.txt"

if [ ! -f backend/go.mod ]; then echo "(chưa có backend ở bước này)"; : > "$RUN/go-unit.json"; : > "$RUN/go-integration.json"; SKIP_GO=1; else SKIP_GO=0; fi
if [ "$SKIP_GO" = 0 ]; then
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
fi

if [ -f frontend/package.json ]; then
  echo "-- vitest"
  LOCKHASH="$(shasum frontend/package-lock.json 2>/dev/null | cut -d' ' -f1)"
  if [ ! -d frontend/node_modules ] || [ "$(cat frontend/node_modules/.lockhash 2>/dev/null)" != "$LOCKHASH" ]; then
    echo "   npm ci (thư viện thay đổi)"; ( cd frontend && npm ci --silent ) > "$RUN/npm-ci.log" 2>&1 && echo "$LOCKHASH" > frontend/node_modules/.lockhash
  fi
  ( cd frontend && npx vitest run --reporter=json --outputFile="$RUN/vitest.json" ) > "$RUN/vitest.log" 2>&1
else echo "(chưa có frontend ở bước này)"; fi

python3 "$HERE/test-summary.py" "$RUN"
