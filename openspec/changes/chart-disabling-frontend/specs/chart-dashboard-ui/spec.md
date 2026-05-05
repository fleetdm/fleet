## ADDED Requirements

### Requirement: Chart card renders an empty state when historical-data collection is disabled

The chart card SHALL render a "Data collection disabled" empty state in
place of the visualization whenever the active dataset's collection is
disabled. The empty-state condition is the AND of the global and (when a
fleet is selected) per-fleet `features.historical_data.<config-key>` value.

The chart card SHALL continue to render the dataset selector, time range
selector, and filter gear so the user can switch to a still-enabled
dataset without leaving the page.

#### Scenario: Empty state on globally disabled dataset

- **GIVEN** `appConfig.features.historical_data.uptime = false`
- **AND** the dashboard is viewed in "All teams" scope
- **WHEN** the chart card has the uptime dataset selected
- **THEN** the visualization area SHALL render the
  "Data collection disabled" empty state
- **AND** the dataset selector, time range selector, and filter gear
  SHALL remain visible and functional

#### Scenario: Empty state on per-fleet disabled dataset

- **GIVEN** `appConfig.features.historical_data.vulnerabilities = true`
- **AND** the active fleet has `historical_data.vulnerabilities = false`
- **AND** the dashboard is viewed scoped to that fleet
- **WHEN** the chart card has a vulnerabilities-driven dataset selected
- **THEN** the visualization area SHALL render the
  "Data collection disabled" empty state

#### Scenario: Chart renders when both global and fleet are enabled

- **GIVEN** the global and fleet sub-keys for the active dataset are
  both `true` (or the fleet sub-key is absent for All-teams scope)
- **WHEN** the chart card loads
- **THEN** the visualization SHALL render normally
- **AND** the empty state SHALL NOT appear

#### Scenario: Switching to a still-enabled dataset clears the empty state

- **GIVEN** the empty state is rendered for dataset A
- **AND** dataset B's collection is enabled
- **WHEN** the user selects dataset B from the dataset dropdown
- **THEN** the empty state SHALL be replaced by dataset B's visualization

#### Scenario: Datasets without a config-key mapping render normally

- **GIVEN** a dataset whose internal name has no entry in
  `DATASET_CONFIG_KEY` (e.g. a future dataset added before its config-key
  mapping is wired)
- **WHEN** the chart card has that dataset selected
- **THEN** the visualization SHALL render normally
- **AND** the empty state SHALL NOT appear (no config-key means no
  toggle exists for it yet, so collection is implicitly enabled)

### Requirement: Empty-state copy points users to the global Advanced settings page

The "Data collection disabled" empty state SHALL include the dataset's
human-readable label and a link to `/settings/organization/advanced`. It
SHALL NOT attempt to deep-link into a specific fleet's settings page.

#### Scenario: Empty-state content

- **WHEN** the empty state is rendered for the "Hosts active" dataset
- **THEN** the empty state SHALL display "Hosts active" as the dataset name
- **AND** the empty state SHALL include a link to the
  `/settings/organization/advanced` page
- **AND** the link text SHALL describe re-enabling collection (e.g.
  "Manage data collection in Advanced settings")
