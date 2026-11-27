#!/usr/bin/env bash
# Chạy Claude Code ở "chế độ tự chạy" (CLAUDE.md §7c) qua nhiều nhánh nối tiếp.
# Mỗi nhánh mọc từ nhánh trước, kết thúc bằng feature-gate ĐẠT + PR draft. Dừng khi một nhánh không đạt.
#
#   ./scripts/auto-run.sh                      # chạy giai đoạn 1: scaffold → auth → org → projects → tasks
#   ./scripts/auto-run.sh --phase 2            # giai đoạn 2: docs/manual-testing → kanban → subtasks → logtime → review → reports → ai
#   ./scripts/auto-run.sh --only feature/org   # chạy lại đúng một nhánh
#   Hết giới hạn sử dụng: script tự chờ tới giờ mở lại rồi làm tiếp nhánh dở (tối đa MAX_LIMIT_RETRIES=6 lần).
#   ./scripts/auto-run.sh --dry-run            # chỉ in kế hoạch
#   ./scripts/auto-run.sh --from feature/auth  # bỏ qua các nhánh trước (đã xong), chạy tiếp từ nhánh này
#   ./scripts/auto-run.sh --base <nhánh>       # đổi nhánh gốc (mặc định: main ở giai đoạn 1, feature/kanban ở giai đoạn 2)
#
# Chế độ quyền mặc định: auto (đặt PERMISSION_MODE=acceptEdits để chặt hơn). Nếu bản Claude Code chưa hỗ trợ "auto" ở chế độ -p, đặt PERMISSION_MODE=acceptEdits.
# Log từng nhánh: .ai-log/runs/<nhanh>.jsonl (không commit). Mac được giữ thức trong lúc chạy.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"
PHASE=1; ONLY=""; DRY=0; BASE_ARG=""; FROM=""; MAX_TURNS="${MAX_TURNS:-300}"
while [ $# -gt 0 ]; do
  case "$1" in
    --phase) PHASE="$2"; shift 2 ;;
    --only) ONLY="$2"; shift 2 ;;
    --dry-run) DRY=1; shift ;;
    --base) BASE_ARG="$2"; shift 2 ;;
    --from) FROM="$2"; shift 2 ;;
    *) echo "Tham số không hợp lệ: $1"; exit 2 ;;
  esac
done

# ---- Kế hoạch: nhánh | phạm vi -------------------------------------------------
PHASE1=(
"chore/scaffold|Khung dự án, KHÔNG có tính năng nghiệp vụ. backend/: go mod init github.com/BaoCool-dev/WorkTrack/backend; cấu trúc Clean Architecture theo backend/CLAUDE.md; router chi với /healthz và /readyz (pgxpool; interface Pinger để unit test + 1 test tích hợp testcontainers-go có build tag integration); cấu hình từ biến môi trường theo .env.example; slog JSON + request ID; graceful shutdown; .golangci.yml; sqlc.yaml; thư mục db/migrations, db/queries. frontend/: Vite + React 19 + TS strict + Tailwind v4 (@tailwindcss/vite) + React Router + TanStack Query; theme.css đúng token docs/design/design-system.md §1; @fontsource/be-vietnam-pro 400/600; trang tạm gọi /healthz; ESLint + Prettier; Vitest + RTL + 1 test. docker-compose.yml: postgres:16, mailpit, api, web, healthcheck, volume. .github/workflows/ci.yml: build, lint, test Go + FE, test tích hợp, gitleaks, docker build. Makefile: dev, lint, test, test-integration, sqlc, migrate-up, migrate-new, cover, seed (seed để trống, các nhánh sau bổ sung) chạy được."
"feature/auth|CF-01. Backend: bảng app_user, role (4 vai trò Admin/Manager/Member/CEO), refresh_token; Admin tạo tài khoản → email kích hoạt qua Mailpit (B7) → đặt mật khẩu (B8); đăng nhập; JWT access + refresh xoay vòng (B4); đăng xuất; GET /api/v1/auth/me; khóa đăng nhập (B8); module internal/application/authz là nơi duy nhất quyết định quyền, CEO theo B2; middleware RequireAuth; giới hạn tần suất đăng nhập/làm mới. Frontend: trang đăng nhập, trang kích hoạt/đặt mật khẩu, AuthProvider (access token trong bộ nhớ, tự làm mới), route guard theo vai trò, AppShell (sidebar 236px + topbar theo design-system §4, menu theo vai trò), shared/ui cơ bản (Button, Input, Badge, Toast, Dialog, EmptyState, ErrorState, Skeleton). make seed: 1 tài khoản mỗi vai trò (B11). Test đủ 401/403 cho mọi vai trò."
"feature/org|CF-02. Phòng ban dạng cây, chức danh, danh mục kỹ năng, hồ sơ kỹ năng nhân sự (Beginner/Intermediate/Senior); màn Nhân sự & phòng ban cho Admin theo artboard 10 nếu có trong docs/design/artboards (danh sách có tìm kiếm, lọc, phân trang B9; form thêm/sửa với validate; lỗi 'Email đã tồn tại'). KHÔNG làm nhập/xuất Excel (Mở rộng). Mở rộng make seed với phòng ban, kỹ năng mẫu."
"feature/projects|CF-03. Manager tạo dự án trống (mẫu dự án là Mở rộng — không làm), sửa thông tin, trạng thái dự án (Mới tạo/Đang tiến hành/Hoàn thành/Tạm dừng/Đã hủy với badge đúng màu), thêm/bớt thành viên; cấu hình 2–8 cột Kanban (thêm, đổi tên, xóa, sắp xếp; mỗi cột có status_group todo|in_progress|done; cảnh báo khi xóa cột có việc); bảng workflow_transition với luật mặc định theo worktrack-domain (CHỈ lưu dữ liệu + API đọc/sửa cấu hình — việc ÁP DỤNG luật khi kéo thẻ thuộc feature/kanban do người dùng tự code). Danh sách dự án theo vai trò (Member chỉ thấy dự án mình tham gia; CEO thấy tất cả). Màn cấu hình theo artboard 12 nếu có."
"feature/tasks|CF-06. Công việc cha: Manager tạo, sửa, giao việc; Member tự nhận việc từ Backlog (giới hạn WIP là Mở rộng — không chặn, chỉ hiển thị số đếm); Story Point và giờ ước tính quy đổi theo B1 (tham số cấu hình); hạn chót; nhãn; API danh sách công việc theo dự án (có lọc, phân trang) và chi tiết; panel chi tiết công việc (không gồm công việc con, bình luận, Log time — các nhánh sau); trang 'Công việc của tôi' dạng bảng. KHÔNG làm bảng Kanban, kéo thả, chuyển cột hay realtime và KHÔNG tạo API chuyển cột — tất cả thuộc feature/kanban do người dùng tự code. Công việc mới tạo nằm ở cột đầu tiên có status_group=todo. Mở rộng make seed với công việc mẫu."
)
PHASE2=(
"docs/manual-testing|KHÔNG sửa code sản phẩm. Viết docs/testing/manual-test-guide.md: hướng dẫn cho người dùng CHƯA biết gì tự kiểm thử thủ công mọi chức năng đã có (scaffold, auth, org, projects, tasks), viết tiếng Việt, từng bước đánh số, dựa trên code thật (đọc config, router, migrations, seed, scripts) — không đoán. Gồm: (1) chuẩn bị môi trường không Docker theo B12: kiểm tra PostgreSQL và Mailpit đang chạy, liệt kê ĐÚNG mọi biến .env mà code đọc (giá trị mẫu, biến bắt buộc, JWT_SECRET tối thiểu 32 ký tự sinh bằng openssl rand -hex 32), make migrate-up, make seed (mật khẩu in ra màn hình và đổi mỗi lần chạy), make dev-local, các URL web/API/Mailpit; cách xử lý lỗi thường gặp khi khởi động; (2) đăng nhập bằng 4 tài khoản mẫu, đăng xuất, làm mới phiên, khóa sau nhiều lần sai; (3) luồng Admin tạo tài khoản → xem email trong Mailpit → kích hoạt → đặt mật khẩu; (4) với từng chức năng và từng vai trò: thao tác cụ thể, kết quả mong đợi, các ca lỗi, validate và phân quyền nên thử, câu SQL xem dữ liệu tương ứng; (5) xem CSDL bằng psql (\\dt, \\d tên_bảng, truy vấn mẫu) và pgAdmin (cách thêm server), bảng nào chứa gì; (6) thử API bằng curl (lấy access token rồi gọi endpoint); (7) đặt lại dữ liệu dev an toàn; (8) checklist kiểm thử theo CF-01, CF-02, CF-03, CF-06 để đánh dấu. Chạy thử từng lệnh và câu SQL trong tài liệu trên máy để chắc chắn đúng. Gate cho nhánh tài liệu: mọi lệnh và SQL chạy được; không cần test UI. Nhật ký AI chế độ tối giản (một entry)."
"feature/kanban|CF-04, CF-05. Claude viết bản đầu, người dùng sẽ chỉnh sửa — code dễ đọc, comment giải thích vì sao ở chỗ quan trọng (CLAUDE.md §7b). Backend: API di chuyển công việc cha giữa các cột và đổi thứ tự trong cột, áp dụng luật workflow_transition theo vai trò (dữ liệu đã có từ feature/projects); Member không đưa việc vào hoặc ra nhóm done; giới hạn WIP theo dự án (mặc định 5, chỉ đếm việc cha ở nhóm in_progress, Manager chỉnh trong cấu hình dự án) áp dụng cả khi kéo vào cột in_progress và khi nhận việc; chống ghi đè khi hai người cùng kéo (version); lưu lịch sử chuyển cột. Realtime theo skill realtime-websocket (coder/websocket): xác thực JWT và kiểm tra thành viên trước khi nhận kết nối, sự kiện task.created, task.updated, task.moved, presence; FE cập nhật cache TanStack Query, tự nối lại, dải cảnh báo khi mất kết nối. Frontend: bảng Kanban dnd-kit theo design-system §6 và artboard 02, 04, 13 nếu có: thẻ chuẩn, header cột có số đếm, chip WIP, vùng thả hợp lệ hoặc bị chặn kèm lý do cụ thể, cập nhật lạc quan + hoàn tác + dải lỗi khi lưu thất bại, lọc theo nhãn và người nhận, ít nhất 6 thẻ mỗi cột ở 1366x768, biến thể Member; hộp thoại chặn khi WIP đầy. Viết script đo KPI 1 (độ trễ đồng bộ giữa 2 client, tối thiểu 30 lần, p95) trong scripts/ và lưu kết quả vào docs/testing/evidence/. Bổ sung phần Kanban vào docs/testing/manual-test-guide.md. Nhật ký AI chế độ tối giản (một entry) nhưng vẫn có trailer AI-Log trên mỗi commit."
"feature/subtasks|CF-07. Công việc con 1 cấp do Member tạo (không Story Point, CHECK ở CSDL), Member tự chuyển trạng thái con; bình luận gắn công việc; thông báo trong hệ thống (chuông, đã đọc/chưa đọc) cho: được giao việc, có bình luận, bị từ chối nghiệm thu."
"feature/logtime|CF-08. Log time trên công việc cha và con (phút > 0, không log vào việc đã Done quá 24 giờ); tổng giờ thực tế = giờ cha + giờ con; thanh so giờ trên thẻ theo design-system §6 và ngưỡng B5; modal Log giờ; lịch sử log."
"feature/review|CF-09. Luồng nghiệm thu: Member đưa công việc cha sang cột Chờ nghiệm thu (cảnh báo nếu còn con chưa xong, vẫn cho phép); Manager Duyệt (sang nhóm done) hoặc Từ chối bắt buộc lý do (về in_progress, tính tỷ lệ làm lại); dùng luật chuyển cột của feature/kanban, không viết lại."
"feature/reports|CF-10, CF-11. Báo cáo tỷ lệ tuân thủ thời hạn và độ lệch giờ (chỉ công việc gốc; giờ con cộng lên cha) theo công thức worktrack-domain; Dashboard dự án cho Manager (artboard 07/08); Dashboard toàn công ty cho CEO CHỈ XEM (artboard 09, không endpoint ghi); test so khớp 100% với bảng tính tay trên bộ dữ liệu mẫu ≥ 30 công việc (KPI 3)."
"feature/ai|CF-12. Trợ lý AI Gemini qua port AIAssistant: phân rã yêu cầu thành công việc cha có Story Point; gợi ý 3 nhân sự kèm lý do; kết quả luôn là bản xem trước (Chấp nhận/Sửa/Bỏ qua), timeout, bảng ai_usage_log ghi thời gian phản hồi; không gửi dữ liệu nhạy cảm vào prompt. Nếu chưa có GEMINI_API_KEY: cài adapter giả lập để test và chạy giao diện, ghi rõ trong PR."
)

if [ "$PHASE" = "2" ]; then PLAN=("${PHASE2[@]}"); START_BASE="feature/tasks"; else PLAN=("${PHASE1[@]}"); START_BASE="main"; fi
[ -n "$BASE_ARG" ] && START_BASE="$BASE_ARG"
for _name in "$ONLY" "$FROM"; do
  [ -z "$_name" ] && continue
  _found=0; for item in "${PLAN[@]}"; do [ "${item%%|*}" = "$_name" ] && _found=1; done
  if [ "$_found" = 0 ]; then
    echo "✗ Nhánh '$_name' không có trong kế hoạch giai đoạn $PHASE. Có thể bạn cần thêm --phase 2."; exit 2
  fi
done

gate_file() { echo "docs/gates/$(echo "$1" | tr '/' '-').md"; }
notify() { command -v osascript >/dev/null && osascript -e "display notification \"$1\" with title \"WorkTrack auto-run\"" >/dev/null 2>&1; echo "$1"; }

echo "== Kế hoạch (giai đoạn $PHASE), nhánh gốc: $START_BASE"
_skip="$FROM"; for item in "${PLAN[@]}"; do b="${item%%|*}"; [ -n "$_skip" ] && [ "$b" != "$_skip" ] && continue; _skip=""; [ -n "$ONLY" ] && [ "$b" != "$ONLY" ] && continue; echo "   - $b"; done
[ "$DRY" = 1 ] && exit 0

# ---- Kiểm tra trước khi chạy ----------------------------------------------------
# Dùng đăng nhập claude.ai (gói Pro/Max) thay vì API key: ở chế độ -p, ANTHROPIC_API_KEY được ưu tiên
# nếu có trong môi trường. Đặt USE_API_KEY=1 nếu thật sự muốn tính phí theo API key.
if [ -z "${USE_API_KEY:-}" ] && { [ -n "${ANTHROPIC_API_KEY:-}" ] || [ -n "${ANTHROPIC_AUTH_TOKEN:-}" ]; }; then
  echo "ℹ Bỏ qua ANTHROPIC_API_KEY/ANTHROPIC_AUTH_TOKEN trong môi trường — dùng đăng nhập claude.ai."
  unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN
fi
# Token dài hạn cho chạy không người trông (tạo bằng: claude setup-token; lưu vào Keychain, xem docs/setup.md)
if [ -z "${CLAUDE_CODE_OAUTH_TOKEN:-}" ] && command -v security >/dev/null; then
  _t="$(security find-generic-password -s worktrack-claude-token -w 2>/dev/null || true)"
  [ -n "$_t" ] && export CLAUDE_CODE_OAUTH_TOKEN="$_t"
fi
need() { command -v "$1" >/dev/null || { echo "✗ Thiếu lệnh: $1"; exit 1; }; }
need claude; need git; need gh; need go; need node; need python3
if command -v docker >/dev/null && docker info >/dev/null 2>&1; then
  ENV_NOTE="Máy có Docker: chạy được make dev và make test-integration ở máy."
else
  echo "⚠ Không có Docker — dùng PostgreSQL/Mailpit cài bằng Homebrew (docs/decisions.md B12)."
  ENV_NOTE="Máy KHÔNG có Docker (docs/decisions.md B12): dùng PostgreSQL và Mailpit cài bằng Homebrew theo .env; chạy app bằng make dev-local; test tích hợp và docker build chỉ chạy trong CI — sau khi push, kiểm tra CI bằng gh pr checks / gh run view và ghi kết quả vào báo cáo gate."
  pg_isready -h localhost -p 5432 >/dev/null 2>&1 || echo "⚠ PostgreSQL ở localhost:5432 chưa sẵn sàng — brew services start postgresql@16"
fi
gh auth status >/dev/null 2>&1 || { echo "✗ Chưa đăng nhập gh — chạy: gh auth login"; exit 1; }
[ -z "$(git status --porcelain)" ] || { echo "✗ Có thay đổi chưa commit. Commit hoặc stash trước."; git status --short; exit 1; }
[ "$(git config core.hooksPath)" = ".githooks" ] || git config core.hooksPath .githooks
mkdir -p .ai-log/runs

# Giữ Mac không ngủ trong lúc chạy
command -v caffeinate >/dev/null && { caffeinate -dims -w $$ & }

BASE="$START_BASE"
for item in "${PLAN[@]}"; do
  BR="${item%%|*}"; SCOPE="${item#*|}"
  if [ -n "$FROM" ]; then if [ "$BR" = "$FROM" ]; then FROM=""; [ -n "$BASE_ARG" ] && BASE="$BASE_ARG"; else BASE="$BR"; continue; fi; fi
  if [ -n "$ONLY" ] && [ "$BR" != "$ONLY" ]; then BASE="$BR"; continue; fi
  echo; echo "================ $BR  (từ $BASE) ================"

  if git show-ref --verify --quiet "refs/heads/$BR"; then git switch -q "$BR"
  else git switch -q "$BASE" && git switch -q -c "$BR"; fi

  SLUG="$(echo "$BR" | tr '/' '-')"
  ATTEMPT=1
  while :; do
    DONE_COMMITS="$(git rev-list --count "$BASE..HEAD" 2>/dev/null || echo 0)"
    RESUME_NOTE=""
    if [ "$DONE_COMMITS" -gt 0 ]; then
      RESUME_NOTE="
NHÁNH ĐÃ LÀM DỞ ($DONE_COMMITS commit so với $BASE, có thể có thay đổi chưa commit). Trước khi viết thêm: đọc git log $BASE..HEAD, git status, entry nhật ký AI của nhánh và báo cáo gate (nếu có); chạy lại toàn bộ test; rồi làm TIẾP phần còn thiếu — không làm lại việc đã xong, không tạo entry nhật ký mới (bổ sung vào entry cũ)."
    fi
    PROMPT="Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng, người dùng không có mặt.
Nhánh hiện tại: $BR (mọc từ $BASE). PR sẽ tạo với --base $BASE --draft.
Môi trường: $ENV_NOTE$RESUME_NOTE

Phạm vi nhánh này: $SCOPE

Quy trình bắt buộc:
1. Đọc CLAUDE.md, docs/decisions.md, CONTRIBUTING.md, skill worktrack-domain; nếu là phần giao diện: docs/design/design-system.md và artboard liên quan trong docs/design/artboards/ (nếu có).
2. Làm theo skill implement-ticket theo lát dọc: migration → sqlc → domain → application → httpapi → frontend; test viết trước cho logic nghiệp vụ.
3. Tra cú pháp thư viện bằng Context7, không đoán. Lựa chọn chưa có trong docs/decisions.md: tự chọn theo §7c và ghi vào mục C.
4. Commit thành nhiều commit nhỏ hợp lý, mỗi commit qua skill ai-usage-log (một entry cho cả nhánh, slug theo tên nhánh, model đúng model id đang chạy). Tác giả commit theo git config sẵn có. Commit sau mỗi phần làm xong để không mất việc nếu phiên bị dừng giữa chừng.
5. Nếu docs/testing/manual-test-guide.md đã có: bổ sung phần kiểm thử thủ công cho tính năng của nhánh này (thao tác, kết quả mong đợi, SQL xem dữ liệu).
5b. Khi xong: chạy skill feature-gate đầy đủ; sửa tới khi ĐẠT (tối đa 3 vòng). Báo cáo ở $(gate_file "$BR").
6. ĐẠT → git push -u origin $BR → gh pr create --base $BASE --draft (mô tả: mục tiêu, mã CF, tóm tắt gate, giả định mục C).
   CHƯA ĐẠT → vẫn commit báo cáo gate, push nhánh, KHÔNG tạo PR, kết thúc và nêu rõ cần người dùng làm gì.
Cuối cùng in đúng một dòng: GATE=ĐẠT hoặc GATE=CHƯA_ĐẠT."

    LOG=".ai-log/runs/$SLUG$([ "$ATTEMPT" -gt 1 ] && echo "-$ATTEMPT").jsonl"
    claude -p "$PROMPT" \
      --permission-mode "${PERMISSION_MODE:-auto}" \
      --disallowedTools "AskUserQuestion" \
      --max-turns "$MAX_TURNS" \
      --output-format stream-json --verbose \
      > "$LOG" 2> "${LOG%.jsonl}.err"
    CODE=$?

    RESULT="$(python3 - "$LOG" <<'PY2'
import json,sys
last=""
for line in open(sys.argv[1],encoding="utf-8"):
    try: d=json.loads(line)
    except Exception: continue
    if d.get("type")=="result": last=d.get("result") or ""
print(last)
PY2
)"
    echo "$RESULT" | tail -15

    # Hết giới hạn sử dụng → chờ tới giờ mở lại rồi làm tiếp đúng nhánh này
    if printf '%s\n%s' "$RESULT" "$(tail -c 3000 "${LOG%.jsonl}.err" 2>/dev/null)" | grep -qiE "hit your .*limit|usage limit|limit reached|rate.?limit"; then
      if [ "$ATTEMPT" -ge "${MAX_LIMIT_RETRIES:-6}" ]; then notify "Dừng ở $BR — hết giới hạn quá ${MAX_LIMIT_RETRIES:-6} lần"; exit 1; fi
      WAIT="$(printf '%s' "$RESULT" | python3 -c '
import re,sys,datetime
s=sys.stdin.read()
m=re.search(r"resets\s+(\d{1,2})(?::(\d{2}))?\s*([ap]m)",s,re.I)
now=datetime.datetime.now()
if not m: print(3600); sys.exit()
h=int(m.group(1))%12+(12 if m.group(3).lower()=="pm" else 0); mi=int(m.group(2) or 0)
t=now.replace(hour=h,minute=mi,second=0,microsecond=0)
if t<=now: t+=datetime.timedelta(days=1)
print(int((t-now).total_seconds())+180)')"
      notify "Hết giới hạn ở $BR — tự chạy tiếp lúc $(date -v+"${WAIT}"S +%H:%M 2>/dev/null || echo "sau $((WAIT/60)) phút")"
      sleep "$WAIT"
      ATTEMPT=$((ATTEMPT+1))
      continue
    fi
    break
  done

  GATE="$(gate_file "$BR")"
  if [ $CODE -ne 0 ] || ! echo "$RESULT" | grep -q "GATE=ĐẠT" || [ ! -f "$GATE" ]; then
    notify "Dừng ở $BR — xem $GATE và .ai-log/runs/"; exit 1
  fi
  if ! git ls-remote --exit-code --heads origin "$BR" >/dev/null 2>&1; then
    notify "Dừng ở $BR — nhánh chưa được push"; exit 1
  fi
  notify "Xong $BR — PR draft đã tạo"
  BASE="$BR"
done

if [ "$PHASE" = "1" ] && [ -z "$ONLY" ]; then
  notify "Giai đoạn 1 xong. Tiếp: ./scripts/auto-run.sh --phase 2"
else
  notify "Hoàn tất."
fi
