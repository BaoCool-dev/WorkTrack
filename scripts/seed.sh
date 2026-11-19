#!/usr/bin/env bash
# Tạo dữ liệu mẫu (B11). Nạp .env giống dev-local, không in giá trị cấu hình.
set -euo pipefail
cd "$(dirname "$0")/.."
if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi
cd backend && go run ./cmd/seed
