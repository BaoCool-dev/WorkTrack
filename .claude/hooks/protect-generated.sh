#!/usr/bin/env bash
# PreToolUse: chặn sửa tay code sinh tự động (sqlc) và migration đã commit.
# Exit 2 = chặn, thông điệp stderr được gửi lại cho Claude.
set -euo pipefail

input="$(cat)"
if command -v jq >/dev/null 2>&1; then
  file="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')"
else
  file="$(printf '%s' "$input" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))')"
fi
[ -z "$file" ] && exit 0

case "$file" in
  */internal/infrastructure/postgres/sqlcgen/*)
    echo "Không sửa tay code sinh bởi sqlc ($file). Sửa backend/db/queries/*.sql rồi chạy 'make sqlc'." >&2
    exit 2
    ;;
  */db/migrations/*.sql)
    # Cho phép tạo/sửa migration chưa commit; chặn nếu file đã nằm trong lịch sử git.
    dir="${CLAUDE_PROJECT_DIR:-.}"
    if [ -f "$file" ] && git -C "$dir" ls-files --error-unmatch "$file" >/dev/null 2>&1; then
      echo "Migration $file đã được commit. Không sửa migration đã áp dụng — tạo migration mới: make migrate-new name=<ten>." >&2
      exit 2
    fi
    ;;
esac
exit 0
