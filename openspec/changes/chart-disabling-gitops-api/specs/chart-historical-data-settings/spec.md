## ADDED Requirements

### Requirement: Historical-data settings live under `features.historical_data` on both global and fleet config

Fleet SHALL expose a `historical_data` object nested under `features` on both
`AppConfig` (global) and every fleet's `Team.Config` (per-fleet, embedded via
`Features`). The object SHALL contain one boolean sub-key per dataset whose
collection can be toggled by an admin. v1 sub-keys:

- `uptime` — controls collection of the `uptime` historical dataset (drives
  the Hosts Active dashboard chart).
- `vulnerabilities` — controls collection of the `cve` historical dataset
  (drives the Vulnerability Exposure dashboard chart).

Both sub-keys SHALL be plain booleans (no pointer/tri-state). A `true` value
means "collect this dataset"; a `false` value means "do not collect."

#### Scenario: New install reads global config with both sub-keys true

- **GIVEN** a freshly initialized Fleet deployment
- **WHEN** the global config is read via `GET /api/v1/fleet/config`
- **THEN** the response SHALL include `features.historical_data.uptime = true`
- **AND** the response SHALL include `features.historical_data.vulnerabilities = true`

#### Scenario: Existing deployment with no stored `historical_data` reads as defaults

- **GIVEN** an `app_config_json` row whose stored JSON omits the
  `historical_data` key (i.e. predates this change)
- **WHEN** the global config is read via `AppConfig`
- **THEN** the returned config SHALL include `features.historical_data.uptime = true`
- **AND** the returned config SHALL include `features.historical_data.vulnerabilities = true`

#### Scenario: Upgraded deployment is backfilled to defaults

- **GIVEN** an existing Fleet deployment that runs migration
  `20260423161823_AddHostSCDData`
- **WHEN** the migration completes
- **THEN** every `app_config_json` row SHALL have
  `features.historical_data.uptime = true` and
  `features.historical_data.vulnerabilities = true`
- **AND** every `teams.config` row SHALL have the same values

#### Scenario: New fleet defaults both sub-keys to true

- **GIVEN** a Fleet deployment with at least one existing fleet
- **WHEN** a new fleet is created via the standard team-create path
- **THEN** the fleet's `features.historical_data.uptime` SHALL be `true`
- **AND** the fleet's `features.historical_data.vulnerabilities` SHALL be `true`

### Requirement: PATCH endpoints merge `features.historical_data` sub-keys

Fleet SHALL accept partial `features.historical_data` payloads on both
`POST /api/v1/fleet/config` (`ModifyAppConfig`) and
`PATCH /api/v1/fleet/fleets/{id}` (`ModifyTeam`), merging them into the
existing stored config. A sub-key omitted from the payload SHALL retain
its current stored value.

Both endpoints SHALL accept the same JSON shape for the request body:
`{"features": {"historical_data": {...}}}`. On the fleet PATCH endpoint,
`historical_data` SHALL be the only `features` sub-field that can be
written; other `features` sub-fields (e.g. `enable_host_users`,
`enable_software_inventory`, `additional_queries`,
`detail_query_overrides`) remain writable per-fleet only via the
`/spec/fleets` GitOps path. Unknown sub-fields under `features` on the
fleet PATCH endpoint SHALL be silently ignored — they SHALL NOT modify
stored state and SHALL NOT cause the request to fail. (This matches the
existing decoder convention on `PATCH /fleets/{id}`; strict decoding for
that endpoint is out of scope for this capability.)

The stored location and GET response shape are
`features.historical_data` on both global and fleet configs.

#### Scenario: PATCH only `vulnerabilities=false` on global config preserves `uptime`

- **GIVEN** the global config has `features.historical_data.uptime = true`
- **AND** the global config has `features.historical_data.vulnerabilities = true`
- **WHEN** an admin sends
  `POST /api/v1/fleet/config` with body
  `{"features": {"historical_data": {"vulnerabilities": false}}}`
- **THEN** the saved global config SHALL have
  `features.historical_data.uptime = true` (unchanged)
- **AND** the saved global config SHALL have
  `features.historical_data.vulnerabilities = false`

#### Scenario: PATCH only `uptime=false` on a fleet preserves `vulnerabilities`

- **GIVEN** a fleet with
  `features.historical_data.uptime = true` and
  `features.historical_data.vulnerabilities = true`
- **WHEN** an admin sends
  `PATCH /api/v1/fleet/fleets/{id}` with body
  `{"features": {"historical_data": {"uptime": false}}}`
- **THEN** the saved fleet config SHALL have
  `features.historical_data.uptime = false`
- **AND** the saved fleet config SHALL have
  `features.historical_data.vulnerabilities = true` (unchanged)
- **AND** a subsequent `GET /api/v1/fleet/fleets/{id}` SHALL return
  `features.historical_data.uptime = false` and
  `features.historical_data.vulnerabilities = true`

#### Scenario: Fleet PATCH silently ignores unknown sub-fields under `features`

- **GIVEN** a fleet with `features.enable_host_users = true`
- **WHEN** an admin sends
  `PATCH /api/v1/fleet/fleets/{id}` with body
  `{"features": {"enable_host_users": false}}`
- **THEN** the response SHALL be 200
- **AND** the saved fleet's `features.enable_host_users` SHALL remain `true`

### Requirement: GitOps applies `historical_data` via the existing Features overwrite path, with sub-keys defaulting to `true` when omitted

Fleet SHALL accept `features.historical_data` in GitOps YAML for both
global config and per-fleet config. On every `fleetctl gitops` apply,
the gitops client SHALL inject `historical_data.uptime: true` and
`historical_data.vulnerabilities: true` into the request payload for
sub-keys not explicitly specified in the YAML, mirroring the existing
default-injection pattern for `features.enable_software_inventory`.

The practical effect: a YAML that omits `features` entirely, or that
includes a `features` block with no `historical_data`, or that includes
`historical_data` with one sub-key but not the other, SHALL leave the
omitted sub-keys at their default `true` value after apply — even
though the server's underlying `Overwrite=true` semantics would
otherwise zero them. This SHALL apply identically to global YAML and
fleet YAML.

This requirement applies only to `fleetctl gitops`. `fleetctl apply`
uses `Overwrite=false` and inherently leaves omitted fields at their
prior stored value — no client-side injection is needed for that path.

#### Scenario: GitOps YAML applies explicit `historical_data` values

- **GIVEN** a Fleet deployment with default `historical_data` (both true)
- **WHEN** an admin runs `fleetctl gitops apply` against a YAML containing
  `features.historical_data.uptime: true` and
  `features.historical_data.vulnerabilities: false`
- **THEN** the resulting global config SHALL have
  `features.historical_data.uptime = true`
- **AND** the resulting global config SHALL have
  `features.historical_data.vulnerabilities = false`

#### Scenario: Fleet GitOps YAML applies explicit `historical_data` values

- **GIVEN** a fleet with default `historical_data` (both true)
- **WHEN** the fleet's GitOps YAML is applied with
  `features.historical_data.uptime: false`
- **THEN** the resulting fleet config SHALL have
  `features.historical_data.uptime = false`
- **AND** the resulting fleet config SHALL have
  `features.historical_data.vulnerabilities = true`

#### Scenario: GitOps YAML omitting `features` entirely defaults both sub-keys to true

- **GIVEN** the global config has
  `features.historical_data.uptime = false` (set out-of-band) and
  `features.historical_data.vulnerabilities = false` (set out-of-band)
- **WHEN** an admin runs `fleetctl gitops apply` against a YAML whose
  `org_settings` block omits `features` entirely
- **THEN** the resulting global config SHALL have
  `features.historical_data.uptime = true`
- **AND** the resulting global config SHALL have
  `features.historical_data.vulnerabilities = true`

#### Scenario: GitOps YAML with one sub-key only defaults the other

- **GIVEN** the global config has both sub-keys `false`
- **WHEN** an admin runs `fleetctl gitops apply` against a YAML where
  `features.historical_data` contains `uptime: false` only
- **THEN** the resulting global config SHALL have
  `features.historical_data.uptime = false` (honoring the explicit value)
- **AND** the resulting global config SHALL have
  `features.historical_data.vulnerabilities = true` (defaulted)

#### Scenario: Fleet GitOps YAML omitting `historical_data` defaults both sub-keys to true

- **GIVEN** a fleet with both sub-keys `false`
- **WHEN** the fleet's GitOps YAML is applied with a `features` block that
  omits `historical_data`
- **THEN** the resulting fleet config SHALL have
  `features.historical_data.uptime = true`
- **AND** the resulting fleet config SHALL have
  `features.historical_data.vulnerabilities = true`

### Requirement: Toggling a sub-key emits one audit activity per affected sub-key

Fleet SHALL emit exactly one audit activity per `features.historical_data`
sub-key whose value changes during a `ModifyAppConfig` or `ModifyTeam` save:

- A sub-key flipping from `true` to `false` SHALL emit a
  `disabled_historical_dataset` activity.
- A sub-key flipping from `false` to `true` SHALL emit an
  `enabled_historical_dataset` activity.
- A save operation that does not change the value of a sub-key SHALL NOT
  emit an activity for that sub-key.

The activity payload SHALL include:

- `dataset` (string) — the **config key** (`"uptime"` or `"vulnerabilities"`),
  not the internal dataset name.
- `fleet_id` (uint, nullable) — `null` for global toggles; the fleet's ID
  for per-fleet toggles.
- `fleet_name` (string, nullable) — `null` for global toggles; the fleet's
  name for per-fleet toggles.

#### Scenario: Disabling vulnerabilities globally emits one disabled activity

- **GIVEN** the global config has
  `features.historical_data.vulnerabilities = true`
- **WHEN** an admin PATCHes the global config with
  `{"features": {"historical_data": {"vulnerabilities": false}}}`
- **THEN** Fleet SHALL emit exactly one activity of type
  `disabled_historical_dataset`
- **AND** the activity payload SHALL be
  `{"dataset": "vulnerabilities", "fleet_id": null, "fleet_name": null}`

#### Scenario: Disabling uptime on a fleet emits one fleet-scoped activity

- **GIVEN** fleet `42` (named `EMEA`) has
  `features.historical_data.uptime = true`
- **WHEN** an admin PATCHes the fleet with
  `{"features": {"historical_data": {"uptime": false}}}`
- **THEN** Fleet SHALL emit exactly one activity of type
  `disabled_historical_dataset`
- **AND** the activity payload SHALL be
  `{"dataset": "uptime", "fleet_id": 42, "fleet_name": "EMEA"}`

#### Scenario: Flipping both sub-keys in one request emits two activities

- **GIVEN** the global config has both sub-keys `true`
- **WHEN** an admin PATCHes the global config with
  `{"features": {"historical_data": {"uptime": false, "vulnerabilities": false}}}`
- **THEN** Fleet SHALL emit exactly two activities, both of type
  `disabled_historical_dataset`
- **AND** one activity SHALL have payload `dataset = "uptime"`
- **AND** one activity SHALL have payload `dataset = "vulnerabilities"`

#### Scenario: No-op PATCH emits zero activities

- **GIVEN** the global config has
  `features.historical_data.uptime = true` and
  `features.historical_data.vulnerabilities = false`
- **WHEN** an admin PATCHes the global config with the same values:
  `{"features": {"historical_data": {"uptime": true, "vulnerabilities": false}}}`
- **THEN** Fleet SHALL emit zero `enabled_historical_dataset` activities
- **AND** Fleet SHALL emit zero `disabled_historical_dataset` activities

#### Scenario: Toggling the same dataset on two fleets emits two distinct activities

- **GIVEN** fleet `1` and fleet `2` both have
  `features.historical_data.vulnerabilities = true`
- **WHEN** an admin PATCHes fleet `1` to disable `vulnerabilities`
- **AND** in a separate request PATCHes fleet `2` to disable `vulnerabilities`
- **THEN** Fleet SHALL emit two `disabled_historical_dataset` activities
- **AND** one SHALL carry `fleet_id = 1`
- **AND** the other SHALL carry `fleet_id = 2`

### Requirement: A mapping helper translates internal dataset names to config sub-keys

The `HistoricalDataSettings` type SHALL expose a method
`Enabled(dataset string) (bool, error)` that takes an **internal dataset
name** (the value returned by `Dataset.Name()`) and returns the corresponding
config sub-key value. The helper SHALL be the single canonical mapping point
between internal dataset names and config sub-keys; consumers SHALL NOT
hardcode the translation elsewhere.

The mapping SHALL be:

- `"uptime"` → `HistoricalData.Uptime`
- `"cve"` → `HistoricalData.Vulnerabilities`

For any dataset name not in the safelist, the helper SHALL return
`(false, error)` where the error wraps a clear "unknown dataset" message.

#### Scenario: Helper returns the Uptime field for the `"uptime"` dataset

- **GIVEN** `HistoricalDataSettings{Uptime: true, Vulnerabilities: false}`
- **WHEN** `Enabled("uptime")` is called
- **THEN** the call SHALL return `(true, nil)`

#### Scenario: Helper returns the Vulnerabilities field for the `"cve"` dataset

- **GIVEN** `HistoricalDataSettings{Uptime: true, Vulnerabilities: false}`
- **WHEN** `Enabled("cve")` is called
- **THEN** the call SHALL return `(false, nil)`

#### Scenario: Helper returns an error for an unknown dataset

- **GIVEN** `HistoricalDataSettings{Uptime: true, Vulnerabilities: true}`
- **WHEN** `Enabled("policy_compliance")` is called
- **THEN** the call SHALL return `(false, err)` where `err` is non-nil
- **AND** the error message SHALL identify the unknown dataset name

### Requirement: Strict decoding rejects unknown sub-keys under `historical_data`

`historical_data` sub-keys SHALL be validated by Fleet's existing strict
decoding. A payload (JSON or YAML) containing an unknown sub-key under
`historical_data` SHALL be rejected with a 4xx response (or, for GitOps,
a parse error from `fleetctl`).

#### Scenario: Misspelled sub-key is rejected

- **WHEN** an admin sends
  `POST /api/v1/fleet/config` with body
  `{"features": {"historical_data": {"vulnerabilites": false}}}`
  (note the typo)
- **THEN** the response SHALL be a 4xx error
- **AND** the saved global config SHALL be unchanged

### Requirement: Effective-collection rule (documented for consumer enforcement)

Fleet SHALL document, in both the GitOps YAML reference and the REST API
reference, that a dataset is collected for a given host only when both the
global sub-key AND the host's fleet sub-key are `true` — and that hosts
with no fleet (`team_id = 0` or `NULL`) follow the global value directly.
Enforcement of this rule lives in the cron-gating change that consumes the
`Enabled(dataset)` helper; this capability SHALL NOT itself enforce the
rule.

The effective state for `(host, dataset)` is:

```
collected(host, dataset) = global.historical_data.<key> AND fleet(host).historical_data.<key>
```

where `<key>` is the config sub-key for the dataset (per the mapping
helper).

#### Scenario: Documentation captures the AND rule

- **GIVEN** the GitOps YAML reference and the REST API reference
- **WHEN** an admin reads either reference
- **THEN** the doc SHALL state that a dataset is collected for a host only
  when the global sub-key is `true` AND the host's fleet sub-key is `true`
- **AND** the doc SHALL state that hosts with no fleet follow the global
  value directly
