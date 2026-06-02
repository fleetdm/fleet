## ADDED Requirements

### Requirement: Windows/Linux setup-experience software is gated by its associated policy

Each Windows/Linux setup-experience software installer SHALL be "policy-gated" when a team policy references it. Concretely, an
item that is a software installer (`software_installers.install_during_setup = 1`) is gated when a team policy for the host's team has
`software_installer_id` equal to that installer. The association SHALL be discovered server-side with no REST API, YAML,
fleetctl, fleetd, activity, or permission changes, and SHALL be recorded on the setup-experience status row at enqueue time as an
internal (`json:"-"`) `policy_id`. Items with no associated policy SHALL install unconditionally exactly as before. VPP items and
macOS/iOS/iPadOS items SHALL never be policy-gated.

#### Scenario: Associated policy recorded at enqueue

- **WHEN** a Windows or Linux host calls `/setup_experience/init` and an installer flagged `install_during_setup` is targeted by
  a team policy's install-software automation
- **THEN** the inserted `setup_experience_status_results` row SHALL carry the matching `policy_id`

#### Scenario: No association leaves behavior unchanged

- **WHEN** the targeted installer has no team policy referencing it
- **THEN** the row's `policy_id` SHALL be NULL and the item SHALL be installed during setup as before

#### Scenario: Apple platforms are never gated

- **WHEN** the host platform is macOS, iOS, or iPadOS, or the item is a VPP app
- **THEN** the item SHALL have no associated policy and SHALL install unconditionally, regardless of any policy referencing the
  installer

### Requirement: Associated policies are evaluated during setup; all others remain skipped

For a host in setup experience, the server SHALL distribute the policy queries for that host's associated setup-experience
policies (and only those), reversing — for those policies only — the existing behavior that skips all policy queries during setup
(`server/service/osquery.go:922-929`). Unrelated team policies SHALL remain unevaluated during setup so their automations do not
fire mid-setup. The result SHALL reflect this enrollment, not a stale prior result.

#### Scenario: Only associated policies run during setup

- **WHEN** a host in setup experience has one associated setup-experience policy and several unrelated team policies
- **THEN** the server SHALL send only the associated policy's query and SHALL NOT send the unrelated policy queries

#### Scenario: Fresh result on a newly enrolled host

- **WHEN** a freshly enrolled host with no prior `policy_membership` for the associated policy answers the associated policy
  query
- **THEN** the setup-experience flow SHALL act on that result, not on any stale prior value

### Requirement: Install is skipped when the associated policy passes

When a policy-gated item's associated policy returns `passes = true`, the server SHALL move the item to the terminal `success`
state without attempting any install, and the on-host setup experience SHALL progress to completion. No new status enum value
SHALL be introduced; the skipped outcome reuses `success`.

#### Scenario: Up-to-date app is skipped

- **WHEN** the associated policy passes (app present and up-to-date)
- **THEN** no `InsertSoftwareInstallRequest` SHALL be issued for the item
- **AND** the item SHALL reach `success`
- **AND** the host SHALL not be shown the item as installing or failed

### Requirement: Policy failure runs the policy's install-software automation, tracked by setup experience

When a policy-gated item's associated policy returns `passes = false`, the policy's install-software automation SHALL run and the
item SHALL track it. The automation runs as it would outside setup, the setup-experience item SHALL be linked to that
automation's `host_software_installs` execution and SHALL mirror its terminal state into `success`/`failure`, exactly one install
SHALL occur (no double-install), and every item SHALL reach a terminal state (no hang). Run-script automations are out of scope.

#### Scenario: Outdated/missing app is installed by the policy automation

- **WHEN** the associated install-software policy fails because the app is missing or outdated
- **THEN** the policy's install-software automation SHALL install the correct installer
- **AND** the setup-experience item SHALL track that install and reach `success` on completion or `failure` on terminal failure
- **AND** the app SHALL be installed exactly once

#### Scenario: Mixed batch reaches terminal states

- **WHEN** a host has a passing-policy item, a failing-policy item, and a no-policy item
- **THEN** the passing-policy item SHALL be skipped, the failing-policy and no-policy items SHALL install, and every item SHALL
  reach a terminal state

### Requirement: Policy scope that excludes the host does not hang setup

When the associated policy's scope excludes the host, the server SHALL NOT leave the item waiting indefinitely. Because the policy
query is not delivered and no result arrives, the server SHALL treat the item as un-gated and install it during setup.

#### Scenario: Out-of-scope policy falls back to install

- **WHEN** the associated policy's scope excludes the enrolling host
- **THEN** the item SHALL be installed during setup and SHALL reach a terminal state

### Requirement: Setup experience copy reflects policy gating on Windows and Linux

The Windows and Linux Install software tabs SHALL show updated copy describing policy-gated installs. On **Controls > Setup
experience > Install software**, the description below the software table on those tabs SHALL match the approved wireframe copy,
while the macOS, iOS, and iPadOS copy SHALL be unchanged.

#### Scenario: Windows and Linux copy updated

- **WHEN** an admin views the Windows or Linux tab
- **THEN** the description below the table SHALL show the wireframe copy describing policy-gated installs

#### Scenario: macOS copy unchanged

- **WHEN** an admin views the macOS tab
- **THEN** the description SHALL be unchanged from current behavior
