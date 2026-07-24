# Spec: Windows automatic enrollment default fleet configuration

## ADDED Requirements

### Requirement: Global default fleet setting for Windows automatic enrollment
Fleet SHALL store a single, global, optional default fleet for Windows automatic enrollment. The setting SHALL reference a fleet by id internally (nullable foreign key to `teams`) and SHALL be exposed by fleet name in the config API. The setting SHALL be available only with a Premium license.

#### Scenario: Read via config API
- **WHEN** a global admin calls `GET /api/v1/fleet/config` on a Premium server with the default set to fleet "Workstations"
- **THEN** the response includes `mdm.windows_automatic_enrollment.default_fleet: "Workstations"`

#### Scenario: Read when unset
- **WHEN** a global admin calls `GET /api/v1/fleet/config` and no default fleet is configured
- **THEN** the response includes `mdm.windows_automatic_enrollment.default_fleet: ""`

#### Scenario: Set via config API
- **WHEN** a global admin calls `PATCH /api/v1/fleet/config` with `mdm.windows_automatic_enrollment.default_fleet` set to the name of an existing fleet
- **THEN** the setting is persisted, and subsequent reads return that fleet name

#### Scenario: Clear via config API
- **WHEN** a global admin patches `mdm.windows_automatic_enrollment.default_fleet` with an empty string
- **THEN** the default fleet is cleared (hosts enroll into "No team")

#### Scenario: Omitted key is a no-op
- **WHEN** a config PATCH omits `mdm.windows_automatic_enrollment`
- **THEN** the existing default fleet value is unchanged

#### Scenario: Unknown fleet name rejected
- **WHEN** a config PATCH references a fleet name that does not exist
- **THEN** the request fails with a 422 invalid argument error and the setting is unchanged

#### Scenario: Free license rejected
- **WHEN** `mdm.windows_automatic_enrollment.default_fleet` is set to a non-empty value on a Free-tier server
- **THEN** the request fails with a license error and the response never includes the setting on Free tier

#### Scenario: Non-admin rejected
- **WHEN** a user without global admin (or GitOps) privileges attempts to modify the setting
- **THEN** the request fails authorization

### Requirement: Activity logged when the default fleet changes
Fleet SHALL create an `edited_windows_automatic_enrollment_default_fleet` activity whenever the default fleet value changes, with `fleet_id` and `fleet_name` fields reflecting the new value (both null when cleared). The activity SHALL NOT be created when a config write leaves the value unchanged.

#### Scenario: Changed value logs activity
- **WHEN** the default fleet changes from unset to "Workstations" (via UI, API, or GitOps)
- **THEN** an `edited_windows_automatic_enrollment_default_fleet` activity is created with the fleet's id and name, rendered in the feed as "edited the default fleet for Windows automatic enrollment hosts to Workstations."

#### Scenario: Cleared value logs activity with nulls
- **WHEN** the default fleet is cleared
- **THEN** the activity is created with `fleet_id: null` and `fleet_name: null`

#### Scenario: Unchanged value logs nothing
- **WHEN** a config PATCH writes the same default fleet that is already set
- **THEN** no `edited_windows_automatic_enrollment_default_fleet` activity is created

### Requirement: GitOps applies and exports the setting
`fleetctl gitops` SHALL apply `org_settings.mdm.windows_automatic_enrollment.default_fleet` (fleet name, object form), including when the referenced fleet is created in the same run, and `fleetctl generate-gitops` SHALL export the current value. Omitting the key SHALL leave the value unchanged.

#### Scenario: Apply with fleet created in same run
- **WHEN** a GitOps run declares a new fleet "Windows Workstations" and sets it as `default_fleet`
- **THEN** the run succeeds and the default fleet is set after the fleet is created

#### Scenario: Apply with unknown fleet fails
- **WHEN** a GitOps run sets `default_fleet` to a name not present in Fleet or in the run's declared fleets
- **THEN** the run fails validation before applying

#### Scenario: Dry run
- **WHEN** the same configuration is applied with `--dry-run`
- **THEN** validation runs (including same-run fleet assumptions) and nothing is persisted

#### Scenario: Export
- **WHEN** `fleetctl generate-gitops` runs against a server with a default fleet configured
- **THEN** the generated org settings YAML includes `mdm.windows_automatic_enrollment.default_fleet` with the fleet name

### Requirement: Deleting the default fleet clears the setting
Deleting the fleet currently referenced as the Windows automatic enrollment default SHALL clear the setting rather than block the deletion, matching ABM default team behavior.

#### Scenario: Default fleet deleted
- **WHEN** a global admin deletes the fleet that is set as the default
- **THEN** the deletion succeeds, the setting reads as unassigned, and subsequent automatic enrollments go to "No team"
