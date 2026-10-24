# Hot-table transaction and lock discipline

> **Template:** 🔧 Reliability · **Labels:** `reliability`, `:help-engineering` · **Effort:** M–L · **Wave:** 3

## Problem

Under fleet-scale write load, what falls over first is usually lock contention, not CPU: transactions that hold row/gap locks on hot tables (`hosts`, `host_seen_times`, software/label junction tables, MDM command queues) while doing other slow work — extra queries, loops, external calls — serialize the hot path. Deadlock retries then amplify load exactly when the server is busiest.

## Impact

Appears only at high concurrency, so it ships invisibly and manifests as p99 collapse and deadlock storms at the largest customer. It also sets the effective capacity ceiling: shedding (issues 02/04) protects the server from overload, but lock discipline determines how much throughput exists below the ceiling.

## Proposed fix

- **Identify the hot set** empirically: run the proportionate rig (issue 17) at capacity and pull MySQL lock-wait and deadlock stats (`performance_schema`, innodb status) to rank tables/statements by lock wait time — audit from data, not from reading every transaction.
- For each top offender: shorten the transaction (move reads out, precompute before BEGIN), reduce lock footprint (narrower updates, avoid gap-locking patterns, consistent ordering to cut deadlocks), or restructure (e.g. insert-only + periodic compaction instead of contended upserts).
- **Regression guard:** record a lock-wait-per-check-in baseline in the cost-budget harness (issue 18) so future PRs that add lock time to the hot path show up in branch-vs-main comparison.
- Document the rules (short transactions, no slow work under locks, lock ordering) in `.claude/rules/fleet-database.md` / contributor docs.

## Condition of satisfaction

- Ranked lock-contention baseline published from the rig.
- Top N (≥5) offenders restructured with measured lock-wait reduction.
- Lock-wait metric wired into the issue 18 harness.
