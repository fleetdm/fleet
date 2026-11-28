# Load protection issue drafts — review before filing

One file per GitHub issue. Each file's header block says which template to use (🔧 Reliability vs 🎟 Story), suggested labels, effort, and wave. Per PR review (v5.0 is already loaded), **09 mandatory pagination is the only item targeting the v5.0 breaking window**; 05 and 10 were reframed to ship additively, with their contract-enforcement portions deferred to a later major. Filing order within a wave doesn't matter except where "Depends on" says otherwise.

**End goal:** no matter how high load gets, the server keeps functioning — sheds deferrable work, protects critical flows and the UI, signals backoff agents honor, recovers on its own.

| # | File | Title | Effort | Wave | v5.0 breaking |
|---|------|-------|--------|------|---------------|
| 00 | 00-epic-load-protection.md | Epic: server load protection for large fleets | — | all | — |
| 01 | 01-overload-observability.md | Overload observability (pool stats, metrics, shed counters) | S | 1 | |
| 02 | 02-concurrency-cap-middleware.md | Per-container in-flight concurrency cap | S–M | 1 | |
| 03 | 03-never-node-invalid-on-error.md | osquery errors must never signal node_invalid | S | 1 | |
| 04 | 04-db-pressure-load-shedding.md | Adaptive load shedding driven by DB pressure | M | 2 | |
| 05 | 05-overload-semantics-priority-tiers.md | Uniform overload semantics + endpoint priority tiers | L | 2–3 | additive now, enforce next major |
| 06 | 06-fleetd-honors-retry-after.md | fleetd honors Retry-After (start early — agent release cycle) | M | 2 | |
| 07 | 07-fanout-safety-splay-convergence.md | Fan-out safety: splay + reconcile convergence | M | 2 | |
| 08 | 08-background-job-chunking.md | Background job audit: paginate, cap, lock | M | 2 | |
| 09 | 09-v5-mandatory-pagination.md | Mandatory pagination on all list endpoints | M–L | 3 | ✔ |
| 10 | 10-payload-size-caps.md | Payload size caps on agent-submitted data (incl. osquery 413 retry-loop verification) | M | 3 | additive (truncate-and-flag) |
| 11 | 11-retention-growth-tables.md | Retention for every activity-growth table | M–L | 3 | |
| 12 | 12-chunk-bulk-ops.md | Chunk bulk operations and IN lists | S–M | 2 | |
| 13 | 13-hot-table-lock-discipline.md | Hot-table transaction/lock discipline | M–L | 3 | |
| 14 | 14-checkin-path-cost-audit.md | Check-in path cost audit (write-on-change + caching) | M | 3 | |
| 15 | 15-circuit-breakers-external-deps.md | Circuit breakers for external dependencies | M | 3 | |
| 16 | 16-seeded-scale-snapshots.md | Seeded-scale DB snapshots | M | local track | |
| 17 | 17-proportionate-local-rig.md | Proportionate local load test rig | M | local track | |
| 18 | 18-cost-budgets-regression-harness.md | Cost budgets + branch-vs-main harness | M | local track | |
| 19 | 19-scale-checklist-process.md | Scale checklist + load-test gate | S | 1 | |

## Sequencing logic

- **Wave 1** buys safety margin and visibility (01, 02, 03, 19) — small, ship now.
- **Local track** (16 → 17 → 18) runs in parallel from day one; most other issues' conditions of satisfaction reference the rig, so it's the critical enabler.
- **06 starts immediately** despite being wave 2: it rides the fleetd release cycle and customers upgrade agents slowly.
- **Wave 2** makes overload survivable (04, 05, 07, 08, 12).
- **Wave 3** raises the capacity ceiling (10, 11, 13, 14, 15) and lands 09, the one v5.0 breaking change.
- **v5.0.0 deadline:** only 09 (mandatory pagination) must land its breaking portion by the release; 05 and 10 ship additively on their own schedule.

## Cross-links to existing issues (coordinate on filing)

- 06 ↔ **#45553** "Intelligent backoff for all agent-to-server communications" (P2 postmortem action item, #g-orchestration, no milestone) — kept separate: 06 is the narrow Retry-After compliance piece; cross-link both ways.
- 11 ↔ **#54349** "Retro AI: Audit unbounded database tables" (#g-auto-patching) — #54349 is the audit half; 11 consumes its output for retention/retrofit.
- 14 notes **#54009** — async host processing is removed in v5.0; audit targets the post-removal paths.

## Open questions to settle during review

1. Product group routing — which of these go to MDM vs orchestration vs software boards; the epic probably needs an owner per wave.
2. Tier contents in 05's table need product sign-off, not just engineering.
3. Limit values in 10 should be validated against cloud telemetry before we commit numbers.
4. Whether 09 (mandatory pagination) is one story or an epic with per-domain sub-tasks (hosts/software/vulns/other).
5. Snapshot storage location for 16 (S3 bucket vs GH release artifacts).

## Resolved during PR review (Oct–Nov 2026)

- v5.0 scope: 09 only; 05 and 10 reframed additive-first (getvictor: release already loaded, pagination highest priority).
- Observability is OTEL-first — pool stats already exported via `otelsql.RegisterDBStatsMetrics`; direct Prometheus support is being dropped (01 updated).
- 03 expanded to orbit endpoints, not just osquery.
- Jordan's osquery oversized-batch retry-loop concern folded into 10 as its mandatory first task.
- 17 gained an AWS-calibration condition of satisfaction; 02 exempts long-lived WebSockets from the semaphore; 06 jitter is additive-only above Retry-After; 07 adds fleet-size-aware fan-out.
