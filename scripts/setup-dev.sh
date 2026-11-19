#!/usr/bin/env bash
# Cài công cụ dev + binary mà plugin LSP/MCP của Claude Code cần (macOS, Homebrew).
# Chạy lại an toàn: bỏ qua cái đã có.
set -euo pipefail

say() { printf '\n\033[1;34m==> %s\033[0m\n' "$*"; }
have() { command -v "$1" >/dev/null 2>&1; }

say "Kiểm tra Homebrew"
have brew || { echo "Cần Homebrew: https://brew.sh"; exit 1; }

say "Go, Node, Docker"
have go   || brew install go
have node || brew install node
have docker || echo "⚠ Chưa có Docker Desktop — cần cho docker compose, testcontainers và Postgres MCP."

say "Công cụ Go (gopls cho LSP + MCP, goimports, goose)"
have gopls     || go install golang.org/x/tools/gopls@latest
have goimports || go install golang.org/x/tools/cmd/goimports@latest
have goose     || go install github.com/pressly/goose/v3/cmd/goose@latest

say "sqlc, golangci-lint, gitleaks, k6, jq"
for t in sqlc golangci-lint gitleaks k6 jq; do have "$t" || brew install "$t"; done

say "TypeScript language server (cho plugin typescript-lsp)"
have typescript-language-server || npm install -g typescript-language-server typescript

say "Kiểm tra PATH cho binary Go"
GOBIN_DIR="$(go env GOPATH)/bin"
case ":$PATH:" in
  *":$GOBIN_DIR:"*) echo "OK: $GOBIN_DIR có trong PATH" ;;
  *) echo "⚠ Thêm vào ~/.zshrc:  export PATH=\"\$PATH:$GOBIN_DIR\"  rồi mở lại terminal" ;;
esac

say "Hook Claude + git hook nhật ký AI"
REPO="$(cd "$(dirname "$0")/.." && pwd)"
chmod +x "$REPO"/.claude/hooks/* "$REPO"/.githooks/* "$REPO"/scripts/ai-log/ai_log.py
if git -C "$REPO" rev-parse --git-dir >/dev/null 2>&1; then
  git -C "$REPO" config core.hooksPath .githooks && echo "OK: bật .githooks (commit-msg kiểm tra AI-Log)"
else
  echo "⚠ Chưa phải repo git — sau khi git init, chạy: git config core.hooksPath .githooks"
fi

say "Phiên bản"
go version; node --version; gopls version | head -1 || true
sqlc version || true; golangci-lint --version || true; goose --version || true

cat <<'EOF'

Xong phần công cụ. Bước tiếp theo (xem docs/setup.md):
  1. cd vào repo, chạy `claude` → chấp nhận tin cậy thư mục → đồng ý cài plugin & MCP của dự án.
  2. Trong Claude: /plugin  (kiểm tra plugin đã bật)   /mcp  (kiểm tra MCP đã kết nối)
EOF
