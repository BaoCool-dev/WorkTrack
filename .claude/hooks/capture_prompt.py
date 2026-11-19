#!/usr/bin/env python3
"""UserPromptSubmit hook: ghi nguyên văn prompt vào .ai-log/prompts-YYYY-MM.jsonl
để Nhật ký sử dụng AI dùng prompt THẬT, không phải prompt dựng lại.
Không bao giờ chặn phiên: mọi lỗi đều bỏ qua, luôn exit 0.
.ai-log/ nằm trong .gitignore — dữ liệu thô chỉ ở máy bạn."""
import datetime
import json
import os
import re
import sys
from pathlib import Path

# Che các chuỗi trông giống secret trước khi lưu (rubric M18)
PATTERNS = [
    r"AIza[0-9A-Za-z_\-]{30,}",                                  # Google API key
    r"sk-[A-Za-z0-9_\-]{20,}",                                   # khóa kiểu sk-
    r"gh[pousr]_[A-Za-z0-9]{30,}",                               # GitHub token
    r"eyJ[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+\.[A-Za-z0-9_\-]+",     # JWT
    r"(?i)(password|passwd|mật khẩu)\s*[:=]\s*\S+",
    r"postgres(ql)?://\S+",
]


def model_from_transcript(tp):
    try:
        if tp and Path(tp).exists():
            for line in reversed(Path(tp).read_text(encoding="utf-8").splitlines()[-400:]):
                try:
                    m = (json.loads(line).get("message") or {}).get("model")
                except (json.JSONDecodeError, AttributeError):
                    continue
                if m:
                    return m
    except OSError:
        pass
    return ""


def main():
    data = json.load(sys.stdin)
    prompt = data.get("prompt") or ""
    if not prompt.strip():
        return
    for p in PATTERNS:
        prompt = re.sub(p, "[ĐÃ CHE]", prompt)
    root = Path(os.environ.get("CLAUDE_PROJECT_DIR") or data.get("cwd") or ".")
    out_dir = root / ".ai-log"
    out_dir.mkdir(exist_ok=True)
    now = datetime.datetime.now().astimezone()
    rec = {"ts": now.isoformat(timespec="seconds"), "session": data.get("session_id", ""),
           "model": model_from_transcript(data.get("transcript_path")), "prompt": prompt}
    with open(out_dir / f"prompts-{now:%Y-%m}.jsonl", "a", encoding="utf-8") as f:
        f.write(json.dumps(rec, ensure_ascii=False) + "\n")


if __name__ == "__main__":
    try:
        main()
    except Exception:
        pass
    sys.exit(0)
