## ADDED Requirements

### Requirement: Password retrieval for Windows hosts
`GET /api/v1/fleet/hosts/{id}/managed_account_password` SHALL return the escrowed username and password for Windows hosts (in addition to macOS), restricted to global and team admins and maintainers, and SHALL log a `viewed_managed_local_account` activity. For Windows hosts the retrieval MUST NOT arm the auto-rotate timer, and the response MUST omit `auto_rotate_at` and `pending_rotation`.

#### Scenario: Admin retrieves a Windows password
- **WHEN** a team admin requests the managed account password for a Windows host with an escrowed row
- **THEN** the response contains username `_fleetadmin` and the plaintext password, the viewed activity is logged, and the row's `auto_rotate_at` remains NULL

#### Scenario: Observer denied
- **WHEN** an observer requests the managed account password
- **THEN** the API returns an authorization error

### Requirement: Rotation unavailable for Windows
`POST /api/v1/fleet/hosts/{id}/managed_account_password/rotate` SHALL reject Windows hosts with an error stating rotation is not available for Windows (rotation ships in #43489), and the rotation cron MUST never select Windows-shaped rows (NULL `command_uuid`, `account_uuid`, and `auto_rotate_at`).

#### Scenario: Rotate rejected on Windows
- **WHEN** an admin calls the rotate endpoint for a Windows host
- **THEN** the API returns a 4xx error naming the Windows rotation constraint

#### Scenario: Cron excludes Windows rows
- **WHEN** the auto-rotation cron queries for candidates
- **THEN** rows with NULL `command_uuid`, `account_uuid`, and `auto_rotate_at` are never returned

### Requirement: Host details surface
The host detail API response SHALL include `mdm.os_settings.managed_local_account` (status, password availability) for Windows hosts when Windows MDM is enabled, and the Host details UI SHALL show the "Show managed account" action for Windows hosts with a managed account row, opening the existing modal without rotation controls.

#### Scenario: Host detail response populated
- **WHEN** a Windows host with an escrowed row is fetched via the host detail API
- **THEN** the response includes `mdm.os_settings.managed_local_account` with `status: "verified"` and `password_available: true`

#### Scenario: Action visible and rotation-free
- **WHEN** an admin opens Host details > Actions for such a Windows host
- **THEN** "Show managed account" is available and the modal shows the credentials with no rotate button and no auto-rotate banner

#### Scenario: Pending escrow disables the action
- **WHEN** the host's managed account row is absent or not yet verified
- **THEN** the action is hidden or disabled with the pending tooltip
