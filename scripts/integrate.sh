#!/usr/bin/env bash
# Gộp LẦN LƯỢT các nhánh vào một nhánh tích hợp mới (mặc định release/v1, KHÔNG đụng main).
# Sau MỖI lần gộp: chạy lại toàn bộ test, rà cấu hình, tìm và dọn code rác, chạy lại test, viết báo cáo,
# gắn tag integ/NN-<nhánh> và đẩy nhánh sao lưu integration/NN-<nhánh> lên GitHub: git diff integ/03-auth integ/04-org
#
#   ./scripts/integrate.sh                    # chạy hết các bước còn lại
#   ./scripts/integrate.sh --step             # chỉ làm 1 bước tiếp theo rồi dừng (để bạn kiểm xong mới chạy tiếp)
#   ./scripts/integrate.sh --target release/v1 --no-push
#
# Báo cáo từng bước: docs/integration/NN-<nhánh>.md. Kết quả thô: .ai-log/integration/ (không commit).
set -uo pipefail
ROOT="${REPO_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"; cd "$ROOT"
# Chạy từ bản sao ở thư mục tạm: khi chuyển/gộp nhánh, thư mục scripts/ trong repo thay đổi theo từng bước.
if [ -z "${INTEGRATE_TOOLS:-}" ]; then
  T="$(mktemp -d)"; cp scripts/integrate.sh scripts/run-all-tests.sh scripts/test-summary.py "$T/"
  INTEGRATE_TOOLS="$T" REPO_ROOT="$ROOT" exec bash "$T/integrate.sh" "$@"
fi
RUNTESTS="$INTEGRATE_TOOLS/run-all-tests.sh"
TARGET="release/v1"; ONE=0; PUSH=1
while [ $# -gt 0 ]; do case "$1" in
  --target) TARGET="$2"; shift 2 ;; --step) ONE=1; shift ;; --no-push) PUSH=0; shift ;;
  *) echo "Tham số không hợp lệ: $1"; exit 2 ;; esac; done
[ "$TARGET" = "main" ] && { echo "✗ Không tích hợp thẳng vào main."; exit 2; }

PLAN=(chore/ai-workflow chore/scaffold feature/auth feature/org feature/projects feature/tasks docs/manual-testing feature/kanban test/test-cases)
# Các chức năng làm sau (giai đoạn 2) — chỉ đưa vào kế hoạch khi nhánh đã tồn tại
for b in feature/subtasks feature/logtime feature/review feature/reports feature/ai; do
  git show-ref -q "refs/heads/$b" && PLAN+=("$b")
done
FINAL="${PLAN[${#PLAN[@]}-1]}"

if [ -z "${USE_API_KEY:-}" ]; then unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN; fi
# Token dài hạn cho chạy không người trông (tạo bằng: claude setup-token; lưu vào Keychain, xem docs/setup.md)
if [ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ] && command -v security >/dev/null; then
  _t="$(security find-generic-password -s worktrack-claude-token -w 2>/dev/null || true)"
  [ -n "$_t" ] && export CLAUDE_CODE_OAUTH_TOKEN="$_t"
fi
need() { command -v "$1" >/dev/null || { echo "✗ Thiếu lệnh: $1"; exit 1; }; }
need claude; need git; need gh; need go; need node; need python3
[ -z "$(git status --porcelain)" ] || { echo "✗ Có thay đổi chưa commit."; git status --short; exit 1; }
for b in "${PLAN[@]}"; do git show-ref -q "refs/heads/$b" || { echo "✗ Thiếu nhánh cục bộ $b"; exit 1; }; done
notify() { command -v osascript >/dev/null && osascript -e "display notification \"$1\" with title \"WorkTrack integrate\"" >/dev/null 2>&1; echo "$1"; }
command -v caffeinate >/dev/null && { caffeinate -dims -w $$ & }

echo "-- công cụ tìm code rác"
command -v deadcode >/dev/null || go install golang.org/x/tools/cmd/deadcode@latest
export PATH="$PATH:$(go env GOPATH)/bin"

git show-ref -q "refs/heads/$TARGET" || { git switch -q -c "$TARGET" main && echo "== Tạo $TARGET từ main"; }
git switch -q "$TARGET"

PREV_TAG=""
for i in "${!PLAN[@]}"; do
  BR="${PLAN[$i]}"; NN="$(printf '%02d' $((i+1)))"; SLUG="$(echo "${BR#*/}" | tr '/' '-')"; TAG="integ/$NN-$SLUG"
  if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null && git merge-base --is-ancestor "$BR" "$TARGET"; then
    PREV_TAG="$TAG"; continue   # bước đã xong
  fi
  echo; echo "================ Bước $NN/${#PLAN[@]}: gộp $BR vào $TARGET ================"
  RUN=".ai-log/integration/$NN-$SLUG"; mkdir -p "$RUN"
  BEFORE="$(git rev-parse HEAD)"

  if ! git merge --no-ff --no-edit -m "merge: gộp $BR vào $TARGET (bước $NN)" "$BR" > "$RUN/merge.log" 2>&1; then
    CONFLICT="Gộp bị XUNG ĐỘT (xem git status, $RUN/merge.log). Giải quyết xung đột trước: ưu tiên giữ phần dọn dẹp của các bước trước nếu đoạn đó không còn dùng ở bản cuối $FINAL, còn lại lấy theo $BR; rồi git commit để hoàn tất merge."
  else CONFLICT="Gộp không xung đột."; fi

  if [ -z "$(git status --porcelain)" ]; then
    REPO_ROOT="$ROOT" bash "$RUNTESTS" "$RUN/after-merge" > "$RUN/after-merge.log" 2>&1
    [ -f backend/go.mod ] && ( cd backend && deadcode -test ./... ) > "$RUN/deadcode.txt" 2>&1
    [ -f frontend/package.json ] && ( cd frontend && npx -y knip --no-progress ) > "$RUN/knip.txt" 2>&1
  fi

  PROMPT="Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng. Đang TÍCH HỢP từng bước vào nhánh $TARGET (KHÔNG phải main).
Bước $NN/${#PLAN[@]}: vừa gộp nhánh $BR (trước gộp HEAD=$BEFORE${PREV_TAG:+, tag bước trước $PREV_TAG}). $CONFLICT
Tuyệt đối: không push/merge main; không rebase, amend, force-push; không sửa các nhánh nguồn; không tạo/xóa tag (script làm).

Đầu vào đã chạy sẵn (bước đầu chưa có backend/frontend thì các file này có thể trống — khi đó chỉ rà cấu hình và tài liệu): $RUN/after-merge/summary.md (test thật sau gộp), $RUN/deadcode.txt (hàm Go không tới được), $RUN/knip.txt (file/export/dependency FE không dùng). Thay đổi mà bước này mang vào: git diff --stat $BEFORE..HEAD.

Việc phải làm:
1. TEST: đọc summary sau gộp. Test lỗi → xác định do gộp (xung đột ngầm) hay lỗi có sẵn ở $BR. Do gộp → sửa tối thiểu, commit fix(integration): ...; có sẵn → tạo GitHub issue theo .github/ISSUE_TEMPLATE/bug_report.md (tìm trùng trước), KHÔNG sửa.
2. CẤU HÌNH (phần bước này mang vào và tổng thể): biến môi trường code đọc so với .env.example (thiếu, thừa, mặc định sai — sửa .env.example, không bao giờ đọc/sửa .env); docker-compose.yml; .github/workflows (lệnh/script được gọi có tồn tại, phiên bản Go/Node khớp go.mod/package.json); Makefile (target trỏ tới script có thật); go mod tidy và npm (lockfile) không tạo khác biệt; sqlc generate không tạo khác biệt; migration goose: tên tăng dần không trùng, goose -dir backend/db/migrations validate; .gitignore phủ file sinh ra; .claude/settings.json, .mcp.json hợp lệ; gitleaks detect sạch. Mỗi nhóm sửa một commit fix(config): ...
3. CODE RÁC: dựa trên deadcode, knip, golangci-lint (unused), cộng tự rà: TODO/FIXME, console.log hoặc fmt.Println gỡ lỗi, code bị comment-out, file không ai import, dependency không dùng, script/tài liệu trỏ tới thứ không còn, trang/route giữ chỗ không còn cần.
   QUY TẮC: chỉ xóa khi KHÔNG còn dùng ở bản cuối $FINAL — kiểm bằng git grep -n '<tên>' $FINAL -- backend frontend scripts và git show $FINAL:<file>. Bản cuối còn dùng → GIỮ, ghi 'dùng ở bước sau'. Không xóa test, migration, nhật ký AI, báo cáo gate, tài liệu kiểm thử. Mỗi loại dọn một commit chore(cleanup): ... để dễ xem.
4. Chạy lại: REPO_ROOT=$ROOT bash $RUNTESTS $RUN/after-cleanup (dùng đúng lệnh này, không dùng scripts/run-all-tests.sh trong repo vì có thể chưa tồn tại ở bước này) — số test Pass không được giảm so với sau gộp. Commit dọn nào làm đỏ → git revert commit đó, ghi lại.
5. BÁO CÁO docs/integration/$NN-$SLUG.md: (a) nhánh gộp: số commit, chức năng mang vào; (b) test sau gộp và sau dọn (bảng số liệu); (c) bảng cấu hình: vấn đề | mức | xử lý | commit; (d) bảng code rác: mục | bằng chứng | xử lý (đã xóa / giữ vì dùng ở bước sau / giữ vì lý do khác) | commit; (e) Việc người dùng tự kiểm: lệnh cụ thể (git diff ${PREV_TAG:-main}..HEAD -- <thư mục>, make dev-local, màn hình nên mở, vai trò nên thử); (f) Không chắc chắn: mọi quyết định AI không chắc, để người dùng quyết.
6. Commit báo cáo qua skill ai-usage-log — nhật ký tối giản: MỘT entry chung cho cả quá trình tích hợp (docs/ai-log/entries/*-integration-*.md; chưa có thì tạo, có rồi thì thêm mục cho bước $NN).
In dòng cuối: STEP=OK hoặc STEP=FAIL kèm lý do."

  ATTEMPT=1
  while :; do
    LOG="$RUN/claude$([ "$ATTEMPT" -gt 1 ] && echo "-$ATTEMPT").jsonl"
    claude -p "$PROMPT" --permission-mode "${PERMISSION_MODE:-auto}" --disallowedTools "AskUserQuestion" \
      --max-turns "${MAX_TURNS:-250}" --output-format stream-json --verbose > "$LOG" 2> "${LOG%.jsonl}.err"
    RESULT="$(python3 -c 'import json,sys
r=""
for l in open(sys.argv[1],encoding="utf-8"):
    try: d=json.loads(l)
    except Exception: continue
    if d.get("type")=="result": r=d.get("result") or ""
print(r)' "$LOG")"
    echo "$RESULT" | tail -6
    if printf '%s' "$RESULT" | grep -qiE "hit your .*limit|usage limit|limit reached"; then
      [ "$ATTEMPT" -ge 6 ] && { notify "Dừng ở bước $NN — hết giới hạn quá 6 lần"; exit 1; }
      WAIT="$(printf '%s' "$RESULT" | python3 -c '
import re,sys,datetime
s=sys.stdin.read(); m=re.search(r"resets\s+(\d{1,2})(?::(\d{2}))?\s*([ap]m)",s,re.I); now=datetime.datetime.now()
if not m: print(3600); sys.exit()
h=int(m.group(1))%12+(12 if m.group(3).lower()=="pm" else 0); t=now.replace(hour=h,minute=int(m.group(2) or 0),second=0,microsecond=0)
t=t if t>now else t+datetime.timedelta(days=1); print(int((t-now).total_seconds())+180)')"
      notify "Hết giới hạn ở bước $NN — tự chạy tiếp sau $((WAIT/60)) phút"; sleep "$WAIT"; ATTEMPT=$((ATTEMPT+1)); continue
    fi
    break
  done

  if ! printf '%s' "$RESULT" | grep -q "STEP=OK" || [ -n "$(git status --porcelain)" ] || [ ! -f "docs/integration/$NN-$SLUG.md" ]; then
    notify "Dừng ở bước $NN ($BR) — xem $RUN/ và docs/integration/"; exit 1
  fi
  git tag -f "$TAG" >/dev/null
  SNAP="integration/$NN-$SLUG"; git branch -f "$SNAP" HEAD >/dev/null
  if [ "$PUSH" = 1 ]; then
    git push -q -u origin "$TARGET"
    git push -q origin "$SNAP" "refs/tags/$TAG" 2>/dev/null || echo "⚠ Không đẩy được $SNAP hoặc tag $TAG (đã có trên GitHub?)"
  fi
  notify "Xong bước $NN: $BR — báo cáo docs/integration/$NN-$SLUG.md"
  PREV_TAG="$TAG"
  [ "$ONE" = 1 ] && { echo "Dừng theo --step. Kiểm tra xong thì chạy lại lệnh để làm bước tiếp theo."; exit 0; }
done
notify "Tích hợp xong vào $TARGET. main chưa bị đụng tới."
