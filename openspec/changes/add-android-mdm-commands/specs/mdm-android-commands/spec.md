## ADDED Requirements

### Requirement: Lock host endpoint supports Android

The `POST /api/v1/fleet/hosts/:id/lock` endpoint SHALL accept Android hosts. When the target host's `FleetPlatform()` is
`android` and Android MDM is enabled and configured for the host's team, the endpoint SHALL issue an Android Management API
(AMAPI) `LOCK` command via `EnterprisesDevicesService.IssueCommand`, persist the resulting operation name and a
Fleet-generated `command_uuid` in `mdm_android_commands`, set `host_mdm_actions.lock_ref = command_uuid` and
`host_mdm_actions.fleet_platform = 'android'`, emit `ActivityTypeLockedHost` with `ViewPIN: false`, and return a
`CommandEnqueueResult` payload with `request_type = "LOCK"` and `platform = "android"`. The endpoint SHALL remain
Premium-only. The endpoint SHALL apply the existing pending-state guards (`IsPendingLock`, `IsPendingUnlock`, `IsPendingWipe`,
`IsWiped`, `IsLocked`) to Android hosts unchanged. Validation SHALL reject Android hosts when Android MDM is not enabled and
configured with a 400 BadRequestError naming the AndroidMDMNotConfigured condition.

#### Scenario: Lock a BYO Android host with MDM enabled

- **WHEN** an admin POSTs `/hosts/123/lock` for a BYO Android host (platform=`android`, `companyOwned=false`, MDM enrolled)
- **THEN** the server SHALL call `EnterprisesDevicesService.IssueCommand` with `Type: "LOCK"` against the host's AMAPI device
  resource
- **AND** SHALL insert a row in `mdm_android_commands` with `status='pending'`, `command_type='LOCK'`, the AMAPI operation
  name, and `request_payload` containing the rendered Command
- **AND** SHALL update `host_mdm_actions` so `lock_ref` equals the new `command_uuid` and `fleet_platform='android'`
- **AND** SHALL log `ActivityTypeLockedHost` with `ViewPIN=false`
- **AND** SHALL return HTTP 200 with `{command_uuid, request_type: "LOCK", platform: "android"}`

#### Scenario: Lock a COBO Android host with MDM enabled

- **WHEN** an admin POSTs `/hosts/123/lock` for a COBO Android host (`companyOwned=true`)
- **THEN** the server SHALL behave identically to the BYO case above. (AMAPI's LOCK on COBO locks the entire device; on BYO it
  locks the work profile. The server does not distinguish the two — the device behavior is dictated by AMAPI based on its
  knowledge of the device's ownership.)

#### Scenario: Lock fails when Android MDM is not configured

- **WHEN** an admin POSTs `/hosts/123/lock` for an Android host on a team where Android MDM is not enabled or configured
- **THEN** the server SHALL return HTTP 400 `BadRequestError` with `Message: fleet.AndroidMDMNotConfiguredMessage`
- **AND** SHALL NOT call AMAPI
- **AND** SHALL NOT modify `host_mdm_actions` or `mdm_android_commands`

#### Scenario: Lock blocked when prior Lock pending

- **WHEN** an admin POSTs `/hosts/123/lock` for an Android host that already has `host_mdm_actions.lock_ref` set and the
  referenced `mdm_android_commands` row has `status='pending'`
- **THEN** the server SHALL return HTTP 422 with a message identifying the pending-lock condition (same string as the existing
  Apple/Windows path)
- **AND** SHALL NOT issue a second LOCK command

### Requirement: Wipe host endpoint supports company-owned Android only

The `POST /api/v1/fleet/hosts/:id/wipe` endpoint SHALL accept Android hosts only when `companyOwned=true`. For BYO Android hosts,
the endpoint SHALL return a 400 BadRequestError directing the admin to use Unenroll. For COBO hosts, the endpoint SHALL issue
an AMAPI `WIPE` command, persist tracking state analogous to Lock (with `wipe_ref` instead of `lock_ref`, `command_type='WIPE'`),
emit `ActivityTypeWipedHost`, and return a `CommandEnqueueResult` with `request_type="WIPE"` and `platform="android"`.
Premium-only. Pending-state guards apply unchanged.

#### Scenario: Wipe a COBO Android host

- **WHEN** an admin POSTs `/hosts/123/wipe` for a COBO Android host with MDM enabled
- **THEN** the server SHALL call `IssueCommand` with `Type: "WIPE"`
- **AND** SHALL insert a row in `mdm_android_commands` (`command_type='WIPE'`, `status='pending'`)
- **AND** SHALL set `host_mdm_actions.wipe_ref = command_uuid`, `fleet_platform='android'`
- **AND** SHALL log `ActivityTypeWipedHost`
- **AND** SHALL return HTTP 200 with `{command_uuid, request_type: "WIPE", platform: "android"}`

#### Scenario: Wipe is rejected for BYO Android

- **WHEN** an admin POSTs `/hosts/123/wipe` for a BYO Android host (`companyOwned=false`)
- **THEN** the server SHALL return HTTP 400 BadRequestError with a message indicating that Wipe is not supported for
  personally-owned Android hosts and directing the admin to Unenroll instead
- **AND** SHALL NOT call AMAPI
- **AND** SHALL NOT modify `host_mdm_actions` or `mdm_android_commands`

### Requirement: Clear passcode endpoint supports Android (BYO and COBO)

The `POST /api/v1/fleet/hosts/:id/clear_passcode` endpoint SHALL accept Android hosts (both BYO and COBO). When the target
host is Android, the endpoint SHALL issue an AMAPI `RESET_PASSWORD` command with `newPassword=""` (empty), persist a row in
`mdm_android_commands` (`command_type='RESET_PASSWORD'`, `status='pending'`), emit `ActivityTypeClearedPasscode`, and return
`{command_uuid, request_type: "RESET_PASSWORD", platform: "android"}`. The endpoint SHALL NOT generate or return a password.
The endpoint SHALL NOT modify `host_mdm_actions` (Clear passcode is fire-and-flash; no pending badge state in the UI). The
endpoint SHALL remain Premium-only.

#### Scenario: Clear passcode on BYO Android clears work-profile passcode

- **WHEN** an admin POSTs `/hosts/123/clear_passcode` for a BYO Android host with MDM enabled
- **THEN** the server SHALL call `IssueCommand` with `Type: "RESET_PASSWORD"` and `NewPassword: ""`
- **AND** SHALL insert a row in `mdm_android_commands` with `command_type='RESET_PASSWORD'`
- **AND** SHALL NOT modify `host_mdm_actions`
- **AND** SHALL log `ActivityTypeClearedPasscode`
- **AND** SHALL return HTTP 200 with `{command_uuid, request_type: "RESET_PASSWORD", platform: "android"}` (no password field)

#### Scenario: Clear passcode on COBO Android clears device passcode

- **WHEN** an admin POSTs `/hosts/123/clear_passcode` for a COBO Android host with MDM enabled
- **THEN** the server SHALL behave identically to the BYO case above. (AMAPI's RESET_PASSWORD scopes to the device passcode on
  COBO and the work-profile passcode on BYO; the server does not differentiate.)

### Requirement: BYO Android Unenroll sends WIPE command but emits unenroll activity

The `POST /api/v1/fleet/hosts/:id/mdm` (unenroll) endpoint SHALL, for BYO Android hosts (`companyOwned=false`), issue an AMAPI
`WIPE` command instead of `EnterprisesDevicesDelete`. The WIPE command on a BYO device removes the work profile (per AMAPI
documentation) and survives the host being offline more than 30 days (unlike `device.delete` which expires server-side). For
COBO Android hosts, the existing `EnterprisesDevicesDelete` behavior SHALL remain unchanged. The user-facing operation
remains "Unenroll" for both ownership modes — `ActivityTypeMDMUnenrolled` SHALL be emitted unchanged (existing behavior).
`ActivityTypeWipedHost` SHALL NOT be emitted for BYO Unenroll (product decision 2026-05-20). The endpoint SHALL NOT write
`host_mdm_actions.wipe_ref` for BYO Unenroll (no user-facing wipe state — the WIPE is an internal mechanism only).

#### Scenario: BYO Android Unenroll issues WIPE command, emits MDMUnenrolled activity

- **WHEN** an admin POSTs `/hosts/123/mdm` (unenroll) for a BYO Android host
- **THEN** the server SHALL call `IssueCommand` with `Type: "WIPE"` against the host's AMAPI device resource
- **AND** SHALL NOT call `EnterprisesDevicesDelete`
- **AND** SHALL insert a row in `mdm_android_commands` (`command_type='WIPE'`, `status='pending'`)
- **AND** SHALL NOT modify `host_mdm_actions` (no `wipe_ref` write, no `fleet_platform` write)
- **AND** SHALL log `ActivityTypeMDMUnenrolled` (existing behavior, unchanged)
- **AND** SHALL NOT log `ActivityTypeWipedHost`

#### Scenario: COBO Android Unenroll continues to call device.delete

- **WHEN** an admin POSTs `/hosts/123/mdm` (unenroll) for a COBO Android host
- **THEN** the server SHALL call `EnterprisesDevicesDelete` (unchanged behavior)
- **AND** SHALL emit `ActivityTypeMDMUnenrolled` (unchanged behavior)
- **AND** SHALL NOT insert a row in `mdm_android_commands`

### Requirement: No host-header badges for Android

The host details page header SHALL NOT display "Wipe pending", "Wiped", "Lock pending", "Locked", or "Clear passcode
pending" badges for Android hosts, regardless of the underlying `host.mdm.device_status` value or `host_mdm_actions` state.
The badge components SHALL continue to render for non-Android platforms (iOS, iPadOS, macOS) unchanged. The server-side
pending-state guards (e.g. `lockWipe.IsPendingLock()`) SHALL continue to function for Android (they block double-issue of
the same command) — only the UI rendering is gated.

#### Scenario: Android host with wipe_pending status does not display badge

- **GIVEN** an Android host with `host.mdm.device_status = 'wipe_pending'`
- **WHEN** the host details page is rendered
- **THEN** the page SHALL NOT display a "Wipe pending" badge next to the host name

#### Scenario: iOS host with wipe_pending status still displays badge (regression check)

- **GIVEN** an iOS host with `host.mdm.device_status = 'wipe_pending'`
- **WHEN** the host details page is rendered
- **THEN** the page SHALL display a "Wipe pending" badge next to the host name (existing behavior preserved)

### Requirement: Persist Android MDM commands in `mdm_android_commands`

The system SHALL persist every Android Management API command issued via Fleet in a new `mdm_android_commands` table. The
table SHALL be keyed by a Fleet-generated `command_uuid` (VARCHAR(36)) and SHALL hold the AMAPI operation name
(VARCHAR(255)), command type (`LOCK | RESET_PASSWORD | WIPE`), status (`pending | acknowledged | error`), optional
error_code/error_message, the request payload as JSON, and `created_at`/`updated_at` timestamps with microsecond precision
(`DATETIME(6) NOT NULL DEFAULT NOW(6)` / `DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6)`). The table SHALL be indexed
on `host_uuid` (for host-details lookups and joins) and `operation_name` (for Pub/Sub COMMAND notification lookups).

#### Scenario: A new LOCK command writes a pending row

- **WHEN** the server issues a LOCK command via `LockAndroidHost`
- **THEN** a row SHALL be inserted in `mdm_android_commands` with `command_type='LOCK'`, `status='pending'`,
  `operation_name` populated, `request_payload` set to the rendered AMAPI Command, `created_at` and `updated_at` set to
  the current microsecond timestamp

#### Scenario: Acknowledged command updates the row

- **WHEN** the Pub/Sub COMMAND handler receives a successful acknowledgement (no `errorCode`) for an operation matching a
  row in `mdm_android_commands`
- **THEN** the row's `status` SHALL be updated to `acknowledged`
- **AND** `updated_at` SHALL advance via the `ON UPDATE NOW(6)` clause

#### Scenario: Failed command captures error details

- **WHEN** the Pub/Sub COMMAND handler receives a Command resource with a non-empty `errorCode`
- **THEN** the row's `status` SHALL be set to `error`, `error_code` set to the AMAPI value (e.g. `"UNSUPPORTED"`,
  `"INVALID_VALUE"`), and `error_message` set to a derived human-readable description
- **AND** the corresponding `host_mdm_actions.lock_ref` or `wipe_ref` SHALL remain set so the admin can see the failed
  command lingering in pending state and re-issue

### Requirement: Pub/Sub COMMAND notification handler updates command state

The `ProcessPubSubPush` dispatcher in `server/mdm/android/service/pubsub.go` SHALL handle the previously declared but
unhandled `android.PubSubCommand` notification type. The handler SHALL authenticate the Pub/Sub token via the existing
`authenticatePubSub` helper, decode the notification payload as an `androidmanagement.Command` resource, look up the
matching `mdm_android_commands` row by AMAPI operation name, and update the row's `status` (to `acknowledged` or `error`)
and error fields. The handler SHALL NOT modify `host_mdm_actions` — refs remain set; `GetHostLockWipeStatus` joins to
`mdm_android_commands` and computes `IsLocked()` / `IsWiped()` from the joined status. (This mirrors Apple's state model,
where `lock_ref` stays set after ack and the joined `nano_commands` row carries the `Acknowledged` status; refs are only
cleared on transitions like Unlock.) If the notification refers to an operation Fleet does not track, the handler SHALL
log at debug level and return success (idempotent no-op).

#### Scenario: Successful LOCK acknowledgement updates command status; ref stays set

- **GIVEN** a row exists in `mdm_android_commands` with `command_type='LOCK'`, `status='pending'`, and a corresponding
  `host_mdm_actions.lock_ref` pointing to it
- **WHEN** the Pub/Sub endpoint receives a `notificationType=COMMAND` payload with no `errorCode` for that operation
- **THEN** the handler SHALL update the row's `status` to `acknowledged`
- **AND** the handler SHALL NOT modify `host_mdm_actions` (the `lock_ref` remains set)
- **AND** a subsequent call to `GetHostLockWipeStatus(ctx, host)` SHALL return a `HostLockWipeStatus` whose `IsLocked()`
  evaluates to true (computed from the joined `acknowledged` row)

#### Scenario: Failed WIPE acknowledgement persists error; ref stays set

- **GIVEN** a row exists in `mdm_android_commands` with `command_type='WIPE'`, `status='pending'`, and a corresponding
  `host_mdm_actions.wipe_ref`
- **WHEN** the Pub/Sub endpoint receives a `notificationType=COMMAND` payload with `errorCode='UNSUPPORTED'` for that
  operation
- **THEN** the handler SHALL update the row's `status` to `error` and persist `error_code` and `error_message`
- **AND** the corresponding `host_mdm_actions.wipe_ref` SHALL remain set, so the admin can investigate the failed command
  before reissuing

#### Scenario: Unknown command notification is a no-op

- **GIVEN** no row in `mdm_android_commands` matches the operation name in an incoming COMMAND notification
- **WHEN** the handler is invoked
- **THEN** the handler SHALL log at debug level and return HTTP 200 with no error
- **AND** the handler SHALL NOT modify any other table

### Requirement: GetHostLockWipeStatus computes Android pending/done state

The `GetHostLockWipeStatus(ctx, host)` datastore method SHALL include an `case "android":` branch that, for hosts with
`fleet_platform = 'android'` in `host_mdm_actions`, reads the referenced `mdm_android_commands` rows (one per non-NULL
`lock_ref` / `wipe_ref`) and populates the `HostLockWipeStatus` struct's status fields with semantics equivalent to the
Apple branch. The same logic SHALL apply in `GetHostsLockWipeStatusBatch`. The `HostLockWipeStatus` struct methods
`IsPendingLock()`, `IsLocked()`, `IsPendingWipe()`, `IsWiped()` in `server/fleet/scripts.go` SHALL also be extended with
an `android` arm that reads the joined `mdm_android_commands.status` (`pending` ⇒ `IsPending*` true; `acknowledged` ⇒
`IsLocked()` / `IsWiped()` true; any other value ⇒ false), since the existing arms fall through to script-based or
default-false paths for Android.

#### Scenario: Pending LOCK on Android returns IsPendingLock

- **GIVEN** a host with `host_mdm_actions.lock_ref` set to a `command_uuid`, `fleet_platform='android'`, and the
  corresponding `mdm_android_commands.status='pending'`
- **WHEN** `GetHostLockWipeStatus` is called
- **THEN** the returned `*fleet.HostLockWipeStatus` SHALL satisfy `IsPendingLock() == true`
- **AND** SHALL satisfy `IsLocked() == false`

#### Scenario: Acknowledged LOCK on Android returns IsLocked

- **GIVEN** a host with `host_mdm_actions.lock_ref` set and `mdm_android_commands.status='acknowledged'`
- **WHEN** `GetHostLockWipeStatus` is called
- **THEN** the returned `*fleet.HostLockWipeStatus` SHALL satisfy `IsLocked() == true` and `IsPendingLock() == false`

### Requirement: Frontend dropdown surfaces Android commands per ownership mode

The host details Actions dropdown SHALL surface Lock, Wipe, Clear passcode, and Unenroll for Android hosts according to the
following visibility table:

| Host state | Lock | Wipe | Clear passcode | Unenroll |
|---|:---:|:---:|:---:|:---:|
| BYO Android, MDM on, Premium | ✓ | — | ✓ | ✓ |
| COBO Android, MDM on, Premium | ✓ | ✓ | ✓ | — |
| Any Android, MDM off | — | — | — | — |
| Any Android, free tier | — | — | — | — |

The "Wipe" item SHALL NOT appear for BYO Android; the "Unenroll" item SHALL NOT appear for COBO Android (COBO uses
"Delete" instead). Both items SHALL be hidden when Android MDM is not enabled and configured. All items SHALL respect the
existing Admin/Maintainer role requirement and Premium-tier requirement.

#### Scenario: BYO Android with MDM on shows Lock, Clear passcode, Unenroll

- **WHEN** the dropdown is rendered for a BYO Android host with `companyOwned=false`, MDM enabled, Premium tier, and the
  viewing user is an Admin or Maintainer
- **THEN** the menu SHALL include `Lock`, `Clear passcode`, and `Unenroll`
- **AND** the menu SHALL NOT include `Wipe`

#### Scenario: COBO Android with MDM on shows Lock, Wipe, Clear passcode

- **WHEN** the dropdown is rendered for a COBO Android host with `companyOwned=true`, MDM enabled, Premium tier, and the
  viewing user is an Admin or Maintainer
- **THEN** the menu SHALL include `Lock`, `Clear passcode`, and `Wipe`
- **AND** the menu SHALL NOT include `Unenroll`

#### Scenario: Android with MDM off hides all command items

- **WHEN** the dropdown is rendered for an Android host on a team where Android MDM is not enabled
- **THEN** the menu SHALL NOT include `Lock`, `Wipe`, `Clear passcode`, or `Unenroll`

### Requirement: fleetctl `mdm clear-passcode` subcommand

The fleetctl CLI SHALL include a new `mdm clear-passcode --host=<ident>` subcommand that wraps the
`POST /api/v1/fleet/hosts/:id/clear_passcode` endpoint. The subcommand SHALL print a confirmation message on success and an
error message on failure. The subcommand SHALL NOT print or store any password (no password is generated server-side).

#### Scenario: fleetctl clear-passcode succeeds

- **WHEN** an admin runs `fleetctl mdm clear-passcode --host=<ident>` against an Android host with MDM enabled
- **THEN** fleetctl SHALL POST to `/hosts/:id/clear_passcode` with the resolved host ID
- **AND** SHALL print a success line indicating the command was queued
- **AND** SHALL NOT print any password

#### Scenario: fleetctl clear-passcode fails for non-mobile platform

- **WHEN** an admin runs `fleetctl mdm clear-passcode --host=<ident>` against a Windows or Linux host
- **THEN** fleetctl SHALL surface the server's 400 BadRequestError message indicating that Clear passcode is only
  supported on mobile platforms
- **AND** SHALL exit with a non-zero status code
