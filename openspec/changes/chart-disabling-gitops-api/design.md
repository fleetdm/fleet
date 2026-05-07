## Context

Issue #44077 ships per-dataset "disable data collection" switches for the
dashboard charts. The work decomposes naturally:

1. **Config surface** (this change) — Go struct, defaults, PATCH/GitOps
   plumbing, audit activities, mapping helper.
2. **Cron gating** — orchestrator skips disabled datasets, passes
   enabled-fleet sets into `Dataset.Collect`.
3. **Data drop on disable** — truncate per-dataset rows / scrub per-fleet
   rows when a flag flips.
4. **Frontend** — Advanced card, Fleet Settings section, confirmation
   dialog, dashboard empty state.

This change is the foundation; the others are separate. The natural home
for these flags is `AppConfig.Features` because `Team.Config` already embeds
`Features`, giving us per-fleet overrides "for free."

## Decisions

### 1. Nested object, not flat booleans

```go
type Features struct {
    ...
    HistoricalData HistoricalDataSettings `json:"historical_data"`
}

type HistoricalDataSettings struct {
    Uptime          bool `json:"uptime"`
    Vulnerabilities bool `json:"vulnerabilities"`
}
```

Rather than `EnableUptimeHistoricalData bool` /
`EnableVulnerabilitiesHistoricalData bool` at the top level of `Features`.

**Rationale**: the dataset catalog is growing — policy compliance is next on
deck per the project notes. A flat-boolean approach pollutes the top-level
`Features` namespace and makes GitOps YAML noisier with each addition. The
nested object keeps all dataset toggles under one key and makes per-dataset
iteration trivial.

### 2. Config key vs internal dataset name — explicit mapping helper

The issue mandates the config keys `uptime` and `vulnerabilities`. The
internal dataset names (used in DB rows, API paths, code) are `uptime` and
`cve`. The mismatch needs a single mapping point.

Considered alternatives:

- **Implicit mapping in field names + JSON tags** — works, but every consumer
  must hardcode the `cve → Vulnerabilities` translation. Easy to drift.
- **Map-shaped settings** (`map[string]bool`) keyed by config-key strings —
  loses Go type safety on YAML unmarshal; strict decoding can't validate
  unknown sub-keys.
- **`ConfigKey()` method on the `Dataset` interface** — couples the dataset
  abstraction to a config layer concern.
- **Rename the internal dataset (`cve` → `vulnerabilities`)** — would ripple
  into DB rows, existing API paths, the cve-chart change, and existing
  tests.

Chosen: a method on the settings type itself.

```go
func (h HistoricalDataSettings) Enabled(dataset string) (bool, error) {
    switch dataset {
    case "uptime": return h.Uptime, nil
    case "cve":    return h.Vulnerabilities, nil
    default:       return false, fmt.Errorf("unknown dataset %q", dataset)
    }
}
```

**Rationale**:

- Keeps the struct shape (good for YAML strict decoding and grep-able field
  references on the Go side).
- One greppable place to update when a dataset is added.
- Safelist switch — no string interpolation into JSON paths or SQL.
- Returns an explicit error for unknown datasets, so a caller can surface a
  programmer error instead of silently treating it as "disabled."
- Lives on the type that owns the data, not on a separate adapter or the
  `Dataset` interface; the `Dataset` interface stays focused on collection.

### 3. Defaults true for new installs and upgrades — and a one-time backfill

`Features.ApplyDefaults()` sets `HistoricalData.Uptime = true` and
`HistoricalData.Vulnerabilities = true`.
`ApplyDefaultsForNewInstalls()` already delegates to `ApplyDefaults()`.

The datastore code calls these methods *before* unmarshaling stored JSON
on both read paths — `appConfigDB` and `teamFeaturesDB`. So existing rows
whose stored JSON simply *omits* `historical_data` read back with both
sub-fields `true`.

**The catch: stored `false` defeats the pre-unmarshal default.** Earlier
migrations such as `20260427134220` use the `updateAppConfigJSON` helper
(and team-config equivalents use the same inline pattern), which:

1. unmarshals the stored JSON into `fleet.AppConfig`,
2. runs the callback, and
3. re-marshals the **whole struct** back to the row.

The moment `HistoricalData` exists as a struct field in Go, step 3
persists its zero value (`false`) into stored JSON for every existing
row, even though the migration's callback never touched it. Subsequent
reads then see explicit `false` and the pre-unmarshal default loses.

Plain `bool` can't distinguish "not set" from "explicit false," so we
can't recover the intended default at read time. Three ways out:

- **(A) backfill migration** that writes `historical_data: {true, true}`
  on every existing row. Chosen.
- **(B) refactor `updateAppConfigJSON` to do `JSON_SET` instead of
  struct round-trip**, which addresses the trap class for every field.
  The right long-term fix but a large blast radius (rewriting ~12 merged
  migrations). Filed as follow-up backlog work.
- **(C) `optjson.Bool` storage type for the sub-keys, plus a
  post-unmarshal `FillDefaults` step.** Future-proofs additions to
  `HistoricalDataSettings` specifically (e.g., the next dataset
  toggle) but doesn't help any other plain-bool field added later, and
  pulls `optjson.Bool` into a wider surface (test literals, GET
  responses, spec wording). Considered and rejected as
  disproportionate for this PR.

The "any-window clobber" risk that normally complicates a backfill
migration — that an admin could deliberately set `false` between when
this code ships and when the backfill runs, only for the backfill to
clobber it — does not apply here: this code lands in 4.85.0, which is
the first release exposing the toggle. No admin can have set `false`
prior to the backfill running.

The backfill is co-located with migration `20260423161823_AddHostSCDData`
(the chart data table), so the same migration that introduces chart
storage also turns the toggles on. AppConfig uses `updateAppConfigJSON`
(safe here because the values being written are non-zero, so the
round-trip preserves them); team configs use `JSON_MERGE_PATCH` to add
or replace `features.historical_data` per row without round-tripping
the whole TeamConfig struct.

**Rationale for defaults `true`**: existing deployments should keep
getting the charts they'll start seeing once the dashboard chart UI
lands. Defaulting off on upgrade would silently break the "dashboards
just work" story. `EnableSoftwareInventory` defaulting on only for new
installs is a different concern (consent for privacy-sensitive
collection); the historical-data rollups here are derived from data
Fleet already collects, so there's no consent wrinkle.

### 4. PATCH merge — global is free, fleet uses an `optjson.Bool` payload subset

The global and fleet endpoints have structurally different request
decoders, so PATCH merge requires different mechanisms:

**Global (`POST /api/v1/fleet/config`):** `ModifyAppConfig` takes raw JSON
bytes (`p []byte`) and unmarshals them into the existing config. Go's
JSON decoder recurses into nested structs and only touches fields present
in the payload. So `{"features": {"historical_data": {"vulnerabilities":
false}}}` flips `vulnerabilities` and leaves `uptime` untouched. No
custom merge logic.

**Fleet (`PATCH /api/v1/fleet/fleets/{id}`):** `ModifyTeam` takes a
parsed `TeamPayload`, not raw bytes. `TeamPayload` has no `Features`
field today, so `features.historical_data` would be silently dropped.
Two ways to wire it:

1. **Refactor `ModifyTeam` to take raw JSON** — large surface change,
   touches every other field on the endpoint.
2. **Add a focused `Features` field to `TeamPayload`** — small surface
   change, follows the existing `MDM *TeamPayloadMDM` /
   `WebhookSettings *TeamWebhookSettings` pattern.

Option 2 chosen. New types in `server/fleet/teams.go`:

```go
type TeamPayloadFeatures struct {
    HistoricalData *HistoricalDataPayload `json:"historical_data"`
}

type HistoricalDataPayload struct {
    Uptime          optjson.Bool `json:"uptime"`
    Vulnerabilities optjson.Bool `json:"vulnerabilities"`
}
```

`TeamPayloadFeatures` is a *payload-only subset* of `Features`. Only the
sub-fields it declares can be written via this endpoint; other Features
fields (`enable_host_users`, `enable_software_inventory`,
`additional_queries`, `detail_query_overrides`) remain settable per-fleet
only via `/spec/fleets`. The narrow surface keeps a new auth-review
question off the table for v1: this endpoint can already set fleet
identity and integrations, but historically not Features.

`HistoricalDataPayload` uses `optjson.Bool` per sub-field instead of plain
`bool` so a sub-key omitted from the PATCH body retains its current
stored value (`Valid == false`), while a sub-key explicitly sent as
`false` flips the stored value (`Valid == true, Value == false`). This is
exactly how `MDM.EnableDiskEncryption` already behaves on this endpoint.

`ModifyTeam` applies the partial:

```go
oldHistoricalData := team.Config.Features.HistoricalData
if payload.Features != nil && payload.Features.HistoricalData != nil {
    if payload.Features.HistoricalData.Uptime.Valid {
        team.Config.Features.HistoricalData.Uptime = payload.Features.HistoricalData.Uptime.Value
    }
    if payload.Features.HistoricalData.Vulnerabilities.Valid {
        team.Config.Features.HistoricalData.Vulnerabilities = payload.Features.HistoricalData.Vulnerabilities.Value
    }
}
// SaveTeam, then diff old vs new and emit activities.
```

The wire shape is identical to global
(`{features: {historical_data: {...}}}`), the storage location is
identical (`team.Config.Features.HistoricalData`), and the GET response
shape is identical (`features.historical_data` on the read side, by
virtue of `Team.MarshalJSON` embedding `TeamConfig`). Only the decoder
plumbing differs.

**Strict decoding:** the global `/config` endpoint uses
`appConfig.EnableStrictDecoding()` to reject unknown fields. The fleet
PATCH endpoint has no such mechanism today; unknown sub-fields under
`features` (e.g. an admin trying `features.enable_host_users`) are
silently ignored. That's the existing convention for this endpoint, and
this change does not introduce strict decoding for it.

**GitOps overwrite + client-side defaulting:**
`ApplySpecOptions.Overwrite = true` is what the gitops client passes when
applying global config or team specs (see `server/service/client.go`).
That mode replaces `Features` wholesale, which by itself would zero any
sub-key not explicitly present in the YAML — including
`historical_data` sub-keys, even though their `ApplyDefaults` value is
`true`. Concretely, an admin who runs `fleetctl gitops` with a YAML
that doesn't mention `historical_data` would see both datasets silently
disabled on every apply, contradicting the upgrade-friendly default of
`true` documented elsewhere.

To avoid that, the gitops client SHALL inject default `true` values for
any `historical_data` sub-key not explicitly set in the YAML, mirroring
the existing carve-out for `enable_software_inventory` at
`server/service/client.go:1965-1978`. The same defaulting SHALL be
applied on the team-spec gitops path. After this change:

- YAML with no `features` block → client adds
  `features.historical_data: {uptime: true, vulnerabilities: true}` to
  the request payload.
- YAML with a `features` block but no `historical_data` → client adds
  the full `historical_data: {uptime: true, vulnerabilities: true}`.
- YAML with `historical_data: {uptime: false}` → client adds the
  missing `vulnerabilities: true` sub-key, leaving `uptime: false`
  intact.
- YAML with both sub-keys explicit → no change.

This injection is gitops-only. `fleetctl apply` uses
`Overwrite=false`, which is partial-merge by JSON unmarshal: omitted
fields are left at their prior stored value. No client-side injection
is needed for that path; trying to add one would silently re-enable
fields the admin has explicitly disabled via the API or UI, which is
the wrong default for `apply`.

### 5. Activity per sub-field, scoped with `fleet_id` / `fleet_name`

```go
type ActivityTypeEnabledHistoricalDataset struct {
    Dataset   string  `json:"dataset"`
    FleetID   *uint   `json:"fleet_id"`
    FleetName *string `json:"fleet_name"`
}

type ActivityTypeDisabledHistoricalDataset struct {
    Dataset   string  `json:"dataset"`
    FleetID   *uint   `json:"fleet_id"`
    FleetName *string `json:"fleet_name"`
}
```

Activity-type strings: `enabled_historical_dataset` /
`disabled_historical_dataset`.

One activity per sub-field that flipped, per request. No-op PATCHes emit
zero activities. Global emits with `fleet_id` / `fleet_name` `nil`;
per-fleet emits with both populated.

The `dataset` payload uses the **config key**, not the internal dataset
name — i.e. `"vulnerabilities"`, not `"cve"`. The audit log is admin-facing,
and admins encounter the config key in YAML and (eventually) the UI.
Surfacing the internal name would force admins to learn an undocumented
translation.

**Rationale**: matches the existing "global or fleet-scoped" activity
pattern (e.g. `ActivityTypeEnabledMacosDiskEncryption` at
`server/fleet/activities.go:754`). No `renameto` JSON tags because these
types are new and have no legacy naming to migrate from.

### 6. Effective-value semantics live in the consumer, not here

A dataset is collected for a host IFF
`global.HistoricalData.<field>` AND `fleet(host).HistoricalData.<field>`.
No-team hosts follow the global value directly.

This change *does not* enforce that semantic — it ships the data shape and
the mapping helper. The cron-side enforcement (which loads global +
team-scoped configs and computes the enabled-fleet set per dataset) lives
in a separate change against `server/chart/`. Reason: the cron lives on
the dashboard-charts-backend branch, has its own datastore adapter, and is
naturally where the AND rule executes.

This change documents the AND rule in the spec deltas (so the contract is
captured even though it isn't yet enforced) but does not add cron code.

## Risks / Trade-offs

- **[Trade-off] GitOps re-enables out-of-band disables.** Because the
  client injects `historical_data: {uptime: true, vulnerabilities: true}`
  defaults whenever the YAML doesn't explicitly set them, an admin who
  disables a dataset via the API or UI on a GitOps-managed deployment
  will see the next `fleetctl gitops apply` re-enable it — unless the
  YAML pins the disabled value. This is the same trade-off
  `enable_software_inventory` carries today and is the right default
  for the upgrade-friendly "dashboards just work" intent: GitOps state
  is the source of truth on those deployments, and out-of-band
  disables are unsanctioned. Documented in the YAML reference: "If
  you manage Fleet via GitOps and want a dataset disabled, include
  `historical_data` explicitly in your YAML — otherwise each apply
  defaults to enabled."
- **[Risk] Polarity confusion in activities vs UI.** The future UI inverts
  the bool to "Disable X" checkboxes, but the API and audit log use the
  positive `enabled / disabled` polarity directly. An admin who reads
  `disabled_historical_dataset { dataset: "vulnerabilities" }` sees the
  same word "disabled" they clicked in the UI; the polarity matches. The
  trap is internal: a code reader who sees `Vulnerabilities: false` in a
  config might forget that means "disabled, do not collect." Mitigated by
  Go field naming (`HistoricalData.Vulnerabilities` reads as "is
  vulnerabilities historical-data on?") and by the `Enabled(dataset)`
  helper enforcing the read pattern.
- **[Trade-off] Two activity types vs one with a verb payload.** The audit
  log gains one "verb" per toggle direction. Existing tooling that scrapes
  activity types by name needs to know about the new types. Accepted:
  Fleet has many `enabled_*` / `disabled_*` pairs and integrators
  recognize the pattern.
- **[Trade-off] Cron consumer arrives separately.** Once this change
  merges, `historical_data` is settable but not enforced — disabling a
  dataset has no effect on collection until the cron-gating change lands.
  The activities and config still record correctly. Mitigation: land the
  cron-gating change close behind, and don't surface UI checkboxes (a
  third change) until at least the cron gating is in.
- **[Risk] Strict decoding rejects unknown dataset keys.** A YAML or JSON
  payload with a typo (`historical_data: { vulnerabilites: false }`) is
  rejected by `EnableStrictDecoding`. This is the desired behavior — the
  alternative is silently ignoring the typo and leaving the dataset
  enabled. Documented in the YAML reference.
