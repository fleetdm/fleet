## Why

Issue #44077 ("Allow disabling data collection for charts") needs frontend UI
on top of the now-shipped GitOps/API surface (`features.historical_data`).
This change delivers the three frontend touchpoints the API change called out
as out-of-scope:

- Global checkboxes on `/settings/organization/advanced` to toggle each
  dataset's collection.
- Per-fleet checkboxes on the fleet settings page, with the `global AND fleet`
  precedence surfaced in the UI (fleet checkbox locked when global is off).
- A "Data collection disabled" empty state on the dashboard's chart card so
  admins understand why a chart isn't rendering.

It also folds in a small refactor — moving `ChartCard/types.ts` into
`frontend/interfaces/charts.ts` — because the new label/config-key mapping
helpers belong with the chart types and three different surfaces (Advanced,
TeamSettings, ChartCard) need to share them.

## What Changes

### Shared types & helpers

- Move `frontend/pages/DashboardPage/cards/ChartCard/types.ts` to
  `frontend/interfaces/charts.ts`. Update existing imports in `ChartCard.tsx`,
  `LineChartViz.tsx`, `CheckerboardViz.tsx`, and `ChartFilterModal/`. No
  shape changes.
- Add to `interfaces/charts.ts`:
  - `HISTORICAL_DATA_CONFIG_KEYS` — the v1 set of config keys
    (`"uptime" | "vulnerabilities"`).
  - `DATASET_CONFIG_KEY` — `Record<internalName, configKey>` mapping
    (`{ uptime: "uptime", cve: "vulnerabilities" }`). The single source of
    truth for the internal-name ↔ config-key translation on the frontend,
    paralleling the backend `HistoricalDataSettings.Enabled(dataset)` helper.
  - `DATASET_LABEL` — `Record<configKey, string>` (`{ uptime: "Hosts active",
    vulnerabilities: "Vulnerabilities" }`). Used by checkboxes, confirmation
    modal, and empty state.
  - `isHistoricalDataEnabled(global, fleet, configKey)` — the AND rule. Pure
    function; both args optional so callers without a fleet (Advanced page)
    pass only `global`.

### Config interfaces & mocks

- Extend `IFeatures` (in `frontend/interfaces/config.ts` and
  `frontend/interfaces/team.ts` if it has its own) with
  `historical_data: { uptime: boolean; vulnerabilities: boolean }`.
- Update `frontend/__mocks__/configMock.ts` and any team mocks to include
  the new sub-object with both keys `true`.

### Global Advanced page

- In `frontend/pages/admin/OrgSettingsPage/cards/Advanced/Advanced.tsx`:
  - Add a new `<SectionHeader title="Activity & data retention" />` at the
    bottom of the form (above the Save button). This introduces SectionHeader
    inside `SettingsSection` for this page, matching the precedent in
    `TeamSettings.tsx`.
  - Under the new subheading, add two `<Checkbox>` inputs:
    - "Disable hosts active" — form key `disableHostsActive`, bound to
      `!features.historical_data.uptime`
    - "Disable vulnerabilities" — form key `disableVulnerabilities`, bound to
      `!features.historical_data.vulnerabilities`
  - Both checkboxes wrapped in `<GitOpsModeTooltipWrapper>` so they grey out
    when GitOps mode is enabled, matching surrounding fields.
  - On submit, invert the form booleans back into
    `features.historical_data.{uptime, vulnerabilities}`.
  - On submit, if any dataset moves from collecting → disabled (compared to
    the originally-loaded values), open `<ConfirmDataCollectionDisableModal>`
    before issuing the PATCH. Modal lists the dataset(s) and is global-scoped.
  - Available on **both Free and Premium tiers** (matches existing Advanced
    page behavior).

### Per-fleet TeamSettings page

- In `frontend/pages/admin/TeamManagementPage/TeamDetailsWrapper/TeamSettings/
  TeamSettings.tsx`:
  - Add a `<SectionHeader title="Activity & data retention" />` after the
    existing Host expiry settings section.
  - Two checkboxes mirroring the global page, bound to
    `!teamConfig.features.historical_data.{uptime, vulnerabilities}`.
  - Each checkbox is `disabled` when the **global** sub-key is `false` (i.e.
    globally disabled), with `labelTooltipContent="Disabled globally"`. The
    fleet's own value is preserved underneath the lock — flipping global
    back on re-exposes whatever the fleet had stored.
  - Wrap in `<GitOpsModeTooltipWrapper>` per the page's existing pattern.
  - Confirmation modal on save with fleet-scoped copy, identical mechanics
    to the global page.
  - Per-fleet UI is only reachable on Premium (TeamSettings page is already
    premium-gated); no extra check needed here.

### Confirmation modal (shared)

- New `ConfirmDataCollectionDisableModal.tsx`. Inputs: scope ("global" |
  "fleet"), the `configKey[]` being disabled, fleet name (when scope=fleet).
- Body lists the dataset labels prominently (e.g. "Hosts active",
  "Vulnerabilities") and explains what disabling means: collection
  stops; previously collected data is not retained.
- Buttons: "Save and disable" (destructive variant) / "Cancel". A
  single click on the destructive button proceeds; there is no extra
  type-to-confirm step.

### Dashboard empty state

- In `DashboardPage.tsx`, load both global config (already loaded) and the
  current fleet's config (when `currentTeamId` is a positive fleet ID;
  otherwise pass `undefined`). Pass `historicalDataEnabled: { uptime,
  vulnerabilities }` (post-AND) into `<ChartCard>`.
- `ChartCard.tsx` reads `historicalDataEnabled[currentDataset.configKey]`.
  When `false`, render a new `<DataCollectionDisabledState>` in place of the
  visualization.
  - Empty-state copy mentions which dataset is disabled and points to
    `/settings/organization/advanced` (link only — no per-fleet deeplink to
    avoid a permission/team-context maze).
  - The dataset selector, time range, and filter gear remain functional —
    the user can switch to an enabled dataset.
- The lookup uses `DATASET_CONFIG_KEY` so datasets that don't yet have a
  config-key mapping (future additions) fall through as "enabled" rather
  than throwing.

## Capabilities

### Modified Capabilities

- `chart-dashboard-ui` — adds the "data collection disabled" empty-state
  requirement.

### Added Capabilities

- `chart-historical-data-settings-ui` — describes the global Advanced
  checkboxes, per-fleet TeamSettings checkboxes, the global-AND-fleet
  lockout UX, the save-time confirmation modal, and the GitOps-mode
  lockout for both surfaces.

## Impact

- **No backend work.** All endpoints, defaults, and audit activities are
  already shipped in the `chart-disabling-gitops-api` change. This change
  is pure frontend.
- **Refactor blast radius is small.** The `ChartCard/types.ts` move touches
  4 files (`ChartCard.tsx`, `LineChartViz.tsx`, `CheckerboardViz.tsx`,
  `ChartFilterModal/`). All other consumers come from this same change.
- **Empty state uses post-AND value.** `DashboardPage` does the merge once;
  `ChartCard` only sees the effective bool, not separate global/fleet bits.
  Avoids duplicating the AND rule across components.
- **Confirmation modal fires only on enable→disable transitions.** No-op
  saves and re-enabling produce no modal.

## Out of Scope

- **Cron gating** in `server/chart/`. Separate change; uses the
  `Enabled(dataset)` helper shipped by the API change.
- **Data drop on disable.** Separate change; the modal mentions that
  collection stops, but does not promise immediate deletion (the eventual
  truncate is the cron's concern, not the frontend's).
- **Relocating existing "Delete activities" / "Activity log retention
  window"** under the new "Activity & data retention" subheading. The
  subheading is forward-looking; existing fields stay where they are
  to keep the diff minimal.
- **Per-fleet deeplink in the empty-state copy.** Empty state points to
  the global page only; routing into the right fleet's settings would
  require team-permission checks the empty state shouldn't own.
- **A third dataset (`policy-compliance`).** When it lands, it will need
  a config-key mapping, label, and a row in both checkbox sections —
  but the data-driven `DATASETS` array and the helper-based lookups
  here are designed to absorb additions without further refactoring.
