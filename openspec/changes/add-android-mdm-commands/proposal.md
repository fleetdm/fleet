## Why

IT admins managing Android hosts cannot currently issue Lock, Wipe, or Clear-passcode MDM commands from Fleet. The existing
host details "Actions" dropdown hides these options for Android, and the corresponding REST/`fleetctl`/UI surfaces explicitly
exclude `android` in their platform branches (e.g. `frontend/.../HostActionsDropdown/helpers.tsx:188` `!isAndroid(hostPlatform)`
guards in `canLockHost`, `canWipeHost`, `canUnlock`; `ee/server/service/hosts.go:65` switch covers only
`darwin|ios|ipados|windows|linux`; `ee/server/service/mdm.go:1810` `clearPasscodeApple` is the only branch). PR #43266 already
merged the REST API documentation contract for these endpoints to support Android. Tracked in issue #41683 for milestone 4.87.0.

In parallel, the existing BYO Android Unenroll path calls `EnterprisesDevicesDelete` (`device.delete`). Per the Figma dev note for
this story, that API call expires if the host is offline more than 30 days, leaving the work profile installed indefinitely. The
remedy is to switch BYO Unenroll to send an AMAPI `WIPE` command (which on a BYO device removes the work profile, per AMAPI docs)
so the queued command waits for the device to come online instead of expiring server-side.

## What Changes

This change implements the end-to-end command flow on top of the merged REST API contract:

- **AMAPI client capability**: extend `androidmgmt.Client` with `EnterprisesDevicesIssueCommand(ctx, deviceName, command) (*Operation, error)`,
  implemented in `google_client.go` and `proxy_client.go`, plus a mock in `server/mdm/android/mock/`.
- **android.Service** gains three new methods — `LockAndroidHost`, `WipeAndroidHost`, `ClearAndroidPasscode` — each of which
  resolves the host's AMAPI device name, issues the appropriate command, persists tracking state, and returns to the caller.
- **EE service dispatch**: add `case "android":` branches to `enqueueLockHostRequest`, `enqueueWipeHostRequest`, and `ClearPasscode`,
  mirroring the existing `case "android":` for unenroll in `server/service/mdm.go:3737`.
- **Database**: new table `mdm_android_commands` keyed by Fleet-generated `command_uuid`, holding the AMAPI operation name and
  command status. `host_mdm_actions.{lock_ref, wipe_ref}` continues to be the platform-agnostic state spine — pointers into
  this new table for Android, exactly as they point into `nano_commands` for Apple.
- **Pub/Sub COMMAND handler**: implement the previously-declared `android.PubSubCommand` notification path
  (`server/mdm/android/pubsub.go:9`, currently falling through to `default:` in `ProcessPubSubPush`). The handler updates
  `mdm_android_commands.status` to `acknowledged` or `error` (persisting `error_code` / `error_message` when present). The
  handler does NOT clear `host_mdm_actions.{lock_ref, wipe_ref}` on success — the refs stay set so `GetHostLockWipeStatus`
  can join to the now-acknowledged command row and compute `IsLocked()` / `IsWiped()`. This mirrors the Apple state model
  where `lock_ref` remains and the joined `nano_commands` row carries the `Acknowledged` status. Command errors are surfaced
  via the persisted row + structured logs; lock/wipe activities are emitted at enqueue time (not at ack), so there is no
  activity-failure event to fire.
- **BYO Unenroll behavior change** (in scope, per Figma dev note): `UnenrollAndroidHost` for BYO hosts switches from
  `EnterprisesDevicesDelete` to `IssueCommand(WIPE)`. COBO unenroll continues to use `EnterprisesDevicesDelete` (matches the
  Figma's "Delete" menu item, which is separate from "Wipe"). The user-facing operation remains "Unenroll" — the WIPE command
  is an internal implementation detail to survive >30 days offline. BYO Unenroll continues to emit `ActivityTypeMDMUnenrolled`
  (existing behavior). BYO Unenroll does NOT write `host_mdm_actions.wipe_ref` — there's no user-facing wipe state, so no
  pending-state spine entry is needed.
- **GetHostLockWipeStatus** gains an `case "android":` arm that joins `host_mdm_actions` to `mdm_android_commands` to surface
  pending/acknowledged/error states for server-side pending-state guards (blocks double-issue).
  **The host-details "Wipe pending" / "Wiped" badges SHALL NOT display for Android** (product decision, supersedes Figma badge
  screenshots and the original test plan line "Wipe pending badge next to host name ... like we do for iOS/iPadOS"). The
  badge components exist for iOS/iPadOS; the frontend filters Android out of the display path.
- **Activity log**: reuse the existing `ActivityTypeLockedHost`, `ActivityTypeWipedHost`, `ActivityTypeClearedPasscode`,
  `ActivityTypeMDMUnenrolled` types. Activity strings in the Figma activity feed match Locked/Wiped/ClearedPasscode for the
  three new commands. **BYO Android Unenroll continues to emit `ActivityTypeMDMUnenrolled` (existing behavior preserved)**
  — product decision, supersedes the Figma activity feed which shows `Tress wiped Huck's Pixel 10.` on a BYO host. The
  WIPE command is an internal mechanism; the user-facing operation remains "Unenroll." No new activity types.
- **Frontend dropdown**: rewrite the Android-exclusion guards in `HostActionsDropdown/helpers.tsx` to platform-aware logic that
  shows Clear passcode + Lock + Unenroll for BYO Android, and Clear passcode + Lock + Wipe + Delete for COBO Android, hidden
  when MDM is disabled. Premium-only.
- **Frontend modals**: three new confirmation modals (Lock, Wipe, Clear passcode) following the same pattern as the existing
  Apple modals, with verbatim copy from the Figma "Ready" page. Each requires an "I wish to ..." checkbox before the confirm
  button enables.
- **fleetctl**: `fleetctl mdm lock --host=X` and `fleetctl mdm wipe --host=X` extend to accept Android hosts (no new
  subcommands — server-side validation in `LockHost`/`WipeHost` already drives platform behavior). New `fleetctl mdm
  clear-passcode --host=X` subcommand wraps the existing `POST /hosts/:id/clear_passcode` endpoint.

Non-goals:

- **No password generation for Clear passcode.** Product confirmed 2026-05-20: clear passcode CLEARS the passcode, does NOT
  generate a new one. The server sends AMAPI `RESET_PASSWORD` with an empty `newPassword`. Matches Figma modal copy: "The
  user can unlock the device without entering a passcode" (COBO) and "This only clears the work profile passcode" (BYO).
  The issue body's line "fleetctl mdm clear-passcode ... returns the generated password" is superseded — `fleetctl mdm
  clear-passcode` returns `{command_uuid, request_type, platform}` with no password field.
- **No `operations.get` polling.** AMAPI command completion is signaled exclusively via the Pub/Sub `COMMAND` notification,
  matching how the rest of Fleet's Android MDM tracks state (enrollment and status reports are also Pub/Sub-only). Customers
  who don't have working Pub/Sub already fail at enrollment; commands have the same dependency.
- **No host-header badges of any kind for Android.** Per product decision (2026-05-20), Android does NOT get "Wipe pending"
  / "Wiped" / "Lock pending" / "Locked" / "Clear passcode pending" badges. This supersedes the Figma badge screenshots and
  the issue body's test plan line about "Wipe pending badge ... like we do for iOS/iPadOS." The badge components for
  iOS/iPadOS remain; the frontend gates them on platform to exclude Android.
- **No re-view of cleared passcode.** Since no password is generated, there's nothing to re-view.
- **No janitor for command expiry in v1.** AMAPI commands have a default 10-minute duration but can be set to longer (no max).
  If a command expires server-side without Pub/Sub COMMAND notification, the host stays in a pending state until manually
  cleaned up. **Spike output may revise this** (see `ai/android-commands/spike-plan.md`); if AMAPI emits a COMMAND notification
  on expiry, no janitor is needed.
- **No changes to the `Show MDM commands` host-activity toggle behavior.** The existing UI surface that lists raw MDM commands
  (Apple, Windows) gains Android commands via the same code path — but only if the toggle is already wired to read from
  `mdm_android_commands`. Out of scope to refactor that toggle; in scope to populate the table so it can be surfaced later.

## Capabilities

### New Capabilities

- `mdm-android-commands`: Issuing Lock, Wipe, and Clear-passcode commands to Android hosts via AMAPI, tracking command state
  via Pub/Sub COMMAND notifications, reusing the cross-platform `host_mdm_actions` state spine for server-side pending-state
  guards, and surfacing command outcomes via existing activity items. No host-header badges for Android (product decision —
  see design.md). BYO vs COBO routing is enforced both server-side (platform branches in `ee/server/service/`) and
  client-side (dropdown visibility in `helpers.tsx`).

### Modified Capabilities

None — no prior accepted spec covers Android MDM commands in `openspec/specs/`.

## Impact

- **Database**: one additive migration creating `mdm_android_commands` with `DATETIME(6) NOT NULL DEFAULT NOW(6)` /
  `DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6)` timestamp precision (matching the convention in
  `20251028140300_AddInHouseAppsToUnifiedQueue.go` and `20251106000000_AddConditionalAccessSCEPTables.go`).
- **Server types**: `server/mdm/android/android.go` gains a `MDMAndroidCommand` struct; `server/mdm/android/service.go`
  gains three interface methods; `server/mdm/android/service/androidmgmt/client.go` gains the IssueCommand method.
- **Service layer**: dispatch additions in `ee/server/service/hosts.go` (Lock, Wipe enqueue functions) and
  `ee/server/service/mdm.go` (`ClearPasscode` restructured from `if IsApplePlatform` to `switch host.FleetPlatform()`).
- **Datastore**: new methods on `fleet.Datastore` — `NewAndroidMDMCommand`, `GetAndroidMDMCommandByUUID`,
  `UpdateAndroidMDMCommandFromPubSub`, `GetAndroidMDMCommandByOperationName`. `GetHostLockWipeStatus` gains an
  `case "android":` arm.
- **BYO Unenroll change**: `server/mdm/android/service/service.go` `UnenrollAndroidHost` branches on `companyOwned`. For BYO,
  swaps the AMAPI call from `EnterprisesDevicesDelete` to `IssueCommand(WIPE)` and inserts a `mdm_android_commands` row.
  Does NOT write `host_mdm_actions.wipe_ref` (no user-facing wipe state for BYO Unenroll). Continues to emit
  `ActivityTypeMDMUnenrolled` via the existing `MDMUnenroll` caller (no activity change).
- **Frontend**: `frontend/pages/hosts/details/HostDetailsPage/HostActionsDropdown/helpers.tsx` reworked to remove
  `!isAndroid(hostPlatform)` guards; three new modals under `.../modals/` (LockHostModal, WipeHostModal, ClearPasscodeModal —
  the latter two need Android-specific copy variants). `UnenrollMdmModal.tsx` body copy is unchanged for Android — the
  user-facing flow stays "unenroll" with no mention of wipe semantics. The badge-rendering code on the host details page
  header must be gated on platform to exclude Android (no "Wipe pending" / "Wiped" badges for Android). The "Show MDM
  commands" toggle on the host activity card already exists; this change populates the new table but does not modify the
  toggle's read path.
- **fleetctl**: `cmd/fleetctl/fleetctl/mdm.go` gains an `mdmClearPasscodeCommand` subcommand and updates the help text on
  existing `lock` / `wipe` subcommands to acknowledge Android support.
- **Pub/Sub**: `server/mdm/android/service/pubsub.go` adds `handlePubSubCommand`. The `android.PubSubCommand` constant is
  already declared but unhandled.
- **Tests**: integration tests under `server/service/integration_android_*_test.go` covering happy path + error path for
  Lock, Wipe, Clear passcode on both BYO and COBO; Pub/Sub COMMAND notification handler unit tests under
  `server/mdm/android/service/pubsub_test.go`; datastore tests for the new `mdm_android_commands` table.
- **Docs**: REST API docs for the three endpoints are already merged by PR #43266. Update the feature guide
  https://fleetdm.com/guides/lock-wipe-hosts. Update activity audit log doc at
  `docs/Contributing/reference/audit-logs.md` if existing entries omit Android platform examples.
- **Risk**: Medium. Wipe is destructive and irreversible. Premium-only; both backend service method and frontend dropdown
  must check tier. BYO Unenroll behavior change is a semantic change to a shipped feature — see the spike plan for
  pre-flight validation on real devices before merging. Load testing: minimal new load (commands are admin-initiated, low
  volume), but the new `mdm_android_commands` table joins into `GetHostLockWipeStatus`, which is read on every host details
  page load — index `host_uuid` to keep host details latency stable.
