# Chart bounded context

Time-series charting for the Fleet dashboard. Given a metric like "how many hosts
were online" or "how many hosts were affected by a critical CVE," this package
records a per-hour history and serves it back as bucketed data points the frontend
renders as a line, checkerboard, etc.

This is a **self-contained bounded context** (like `server/activity/` and
`server/mdm/`), not the traditional `server/fleet` → `server/service` →
`server/datastore` layering. It owns its own types, datastore, service, HTTP
transport, and bootstrap, and it must not import `server/fleet` or
`server/contexts/viewer` directly — an architecture test enforces this (see
[Dependency rules](#dependency-rules)).

## Table of contents

- [Mental model](#mental-model)
- [Package layout](#package-layout)
- [Dependency rules](#dependency-rules)
- [Data model: SCD type-2 + bitmaps](#data-model-scd-type-2--bitmaps)
- [Sample strategies](#sample-strategies)
- [The write path (collection)](#the-write-path-collection)
- [The read path (GetChartData)](#the-read-path-getchartdata)
- [Scoping, config gating, and scrubbing](#scoping-config-gating-and-scrubbing)
- [How to add a new dataset](#how-to-add-a-new-dataset)
- [How charts reach the frontend](#how-charts-reach-the-frontend)
- [Testing](#testing)
- [File reference](#file-reference)

## Mental model

A **dataset** answers one question over time ("uptime," "cve"). On a schedule, a
collection cron asks each dataset to **collect** the current state: it produces a
set of host IDs (optionally grouped by an *entity*, e.g. one host-set per CVE) and
hands them to the datastore as a [roaring bitmap](https://roaringbitmap.org/). The
datastore folds that observation into a slowly-changing-dimension (SCD) history in
the `host_scd_data` table.

On read, a chart request resolves which hosts the caller may see (a "filter mask"
bitmap), walks the SCD history bucket by bucket, ANDs each bucket's host-set
against the mask, and returns the population count per bucket. Everything is set
math on bitmaps; a "value" on the chart is always a distinct-host count.

```
collection cron (cmd/fleet/cron.go)
  └─ Service.CollectDatasets(scope)
       └─ Dataset.Collect()  ──reads hosts──▶ DatasetStore (FindOnlineHostIDs, AffectedHostIDsByCVE, …)
            └─ store.RecordBucketData(strategy, map[entity]*bitmap)  ──writes──▶ host_scd_data

HTTP GET /api/_version_/fleet/charts/{metric}
  └─ getChartDataEndpoint ─▶ Service.GetChartData()
       ├─ ViewerProvider.ViewerScope()      → team scope for authz + data
       ├─ authz.Authorize()                 → fail closed
       ├─ GetHostIDsForFilter() → mask      → cached per filter (hostFilterCache)
       └─ GetSCDData(...) walks buckets, AND mask, popcount ─▶ []DataPoint
```

## Package layout

| Path | Role | May depend on |
|------|------|---------------|
| `server/chart` (root) | Bitmap helpers (`blob.go`), `Dataset` implementations (`datasets.go`), shared constants | `api` only |
| `server/chart/api` | **Public surface.** `Service`, `Dataset`, `DatasetStore`, `ViewerProvider` interfaces; request/response types | nothing in Fleet |
| `server/chart/api/http` | HTTP request/response DTOs (wire tags, e.g. `fleet_id`) | `api` only |
| `server/chart/internal/types` | Internal `Datastore` interface, `HostFilter` | `api` only |
| `server/chart/internal/mysql` | MySQL `Datastore` implementation, SQL, SCD read/write | chart, types, platform |
| `server/chart/internal/service` | `Service` implementation, authz, host-filter cache, bucket math | chart, platform |
| `server/chart/bootstrap` | `New(...)` wires datastore + service + routes; entry point for `serve.go` | mysql, service, api, platform |
| `server/chart/internal/testutils` | Test helpers | — |

External code only ever touches `bootstrap.New` and the `api` package. The
anti-corruption layer that bridges to legacy Fleet types lives **outside** this
tree in `server/acl/chartacl` (the only place that imports both chart `api` and
`server/contexts/viewer`).

## Dependency rules

`arch_test.go` runs `archtest` assertions on every package. The rules in plain
English:

- `api` has **zero** Fleet dependencies (it's the contract).
- `api/http`, `internal/types`, and the root `chart` package depend on **`api`
  only**.
- `internal/mysql` and `internal/service` may additionally use `server/platform/...`
  and `server/contexts/...`, plus other chart packages.
- **No chart package may import `server/fleet` or `server/contexts/viewer`.**

If you need something from legacy Fleet (the current user, a fleet config value),
do not import it — add a narrow interface to `api` (see `ViewerProvider`) and
implement it in `server/acl/chartacl`. Then wire it through `bootstrap.New`.

## Data model: SCD type-2 + bitmaps

Everything lives in one table, `host_scd_data`:

```sql
CREATE TABLE host_scd_data (
  id            bigint unsigned AUTO_INCREMENT,
  dataset       varchar(50)   NOT NULL,             -- "uptime", "cve", …
  entity_id     varchar(100)  NOT NULL DEFAULT '',  -- "" for single-dimension; CVE id for cve
  host_bitmap   mediumblob    NOT NULL,             -- serialized host-id set
  valid_from    datetime      NOT NULL,
  valid_to      datetime      NOT NULL DEFAULT '9999-12-31 00:00:00',  -- sentinel = "still open"
  encoding_type tinyint       NOT NULL DEFAULT 0,   -- 0 = dense, 1 = roaring
  PRIMARY KEY (id),
  UNIQUE KEY uniq_entity_bucket (dataset, entity_id, valid_from),
  KEY idx_dataset_range  (dataset, valid_from, valid_to),
  KEY idx_valid_to_dataset (valid_to, dataset, entity_id)
);
```

A **row is a host-set that was valid over `[valid_from, valid_to)`**. The
`9999-12-31` sentinel means "currently open." This is a textbook
[slowly-changing-dimension type-2](https://en.wikipedia.org/wiki/Slowly_changing_dimension#Type_2:_add_new_row)
table: state changes append a new row and close the old one rather than mutating
in place, so history is preserved.

`entity_id` is the sub-dimension. Single-dimension datasets (uptime) use the empty
string. Multi-dimension datasets (cve) write one row per entity (per CVE) and the
read path ORs across entities to get a distinct-host union.

### Bitmap encoding (`blob.go`)

`host_bitmap` stores a set of host IDs. Two on-disk formats, discriminated by
`encoding_type`:

- **`EncodingDense` (0)** — legacy raw bit-array, `bit n set ⇔ host n in set`. Only
  read, never written anymore. Old rows decode transparently and age out via
  retention.
- **`EncodingRoaring` (1)** — portable [RoaringBitmap](https://github.com/RoaringBitmap/roaring)
  serialization. **All new writes use this.**

Two in-memory representations, and the distinction matters:

- **`Blob{Bytes, Encoding}` — storage form.** Only at the DB I/O boundary. Built by
  `HostIDsToBlob` / `BitmapToBlob`; consumed by INSERT/UPDATE.
- **`*roaring.Bitmap` — op form.** Everything else: all set math
  (`BlobAND/OR/ANDNOT`, `BlobPopcount`) and change detection. Built by `NewBitmap`
  or `DecodeBitmap`.

Encoding-awareness is confined to `DecodeBitmap` (storage→op) and `BitmapToBlob`
(op→storage). The rest of the code works in op form and never thinks about bytes.
Change detection compares op-form bitmaps with `roaring.Equals` — never bytes,
because a dense row and a roaring row can represent the same set with different
bytes.

## Sample strategies

A dataset declares a `SampleStrategy` that governs how observations combine within
a write-bucket and how rows collapse across buckets. **All collectors write at 1h
granularity** regardless of the *display* resolution requested at read time.

### `SampleStrategyAccumulate`

*"hosts observed doing the thing at any point during the bucket."* Used by
**uptime**.

- **Write:** every row is born *closed* (`valid_to = bucketStart + bucketSize` at
  insert). Repeated samples within the same bucket OR-merge into the existing row
  (ODKU on `uniq_entity_bucket`). A sample in a new bucket starts a fresh row. No
  explicit close step, no cross-bucket collapse.
- **Read:** a bucket's value = OR of every row whose interval overlaps the bucket.

### `SampleStrategySnapshot`

*"state as of the end of the bucket."* Used by **cve**.

- **Write:** rows align to 1h boundaries. The latest sample in a write-bucket
  overwrites via ODKU (last-write-wins). Across buckets, **unchanged** state keeps
  the row open (`valid_to` stays sentinel); a **changed** sample closes the prior
  row at the new boundary and opens a new one. An entity that disappears from the
  input has its open row closed.
- **Read:** for each entity, pick the row active at `bucketEnd`, then OR across
  entities.

> **Snapshot collectors must call `RecordBucketData` even with an empty map.** An
> empty input is meaningful — it means "no entities are in the tracked state right
> now," which must close any still-open rows. Accumulate short-circuits on empty
> (nothing to merge, no state to reconcile). See the comments in `datasets.go`
> (`CVEDataset.Collect`) and `data.go` (`RecordBucketData`).

The read-side aggregation for both strategies lives in `aggregateBucket`
(`internal/mysql/data.go`).

## The write path (collection)

1. A cron (`newChartDataCollectionSchedule` in `cmd/fleet/cron.go`, default 1h)
   calls `Service.CollectDatasets(ctx, now, scope)`.
2. `scope` is a `CollectScopeFn` built fresh each tick from AppConfig + team
   configs (`buildChartScopeResolver`). For each dataset it returns `(skip,
   disabledFleetIDs)` — whether the dataset is globally off, and which fleets opted
   out.
3. For each registered dataset, `Collect(ctx, store, now, disabledFleetIDs)` runs.
   A failure is logged and the loop continues — one dataset can't block the others.
4. `Collect` reads host state through the narrow `DatasetStore` interface
   (`FindOnlineHostIDs`, `AffectedHostIDsByCVE`, `TrackedCriticalCVEs`), builds
   `map[entityID]*roaring.Bitmap`, and calls `store.RecordBucketData(...)` with its
   strategy.
5. `RecordBucketData` dispatches to `recordAccumulate` or `recordSnapshot`, which
   serialize via `BitmapToBlob` and upsert.

Retention: a separate cleanup cron calls `CleanupData(days)` →
`CleanupSCDData`, which deletes *closed* rows older than the cutoff in batches.
Open rows (sentinel `valid_to`) are never deleted.

## The read path (`GetChartData`)

`internal/service/service.go::GetChartData` is the heart of the read side:

1. **Resolve scope.** `ViewerProvider.ViewerScope(ctx)` returns `(isGlobal,
   teamIDs)`. Fails closed if there's no viewer (requests sit behind authenticated
   middleware; absence means misconfiguration).
2. **Authorize.** Explicit `team_id` → `Host{TeamID}` + `ActionRead` (Rego enforces
   team-role match). No `team_id` → `Host{}` + `ActionList` (global users pass;
   team users are scoped by data below).
3. **Validate** metric exists, `1 ≤ days ≤ 31`, resolution is 0 or a positive
   divisor of 24.
4. **Build the filter mask.** `effectiveTeamIDs` collapses the team scope, then
   `GetHostIDsForFilter` resolves team/label/platform/include/exclude into a host-id
   list → `NewBitmap`. This is memoized per canonicalized filter by `hostFilterCache`
   (60s TTL, singleflight-collapsed). The mask encodes "currently visible hosts,"
   which incidentally drops hosts deleted since the SCD rows were written.
5. **Walk buckets.** `GetSCDData` selects every row overlapping the range,
   decodes each once, then for each bucket: `aggregateBucket` (per strategy) → AND
   the mask → `popcount` → one `DataPoint`. Zero buckets are emitted as `0`, not
   omitted.
6. **Respond** with metric, visualization, `TotalHosts` (popcount of the mask),
   resolution label, applied filters, and the data points.

Bucket boundaries are aligned to the client's local time via `tz_offset`
(`computeBucketRange`), so an "hourly" or "daily" chart lines up with the user's
day.

### Key invariant: nil vs empty

Several layers depend on the difference between a `nil` slice and an empty non-nil
slice. **Do not "normalize" one to the other.**

- `HostFilter.TeamIDs`: `nil` = no team filter (all hosts); `[]uint{}` = team user
  with zero teams → SQL emits `1=0` (see nothing); `[]uint{0}` = no-team hosts
  (`team_id IS NULL`).
- `GetSCDData` `entityIDs`: `nil` = match every entity; `[]uint{}` non-nil = match
  nothing (zero-valued buckets), avoiding an `IN ()` syntax error.
- `TrackedCriticalCVEs` returns a non-nil empty slice when nothing matches so the
  caller can tell "filter resolved to empty" from "no filter."

## Scoping, config gating, and scrubbing

Whether a dataset collects at all is gated by `HistoricalDataSettings` in AppConfig
(global) and per-team config (`Features.HistoricalData`). The cron's scope resolver
translates these into `skip` / `disabledFleetIDs`.

When an admin **disables** a dataset, already-collected data must be removed. That
flip is handled by `fleet.OnHistoricalDataChanged` (in `server/fleet/historical_data.go`),
which enqueues a worker job *after* the config commit:

- **Global disable** → `chart_scrub_dataset_global` job →
  `Service.ScrubDatasetGlobal` → `DeleteAllForDataset` (batched delete of all rows
  for the dataset).
- **Per-fleet disable** → `chart_scrub_dataset_fleet` job →
  `Service.ScrubDatasetFleet` → resolve the fleets' host IDs into a mask, then
  `ApplyScrubMaskToDataset` walks every row and `BlobANDNOT`s the mask out. Both are
  idempotent.

The worker jobs live in `server/worker/chart_scrub.go`. Note the dataset-name
strings (`"uptime"`, `"cve"`) are mirrored in three places — the `Dataset.Name()`
return, the scrub job payloads, and `OnHistoricalDataChanged`'s change list. They
must stay in sync, since `host_scd_data.dataset` is the join key for all of it.

## How to add a new dataset

Worked example: a "battery health" dataset showing how many hosts had a healthy
battery.

1. **Implement `api.Dataset`** in `server/chart/datasets.go`:

   ```go
   type BatteryDataset struct{}

   func (b *BatteryDataset) Name() string                       { return "battery" }
   func (b *BatteryDataset) DefaultResolutionHours() int        { return 24 }
   func (b *BatteryDataset) SampleStrategy() api.SampleStrategy { return api.SampleStrategySnapshot }
   func (b *BatteryDataset) DefaultVisualization() string       { return "line" }

   func (b *BatteryDataset) Collect(ctx context.Context, store api.DatasetStore, now time.Time, disabledFleetIDs []uint) error {
       hostIDs, err := store.FindHealthyBatteryHostIDs(ctx, disabledFleetIDs) // new store method
       if err != nil {
           return err
       }
       bucketStart := now.UTC().Truncate(time.Hour)
       // Snapshot: always record, even when empty, so open rows close.
       return store.RecordBucketData(ctx, b.Name(), bucketStart, time.Hour, b.SampleStrategy(),
           map[string]*roaring.Bitmap{"": chart.NewBitmap(hostIDs)})
   }
   ```

   Pick the strategy deliberately: *accumulate* for "seen doing X at any point"
   (uptime-like), *snapshot* for "in state X as of now" (inventory-like). Use a
   non-empty `entity_id` only if you need a sub-dimension you'll OR across (like
   per-CVE).

2. **Add the collection query** to the store. Define the method on **both**
   `api.DatasetStore` (`api/chart.go`) and `internal/types.Datastore`
   (`internal/types/chart.go`) — `Collect` only sees `DatasetStore`, but the
   concrete MySQL type must satisfy `types.Datastore` — then implement it in
   `internal/mysql/charts.go`. Keep these read-only and bounded; stream large joins
   (see `streamCVEHostPairs`).

3. **Register the dataset** in `cmd/fleet/serve.go::createChartBoundedContext`:

   ```go
   chartSvc.RegisterDataset(&chart.BatteryDataset{})
   ```

   An unregistered metric returns a 400 from `GetChartData`.

4. **Wire config gating** (if the dataset is opt-in/out): add a sub-key to
   `HistoricalDataSettings`, teach `Enabled(name)` about it, and add a row to the
   change list in `fleet.OnHistoricalDataChanged` so disabling it triggers a scrub.
   The scrub workers are dataset-agnostic and need no changes.

5. **Test** the collector and the read aggregation. MySQL-backed tests live in
   `internal/mysql/*_test.go`; service-level behavior in `internal/service/*_test.go`.

No migration is needed — new datasets reuse `host_scd_data`, keyed by the new
`dataset` string. A migration is only needed if you change the table shape (e.g. a
new column or index).

## How charts reach the frontend

- Route: `GET /api/_version_/fleet/charts/{metric}` (registered in
  `internal/service/handler.go`).
- Query params: `days`, `resolution` (hours), `tz_offset` (minutes, from JS
  `Date.getTimezoneOffset()`), `fleet_id` (note the teams→fleets rename — the wire
  name is `fleet_id`, the Go field stays `TeamID`), `label_ids`, `platforms`,
  `include_host_ids`, `exclude_host_ids` (comma lists).
- The response carries `visualization` (from `DefaultVisualization()`), so the
  frontend learns how to render each metric from the backend rather than hardcoding
  it.

## Testing

```bash
# Fast, no external deps (bitmap helpers, arch test):
go test ./server/chart/...

# MySQL-backed datastore + service tests:
MYSQL_TEST=1 go test ./server/chart/...

# A single test:
MYSQL_TEST=1 go test -run TestName ./server/chart/internal/mysql/...
```

`arch_test.go` is part of `go test ./server/chart/...` and will fail the build if a
package grows a forbidden dependency (e.g. an accidental `server/fleet` import).
When you add a store method, run `go test ./server/service/` too — uninitialized
mocks elsewhere can crash if an interface method is missing.

## File reference

| File | What's in it |
|------|--------------|
| `blob.go` | Bitmap encode/decode, storage-form vs op-form, set ops |
| `datasets.go` | `UptimeDataset`, `CVEDataset` — the `Dataset` implementations |
| `api/service.go` | `Service`, `ViewerProvider`, `CollectScopeFn` |
| `api/chart.go` | `Dataset`, `DatasetStore`, `SampleStrategy`, request/response types |
| `api/http/types.go` | HTTP wire DTOs |
| `internal/types/chart.go` | `Datastore` interface, `HostFilter` (nil/empty semantics) |
| `internal/service/service.go` | `GetChartData`, scope/authz, scrub, bucket range math |
| `internal/service/host_cache.go` | Per-filter mask cache (TTL + singleflight) |
| `internal/service/handler.go` | Route registration + endpoint decode |
| `internal/mysql/data.go` | SCD read/write: `RecordBucketData`, `GetSCDData`, cleanup, scrub |
| `internal/mysql/charts.go` | Host-filter SQL, online-host query, CVE collection + tracked-CVE filter |
| `bootstrap/bootstrap.go` | `New(...)` — wires the context together |
| `arch_test.go` | Enforces the dependency rules above |

Related code outside this tree:

- `server/acl/chartacl/` — anti-corruption layer (viewer adapter).
- `cmd/fleet/serve.go` — `createChartBoundedContext`, dataset registration.
- `cmd/fleet/cron.go` — collection schedule, scope resolver, cleanup.
- `server/worker/chart_scrub.go` — scrub worker jobs.
- `server/fleet/historical_data.go` — config-flip → scrub/activity orchestration.
- `server/datastore/mysql/migrations/tables/20260423161823_AddHostSCDData.go` — the table migration.
- `tools/charts-backfill/`, `tools/charts-collect/` — dev tools (see below).

## Dev tools: populating chart data

In dev you usually don't have hours of real collection history to chart. Two
standalone tools under `tools/` write directly to `host_scd_data` so you have
something to render. Each has its own README with the full flag reference; this is
the orientation.

### `tools/charts-backfill` — synthetic history

Generates **fake but realistically-shaped** history for a dataset. This is the one
you want for frontend work or eyeballing a chart — point it at your local DB and it
fabricates days of data in seconds. Safe to re-run (ODKU merge), always writes
roaring encoding.

Crucially, it backfills in the mode that **matches the dataset's
`SampleStrategy`** so the data looks like what production would eventually produce:

- **Accumulate datasets** (uptime) → 24 independent hourly rows per day,
  each a fresh random sample, each bounded to its single hour.
- **Snapshot datasets** (cve) → per-entity state-segment rows: most entities get one
  long open row, a small fraction "flip" on day boundaries to produce closed
  segments. CVE cardinality follows a long-tail distribution (most CVEs touch a
  handful of hosts, a few are browser/kernel-wide) so unioning hundreds of entities
  doesn't saturate at fleet size. The final segment per entity stays open
  (`valid_to` = sentinel) so the live collector compares against it on its next
  tick rather than stacking a row on top.

```bash
# 30 days of uptime for all hosts in the local DB:
go run ./tools/charts-backfill --dataset uptime --days 30

# CVE data using the same CVE set production would track:
go run ./tools/charts-backfill --dataset cve --days 30 --use-tracked-cves

# Scope to specific hosts / entities / a custom DSN:
go run ./tools/charts-backfill --dataset uptime --days 7 --host-ids 1,2,3
go run ./tools/charts-backfill --dataset cve --days 30 --entity-ids CVE-2024-1,CVE-2024-2
go run ./tools/charts-backfill --mysql-dsn "fleet:fleet@tcp(localhost:3306)/fleet"
```

Key flags: `--dataset` (default `uptime`), `--days` (30), `--start-date`
(`YYYY-MM-DD`, defaults to `now - days`), `--host-ids` (default: all hosts),
`--entity-ids`, `--use-tracked-cves` (cve only — auto-discovers entity IDs via the
production tracked-CVE query; needs vuln data populated), `--mysql-dsn`. Full table
in `tools/charts-backfill/README.md`.

If you add a snapshot-strategy dataset, add its name to `snapshotDatasets` in
`tools/charts-backfill/main.go` (and a density range in `densityRange`) so the tool
generates it in the right shape; otherwise it defaults to the accumulate/hourly
model.

### `tools/charts-collect` — real data from a live Fleet

The other tool pulls **real** state from a running Fleet instance over the REST API
(currently uptime + CVE) and writes it into a local DB. It's the out-of-process
stand-in for the in-server collection cron — designed to run hourly against e.g.
dogfood. Use this when you want a chart backed by real fleet data rather than
synthetic noise.

```bash
go run ./tools/charts-collect --fleet-url https://dogfood.fleetdm.com --fleet-token <token>
```

Targets are also configurable via `FLEET_URL` / `FLEET_TOKEN` / `MYSQL_DSN`, and
the DSN falls back to the standard `FLEET_MYSQL_*` env vars. See
`tools/charts-collect/README.md`.

> Both tools duplicate a few storage constants (the `9999-12-31` open sentinel,
> upsert batch size, roaring encoding) because they write `host_scd_data`
> out-of-process and can't import the internal mysql package. If you change the
> storage format or those constants in `internal/mysql/data.go`, update the tools to
> match.
