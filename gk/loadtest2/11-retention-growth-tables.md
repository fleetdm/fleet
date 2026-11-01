# Retention story for every activity-growth table

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M–L · **Wave:** 3

## Problem

Tables that grow with *activity* (not with entity count) grow forever unless something deletes: script results, activity feeds, command queues/results, query result history, upcoming/past activities, audit-ish logs. Some have cleanup today; nobody has inventoried all of them. At 400k hosts, an activity table without retention gains tens of millions of rows per week until queries slow, backups balloon, and migrations on it become outage-length.

## Impact

Slow-burn but compounding — the failure arrives months after the feature shipped, at the biggest customer first, and the remediation (deleting hundreds of millions of rows, altering a huge table) is itself a risky operation.

## Proposed fix

- **Inventory:** every table, classified entity-growth vs. activity-growth; for activity tables, current retention mechanism or NONE. Table goes in this issue.
- **Retrofit:** each NONE gets a retention policy (TTL, per-host row cap, or both — product sign-off where data is user-visible) and a cleanup job meeting issue 08's four requirements (paged deletes with limits — never one giant DELETE).
- **Verify the existing** cleanups actually keep up at scale: measured delete throughput vs. 400k-host insert rate on the seeded snapshot (issue 16).
- **Policy going forward:** a PR creating an activity-growth table ships retention in the same PR — enforced via the scale checklist (issue 19) and migration review.

## Condition of satisfaction

- Inventory complete with zero NONE rows remaining (or follow-up issues filed per table with owners).
- Cleanup jobs demonstrated to sustain equilibrium (delete rate ≥ insert rate) on the local rig at simulated 400k-host activity volume.

## Evidence

Recent related work: script results cleanup guards (#54439/#54448) — an instance of exactly this class, fixed reactively; this issue makes the class systematic.
