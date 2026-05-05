## Context

The `chart-disabling-gitops-api` change shipped the config shape, defaults,
and audit activities for `features.historical_data`. The cron-gating and
data-drop changes are deferred. This change wires the config to the three
remaining frontend surfaces:

1. Global Advanced settings — admins toggle collection per dataset.
2. Per-fleet TeamSettings — admins toggle per dataset, with a UI lockout
   when global says "off."
3. Dashboard chart card — when collection is off (globally or for the
   current fleet), the chart shows an empty state instead of an
   incomplete/empty visualization.

The frontend has three places that need the same dataset-name/label
information, plus the AND-precedence rule. Each surface re-deriving these
would invite drift; centralizing them is the main design call.

## Decisions

### 1. Move chart types to `frontend/interfaces/charts.ts`

`frontend/pages/DashboardPage/cards/ChartCard/types.ts` becomes
`frontend/interfaces/charts.ts`. Same shapes; new home.

**Rationale**: `frontend/interfaces/` is the established home for
cross-cutting type definitions (per `.claude/rules/fleet-frontend.md`:
"Interface files live in `frontend/interfaces/` with `I` prefix"). The
new helpers (`DATASET_CONFIG_KEY`, `DATASET_LABEL`,
`isHistoricalDataEnabled`) need to be imported from at least three
locations: `Advanced.tsx`, `TeamSettings.tsx`, and the chart card. A
file under a single page directory is the wrong location for that.
Moving the existing types alongside means everything chart-related has
one canonical import path.

**Tradeoff**: small diff churn in the four current importers. Cheap.

### 2. Form state stores the inverted "disable" boolean

The Advanced page already uses "Disable X" framing (`disableLiveQuery`,
`disableScripts`, `disableAIFeatures`, `disableQueryReports`). The new
checkboxes follow suit: form keys `disableHostsActive` and
`disableVulnerabilities`, both `true` when the user wants collection off.

At load, `disable* = !appConfig.features.historical_data.<key>`.
At submit, `historical_data.<key> = !form.disable*`.

**Rationale**: matches surrounding page idiom; "Disable" is also the
issue-title wording. The inversion happens in two well-defined places
(load mapper and submit mapper), not scattered.

**Alternative considered**: store the raw "enable" bool in form state,
flip just for the label. Rejected — would diverge from the rest of the
Advanced page's naming and be subtly confusing when reading the form
state.

### 3. The AND rule lives in one helper, applied at the boundary

```ts
function isHistoricalDataEnabled(
  global: HistoricalDataSettings | undefined,
  fleet: HistoricalDataSettings | undefined,
  configKey: HistoricalDataConfigKey,
): boolean {
  const g = global?.[configKey] ?? true;   // tolerate missing config
  const f = fleet?.[configKey] ?? true;    // tolerate "all teams" / no fleet
  return g && f;
}
```

`DashboardPage` calls this once per config key, packs the result into
`historicalDataEnabled: { uptime: bool, vulnerabilities: bool }`, and
passes it to `ChartCard`. `ChartCard` only sees the post-AND boolean
and never knows or cares which side disabled it.

**Rationale**: the chart card's job is "render or empty-state." It
shouldn't reproduce business rules. If the rule changes (e.g. add an
"inherited" tri-state later), only the helper and `DashboardPage`
update.

**Tradeoff**: empty-state copy can't say "globally disabled" vs "fleet
disabled" without re-loading both bits. Decision: copy says only
"data collection is disabled" with a link to the Advanced page. If
admins want fleet-level visibility, they go to TeamSettings. Avoids
duplicating the precedence logic in the chart card just for the copy.

### 4. Per-fleet checkbox: `disabled` lockout, not hide-when-globally-off

When global `historical_data.uptime = false`, the fleet's "Disable hosts
active" checkbox renders as `disabled` with
`labelTooltipContent="Disabled globally"`. The visible state still
reflects the fleet's *stored* value (so admins can see what the fleet
will revert to if global flips on).

**Rationale**: this matches the precedent set by `TeamHostExpiryToggle`,
which renders the team's setting alongside the global state. Hiding the
checkbox would surprise admins who saved a per-fleet preference and
later wonder "where did it go?"

**Alternative considered**: render the checkbox as if it were unchecked
(reflecting the effective `false`) when locked. Rejected — would
quietly overwrite the fleet's stored preference visually.

### 5. Confirmation modal at Save, not at toggle

The modal opens on form submission, only when at least one dataset
flipped from collecting → disabled compared to the originally-loaded
config. Re-enabling and no-op saves submit without prompting.

**Rationale**: matches the user's intent ("confirm saving"). Toggling
checkboxes mid-edit shouldn't keep popping modals; the destructive
moment is when the change persists. The diff calculation is a single
comparison between original and form state — trivial.

**What the modal does NOT promise**: immediate data deletion. The data
drop is the cron/data-drop change's responsibility and may be async.
The modal copy: "Disabling will stop collecting [dataset] going
forward. Previously collected data may continue to be retained." This
keeps the frontend honest about the cross-change boundary.

### 5a. Confirmation is a single click

The confirmation modal opens, names the dataset(s) being disabled
prominently, and exposes a single destructive "Save and disable" action
alongside Cancel. There is no type-to-confirm input.

**Rationale**: an earlier iteration required the user to type the
comma-separated config keys of each dataset being disabled. Product
review pulled that out — the destructive-modal interception alone is
sufficient friction for this action. Re-enabling is one click; it
felt asymmetric (and overly hostile) to require typing on the disable
side.

**What we kept from the earlier design**: the modal still names the
datasets clearly (using the human-readable labels — `DATASET_LABEL`),
warns about consequences, and is scope-aware (global vs fleet copy).
Those carry the "are you sure" weight without the friction.

**Reuse**: same modal/component on both the global and fleet pages.
The only difference between the two is scope-aware copy.

### 6. `DashboardPage` loads team config when needed

`DashboardPage` already loads `appConfig`. To compute the AND rule for
the active fleet, it also needs that fleet's config. When
`currentTeamId` is a positive ID, fetch
`teamsAPI.load(currentTeamId).team.config.features.historical_data`.
When it's `-1` (All teams) or `0` (No team), pass `undefined` for the
fleet side — `isHistoricalDataEnabled` treats missing as `true`.

**Rationale**: the dashboard already does team-scoped data fetching
elsewhere; piggybacking on the existing `useTeamIdParam` hook is
straightforward. Pushing the merge up to `DashboardPage` keeps
`ChartCard` a presentational component.

**Tradeoff**: adds a `teamsAPI.load` call to the dashboard. Cached via
React Query (stale time matches existing patterns); negligible cost.

### 7. Tier handling

- Global Advanced page: visible on Free **and** Premium. The Advanced
  page already renders for Free; new checkboxes inherit that.
- Per-fleet TeamSettings page: only reachable on Premium. The new
  checkboxes inherit that gating; no per-component checks needed.

### 8. GitOps-mode lockout

Both surfaces wrap the new checkboxes in `<GitOpsModeTooltipWrapper>`,
matching every other writable field on these pages. When GitOps mode
is on, the API would reject the write anyway; the wrapper provides
the visual + tooltip.

## Risks / Trade-offs

- **Empty-state copy is generic** (decision 3). Admins seeing the
  empty state on the dashboard need to check both global Advanced
  *and* per-fleet TeamSettings to know who turned it off. Mitigation:
  the global Advanced link lands them on the most likely culprit
  (since fleet-level disable is rarer than global disable).
- **Free-tier admins on the Advanced page can disable
  `vulnerabilities`** even though Free deployments don't have the
  vulnerability chart on the dashboard yet. The setting takes effect
  silently in case they later add Premium / a vuln chart. Acceptable;
  matches how other Free-tier feature flags behave.
- **The interface relocation in decision 1** is a refactor mixed
  with the feature change. We could split it into a separate
  pre-PR. Decision: keep together — the new helpers belong with the
  types from day one, and reviewers benefit from seeing the full
  picture.

## Migration plan

None — this is a frontend-only change reading shipped backend data.
Existing config rows (with both sub-keys defaulting `true` via the
backend `ApplyDefaults`) render as "Disable" checkboxes unchecked,
which is the correct existing-behavior preserved state.
