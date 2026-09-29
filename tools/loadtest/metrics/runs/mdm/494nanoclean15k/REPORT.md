# Apple MDM command cleanup load test — 15k hosts (#52597)

**Workspace:** `494nanoclean15k`
**Dates:** 2026-09-28 → 2026-09-29
**Goal:** Measure the nano_ cleanup shipped in #49851 (sweeps + knobs, #52596) on a database seeded the way a large customer's looks: upgrade cost (index migrations), backlog drain at default and raised knobs, steady-state query load versus the pre-cleanup build.
**Result:** Upgrade ALTERs on 12M-row tables took 14 + 11 minutes (26 min total downtime). At default knobs the cleanup removes 3k rows/h — below this fleet's 36k rows/h inflow — so a backlog never drains. Raising the knobs changes nothing: the 10-minute run cap (a code constant) binds at ~60k rows/h. With the cap lifted to 45 min and runs back to back the mailbox drains ~310k rows/h (sweep + per-ack cleanup) and this 35M-row backlog would settle in ~5 days at ~0.8 GB. Steady-state cost of the cleanup on the writer is small (deletes at ~0.4 load); the older per-ack refetch cleanup, now on a 24 h window, is the heaviest writer statement during the drain. No data loss or errors beyond the run-cap cancellation; no reader/redis/mock issues attributable to the cleanup.

## Setup

| Piece | Value |
|---|---|
| Hosts | 15,000 simulated: 10,000 macOS (team Ducks) + 5,000 iOS (No team), all MDM-enrolled via mock APNs |
| Baseline build | `fleetdm/fleet:v4.92.1` (no cleanup code) |
| Seed build | 4.92.1 + iOS refetch every 1 min instead of 1 h (`loadtest-52597-seed`) |
| Cleanup build | main @ a9b2f46d38 (`loadtest-52597-cleanup`), stack #53599 → #53935 |
| Profiles | 50 per scope (10 unscoped + 4 per manual label × 10 labels), applied to both scopes |
| Fleet | 10 Fargate tasks, 1024 CPU / 4 GB |
| Aurora MySQL 8.0 | writer db.r6g.large; reader db.r6g.large → **db.r6g.xlarge from 06:40Z 09-29** (see incident) |
| Redis | cache.r6g.large × 3; mock APNs 1 → 2 tasks |

## Timeline (UTC)

| When | What |
|---|---|
| 09-28 11:02 | Row-count recorder installed (MySQL events, hourly exact + 5-min estimate) |
| 09-28 11:20–12:00 | 15k hosts enrolled (osquery + MDM) |
| 09-28 12:02 / 13:00 | Profiles seeded (iOS scope, then Mac team) — 235k assignments delivered, 0 failed |
| 09-28 14:00–15:00 | **Baseline snapshot** (seed build, steady refetch traffic) |
| 09-28 → 09-29 02:30 | Organic soak: +98,776 rows/h (measured) |
| 09-29 02:30 | Mock APNs Redis-pool exhaustion stalled pushes (restarted 03:57) |
| 09-29 03:51–04:54 | Multiplier: +33.0M rows → **35.0M rows / 30 GB**, 800 slips per device |
| 09-29 05:15–05:41 | Upgrade migrations (Fleet down) |
| 09-29 05:48 | Cleanup build serving |
| 09-29 05:51–06:40 | **Incident:** reader restart loop (see below) |
| 09-29 07:12 | First clean cleanup run — **default-knob window start** |
| 09-29 09:43 | Caps raised to 500,000 / 500,000 (`loadtest-52597-aggressive-knobs`); 10:12 first capped run |
| 09-29 10:59 | Run cap 10 → 45 min (`loadtest-52597-runcap`, load-test only) |
| 09-29 11:00–13:18 | Three back-to-back 45-min runs (load-test build); 13:30 last exact sample: 33.99M rows / 32.6 GB |
| 09-29 13:34 | Teardown (osquery-perf, then infra) |

## Migration timings (35M-row tables)

| Migration | Duration |
|---|---|
| `AddNanoCommandResultsCleanupIndex` (add `idx_ncr_status_updated_at`, drop `status`) on 12.0M rows | **14 min 16 s** |
| `AddNanoCommandsCreatedAtIndex` on 11.0M rows | **11 min 09 s** |
| `AddEnabledIndexToNanoEnrollments` (15k rows) | 27 s |
| `AddHostMDMAppleProfilesCommandUUIDIndex` (235k rows) | 4 s |
| `AddNanoCleanupGuardIndexes` | < 1 s |
| All other migrations 4.92 → main | ~5 s combined |
| **Total, Fleet unavailable** | **26 min** |

Note: the loadtest infra workflow's migrations waiter gives up after ~17 min, so the first apply failed and left Fleet scaled to 0 until a re-apply. Infra note filed.

## Drain

### Default knobs (1,000 pairs / 1,000 commands per hourly run)

Three hourly runs, 07:12 / 08:12 / 09:12 UTC, identical: `inactive_pairs_deleted=1000`, `commands_deleted=1000`, short/standard/orphan sweeps skipped once the shared row budget was spent, `row_budget_exhausted=true`. Our job took 56–61 s of each ~65 s cron group run. Exact recorder samples: slips 11,995,898 → 11,994,898 → 11,993,898 (−1,000/h per table), 29.9 GB unchanged.

Net effect on a 35M-row backlog: −3,000 rows/h against a measured inflow of ~36k rows/h on this build (14.6k–21.5k queue rows/h plus their results; 5k iPhones refetching hourly — the ~99k rows/h seen during the seed phase came from the seed build's artificial 1-minute refetch). The default cannot keep up with this fleet's inflow, let alone drain a backlog.

Device-traffic note: from ~04:20Z to 08:49Z the simulated devices were silent (the mock APNs lost the pushes for one outstanding refetch per iPhone during its stall/redeploy; on this build a silent device is only re-pushed by the APNs sweep after 24 h). At 08:49Z one push per enrollment was re-sent by hand straight to the mock (15,000 × POST, all 200); devices answered within minutes (4,977/5,000 iPhones refreshed by 09:05Z) and Fleet's hourly refetch cycle resumed unaided. Fleet was not triggered and no knob was touched.

### Raised knobs (500,000 / 500,000)

Deployed 09:43Z (Fleet restart, no migration). **First run 10:12Z: the 10-minute run cap (`appleCommandCleanupMaxRunTime`) is the binding limit, not the scan cap.** The job spent the whole run inside the inactive purge and was cancelled mid-DELETE by the run-cap context: `cleanup apple mdm commands: delete nano enrollment queue rows: context deadline exceeded`. Consequences: recorded as a cron error, no stats log line, cursors not persisted, last batch rolled back. Measured by the recorder: **~29,400 inactive pairs deleted in 10 min** (2,196,860 → 2,167,433), i.e. ~50 pairs/s, ≈ 60k rows/h at the hourly cadence — below the fleet's ~99k rows/h inflow. The retention tiers were never reached (the 2.2M inactive slips alone would take ~75 hourly runs).

Writer during the run: CPU 90–95%, ReadIOPS 1.4k–3k (cold pages), DMLLatency 60–70 ms. Performance Insights over the run window (~11 AAS on 2 vCPUs): #1 `get next command` nano join (4.35) from device check-ins; #2 the per-acknowledgement refetch cleanup `DELETE … WHERE id = ? AND command_uuid IN` (2.85) — with the 24 h window and 800 old slips per iPhone every refetch ack now deletes rows; #3 COMMIT (1.48); #4 the sweep's own `DELETE … WHERE (id, command_uuid) IN` (0.35). The sweep's statements are a small share; the run was mostly contending with saturated device traffic (devices were catching up after the 08:49Z re-push). The 11:12Z scheduled tick was pre-empted by the experiment below.

### Run cap lifted to 45 minutes (load-test-only build `loadtest-52597-runcap`, same caps, runs triggered back to back)

Deployed 10:59Z. Run #1 triggered 11:00:47Z, cut off by the cap at 11:45:59Z, still inside the inactive purge: **61,202 pairs in 45 min (~23 pairs/s)**, slower per second than the 10-min run because device check-ins (~46 concurrent `get next command` queries at ~1 s each) and the per-ack refetch deletes contend with the sweep's batch deletes on the same table; writer CPU 60–90% throughout. Recorder exact samples 10:30→11:30Z: queue −58,868, results −64,460, commands +9 (the mop never ran — no run has yet finished the pair sweeps). With back-to-back runs the gross rate is ≈80k pairs/h ≈ 160k rows/h against ~99k rows/h inflow: net ≈ −60k rows/h, i.e. ~24 days for this 35M-row backlog. Performance Insights over run #1 (`…-114600Z-45m`): **#1 the per-acknowledgement refetch cleanup `DELETE FROM nano_enrollment_queue WHERE id = ? AND command_uuid IN (…)` at load 7.04** (half the writer), #2 `get next command` 3.0, #3 COMMIT 1.0, #5 the sweep's own batch DELETE 0.43; deadlocks 0.13/min; reader lag up to 5.6 s. 19,179 refetch acknowledgements/hour each run the per-ack cleanup (its window became the 24 h short retention in the cutover, and every iPhone carries ~800 old slips), so the two deleters contend for the same rows. With the sweep in place the per-ack path is redundant on a backlog and is now the most expensive writer statement — a decision for the team (disable it, restore a wide window, or keep it only as a fallback). Runs #2 (11:46:06→12:31:20) and #3 (12:31:53→13:18:19) behaved the same: 51,390 and 52,371 inactive pairs, both cut by the cap (#3 inside the results delete, the furthest any run got). Over the three runs the inactive slips went 2,167,433 → 2,002,470; no run reached the retention tiers or the command mop, so `nano_commands` never shrank. Performance Insights over the whole window (`…-131800Z-138m`): per-ack refetch cleanup 3.88, `get next command` 1.74, COMMIT 0.66, the sweep's two batch deletes 0.38 + 0.32; writer CPU 66%, deadlocks 0.27/min (the two deleters colliding on the same rows).

### Days to settle (from measured rates)

Inflow on the cleanup build: **~36k rows/h** (14.6k–21.5k queue rows/h plus their results, ~17 commands/h; 5k iPhones refetching hourly). Bytes per row ~959 (32.6 GB / 34M rows at the last sample). Rates below come from the recorder's exact hourly samples (net change + inflow), so they count **every deletion path**: the sweep's purge and the per-ack refetch cleanup together.

| Setting | Rows removed / h | Net vs inflow | 35M-row backlog settles in | Steady state |
|---|---|---|---|---|
| Default knobs (1,000 / 1,000), hourly, 10-min cap | 3,000 | −33k/h (grows) | never | — |
| Caps 500k, 10-min cap, hourly (as shipped) | ~60k (measured 10:12 run) | +24k/h | ~60 days | — |
| Caps 500k, 45-min cap, back to back (load-test build) | ~310k | +274k/h | **~5.3 days** | ~864k rows / 0.8 GB (one 24 h window of inflow) |
| No cleanup | 0 | −36k/h | — | +302 GB / year for this fleet |

Caveats: the 45-min figure required a code constant change and back-to-back triggering; at the shipped hourly cadence the same build would take ~4× longer (~23 days). The per-ack path contributed more than half of the removed rows in the 45-min phase; the sweep alone is slower. The synthetic backlog references nothing, so no candidates were vetoed — a real backlog with live profile/lock references drains more slowly.

## Steady state: top queries, baseline vs cleanup build

`compare-metrics.sh` baseline (seed build, ~1M mailbox rows, 09-28 14–15Z) → cleanup build at default knobs (35M rows, 09-29 07:14–09:24Z). Same 15k hosts, same profiles; the backlog size is the variable.

| Metric | Baseline (1M rows) | Cleanup build, default knobs (35M rows) |
|---|---|---|
| Writer CPU | 35.6% | 30.7% |
| Writer select latency | 0.6 ms | 22.3 ms |
| Writer insert latency | 2.9 ms | 8.2 ms |
| Writer read IOPS | ~0 | 281 (cold pages of the big tables) |
| `get next command` nano join (writer top SQL) | load 0.03, #2 | **load 1.37, #1** |
| Cleanup's own deletes (`nano_command_results`, `nano_enrollment_queue`) | — | 0.15 + 0.11, #3 and #4 |
| Reader CPU | 82.8% (r6g.large) | 29.4% (r6g.xlarge) |

Reading: the command-fetch query every device runs on check-in costs ~45× more against a 35M-row mailbox than against 1M rows — that is the customer-visible cost the cleanup exists to remove. The cleanup's own statements enter the writer's top 5 at roughly a fifth of that query's load and do not appear on the reader. Latency deltas are the table size, not the cleanup (default knobs delete 3k rows/h). Caveats: the baseline window carried the mock APNs pool-exhaustion storm (57k 5xx/h, ALB p99 18 s), so ALB and error rows are not comparable; the reader class differs between the windows.

## Incident: reader restart loop after the upgrade (not caused by the cleanup)

From 05:51Z the r6g.large reader fell behind the writer and Aurora restarted it every 3–4 min for 50 min; all 16 jobs of the 06:12 cleanups cron failed on `connection refused`. Writer load: `host_software_installed_paths` per-row `executable_hashes` rewrite (one-time wave after the 2026-09-21 migration, 15k hosts) + COMMIT volume; reader load: the Apple profile reconciler label-scoping query (main's drain loop). Resizing the reader to db.r6g.xlarge ended it (lag 2 min → 20 ms). Sizing note for 4.94: a reader at least one class above r6g.large for 15k MDM hosts.

## Other findings

- Default cap cannot keep up with 5k iPhones' refetch inflow (45k pairs/h in vs 1k/h out): knob-default recommendation.
- Drain throughput is bounded by the scan cap, not the knob (~350k pairs/h max): make `nanoCleanupMaxScansPerRun` configurable or higher.
- Mock APNs: Redis client pool exhaustion under a 15k reconnect storm.
- Recorder cost: exact `COUNT(*)` on the three tables takes ~3 min at 35M rows.

## Files

`494nanoclean15k-2026-09-28-150000Z-1h.*` baseline · `…-2026-09-29-020000Z-3h.*` overnight soak · `…-2026-09-29-061829Z-30m.*` incident · `…-2026-09-29-092425Z-130m.*` default knobs (3 runs) · `…-2026-09-29-102300Z-12m.*` first 500k-cap run (10-min cap) · `…-2026-09-29-114600Z-45m.*` first 45-min run · `…-2026-09-29-131800Z-138m.*` three 45-min runs. Demo replay page (private): https://claude.ai/artifact/6nCPpPRfPScPC8KCg3yjrw
