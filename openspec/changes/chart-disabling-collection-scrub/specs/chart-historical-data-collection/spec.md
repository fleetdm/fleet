## ADDED Requirements

### Requirement: Cron skips datasets disabled globally

The collection cron orchestrator SHALL check
`AppConfig.Features.HistoricalData.Enabled(<dataset name>)` before
invoking each registered dataset's `Collect` method. If the result is
`false`, the orchestrator SHALL NOT call `Collect` for that dataset on
this tick. No new rows SHALL be written to `host_scd_data` for that
dataset, and no helper queries (e.g. `FindRecentlySeenHostIDs`,
`AffectedHostIDsByCVE`) SHALL run.

If the orchestrator cannot load `AppConfig` or the team list needed
to build the scope resolver, it SHALL fail the tick (log the error
and return) rather than fall back to unscoped collection. Falling
back to unscoped collection would silently undo the disable contract
for the duration of the config-load outage. The cron retries on its
next interval.

#### Scenario: Global flag is false for a dataset

- **WHEN** `appCfg.Features.HistoricalData.Vulnerabilities` is `false`
  and the cron tick fires
- **THEN** the orchestrator skips the CVE dataset entirely
- **AND** no rows are inserted, updated, or closed in `host_scd_data`
  for `dataset = 'cve'`

#### Scenario: Global flag is true for a dataset

- **WHEN** `appCfg.Features.HistoricalData.Uptime` is `true` and the
  cron tick fires
- **THEN** the orchestrator proceeds to mask construction and
  invokes `UptimeDataset.Collect`

#### Scenario: Scope-resolution failure halts the tick

- **WHEN** `AppConfig` or `ListTeams` returns an error while building
  the scope resolver
- **THEN** the orchestrator logs the error and returns it from the
  tick
- **AND** no `Collect` calls run on this tick (no datasets collected,
  including ones that would otherwise be enabled)
- **AND** the cron retries on its next scheduled interval

### Requirement: Cron filters per-fleet-disabled hosts via SQL push-down

For each globally-enabled dataset, the orchestrator SHALL compute a
`disabledFleetIDs []uint` slice — the set of team IDs where the
dataset's per-team flag is `false`. The slice SHALL be passed to
`Dataset.Collect`, which SHALL forward it to the relevant
`DatasetStore` query method. The SQL layer SHALL add
`AND (h.team_id IS NULL OR h.team_id NOT IN (?))` to the existing
collection query when the slice is non-empty.

Hosts with no team (`team_id IS NULL`) follow the global value
directly: when the global flag is `true`, no-team hosts are always
included regardless of `disabledFleetIDs` membership.

`disabledFleetIDs == nil` or `len(disabledFleetIDs) == 0` means
"no fleets are disabled for this dataset"; the SQL builder SHALL NOT
add the filter clause in that case, preserving today's exact query.

#### Scenario: Global on, all fleets on

- **WHEN** every team's `Features.HistoricalData.Uptime` is `true` and
  global is `true`
- **THEN** the orchestrator passes `disabledFleetIDs = nil` (or
  empty) to `Collect`
- **AND** the SQL query runs without any team filter clause
- **AND** rows written by this tick contain bits for every recently-
  seen host

#### Scenario: Global on, one fleet off

- **WHEN** team 7's `Features.HistoricalData.Uptime` is `false` and
  all other teams plus global are `true`
- **THEN** the orchestrator passes `disabledFleetIDs = [7]` to
  `Collect`
- **AND** the SQL query adds
  `AND (h.team_id IS NULL OR h.team_id NOT IN (7))`
- **AND** rows in `host_scd_data` for `dataset = 'uptime'` written by
  this tick contain no bits for hosts in team 7

#### Scenario: Per-dataset isolation

- **WHEN** team 7 has uptime enabled but vulnerabilities disabled
- **THEN** the uptime tick's `disabledFleetIDs` does NOT include 7
- **AND** the cve tick's `disabledFleetIDs` includes 7

#### Scenario: No-team hosts always included when global is on

- **WHEN** global is `true` for a dataset and `disabledFleetIDs`
  contains every team in Fleet
- **THEN** the SQL filter still allows `team_id IS NULL`
- **AND** no-team hosts contribute to the written bitmap

### Requirement: Dataset.Collect and DatasetStore signatures carry disabledFleetIDs

The `Dataset.Collect` method SHALL accept a `disabledFleetIDs []uint`
parameter:

```go
Collect(ctx context.Context, store DatasetStore, now time.Time, disabledFleetIDs []uint) error
```

The `DatasetStore` query methods SHALL accept the same parameter and
push it down into the SQL `WHERE` clause:

```go
FindRecentlySeenHostIDs(ctx context.Context, since time.Time, disabledFleetIDs []uint) ([]uint, error)
AffectedHostIDsByCVE(ctx context.Context, disabledFleetIDs []uint) (map[string][]uint, error)
```

A `nil` or zero-length slice means "no fleets disabled"; the SQL
SHALL NOT add a filter clause and the query SHALL behave exactly as
before. Implementations SHALL NOT post-process the result in Go to
apply a fleet filter — the filter MUST be applied at the SQL layer
so disabled-fleet rows are never read or transferred.

#### Scenario: Nil disabledFleetIDs is pass-through

- **WHEN** `Collect` is invoked with `disabledFleetIDs == nil`
- **THEN** the underlying SQL has no team filter clause
- **AND** every recently-seen host (uptime) or CVE-host pair (cve)
  contributes to the result

#### Scenario: Empty slice is pass-through

- **WHEN** `Collect` is invoked with `disabledFleetIDs == []uint{}`
- **THEN** behavior is identical to `nil`

#### Scenario: Populated slice excludes those teams' hosts at SQL

- **WHEN** `Collect` is invoked with `disabledFleetIDs == [5, 7]`
- **THEN** the SQL adds `AND (h.team_id IS NULL OR h.team_id NOT IN (5, 7))`
- **AND** the result set excludes every host whose `team_id` is 5 or 7
- **AND** no in-memory filtering happens in Go before
  `RecordBucketData` is called

### Requirement: Disable-flip enqueues an asynchronous scrub

The service layer SHALL enqueue a scrub job in Fleet's `jobs` table
whenever `features.historical_data.<sub-key>` transitions from
`true` to `false` — globally via `ModifyAppConfig`, per-fleet via
`ModifyTeam`, or via either GitOps apply path — after the
corresponding config commit completes successfully.

Jobs MUST be enqueued AFTER the relevant `SaveAppConfig` /
`SaveTeam` commit has succeeded. The service layer MUST NOT enqueue
a scrub for a no-op flip (already-disabled stays disabled, or PATCH
submits the same value back) and MUST NOT enqueue a scrub for a
`false → true` flip.

Enqueue failure after the config commit SHALL be logged but MUST NOT
be returned to the API caller as an error. `SaveAppConfig` /
`SaveTeam` has already committed and the disable activity has
already been emitted; surfacing an enqueue error to the client would
manufacture a partial-success-but-API-says-fail state, and a client
retry sees an empty diff and never re-enqueues. The activity is the
durable record that the disable happened; if the operator notices
stale data later, re-toggling the flag re-fires the enqueue.

The job-payload `dataset` field SHALL be the internal chart dataset
name (`uptime`, `cve`), not the public config sub-key
(`uptime`, `vulnerabilities`). The scrub worker forwards this value
straight into the chart store's `dataset = ?` clause, which matches
`host_scd_data.dataset`. The mapping happens at enqueue time so the
worker payload is stable and storage-aligned. Activities continue
to use the public sub-key.

Two job types are defined:

- `chart_scrub_dataset_global` — payload `{"dataset": "<name>"}`.
  Enqueued once per dataset whose global flag flipped to `false`.
- `chart_scrub_dataset_fleet` — payload
  `{"dataset": "<name>", "fleet_ids": [<ids...>]}`. The `fleet_ids`
  slice is always a list (never a scalar) so the worker handler can
  treat single-team and multi-team scrubs uniformly. The
  enqueue side MAY emit one job per (dataset, team) flip — see the
  Coalescing note below.

**Coalescing**: a future optimization MAY collapse multiple
`chart_scrub_dataset_fleet` enqueues for the same dataset within a
single batch operation (e.g. GitOps `ApplyTeamSpecs`) into one job
with a multi-element `fleet_ids` slice. The current implementation
emits one job per (dataset, team) flip and accepts the redundant
table walks; the worker's behavior is identical regardless of slice
length. Implementations MAY also dedup global vs per-team flips for
the same dataset within one call (the global DELETE subsumes
per-team scrubs); this is not currently required.

#### Scenario: Global flip to false

- **WHEN** an admin PATCHes `features.historical_data.uptime: false`
  via `ModifyAppConfig` and the previous value was `true`
- **THEN** `SaveAppConfig` commits first
- **AND** the existing `disabled_historical_dataset` activity is
  emitted
- **AND** exactly one `chart_scrub_dataset_global` job with payload
  `{"dataset":"uptime"}` is inserted into the `jobs` table

#### Scenario: Per-fleet flip to false (single team)

- **WHEN** an admin PATCHes a team with
  `features.historical_data.vulnerabilities: false` and the previous
  team value was `true`
- **THEN** `SaveTeam` commits first
- **AND** the existing scoped `disabled_historical_dataset` activity
  is emitted with the team's id and name
- **AND** exactly one `chart_scrub_dataset_fleet` job with payload
  `{"dataset":"cve","fleet_ids":[<team.id>]}` is inserted into the
  `jobs` table

#### Scenario: GitOps batch flips multiple fleets

- **WHEN** a GitOps apply submits team specs that disable the cve
  dataset on three previously-enabled teams (ids 5, 7, 11)
- **THEN** all three `SaveTeam` commits complete first
- **AND** at least one `chart_scrub_dataset_fleet` job is enqueued
  covering the disabled (dataset, team) flips. The current
  implementation emits one job per flip
  (`fleet_ids:[5]`, `fleet_ids:[7]`, `fleet_ids:[11]`); a future
  optimization MAY emit a single coalesced job
  (`fleet_ids:[5,7,11]`). Either is spec-compliant.
- **AND** three scoped `disabled_historical_dataset` activities are
  emitted (one per team, existing behavior)

#### Scenario: No-op PATCH

- **WHEN** an admin PATCHes the same `historical_data` values that
  are already stored
- **THEN** no scrub jobs are enqueued
- **AND** no activities are emitted (existing behavior)

#### Scenario: Re-enable after disable

- **WHEN** an admin disables a dataset, then re-enables it before any
  scrub job runs
- **THEN** the queued scrub job is NOT cancelled
- **AND** the scrub runs on its scheduled tick and removes
  pre-disable rows or bits per its handler logic

### Requirement: Global scrub handler deletes all rows for the dataset

The `chart_scrub_dataset_global` worker SHALL delete every row in
`host_scd_data` whose `dataset` column matches the payload, in
batches with `LIMIT 5000` per statement, looping until the affected
row count is zero.

Rows MUST be deleted regardless of `valid_to` state (closed or open
sentinel). The handler MUST NOT lock other queries for unbounded
durations — each `DELETE ... LIMIT 5000` statement is a separate
transaction.

#### Scenario: Successful global scrub

- **WHEN** the worker picks up a `chart_scrub_dataset_global` job
  with payload `{"dataset":"uptime"}`
- **THEN** `DELETE FROM host_scd_data WHERE dataset = 'uptime' LIMIT 5000`
  runs in a loop
- **AND** the loop terminates when affected rows = 0
- **AND** no rows with `dataset = 'uptime'` remain
- **AND** the job is marked successful

#### Scenario: Worker restart mid-scrub

- **WHEN** the worker process is killed mid-loop and the job is
  retried by the framework
- **THEN** the next run continues with whatever rows remain
- **AND** the eventual end state has no `dataset = '<name>'` rows

### Requirement: Per-fleet scrub handler ANDNOTs the fleets' host bits

The `chart_scrub_dataset_fleet` worker SHALL:

1. Resolve the current set of host IDs across every fleet in the
   payload's `fleet_ids` slice:
   `SELECT id FROM hosts WHERE team_id IN (<fleet_ids...>)`. If the
   result is empty (every listed fleet was deleted or all their
   hosts moved out), the handler MUST mark the job successful and
   return without further work (no-op).
2. Build a single mask via `chart.HostIDsToBlob(hostIDs)` covering
   the union of all listed fleets' current members.
3. Walk `host_scd_data` rows for the dataset in id-order with
   `LIMIT 5000` and `id > <last>` paging.
4. For each row, compute `chart.BlobANDNOT(host_bitmap, mask)` and
   `UPDATE host_scd_data SET host_bitmap = ? WHERE id = ?`.
5. Continue until the page is empty.

The handler MUST NOT delete rows. The handler MUST NOT acquire
table-level locks; per-row `UPDATE` statements rely on InnoDB row
locking. The row walk runs once per dataset regardless of how many
fleets are in `fleet_ids`.

Both the host-membership lookup (step 1) and the row-paging `SELECT`
(step 3) SHALL run against the primary database, not a replica.
Both feed an immediately-following write whose correctness depends
on the read result; replica lag could either scrub the wrong host
set or terminate the loop while rows still exist on the primary.

#### Scenario: Per-fleet scrub clears bits (single fleet)

- **WHEN** the worker picks up a `chart_scrub_dataset_fleet` job for
  `{dataset: "uptime", fleet_ids: [7]}`
- **AND** fleet 7 currently has hosts with IDs {12, 47, 99}
- **THEN** the mask has bits set at positions 12, 47, 99
- **AND** every row in `host_scd_data WHERE dataset = 'uptime'` is
  read, ANDNOT-applied, and rewritten
- **AND** no row's `host_bitmap` has bits set at positions 12, 47, 99
  after the job completes

#### Scenario: Per-fleet scrub clears bits (multiple fleets coalesced)

- **WHEN** the worker picks up a `chart_scrub_dataset_fleet` job for
  `{dataset: "cve", fleet_ids: [5, 7, 11]}`
- **AND** the union of those fleets' current hosts is {3, 12, 47, 99, 200}
- **THEN** the single mask has bits set at all five positions
- **AND** the row walk for `dataset = 'cve'` happens exactly once
- **AND** no row's `host_bitmap` retains bits at any of those
  positions after the job completes

#### Scenario: All listed fleets have no hosts at scrub time

- **WHEN** every fleet in `fleet_ids` was deleted or all their hosts
  moved out before the scrub ran
- **THEN** `SELECT id FROM hosts WHERE team_id IN (...)` returns
  zero rows
- **AND** the handler marks the job successful with no further DB
  writes
- **AND** the disable activity remains as the only durable record of
  the operation

#### Scenario: Bits for hosts that left a listed fleet pre-scrub

- **WHEN** a host was in fleet 7 at disable time but moved to fleet 9
  before the scrub ran (and 9 is not in `fleet_ids`)
- **THEN** the host's bit is NOT in the scrub mask
- **AND** the host's bit remains in any pre-disable rows of
  `host_scd_data` for that dataset
- **AND** this is the documented "best-effort" behavior

### Requirement: Scrub and collection cron run concurrently without locks

The system SHALL NOT acquire a global, table-level, or distributed
lock to coordinate scrub workers and the collection cron. The
correctness of concurrent execution depends on the
config-commit-before-job-enqueue invariant: every cron run after
the disable commit reads the new config and excludes the disabled
scope from new writes.

#### Scenario: Concurrent accumulate write and scrub on the same row

- **WHEN** the uptime cron's ODKU OR-merge runs against row R while
  the scrub worker is mid-ANDNOT on row R
- **THEN** InnoDB row-locks each statement independently, so neither
  read-modify-write tears at the byte level
- **AND** the cron's contribution contains no disabled-scope bits
  (because cron post-disable filters them out)
- **AND** the final state of row R has no disabled-scope bits,
  regardless of statement order

#### Scenario: Concurrent snapshot transition and scrub

- **WHEN** the cve cron observes the open row's bitmap differs from
  the new (no-disabled-scope) state, closes the open row, and opens
  a new one
- **AND** the scrub worker subsequently processes the now-closed row
- **THEN** ANDNOT removes the disabled-scope bits from the closed row
- **AND** the new open row never had disabled-scope bits to begin with

### Requirement: BlobANDNOT helper

`server/chart/blob.go` SHALL export a `BlobANDNOT(a, mask []byte) []byte`
helper. The result length SHALL equal `len(a)`. If `mask` is shorter
than `a`, it SHALL zero-extend (high bytes of `a` pass through
unchanged). If `mask` is longer than `a`, the excess bytes of `mask`
are ignored.

#### Scenario: Equal-length operands

- **WHEN** `a = [0xFF, 0x0F]` and `mask = [0x0F, 0xFF]`
- **THEN** `BlobANDNOT(a, mask) == [0xF0, 0x00]`

#### Scenario: Mask shorter than a

- **WHEN** `a = [0xFF, 0xFF, 0xFF]` and `mask = [0x0F]`
- **THEN** `BlobANDNOT(a, mask) == [0xF0, 0xFF, 0xFF]`

#### Scenario: Mask longer than a

- **WHEN** `a = [0xFF]` and `mask = [0x0F, 0xFF]`
- **THEN** `BlobANDNOT(a, mask) == [0xF0]`

#### Scenario: Empty inputs

- **WHEN** `a = nil` or `a = []byte{}`
- **THEN** `BlobANDNOT(a, mask) == nil`
