## Shared types & helpers

- [x] **Move chart types to `frontend/interfaces/charts.ts`**
  Move `frontend/pages/DashboardPage/cards/ChartCard/types.ts` to
  `frontend/interfaces/charts.ts`. No shape changes. Update imports in
  `ChartCard.tsx`, `LineChartViz.tsx`, `CheckerboardViz.tsx`, and
  `ChartFilterModal/`.

- [x] **Add `HISTORICAL_DATA_CONFIG_KEYS` constant**
  Export `const HISTORICAL_DATA_CONFIG_KEYS = ["uptime", "vulnerabilities"]
  as const;` and a derived `HistoricalDataConfigKey` type in
  `interfaces/charts.ts`.

- [x] **Add `DATASET_CONFIG_KEY` mapping**
  `Record<internalName, HistoricalDataConfigKey>` covering every dataset
  the frontend knows about. v1: `{ uptime: "uptime", cve: "vulnerabilities" }`.

- [x] **Add `DATASET_LABEL` mapping**
  `Record<HistoricalDataConfigKey, string>`:
  `{ uptime: "Hosts online", vulnerabilities: "Vulnerabilities" }`.

- [x] **Add `isHistoricalDataEnabled` helper**
  `(global, fleet, configKey) => boolean`. Both args optional; missing
  means `true` (no-op default). Returns `globalEnabled && fleetEnabled`.

- [x] **Unit tests for the helpers**
  Cover: missing global, missing fleet, both true, global false / fleet
  true (and vice versa), unknown config key (TS catches at compile time;
  runtime test for the lookup wrappers).

## Config interfaces & mocks

- [x] **Extend `IFeatures` with `historical_data`**
  In `frontend/interfaces/config.ts` (and team config interface if
  separate), add
  `historical_data: { uptime: boolean; vulnerabilities: boolean }`.

- [x] **Update mocks**
  `frontend/__mocks__/configMock.ts` and any team config mock get the
  new sub-object with both keys `true`.

## Global Advanced page

- [x] **Add "Activity & data retention" subheading**
  In `Advanced.tsx`, add `<SectionHeader title="Activity & data retention" />`
  inside the form, above the Save button.

- [x] **Add "Disable hosts online" checkbox**
  Form key `disableHostsActive`. Wrapped in `GitOpsModeTooltipWrapper`.
  Tooltip: explains what the toggle does ("When enabled, Fleet stops
  collecting hourly hosts-active data...").

- [x] **Add "Disable vulnerabilities" checkbox**
  Form key `disableVulnerabilities`. Wrapped in `GitOpsModeTooltipWrapper`.
  Tooltip per the same pattern.

- [x] **Wire load mapping**
  Initialize form state with
  `disableHostsActive = !appConfig.features.historical_data.uptime`,
  `disableVulnerabilities = !appConfig.features.historical_data.vulnerabilities`.

- [x] **Wire submit mapping**
  In `formDataToSubmit`, include
  `features: { historical_data: { uptime: !disableHostsActive,
  vulnerabilities: !disableVulnerabilities } }`. Verify it merges
  cleanly with any other `features` writes the page already does.

- [x] **Compute "what's flipping" before submit**
  Compare current form state vs originally-loaded values. Build
  `disabledNow: HistoricalDataConfigKey[]` of datasets moving
  enabled → disabled. If empty, save without confirmation.

- [x] **Hook up confirmation modal**
  When `disabledNow.length > 0`, show
  `<ConfirmDataCollectionDisableModal scope="global" datasets={disabledNow} />`
  before issuing the PATCH.

- [x] **Unit tests for Advanced.tsx changes**
  - Renders both checkboxes inside the new section
  - Load inversion correct
  - Submit produces expected payload
  - Confirmation modal opens only on enable→disable transitions
  - GitOps mode disables both checkboxes

## Per-fleet TeamSettings page

- [x] **Add "Activity & data retention" subheading**
  After the Host expiry settings section. Lives inside the new
  `HistoricalDataTeamControls` subcomponent.

- [x] **Add per-fleet checkboxes**
  Two checkboxes mirroring the global page; wrapped in
  `GitOpsModeTooltipWrapper`. Form keys
  `disableHostsActive`, `disableVulnerabilities`.

- [x] **Implement global lockout**
  Each checkbox is `disabled` when the corresponding global sub-key is
  `false`. When disabled, set `labelTooltipContent="Disabled globally"`.
  Visible value still reflects the fleet's stored preference.

- [x] **Wire load mapping**
  Initialize from `teamConfig.features.historical_data` with the same
  inversion as the global page. `appConfig` is already loaded; reach
  through it for the lockout determination.

- [x] **Wire submit mapping**
  Include `features: { historical_data: { ... } }` in the
  `teamsAPI.update` payload. Required adding `features` to
  `IUpdateTeamFormData` and a pass-through in the `update` method.

- [x] **Reuse the confirmation modal with fleet scope**
  `<ConfirmDataCollectionDisableModal scope="fleet"
  fleetName={teamConfig.name} datasets={disabledNow} />`.

- [x] **Unit tests for TeamSettings.tsx changes**
  Coverage delivered via the `HistoricalDataTeamControls` subcomponent
  tests (`components/HistoricalDataTeamControls/...tests.tsx`): lockout
  rendering, "Disabled globally" tooltip, stored fleet value preserved
  while locked, locked checkbox does not call onChange. Submit-payload
  shape is validated by the type-checked call to `teamsAPI.update`.
  - Lockout renders correctly when global is off
  - Tooltip surfaces "Disabled globally"
  - Stored fleet value is preserved across global lockout/unlockout
  - Submit payload correct
  - GitOps mode disables both checkboxes

## Confirmation modal

- [x] **Create `ConfirmDataCollectionDisableModal.tsx`**
  Props: `scope: "global" | "fleet"`, `datasets: HistoricalDataConfigKey[]`,
  `fleetName?: string`, `isUpdating: boolean`, `onConfirm: () => void`,
  `onCancel: () => void`. Body lists dataset labels via `DATASET_LABEL`
  prominently. Buttons: "Save and disable" (destructive variant) /
  "Cancel". A single click on Save and disable proceeds.

- [x] **Test the modal**
  - Renders correct dataset labels prominently
  - Scope-aware copy (global vs fleet, including fleet name)
  - Save and disable calls `onConfirm`
  - Cancel calls `onCancel`

## Dashboard empty state

- [x] **Load team config in `DashboardPage`**
  Used the already-loaded `teams` list; team's `features.historical_data`
  is read from `teams.find(t => t.id === currentTeamId)?.features`. No
  extra fetch needed.

- [x] **Compute `historicalDataEnabled` map**
  In `DashboardPage`, derive `{ uptime, vulnerabilities }` via
  `isHistoricalDataEnabled(global, fleet, configKey)`.

- [x] **Pass `historicalDataEnabled` into `ChartCard`**
  Added the prop to `IChartCardProps`.

- [x] **Implement `DataCollectionDisabledState` component**
  Sibling of `LineChartViz` / `CheckerboardViz` in the ChartCard
  directory. Renders an empty-state panel with the dataset label and a
  "Turn on" button: deeplinks to the current fleet's settings when
  team-scoped, otherwise to `/settings/organization/advanced`. Button
  is hidden (and copy swaps to "Ask an admin to turn on…") for users
  without access to the destination — gated via `AppContext`
  (`isGlobalAdmin` / `isTeamAdmin`).

- [x] **Render the empty state when disabled**
  In `ChartCard`, look up
  `historicalDataEnabled[DATASET_CONFIG_KEY[currentDataset.name]]`.
  Datasets without a config-key mapping default to enabled.
  Disabled-state branch in `renderChart()`; the chart `useQuery` is
  disabled via `enabled: datasetCollectionEnabled` to skip the wasted
  request.

- [x] **Update existing ChartCard tests**
  Added cases for the empty state and for normal rendering when
  `historicalDataEnabled` is supplied.

## Activity feed rendering

- [x] **Add `EnabledHistoricalDataset` / `DisabledHistoricalDataset` to `ActivityType`**
  In `frontend/interfaces/activity.ts`, extend the `ActivityType` enum with
  the two new values: `EnabledHistoricalDataset = "enabled_historical_dataset"`
  and `DisabledHistoricalDataset = "disabled_historical_dataset"`.

- [x] **Type the activity details payload**
  Add an `IActivityDetails`-shaped entry (or extend the existing union)
  capturing `dataset: string`, `fleet_id: number | null`,
  `fleet_name: string | null` for the two new activity types. Match the
  surrounding patterns in the file.

- [x] **Add tagged-template renderers in `GlobalActivityItem.tsx`**
  Two renderers — one for enable, one for disable — that:
  - Look up the friendly label via `DATASET_LABEL[details.dataset]`.
    On miss, fall back to a sentence-cased rendering of the raw config
    key (`_` / `-` → space, first letter capitalized, rest lowercase).
  - Render `... data collection for <b>{label}</b>` on global, and
    `... data collection for <b>{label}</b> for the <b>{fleet_name}</b> fleet`
    when `fleet_name` is present.
  - Wire both into the switch in the appropriate place.

- [x] **Tests for `GlobalActivityItem`**
  - Global enable renders "Enabled data collection for **Hosts online**."
  - Global disable renders "Disabled data collection for **Vulnerabilities**."
  - Fleet-scoped enable includes "for the **Engineering** fleet."
  - Fleet-scoped disable includes "for the **Engineering** fleet."
  - Unknown dataset key (e.g. `policy_compliance`) renders as a
    sentence-cased label ("Policy compliance") without throwing.

- [x] **Update `frontend/__mocks__/activityMock.ts` if needed**
  Skipped: tests use inline `createMockActivity({ type, details })` fixtures;
  no shared mock entry needed.

## Documentation

- [x] **Update user-facing docs**
  Skipped: there's no dedicated dashboard docs page in `docs/01-Using-Fleet/`
  to amend (no `dashboard.md`), and the in-product copy on the empty state
  itself plus its link to Advanced settings provides the user-facing
  context. The API/GitOps docs for `features.historical_data` are the
  responsibility of the upstream `chart-disabling-gitops-api` change.


## Verification

- [x] **Run `make lint-js`** and resolve any issues.
  Result: 0 errors, 348 pre-existing warnings (unchanged from baseline).
- [x] **Run `yarn test`** and confirm all new and existing tests pass.
  Result: 245 suites passed, 1787 tests passed.
- [ ] **Manual smoke test**
  - Disable globally → empty state on the dashboard, fleet checkboxes
    locked.
  - Re-enable globally → fleet checkboxes unlock with their stored
    state.
  - Disable per-fleet → empty state shows only when that fleet is
    selected on the dashboard.
  - Toggle in GitOps mode → both checkboxes are greyed out.
  - Save with no changes → no confirmation modal.
  - Save re-enabling a previously-disabled dataset → no confirmation
    modal.
  - Save disabling a dataset → confirmation modal lists the right
    dataset(s); Cancel leaves config untouched.
  - In the confirmation modal: clicking "Save and disable" issues the
    PATCH and closes the modal.
  - Activity feed shows correctly-formatted rows for enable/disable, both
    global and fleet-scoped, with the dataset label and (when scoped) the
    "for the **X** fleet" suffix.
