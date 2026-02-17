#!/usr/bin/env python3
"""Issue analyzer: scores open GitHub issues by "shipability" — likelihood of
getting reviewed and merged quickly.

Uses the ``gh`` CLI for all GitHub API access.  Stdlib only, no third-party
dependencies.

Usage:
    python3 tools/issue-analyzer/issue_analyzer.py [OPTIONS]
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from datetime import datetime, timezone

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

REPO = "fleetdm/fleet"

GROUP_ALIASES = {
    "mdm": "#g-mdm",
    "software": "#g-software",
    "soft": "#g-software",
    "orchestration": "#g-orchestration",
    "orch": "#g-orchestration",
    "sec": "#g-security-compliance",
}

COMPLEXITY_KEYWORDS = [
    "database migration",
    "schema",
    "REST API",
    "breaking change",
    "load test",
    "multi-platform",
]

# ---------------------------------------------------------------------------
# GitHub helpers (via ``gh`` CLI)
# ---------------------------------------------------------------------------

GH_ISSUE_FIELDS = (
    "number,title,labels,state,assignees,createdAt,updatedAt,comments,body,url"
)
GH_PR_FIELDS = "number,title,labels,state,additions,deletions,createdAt,mergedAt,body,url"


def _run_gh(args: list[str]) -> str:
    """Run a ``gh`` CLI command and return stdout."""
    try:
        result = subprocess.run(
            ["gh"] + args,
            capture_output=True,
            text=True,
            check=True,
            timeout=120,
        )
        return result.stdout
    except FileNotFoundError:
        print("Error: 'gh' CLI is not installed. Install from https://cli.github.com/", file=sys.stderr)
        sys.exit(1)
    except subprocess.CalledProcessError as exc:
        print(f"Error running gh: {exc.stderr.strip()}", file=sys.stderr)
        sys.exit(1)


def fetch_open_issues(limit: int) -> list[dict]:
    raw = _run_gh([
        "issue", "list",
        "--repo", REPO,
        "--state", "open",
        "--json", GH_ISSUE_FIELDS,
        "--limit", str(limit),
    ])
    return json.loads(raw) if raw.strip() else []


def fetch_closed_issues(limit: int) -> list[dict]:
    raw = _run_gh([
        "issue", "list",
        "--repo", REPO,
        "--state", "closed",
        "--json", GH_ISSUE_FIELDS,
        "--limit", str(limit),
    ])
    return json.loads(raw) if raw.strip() else []


def fetch_merged_prs(limit: int) -> list[dict]:
    raw = _run_gh([
        "pr", "list",
        "--repo", REPO,
        "--state", "merged",
        "--json", GH_PR_FIELDS,
        "--limit", str(limit),
    ])
    return json.loads(raw) if raw.strip() else []


def fetch_open_prs(limit: int) -> list[dict]:
    raw = _run_gh([
        "pr", "list",
        "--repo", REPO,
        "--state", "open",
        "--json", GH_PR_FIELDS,
        "--limit", str(limit),
    ])
    return json.loads(raw) if raw.strip() else []


def search_linked_prs(issue_numbers: list[int]) -> dict[int, str]:
    """Search for PRs that reference specific issue numbers.

    Uses ``gh search prs`` to find PRs (open or merged) mentioning each issue.
    Returns {issue_number: pr_state} where state is 'open' or 'merged'.
    """
    if not issue_numbers:
        return {}

    # Batch search: query for all issue numbers at once
    # gh search has a query length limit, so batch in groups
    linked: dict[int, str] = {}
    for issue_num in issue_numbers:
        try:
            raw = _run_gh([
                "search", "prs",
                "--repo", REPO,
                "--json", "number,title,body,state",
                "--limit", "5",
                "--", str(issue_num),
            ])
        except SystemExit:
            continue
        prs = json.loads(raw) if raw.strip() else []
        for pr in prs:
            pr_text = (pr.get("title", "") + " " + (pr.get("body") or ""))
            if f"#{issue_num}" in pr_text or f"/issues/{issue_num}" in pr_text:
                state = pr.get("state", "").lower()
                if state == "merged" or issue_num not in linked:
                    linked[issue_num] = state
    return linked


# ---------------------------------------------------------------------------
# Parsing helpers
# ---------------------------------------------------------------------------

def _label_names(issue: dict) -> set[str]:
    return {l["name"].strip().lower() for l in issue.get("labels", [])}


def _label_names_original(issue: dict) -> list[str]:
    return [l["name"].strip() for l in issue.get("labels", [])]


def _parse_dt(s: str) -> datetime:
    """Parse ISO-8601 timestamps returned by ``gh``."""
    s = s.replace("Z", "+00:00")
    return datetime.fromisoformat(s)


def _now() -> datetime:
    return datetime.now(timezone.utc)


def _days_between(a: datetime, b: datetime) -> float:
    return abs((b - a).total_seconds()) / 86400


def _group_labels(issue: dict) -> list[str]:
    return [l["name"].strip() for l in issue.get("labels", []) if l["name"].strip().lower().startswith("#g-")]


def _priority(issue: dict) -> str | None:
    for l in issue.get("labels", []):
        name = l["name"].strip().upper()
        if name in ("P0", "P1", "P2"):
            return name
    return None


def _has_label(issue: dict, target: str) -> bool:
    return target.lower() in _label_names(issue)


def _has_label_prefix(issue: dict, prefix: str) -> bool:
    return any(n.startswith(prefix.lower()) for n in _label_names(issue))


# ---------------------------------------------------------------------------
# Historical analysis
# ---------------------------------------------------------------------------

def _extract_resolved_issues(text: str | None) -> list[int]:
    """Extract issue numbers from closing keywords and '#XXXX' references.

    Matches GitHub closing keywords (Resolves, Fixes, Closes) and bare #XXXX
    references in PR bodies and titles.
    """
    if not text:
        return []
    # GitHub closing keywords: close/closes/closed, fix/fixes/fixed, resolve/resolves/resolved
    keyword_refs = re.findall(
        r"(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s+#(\d+)", text, re.IGNORECASE
    )
    # Also match "Related issue: ... #XXXX" (from PR template)
    related_refs = re.findall(r"Related\s+issue.*?#(\d+)", text, re.IGNORECASE)
    # Match full URL references: github.com/.../issues/XXXX
    url_refs = re.findall(r"github\.com/[^/]+/[^/]+/issues/(\d+)", text)
    all_refs = set(int(n) for n in keyword_refs + related_refs + url_refs)
    return list(all_refs)


def build_pr_issue_map(prs: list[dict]) -> dict[int, dict]:
    """Build a mapping from issue number to PR for a list of PRs."""
    issue_to_pr: dict[int, dict] = {}
    for pr in prs:
        for issue_num in _extract_resolved_issues(pr.get("body")):
            issue_to_pr[issue_num] = pr
    return issue_to_pr


def build_history(closed_issues: list[dict], merged_prs: list[dict],
                  open_prs: list[dict] | None = None) -> dict:
    """Build historical profile from closed issues and merged PRs.

    Returns dict with:
        label_speed: {label: median_days_to_close}
        pr_size_by_label: {label: median_lines_changed}
        issue_to_pr: {issue_number: pr_dict}  (merged PRs)
        issues_with_open_pr: set of issue numbers with open PRs
        issues_with_merged_pr: set of issue numbers with merged PRs
    """
    # Map issue numbers to merged PRs via "Resolves #" pattern
    issue_to_pr = build_pr_issue_map(merged_prs)
    issues_with_merged_pr = set(issue_to_pr.keys())

    # Map issue numbers to open PRs
    issues_with_open_pr: set[int] = set()
    if open_prs:
        issues_with_open_pr = set(build_pr_issue_map(open_prs).keys())

    # Label -> list of days-to-close
    label_days: dict[str, list[float]] = {}
    for issue in closed_issues:
        created = _parse_dt(issue["createdAt"])
        closed = _parse_dt(issue["updatedAt"])  # updatedAt ≈ closedAt for closed issues
        days = _days_between(created, closed)
        for lbl in _label_names(issue):
            label_days.setdefault(lbl, []).append(days)

    # Label -> list of PR sizes
    label_pr_size: dict[str, list[int]] = {}
    for pr in merged_prs:
        size = pr.get("additions", 0) + pr.get("deletions", 0)
        for lbl in _label_names(pr):
            label_pr_size.setdefault(lbl, []).append(size)

    def _median(vals: list[float]) -> float:
        s = sorted(vals)
        n = len(s)
        if n == 0:
            return 0.0
        mid = n // 2
        return (s[mid] + s[mid - 1]) / 2 if n % 2 == 0 else s[mid]

    label_speed = {lbl: _median(days) for lbl, days in label_days.items()}
    pr_size_by_label = {lbl: _median(sizes) for lbl, sizes in label_pr_size.items()}

    return {
        "label_speed": label_speed,
        "pr_size_by_label": pr_size_by_label,
        "issue_to_pr": issue_to_pr,
        "issues_with_open_pr": issues_with_open_pr,
        "issues_with_merged_pr": issues_with_merged_pr,
    }


# ---------------------------------------------------------------------------
# Complexity estimation
# ---------------------------------------------------------------------------

def estimate_complexity(issue: dict) -> str:
    """Heuristic complexity estimate: low, medium, or high."""
    body = issue.get("body") or ""
    score = 0

    # Body length
    if len(body) > 2000:
        score += 2
    elif len(body) > 800:
        score += 1

    # Checkbox count (from issue templates)
    checkboxes = len(re.findall(r"- \[[ x]\]", body))
    if checkboxes > 8:
        score += 2
    elif checkboxes > 3:
        score += 1

    # Complexity keywords
    body_lower = body.lower()
    for kw in COMPLEXITY_KEYWORDS:
        if kw.lower() in body_lower:
            score += 1

    if score >= 4:
        return "high"
    elif score >= 2:
        return "medium"
    return "low"


# ---------------------------------------------------------------------------
# Scoring
# ---------------------------------------------------------------------------

def score_issue(issue: dict, history: dict, now: datetime) -> dict:
    """Score an issue on 0-100 shipability scale.

    Returns dict with: score, reasons (list of str), difficulty, breakdown (dict).
    """
    breakdown: dict[str, int] = {}
    reasons: list[str] = []
    labels = _label_names(issue)
    original_labels = _label_names_original(issue)

    # --- Positive signals ---

    # :release label
    if _has_label(issue, ":release"):
        breakdown[":release"] = 20
        reasons.append(":release")

    # Priority
    prio = _priority(issue)
    if prio == "P0":
        breakdown["priority"] = 15
        reasons.append("P0")
    elif prio == "P1":
        breakdown["priority"] = 12
        reasons.append("P1")

    # Bug label
    if "bug" in labels:
        breakdown["bug"] = 10
        reasons.append("bug")

    # Sub-task label
    if "~sub-task" in labels:
        breakdown["sub-task"] = 10
        reasons.append("~sub-task")

    # Single group ownership
    groups = _group_labels(issue)
    if len(groups) == 1:
        breakdown["single_group"] = 8
        reasons.append(groups[0])
    elif len(groups) > 1:
        breakdown["multi_group"] = -5
        reasons.append("multi-group")

    # Customer/prospect
    if _has_label_prefix(issue, "customer-") or _has_label_prefix(issue, "prospect-"):
        breakdown["customer"] = 7
        reasons.append("customer/prospect")

    # Freshness (linear decay over 90 days, max +8)
    created = _parse_dt(issue["createdAt"])
    age_days = _days_between(created, now)
    freshness = max(0, 8 - (age_days / 90) * 8)
    breakdown["freshness"] = round(freshness)

    # Historical merge speed for label combo (max +7)
    label_speed = history.get("label_speed", {})
    speeds = [label_speed[l] for l in labels if l in label_speed]
    if speeds:
        median_speed = sorted(speeds)[len(speeds) // 2]
        # Faster close → higher score; cap at 7 days for 0 bonus
        speed_score = max(0, 7 - (median_speed / 7) * 7)
        breakdown["history_speed"] = round(speed_score)

    # Has assignee
    if issue.get("assignees"):
        breakdown["assignee"] = 5
        reasons.append("assigned")

    # Recent comment activity (within 14 days, max +5)
    comments = issue.get("comments", [])
    if isinstance(comments, list):
        recent = sum(
            1 for c in comments
            if _days_between(_parse_dt(c.get("createdAt", issue["createdAt"])), now) <= 14
        )
        activity = min(5, recent * 2)
        if activity > 0:
            breakdown["recent_activity"] = activity
    elif isinstance(comments, int) and comments > 0:
        # gh sometimes returns comment count instead of list
        updated = _parse_dt(issue["updatedAt"])
        if _days_between(updated, now) <= 14:
            breakdown["recent_activity"] = min(5, comments)

    # Low complexity bonus
    difficulty = estimate_complexity(issue)
    if difficulty == "low":
        breakdown["low_complexity"] = 5

    # --- Penalties ---

    # Story without :release
    if "story" in labels and ":release" not in labels:
        breakdown["story_no_release"] = -15
        reasons.append("story (no :release)")

    # No labels
    if not labels:
        breakdown["no_labels"] = -20
        reasons.append("no labels")

    # No product group
    if not groups:
        breakdown["no_group"] = -8

    # :product label (needs design review)
    if _has_label(issue, ":product"):
        breakdown["product_review"] = -10
        reasons.append(":product")

    # Stale (>90 days, no recent comments)
    if age_days > 90 and breakdown.get("recent_activity", 0) == 0:
        breakdown["stale"] = -10
        reasons.append("stale")

    # Already has a merged PR → effectively done, exclude from results
    issue_num = issue.get("number", 0)
    if issue_num in history.get("issues_with_merged_pr", set()):
        breakdown["has_merged_pr"] = -100
        reasons.append("PR already merged")

    # Has an open PR → work in progress, deprioritize
    elif issue_num in history.get("issues_with_open_pr", set()):
        breakdown["has_open_pr"] = -30
        reasons.append("PR in progress")

    total = max(0, min(100, sum(breakdown.values())))

    return {
        "score": total,
        "reasons": reasons,
        "difficulty": difficulty,
        "breakdown": breakdown,
    }


def suggest_approach(issue: dict, difficulty: str) -> str:
    """Generate a brief suggested approach based on labels and complexity."""
    labels = _label_names(issue)
    parts = []

    if "bug" in labels:
        parts.append("Reproduce, identify root cause, write regression test, fix")
    elif "~sub-task" in labels:
        parts.append("Implement scoped change per parent story requirements")
    elif "story" in labels:
        parts.append("Break into sub-tasks if not already decomposed")
    else:
        parts.append("Triage and clarify scope before starting")

    if difficulty == "high":
        parts.append("consider pairing or splitting into smaller PRs")
    elif difficulty == "low":
        parts.append("straightforward single-PR fix")

    return "; ".join(parts)


# ---------------------------------------------------------------------------
# Filtering
# ---------------------------------------------------------------------------

def filter_issues(
    issues: list[dict],
    group: str | None,
    issue_type: str | None,
    exclude_customer: bool,
) -> list[dict]:
    """Filter issues by group, type, and customer exclusion."""
    result = []
    target_group = GROUP_ALIASES.get(group, group) if group else None

    for issue in issues:
        labels = _label_names(issue)
        original_labels = _label_names_original(issue)

        # Group filter
        if target_group:
            if not any(l.lower() == target_group.lower() for l in [la["name"].strip() for la in issue.get("labels", [])]):
                continue

        # Type filter
        if issue_type:
            type_map = {
                "bug": "bug",
                "sub-task": "~sub-task",
                "story": "story",
                "quick-win": None,  # handled specially
            }
            target = type_map.get(issue_type)
            if issue_type == "quick-win":
                # Quick-win: bugs or sub-tasks with short body
                if "bug" not in labels and "~sub-task" not in labels:
                    continue
                body = issue.get("body") or ""
                if len(body) > 1500:
                    continue
            elif target and target not in labels:
                continue

        # Exclude customer/prospect
        if exclude_customer:
            if _has_label_prefix(issue, "customer-") or _has_label_prefix(issue, "prospect-"):
                continue

        result.append(issue)

    return result


# ---------------------------------------------------------------------------
# Output formatters
# ---------------------------------------------------------------------------

def _truncate(s: str, width: int) -> str:
    return s[:width - 1] + "\u2026" if len(s) > width else s


def format_table(scored: list[dict], verbose: bool) -> str:
    """Format as an ASCII table."""
    lines = []
    header = f" {'#':<6}| {'Score':>5} | {'Diff.':<8} | {'Title':<36} | Key Reasons"
    sep = "-" * 7 + "|" + "-" * 7 + "|" + "-" * 10 + "|" + "-" * 38 + "|" + "-" * 30
    lines.append(header)
    lines.append(sep)

    for entry in scored:
        issue = entry["issue"]
        s = entry["scoring"]
        reasons_str = ", ".join(s["reasons"][:5]) if s["reasons"] else "-"
        title = _truncate(issue["title"], 36)
        line = f" {issue['number']:<6}| {s['score']:>5} | {s['difficulty']:<8} | {title:<36} | {reasons_str}"
        lines.append(line)

        if verbose:
            for k, v in sorted(s["breakdown"].items(), key=lambda x: -abs(x[1])):
                sign = "+" if v >= 0 else ""
                lines.append(f"        {k}: {sign}{v}")
            lines.append("")

    return "\n".join(lines)


def format_json(scored: list[dict]) -> str:
    """Format as JSON."""
    output = []
    for entry in scored:
        issue = entry["issue"]
        s = entry["scoring"]
        output.append({
            "number": issue["number"],
            "title": issue["title"],
            "url": issue.get("url", ""),
            "score": s["score"],
            "difficulty": s["difficulty"],
            "reasons": s["reasons"],
            "suggested_approach": s["approach"],
            "breakdown": s["breakdown"],
            "labels": _label_names_original(issue),
        })
    return json.dumps(output, indent=2)


def format_markdown(scored: list[dict]) -> str:
    """Format as GitHub-compatible markdown table."""
    lines = [
        "| # | Score | Difficulty | Title | Key Reasons |",
        "|---|-------|------------|-------|-------------|",
    ]
    for entry in scored:
        issue = entry["issue"]
        s = entry["scoring"]
        reasons_str = ", ".join(s["reasons"][:5]) if s["reasons"] else "-"
        title = _truncate(issue["title"], 50).replace("|", "\\|")
        lines.append(
            f"| [{issue['number']}]({issue.get('url', '')}) "
            f"| {s['score']} "
            f"| {s['difficulty']} "
            f"| {title} "
            f"| {reasons_str} |"
        )
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Analyze GitHub issues for shipability — likelihood of quick review and merge."
    )
    parser.add_argument("--group", type=str, default=None,
                        help="Filter by product group (mdm, software, orchestration, sec)")
    parser.add_argument("--type", type=str, default=None, dest="issue_type",
                        help="Filter by issue type (bug, sub-task, story, quick-win)")
    parser.add_argument("--min-score", type=int, default=0,
                        help="Minimum score to display (default: 0)")
    parser.add_argument("--exclude-customer", action="store_true",
                        help="Exclude customer/prospect issues")
    parser.add_argument("--limit", type=int, default=200,
                        help="Open issues to fetch (default: 200)")
    parser.add_argument("--history", type=int, default=100,
                        help="Closed issues for historical analysis (default: 100)")
    parser.add_argument("--format", type=str, default="table", choices=["table", "json", "markdown"],
                        dest="output_format",
                        help="Output format: table (default), json, markdown")
    parser.add_argument("--top", type=int, default=20,
                        help="Show top N results (default: 20)")
    parser.add_argument("--verbose", action="store_true",
                        help="Show full scoring breakdown")

    args = parser.parse_args()

    # Fetch data (4 API calls)
    print("Fetching open issues...", file=sys.stderr)
    open_issues = fetch_open_issues(args.limit)

    print("Fetching closed issues for history...", file=sys.stderr)
    closed_issues = fetch_closed_issues(args.history)

    print("Fetching merged PRs for history...", file=sys.stderr)
    merged_prs = fetch_merged_prs(args.history)

    print("Fetching open PRs...", file=sys.stderr)
    open_prs = fetch_open_prs(args.history)

    print(f"Analyzing {len(open_issues)} open issues "
          f"(history: {len(closed_issues)} closed, {len(merged_prs)} merged PRs, "
          f"{len(open_prs)} open PRs)...",
          file=sys.stderr)

    # Build historical profile
    history = build_history(closed_issues, merged_prs, open_prs)

    # Filter
    filtered = filter_issues(open_issues, args.group, args.issue_type, args.exclude_customer)

    # Score (initial pass — before PR link check)
    now = _now()
    scored = []
    for issue in filtered:
        s = score_issue(issue, history, now)
        s["approach"] = suggest_approach(issue, s["difficulty"])
        scored.append({"issue": issue, "scoring": s})

    # Sort by score descending, then by issue number descending
    scored.sort(key=lambda x: (-x["scoring"]["score"], -x["issue"]["number"]))

    # Apply min-score filter
    scored = [e for e in scored if e["scoring"]["score"] >= args.min_score]

    # Take a candidate pool (2x top to allow for filtering)
    candidates = scored[: args.top * 2]

    # Search for linked PRs only on top candidates (not all 200 issues)
    already_known = history["issues_with_merged_pr"] | history["issues_with_open_pr"]
    candidate_numbers = [
        e["issue"]["number"] for e in candidates
        if e["issue"]["number"] not in already_known
    ]
    if candidate_numbers:
        print(f"Checking {len(candidate_numbers)} top candidates for linked PRs...",
              file=sys.stderr)
        linked_prs = search_linked_prs(candidate_numbers)
        for issue_num, state in linked_prs.items():
            if state == "merged":
                history["issues_with_merged_pr"].add(issue_num)
            elif state == "open":
                history["issues_with_open_pr"].add(issue_num)

        # Re-score candidates with updated PR info
        rescored = []
        for entry in candidates:
            s = score_issue(entry["issue"], history, now)
            s["approach"] = suggest_approach(entry["issue"], s["difficulty"])
            rescored.append({"issue": entry["issue"], "scoring": s})
        rescored.sort(key=lambda x: (-x["scoring"]["score"], -x["issue"]["number"]))
        # Filter out issues with merged PRs (score 0) and apply min-score
        scored = [
            e for e in rescored
            if e["scoring"]["score"] >= max(1, args.min_score)
        ]
    else:
        scored = candidates

    # Limit results
    scored = scored[: args.top]

    if not scored:
        print("No issues matched the given criteria.", file=sys.stderr)
        sys.exit(0)

    # Output
    if args.output_format == "json":
        print(format_json(scored))
    elif args.output_format == "markdown":
        print(format_markdown(scored))
    else:
        print(format_table(scored, args.verbose))


if __name__ == "__main__":
    main()
