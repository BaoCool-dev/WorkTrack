#!/usr/bin/env bash
# MỘT LỆNH: kiểm thử toàn bộ chức năng đã có theo chuẩn docs/testing/test-design-standard.md
# (bài giảng Kiểm thử phần mềm HCM-UTE): chạy thật mọi test tự động → Claude Code phân tích điều kiện kiểm thử,
# thiết kế ca bằng EP/BVA/bảng quyết định/chuyển trạng thái/use case, viết thêm test còn thiếu, chạy lại,
# lập bảng test case, báo lỗi lên GitHub Issues và viết báo cáo tổng kết kiểm thử.
#
#   ./scripts/test-cases.sh            # đầy đủ; ca thủ công ghi "Chưa chạy"
#   ./scripts/test-cases.sh --ui       # thêm: Claude chạy app và thực thi ca giao diện bằng Playwright, chụp minh chứng
#   ./scripts/test-cases.sh --no-ai    # chỉ chạy test thật và in tóm tắt (= scripts/run-all-tests.sh)
#
# Chạy trên nhánh tính năng mới nhất (các nhánh xếp chồng nên nhánh cuối có đủ chức năng).
# Kết quả: nhánh test/test-cases + PR draft; kết quả thô ở .ai-log/test-runs/ (không commit).
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"; cd "$ROOT"
UI=0; AI=1
for a in "$@"; do case "$a" in --ui) UI=1 ;; --no-ai) AI=0 ;; *) echo "Tham số không hợp lệ: $a"; exit 2 ;; esac; done

if [ -z "${USE_API_KEY:-}" ]; then unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN; fi
need() { command -v "$1" >/dev/null || { echo "✗ Thiếu lệnh: $1"; exit 1; }; }
need go; need node; need python3; need git; [ "$AI" = 1 ] && { need claude; need gh; }
[ -z "$(git status --porcelain)" ] || { echo "✗ Có thay đổi chưa commit. Commit hoặc stash trước."; git status --short; exit 1; }
notify() { command -v osascript >/dev/null && osascript -e "display notification \"$1\" with title \"WorkTrack test-cases\"" >/dev/null 2>&1; echo "$1"; }

SRC_BRANCH="$(git branch --show-current)"
[ "$SRC_BRANCH" = "test/test-cases" ] && SRC_BRANCH="$(git rev-parse --abbrev-ref '@{-1}' 2>/dev/null || echo feature/kanban)"
STAMP="$(date +%Y%m%d-%H%M)"; RUN=".ai-log/test-runs/$STAMP"
command -v caffeinate >/dev/null && { caffeinate -dims -w $$ & }

./scripts/run-all-tests.sh "$RUN"
[ "$AI" = 0 ] && { notify "Đã chạy xong test — xem $RUN/summary.md"; exit 0; }

if git show-ref --verify --quiet refs/heads/test/test-cases; then
  git switch -q test/test-cases && git merge -q --no-edit "$SRC_BRANCH" || { echo "✗ Không gộp được $SRC_BRANCH vào test/test-cases"; exit 1; }
else
  git switch -q -c test/test-cases "$SRC_BRANCH"
fi
mkdir -p docs/testing/runs && cp "$RUN/summary.md" "docs/testing/runs/$STAMP-before.md"

UI_RULE="Ca giao diện/thủ công: không chạy app — Trạng thái 'Chưa chạy', Kết quả thực tế để trống."
[ "$UI" = 1 ] && UI_RULE="Ca giao diện/thủ công: make seed, chạy make dev-local ở chế độ nền, dùng Playwright MCP ở 1366x768 thực thi từng ca theo docs/testing/manual-test-guide.md; ghi Kết quả thực tế đúng như quan sát và Trạng thái; lưu ảnh docs/testing/evidence/<MÃ_TC>.png cho mọi ca Fail và 1 ca tiêu biểu mỗi chức năng; tắt app khi xong. Ca realtime cần 2 trình duyệt: mở 2 tab với 2 vai trò."

PROMPT="Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng. Nhánh: test/test-cases (mọc từ $SRC_BRANCH). KHÔNG sửa code sản phẩm, KHÔNG sửa hay xóa test có sẵn.

Nhiệm vụ: kiểm thử TOÀN BỘ chức năng đã có của WorkTrack theo đúng chuẩn trong docs/testing/test-design-standard.md (skill test-case-design) — tài liệu này tóm tắt bài giảng Kiểm thử phần mềm của trường và là căn cứ chấm. Đọc kỹ chuẩn đó trước tiên.

Nguồn: kết quả chạy THẬT $RUN/summary.json và summary.md (mọi test, trạng thái, mã CF, độ phủ), log thô trong $RUN/; docs/testing/traceability-matrix.md, manual-test-guide.md, docs/gates/*.md, docs/decisions.md, skill worktrack-domain; code (validator, domain, router, migrations) để lấy đúng miền giá trị và luật.

Các bước:
1. PHÂN TÍCH — mỗi chức năng đã có (CF-01, CF-02, CF-03, CF-04/05, CF-06 và các CF khác nếu code đã có): bảng điều kiện kiểm thử TCond_<MODULE>_<NN> có cột nguồn; mục 'Câu hỏi cần làm rõ' cho yêu cầu mơ hồ.
2. THIẾT KẾ — áp dụng kỹ thuật theo bảng §4 của chuẩn, trình bày TƯỜNG MINH trong tài liệu: bảng EP + BVA cho từng trường có miền giá trị (lấy ngưỡng từ code); bảng quyết định cho phân quyền vai trò × hành động, luật chuyển cột vai trò × nhóm nguồn × nhóm đích, đăng nhập; sơ đồ chuyển trạng thái Mermaid cho tài khoản/đăng nhập sai/công việc/dự án kèm ca chuyển hợp lệ và không hợp lệ; use case (chính/thay thế/ngoại lệ) cho luồng chính; checklist đoán lỗi bảo mật (IDOR, token, XSS, SQL injection, bấm hai lần, hai người cùng sửa). Mỗi ca suy ra từ một điều kiện và một kỹ thuật.
3. LẤP KHOẢNG TRỐNG — với ca ưu tiên Cao tự động hóa được mà chưa có test: VIẾT THÊM test tự động mới (Go table-driven hoặc Vitest), tên chứa mã TC; không sửa test cũ, không sửa code sản phẩm. Sau đó chạy lại: ./scripts/run-all-tests.sh $RUN-after  — mọi Kết quả thực tế/Trạng thái lấy từ lần chạy này.
4. THỰC THI THỦ CÔNG — $UI_RULE
5. BÁO LỖI — mỗi ca Fail (kể cả test mới viết bị Fail vì code sai): tìm trùng bằng gh issue list --search, nếu chưa có thì gh issue create theo mẫu .github/ISSUE_TEMPLATE/bug_report.md (label bug, severity, priority, các bước tái hiện, mong đợi/thực tế, môi trường = commit + máy); ghi số issue vào ca. Không sửa lỗi ở nhánh này.
6. ĐẦU RA:
   - docs/testing/test-cases.md: đầu file ngày, nhánh, commit, bảng tổng hợp theo chức năng (số ca, theo kỹ thuật, tự động/thủ công, Pass/Fail/Chưa chạy, tỷ lệ Pass); rồi mỗi chức năng một mục gồm (a) điều kiện kiểm thử, (b) thiết kế theo kỹ thuật, (c) bảng test case đủ cột chuẩn §2: Mã TC (TC_<MODULE>_<NNN>, giữ mã cũ nếu file đã có) | Mô tả | Điều kiện kiểm thử | Kỹ thuật | Tiền điều kiện | Dữ liệu kiểm thử | Các bước | Kết quả mong đợi | Tiêu chí Pass/Fail | Kết quả thực tế | Trạng thái | Ưu tiên | Loại | Hình thức | Test tự động | Bug.
   - docs/testing/test-cases.csv: mọi ca, thêm cột Chức năng đầu; UTF-8 có BOM, RFC 4180 (mở bằng Excel tiếng Việt).
   - docs/testing/test-report.md: báo cáo tổng kết theo chuẩn §8 — phạm vi, môi trường, số liệu, độ phủ, danh sách lỗi theo severity/trạng thái, đối chiếu tiêu chí kết thúc, kết luận đạt/chưa đạt, bài học.
   - cập nhật docs/testing/traceability-matrix.md thêm cột Mã TC; lưu docs/testing/runs/$STAMP-after.md.
7. TỰ KIỂM: số liệu tổng hợp khớp chi tiết; mọi tên test ở cột Test tự động có thật trong summary sau khi chạy lại; CSV đọc được bằng python csv; mỗi chức năng có ít nhất: 1 bảng EP/BVA, 1 bảng quyết định hoặc sơ đồ trạng thái, ca phân quyền cho cả 4 vai trò, ca âm và ca biên.
Kết thúc: commit thành vài commit hợp lý qua skill ai-usage-log (nhật ký AI chế độ tối giản, type test, một entry), trailer AI-Log bắt buộc; git push -u origin test/test-cases; tạo PR draft --base $SRC_BRANCH nếu chưa có. In đúng một dòng cuối: DONE=OK hoặc DONE=FAIL kèm lý do."

ATTEMPT=1
while :; do
  LOG="$RUN/claude$([ "$ATTEMPT" -gt 1 ] && echo "-$ATTEMPT").jsonl"
  claude -p "$PROMPT" --permission-mode "${PERMISSION_MODE:-auto}" --disallowedTools "AskUserQuestion" \
    --max-turns "${MAX_TURNS:-400}" --output-format stream-json --verbose > "$LOG" 2> "${LOG%.jsonl}.err"
  RESULT="$(python3 -c 'import json,sys
r=""
for l in open(sys.argv[1],encoding="utf-8"):
    try: d=json.loads(l)
    except Exception: continue
    if d.get("type")=="result": r=d.get("result") or ""
print(r)' "$LOG")"
  echo "$RESULT" | tail -8
  if printf '%s' "$RESULT" | grep -qiE "hit your .*limit|usage limit|limit reached"; then
    [ "$ATTEMPT" -ge 6 ] && { notify "Dừng — hết giới hạn quá 6 lần"; exit 1; }
    WAIT="$(printf '%s' "$RESULT" | python3 -c '
import re,sys,datetime
s=sys.stdin.read(); m=re.search(r"resets\s+(\d{1,2})(?::(\d{2}))?\s*([ap]m)",s,re.I); now=datetime.datetime.now()
if not m: print(3600); sys.exit()
h=int(m.group(1))%12+(12 if m.group(3).lower()=="pm" else 0); t=now.replace(hour=h,minute=int(m.group(2) or 0),second=0,microsecond=0)
t=t if t>now else t+datetime.timedelta(days=1); print(int((t-now).total_seconds())+180)')"
    notify "Hết giới hạn — tự chạy tiếp sau $((WAIT/60)) phút"; sleep "$WAIT"; ATTEMPT=$((ATTEMPT+1)); continue
  fi
  break
done
if printf '%s' "$RESULT" | grep -q "DONE=OK"; then notify "Kiểm thử xong — docs/testing/test-cases.md, test-report.md, PR draft"
else notify "Chưa xong — xem $RUN/claude*.jsonl"; exit 1; fi
