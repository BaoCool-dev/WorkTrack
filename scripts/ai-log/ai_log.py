#!/usr/bin/env python3
"""Công cụ Nhật ký sử dụng AI (AI Usage Log) cho WorkTrack.

Chỉ dùng thư viện chuẩn Python 3.9+. Chạy từ gốc repo:

  python3 scripts/ai-log/ai_log.py new --slug log-time --type code --scope "CF-08, UC07"
  python3 scripts/ai-log/ai_log.py prompts [--since-last-ai-commit | --since 2026-11-20T08:00]
  python3 scripts/ai-log/ai_log.py human-diff docs/ai-log/entries/<file>.md
  python3 scripts/ai-log/ai_log.py check
  python3 scripts/ai-log/ai_log.py export [--out dist/AI_USAGE_LOG.md]
  python3 scripts/ai-log/ai_log.py form

Liên kết nhật ký ↔ commit KHÔNG lưu hash trong file nhật ký (tránh vòng lặp amend).
Commit mang trailer `AI-Log: <đường dẫn entry>`; script đọc ngược từ git log.
"""
from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import subprocess
import sys
from collections import OrderedDict, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ENTRIES = ROOT / "docs" / "ai-log" / "entries"
TEMPLATE = ROOT / "docs" / "ai-log" / "entry-template.md"
RAW = ROOT / ".ai-log"  # prompt thô do hook ghi, KHÔNG commit

SECTIONS = [
    "Mục tiêu",
    "Prompt đã dùng",
    "Nội dung AI tạo ra",
    "Phần tôi đã sửa / hoàn thiện",
    "Lỗi / ảo giác của AI",
    "Kiểm chứng",
]
REQUIRED_META = ["date", "tool", "model", "type"]
TYPES = {"code", "test", "docs", "design", "review", "debug", "research"}
PLACEHOLDER = "[NGƯỜI DÙNG ĐIỀN]"
HALLU_ROW = re.compile(r"^\|(?!\s*-)(?!\s*Mô tả)(.+)\|\s*$")


# ---------------------------------------------------------------- tiện ích
def git(*args: str) -> str:
    try:
        return subprocess.run(["git", *args], cwd=ROOT, check=True,
                              capture_output=True, text=True).stdout
    except (subprocess.CalledProcessError, FileNotFoundError):
        return ""


def rel(p: Path) -> str:
    return p.resolve().relative_to(ROOT).as_posix()


def parse_entry(path: Path) -> dict:
    text = path.read_text(encoding="utf-8")
    meta: dict[str, str] = {}
    body = text
    m = re.match(r"^---\n(.*?)\n---\n(.*)$", text, re.S)
    if m:
        for line in m.group(1).splitlines():
            if ":" in line:
                k, v = line.split(":", 1)
                meta[k.strip()] = v.strip()
        body = m.group(2)
    sections: dict[str, str] = OrderedDict()
    cur = None
    for line in body.splitlines():
        h = re.match(r"^##\s+(.+?)\s*$", line)
        if h:
            cur = h.group(1)
            sections[cur] = ""
        elif cur:
            sections[cur] += line + "\n"
    sections = {k: v.strip() for k, v in sections.items()}
    return {"path": path, "rel": rel(path), "meta": meta, "sec": sections, "title": meta.get("title", path.stem)}


def load_entries() -> list[dict]:
    if not ENTRIES.exists():
        return []
    return [parse_entry(p) for p in sorted(ENTRIES.glob("*.md"))]


def strip_comments(s: str) -> str:
    return re.sub(r"<!--.*?-->", "", s, flags=re.S).strip()


def commits_by_entry() -> dict[str, list[dict]]:
    """Đọc trailer AI-Log từ toàn bộ lịch sử git."""
    out = git("log", "--format=%H%x1f%h%x1f%ad%x1f%s%x1f%(trailers:key=AI-Log,valueonly,separator=%x1e)"
              "%x1f%(trailers:key=AI-Assisted,valueonly)%x1f%(trailers:key=Human-Edit,valueonly)%x1d",
              "--date=format:%Y-%m-%d %H:%M")
    res: dict[str, list[dict]] = defaultdict(list)
    for rec in out.split("\x1d"):
        rec = rec.strip("\n")
        if not rec:
            continue
        parts = rec.split("\x1f")
        if len(parts) < 7:
            continue
        full, short, date, subj, logs, assisted, human = parts
        for lp in [x.strip() for x in logs.split("\x1e") if x.strip()]:
            res[lp].append({"hash": full, "short": short, "date": date, "subject": subj,
                            "ai": bool(assisted.strip()), "human": bool(human.strip())})
    return res


def ai_commits_without_log() -> list[str]:
    out = git("log", "--format=%h%x1f%s%x1f%(trailers:key=AI-Assisted,valueonly)%x1f%(trailers:key=AI-Log,valueonly)%x1f%b%x1d")
    bad = []
    for rec in out.split("\x1d"):
        parts = rec.strip("\n").split("\x1f")
        if len(parts) < 5:
            continue
        short, subj, assisted, log, body = parts
        coauthored_ai = re.search(r"Co-Authored-By:.*(Claude|Copilot|Gemini|GPT)", body, re.I)
        if (assisted.strip() or coauthored_ai) and not log.strip():
            bad.append(f"{short} {subj}")
    return bad


# ---------------------------------------------------------------- lệnh
def cmd_new(a) -> int:
    ENTRIES.mkdir(parents=True, exist_ok=True)
    today = dt.date.today().isoformat()
    n = len(list(ENTRIES.glob(f"{today}-*.md"))) + 1
    slug = re.sub(r"[^a-z0-9-]+", "-", a.slug.lower()).strip("-")
    path = ENTRIES / f"{today}-{n:02d}-{slug}.md"
    if a.type not in TYPES:
        print(f"type phải là một trong: {', '.join(sorted(TYPES))}", file=sys.stderr)
        return 2
    tpl = TEMPLATE.read_text(encoding="utf-8")
    tpl = (tpl.replace("{{date}}", today).replace("{{title}}", a.title or slug.replace("-", " "))
              .replace("{{tool}}", a.tool).replace("{{model}}", a.model or "")
              .replace("{{surface}}", a.surface).replace("{{type}}", a.type).replace("{{scope}}", a.scope or ""))
    path.write_text(tpl, encoding="utf-8")
    print(rel(path))
    return 0


def last_ai_commit_time() -> str | None:
    out = git("log", "-1", "--format=%cI", "--grep=^AI-Log:")
    return out.strip() or None


def cmd_prompts(a) -> int:
    since = a.since
    if a.since_last_ai_commit:
        since = last_ai_commit_time()
    files = sorted(RAW.glob("prompts-*.jsonl")) if RAW.exists() else []
    if not files:
        print("(Chưa có prompt nào được ghi. Kiểm tra hook UserPromptSubmit trong .claude/settings.json.)")
        return 0
    since_dt = dt.datetime.fromisoformat(since) if since else None
    if since_dt and since_dt.tzinfo is None:
        since_dt = since_dt.astimezone()
    shown = 0
    for f in files:
        for line in f.read_text(encoding="utf-8").splitlines():
            try:
                r = json.loads(line)
            except json.JSONDecodeError:
                continue
            ts = dt.datetime.fromisoformat(r["ts"])
            if since_dt and ts <= since_dt:
                continue
            model = r.get("model") or "?"
            print(f"- [{ts:%Y-%m-%d %H:%M}] ({model}, phiên {r.get('session', '')[:8]})\n  > "
                  + r["prompt"].strip().replace("\n", "\n  > "))
            shown += 1
    if not shown:
        print("(Không có prompt mới kể từ mốc đã chọn.)")
    return 0


def cmd_human_diff(a) -> int:
    """Chèn bằng chứng khách quan: thay đổi của người dùng sau commit AI cuối cùng của entry."""
    path = Path(a.entry).resolve()
    key = rel(path)
    commits = commits_by_entry().get(key, [])
    ai = [c for c in commits if c["ai"] and not c["human"]]
    human = [c for c in commits if c["human"]]
    if not ai or not human:
        print("Cần ít nhất 1 commit AI (AI-Assisted) và 1 commit Human-Edit trỏ tới entry này.", file=sys.stderr)
        return 1
    base = ai[0]["hash"]  # git log mới nhất trước → ai[0] là commit AI gần nhất
    head = human[0]["hash"]
    stat = git("diff", "--stat", f"{base}..{head}", "--", ".", f":(exclude){key}").strip()
    block = (f"<!-- human-diff:start -->\n**Thay đổi của tôi sau commit AI** "
             f"(`git diff {base[:7]}..{head[:7]}`):\n\n```\n{stat or '(không có thay đổi mã)'}\n```\n"
             f"<!-- human-diff:end -->")
    text = path.read_text(encoding="utf-8")
    if "<!-- human-diff:start -->" in text:
        text = re.sub(r"<!-- human-diff:start -->.*?<!-- human-diff:end -->", block, text, flags=re.S)
    else:
        text = re.sub(r"(## Phần tôi đã sửa / hoàn thiện\n)", r"\1" + block.replace("\\", "\\\\") + "\n\n", text, count=1)
    path.write_text(text, encoding="utf-8")
    print(f"Đã cập nhật {key}")
    return 0


def hallucinations(e: dict) -> list[str]:
    rows = []
    for line in e["sec"].get("Lỗi / ảo giác của AI", "").splitlines():
        m = HALLU_ROW.match(line)
        if m:
            cells = [c.strip() for c in m.group(1).split("|")]
            if any(cells) and not all(c in ("", "Không") for c in cells):
                rows.append(cells)
    return rows


def cmd_check(a) -> int:
    entries = load_entries()
    by = commits_by_entry()
    problems, warnings = [], []
    for e in entries:
        meta, sec = e["meta"], e["sec"]
        for k in REQUIRED_META:
            if not meta.get(k):
                problems.append(f"{e['rel']}: thiếu '{k}' ở phần đầu")
        if meta.get("type") and meta["type"] not in TYPES:
            problems.append(f"{e['rel']}: type '{meta['type']}' không hợp lệ")
        for s in SECTIONS:
            if s not in sec:
                problems.append(f"{e['rel']}: thiếu mục '## {s}'")
            elif not strip_comments(sec[s]):
                (warnings if s == "Lỗi / ảo giác của AI" else problems).append(f"{e['rel']}: mục '{s}' đang trống")
        if PLACEHOLDER in sec.get("Phần tôi đã sửa / hoàn thiện", ""):
            warnings.append(f"{e['rel']}: chưa điền 'Phần tôi đã sửa / hoàn thiện'")
        needs_commit = meta.get("commit", "").lower() != "none"
        if needs_commit and e["rel"] not in by:
            warnings.append(f"{e['rel']}: chưa có commit nào mang 'AI-Log: {e['rel']}' "
                            "(ghi 'commit: none' nếu là việc không có commit, vd. thiết kế, tài liệu ngoài repo)")
    for c in ai_commits_without_log():
        problems.append(f"commit {c}: có AI-Assisted/Co-Authored-By AI nhưng thiếu trailer AI-Log")
    for k in by:
        if not (ROOT / k).exists():
            problems.append(f"trailer AI-Log trỏ tới file không tồn tại: {k}")
    total_h = sum(len(hallucinations(e)) for e in entries)
    print(f"Entries: {len(entries)} · Commit có AI-Log: {sum(len(v) for v in by.values())} · "
          f"Lỗi/ảo giác đã ghi: {total_h}/5 (mục tiêu M16)")
    for w in warnings:
        print("  ⚠ " + w)
    for p in problems:
        print("  ✗ " + p)
    if not problems:
        print("OK — nhật ký hợp lệ." + (" (còn cảnh báo)" if warnings else ""))
    return 1 if problems else 0


def short(s: str, n: int = 600) -> str:
    s = strip_comments(s)
    return s if len(s) <= n else s[:n].rstrip() + " …[rút gọn]"


def group_by_tool(entries: list[dict]) -> dict[tuple, list[dict]]:
    g: dict[tuple, list[dict]] = OrderedDict()
    for e in entries:
        key = (e["meta"].get("tool", "?"), e["meta"].get("model", "?"))
        g.setdefault(key, []).append(e)
    return g


def render_form(entries: list[dict]) -> str:
    """Khối dán vào form nộp bài: mỗi công cụ một thẻ với 5 ô."""
    out = []
    for (tool, model), es in group_by_tool(entries).items():
        prompts = []
        for e in es:
            first = next((l for l in strip_comments(e["sec"].get("Prompt đã dùng", "")).splitlines()
                          if l.strip().startswith(">")), "")
            if first:
                prompts.append(f"[{e['meta'].get('date')}] {first.lstrip('> ').strip()}")
        made = [f"- [{e['meta'].get('date')}] {e['title']}: "
                + short(e['sec'].get('Nội dung AI tạo ra', ''), 160).replace("\n", " ") for e in es]
        fixed = [f"- [{e['meta'].get('date')}] {e['title']}: "
                 + short(re.sub(r"<!-- human-diff:start -->.*?<!-- human-diff:end -->", "",
                                e['sec'].get('Phần tôi đã sửa / hoàn thiện', ''), flags=re.S), 160).replace("\n", " ")
                 for e in es]
        out.append(
            f"### Công cụ AI: {tool}\n**Phiên bản / model:** {model}\n\n"
            f"**Prompt đã dùng** ({len(es)} phiên, tiêu biểu):\n" + "\n".join(f"- {p}" for p in prompts[:8]) +
            "\n\n**Nội dung AI tạo ra:**\n" + "\n".join(made) +
            "\n\n**Phần bạn đã sửa / hoàn thiện:**\n" + "\n".join(fixed) + "\n")
    return "\n---\n\n".join(out)


def cmd_form(a) -> int:
    print(render_form(load_entries()))
    return 0


def cmd_export(a) -> int:
    entries = load_entries()
    by = commits_by_entry()
    total_commits = len([l for l in git("rev-list", "--all").splitlines() if l])
    ai_commits = len({c["hash"] for v in by.values() for c in v if c["ai"]})
    human_commits = len({c["hash"] for v in by.values() for c in v if c["human"]})
    types = defaultdict(int)
    for e in entries:
        types[e["meta"].get("type", "?")] += 1
    halls = [(e, h) for e in entries for h in hallucinations(e)]

    L = ["# Nhật ký sử dụng AI — WorkTrack", "",
         f"_Xuất lúc {dt.datetime.now():%H:%M %d/%m/%Y} bằng `scripts/ai-log/ai_log.py export`. "
         "Mỗi mục đối chiếu được với commit qua trailer `AI-Log`._", "",
         "## 1. Tổng quan", "",
         "| Chỉ số | Giá trị |", "|---|---|",
         f"| Số phiên làm việc với AI được ghi | {len(entries)} |",
         f"| Commit có AI hỗ trợ / tổng commit | {ai_commits} / {total_commits} |",
         f"| Commit người dùng hoàn thiện sau AI (Human-Edit) | {human_commits} |",
         f"| Lỗi / ảo giác AI đã phát hiện và sửa | {len(halls)} |",
         "| Phân loại | " + ", ".join(f"{k}: {v}" for k, v in sorted(types.items())) + " |", "",
         "## 2. Khai báo theo công cụ (dán vào form nộp bài)", "", render_form(entries), "",
         "## 3. Lỗi / ảo giác của AI đã phát hiện", "",
         "| # | Ngày | Phiên | Mô tả | Nguyên nhân | Cách phát hiện | Cách sửa |", "|---|---|---|---|---|---|---|"]
    for i, (e, h) in enumerate(halls, 1):
        cells = (h + ["", "", "", ""])[:4]
        L.append(f"| {i} | {e['meta'].get('date')} | {e['title']} | " + " | ".join(cells) + " |")
    L += ["", "## 4. Quy trình kiểm soát đầu ra AI", "",
          (ROOT / "docs/ai-log/README.md").read_text(encoding="utf-8").split("<!-- policy:start -->")[-1]
          .split("<!-- policy:end -->")[0].strip(), "",
          "## 5. Nhật ký chi tiết theo thời gian", ""]
    for e in entries:
        m = e["meta"]
        L += [f"### {m.get('date')} — {e['title']}", "",
              f"- **Công cụ:** {m.get('tool')} · **Model:** {m.get('model')} · **Môi trường:** {m.get('surface', '')}",
              f"- **Loại:** {m.get('type')} · **Phạm vi:** {m.get('scope', '')}"]
        cs = by.get(e["rel"], [])
        if cs:
            L.append("- **Commit:** " + ", ".join(
                f"`{c['short']}`{' (người dùng sửa)' if c['human'] else ''}" for c in reversed(cs)))
        elif m.get("commit", "").lower() == "none":
            L.append("- **Commit:** không có (việc ngoài mã nguồn)")
        L.append("")
        for s in SECTIONS:
            content = strip_comments(e["sec"].get(s, "")).replace("<!-- human-diff:start -->", "")
            if content:
                L += [f"**{s}**", "", content, ""]
    out = Path(a.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text("\n".join(L), encoding="utf-8")
    print(f"Đã xuất {out} ({out.stat().st_size // 1024} KB)")
    if a.docx:
        try:
            subprocess.run(["pandoc", str(out), "-o", str(out.with_suffix('.docx'))], check=True)
            print(f"Đã xuất {out.with_suffix('.docx')}")
        except (FileNotFoundError, subprocess.CalledProcessError):
            print("Không chạy được pandoc (brew install pandoc). Bỏ qua bản .docx.", file=sys.stderr)
    return 0


def main() -> int:
    p = argparse.ArgumentParser(description="AI Usage Log — WorkTrack")
    sp = p.add_subparsers(dest="cmd", required=True)
    n = sp.add_parser("new", help="tạo entry mới từ template")
    n.add_argument("--slug", required=True)
    n.add_argument("--title")
    n.add_argument("--type", default="code")
    n.add_argument("--scope", default="")
    n.add_argument("--tool", default="Claude Code")
    n.add_argument("--model", default="")
    n.add_argument("--surface", default="terminal")
    n.set_defaults(fn=cmd_new)
    pr = sp.add_parser("prompts", help="in prompt đã được hook ghi lại")
    pr.add_argument("--since")
    pr.add_argument("--since-last-ai-commit", action="store_true")
    pr.set_defaults(fn=cmd_prompts)
    hd = sp.add_parser("human-diff", help="chèn diff thay đổi của người dùng sau commit AI")
    hd.add_argument("entry")
    hd.set_defaults(fn=cmd_human_diff)
    sp.add_parser("check", help="kiểm tra nhật ký và liên kết commit").set_defaults(fn=cmd_check)
    ex = sp.add_parser("export", help="xuất file nộp bài")
    ex.add_argument("--out", default="dist/AI_USAGE_LOG.md")
    ex.add_argument("--docx", action="store_true")
    ex.set_defaults(fn=cmd_export)
    sp.add_parser("form", help="in khối dán vào form nộp bài").set_defaults(fn=cmd_form)
    a = p.parse_args()
    return a.fn(a)


if __name__ == "__main__":
    sys.exit(main())
