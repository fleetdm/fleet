## ADDED Requirements

### Requirement: End user can submit a BitLocker PIN for their host
The server SHALL accept a BitLocker PIN from the My device page at `POST /api/_version_/fleet/device/{token}/disk_encryption_pin` with body `{"pin": "<string>"}`, authenticated by the device token and rate limited by the existing device error limiter. The server SHALL validate that the PIN is 6 to 20 ASCII digits and SHALL store it encrypted with the server private key in a per-host request row with status `pending`, replacing any earlier request for that host.

#### Scenario: Valid submission on an eligible host
- **WHEN** a Windows host that is MDM-enrolled with Fleet, not a server, on a fleet with disk encryption and `require_bitlocker_pin` enabled, with `tpm_pin_set` false and a PIN-capable fleetd, submits a 6 to 20 digit PIN
- **THEN** the server stores the encrypted PIN with status `pending` and responds `204 No Content`

#### Scenario: Resubmission replaces the earlier request
- **WHEN** a host that already has a `pending` or `failed` request submits a new PIN
- **THEN** the earlier row is replaced and the new PIN becomes the single `pending` request for that host

#### Scenario: PIN fails validation
- **WHEN** the submitted PIN is shorter than 6 characters, longer than 20 characters, or contains a non-digit
- **THEN** the server responds `422` with a validation error naming `pin`, and stores nothing

#### Scenario: Host is not eligible
- **WHEN** the host is not Windows, is a Windows Server, is not MDM-enrolled with Fleet, its fleet does not require a BitLocker PIN, its PIN is already set, or its fleetd is not PIN-capable
- **THEN** the server responds `400` and stores nothing

### Requirement: The PIN is confidential on the server
The server SHALL never write the PIN to logs, error payloads or activities, SHALL never return it on any user-authenticated endpoint, and SHALL clear the ciphertext the moment orbit fetches it. A `pending` request older than 5 minutes SHALL be treated as expired: not advertised to orbit, not returned to orbit, and reported to the page as `failed` with a timeout reason.

#### Scenario: Only orbit can read the PIN
- **WHEN** any user-authenticated or device-token endpoint that exposes the request is called
- **THEN** the response contains at most the request `status` and sanitized `error`, never the PIN

#### Scenario: Request expires undelivered
- **WHEN** a `pending` request is older than 5 minutes and orbit has not fetched it
- **THEN** the orbit config no longer advertises it, a fetch returns not found, and the device host response reports `pin_request.status = "failed"` with a timeout error

### Requirement: Orbit is told a PIN request is pending
The orbit config response SHALL include `notifications.bitlocker_pin_request_pending = true` for a host only while a non-expired `pending` request exists and the eligibility conditions from the submission requirement still hold. It SHALL be omitted otherwise.

#### Scenario: Pending request on an eligible host
- **WHEN** orbit polls its config for a host with a fresh `pending` request and the fleet still requires a PIN
- **THEN** `bitlocker_pin_request_pending` is `true`

#### Scenario: Requirement turned off after submission
- **WHEN** the fleet's `require_bitlocker_pin` is disabled, or the host moves to a fleet that does not require one, while a request is `pending`
- **THEN** `bitlocker_pin_request_pending` is omitted and the request is discarded

### Requirement: Orbit fetches the PIN once
`POST /api/fleet/orbit/disk_encryption_pin/request`, authenticated by the orbit node key and restricted to Windows MDM-enrolled hosts, SHALL return `{"pin": "<digits>"}` for a `pending`, non-expired request, set the request status to `delivered`, and clear the stored ciphertext so no second fetch can return it.

#### Scenario: Successful fetch
- **WHEN** orbit calls the request endpoint while a fresh `pending` request exists
- **THEN** the response contains the plaintext PIN exactly once and the row becomes `delivered` with no ciphertext

#### Scenario: Nothing to fetch
- **WHEN** orbit calls the request endpoint with no `pending` request, an expired request, or a `delivered` request
- **THEN** the server responds `404` and returns no PIN

### Requirement: Orbit reports the outcome
`POST /api/fleet/orbit/disk_encryption_pin` SHALL accept `{"outcome": "set" | "failed", "client_error": "<string>"}` from a Windows MDM-enrolled host. `client_error` MUST be non-empty when `outcome` is `failed` and SHALL be trimmed and truncated to 255 characters. Unknown outcomes SHALL be rejected with `400`.

#### Scenario: PIN set
- **WHEN** orbit reports `outcome = "set"`
- **THEN** the server sets `host_disks.tpm_pin_set = true`, marks the request row `set` so the waiting page can observe the result, records a `created_disk_encryption_pin` activity with `host_id` and `host_display_name` and no user actor, and requests a host refetch

#### Scenario: Terminal request rows are cleaned up
- **WHEN** a request row has been in `set` or `failed` for longer than 24 hours, or the host submits a new PIN
- **THEN** the row is deleted or replaced; a terminal row holds no ciphertext, because delivery cleared it

#### Scenario: PIN failed
- **WHEN** orbit reports `outcome = "failed"` with a reason
- **THEN** the server stores `status = "failed"` and the sanitized reason on the request row, leaves `tpm_pin_set` unchanged, and records no activity

#### Scenario: Failed report without a reason
- **WHEN** orbit reports `outcome = "failed"` with an empty `client_error`
- **THEN** the server responds `422` naming `client_error`

### Requirement: The device host response exposes request state and agent capability
`GET /api/_version_/fleet/device/{token}` SHALL include, under `mdm.os_settings.disk_encryption`, `fleetd_can_set_pin` (boolean, true when the host's most recent Windows MDM enrollment has the `windows_bitlocker_pin` capability persisted) and, while a request row exists, `pin_request` with `status` (`pending`, `delivered`, `set`, `failed`) and `error` (sanitized, empty unless `failed`). Every status in that enum SHALL be observable by the page: a successful apply leaves the row in `set` rather than deleting it, so the waiting modal has a positive signal and does not have to infer success from the absence of a row.

#### Scenario: Capable agent, request in flight
- **WHEN** the page polls the device host endpoint after submitting a PIN to a host whose fleetd advertised the capability
- **THEN** the response shows `fleetd_can_set_pin = true` and `pin_request.status` reflecting the current row

#### Scenario: Old agent
- **WHEN** the host's fleetd has never advertised `windows_bitlocker_pin`
- **THEN** `fleetd_can_set_pin = false` and `pin_request` is omitted

### Requirement: The fleetd PIN capability is persisted per Windows MDM enrollment
When the orbit config request carries `windows_bitlocker_pin` in `X-Fleet-Capabilities`, the server SHALL persist `fleetd_bitlocker_pin_capable = true` on the host's most recent `mdm_windows_enrollments` row, and SHALL persist `false` when a later request omits it. The column SHALL default to false so existing enrollments are treated as not capable until the agent reports.

#### Scenario: Agent upgrades
- **WHEN** a host previously on fleetd without the capability polls config with `windows_bitlocker_pin` in its capabilities header
- **THEN** the enrollment row flips to capable on that poll

### Requirement: Fleet Desktop is told the host needs a PIN
`GET /api/_version_/fleet/device/{token}/desktop` SHALL include `notifications.needs_bitlocker_pin = true` only when the host's disk encryption `action_required` is `create_pin` and `fleetd_can_set_pin` is true. It SHALL be omitted otherwise.

#### Scenario: PIN needed and agent capable
- **WHEN** Fleet Desktop fetches its summary for a Windows host in Action required for a missing PIN with a capable fleetd
- **THEN** `needs_bitlocker_pin` is `true`

#### Scenario: PIN set or agent not capable
- **WHEN** the host's PIN is set, the fleet no longer requires one, or fleetd is not capable
- **THEN** `needs_bitlocker_pin` is omitted

### Requirement: New activity type `created_disk_encryption_pin`
The server SHALL define activity `created_disk_encryption_pin` with details `host_id` and `host_display_name`, created with no user actor, documented in the audit log reference, and listed in the activity type filter as "Created a disk encryption PIN".

#### Scenario: Activity recorded on success
- **WHEN** orbit reports a successful PIN set for host "MU-TH-UR"
- **THEN** the global activity feed and the host's activity tab show "End user created a disk encryption PIN for MU-TH-UR."
