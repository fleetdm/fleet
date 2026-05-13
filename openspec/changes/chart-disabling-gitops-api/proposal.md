## Why

Issue #44077 ("Allow disabling data collection for charts") adds admin-visible
switches to disable per-dataset historical-data collection for the dashboard
charts. The frontend, cron gating, and data-drop work are tracked separately;
this change delivers the **GitOps / API surface** those consumers depend on:

- A new `features.historical_data` config object with a sub-key per dataset.
- Both `POST /api/v1/fleet/config` and `PATCH /api/v1/fleet/teams/{id}` accept
  the new keys with standard PATCH-merge semantics.
- GitOps (global YAML and fleet YAML) accepts the new keys.
- A pair of audit activities fire when a dataset is enabled or disabled.
- A small mapping helper makes the "config key vs internal dataset name"
  translation explicit and centralized — the issue mandates the config keys
  `uptime` and `vulnerabilities`, but the internal dataset names are `uptime`
  and `cve`. The mismatch needs a single, greppable mapping point so callers
  don't hardcode the translation.

Default for both sub-keys is `true`. Existing deployments upgrade with both
datasets enabled, preserving current behavior.

## What Changes

### Data shape

- Add `HistoricalDataSettings` struct to `server/fleet/app.go`:
  ```go
  type HistoricalDataSettings struct {
      Uptime          bool `json:"uptime"`           // dataset "uptime"
      Vulnerabilities bool `json:"vulnerabilities"`  // dataset "cve"
  }
  ```
  Both fields **without** `omitempty` so an explicit `false` round-trips
  through GitOps overwrite mode (matches `EnableHostUsers` /
  `EnableSoftwareInventory`).
- Add `HistoricalData HistoricalDataSettings` to `Features` with JSON tag
  `historical_data` (no `omitempty`).
- Update the existing "WARNING: account in the Features Clone implementation"
  comment block. `HistoricalDataSettings` is value-type-only so
  `clone := *f` already deep-copies it; verify in unit test.
- Because `Team.Config` embeds `Features`, `historical_data` is automatically
  available per-fleet with no schema change.

### Mapping helper

- Add a method on `HistoricalDataSettings`:
  ```go
  func (h HistoricalDataSettings) Enabled(dataset string) (bool, error) {
      switch dataset {
      case "uptime": return h.Uptime, nil
      case "cve":    return h.Vulnerabilities, nil
      default:       return false, fmt.Errorf("unknown dataset %q", dataset)
      }
  }
  ```
  The cron consumer (separate change) will use this. The mapping lives once,
  on the type that owns the data, with a safelist switch that prevents string
  interpolation into JSON paths and provides a clear error for unknown
  datasets.

### Defaults

- `Features.ApplyDefaults()` sets both sub-fields `true`.
  `ApplyDefaultsForNewInstalls()` already delegates to `ApplyDefaults()`; no
  separate change needed.
- The existing pre-unmarshal `ApplyDefaults` priming on both read paths
  (`server/datastore/mysql/app_configs.go` and
  `server/datastore/mysql/teams.go`) means existing rows whose stored
  JSON omits `historical_data` read back with both sub-fields `true`.
- A small backfill is added to migration `20260423161823_AddHostSCDData`
  to set `features.historical_data.{uptime,vulnerabilities}` to `true` on
  every `app_config_json` row and every `teams.config` row. This is
  required because earlier migrations using `updateAppConfigJSON` (and
  the inline TeamConfig round-trip pattern) re-marshal the whole struct
  on save, and would otherwise stamp the new field's zero value (`false`)
  into stored JSON the moment it appeared in Go — silently degrading the
  upgrade default to `false`. The backfill runs in the same migration
  that creates `host_scd_data` (the chart data table) so it's
  conceptually grouped with the rest of the chart-disabling work, and
  this code lands in 4.85.0 before any deployment exposes the toggle to
  admins.

### API surface

- `POST /api/v1/fleet/config` (`ModifyAppConfig`) accepts the new keys.
  PATCH-merge works for free: the endpoint unmarshals raw JSON into the
  existing config, and Go's JSON decoder recurses into nested structs,
  only touching fields present in the payload.
- `PATCH /api/v1/fleet/fleets/{id}` (`ModifyTeam`) accepts the same
  `{features: {historical_data: {...}}}` shape, but the wiring is
  different because `ModifyTeam` takes a parsed `TeamPayload` rather
  than raw bytes. Add a `TeamPayloadFeatures` payload-only subset of
  `Features` containing just `HistoricalData *HistoricalDataPayload`,
  and a `HistoricalDataPayload` whose sub-fields are `optjson.Bool`
  (matching the `optjson.Bool`-based partial-PATCH pattern already used
  by `mdm.enable_disk_encryption` on this endpoint). Sub-keys whose
  `Valid` is `false` are left untouched; sub-keys with explicit values
  flip the stored fleet config. Other `features` sub-fields
  (`enable_host_users`, `enable_software_inventory`,
  `additional_queries`, `detail_query_overrides`) remain writable
  per-fleet only via `/spec/fleets`. Storage and read-back shapes are
  identical to global (`features.historical_data`); only the request
  decoding is structurally different.
- GitOps (global config YAML and fleet YAML) accepts the keys via the
  same `Features` unmarshal that already handles other Features fields.
  `ApplySpecOptions.Overwrite=true` is the same risk profile as every
  other `Features` field today; documented in the GitOps YAML reference.

### Activities

- Add two activity types:
  ```go
  type ActivityTypeEnabledHistoricalDataset struct {
      Dataset   string  `json:"dataset"`     // config key: "uptime" or "vulnerabilities"
      FleetID   *uint   `json:"fleet_id"`    // nil for global
      FleetName *string `json:"fleet_name"`  // nil for global
  }

  type ActivityTypeDisabledHistoricalDataset struct {
      Dataset   string  `json:"dataset"`
      FleetID   *uint   `json:"fleet_id"`
      FleetName *string `json:"fleet_name"`
  }
  ```
  Activity-type strings: `enabled_historical_dataset` /
  `disabled_historical_dataset`.
- One activity per sub-field that flipped per request. No-op PATCH (same
  values back) emits zero activities. Global emits with `fleet_id` /
  `fleet_name` `nil`; per-fleet emits with both populated.
- The `dataset` payload uses the **config key** (`vulnerabilities`, not
  `cve`) since the audit log is admin-facing and admins see config keys in
  YAML and (eventually) the UI.

### Docs

- `docs/REST API/rest-api.md` — global config and fleet-modify sections gain
  a `features.historical_data` shape and example payload.
- `docs/Configuration/yaml-files.md` — global and fleet sections document
  the keys, defaults, and the GitOps overwrite-on-omit behavior.
- `docs/Contributing/reference/audit-logs.md` — entries for both new
  activity types with payload field documentation and global + fleet-scoped
  examples.

## Capabilities

### Added Capabilities

- `chart-historical-data-settings` — describes the config shape, defaults,
  PATCH-merge semantics, GitOps overwrite behavior, dataset-name ↔ config-key
  mapping, fleet-scoped settings inheritance, and audit activity emission for
  `features.historical_data`.

## Impact

- **One backfill migration.** `Features` is stored as a JSON blob in
  `app_config_json` and `teams.config`. Migration `20260423161823` (which
  also creates `host_scd_data`) writes
  `features.historical_data.{uptime,vulnerabilities} = true` on every
  existing row. The `ApplyDefaults`-before-unmarshal pattern already in
  the storage read paths covers fresh installs and any row whose stored
  JSON simply omits the key.
- **No API version bump.** New keys in an existing JSON body are
  backwards-compatible. `EnableStrictDecoding` rejects unknown fields;
  `historical_data` is known after this change.
- **Consumer changes deferred.** The cron gating that *uses* the helper, the
  data-drop on disable, the dashboard "data collection disabled" empty state,
  and the Advanced / Fleet Settings UI checkboxes are all separate changes.
  This change ships the API contract those consumers depend on.

## Out of Scope

- **Cron gating in `server/chart/`.** A separate change wires the
  `Enabled(dataset)` helper into `Service.CollectDatasets` and adds a
  per-fleet filter on `FindRecentlySeenHostIDs`. This change ships the
  helper but does not call it.
- **Data drop on disable.** The issue requires that disabling a dataset
  globally truncates its `host_scd_data` rows, and disabling per-fleet
  scrubs that fleet's contributions. This is a separate change.
- **Frontend UI.** "Disable hosts active" / "Disable vulnerabilities"
  checkboxes on Advanced + Fleet Settings, the confirmation dialog, and the
  dashboard empty-state messaging are all separate changes.
- **Per-dataset defaults other than `true`.** A future expensive/opt-in
  dataset would need its own design conversation.
- **Tri-state "inherit from global"** on the fleet value. Plain bool with
  `true/true` defaults; the `global AND fleet` rule is the only precedence
  mechanic. Cron-side enforcement is the consumer's concern.
- **`DeviceFeatures` surfacing.** The device-endpoint subset of app config
  does not expose `historical_data`. This setting controls server-side
  rollup collection; nothing about device behavior changes.
