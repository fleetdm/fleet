# Background job audit: paginate, cap, and lock every cron

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M · **Wave:** 2

## Problem

Cron/worker jobs written against small fleets tend toward the "select all hosts, loop" shape — bounded at 1k hosts, broken at 400k: one query materializes the whole fleet, the loop runs for hours, holds resources, and if the schedule fires again you get overlap or an ever-growing backlog.

## Impact

Jobs are where scale problems hide longest because nothing times out and no user sees latency — until the job starts starving the hot path, missing its schedule entirely (vuln processing, profile reconciliation falling behind), or blowing memory.

## Proposed fix

Audit every registered cron/schedule and worker job against four requirements; retrofit the ones that fail:

1. **Paged, not materialized** — process hosts/rows in keyset-paginated batches with a bounded batch size; never load the full fleet into memory.
2. **Per-run cap** — a maximum work budget per run, carrying a cursor so the next run resumes; a backlog slows convergence, never produces a six-hour run.
3. **Non-overlapping** — a lock (the existing schedule locking) verified for every job, including ad-hoc worker jobs.
4. **Resumable** — a crash mid-run loses at most one batch of progress.

Produce the audit as a table in the issue (job → four checkmarks → gap). Fix the high-traffic offenders in this issue; file follow-ups for long-tail jobs. Add the four requirements to the scale checklist (issue 19) so new jobs comply by construction.

## Condition of satisfaction

- Audit table complete for every job registered in the schedule/worker systems.
- Jobs touching all hosts (host counts, vuln processing, profile reconciliation, cleanup jobs) verified or retrofitted to all four requirements.
- Local rig check: with a 100k-host seeded snapshot (issue 16), no job's single run exceeds its documented budget.

## Evidence

Fleet's schedule package already provides cross-container locking; the gaps are typically pagination and per-run caps, not locking.
