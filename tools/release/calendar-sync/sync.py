#!/usr/bin/env python3
"""Sync the Fleet releases Google Calendar with GitHub milestone due dates.

Reads open milestones from fleetdm/fleet, finds the corresponding events on the
"Fleet releases" calendar ("Release day", "Release candidate", "Develop"), and
proposes title renames and date adjustments so they match the milestones.

Usage:
    python sync.py              # dry-run, prints proposed changes
    python sync.py --apply      # actually apply changes

Auth (first match wins):
    --oauth                       # force interactive Google login (local testing)
    GCAL_SERVICE_ACCOUNT_JSON     # raw JSON contents of a Google service account key
    GOOGLE_APPLICATION_CREDENTIALS  # OR path to a service account JSON file
    (none of the above)           # falls back to interactive OAuth user login

    GCAL_OAUTH_CLIENT_SECRET      # path to OAuth client secret JSON (default: ./client_secret.json)
    GITHUB_TOKEN / GH_TOKEN       # optional; raises rate limit when calling GitHub
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
import re
import sys
from dataclasses import dataclass, field
from typing import Optional

import requests
from google.oauth2 import service_account
from googleapiclient.discovery import build


CALENDAR_ID = "c_v7943deqn1uns488a65v2d94bs@group.calendar.google.com"
GITHUB_REPO = "fleetdm/fleet"
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")

# A milestone is "out-of-band" if its gap to the previous milestone is shorter
# than this. Normal cadence is 21 days; out-of-band patches land mid-sprint.
OUT_OF_BAND_GAP_DAYS = 14

# Default duration for the short RC event created for an out-of-band release.
PATCH_RC_DURATION_DAYS = 5

# Time-zone for date computations (the calendar's display zone).
TZ_NAME = "America/Chicago"

# Matching tolerances: a calendar event date must be within this many days of
# a milestone's due date to be considered a match.
RELEASE_DAY_MATCH_TOLERANCE_DAYS = 5
RC_END_MATCH_TOLERANCE_DAYS = 5
DEVELOP_END_TO_DUE_TARGET_DAYS = 14   # Develop_end + 14d ~= milestone due
DEVELOP_END_TO_DUE_TOLERANCE_DAYS = 5
DEVELOP_SPAN_DAYS = 18                 # Monday start -> Friday display-end (3-week sprint)

RELEASE_DAY_RE = re.compile(r"^Release day: minor release - (\d+\.\d+\.\d+)\s*$")
RC_RE = re.compile(r"^Release candidate \(next release - (\d+\.\d+\.\d+)\)\s*$")
DEVELOP_RE = re.compile(r"^Develop \(next release - (\d+\.\d+\.\d+)\)\s*$")

SCOPES = ["https://www.googleapis.com/auth/calendar"]

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
DEFAULT_CLIENT_SECRET = os.path.join(SCRIPT_DIR, "client_secret.json")
DEFAULT_TOKEN_PATH = os.path.join(SCRIPT_DIR, "token.json")


@dataclass
class Milestone:
    number: int
    title: str   # "4.88.0"
    due: dt.date
    out_of_band: bool = False


@dataclass
class CalEvent:
    id: str
    summary: str
    start: dt.date
    end: Optional[dt.date]  # exclusive for all-day events
    raw: dict

    @property
    def is_all_day(self) -> bool:
        return "date" in self.raw.get("start", {})


@dataclass
class Action:
    kind: str   # "rename", "move", "create", "noop"
    description: str
    event: Optional[CalEvent] = None
    new_summary: Optional[str] = None
    new_start: Optional[dt.date] = None
    new_end: Optional[dt.date] = None
    category: Optional[str] = None  # "release_day" | "rc" | "develop" (for create)


def fetch_milestones() -> list[Milestone]:
    token = os.environ.get("GITHUB_TOKEN") or os.environ.get("GH_TOKEN")
    headers = {"Accept": "application/vnd.github+json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    url = f"https://api.github.com/repos/{GITHUB_REPO}/milestones"
    out: list[Milestone] = []
    page = 1
    while True:
        r = requests.get(
            url,
            headers=headers,
            params={"state": "open", "per_page": 100, "page": page},
            timeout=30,
        )
        r.raise_for_status()
        data = r.json()
        if not data:
            break
        for m in data:
            if not m.get("due_on") or not VERSION_RE.match(m["title"]):
                continue
            due = dt.datetime.fromisoformat(m["due_on"].replace("Z", "+00:00")).date()
            out.append(Milestone(number=m["number"], title=m["title"], due=due))
        if len(data) < 100:
            break
        page += 1
    out.sort(key=lambda x: x.due)
    # A milestone is "out-of-band" only if it has SHORT gaps to BOTH neighbors.
    # That avoids flagging the regular minor that immediately follows an OOB
    # insertion (which has a short prev-gap but a normal next-gap).
    for i, m in enumerate(out):
        prev_gap = (m.due - out[i - 1].due).days if i > 0 else None
        next_gap = (out[i + 1].due - m.due).days if i + 1 < len(out) else None
        if (
            prev_gap is not None
            and prev_gap < OUT_OF_BAND_GAP_DAYS
            and next_gap is not None
            and next_gap < OUT_OF_BAND_GAP_DAYS
        ):
            m.out_of_band = True
    return out


def oauth_credentials(client_secret: str, token_path: str):
    """Interactive user login for local use. Caches a refreshable token so the
    browser prompt only appears the first time (or after the token is revoked)."""
    from google.auth.transport.requests import Request
    from google.oauth2.credentials import Credentials
    from google_auth_oauthlib.flow import InstalledAppFlow

    creds = None
    if os.path.exists(token_path):
        creds = Credentials.from_authorized_user_file(token_path, SCOPES)
    if not creds or not creds.valid:
        if creds and creds.expired and creds.refresh_token:
            creds.refresh(Request())
        else:
            if not os.path.exists(client_secret):
                sys.exit(
                    f"ERROR: OAuth client secret not found at {client_secret}.\n"
                    "Create an OAuth client ID (type: Desktop app) in Google Cloud "
                    "Console → APIs & Services → Credentials, download the JSON, and "
                    "either save it there or set GCAL_OAUTH_CLIENT_SECRET to its path."
                )
            flow = InstalledAppFlow.from_client_secrets_file(client_secret, SCOPES)
            creds = flow.run_local_server(port=0)
        with open(token_path, "w", encoding="utf-8") as f:
            f.write(creds.to_json())
        print(f"Saved OAuth token to {token_path}")
    return creds


def gcal_service(use_oauth: bool = False, client_secret: str = DEFAULT_CLIENT_SECRET,
                 token_path: str = DEFAULT_TOKEN_PATH):
    creds_json = os.environ.get("GCAL_SERVICE_ACCOUNT_JSON")
    creds_path = os.environ.get("GOOGLE_APPLICATION_CREDENTIALS")
    if use_oauth or (not creds_json and not creds_path):
        creds = oauth_credentials(client_secret, token_path)
    elif creds_json:
        info = json.loads(creds_json)
        creds = service_account.Credentials.from_service_account_info(info, scopes=SCOPES)
    else:
        creds = service_account.Credentials.from_service_account_file(creds_path, scopes=SCOPES)
    return build("calendar", "v3", credentials=creds, cache_discovery=False)


def fetch_events(service, start: dt.date, end: dt.date) -> list[CalEvent]:
    out: list[CalEvent] = []
    page_token = None
    while True:
        resp = (
            service.events()
            .list(
                calendarId=CALENDAR_ID,
                timeMin=dt.datetime.combine(start, dt.time.min).isoformat() + "Z",
                timeMax=dt.datetime.combine(end, dt.time.min).isoformat() + "Z",
                singleEvents=True,
                orderBy="startTime",
                maxResults=250,
                pageToken=page_token,
            )
            .execute()
        )
        for e in resp.get("items", []):
            s = e.get("start", {})
            en = e.get("end", {})
            if "date" in s:
                start_date = dt.date.fromisoformat(s["date"])
                end_date = dt.date.fromisoformat(en["date"]) if "date" in en else None
            else:
                start_date = dt.datetime.fromisoformat(
                    s["dateTime"].replace("Z", "+00:00")
                ).date()
                end_date = (
                    dt.datetime.fromisoformat(en["dateTime"].replace("Z", "+00:00")).date()
                    if "dateTime" in en
                    else None
                )
            out.append(
                CalEvent(
                    id=e["id"],
                    summary=e.get("summary", ""),
                    start=start_date,
                    end=end_date,
                    raw=e,
                )
            )
        page_token = resp.get("nextPageToken")
        if not page_token:
            break
    return out


def categorize(event: CalEvent) -> tuple[Optional[str], Optional[str]]:
    s = event.summary or ""
    for cat, regex in (
        ("release_day", RELEASE_DAY_RE),
        ("rc", RC_RE),
        ("develop", DEVELOP_RE),
    ):
        m = regex.match(s)
        if m:
            return cat, m.group(1)
    return None, None


def closest_milestone_by_due(
    milestones: list[Milestone], target_date: dt.date, tolerance_days: int
) -> Optional[Milestone]:
    best, best_gap = None, None
    for m in milestones:
        gap = abs((m.due - target_date).days)
        if gap <= tolerance_days and (best_gap is None or gap < best_gap):
            best, best_gap = m, gap
    return best


def match_develop_to_milestone(
    milestones: list[Milestone], develop_end: dt.date
) -> Optional[Milestone]:
    """Develop event ends ~2 weeks before its corresponding release.
    Skip out-of-band milestones (patches don't have a Develop sprint)."""
    best, best_gap = None, None
    for m in milestones:
        if m.out_of_band:
            continue
        target_due = develop_end + dt.timedelta(days=DEVELOP_END_TO_DUE_TARGET_DAYS)
        gap = abs((m.due - target_due).days)
        if gap <= DEVELOP_END_TO_DUE_TOLERANCE_DAYS and (best_gap is None or gap < best_gap):
            best, best_gap = m, gap
    return best


def desired_release_day(m: Milestone) -> tuple[dt.date, dt.date]:
    """All-day event: start=due, end=due+1 (exclusive)."""
    return m.due, m.due + dt.timedelta(days=1)


def desired_rc(m: Milestone, current_start: Optional[dt.date]) -> tuple[dt.date, dt.date]:
    """RC end = due+1 (exclusive, so display ends on release day).
    For out-of-band milestones with no existing start, default to a short window
    of PATCH_RC_DURATION_DAYS ending on the release day."""
    end_excl = m.due + dt.timedelta(days=1)
    if current_start is not None:
        return current_start, end_excl
    start = m.due - dt.timedelta(days=PATCH_RC_DURATION_DAYS - 1)
    return start, end_excl


def desired_develop(
    current_start: Optional[dt.date], m: Milestone
) -> tuple[dt.date, dt.date]:
    """Develop display-end = milestone due - 14 days. For existing events the start is
    preserved; for a new event (current_start is None) the start is the Monday three
    weeks before the Friday display-end (a 3-week sprint)."""
    end_excl = m.due - dt.timedelta(days=DEVELOP_END_TO_DUE_TARGET_DAYS - 1)
    if current_start is None:
        display_end = end_excl - dt.timedelta(days=1)
        start = display_end - dt.timedelta(days=DEVELOP_SPAN_DAYS)
        return start, end_excl
    return current_start, end_excl


def fmt_date(d: Optional[dt.date]) -> str:
    return d.isoformat() if d else "?"


def plan_actions(
    milestones: list[Milestone], events: list[CalEvent], today: dt.date
) -> list[Action]:
    actions: list[Action] = []
    matched_milestone_ids: dict[str, set[int]] = {
        "release_day": set(),
        "rc": set(),
        "develop": set(),
    }

    # 1) Match existing events to milestones, propose rename / move actions.
    for ev in events:
        cat, ver = categorize(ev)
        if cat is None:
            continue

        if cat == "release_day":
            m = closest_milestone_by_due(milestones, ev.start, RELEASE_DAY_MATCH_TOLERANCE_DAYS)
            if not m:
                actions.append(
                    Action(
                        kind="noop",
                        description=f"  ! Release day event '{ev.summary}' on {ev.start} has no matching milestone (within {RELEASE_DAY_MATCH_TOLERANCE_DAYS}d) — skipped",
                        event=ev,
                    )
                )
                continue
            matched_milestone_ids["release_day"].add(m.number)
            new_summary = f"Release day: minor release - {m.title}"
            new_start, new_end = desired_release_day(m)
            changes = []
            if ev.summary != new_summary:
                changes.append(f"title '{ev.summary}' -> '{new_summary}'")
            if ev.start != new_start or ev.end != new_end:
                changes.append(
                    f"dates {fmt_date(ev.start)}..{fmt_date(ev.end)} -> {fmt_date(new_start)}..{fmt_date(new_end)}"
                )
            if changes:
                actions.append(
                    Action(
                        kind="rename" if "title" in changes[0] else "move",
                        description=f"  Release day {m.title}: {'; '.join(changes)}",
                        event=ev,
                        new_summary=new_summary,
                        new_start=new_start,
                        new_end=new_end,
                    )
                )

        elif cat == "rc":
            # Match by event END date (exclusive end - 1 = display end = release day = milestone due).
            target = (ev.end - dt.timedelta(days=1)) if ev.end else ev.start
            m = closest_milestone_by_due(milestones, target, RC_END_MATCH_TOLERANCE_DAYS)
            if not m:
                actions.append(
                    Action(
                        kind="noop",
                        description=f"  ! RC event '{ev.summary}' ending {target} has no matching milestone — skipped",
                        event=ev,
                    )
                )
                continue
            matched_milestone_ids["rc"].add(m.number)
            new_summary = f"Release candidate (next release - {m.title})"
            new_start, new_end = desired_rc(m, ev.start)
            changes = []
            if ev.summary != new_summary:
                changes.append(f"title '{ev.summary}' -> '{new_summary}'")
            if ev.end != new_end:
                changes.append(f"end {fmt_date(ev.end)} -> {fmt_date(new_end)}")
            if ev.start != new_start:
                changes.append(f"start {fmt_date(ev.start)} -> {fmt_date(new_start)}")
            if changes:
                actions.append(
                    Action(
                        kind="rename" if "title" in changes[0] else "move",
                        description=f"  RC {m.title}: {'; '.join(changes)}",
                        event=ev,
                        new_summary=new_summary,
                        new_start=new_start,
                        new_end=new_end,
                    )
                )

        elif cat == "develop":
            end_for_match = (ev.end - dt.timedelta(days=1)) if ev.end else ev.start
            m = match_develop_to_milestone(milestones, end_for_match)
            if not m:
                actions.append(
                    Action(
                        kind="noop",
                        description=f"  ! Develop event '{ev.summary}' ending {end_for_match} has no matching minor milestone — skipped",
                        event=ev,
                    )
                )
                continue
            matched_milestone_ids["develop"].add(m.number)
            new_summary = f"Develop (next release - {m.title})"
            new_start, new_end = desired_develop(ev.start, m)
            changes = []
            if ev.summary != new_summary:
                changes.append(f"title '{ev.summary}' -> '{new_summary}'")
            if ev.end != new_end:
                changes.append(f"end {fmt_date(ev.end)} -> {fmt_date(new_end)}")
            if changes:
                actions.append(
                    Action(
                        kind="rename" if "title" in changes[0] else "move",
                        description=f"  Develop {m.title}: {'; '.join(changes)}",
                        event=ev,
                        new_summary=new_summary,
                        new_start=new_start,
                        new_end=new_end,
                    )
                )

    # 2) Find milestones with no matching events and propose creates.
    for m in milestones:
        if m.number not in matched_milestone_ids["release_day"]:
            new_summary = f"Release day: minor release - {m.title}"
            new_start, new_end = desired_release_day(m)
            actions.append(
                Action(
                    kind="create",
                    description=f"  + CREATE Release day {m.title} on {fmt_date(new_start)}",
                    new_summary=new_summary,
                    new_start=new_start,
                    new_end=new_end,
                    category="release_day",
                )
            )
        if m.number not in matched_milestone_ids["rc"]:
            new_summary = f"Release candidate (next release - {m.title})"
            new_start, new_end = desired_rc(m, None)
            actions.append(
                Action(
                    kind="create",
                    description=f"  + CREATE RC {m.title} {fmt_date(new_start)}..{fmt_date(new_end)}"
                    + (" (out-of-band, short window)" if m.out_of_band else ""),
                    new_summary=new_summary,
                    new_start=new_start,
                    new_end=new_end,
                    category="rc",
                )
            )
        if not m.out_of_band and m.number not in matched_milestone_ids["develop"]:
            new_summary = f"Develop (next release - {m.title})"
            new_start, new_end = desired_develop(None, m)
            display_end = new_end - dt.timedelta(days=1)
            if display_end < today:
                # The sprint already ended; don't create a stale past event.
                actions.append(
                    Action(
                        kind="noop",
                        description=(
                            f"  ! Skipping Develop create for {m.title}: "
                            f"sprint ended {display_end} (past)"
                        ),
                    )
                )
            else:
                actions.append(
                    Action(
                        kind="create",
                        description=f"  + CREATE Develop {m.title} {fmt_date(new_start)}..{fmt_date(new_end)}",
                        new_summary=new_summary,
                        new_start=new_start,
                        new_end=new_end,
                        category="develop",
                    )
                )

    return actions


def apply_action(service, action: Action) -> None:
    if action.kind in ("rename", "move"):
        body = {}
        if action.new_summary is not None:
            body["summary"] = action.new_summary
        if action.new_start is not None and action.event and action.event.is_all_day:
            body["start"] = {"date": action.new_start.isoformat()}
        if action.new_end is not None and action.event and action.event.is_all_day:
            body["end"] = {"date": action.new_end.isoformat()}
        service.events().patch(
            calendarId=CALENDAR_ID,
            eventId=action.event.id,
            body=body,
        ).execute()
    elif action.kind == "create":
        body = {
            "summary": action.new_summary,
            "start": {"date": action.new_start.isoformat()},
            "end": {"date": action.new_end.isoformat()},
            "transparency": "transparent",
        }
        service.events().insert(calendarId=CALENDAR_ID, body=body).execute()


def render_plan(
    milestones: list[Milestone], events: list[CalEvent], actions: list[Action]
) -> str:
    lines: list[str] = []
    lines.append(f"# Fleet release calendar sync — {dt.datetime.now().strftime('%Y-%m-%d %H:%M')}")
    lines.append("")
    lines.append(f"Calendar: {CALENDAR_ID}")
    lines.append(f"Repo:     {GITHUB_REPO}")
    lines.append("")
    lines.append("## Milestones")
    for m in milestones:
        tag = " (OUT-OF-BAND)" if m.out_of_band else ""
        lines.append(f"  {m.title}  due {m.due}{tag}")
    lines.append("")
    lines.append("## Events scanned")
    for ev in events:
        cat, ver = categorize(ev)
        if not cat:
            continue
        lines.append(
            f"  [{cat:11s}] '{ev.summary}'  start={ev.start}  end={ev.end}"
        )
    lines.append("")
    lines.append("## Proposed actions")
    creates = [a for a in actions if a.kind == "create"]
    renames = [a for a in actions if a.kind == "rename"]
    moves = [a for a in actions if a.kind == "move"]
    noops = [a for a in actions if a.kind == "noop"]
    if not (creates or renames or moves):
        lines.append("  (no changes — calendar already matches milestones)")
    for a in renames + moves + creates:
        lines.append(a.description)
    if noops:
        lines.append("")
        lines.append("## Warnings / skipped")
        for a in noops:
            lines.append(a.description)
    lines.append("")
    lines.append(
        f"Total: {len(renames)} rename, {len(moves)} move, {len(creates)} create, {len(noops)} warning"
    )
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--apply", action="store_true", help="apply changes (default: dry-run)")
    parser.add_argument(
        "--window-days",
        type=int,
        default=200,
        help="how far into the future to scan calendar events (default: 200)",
    )
    parser.add_argument(
        "--summary-file",
        type=str,
        default=None,
        help="if set, also write the rendered plan to this path (e.g., $GITHUB_STEP_SUMMARY)",
    )
    parser.add_argument(
        "--oauth",
        action="store_true",
        help="force interactive Google login instead of a service account (local testing)",
    )
    parser.add_argument(
        "--client-secret",
        type=str,
        default=os.environ.get("GCAL_OAUTH_CLIENT_SECRET", DEFAULT_CLIENT_SECRET),
        help="path to OAuth client secret JSON (default: ./client_secret.json)",
    )
    parser.add_argument(
        "--token",
        type=str,
        default=DEFAULT_TOKEN_PATH,
        help="path to cache the OAuth user token (default: ./token.json)",
    )
    args = parser.parse_args()

    milestones = fetch_milestones()
    if not milestones:
        print("No open milestones with version-like titles and due dates found.")
        return 1

    today = dt.date.today()
    service = gcal_service(
        use_oauth=args.oauth,
        client_secret=args.client_secret,
        token_path=args.token,
    )
    events = fetch_events(
        service,
        start=today - dt.timedelta(days=30),
        end=today + dt.timedelta(days=args.window_days),
    )

    actions = plan_actions(milestones, events, today)
    plan_text = render_plan(milestones, events, actions)
    print(plan_text)

    if args.summary_file:
        with open(args.summary_file, "a", encoding="utf-8") as f:
            f.write(plan_text + "\n")

    if not args.apply:
        print()
        print("Dry-run only. Re-run with --apply to make these changes.")
        return 0

    mutating = [a for a in actions if a.kind in ("rename", "move", "create")]
    if not mutating:
        print("Nothing to apply.")
        return 0

    print(f"\nApplying {len(mutating)} change(s)...")
    for a in mutating:
        try:
            apply_action(service, a)
            print(f"  ok: {a.description.strip()}")
        except Exception as e:  # noqa: BLE001
            print(f"  FAIL: {a.description.strip()} -- {e}", file=sys.stderr)
            return 2
    return 0


if __name__ == "__main__":
    sys.exit(main())
