## Why

Issue #44077 ("Allow disabling data collection for charts") gives admins
per-dataset switches under `features.historical_data`. The companion change
`chart-disabling-gitops-api` ships the config struct, the `Enabled(dataset)`
helper, the GitOps/API/PATCH plumbing, and the audit activities — but
nothing reads those flags yet. Setting `vulnerabilities: false` today has
no effect on collection or storage.

This change wires the consumer side: the collection cron honors the flags
(skipping disabled datasets globally and excluding disabled fleets'
hosts), and a flip from enabled→disabled triggers an asynchronous scrub
that removes the dataset's existing rows. Privacy is the motivation —
"disable" must mean "stop collecting AND remove what's there," not just
"stop collecting going forward."

## What Changes

### Cron gating

- `Dataset.Collect` gains a `disabledFleetIDs []uint` argument.
  Implementations forward it to their `DatasetStore` query, which
  pushes the filter down into SQL via
  `AND (h.team_id IS NULL OR h.team_id NOT IN (?))`. `nil` or empty
  slice = no scoping (query unchanged).
- `DatasetStore.FindRecentlySeenHostIDs` and
  `DatasetStore.AffectedHostIDsByCVE` gain the same parameter.
- The collection cron orchestrator (in `cmd/fleet/cron.go`, where
  `AppConfig` and team configs are already in scope) supplies a
  per-dataset scope resolver:
  - If `globalCfg.Features.HistoricalData.Enabled(name)` is `false` →
    skip the dataset entirely (no `Collect` call).
  - Else, derive `disabledFleetIDs` from team configs: every team
    whose `Enabled(name)` is `false`.
  - No additional DB round trips — team configs are already loaded.
- `UptimeDataset.Collect` and `CVEDataset.Collect` simply forward the
  slice to their store call; no per-dataset filtering logic runs in
  Go.

### Data scrub

- New worker job types:
  - `chart_scrub_dataset_global` — payload `{dataset string}`. Handler:
    `DELETE FROM host_scd_data WHERE dataset = ?` in batches with `LIMIT N`.
  - `chart_scrub_dataset_fleet` — payload `{dataset string, fleet_ids []uint}`.
    Handler resolves current fleet membership at run time
    (`SELECT id FROM hosts WHERE team_id IN (?)`), builds one mask
    covering all listed fleets, and walks `host_scd_data` rows for the
    dataset in id-order batches, rewriting each `host_bitmap` to
    `bitmap &^ mask` via UPDATE.
- One fleet scrub job per dataset per API call, regardless of how many
  teams flipped in that call: a GitOps apply that disables cve on three
  teams enqueues one job with `fleet_ids: [5, 7, 11]`, not three.
- Per-call dedup: if a single API call disables a dataset both globally
  and on one or more teams, only the global job is enqueued — the
  global DELETE subsumes the per-team scrub.
- Jobs are enqueued from the same diff-detection points that already emit
  the `enabled_historical_dataset` / `disabled_historical_dataset`
  activities (shipped by `chart-disabling-gitops-api`):
  - `ModifyAppConfig` (POST `/api/v1/fleet/config`)
  - `ModifyTeam` (PATCH `/api/v1/fleet/fleets/{id}`)
  - GitOps batch apply paths
- **Sequencing**: the SaveAppConfig / SaveTeam commit MUST complete before
  the scrub job is enqueued. After the commit, every cron run sees the
  new config and excludes the disabled scope from new writes — so scrub
  only has to clean up what was collected before the disable.
- **No re-enable cancellation**: disable→re-enable before the scrub runs
  still wipes pre-disable history. Documented as a property of the
  feature: "disabling a dataset deletes its collected data."

### New helper

- `BlobANDNOT(a, mask []byte) []byte` in `server/chart/blob.go`.
- Result length is `len(a)`; `mask` zero-extends if shorter.

### Docs

- `docs/Configuration/yaml-files.md` — note that disabling a dataset
  triggers asynchronous deletion of its collected data and that
  re-enabling does not restore it.
- `docs/REST API/rest-api.md` — same note on the global config / fleet
  PATCH endpoints that accept `features.historical_data`.
- `docs/Contributing/reference/audit-logs.md` — already documents the
  enable/disable activities (no change here); flag in the description that
  the disable activity implies an asynchronous scrub.

## Capabilities

### New Capabilities

- `chart-historical-data-collection`: cron-side gating semantics
  (skip-when-globally-disabled, mask-when-fleet-disabled, no-team
  follows-global) and the disable-flip data-scrub contract (job types,
  triggering points, sequencing requirement, race-free interaction with
  the collection cron).

### Modified Capabilities

None. The settings capability (`chart-historical-data-settings`)
already documents the activity emission and config shape; this change
adds *consumer* behavior under a new capability rather than amending
that one. The dataset interface capability
(`chart-dataset-interface`) gains a parameter, but the v1 spec there
predates `CollectScope` and adding the arg is an API contract change
captured in the new capability rather than a delta on the old one.

## Impact

- **Code**: `server/chart/api/chart.go` (`Dataset` interface),
  `server/chart/datasets.go` (both `Collect` implementations),
  `server/chart/blob.go` (`BlobANDNOT`),
  `server/chart/internal/service/service.go` (`CollectDatasets`
  signature flow), `cmd/fleet/cron.go` (mask construction),
  `server/service/appconfig.go` and `server/service/teams.go`
  (scrub-job enqueue alongside existing activity emission), the
  GitOps apply path in `server/service/`, plus a new worker
  registration in `cmd/fleet/serve.go` (or wherever workers register
  today).
- **Database**: no schema change. New job rows in the existing
  `jobs` table; `host_scd_data` is read/updated/deleted, schema
  unchanged.
- **Tests**: unit tests for `BlobANDNOT`, mask-application in each
  dataset, the orchestrator's mask construction (including the
  "global on, all fleets on" short-circuit), the scrub handlers
  (global delete, per-fleet ANDNOT walk, empty-fleet no-op), and the
  diff-and-enqueue plumbing in each of the three triggering paths.
- **Backwards compatibility**: `Dataset.Collect` signature changes —
  internal API only, no public consumers, contained to `server/chart/`.
  No API or wire-format change.
- **Out of scope**: frontend UI, scrub-completed activity, snapshot-at-
  disable host capture, restore-on-re-enable.
