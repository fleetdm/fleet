# Load protection issue drafts — review before filing

One file per GitHub issue. Each file's header block says which template to use (🔧 Reliability vs 🎟 Story), suggested labels, effort, and wave. The three **v5.0.0 breaking** items (05, 09, 10) are stories because they change the API contract; everything else fits the reliability template. Filing order within a wave doesn't matter except where "Depends on" says otherwise.

**End goal:** no matter how high load gets, the server keeps functioning — sheds deferrable work, protects critical flows and the UI, signals backoff agents honor, recovers on its own.

| # | File | Title | Effort | Wave | v5.0 breaking |
|---|------|-------|--------|------|---------------|
| 00 | 00-epic-load-protection.md | Epic: server load protection for large fleets | — | all | — |
| 01 | 01-overload-observability.md | Overload observability (pool stats, metrics, shed counters) | S | 1 | |
| 02 | 02-concurrency-cap-middleware.md | Per-container in-flight concurrency cap | S–M | 1 | |
| 03 | 03-never-node-invalid-on-error.md | osquery errors must never signal node_invalid | S | 1 | |
| 04 | 04-db-pressure-load-shedding.md | Adaptive load shedding driven by DB pressure | M | 2 | |
| 05 | 05-v5-overload-semantics-priority-tiers.md | Uniform overload semantics + endpoint priority tiers | L | 2–3 | ✔ |
| 06 | 06-fleetd-honors-retry-after.md | fleetd honors Retry-After (start early — agent release cycle) | M | 2 | |
| 07 | 07-fanout-safety-splay-convergence.md | Fan-out safety: splay + reconcile convergence | M | 2 | |
| 08 | 08-background-job-chunking.md | Background job audit: paginate, cap, lock | M | 2 | |
| 09 | 09-v5-mandatory-pagination.md | Mandatory pagination on all list endpoints | M–L | 3 | ✔ |
| 10 | 10-v5-payload-size-caps.md | Payload size caps on agent-submitted data | M | 3 | ✔ |
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
- **Wave 3** raises the capacity ceiling and lands the remaining v5.0 breaking changes (09, 10, 11, 13, 14, 15).
- **v5.0.0 deadline (~6 weeks):** the three breaking items (05, 09, 10) must land their API-contract portions by then; their long-tail enforcement/audit work can follow the release.

## Open questions to settle during review

1. Product group routing — which of these go to MDM vs orchestration vs software boards; the epic probably needs an owner per wave.
2. Tier contents in 05's table need product sign-off, not just engineering.
3. Limit values in 10 should be validated against cloud telemetry before we commit numbers.
4. Whether 09 (mandatory pagination) is one story or an epic with per-domain sub-tasks (hosts/software/vulns/other).
5. Snapshot storage location for 16 (S3 bucket vs GH release artifacts).
