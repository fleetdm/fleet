# mdm-windows-setup-experience

## Requirements

### Requirement: Windows require_all_software_windows config

Fleet SHALL expose a per-team boolean config field `require_all_software_windows`, defaulting to `false`,
that controls whether the Windows setup experience blocks the end user when any setup-experience software
fails to install.

#### Scenario: Default value on a new team

- GIVEN a newly created team with Windows MDM enabled
- WHEN an admin reads the team's setup experience config
- THEN `require_all_software_windows` is `false`

#### Scenario: Update via REST API

- GIVEN a team exists with Windows MDM enabled
- WHEN an admin sends `PATCH /api/_version_/fleet/setup_experience` with
  `{"fleet_id": 1, "require_all_software_windows": true}`
- THEN the field is persisted
- AND subsequent reads return `true`

#### Scenario: Update via GitOps

- GIVEN a team YAML file sets `setup_experience.require_all_software_windows: true`
- WHEN `fleetctl apply` processes the file
- THEN the stored config reflects `true`
- AND exporting the team back to YAML round-trips the value

#### Scenario: Rejected when Windows MDM is disabled

- GIVEN a team where Windows MDM is NOT enabled
- WHEN an admin attempts to set `require_all_software_windows=true`
- THEN the request is rejected with a validation error

### Requirement: Windows hosts awaiting configuration state

Fleet SHALL track whether each Windows-enrolled host is awaiting setup-experience completion using a
tri-state column `awaiting_configuration` on `mdm_windows_enrollments` (0 = not awaiting, 1 = ESP not yet
issued, 2 = ESP issued and in progress).

#### Scenario: Autopilot OOBE enrollment sets awaiting

- GIVEN a Windows host enrolls via Autopilot
- AND the request includes an Azure JWT or WSTEP binary security token
- AND the request indicates `NotInOobe=false`
- WHEN Fleet stores the enrollment
- THEN `awaiting_configuration=1`
- AND `awaiting_configuration_at=NOW()`

#### Scenario: Programmatic fleetd enrollment does not set awaiting

- GIVEN a Windows host enrolls via fleetd using an orbit node key
- WHEN Fleet stores the enrollment
- THEN `awaiting_configuration=0`

#### Scenario: Post-OOBE Autopilot re-enrollment clears awaiting

- GIVEN a Windows host that previously completed setup re-enrolls via Autopilot
- AND the request indicates `NotInOobe=true`
- WHEN Fleet upserts the enrollment row
- THEN `awaiting_configuration=0`

### Requirement: Initial ESP command on first qualifying checkin

Fleet SHALL send an initial Windows ESP SyncML command covering profiles, SCEP certificates, and setup
software on the first Microsoft Management checkin for a host with `awaiting_configuration=1` and a known
`host_uuid`, and SHALL then transition the host to `awaiting_configuration=2`.

#### Scenario: Software defined, host recently orbit-enrolled

- GIVEN a host has `awaiting_configuration=1` and orbit-enrolled 30 minutes ago
- AND its team has setup-experience software items defined
- WHEN the host sends a Microsoft Management checkin
- THEN Fleet enqueues a SyncML command with DMClient ExpectedPolicies for each profile's top-most node
- AND EnrollmentStatusTracking entries for each setup software item
- AND SCEP entries for profiles containing SCEP certificates
- AND DMClient `TimeoutUntilSyncFailure=3h`
- AND the host row is updated to `awaiting_configuration=2`

#### Scenario: No software defined and orbit registered less than 10 minutes ago

- GIVEN a host has `awaiting_configuration=1` and orbit-enrolled 3 minutes ago
- AND its team has no setup-experience software items defined
- WHEN the host sends a Microsoft Management checkin
- THEN Fleet does nothing and leaves `awaiting_configuration=1`
- AND no SyncML command is enqueued

#### Scenario: No software defined and orbit registered more than 10 minutes ago

- GIVEN a host has `awaiting_configuration=1` and orbit-enrolled 15 minutes ago
- AND its team has no setup-experience software items defined
- WHEN the host sends a Microsoft Management checkin
- THEN Fleet proceeds to the release path
- AND `awaiting_configuration` is set to 0

### Requirement: Ongoing ESP status sync is inline, not stored

Fleet SHALL return current Windows ESP tracking status inline in the SyncML response body on every Microsoft
Management checkin while `awaiting_configuration=2`, and SHALL NOT enqueue these updates as stored commands.

#### Scenario: Three consecutive checkins in state 2

- GIVEN a host has `awaiting_configuration=2`
- WHEN the host makes three consecutive Microsoft Management checkins
- THEN each response contains the current EnrollmentStatusTracking entries reflecting
  `setup_experience_status_results`
- AND no new rows are written to the stored MDM commands table
- AND dropping any one response leaves the next response self-sufficient

### Requirement: Terminal-state release and cancellation

Fleet SHALL finalize the Windows setup experience when the outcome is decided: release the user when the
flow succeeded or when `require_all_software_windows=false` and all items are terminal; block the user with
a "Try again" message when `require_all_software_windows=true` and any item failed or the flow timed out.

#### Scenario: All items succeed

- GIVEN a host has `awaiting_configuration=2`
- AND all `setup_experience_status_results` rows are `success`
- WHEN the host makes a Microsoft Management checkin
- THEN Fleet enqueues a final ESP command releasing the user to the desktop
- AND `awaiting_configuration=0`

#### Scenario: require_all_software_windows=true with a failed install

- GIVEN a host has `awaiting_configuration=2` and its team has `require_all_software_windows=true`
- AND at least one setup-experience software item is in `failure` state
- WHEN the host makes a Microsoft Management checkin
- THEN Fleet cancels remaining pending items via `CancelPendingSetupExperienceSteps`
- AND enqueues a final ESP command with `BlockInStatusPage=true`
- AND `CustomErrorText="Critical software failed to install. Please try again. If this keeps happening,
  please contact your IT admin."`
- AND emits a `canceled_setup_experience` activity citing the first failed software item by `display_name`
  (falling back to `name`)
- AND `awaiting_configuration=0`

#### Scenario: require_all_software_windows=false with a failed install

- GIVEN a host has `awaiting_configuration=2` and its team has `require_all_software_windows=false`
- AND at least one setup-experience software item is in `failure` state
- AND all other items are terminal
- WHEN the host makes a Microsoft Management checkin
- THEN Fleet enqueues a final ESP command releasing the user to the desktop
- AND `CustomErrorText` surfaces the error to the user but does not block
- AND `awaiting_configuration=0`

### Requirement: 3 hour timeout

Fleet SHALL treat a host as timed out if `now - awaiting_configuration_at > 3 hours`, regardless of pending
items.

#### Scenario: Timeout with require_all_software_windows=true

- GIVEN a host has `awaiting_configuration=2` and `awaiting_configuration_at` 3 hours 5 minutes ago
- AND its team has `require_all_software_windows=true`
- WHEN the host makes a Microsoft Management checkin
- THEN Fleet cancels remaining pending items
- AND enqueues a blocking ESP command with `CustomErrorText`
- AND emits a `canceled_setup_experience` activity
- AND `awaiting_configuration=0`

#### Scenario: Timeout with require_all_software_windows=false

- GIVEN a host has `awaiting_configuration=2` and `awaiting_configuration_at` 3 hours 5 minutes ago
- AND its team has `require_all_software_windows=false`
- WHEN the host makes a Microsoft Management checkin
- THEN Fleet marks remaining items failed and enqueues a releasing ESP command
- AND `awaiting_configuration=0`

### Requirement: Windows checkbox on Setup experience > Install software

Fleet SHALL render a "Cancel setup if software fails" checkbox on the Windows tab of the Setup experience
install-software page that reads and writes `require_all_software_windows` through the same
`/api/_version_/fleet/setup_experience` endpoint used by macOS.

#### Scenario: Toggle the checkbox

- GIVEN an admin is on Setup experience > Install software, Windows tab
- AND `require_all_software_windows=false`
- WHEN the admin checks the box and clicks Save
- THEN the UI calls `updateRequireAllSoftwareWindows(true)`
- AND the admin sees a success state
- AND reloading the page shows the box checked

### Requirement: canceled_setup_experience activity covers Windows

Fleet SHALL emit the `canceled_setup_experience` activity type for Windows cancellations with the same
schema used for macOS, naming the first failed software item using `display_name` with a fallback to `name`.

#### Scenario: Windows cancellation produces an activity

- GIVEN a Windows host setup experience is canceled due to a failed install of "Microsoft Office"
- WHEN Fleet finalizes the cancellation
- THEN an activity of type `canceled_setup_experience` is recorded
- AND the payload references "Microsoft Office" via `display_name`
- AND the activity is visible in both the global activity feed and the host's activity feed

### Requirement: require_all_software field naming and backward compatibility

Fleet SHALL expose the macOS variant of the setting as `require_all_software_macos` and SHALL continue to
accept the legacy name `require_all_software` as a YAML/JSON alias for `require_all_software_macos`. Exported
configurations SHALL use the new name under `setup_experience`.

#### Scenario: Legacy YAML still applies

- GIVEN a GitOps YAML file sets `macos_setup.require_all_software: true` (legacy form)
- WHEN `fleetctl apply` processes the file
- THEN the stored value for `require_all_software_macos` is `true`

#### Scenario: Exported YAML uses the new name

- GIVEN a team has `require_all_software_macos=true`
- WHEN the team is exported to YAML via GitOps
- THEN the exported file contains `setup_experience.require_all_software_macos: true`
- AND not the legacy `macos_setup.require_all_software` form
