#!/usr/bin/env bash
# Chạy cả ngày không cần người trông — gõ một lần trước khi đi:
#   1. Gộp từng nhánh đã có vào release/v1 (test, rà cấu hình, dọn code rác, báo cáo, đẩy nhánh sao lưu từng bước)
#   2. Làm tiếp các chức năng còn lại (giai đoạn 2) trên nền release/v1, mỗi chức năng một nhánh + gate + PR draft
#   3. Gộp lần lượt từng chức năng mới vào release/v1 theo cùng quy trình
# KHÔNG đụng main. Hết giới hạn sử dụng thì tự chờ rồi chạy tiếp. Lỗi thì dừng và báo.
set -uo pipefail
cd "$(cd "$(dirname "$0")/.." && pwd)"
LOGDIR=".ai-log/day-run"; mkdir -p "$LOGDIR"; LOG="$LOGDIR/$(date +%Y%m%d-%H%M).log"
notify() { command -v osascript >/dev/null && osascript -e "display notification \"$1\" with title \"WorkTrack day-run\"" >/dev/null 2>&1; echo "$(date +%H:%M) $1" | tee -a "$LOG"; }
command -v caffeinate >/dev/null && { caffeinate -dims -w $$ & }

notify "Bước 1/3: gộp các nhánh đã có vào release/v1"
./scripts/integrate.sh 2>&1 | tee -a "$LOG"; [ "${PIPESTATUS[0]}" = 0 ] || { notify "Dừng ở bước 1 (gộp) — xem docs/integration/ và $LOG"; exit 1; }

notify "Bước 2/3: làm các chức năng còn lại trên nền release/v1"
git switch -q release/v1
./scripts/auto-run.sh --phase 2 --from feature/subtasks --base release/v1 2>&1 | tee -a "$LOG"
[ "${PIPESTATUS[0]}" = 0 ] || notify "Giai đoạn 2 dừng giữa chừng — các chức năng đã xong vẫn được gộp ở bước 3"

notify "Bước 3/3: gộp từng chức năng mới vào release/v1"
git switch -q release/v1 2>/dev/null
./scripts/integrate.sh 2>&1 | tee -a "$LOG"; [ "${PIPESTATUS[0]}" = 0 ] || { notify "Dừng ở bước 3 — xem docs/integration/ và $LOG"; exit 1; }

notify "Xong cả ngày. Xem: git log --oneline --graph release/v1 và docs/integration/"
