## 1. Bitmap helper

- [x] 1.1 Add `BlobANDNOT(a, mask []byte) []byte` to `server/chart/blob.go` per design decision 8 (result length = `len(a)`, mask zero-extends if shorter, excess mask bytes ignored).
- [x] 1.2 Add unit tests in `server/chart/blob_test.go` covering: equal-length, mask-shorter, mask-longer, all-zero mask = identity, all-ones equal-length mask = empty, nil/empty `a`, nil mask.

## 2. Dataset interface + DatasetStore — disabledFleetIDs parameter (push-down)

> **Note**: The first pass of section 2 wired an `allowedMask []byte`
> parameter through `Dataset.Collect`. That has been reverted in favor
> of push-down — see design decision 1 for rationale. The tasks below
> describe the final state.

- [x] 2.1 Update `Dataset.Collect` signature in `server/chart/api/chart.go` to `Collect(ctx context.Context, store DatasetStore, now time.Time, disabledFleetIDs []uint) error`. Doc-comment: `nil` or empty = no scoping; populated = exclude those team IDs from collection.
- [x] 2.2 Update `DatasetStore.FindRecentlySeenHostIDs` and `DatasetStore.AffectedHostIDsByCVE` signatures in `server/chart/api/chart.go` to accept the same `disabledFleetIDs []uint` parameter. Mirror the change in `server/chart/internal/types/chart.go` (the internal `Datastore` interface).
- [x] 2.3 Update `UptimeDataset.Collect` in `server/chart/datasets.go` to forward `disabledFleetIDs` to `store.FindRecentlySeenHostIDs`. No mask logic; just pass through.
- [x] 2.4 Update `CVEDataset.Collect` in `server/chart/datasets.go` to forward `disabledFleetIDs` to `store.AffectedHostIDsByCVE`. No mask logic; just pass through.
- [x] 2.5 Update `FindRecentlySeenHostIDs` SQL in `server/chart/internal/mysql/charts.go` to add `AND (h.team_id IS NULL OR h.team_id NOT IN (?))` when `len(disabledFleetIDs) > 0`. Use `sqlx.In` to expand the slice. Skip the clause entirely when empty/nil.
- [x] 2.6 Update `AffectedHostIDsByCVE` SQL (both the software-side and OS-side subqueries) to JOIN `hosts h` on the host id and add the same WHERE clause when `len(disabledFleetIDs) > 0`. The `streamCVEHostPairs` helper in that file builds both subqueries; thread the parameter through.
- [x] 2.7 Update `CollectDatasets` in `server/chart/internal/service/service.go` to accept a per-dataset scope resolver: `CollectDatasets(ctx context.Context, now time.Time, scope func(name string) (skip bool, disabledFleetIDs []uint)) error`. For each registered dataset: call `scope(name)`; if `skip`, do nothing for that dataset; else call `dataset.Collect(ctx, store, now, disabledFleetIDs)`. If `scope == nil`, treat every dataset as `(skip=false, nil)` — preserves existing test ergonomics.
- [x] 2.8 Update the public `api.Service` interface in `server/chart/api/service.go` to match the new `CollectDatasets` signature.
- [x] 2.9 Update `mockDatastore` in `server/chart/internal/service/service_test.go` for the new `FindRecentlySeenHostIDs` and `AffectedHostIDsByCVE` signatures (add the parameter; default mock returns ignore it).
- [x] 2.10 Update existing tests for the new signatures: `TestCollectDatasetsUptime` and `TestCollectDatasetsCVE` pass `nil` scope; new tests cover scope=nil → all datasets collected, scope returns skip=true → dataset not invoked, scope returns disabledFleetIDs slice → forwarded to the store.
- [ ] 2.11 Add MySQL integration tests in `server/chart/internal/mysql/data_test.go` (or new file) that verify push-down: insert hosts in different teams, call `FindRecentlySeenHostIDs(disabledFleetIDs=[X])`, assert team-X hosts absent. Same for `AffectedHostIDsByCVE`. Requires `MYSQL_TEST=1`.

## 3. Orchestrator — config-aware scope resolver

> **Note**: The mask-based version of section 3 required a new
> `HostIDsForChartScope` method on the main Fleet datastore. Push-down
> eliminates that need — `disabledFleetIDs` is derived from the team
> configs the orchestrator already loads. No new datastore methods.

- [x] 3.1 Locate the function in `cmd/fleet/cron.go` (or `cmd/fleet/serve.go`) that today calls `chartSvc.CollectDatasets(ctx, now)`. Identify how it gets a viewer/admin user — needs one for `svc.ListTeams`.
- [x] 3.2 In that function, before the `CollectDatasets` call: load `appCfg` (already a service method) and the full team list (`svc.ListTeams(ctx, fleet.TeamFilter{User: <admin>})`). Use existing helpers — no new datastore methods.
- [x] 3.3 Build the scope resolver closure (extracted to `buildChartScopeResolver` for testability).
- [x] 3.4 Pass `scope` into `chartSvc.CollectDatasets(ctx, now, scope)` (signature updated in task 2.7).
- [x] 3.5 Unit-test the scope-resolver logic in `cmd/fleet/cron_test.go` (or a new test file) covering: global off → `(skip=true, nil)`; global on, all teams on → `(skip=false, nil)` or empty slice; mixed → `(skip=false, [ids])`; per-dataset isolation (uptime list ≠ cve list).

## 4. Job framework — scrub job types and handlers

- [x] 4.1 Define job-type constants in `server/worker/chart_scrub.go`: `ChartScrubDatasetGlobalJobName` and `ChartScrubDatasetFleetJobName`.
- [x] 4.2 Define payload structs in the same file: `ChartScrubGlobalArgs{Dataset string}` and `ChartScrubFleetArgs{Dataset string, FleetIDs []uint}`. The `FleetIDs` slice is always populated (length 1 for single-PATCH callers, length N for batch callers).
- [x] 4.3 Implement `ChartScrubGlobal` job: unmarshals args, calls `chartSvc.ScrubDatasetGlobal(ctx, dataset)`, which delegates to chart datastore `DeleteAllForDataset` (loop `DELETE ... LIMIT 5000` until zero rows). Each statement is its own transaction.
- [x] 4.4 Implement `ChartScrubFleet` job: unmarshals args, calls `chartSvc.ScrubDatasetFleet(ctx, dataset, fleetIDs)`. Service resolves hosts via `HostIDsInFleets`, builds mask, calls `ApplyScrubMaskToDataset` (paged ANDNOT walk).
- [x] 4.5 Place SQL primitives in chart-context: `DeleteAllForDataset`, `HostIDsInFleets`, `ApplyScrubMaskToDataset` on `types.Datastore` + `internal/mysql/data.go`. Service-level methods on `chart_api.Service`: `ScrubDatasetGlobal`, `ScrubDatasetFleet`.
- [x] 4.6 Register handlers in `cmd/fleet/cron.go::newWorkerIntegrationsSchedule` alongside existing workers (`jira`, `zendesk`, etc.). Threaded `chartSvc` through `cmd/fleet/serve.go::cronSchedules`.
- [x] 4.7 Unit tests in `server/worker/chart_scrub_test.go` and `server/chart/internal/service/service_test.go` covering: forwarding to service, dataset/empty-fleet/empty-dataset edge cases, error propagation, malformed JSON, mask correctness for fleet scrub.

## 5. Disable-flip → enqueue plumbing

Spec relaxed (see `chart-historical-data-collection` spec): one job per
(dataset, scope) flip. Per-call coalescing across batch GitOps applies is
deferred as a future optimization. Cross-flip dedup (global subsumes
per-team) is also deferred.

- [x] 5.1 In `server/service/appconfig.go::ModifyAppConfig`, after `SaveAppConfig` succeeds and the existing activities emit, call `fleet.EnqueueHistoricalDataScrubs(ctx, svc.ds, old, new, nil)` to enqueue one global scrub job per disabled flip.
- [x] 5.2 In `ee/server/service/teams.go::ModifyTeam`, after `SaveTeam` succeeds and activities emit, call `fleet.EnqueueHistoricalDataScrubs(ctx, svc.ds, old, new, &team.ID)` to enqueue one fleet scrub job per disabled flip with this team's ID.
- [ ] 5.3 GitOps batch paths (`ApplyTeamSpecs` etc.): if those paths emit `EmitHistoricalDataActivities` for HistoricalData flips, also call `EnqueueHistoricalDataScrubs` alongside. (Note: a survey of the current code did not locate `EmitHistoricalDataActivities` calls inside `ApplyTeamSpecs` — that is either a gap in `chart-disabling-gitops-api` to address there, or those flips never reach those paths in practice. Either way, this task should be revisited once that question is settled.)
- [x] 5.4 Skip enqueue for no-op flips: handled in `EnqueueHistoricalDataScrubs` (same `old == new` short-circuit as the activity helper).
- [x] 5.5 Skip enqueue for false→true flips: handled in `EnqueueHistoricalDataScrubs`.
- [ ] 5.6 Integration tests in `server/service/integration_*_test.go`:
  - `MYSQL_TEST=1 REDIS_TEST=1`
  - PATCH global config with `vulnerabilities: false`, assert one `chart_scrub_dataset_global` job with payload `{"dataset":"cve"}` (internal dataset name, not the public config sub-key).
  - PATCH a team disabling uptime, assert one `chart_scrub_dataset_fleet` job with `fleet_ids:[<team>]` and `dataset: "uptime"`.
  - PATCH a no-op (already-disabled): zero new jobs.
  - PATCH false→true: zero jobs.
## 6. Documentation

- [ ] 6.1 Update `docs/Configuration/yaml-files.md` to note that disabling a dataset triggers asynchronous deletion of its collected data and that re-enabling does NOT restore prior history.
- [ ] 6.2 Update `docs/REST API/rest-api.md` with the same note on the global config + fleet PATCH endpoints accepting `features.historical_data`.
- [ ] 6.3 Update `docs/Contributing/reference/audit-logs.md` `disabled_historical_dataset` entry to mention that an asynchronous scrub is enqueued and that no follow-up activity fires on scrub completion.

## 7. Verification

- [ ] 7.1 `make lint-go-incremental` passes.
- [ ] 7.2 `go test ./server/chart/...` passes.
- [ ] 7.3 `MYSQL_TEST=1 go test ./server/datastore/mysql/...` passes.
- [ ] 7.4 `MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/...` passes.
- [ ] 7.5 Manual end-to-end smoke (against `make serve`):
  - Create a team, generate a few buckets of data with global+team enabled.
  - Disable the dataset on that team via PATCH.
  - Confirm `disabled_historical_dataset` activity emits.
  - Confirm a `chart_scrub_dataset_fleet` row appears in `jobs`.
  - Wait for worker to run; confirm `host_scd_data` rows for that dataset have the team's host bits cleared.
  - Disable globally; confirm rows for the dataset are deleted entirely.
- [ ] 7.6 `openspec validate chart-disabling-collection-scrub` passes (if available in the toolchain).
