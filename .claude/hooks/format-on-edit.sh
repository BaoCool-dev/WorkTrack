#!/usr/bin/env bash
# PostToolUse: tự format file vừa sửa. Không bao giờ làm hỏng phiên — lỗi format chỉ in cảnh báo.
set -uo pipefail

input="$(cat)"
if command -v jq >/dev/null 2>&1; then
  file="$(printf '%s' "$input" | jq -r '.tool_input.file_path // empty')"
else
  file="$(printf '%s' "$input" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))')"
fi
[ -z "$file" ] || [ ! -f "$file" ] && exit 0

root="${CLAUDE_PROJECT_DIR:-.}"

case "$file" in
  *.go)
    if command -v goimports >/dev/null 2>&1; then
      goimports -w "$file" || echo "goimports lỗi: $file" >&2
    else
      gofmt -w "$file" || echo "gofmt lỗi: $file" >&2
    fi
    ;;
  *.ts|*.tsx|*.css|*.json|*.md)
    if [ -x "$root/frontend/node_modules/.bin/prettier" ] && [[ "$file" == "$root/frontend/"* ]]; then
      (cd "$root/frontend" && ./node_modules/.bin/prettier --write --log-level warn "$file") \
        || echo "prettier lỗi: $file" >&2
    fi
    ;;
esac
exit 0
