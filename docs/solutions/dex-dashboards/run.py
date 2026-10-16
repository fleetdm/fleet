#!/usr/bin/env python3
"""Daily run: collect one Fleet snapshot, then rebuild all three pages.

  run.py --work DIR --history lifecycle=PATH --history software=PATH --history patch=PATH

Writes DIR/out/<page>.html and DIR/out/<page>-history.json. Exits non-zero
without writing pages when the data looks wrong, so nothing bad is published.
"""
import argparse
import json
import os
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
PAGES = ("lifecycle", "software", "patch")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--work", required=True)
    ap.add_argument("--history", action="append", default=[], help="page=path to that page's data/history.json")
    ap.add_argument("--first-run", action="store_true", help="allow missing history (starts a new baseline)")
    args = ap.parse_args()

    hist = dict(h.split("=", 1) for h in args.history)
    for page in PAGES:
        p = hist.get(page)
        if not args.first_run and (not p or not os.path.exists(p) or os.path.getsize(p) == 0):
            sys.exit(f"FATAL: history for {page} is missing ({p}). Re-read it from the artifact; "
                     "pass --first-run only when starting over.")

    os.makedirs(os.path.join(args.work, "out"), exist_ok=True)
    snap = os.path.join(args.work, "snapshot.json")
    subprocess.run([sys.executable, "-I", os.path.join(HERE, "collect.py"), "--out", snap], check=True)
    with open(snap) as f:
        data = json.load(f)
    macs = sum(1 for h in data["hosts"] if h["platform"] == "darwin")
    if macs == 0:
        sys.exit("FATAL: Fleet returned no Macs. Not rebuilding.")

    for page in PAGES:
        out = os.path.join(args.work, "out", page)
        cmd = [sys.executable, "-I", os.path.join(HERE, "build.py"), "--page", page, "--snapshot", snap,
               "--templates", os.path.join(HERE, "templates"), "--out-html", out + ".html",
               "--out-history", out + "-history.json"]
        if hist.get(page) and os.path.exists(hist[page]):
            cmd += ["--history", hist[page]]
        subprocess.run(cmd, check=True)
    print(f"OK macs={macs} generated_at={data['generated_at']}")


if __name__ == "__main__":
    main()
