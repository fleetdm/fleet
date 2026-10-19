#!/usr/bin/env python3
"""Package the dashboard code into one file for the daily routine.

  bundle.py --out DIR

Writes:
  DIR/dex-dashboards.json         publish on the Hardware lifecycle artifact as src/dex-dashboards.json
  DIR/placeholder-<page>.html     first version of each artifact, shown until the routine's first run
and prints the bundle's sha256, which goes in the routine prompt. The routine
refuses to run a bundle whose hash doesn't match.
"""
import argparse
import hashlib
import json
import os

HERE = os.path.dirname(os.path.abspath(__file__))
FILES = [
    "run.py", "collect.py", "build.py",
    "templates/base.css", "templates/base.js",
    "templates/lifecycle.html", "templates/software.html", "templates/patch.html", "templates/ai.html",
    "templates/connectivity.html", "templates/analytics.html",
]
TITLES = {"lifecycle": "Hardware lifecycle", "software": "Software usage", "patch": "Patch velocity", "ai": "AI governance",
          "connectivity": "Connectivity", "analytics": "Analytics"}
PLACEHOLDER = """<title>{title}</title>
<meta name="claude:viewport-fit-sides" content="cover">
<style>
:root {{ --bg: #f4f5f8; --ink: #192147; --muted: #6b7090; }}
@media (prefers-color-scheme: dark) {{ :root:not([data-theme="light"]) {{ --bg: #181a1f; --ink: #e2e4ea; --muted: #8f93a6; color-scheme: dark; }} }}
:root[data-theme="dark"] {{ --bg: #181a1f; --ink: #e2e4ea; --muted: #8f93a6; color-scheme: dark; }}
body {{ background: var(--bg); color: var(--ink); font: 15px/1.5 system-ui, -apple-system, "Segoe UI", sans-serif; }}
main {{ max-width: 640px; margin: 0 auto; padding-block: 64px; padding-left: max(16px, env(safe-area-inset-left, 0px)); padding-right: max(16px, env(safe-area-inset-right, 0px)); }}
p {{ color: var(--muted); }}
</style>
<main>
  <h1>{title}</h1>
  <p>This dashboard fills in after the routine's first run. Each run rebuilds it from your Fleet data.</p>
</main>
"""


def bundle_hash(files):
    return hashlib.sha256(json.dumps(files, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    args = ap.parse_args()
    os.makedirs(args.out, exist_ok=True)
    files = {}
    for rel in FILES:
        with open(os.path.join(HERE, rel)) as f:
            files[rel] = f.read()
    with open(os.path.join(args.out, "dex-dashboards.json"), "w") as f:
        json.dump({"name": "dex-dashboards", "files": files}, f)
    for page, title in TITLES.items():
        with open(os.path.join(args.out, f"placeholder-{page}.html"), "w") as f:
            f.write(PLACEHOLDER.format(title=title))
    print(bundle_hash(files))


if __name__ == "__main__":
    main()
