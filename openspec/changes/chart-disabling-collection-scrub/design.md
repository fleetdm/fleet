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

**Failure handling — fail closed.** If the orchestrator can't load
`AppConfig` or `ListTeams` for the tick, it does NOT fall back to
an unscoped collection. The disable feature's contract is that
disabled scopes stop accumulating new data; a fallback to "collect
everything" silently undoes that contract for the duration of the
config-load outage. The tick logs the error and returns it; the cron
will retry on its next interval (10m). An extended outage degrades
to "no collection" rather than to "collect data the operator just
disabled" — strictly the safer failure mode.

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

**Failure handling — log and continue after commit.** The enqueue is
just an `INSERT INTO jobs` against the same MySQL connection that
just successfully committed `SaveAppConfig` / `SaveTeam`. The failure
window is microscopic, but if it does fail, the call MUST NOT return
an error: `SaveAppConfig` / `SaveTeam` has already committed, the
`disabled_historical_dataset` activity has already been emitted, and
returning an error would manufacture a "partial-success but the API
says fail" UX. The client retries with the same payload, the diff is
now empty, and the scrub never enqueues — strictly worse than the
log-and-continue behavior. If a transient enqueue failure proves
load-bearing in production, the upgrade path is a transactional
outbox (write the job intent in the same transaction as the config
commit). Deferred until/unless observed.

Mapping public config key → internal dataset name at enqueue. The
public `historical_data` sub-keys are `uptime` and `vulnerabilities`
(matching the activity payloads admins see). The chart datasets are
named `uptime` and `cve` internally — `host_scd_data.dataset` stores
`"cve"`, not `"vulnerabilities"`. The scrub worker delegates straight
into the chart store's `dataset = ?` clause, so the job payload MUST
carry the internal name. `EnqueueHistoricalDataScrubs` does the
mapping inline (`vulnerabilities → cve`); activities continue to use
the public name.

**Dedup at enqueue — drop identical pending jobs.** A "wily QA"
scenario (rapid disable→enable→disable→… on the same scope) would
otherwise stack N redundant scrub jobs in the `jobs` table, each one
walking `host_scd_data` once. The walks are *nearly* idempotent (the
table is empty / the bits already cleared by the time later jobs
run), but the read I/O on a multi-million-row table is real. Before
inserting a new scrub job, the enqueuer checks for an existing job
with the same `name` and byte-equal `args`, in `state = 'queued'`. If
one exists, the new enqueue is dropped. Match criteria:

- **Same `name` AND byte-equal `args`** → drop. Go's `encoding/json`
  marshals struct fields in declaration order, so payloads produced
  by `EnqueueHistoricalDataScrubs` are deterministic and byte-equal
  comparison is sound without canonicalization.
- **Different `args`** (e.g. different `fleet_ids`) → keep. The
  scopes are different and both jobs need to run.
- **`state != 'queued'`** (a job is already running, completed, or
  failed) → keep. A running job started against an earlier snapshot
  of the table; data written between job-start and the new disable
  needs a fresh scrub to clean up. Completed/failed jobs don't gate
  future enqueues.

Race window: two concurrent enqueues that both observe "no pending"
and both insert produce one duplicate. The handlers are
near-idempotent, so the worst case is one extra walk — preferable to
adding a UNIQUE index to the shared `jobs` table for a microsecond
window.

Cross-supersession (a pending global-DELETE for `cve` makes any
incoming per-fleet `cve` scrub redundant) is a related but separate
optimization, deferred. Strict equality dedup captures the
QA-thrash case which is the dominant operational concern.

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
3. writeBatch = clamp(2_000_000 / max(len(mask), 1), 1, 200)
                ← target ~2 MB per UPDATE; cap at 200 rows to bound
                  parser/optimizer cost. Computed once per job.
4. lastID = 0
   loop:
     rows = SELECT id, host_bitmap FROM host_scd_data
            WHERE dataset = ? AND id > lastID
            ORDER BY id LIMIT 5000
     if len(rows) == 0: break

     // Compute new bitmaps in Go; drop rows the mask doesn't touch.
     pending = []  // (id, newBitmap)
     for r in rows:
       new = BlobANDNOT(r.host_bitmap, mask)
       if !bytes.Equal(new, r.host_bitmap):
         pending.append((r.id, new))
       lastID = r.id

     // Flush pending in writeBatch-sized CASE/WHEN UPDATEs.
     for chunk in chunks(pending, writeBatch):
       UPDATE host_scd_data
       SET host_bitmap = CASE id
           WHEN ? THEN ?
           WHEN ? THEN ?
           ...
       END
       WHERE id IN (?, ?, ...)
```

Two efficiency mechanics:

- **Skip no-op rows.** `BlobANDNOT(bitmap, mask)` returns the same
  bytes whenever the row contains no masked bits. For typical
  fleet-disable scrubs (one fleet, sparse mask), the majority of
  rows are unaffected — those rows generate no UPDATE at all. The
  win scales inversely with mask density.
- **Chunked CASE/WHEN UPDATEs.** Each affected row no longer costs
  one round trip; instead, up to `writeBatch` rows are written per
  statement. For a read-page of 5000 rows that all need updates and
  a 200-row write batch, that's 25 statements per page instead of
  5000. The IN-list is the same set of ids the CASE enumerates, so
  the planner uses the primary-key index for the row locks.

`fleet_ids` is always a slice in the payload: a single PATCH produces
`[fleet_id]` (length 1), a GitOps batch produces `[5, 7, 11]`. Same
handler. The single `IN` query and single mask mean the row walk
runs once for the dataset regardless of cardinality.

Read batch size (5000) and write batch cap (200) are starting
estimates; both tunable via package vars so tests can shrink them to
exercise multi-batch paths. The 2 MB per-statement target is a soft
ceiling well under MySQL's default 16–64 MB `max_allowed_packet`,
chosen to keep replication binlog events small.

**Reads on the primary.** Three reads in this change are read-then-
write and MUST run against the primary, not a replica:

1. The host-membership lookup (`HostIDsInFleets`) — its result builds
   the mask that drives the immediately-following `UPDATE`. Replica
   lag scrubs the wrong host set.
2. The row-walk paging `SELECT` in `ApplyScrubMaskToDataset` — the
   loop terminates on `len(rows) == 0`. Replica lag stops the loop
   while rows still exist on primary; the next disable won't
   re-enqueue, so those rows are silently retained.
3. The dedup gate in `HasQueuedJobWithArgs` (server/datastore/mysql/
   jobs.go) — its boolean result decides whether `NewJob` inserts.
   Replica lag (typically hundreds of ms, seconds under load) widens
   the dedup window from "microseconds" (Decision 5) to "tail of
   replication lag," letting the rapid disable→enable→disable thrash
   scenario the gate was built for slip through and stack duplicate
   jobs. Handlers are near-idempotent so the impact is extra DB I/O,
   not wrong results, but the dedup contract is materially weakened
   without primary routing.

Use `ctxdb.RequirePrimary(ctx, true)` for all three reads.

Considered: a single statement using MySQL's `BIT_AND` over BLOB —
not portable; MySQL's bit operators don't operate on BLOB element-
wise. Pulling rows into Go and writing back is the only practical
option.

Considered: row-by-row UPDATE keyed by id. Simpler code, but a 5000-
row read-page costs 5000 round trips; for a multi-million-row scrub
the network cost dominates the actual write. Rejected for
inefficiency now that we have to walk a unified table for every
dataset.

Considered: temp-table join (`UPDATE...JOIN tmp ON id`). Round trips
drop further (~2 per page), but you still ship every new bitmap over
the wire and now own DDL/cleanup for the temp table. Marginal
improvement over chunked CASE/WHEN at much higher complexity.
Rejected.

Considered: closing rows instead of mutating bitmaps. Mutating is
simpler (no schema change to support an "open" close marker for
already-closed rows), and the row's identity is stable.

### 7. No lock between scrub and collection cron

Once the disable commit lands, the cron stops writing the disabled
scope's bits — period. New rows / OR-merges into existing rows
contain only allowed-scope bits. So the scrub only needs to clean up
pre-disable state, and **no masked-scope (X) bit can ever be
re-introduced after the commit.** That invariant is what makes
lock-free coexistence possible.

Convergence on X bits:

- **Accumulate (uptime).** The cron only ORs in non-X bits; the
  scrub clears X bits. Whatever interleaving occurs, the final
  bitmap contains no X bits.
- **Snapshot (cve).** Cron closes an existing open row by setting
  `valid_to` and inserts a fresh open row whose bitmap is built
  without X. It does not mutate the `host_bitmap` column of any
  existing row. The scrub mutates `host_bitmap` of rows the cron is
  not concurrently rewriting. No interaction.

Residual race on **allowed-scope** bits (accumulate only):

The scrub is read-modify-write in Go (`SELECT bitmap` → compute
`bitmap &^ mask` → `UPDATE bitmap = computed`), not a single atomic
SQL `UPDATE bitmap = bitmap &^ mask` — MySQL doesn't have BLOB-wise
bitwise operators, so we can't push the ANDNOT down. That means an
ODKU-OR by the cron between the scrub's SELECT and UPDATE is
clobbered:

```
T1 scrub: SELECT bitmap = B
T2 cron:  ODKU       bitmap = B | newAllowed   (allowed-scope bits)
T1 scrub: UPDATE     bitmap = B &^ mask        ← drops newAllowed
```

The lost bits are *allowed-scope* bits cron just collected, not
masked-scope bits (those can't be written post-commit). For
accumulate datasets, the cron OR-merges the same bucket on every
subsequent tick within the bucket's lifetime, so the lost bits are
re-OR'd back in on the next tick. The only durably-lost case is the
race landing on the *final* tick of the bucket's lifetime — narrow
window, single-bucket impact, single-host-per-row granularity.

Snapshot datasets don't experience this race because the cron's
write path doesn't UPDATE `host_bitmap` of existing rows.

Trade-off accepted. Mitigation if real-world impact materializes:
wrap the per-row read-modify-write in an explicit transaction with
`SELECT ... FOR UPDATE`, or add a `version` column for optimistic
concurrency with retry. Both add lock scope or schema work for what
is presently a theoretical loss; deferred until/unless observed.

The `valid_from < startTime` partition I considered earlier turns
out to be unnecessary given the commit-before-enqueue invariant.
Documented here so a future reader doesn't reintroduce it
"defensively."

### 8a. Fold scrub enqueue into `OnHistoricalDataChanged`

Original split: `OnHistoricalDataChanged` emitted activities, the
caller then called `EnqueueHistoricalDataScrubs` after it returned.
Three callers (`ModifyAppConfig`, `ModifyTeam`, `editTeamFromSpec`),
each with the same paired-call shape:

```go
if err := fleet.OnHistoricalDataChanged(...); err != nil {
    return ctxerr.Wrap(...)              // FATAL
}
if err := fleet.EnqueueHistoricalDataScrubs(...); err != nil {
    svc.logger.ErrorContext(...)         // log-and-continue
}
```

Two problems with the split, both surfaced in PR review (CodeRabbit):

1. **Ordering trap.** Activity emission is fatal, scrub enqueue is
   log-and-continue, and they run in that order. If `NewActivity`
   fails on the first flipped sub-key, the call returns before
   enqueue, the config commit has already persisted the new state,
   and the retry sees `oldHD == newHD` and short-circuits. Both
   activity emit *and* scrub enqueue are silently dropped on retry.
   The disable guarantee — that `host_scd_data` rows for the
   disabled scope eventually go away — is lost without any error
   surfaced to the operator.

2. **Duplicated structure.** Both helpers iterate the same sub-key
   list looking for flips. Adding a dataset means updating two
   parallel switch tables and the public-key→internal-dataset
   mapping (`vulnerabilities` ↔ `cve`) implicit across both.

**Decision.** Fold the scrub enqueue into `OnHistoricalDataChanged`.
One function, one iteration of the sub-key list, one place to add
new datasets. For each flipped sub-key:

- **Disable flip (true→false):** enqueue scrub first, then emit the
  disabled activity. Scrub-first matters: an activity-emit failure
  must not skip the scrub. Both errors are collected and joined.
- **Enable flip (false→true):** emit the enabled activity. No scrub.

The merged function returns `errors.Join(errs...)`. Callers
log-and-continue on the joined error (uniform with how scrub
errors were already treated).

**Behavioral change vs. pre-merge: activity-emit failures are now
non-fatal.** Previously they returned a 500 to the API caller.
After: they log and the call succeeds. This is a deliberate priority
swap. Activity emission is audit-log emission; if it fails, the
operator can still see the config diff. Scrub failure to enqueue
means historical data persists indefinitely after the user disabled
collection, which is the contract this PR exists to uphold. And the
pre-merge fatal behavior was already broken for retries — see
problem (1) above. The merged behavior loses no recovery semantics
that the split version had.

Considered alternatives:

- **Keep the split, reorder enqueue before activity at every call
  site.** Fixes the immediate ordering bug, but the pairing
  invariant remains tribal — the next person to add a post-commit
  step has to know to put it after enqueue. Rejected; the structural
  fix is better.
- **Keep activity errors fatal in the merged function.** Possible
  via two error returns or an early-return-on-activity-error path,
  but it reintroduces the lose-the-scrub-on-activity-failure bug
  that motivated the merge. Rejected.
- **Move the merge into the EE service wrapper instead of the
  shared `fleet` package.** Would require duplicating the helper for
  the free-tier `ModifyAppConfig` caller. Rejected; the helper is
  already in `server/fleet/historical_data.go` and used by both
  tiers.

The combined interface (`HistoricalDataActivityEmitter` +
`HistoricalDataScrubEnqueuer`) is satisfied by the existing
service implementations (`fleet.Datastore` + service `NewActivity`).
No new mock surface area beyond what each helper already needed
separately.

This decision supersedes the four-step "save → activity → enqueue"
ordering described at the bottom of decision 5: post-commit, the
caller now invokes one helper, not two.

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
