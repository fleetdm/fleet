## ADDED Requirements

### Requirement: OOBE-gated orbit notification
The server SHALL set `notifications.create_windows_managed_local_account` in the orbit config response only when all of the following hold: the host's Windows MDM enrollment is awaiting configuration (Pending or Active, covering Autopilot ESP and Entra-join-during-OOBE), the host's team (or No team) has `mdm.windows_settings.managed_local_account_settings.enabled`, the requesting fleetd advertises the `windows_managed_local_account` capability, and the license is premium.

#### Scenario: All gates hold
- **WHEN** a capable fleetd on a Windows host in the ESP polls orbit config for a premium team with the setting enabled
- **THEN** the response carries `notifications.create_windows_managed_local_account: true`

#### Scenario: Any gate fails
- **WHEN** the host is outside OOBE, or the setting is off, or fleetd lacks the capability, or the license is Free
- **THEN** the notification is absent

#### Scenario: Re-enrollment after wipe
- **WHEN** a previously provisioned host is wiped and re-enrolls through OOBE with the feature enabled
- **THEN** the notification is sent again even though an escrow row for the host already exists

### Requirement: Device-side account creation
On receiving the notification, fleetd on Windows SHALL ensure a local account named `_fleetadmin` exists with a device-generated random password (32 characters, all four character classes), membership in the local Administrators group (resolved by well-known SID `S-1-5-32-544`), a never-expiring password, and hidden from the sign-in screen via the `Winlogon\SpecialAccounts\UserList` registry value. The flow MUST be idempotent: if the account already exists, fleetd resets its password instead of failing.

#### Scenario: Fresh provisioning
- **WHEN** fleetd processes the notification on a host without the account
- **THEN** `_fleetadmin` exists as a hidden local Administrators member whose password never expires

#### Scenario: Account already exists
- **WHEN** fleetd processes the notification and `_fleetadmin` already exists (crash retry or wipe-less re-run)
- **THEN** fleetd resets the password and continues the flow without error

### Requirement: Password escrow
fleetd SHALL escrow the generated password to `POST /api/fleet/orbit/managed_local_account` over the node-key-authenticated orbit API. The server SHALL encrypt the password at rest in `host_managed_local_account_passwords` with `status` `verified` and a NULL `command_uuid`, and log a `created_managed_local_account` activity. The server SHALL reject requests from non-Windows hosts (eligibility verified via the host's Windows MDM enrollment, since `host.Platform` can be empty during early OOBE) and requests with an empty password. The server MUST NOT reject an escrow because the team setting or license state changed after the notification was sent: the account already exists on the device, and rejecting would orphan it with an unrecoverable password. In that case the server SHALL store the password and log a warning.

#### Scenario: Successful escrow
- **WHEN** fleetd posts the password for an eligible Windows host
- **THEN** the row is stored encrypted with `status = 'verified'` and NULL `command_uuid`, and the created activity is logged once

#### Scenario: Escrow replaces prior password
- **WHEN** a host escrows again (retry or re-enrollment)
- **THEN** the stored password is replaced and pending/rotation columns stay NULL

#### Scenario: Non-Windows escrow rejected
- **WHEN** the posting host has no Windows MDM enrollment
- **THEN** the server rejects the request and stores nothing

#### Scenario: Setting toggled off mid-flow does not orphan the account
- **WHEN** a Windows host escrows after its team's setting was turned off between the notification and the escrow POST
- **THEN** the server stores the password, logs a warning, and the account remains recoverable

#### Scenario: Platform not yet ingested
- **WHEN** a Windows host with an empty `host.Platform` (osquery has not reported yet) but a Windows MDM enrollment escrows during OOBE
- **THEN** the server accepts and stores the password

### Requirement: Crash-safe completion marker
fleetd SHALL persist a completion marker (containing no secret material) only after the server confirms the escrow, SHALL skip processing while the marker exists, and SHALL retry the full flow on the next config fetch when any step fails before the marker is written.

#### Scenario: Crash between creation and escrow
- **WHEN** fleetd dies after creating the account but before a confirmed escrow, and the next OOBE config fetch delivers the notification again
- **THEN** fleetd resets the password, escrows successfully, and writes the marker

#### Scenario: Marker prevents reprocessing
- **WHEN** the notification arrives and the marker exists
- **THEN** fleetd makes no account or registry changes and sends no escrow

### Requirement: Version skew degrades cleanly
Old fleetd against a new server SHALL never receive the notification (capability gate), and new fleetd against an old server (no notification field) SHALL take no action.

#### Scenario: Old fleetd
- **WHEN** a fleetd without the capability polls a server with the feature enabled
- **THEN** the notification is absent and no errors are logged on the host

#### Scenario: New fleetd, old server
- **WHEN** a capable fleetd polls a server that predates the feature
- **THEN** the receiver no-ops
