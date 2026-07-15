# 4.86 Windows MDM Load Test — 30k Hosts

**Workspace:** `486-windows`
**Date:** 2026-05-19
**Goal:** Confirm 4.86 can sustain 30k Windows hosts with MDM enabled across the profile-interaction operations that previously stressed 4.85.
**Result:** All four operations sustained without errors, deadlocks, evictions, or 5xx. Fleet server CPU held steady around 72%, RDS writer peaked at ~50%, replica lag stayed under 11ms. Significant headroom remained on every tier.

## Test plan

Four sequential operations on a fleet of 32k Windows hosts (15 Fleet server containers, 16 loadtest containers, Aurora writer + 2 readers, ElastiCache Redis):

| # | Time (local) | Operation | Window covered |
|---|---|---|---|
| 1 | 5:10 PM | Add 15 profiles to current team | 22:12 → 22:47 UTC (35 min) |
| 2 | 5:51 PM | Move 32k hosts → team with 0 profiles (force-delete profiles) | 22:49 → 23:09 UTC (20 min) |
| 3 | 6:10 PM | Move 32k hosts → team with 15 profiles (transfer + reapply) | 23:08 → 23:38 UTC (30 min) |
| 4 | 6:40 PM | Move 32k hosts → different team with another set of 15 profiles | 23:39 → 00:04 UTC (25 min) |

## Headline numbers

| Metric | Test 1: add 15 | Test 2: → 0 prof | Test 3: → 15 prof | Test 4: team→team |
|---|---|---|---|---|
| **Fleet server CPU (avg)** | 72.29% | 71.55% | 73.78% | 71.56% |
| **Fleet server mem (avg)** | 3.25% | 3.29% | 3.29% | 3.35% |
| **RDS writer CPU (avg)** | 47.62% | 38.20% | 50.11% | 50.72% |
| **RDS writer connections** | 160 | 121 | 155 | 158 |
| **RDS writer IOPS util** | 10.78% | 9.30% | 12.07% | 9.33% |
| **RDS writer InsertLat** | 0.73s | 1.01s | 0.79s | 1.68s |
| **RDS writer SelectLat** | 0.92s | 0.19s | 0.78s | 0.22s |
| **RDS writer threads (AAS)** | 2.66 | 2.01 | 3.05 | 2.93 |
| **RDS reader-1 CPU** | 57.90% | 51.44% | 53.16% | 53.23% |
| **RDS reader-2 CPU** | 64.77% | 68.70% | 76.17% | 74.74% |
| **Aurora replica lag (max)** | ~9 ms | ~10 ms | ~8 ms | ~10 ms |
| **Buffer cache hit ratio** | 100% / 99.99% | 100% / 99.99% | 100% / 99.99% | 100% / 99.99% |
| **ALB 5xx** | 0 | 0 | 0 | 0 |
| **Fleet server errors** | 0 | 0 | 0 | 0 |
| **Deadlocks** | 0 | 0 | 0 | 0 |
| **ALB throughput** | 20.3M req / 17.7 GB | 11.5M req / 9.5 GB | 17.4M req / 15.3 GB | 14.4M req / 12.0 GB |

## Fleet server CPU

CPU stayed within a tight 71.5 – 73.8% band across all four operations — i.e. the server tier behaved identically regardless of which profile operation was running. Memory was negligible at ~3.3%. With 15 containers running steady-state at ~72% and no scale-up triggered, the cluster has ~28% per-container headroom (and could scale out further if needed).

No abnormal container stops, no restarts, no errors in the Fleet log stream over any of the four windows.

## RDS writer

Writer CPU peaked at 50.7% (Test 4) and never came near saturation. The heavier operations — the two "transfer + reapply 15 profiles" cases — are visibly more write-heavy than the "drop profiles to a team with none" case:

- **Test 2 (drop to 0 profiles)** is the lightest: 38% CPU, 121 connections, 2.0 AAS. Profile removal is mostly DELETEs against a single set of MDM tables.
- **Tests 3 and 4 (transfer + apply 15 profiles)** are the heaviest: ~50% CPU, ~155 connections, ~3.0 AAS, 10–12% IOPS utilization. These do INSERTs into the per-host profile/command tables for 32k × 15 = ~480k rows of profile state per operation.

`COMMIT` dominates writer load in both top-SQL captures (1.9–2.1 AAS), which is expected — high write throughput with small transactions. No single application query came close to saturating the writer; #2 onwards is all under 0.2 AAS.

**InsertLat in Test 4 jumped to 1.68s** (vs 0.73–1.01s elsewhere). This is the team-to-team move, which is the most write-intensive (deletes from old team's profile state + inserts to new team's). Still well within healthy range and not surfacing as user-visible latency — the ALB recorded 0 5xx and the host-side request rate held at ~570k req/min.

0 deadlocks across all four operations.

## RDS readers

Reader-1 stayed in the 51 – 58% CPU band; reader-2 ran consistently hotter at 64 – 76%, peaking at 76.17% during Test 3 (the heaviest profile-reapply operation). The asymmetry is just routing — both readers are healthy.

Replica lag stayed between 7.9 and 10.2 ms across all four tests, which is well inside Aurora's healthy range and well below the threshold (20–50 ms) where stale reads start to matter for Fleet's workflows.

Buffer cache hit ratio was 99.99% on both readers throughout — they're serving their working set entirely from memory.

Top reader queries are the expected MDM read paths: `windows_mdm_enrollments` lookups, profile-status queries, and the live-query scheduler's pack/query resolution. None exceeded 0.26 AAS individually.

## Redis

No notable activity. CPU 22–24%, memory ~1.6%, **0 evictions**, ~93–140 active connections. Cache hit rate sat at ~65% — this is the working baseline for Fleet's Redis usage pattern under load and didn't shift during any operation.

## What this means

- **30k Windows hosts with MDM on is sustainable on 4.86 at the current cluster size.** All four operations completed cleanly with significant headroom on every tier.
- **No regression signal** in this run: zero 5xx, zero application errors, zero deadlocks, zero evictions, sub-millisecond increases in replica lag.
- **Reader-2 at 76% CPU during the heaviest operation (Test 3)** is the closest thing to a watchpoint — worth re-checking on the next test if scale increases or the operation pattern changes, but not flagging today.
- **Writer InsertLat of 1.68s in Test 4** is the other thing to keep an eye on if team-to-team moves become a common operation, but it's not user-facing (ALB target response time was effectively 0s and no 5xx).
- Compared to the 4.85 results that prompted this validation, the cluster ran well below the saturation behavior that previously appeared at this scale.

## Source files

- [486-windows-2026-05-19-224746Z-35m.md](486-windows-2026-05-19-224746Z-35m.md) — Test 1 (add 15 profiles)
- [486-windows-2026-05-19-230906Z-20m.md](486-windows-2026-05-19-230906Z-20m.md) — Test 2 (→ 0 profiles)
- [486-windows-2026-05-19-233833Z-30m.md](486-windows-2026-05-19-233833Z-30m.md) — Test 3 (→ 15 profiles)
- [486-windows-2026-05-20-000458Z-25m.md](486-windows-2026-05-20-000458Z-25m.md) — Test 4 (team-to-team move)
