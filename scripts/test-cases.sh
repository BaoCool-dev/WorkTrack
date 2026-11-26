#!/usr/bin/env bash
# MỘT LỆNH: chạy toàn bộ test tự động của mọi tính năng đã có, rồi để Claude Code dựng bảng test case
# cho từng chức năng (docs/testing/test-cases.md + .csv mở được bằng Excel).
#
#   ./scripts/test-cases.sh            # chạy test + dựng bảng (ca thủ công ghi "Chưa chạy")
#   ./scripts/test-cases.sh --ui       # thêm: Claude chạy app, thử các ca giao diện bằng Playwright, ghi kết quả thực tế + ảnh
#   ./scripts/test-cases.sh --no-ai    # chỉ chạy test và in tóm tắt, không gọi Claude
#
# Chạy trên nhánh tính năng mới nhất: các nhánh xếp chồng nên nhánh cuối có đủ mọi tính năng.
# Kết quả thô: .ai-log/test-runs/<thời gian>/ (không commit). Bảng: nhánh docs/test-cases + PR draft.
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
[ "$SRC_BRANCH" = "docs/test-cases" ] && SRC_BRANCH="$(git rev-parse --abbrev-ref '@{-1}' 2>/dev/null || echo feature/kanban)"
STAMP="$(date +%Y%m%d-%H%M)"; RUN=".ai-log/test-runs/$STAMP"; mkdir -p "$RUN"
command -v caffeinate >/dev/null && { caffeinate -dims -w $$ & }
echo "== Chạy test trên nhánh $SRC_BRANCH → $RUN"

# 1. Backend: unit + API
echo "-- go test (unit, API)"
( cd backend && go test -json -count=1 ./... ) > "$RUN/go-unit.json" 2> "$RUN/go-unit.err"

# 2. Backend: tích hợp với PostgreSQL thật (CSDL riêng, không đụng dữ liệu dev)
IT_DB="worktrack_it_testcases"
DEV_URL="postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable"
if command -v psql >/dev/null && psql "$DEV_URL" -Atc "select 1" >/dev/null 2>&1; then
  echo "-- go test -tags=integration (CSDL $IT_DB)"
  psql "$DEV_URL" -Atc "select 1 from pg_database where datname='$IT_DB'" | grep -q 1 \
    || psql "$DEV_URL" -qc "create database $IT_DB owner worktrack" >/dev/null
  ( cd backend && TEST_DATABASE_URL="postgres://worktrack:worktrack@localhost:5432/$IT_DB?sslmode=disable" \
      go test -tags=integration -json -count=1 ./... ) > "$RUN/go-integration.json" 2> "$RUN/go-integration.err"
else
  echo "⚠ Không kết nối được PostgreSQL ở localhost:5432 — bỏ qua test tích hợp (brew services start postgresql@16)."
  : > "$RUN/go-integration.json"
fi

# 3. Frontend
echo "-- vitest"
( cd frontend && npx vitest run --reporter=json --outputFile="$ROOT/$RUN/vitest.json" ) > "$RUN/vitest.log" 2>&1

# 4. Tóm tắt máy đọc được (không dùng AI): mọi test, trạng thái, mã CF trong tên test
python3 - "$RUN" <<'PY'
import json, re, sys, pathlib
run = pathlib.Path(sys.argv[1]); rows = {}
def cf(name): return ",".join(sorted(set(re.findall(r"CF-?\d{2}", name.replace("CF-", "CF")))) ).replace("CF","CF-") or ""
for src in ("go-unit", "go-integration"):
    f = run / f"{src}.json"
    if not f.exists(): continue
    for line in f.read_text(encoding="utf-8", errors="ignore").splitlines():
        try: e = json.loads(line)
        except Exception: continue
        if not e.get("Test") or e.get("Action") not in ("pass", "fail", "skip"): continue
        key = f'{e["Package"]}::{e["Test"]}'
        kind = "Integration" if src == "go-integration" and "integration" in e["Package"] + e["Test"].lower() else ("API" if "httpapi" in e["Package"] else "Unit")
        if src == "go-integration" and key in rows and kind != "Integration": continue
        rows[key] = {"layer": "backend", "kind": kind, "file": e["Package"].split("/backend/")[-1], "test": e["Test"], "status": e["Action"], "cf": cf(e["Test"])}
vf = run / "vitest.json"
if vf.exists() and vf.stat().st_size:
    v = json.loads(vf.read_text(encoding="utf-8"))
    for tr in v.get("testResults", []):
        for a in tr.get("assertionResults", []):
            name = a.get("fullName") or a.get("title", "")
            st = {"passed": "pass", "failed": "fail"}.get(a.get("status"), "skip")
            rows["fe::" + tr["name"] + "::" + name] = {"layer": "frontend", "kind": "Unit/UI", "file": tr["name"].split("/frontend/")[-1], "test": name, "status": st, "cf": cf(name + " " + tr["name"]), "error": (a.get("failureMessages") or [""])[0][:300]}
out = list(rows.values())
(run / "summary.json").write_text(json.dumps(out, ensure_ascii=False, indent=1), encoding="utf-8")
tot = {s: sum(1 for r in out if r["status"] == s) for s in ("pass", "fail", "skip")}
lines = [f"Tổng: {len(out)} test — pass {tot['pass']}, fail {tot['fail']}, skip {tot['skip']}", "", "| Tầng | Loại | Pass | Fail | Skip |", "|---|---|---|---|---|"]
for layer in ("backend", "frontend"):
    for kind in sorted({r["kind"] for r in out if r["layer"] == layer}):
        sel = [r for r in out if r["layer"] == layer and r["kind"] == kind]
        lines.append(f"| {layer} | {kind} | " + " | ".join(str(sum(1 for r in sel if r['status'] == s)) for s in ("pass", "fail", "skip")) + " |")
fails = [r for r in out if r["status"] == "fail"]
if fails:
    lines += ["", "Test lỗi:"] + [f"- {r['file']} :: {r['test']}" for r in fails[:50]]
(run / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
print("\n".join(lines))
PY

[ "$AI" = 0 ] && { notify "Đã chạy xong test — xem $RUN/summary.md"; exit 0; }

# 5. Claude Code dựng bảng test case trên nhánh docs/test-cases
if git show-ref --verify --quiet refs/heads/docs/test-cases; then
  git switch -q docs/test-cases && git merge -q --no-edit "$SRC_BRANCH" || { echo "✗ Không gộp được $SRC_BRANCH vào docs/test-cases"; exit 1; }
else
  git switch -q -c docs/test-cases "$SRC_BRANCH"
fi
mkdir -p docs/testing/runs && cp "$RUN/summary.md" "docs/testing/runs/$STAMP-summary.md"

UI_RULE="Ca giao diện/thủ công: KHÔNG chạy app — ghi Trạng thái 'Chưa chạy', Kết quả thực tế để trống."
[ "$UI" = 1 ] && UI_RULE="Ca giao diện/thủ công: chạy make seed rồi make dev-local ở chế độ nền, dùng Playwright MCP ở 1366x768 thực hiện từng ca theo docs/testing/manual-test-guide.md; ghi Kết quả thực tế đúng như quan sát, Trạng thái Pass/Fail, lưu ảnh docs/testing/evidence/tc-<mã>.png cho ca Fail và cho 1 ca tiêu biểu mỗi chức năng; tắt app khi xong."

PROMPT="Chế độ tự chạy (CLAUDE.md §7c) — không hỏi người dùng. Nhánh: docs/test-cases (mọc từ $SRC_BRANCH). KHÔNG sửa code sản phẩm và KHÔNG sửa test.

Nhiệm vụ: dựng BẢNG TEST CASE cho từng chức năng đã có của WorkTrack, dùng cho Chương 5 báo cáo và minh chứng M27–M32.

Nguồn dữ liệu (đọc hết trước khi viết):
- Kết quả chạy THẬT vừa xong: docs/testing/runs/$STAMP-summary.md và $RUN/summary.json (mọi test, trạng thái, mã CF), log thô trong $RUN/.
- docs/testing/traceability-matrix.md, docs/testing/manual-test-guide.md, docs/gates/*.md, docs/decisions.md, skill worktrack-domain, và code (router, domain, test) để hiểu đúng hành vi.

Đầu ra:
1. docs/testing/test-cases.md:
   - Đầu file: ngày chạy, nhánh, commit; bảng tổng hợp theo chức năng (CF): số ca, tự động/thủ công, Pass/Fail/Chưa chạy; tỷ lệ pass.
   - Mỗi chức năng một mục (CF-01 ... theo thứ tự), bảng cột: STT | Mã TC | Mô tả | Vai trò | Tiền điều kiện | Các bước thực hiện | Dữ liệu kiểm thử | Kết quả mong đợi | Kết quả thực tế | Trạng thái | Loại (Unit/Integration/API/UI) | Phân loại (dương/âm/biên/bảo mật/phân quyền/validate) | Hình thức (Tự động/Thủ công) | Test tự động liên quan.
   - Mã TC dạng TC-CF01-001, ổn định: nếu file đã tồn tại thì GIỮ NGUYÊN mã cũ, chỉ cập nhật kết quả và thêm ca mới ở cuối mục.
   - Ca tự động: gom các test cùng kiểm một hành vi thành 1 ca nghiệp vụ dễ hiểu (không liệt kê từng hàm test như 1 ca); Kết quả thực tế và Trạng thái lấy ĐÚNG từ kết quả chạy thật (Pass/Fail), không suy đoán; liệt kê tên test ở cột cuối.
   - Ca thủ công: lấy từ manual-test-guide (luồng người dùng, giao diện, email Mailpit, realtime 2 trình duyệt). $UI_RULE
   - Mỗi chức năng có đủ ca dương, âm, biên, phân quyền theo từng vai trò, validate, bảo mật liên quan.
   - Cuối file: mục 'Lỗi phát hiện' (test Fail: mã TC, thông điệp lỗi rút gọn, file) và mục 'Chưa kiểm thử được' (vd. docker build, test cần Docker) kèm lý do.
2. docs/testing/test-cases.csv: cùng nội dung các bảng chi tiết, thêm cột Chức năng ở đầu; UTF-8 có BOM, phân cách dấu phẩy, chuẩn RFC 4180 (mở được bằng Excel tiếng Việt).
3. Kiểm tra: số ca và số Pass/Fail trong bảng tổng hợp khớp với chi tiết; CSV đọc được bằng python csv; mọi tên test ở cột cuối có thật trong summary.json.

Kết thúc: commit qua skill ai-usage-log (nhật ký AI chế độ tối giản: một entry, type test) — trailer AI-Log bắt buộc; git push -u origin docs/test-cases; tạo PR draft --base $SRC_BRANCH nếu chưa có PR (đã có thì chỉ push). In đúng một dòng cuối: DONE=OK hoặc DONE=FAIL kèm lý do."

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
if printf '%s' "$RESULT" | grep -q "DONE=OK"; then notify "Bảng test case xong — docs/testing/test-cases.md (.csv), PR draft trên GitHub"
else notify "Chưa xong — xem $RUN/claude*.jsonl"; exit 1; fi
