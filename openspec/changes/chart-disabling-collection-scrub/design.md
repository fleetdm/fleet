## Context

`chart-disabling-gitops-api` (complete) ships:

- `Features.HistoricalData.{Uptime, Vulnerabilities}` config bools.
- `HistoricalDataSettings.Enabled(dataset string) (bool, error)` mapping
  the internal dataset name (`uptime`, `cve`) to the config sub-key.
- `enabled_historical_dataset` / `disabled_historical_dataset` audit
  activities, emitted from `ModifyAppConfig`, `ModifyTeam`, and the
  GitOps apply paths whenever a sub-field flips.
- Defaults: both sub-fields default `true` for new and existing
  installs, with GitOps client-side injection to keep YAML round-trips
  benign.

What's missing: the cron doesn't read these flags, and disabling a
dataset doesn't drop its data. This change wires both behaviors. The
chart bounded context (`server/chart/`) is intentionally insulated
from `fleet/` types — it has its own datastore interface, its own
`api.Dataset` interface, its own bootstrap. So configuration must
reach the chart context via a narrow seam rather than a direct
`fleet.AppConfig` import.

The collection cron lives in `cmd/fleet/cron.go` and already has
`AppConfig` and team configs in scope (it's a top-level orchestrator).
That's the natural place to compute "which datasets to skip" and
"which hosts are allowed."

The disable-flip points (`ModifyAppConfig`, `ModifyTeam`, GitOps apply)
already diff old vs new HistoricalData to emit activities. The same
diff drives scrub-job enqueue.

`host_scd_data` is the unified storage table for chart datasets,
created by migration `20260423161823_AddHostSCDData`. Each row carries
a `host_bitmap MEDIUMBLOB` keyed by `(dataset, entity_id, valid_from)`.
Two write strategies — Accumulate (uptime: ODKU OR-merge into the
current bucket's row) and Snapshot (cve: per-entity open row, closed
on bitmap change). The scrub design must respect both.

## Goals / Non-Goals

**Goals:**

- Cron honors `Enabled(dataset)` globally and per-fleet for new writes.
- Disable-flip removes already-collected data: globally → DELETE; per-
  fleet → ANDNOT each row's `host_bitmap`.
- Scrub runs asynchronously (not in the API request path) and at most
  once per disable-flip per scope.
- Collection cron and scrub worker can run concurrently without locks
  and without losing or re-introducing scrubbed bits.

**Non-Goals:**

- Frontend UI (Advanced card, Fleet Settings, dashboard empty state).
- Snapshot-at-disable host-ID capture. Membership is resolved at
  scrub run time; rare races (host left fleet between disable and
  scrub) are accepted.
- Restore-on-re-enable. Re-enabling resumes new collection only.
- Cancellation of in-flight scrub jobs on rapid disable→re-enable.
- A "scrub completed" activity. The disable activity is the only
  user-visible event.
- Surfacing scrub state via the API.

## Decisions

### 1. Fleet-ID push-down on `Dataset.Collect` and `DatasetStore`

```go
type Dataset interface {
    // ... existing methods ...
    Collect(ctx context.Context, store DatasetStore, now time.Time, disabledFleetIDs []uint) error
}

type DatasetStore interface {
    FindRecentlySeenHostIDs(ctx context.Context, since time.Time, disabledFleetIDs []uint) ([]uint, error)
    AffectedHostIDsByCVE(ctx context.Context, disabledFleetIDs []uint) (map[string][]uint, error)
    // RecordBucketData unchanged.
}
```

Each implementation simply forwards `disabledFleetIDs` to its
store call — no per-dataset filtering logic, no AND step, no
post-processing. The SQL layer adds
`AND (h.team_id IS NULL OR h.team_id NOT IN (?))` to the existing
queries when the slice is non-empty:

- Uptime query already has `hosts h` as base — just adds the WHERE.
- CVE software-side query gains
  `JOIN hosts h ON h.id = hs.host_id` plus the WHERE.
- CVE OS-side query gains
  `JOIN hosts h ON h.id = hos.host_id` plus the WHERE.

`disabledFleetIDs == nil` or `len == 0` → no filter clause is
added; queries run exactly as before.

Considered alternatives:

- **Bitmap mask.** `Dataset.Collect(... allowedMask []byte)`,
  orchestrator builds an "allowed hosts" bitmap from a separate
  `HostIDsForChartScope` query, dataset implementations AND each
  per-bucket bitmap before writing. Initially proposed; rejected on
  closer analysis because: (a) building the mask requires its own
  full-hosts query per dataset per tick — it doesn't actually save
  I/O, just shifts it; (b) introduces a bitmap-length edge case
  (new hosts enrolled between mask-build and host-list query land
  at positions beyond the mask and get truncated for one bucket);
  (c) requires a new datastore method on the main Fleet datastore.
- **`AllowedHosts map[uint]struct{}`.** Same drawbacks as the mask
  plus a second pass per dataset. Rejected for the same reasons.
- **Reuse `HostIDsInTargets`.** The existing
  `Datastore.HostIDsInTargets` accepts a team-id list, but it's
  auth-scoped (requires a `TeamFilter`) and bundles labels +
  explicit host IDs. Heavy for a cron-internal lookup, and only
  relevant if we'd kept the mask approach. Not used.

Chosen: push-down. One query per dataset per tick, query result
already excludes disabled-fleet hosts, no in-memory post-processing,
no new datastore methods on the main Fleet datastore. The cost is
one parameter on each existing `DatasetStore` method and a small
SQL addition (one JOIN + one WHERE clause).

A `nil` or empty slice from the orchestrator means "no fleets are
disabled for this dataset" — the SQL builder skips the filter
clause entirely, preserving today's behavior.

### 2. Scope decision lives in `cmd/fleet/cron.go`, not chart context

```
cmd/fleet/cron.go (existing collection cron, simplified):

  appCfg, _ := svc.AppConfig(ctx)
  teams, _  := svc.ListTeams(ctx, fleet.TeamFilter{User: AdminUser})

  scopeFn := func(name string) (skip bool, disabledFleetIDs []uint) {
      if !appCfg.Features.HistoricalData.Enabled(name) {
          return true, nil
      }
      var disabled []uint
      for _, t := range teams {
          if !t.Config.Features.HistoricalData.Enabled(name) {
              disabled = append(disabled, t.ID)
          }
      }
      return false, disabled
  }

  chartSvc.CollectDatasets(ctx, now, scopeFn)
```

`CollectDatasets` accepts a per-dataset scope resolver. For each
registered dataset, the chart service calls the resolver:

- `(skip=true, _)` → orchestrator says global is off; do not call
  `Collect`.
- `(skip=false, nil/empty)` → call `Collect(ctx, store, now, nil)`;
  no fleets disabled.
- `(skip=false, [fleetIDs...])` → call
  `Collect(ctx, store, now, [fleetIDs...])`; SQL filter excludes those
  fleets.

Considered alternatives:

- **Inject a config provider into the chart service.** Chart context
  imports `fleet.Features` types or a narrow adapter — extra seam,
  more code, no benefit when the cron is already config-aware.
- **Read AppConfig directly from `Service.CollectDatasets`.** Mixes
  cross-context concerns into the chart bounded context.
- **Direct iteration in cron.go.** Orchestrator pulls
  `RegisteredDatasets()` and the store from the chart service, calls
  `dataset.Collect` directly. Simpler at the call site but exposes
  the chart service's iteration and store internals for what is
  effectively a config-injection concern.

Chosen: scope-resolver callback. Chart service stays the iteration
owner; orchestrator supplies config-aware scoping via a closure.
No new datastore methods on the main Fleet datastore —
`disabledFleetIDs` is derived from team configs that the
orchestrator already loads.

### 3. Per-dataset `disabledFleetIDs`, computed each cron tick

Disabled-fleet membership depends on `Enabled(name)` per team, which
differs per dataset. So uptime and cve get separate slices per cron
tick — derived from the same `teams` list with the dataset name
substituted. No extra DB round trip; team configs are already loaded.

Considered: union list shared across datasets. Rejected because the
union over-excludes for datasets where fewer fleets are disabled,
silently dropping fleets that should still collect.

### 4. Job framework: reuse Fleet's `jobs` table

Fleet's `jobs` table (in `server/datastore/mysql/jobs.go`) and worker
infrastructure already provides retries, backoff, and visibility
into pending work. New job types:

- `chart_scrub_dataset_global` with payload `{"dataset":"<name>"}`.
- `chart_scrub_dataset_fleet` with payload `{"dataset":"<name>","fleet_id":<id>}`.

Worker handlers are registered in `cmd/fleet/serve.go` (or wherever
worker registration lives today). Handlers call into a chart-context
function for the actual SQL work — keeping the bitmap walking logic
inside `server/chart/internal/mysql/` while the handler/orchestration
lives in `server/service/`.

Considered: a chart-bounded `chart_scrub_jobs` table with its own
cron. Rejected because the chart context is already coupled to Fleet
infrastructure for collection scheduling, the disable-trigger lives
in `server/service/`, and reusing the existing framework gets us
retries and operational visibility for free.

### 5. Trigger sequence — commit BEFORE enqueue, coalesced per-call

```
ModifyAppConfig / ModifyTeam / GitOps apply (single API call):

  1. Compute new HistoricalData
  2. Diff old vs new across all scopes touched by this call:
       globalDisables  = { dataset | global flipped true→false }
       fleetDisables   = map[dataset] []fleet_id
                           ( all (dataset, fleet) flipping true→false )
  3. Save{AppConfig|Team} commits — all in scope of this call
  4. Emit disabled_historical_dataset activities (existing behavior,
     one per flipped sub-field per scope)
  5. For each dataset in globalDisables:
       enqueue chart_scrub_dataset_global { dataset }
       drop fleetDisables[dataset]   ← per-call dedup; global subsumes
  6. For each remaining (dataset, fleetIDs) in fleetDisables:
       enqueue chart_scrub_dataset_fleet { dataset, fleet_ids }
       (one job covering all flipped fleets for this dataset in this call)
  7. For each newly-enabled: emit enabled activity (existing); no scrub
```

Strict ordering: every Save{AppConfig,Team} commit MUST complete
before scrub enqueue. Any cron run that starts after the commits
reads the new config and excludes the disabled scope from new writes.
Once that invariant holds, the scrub only has to clean up data
collected *before* the disable.

Per-call coalescing matters most for GitOps batch applies, where one
apply might disable cve on a dozen teams. Without coalescing, that's
12 jobs, 12 walks of `host_scd_data`, 12 row-by-row UPDATE loops over
the same data. With coalescing, it's 1 job, 1 walk, each row is
ANDNOTed against the union mask once. Same correctness, ~Nx less I/O.

Per-call dedup matters when admins flip both global and per-team in
the same operation (e.g., GitOps cleanup that ratchets cve fully off).
Without dedup, the per-team scrub does work the global DELETE
overwrites moments later. With dedup, only the global job runs.

Considered alternatives:

- **One job per (dataset, fleet) regardless of call boundary.**
  Simpler enqueue logic, much worse for batch operations. Rejected —
  the API-call boundary is already a natural transaction unit and the
  diff already happens there.
- **Coalesce across API calls (de-duplicate at the worker).** Workers
  would have to scan the jobs table for related pending entries,
  introducing locking/ordering complexity. Rejected; per-call
  coalescing captures the dominant case.
- **Enqueue-first, commit-second.** Rejected — a worker could pick up
  a job before the config-commit becomes visible to other sessions,
  run the scrub, and then the still-stale config-aware cron
  re-introduces bits.

### 6. Scrub handler: global = DELETE, fleet = ANDNOT walk

Global handler (`chart_scrub_dataset_global`):

```sql
-- in a loop until 0 rows affected
DELETE FROM host_scd_data WHERE dataset = ? LIMIT 5000
```

Per-fleet handler (`chart_scrub_dataset_fleet`):

```
1. hostIDs = SELECT id FROM hosts WHERE team_id IN (<fleet_ids...>)
   - if empty (every listed fleet was deleted, or all hosts moved out)
     → log no-op, mark complete.
2. mask = HostIDsToBlob(hostIDs)   ← single mask covering union of fleets
3. lastID = 0
   loop:
     rows = SELECT id, host_bitmap FROM host_scd_data
            WHERE dataset = ? AND id > lastID
            ORDER BY id LIMIT 5000
     if len(rows) == 0: break
     for r in rows:
       new = BlobANDNOT(r.host_bitmap, mask)
       UPDATE host_scd_data SET host_bitmap = ? WHERE id = ?
       lastID = r.id
```

`fleet_ids` is always a slice in the payload: a single PATCH produces
`[fleet_id]` (length 1), a GitOps batch produces `[5, 7, 11]`. Same
handler. The single `IN` query and single mask mean the row walk
runs once for the dataset regardless of cardinality.

Batch size (5000) is a starting estimate; tunable via constant.

Considered: a single statement using MySQL's `BIT_AND` over BLOB —
not portable; MySQL's bit operators don't operate on BLOB element-
wise. Pulling rows into Go and writing back is the only practical
option.

Considered: closing rows instead of mutating bitmaps. Mutating is
simpler (no schema change to support an "open" close marker for
already-closed rows), and the row's identity is stable.

### 7. No lock between scrub and collection cron

Once the disable commit lands, the cron stops writing the disabled
scope's bits — period. New rows / OR-merges into existing rows
contain only allowed-scope bits. So the scrub only needs to clean up
pre-disable state.

Concurrent scrub + cron on the same row converges:

- **Accumulate (uptime).** Scrub sets `bitmap = bitmap &^ mask`.
  Cron's ODKU-OR sets `bitmap = bitmap | new_no_X`. Either order, the
  final state has no X bits. InnoDB row-locks each statement so
  read-modify-write doesn't tear at the byte level.
- **Snapshot (cve).** Cron observes the open row's bitmap differs
  from the new (no-X) state, closes the open row, opens a new one
  without X. Scrub processes the now-closed row and ANDNOTs it. The
  new open row never had X bits.

The `valid_from < startTime` partition I considered earlier turns
out to be unnecessary given the commit-before-enqueue invariant.
Documented here so a future reader doesn't reintroduce it
"defensively."

### 8. New helper `BlobANDNOT`

```go
// BlobANDNOT returns a new blob equal to a with the bits set in mask cleared.
// Result length is len(a). If mask is shorter, it zero-extends (no bits cleared
// past mask's end, leaving high bits of a intact).
func BlobANDNOT(a, mask []byte) []byte {
    if len(a) == 0 { return nil }
    out := make([]byte, len(a))
    n := min(len(a), len(mask))
    for i := 0; i < n; i++ {
        out[i] = a[i] &^ mask[i]
    }
    if n < len(a) {
        copy(out[n:], a[n:])
    }
    return out
}
```

Test coverage: longer/shorter/equal-length mask, all-zero mask =
identity, all-ones mask of equal length = empty, nil inputs.

## Risks / Trade-offs

- **[Risk] Resolved-at-scrub fleet membership misses hosts that left
  the fleet before scrub ran.** Their bits remain in `host_scd_data`.
  Privacy-wise, those bits now "belong" to whatever fleet the host
  moved into; the chart's read-side host filter scopes display by
  current team, so the bits don't leak into the disabled fleet's
  view. Accepted as best-effort. Documented.
- **[Trade-off] Re-enable does not restore data.** An admin who
  disables and immediately re-enables loses pre-disable history.
  Documented in the YAML + REST docs as part of the feature contract.
  No cancellation path means the operational policy is "if you
  toggled by mistake, bits may be gone."
- **[Trade-off] Scrub runs at job-worker cadence, not immediately.**
  There is a window after disable where stale bits exist. The chart
  read path's host filter scopes results to currently-visible hosts,
  so this window doesn't display disabled-fleet data — but the bits
  remain on disk. Not a privacy issue per the read-path filter, but
  worth noting that "disabled" in the API does not synchronously mean
  "deleted on disk."
- **[Risk] Large global scrubs (millions of rows) generate replication
  lag.** Mitigated by `LIMIT 5000` per statement and looping. If the
  job is interrupted mid-run (worker restart), it resumes from the
  remaining rows on the next tick — `DELETE` is naturally idempotent.
- **[Risk] Worker registration lives outside the chart bounded
  context.** The handler imports `server/chart/internal/mysql/` (or a
  small forwarding shim from `server/chart/api/`) to run the actual
  bitmap walk. Keeps the SQL co-located with the rest of `host_scd_data`
  access while the orchestration lives where existing job workers do.
- **[Trade-off] No "scrub completed" activity.** Admins do not get a
  follow-up event. If the scrub fails repeatedly, observability is via
  the jobs table / worker error logs, not via the activity feed.
  Acceptable for v1; can revisit if support burden materializes.
