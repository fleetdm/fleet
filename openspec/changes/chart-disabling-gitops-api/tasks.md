## Backend — types & defaults

- [x] **Add `HistoricalDataSettings` type and `Features.HistoricalData` field**
  In `server/fleet/app.go`, define `HistoricalDataSettings` with `Uptime` and
  `Vulnerabilities` bool fields (JSON tags `uptime`, `vulnerabilities`, no
  `omitempty`). Add a `HistoricalData HistoricalDataSettings` field to
  `Features` (JSON tag `historical_data`, no `omitempty`). Update the
  "WARNING: account in the Features Clone implementation" comment block.

- [x] **Wire up `ApplyDefaults()` so both sub-fields default true**
  In `Features.ApplyDefaults()`, set
  `f.HistoricalData.Uptime = true` and
  `f.HistoricalData.Vulnerabilities = true`.
  `ApplyDefaultsForNewInstalls()` already delegates to `ApplyDefaults()`;
  no change needed there.

- [x] **Verify `Features.Copy()` / `Features.Clone()` deep-copy correctness**
  `HistoricalDataSettings` is value-type-only (two bools), so the existing
  `clone := *f` already deep-copies it. Verify this in the unit test rather
  than adding branch logic. If a future field adds a pointer/slice/map,
  revisit.

- [x] **Add the `Enabled(dataset string) (bool, error)` helper**
  Method on `HistoricalDataSettings` with a safelist switch:
  `"uptime"` → `Uptime`, `"cve"` → `Vulnerabilities`, default → error
  `unknown dataset %q`. This is the single mapping point between internal
  dataset names and config keys.

- [x] **Unit tests for defaults, copy, and helper**
  In `server/fleet/app_test.go` (or the closest existing file):
  - `ApplyDefaults()` sets both sub-fields true.
  - `Features.Copy()` produces an independent `HistoricalData` value
    (mutate the original, assert the copy is unchanged).
  - `Enabled("uptime")` and `Enabled("cve")` return the corresponding
    bool with no error.
  - `Enabled("unknown")` returns an error and `false`.

## Backend — global config endpoint

- [x] **Integration test: `ModifyAppConfig` PATCH preserves untouched sub-field**
  In `server/service/integration_core_test.go` or equivalent: PATCH only
  `{"features": {"historical_data": {"vulnerabilities": false}}}` and
  assert `uptime` stays `true`. Regression guard for the
  JSON-unmarshal-into-existing-struct behavior.

- [x] **Integration test: GitOps overwrite resets omitted sub-fields to defaults**
  Apply a global GitOps config with `features: {}` (omitting
  `historical_data`) and assert both sub-fields return to their defaults
  (`true`, `true`). Documents and pins the snap-back behavior.

- [x] **Integration test: existing pre-change rows read back as `true/true`**
  Set up an `app_config_json` row whose stored JSON omits `historical_data`
  (simulating an upgraded deployment). Read via `AppConfig` and assert both
  sub-fields are `true`. Confirms the `ApplyDefaults`-before-unmarshal path
  handles pre-upgrade data.

## Backend — fleet config endpoint

- [x] **Integration test: `ModifyTeam` PATCH preserves untouched sub-field**
  PATCH `{"features": {"historical_data": {"uptime": false}}}` to
  `PATCH /api/v1/fleet/teams/{id}` and assert the fleet's `uptime` flips to
  `false` while `vulnerabilities` stays at its prior value. Confirm
  `ApplyDefaultsForNewInstalls`-before-unmarshal is exercised on the team
  read path.

- [x] **Integration test: fleet GitOps overwrite resets omitted fields to defaults**
  Apply a fleet GitOps YAML with `features: {}` (omitting
  `historical_data`) and assert both sub-fields return to their defaults
  on that fleet.

- [x] **Integration test: new fleet defaults to `true/true`**
  Create a new fleet, read its config back, assert
  `features.historical_data.uptime` and `.vulnerabilities` are both `true`.
  Catches any regression where `Team.Config.Features` isn't primed with
  defaults on create.

## Backend — activities

- [x] **Add `ActivityTypeEnabledHistoricalDataset` and `ActivityTypeDisabledHistoricalDataset`**
  In `server/fleet/activities.go`, define both types with payload:
  ```go
  Dataset   string  `json:"dataset"`
  FleetID   *uint   `json:"fleet_id"`
  FleetName *string `json:"fleet_name"`
  ```
  Activity-type strings: `enabled_historical_dataset` /
  `disabled_historical_dataset`. No `renameto` tag — these are new types.

- [x] **Emit activities on toggle in `ModifyAppConfig`**
  After save, diff `oldAppConfig.Features.HistoricalData` against
  `appConfig.Features.HistoricalData`. For each sub-field that flipped,
  emit the corresponding activity with `FleetID` / `FleetName` `nil` and
  `Dataset` set to the **config key** (`"uptime"` or `"vulnerabilities"`).
  One activity per dataset that changed; zero activities if nothing changed.

- [x] **Emit activities on toggle in `ModifyTeam`**
  Same diff against the fleet's prior `Features.HistoricalData`. Populate
  `FleetID` and `FleetName` from the fleet being modified. One activity
  per dataset whose value changed for that fleet.

- [x] **Integration test: global activity emission**
  Three cases in `server/service/integration_core_test.go`:
  - PATCH that flips `vulnerabilities` only → one
    `disabled_historical_dataset` activity with
    `dataset="vulnerabilities"`, `fleet_id=null`, `fleet_name=null`.
  - PATCH that flips both → two activities (one enabled, one disabled).
  - PATCH that sends the same values back → zero activities.

- [x] **Integration test: fleet-scoped activity emission**
  Three cases against `PATCH /api/v1/fleet/teams/{id}`:
  - Flip `uptime` only on a fleet → one activity with the fleet's
    `fleet_id` and `fleet_name` populated.
  - Flip both on one fleet → two activities with fleet fields populated.
  - PATCH the same values back → zero activities.
  - Flip the same dataset on two different fleets in separate requests →
    two activities, one per fleet.

## Backend — activities (GitOps fix)

- [x] **Rename `EmitHistoricalDataActivities` → `OnHistoricalDataChanged`**
  Rename the method in `server/service/appconfig.go` (ModifyAppConfig call
  site) and `ee/server/service/teams.go` (ModifyTeam call site). No
  functional change — this clarifies that the method is a lifecycle hook
  that will gain scrub-job logic in future changes.

- [x] **Fix GitOps fleet activity emission in `editTeamFromSpec`**
  In `ee/server/service/teams.go`, in the `editTeamFromSpec` function:
  - Snapshot `oldHistoricalData := team.Config.Features.HistoricalData`
    BEFORE the wholesale `Features` replace (line 1576).
  - After `SaveTeam` succeeds (around line 1851), call
    `OnHistoricalDataChanged(ctx, svc, authz.UserFromContext(ctx),
    oldHistoricalData, team.Config.Features.HistoricalData, &team.ID,
    &team.Name)`. This ensures activities fire when fleet-level GitOps
    flips historical_data, matching the behavior of PATCH endpoints.

- [x] **Integration test: fleet GitOps emits activities on historical_data flip**
  In `cmd/fleetctl/fleetctl/gitops_test.go`, add a test case (alongside
  `TestGitOpsHistoricalDataDefaults`) that:
  - Applies a fleet GitOps YAML that flips `features.historical_data.uptime`
    to `false`.
  - Asserts a `disabled_historical_dataset` activity exists with
    `dataset="uptime"`, the fleet's `fleet_id`, and `fleet_name`.

## GitOps / fleetctl

- [x] **fleetctl GitOps test: global YAML accepts `features.historical_data`**
  Test case in `cmd/fleetctl/fleetctl/gitops_test.go` (or the integration
  suite) that applies a global YAML with `features.historical_data` set and
  confirms the resulting config matches.

- [x] **fleetctl GitOps test: fleet YAML accepts `features.historical_data`**
  Test case that applies a fleet YAML with `features.historical_data` set
  and confirms the resulting fleet config matches.

- [x] **fleetctl GitOps test: omitted `historical_data` snaps to defaults**
  Two cases (global and fleet) — apply a YAML with `features: {}` and
  assert both sub-fields are `true` after apply, even if they were `false`
  in the prior state. Documents the overwrite semantic.

## Docs

- [x] **REST API reference — global config**
  Update `docs/REST API/rest-api.md` (Modify configuration section) with
  the new `features.historical_data` shape and example PATCH payload.

- [x] **REST API reference — fleet config**
  Update the fleet-modify section of `docs/REST API/rest-api.md` with the
  `features.historical_data` shape and example PATCH payload.

- [x] **GitOps YAML reference — global**
  Update `docs/Configuration/yaml-files.md` to document
  `features.historical_data` (global section), default values, and the
  overwrite-on-omit behavior with a one-paragraph note recommending
  GitOps-managed deployments include the key explicitly.

- [x] **GitOps YAML reference — fleet**
  Update the fleet-YAML section of `docs/Configuration/yaml-files.md`
  with `features.historical_data` docs. Note the AND-with-global
  effective-value rule even though enforcement lives in the cron change.

- [x] **Audit log reference**
  Update `docs/Contributing/reference/audit-logs.md` with entries for
  `enabled_historical_dataset` and `disabled_historical_dataset`. Document
  the `dataset`, `fleet_id`, `fleet_name` payload fields with examples
  for both global (`fleet_id: null`) and fleet-scoped
  (`fleet_id: 2, fleet_name: "EMEA"`) variants. Note that `dataset`
  carries the config key, not the internal dataset name.

## Lint & verify

- [x] **Run `make lint-go-incremental` and address any findings**
  Standard hygiene before considering the change ready for review.

- [x] **Run `MYSQL_TEST=1 REDIS_TEST=1 go test ./server/service/... ./server/fleet/...`**
  Confirms the new types, defaults, and activity emission don't break
  adjacent tests (uninitialized mocks crashing other tests is the usual
  failure mode when adding interface methods).

## Defect fix: stored-JSON round-trip clobber → backfill migration

Discovered after the change was initially marked complete: earlier
migrations using `updateAppConfigJSON` (and the inline TeamConfig
round-trip pattern) re-marshal the whole struct on save, persisting the
zero value of the new `HistoricalData` sub-fields as `false/false` in
stored JSON. The next read sees explicit `false` and the pre-unmarshal
`true` default loses. See Design Decision #3 for the rationale; this
PR ships the targeted backfill described there.

- [x] **Backfill historical_data in `20260423161823_AddHostSCDData`**
  Same migration that creates `host_scd_data`. Use
  `updateAppConfigJSON` to set the AppConfig sub-keys to `true`
  (round-trip is safe here since the values being written are non-zero).
  Use `JSON_MERGE_PATCH` for `teams.config` rows to add or replace
  `features.historical_data` per row without round-tripping the whole
  TeamConfig struct.

- [x] **Migration test asserts the backfill**
  `20260423161823_AddHostSCDData_test.go`: insert a team with a
  pre-existing `features` block (no `historical_data`), apply the
  migration, then assert both `app_config_json` and the team's `config`
  have `historical_data.{uptime,vulnerabilities} = true`. Also confirm
  pre-existing `features.*` keys on the team survive `JSON_MERGE_PATCH`.

- [x] **Regenerate `server/datastore/mysql/schema.sql`**
  `make dump-test-schema`. Expected diff: the seeded
  `app_config_json.json_value` now shows
  `"historical_data": {"uptime": true, "vulnerabilities": true}`.

- [x] **Update fleetctl testdata golden files**
  Files rendered from the schema seed now show `historical_data:
  {uptime: true, vulnerabilities: true}`:
  `cmd/fleetctl/fleetctl/testdata/expectedGetConfigAppConfigJson.json`
  and the matching YAML / TeamMaintainer / IncludeServerConfig
  variants; the `generateGitops/` and `macosSetup*` golden files for
  the same reason.

- [x] **Verify lint and the originally failing tests pass**
  `make lint-go-incremental` then targeted runs of
  `TestIntegrationsEnterprise/TestModifyTeamHistoricalData`,
  `TestIntegrationsEnterprise/TestTeamConfigHistoricalDataGitOps`,
  `TestIntegrationsCore/TestAppConfigHistoricalData`, plus
  `TestUp_20260423161823`.

## CI infrastructure (incidental)

- [x] **Wire fleetctl suite to build a local preview image**
  `integrationtest/preview/preview_test.go` and the `fleetctl` job in
  `.github/workflows/test-go-suite.yaml` were updated so the suite builds a
  local `fleetdm/fleet:dev` image and pins `FLEET_PREVIEW_TAG=dev` instead
  of pulling `:main`. Needed because new integration tests added by this
  change exercise paths the published `:main` image didn't yet cover. Not
  a spec concern — recorded here for future bisecting.

## Backlog (out of scope here)

- [ ] **Refactor `updateAppConfigJSON` to use surgical `JSON_SET`**
  The targeted backfill fixes `historical_data` for this PR, but the
  same trap will fire for the next field added with a non-zero default
  on top of existing rows. The general fix is to convert the migration
  helper from struct-round-trip to path-based `JSON_SET`, plus a
  sibling helper for `team_config`. ~12 merged migrations need callsite
  conversion. Mechanical but non-trivial; carry as a separate change
  when there's appetite.
