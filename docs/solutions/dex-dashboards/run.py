#!/usr/bin/env python3
"""One run: collect a Fleet snapshot and the vendors' latest releases, then rebuild every page.

  run.py --work DIR --in DIR [--first-run] [--link lifecycle=URL ... --link overview=URL] [--site DIR]

--in holds each page's last history: IN/<page>/history.json, plus any part
files it lists, or a copy of the last run's WORK/out. For each page, writes WORK/out/<page>/index.html and
WORK/out/<page>/data/ (history.json, and parts when the data is big), then
prints one line per page:

  PUBLISH {"page": ..., "url": ..., "file_path": ..., "files": {...}, "overwrite_unread": [...]}

which is the Artifact publish call for that page, file removals included.
--link gives each page's artifact URL, so the overview links to the dashboards
and each dashboard links back. --site also writes a static copy of every page,
without histories, for hosting on your own domain.

A large fleet can take longer than one 10-minute command. When the time budget
runs out, run.py saves what it collected under WORK and prints CONTINUE; run
the same command again to pick up where it stopped. Exits non-zero without
writing pages when the data looks wrong, so nothing bad is published.
"""
import argparse
import datetime as dt
import json
import os
import shutil
import sys
import time

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import build  # noqa: E402
import collect  # noqa: E402
import vendor  # noqa: E402

PAGES = ("lifecycle", "software", "patch", "ai", "connectivity", "analytics", "sustainability", "policies")
# Room left in the time budget for building and writing pages.
BUILD_RESERVE = 150
# One Artifact publish carries at most 64 MB.
PUBLISH_LIMIT = 60_000_000
# A saved snapshot older than this is from an earlier run, not a resumed one.
RESUME_HOURS = 6


def check_size(page, paths):
    total = 0
    for p in paths:
        size = os.path.getsize(p)
        total += size
        if size > build.FILE_LIMIT:
            sys.exit(f"FATAL: {page} wrote {os.path.basename(p)} at {size} bytes, over the 16 MB artifact file limit.")
    if total > PUBLISH_LIMIT:
        sys.exit(f"FATAL: {page} is {total} bytes in all, over what one publish carries (64 MB).")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--work", required=True)
    ap.add_argument("--in", dest="indir", help="IN/<page>/history.json for each page; leave out with --first-run")
    ap.add_argument("--first-run", action="store_true", help="allow missing history (starts a new baseline)")
    ap.add_argument("--link", action="append", default=[], help="page=URL of that page's artifact")
    ap.add_argument("--site", help="also write a static copy of the pages here, for your own web host")
    ap.add_argument("--budget", type=int, default=540, help="seconds before saving progress and printing CONTINUE")
    args = ap.parse_args()
    args.work = os.path.abspath(args.work)
    started = time.monotonic()
    links = dict(x.split("=", 1) for x in args.link)

    hist = {}
    for page in PAGES:
        p = os.path.join(args.indir, page, "history.json") if args.indir else None
        if p and not os.path.exists(p):
            # A previous run's out folder keeps each page's files under data/.
            p = os.path.join(args.indir, page, "data", "history.json")
        if not args.first_run and (not p or not os.path.exists(p) or os.path.getsize(p) == 0):
            sys.exit(f"FATAL: history for {page} is missing ({p}). Re-read it from the artifact; "
                     "pass --first-run only when starting over.")
        hist[page] = build.load_history(p) if p and os.path.exists(p) else ({}, [], [])

    os.makedirs(args.work, exist_ok=True)
    snap_path = os.path.join(args.work, "snapshot.json")
    snap = None
    if os.path.exists(snap_path):
        with open(snap_path) as f:
            saved = json.load(f)
        age = dt.datetime.now(dt.timezone.utc) - build.parse_ts(saved["generated_at"])
        if age < dt.timedelta(hours=RESUME_HOURS):
            snap = saved
            print(f"Using the snapshot collected at {saved['generated_at']}.")
    if snap is None:
        checkpoint = os.path.join(args.work, "collect")
        snap = collect.collect(collect.from_env(), keep=build.keep_software_row, checkpoint=checkpoint,
                               deadline=started + args.budget - BUILD_RESERVE)
        if snap is None:
            pages = len(os.listdir(os.path.join(checkpoint, "pages")))
            print(f"CONTINUE: collected {pages} pages of hosts so far and saved them under {args.work}. "
                  "Run the same command again to finish.")
            sys.exit(3)
        snap["vendor"] = vendor.fetch()
        collect.write_json(snap_path, snap)
        print(collect.summary(snap))
        print(vendor.summary(snap["vendor"]))
        if time.monotonic() > started + args.budget - BUILD_RESERVE:
            print("CONTINUE: collection finished and is saved. Run the same command again to build the pages.")
            sys.exit(3)

    computers = sum(1 for h in snap["hosts"] if h["platform"] and h["platform"] not in collect.NOT_COMPUTERS)
    if computers == 0:
        sys.exit("FATAL: Fleet returned no macOS, Windows, or Linux hosts. Not rebuilding.")

    templates = os.path.join(HERE, "templates")
    out = os.path.join(args.work, "out")
    shutil.rmtree(out, ignore_errors=True)
    publish, page_data = [], {}
    for page in PAGES:
        h, old_hist_parts, old_page_parts = hist[page]
        data = build.BUILDERS[page](snap, h)
        data["links"] = {"overview": links["overview"]} if links.get("overview") else {}
        page_data[page] = data
        page_dir = os.path.join(out, page)
        html_text, parts = build.render(templates, page, data)
        written = build.write_page(page_dir, html_text, parts)
        written = build.save_history(h, os.path.join(page_dir, "data"), written) + written
        files = {p: {"from": os.path.join(page_dir, p), "contentType": "application/json"} for p in written}
        check_size(page, [os.path.join(page_dir, "index.html")] + [f["from"] for f in files.values()])
        # Parts the last run wrote that this one didn't: remove them from the artifact.
        for p in old_hist_parts + old_page_parts:
            files.setdefault(p, None)
        print(f"{page}: html={len(html_text)}B files={len(written)}")
        publish.append({"page": page, "url": links.get(page), "file_path": os.path.join(page_dir, "index.html"),
                        "files": files, "overwrite_unread": old_page_parts})

    overview = build.build_overview(snap, {page: hist[page][0] for page in PAGES})
    overview["links"] = {p: links.get(p) or f"../{p}/" for p in build.OVERVIEW_FIELDS}
    page_dir = os.path.join(out, "overview")
    html_text, parts = build.render(templates, "overview", overview)
    written = build.write_page(page_dir, html_text, parts)
    check_size("overview", [os.path.join(page_dir, p) for p in ["index.html", *written]])
    publish.append({"page": "overview", "url": links.get("overview"), "file_path": os.path.join(page_dir, "index.html"),
                    "files": {p: {"from": os.path.join(page_dir, p), "contentType": "application/json"} for p in written},
                    "overwrite_unread": []})

    if args.site:
        # Same pages, linked to each other by relative paths, without histories. Claude adds
        # the document head to an artifact; a plain web server needs it in the file.
        head = '<!doctype html>\n<meta charset="utf-8">\n<meta name="viewport" content="width=device-width,initial-scale=1">\n'

        def site_page(out_dir, page, data):
            html_text, parts = build.render(templates, page, data)
            build.write_page(out_dir, head + html_text, parts)

        shutil.rmtree(args.site, ignore_errors=True)
        for page in PAGES:
            page_data[page]["links"] = {"overview": "../index.html"}
            site_page(os.path.join(args.site, page), page, page_data[page])
        # Some hosts don't serve a folder's index.html for "page/", so name the file.
        overview["links"] = {p: f"{p}/index.html" for p in build.OVERVIEW_FIELDS}
        site_page(args.site, "overview", overview)
        for root, _, _ in os.walk(args.site, topdown=False):
            if not os.listdir(root):
                os.rmdir(root)

    for p in publish:
        call = {k: v for k, v in p.items() if k == "page" or v not in (None, [], {})}
        print("PUBLISH " + json.dumps(call, separators=(",", ":")))
    shutil.rmtree(os.path.join(args.work, "collect"), ignore_errors=True)
    os.remove(snap_path)
    print(f"OK computers={computers} generated_at={snap['generated_at']}")


if __name__ == "__main__":
    main()
