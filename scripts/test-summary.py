#!/usr/bin/env python3
"""Tóm tắt kết quả chạy test thật (go test -json, vitest --reporter=json, coverage) thành summary.json/summary.md.
Dùng: python3 scripts/test-summary.py <thư mục run>"""
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
cov = run / "coverage.txt"
if cov.exists():
    tail = [l for l in cov.read_text(encoding="utf-8").splitlines() if l.startswith("total:")]
    if tail: lines += ["", "Độ phủ gói cốt lõi (domain + application): " + tail[-1].split()[-1]]
if fails:
    lines += ["", "Test lỗi:"] + [f"- {r['file']} :: {r['test']}" for r in fails[:50]]
(run / "summary.md").write_text("\n".join(lines) + "\n", encoding="utf-8")
print("\n".join(lines))
