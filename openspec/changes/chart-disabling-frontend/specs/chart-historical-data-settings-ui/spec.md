## ADDED Requirements

### Requirement: Global "Activity & data retention" subheading on the Advanced settings page

The `/settings/organization/advanced` page SHALL include a subheading titled
"Activity & data retention" that contains controls for toggling collection of
the historical-data datasets that drive dashboard charts.

#### Scenario: Subheading visible on Advanced page

- **WHEN** a global admin opens `/settings/organization/advanced`
- **THEN** the page SHALL display a section titled "Activity & data retention"
- **AND** the section SHALL contain a "Disable hosts active" checkbox
- **AND** the section SHALL contain a "Disable vulnerabilities" checkbox

#### Scenario: Available on both Free and Premium tiers

- **WHEN** the deployment is Free or Premium
- **THEN** the "Activity & data retention" section SHALL be visible to
  global admins
- **AND** the checkboxes SHALL behave identically across tiers

### Requirement: Global checkboxes invert the underlying API booleans

Each global checkbox in "Activity & data retention" SHALL represent the
*disabled* state of the corresponding `features.historical_data` sub-key.
Checking the box SHALL store `false`; unchecking SHALL store `true`.

#### Scenario: Loading reflects current API value

- **GIVEN** `appConfig.features.historical_data.uptime = true`
- **WHEN** the Advanced page loads
- **THEN** the "Disable hosts active" checkbox SHALL be unchecked

- **GIVEN** `appConfig.features.historical_data.vulnerabilities = false`
- **WHEN** the Advanced page loads
- **THEN** the "Disable vulnerabilities" checkbox SHALL be checked

#### Scenario: Saving inverts back to API shape

- **GIVEN** the "Disable hosts active" checkbox is checked
- **AND** the "Disable vulnerabilities" checkbox is unchecked
- **WHEN** the form is saved
- **THEN** the `PATCH /api/v1/fleet/config` payload SHALL include
  `features.historical_data.uptime = false`
- **AND** the payload SHALL include
  `features.historical_data.vulnerabilities = true`

### Requirement: Per-fleet "Activity & data retention" subheading on TeamSettings

The fleet settings page SHALL include an "Activity & data retention"
subheading containing the same two checkboxes, scoped to the current fleet.

#### Scenario: Subheading visible on TeamSettings

- **WHEN** a fleet admin opens a fleet's settings page
- **THEN** the page SHALL display an "Activity & data retention" section
- **AND** the section SHALL contain "Disable hosts active" and
  "Disable vulnerabilities" checkboxes scoped to the active fleet

#### Scenario: Premium-tier gating

- **WHEN** the deployment is Free
- **THEN** the per-fleet TeamSettings page SHALL not be reachable
- **AND** therefore the per-fleet "Activity & data retention" section
  SHALL not be visible
- **WHEN** the deployment is Premium
- **THEN** the section SHALL be visible to fleet admins

### Requirement: Per-fleet checkbox locks when global collection is disabled

The fleet-level checkbox for a given dataset SHALL render as disabled with a
"Disabled globally" tooltip whenever
`appConfig.features.historical_data.<key>` is `false`. The fleet's stored
value SHALL be preserved underneath the lock.

#### Scenario: Global off → fleet checkbox locked

- **GIVEN** `appConfig.features.historical_data.uptime = false`
- **WHEN** a fleet admin opens the fleet settings page
- **THEN** the fleet's "Disable hosts active" checkbox SHALL be disabled
  (non-interactive)
- **AND** hovering the checkbox SHALL show a tooltip "Disabled globally"

#### Scenario: Fleet's stored value preserved across global lockout

- **GIVEN** the fleet has `historical_data.uptime = false` saved
- **AND** the global setting is then set to `false`
- **WHEN** the fleet admin opens the fleet settings page
- **THEN** the fleet's "Disable hosts active" checkbox SHALL render as
  checked (matching the fleet's stored value), but disabled
- **WHEN** the global setting is later flipped back to `true`
- **AND** the fleet admin reopens the page
- **THEN** the checkbox SHALL render as checked AND interactive (the
  fleet's stored `false` survived the round trip)

### Requirement: Per-fleet checkbox writes the same API shape

Saving the per-fleet form SHALL submit
`features: { historical_data: { uptime: bool, vulnerabilities: bool } }` as
part of the `PATCH /api/v1/fleet/teams/{id}` payload, using the same
inversion as the global page.

#### Scenario: Per-fleet save payload

- **GIVEN** the fleet's "Disable vulnerabilities" checkbox is checked
- **WHEN** the form is saved
- **THEN** the `PATCH /api/v1/fleet/teams/{id}` payload SHALL include
  `features.historical_data.vulnerabilities = false`

### Requirement: Confirmation modal on Save when collection is being disabled

The page SHALL show a confirmation modal before issuing the PATCH whenever
any dataset moves from collecting (`true`) to disabled (`false`) compared to
the originally-loaded values. The modal SHALL list the affected dataset
labels prominently and SHALL expose a single destructive "Save and disable"
action that issues the PATCH on click. Re-enabling and no-op saves SHALL
NOT show the modal.

#### Scenario: Confirmation on enable → disable transition

- **GIVEN** the page loaded with `uptime = true`
- **AND** the user checks "Disable hosts active"
- **WHEN** the user clicks Save
- **THEN** a confirmation modal SHALL open listing "Hosts active" as the
  dataset being disabled
- **AND** no API request SHALL be issued until the user confirms

#### Scenario: No confirmation on no-op save

- **GIVEN** the page loaded with `uptime = false`
- **AND** the user does not change the checkbox
- **WHEN** the user clicks Save
- **THEN** the confirmation modal SHALL NOT open
- **AND** the PATCH SHALL be issued (or skipped) per the page's normal
  no-op behavior

#### Scenario: No confirmation on re-enable

- **GIVEN** the page loaded with `uptime = false`
- **AND** the user unchecks "Disable hosts active"
- **WHEN** the user clicks Save
- **THEN** the confirmation modal SHALL NOT open
- **AND** the PATCH SHALL be issued

#### Scenario: Confirmation lists all disabling datasets

- **GIVEN** the page loaded with both sub-keys `true`
- **AND** the user checks both "Disable hosts active" and
  "Disable vulnerabilities"
- **WHEN** the user clicks Save
- **THEN** the confirmation modal SHALL list both dataset labels

#### Scenario: Cancel discards the save attempt

- **WHEN** the confirmation modal is open
- **AND** the user clicks Cancel
- **THEN** the modal SHALL close
- **AND** no PATCH SHALL be issued
- **AND** the form state SHALL remain unchanged (the checkboxes stay
  checked, ready to save again or revert)

### Requirement: Confirmation modal lists the affected datasets prominently

The confirmation modal SHALL list each dataset being disabled by its
human-readable label so the user knows exactly which collection is
about to stop.

#### Scenario: Modal lists dataset labels

- **GIVEN** the user is disabling the `uptime` and `vulnerabilities`
  datasets
- **WHEN** the modal opens
- **THEN** the modal SHALL display "Hosts active"
- **AND** the modal SHALL display "Vulnerabilities"

#### Scenario: Single click confirms

- **GIVEN** the user is disabling one or more datasets
- **WHEN** the user clicks "Save and disable"
- **THEN** the PATCH SHALL be issued
- **AND** the modal SHALL close once the PATCH resolves successfully

### Requirement: Confirmation modal copy distinguishes scope

The modal SHALL render different body copy for global vs per-fleet scope.
Per-fleet copy SHALL include the fleet's name.

#### Scenario: Global-scope copy

- **WHEN** the modal opens from the Advanced page
- **THEN** the body SHALL describe the change as affecting the entire
  Fleet deployment

#### Scenario: Fleet-scope copy

- **WHEN** the modal opens from a fleet settings page for fleet "Engineering"
- **THEN** the body SHALL reference fleet "Engineering" by name
- **AND** the body SHALL describe the change as affecting only that fleet

### Requirement: GitOps mode locks out the new checkboxes

The Advanced page and the TeamSettings page SHALL render both new checkboxes
inside `<GitOpsModeTooltipWrapper>` so they are visually disabled and
tooltip-explained when GitOps mode is enabled.

#### Scenario: GitOps mode on Advanced page

- **GIVEN** GitOps mode is enabled
- **WHEN** a global admin opens `/settings/organization/advanced`
- **THEN** "Disable hosts active" SHALL be disabled with the GitOps tooltip
- **AND** "Disable vulnerabilities" SHALL be disabled with the GitOps tooltip

#### Scenario: GitOps mode on TeamSettings

- **GIVEN** GitOps mode is enabled
- **WHEN** a fleet admin opens a fleet settings page
- **THEN** the fleet's "Disable hosts active" SHALL be disabled with the
  GitOps tooltip
- **AND** the fleet's "Disable vulnerabilities" SHALL be disabled with the
  GitOps tooltip
- **AND** the GitOps lockout SHALL take precedence over the
  "Disabled globally" lockout in tooltip selection (GitOps wins)
