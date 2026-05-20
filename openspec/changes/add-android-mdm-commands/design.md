## Context

Fleet supports Lock, Wipe, and Clear-passcode MDM commands across macOS, iOS/iPadOS, Windows, and Linux today. Each platform has
a different transport — Apple via the MDM-protocol commands shipped by `mdmAppleCommander`, Windows via `WipeHostViaWindowsMDM`,
Linux via signed scripts. Common to all platforms: `host_mdm_actions.{lock_ref, wipe_ref}` holds a UUID that points into the
platform's command-tracking table, and `GetHostLockWipeStatus` reads that pointer to compute the cross-platform "is this host
locked / wiped / pending" state.

Android, added in 4.78, has no corresponding command-tracking surface. The only AMAPI-side mutation Fleet performs is policy
PATCH (via `mdm_android_policy_requests`) and `EnterprisesDevicesDelete` on unenroll. AMAPI exposes per-device commands via
`enterprises.devices.issueCommand` returning a long-running `Operation`, with command-completion signals delivered through the
same Pub/Sub mechanism Fleet already uses for enrollment and status reports. Pub/Sub declares a `COMMAND` notification type
(`server/mdm/android/pubsub.go:9`) but the handler currently falls through to `default:` and ignores them.

This change wires AMAPI commands into Fleet's existing state spine, behind the same REST and UI surfaces that already exist for
Apple, Windows, and Linux. The REST API documentation for these endpoints already merged via PR #43266.

The Figma "Ready" page for issue #41683 is the source of truth for UX. Verbatim modal copy, dropdown options per ownership mode,
flash messages, activity strings, and dev notes are captured in `ai/android-commands/figma-content.md`.

## Goals / Non-Goals

**Goals:**

- Round-trip three commands end-to-end: REST → service dispatch → AMAPI `IssueCommand` → device → Pub/Sub COMMAND notification →
  state machine update → activity feed. No host-header badges for Android (product decision 2026-05-20 — see Decision below).
- BYO Unenroll switches to AMAPI `WIPE` so the command survives a host being offline >30 days. Activity logging is unchanged
  (continues to emit `ActivityTypeMDMUnenrolled`).
- Reuse existing activity types (`ActivityTypeLockedHost`, `ActivityTypeWipedHost`, `ActivityTypeClearedPasscode`,
  `ActivityTypeMDMUnenrolled`) — no new types in this change.
- Pub/Sub-only status path. No polling, no janitor cron in v1.
- Premium gated end-to-end (matches existing lock/wipe/clear-passcode behavior).
- Microsecond timestamp precision per CLAUDE.md guidance.

**Non-Goals:**

- Generating or returning a password as part of Clear passcode (per Figma; see Decision below).
- Re-viewable passcodes (no password to re-view).
- "Lock pending" / "Clear passcode pending" badges (Figma only shows Wipe pending / Wiped).
- Lock screen messaging or device-side custom unlock instructions (AMAPI supports it but Figma does not request it).
- AMAPI `START_LOST_MODE` / `STOP_LOST_MODE` (analogous to iOS Lost Mode — out of scope for this story).
- Restructuring the Show MDM commands toggle on host activity. New Android commands populate `mdm_android_commands`; surfacing
  them in that toggle is a follow-up.
- Polling `operations.get` as a fallback for missed Pub/Sub events.

## Decisions

### Dispatch through `android.Service`, not a separate commander

The existing pattern for Android Unenroll is:

```go
// server/service/mdm.go:3737
case "android":
    svc.androidSvc.UnenrollAndroidHost(ctx, host.ID)
```

`androidSvc` already owns AMAPI client construction, authentication-secret retrieval, enterprise resolution, and the import-cycle
dance that comes from passing a `*fleet.Host` across the boundary. Introducing a separate `mdmAndroidCommander` analogous to
`mdmAppleCommander` would duplicate all of this with no benefit — the EE service layer is the right place for the platform switch.

- **Decision:** Add three new methods to `android.Service`:
  - `LockAndroidHost(ctx context.Context, hostID uint) (commandUUID string, err error)`
  - `WipeAndroidHost(ctx context.Context, hostID uint) (commandUUID string, err error)`
  - `ClearAndroidPasscode(ctx context.Context, hostID uint) (commandUUID string, err error)`
- Pre-validation (platform check, pending-state guards, BYO vs COBO, MDM-enabled check) lives in the EE service layer
  (`enqueueLockHostRequest`, `enqueueWipeHostRequest`, `ClearPasscode`), alongside the existing apple/windows/linux validations.
  The android service is a thin command executor.
- **Alternative considered:** A new `mdmAndroidCommander` struct injected into the EE service. Rejected because it gains nothing
  and inverts the dependency direction set by Unenroll. Future Android MDM commands (REBOOT, START_LOST_MODE, etc.) will follow
  the same pattern.

### New table `mdm_android_commands`, with `host_mdm_actions` as the cross-platform state spine

`host_mdm_actions.{lock_ref, wipe_ref}` holds a UUID. For Apple it points into `nano_commands`; for Windows it points into
`mdm_windows_commands`; for Linux it points into `host_script_results`. Android has no equivalent table today.

- **Decision:** Create `mdm_android_commands` with columns:
  - `command_uuid VARCHAR(36) NOT NULL PRIMARY KEY` — Fleet-generated UUID, what `host_mdm_actions.lock_ref` / `wipe_ref` points at.
  - `host_uuid VARCHAR(255) NOT NULL` — for Pub/Sub lookups and joins.
  - `operation_name VARCHAR(255) NOT NULL` — full AMAPI operation name `enterprises/X/devices/Y/operations/Z` (longer than 36,
    so it does **not** fit in `host_mdm_actions.lock_ref`).
  - `command_type VARCHAR(32) NOT NULL` — one of `LOCK`, `RESET_PASSWORD`, `WIPE`.
  - `status VARCHAR(32) NOT NULL` — one of `pending`, `acknowledged`, `error`.
  - `error_code VARCHAR(64) NULL` — AMAPI `errorCode` enum value when failed.
  - `error_message TEXT NULL` — extracted human-readable AMAPI error.
  - `request_payload JSON NULL` — for audit and debugging (mirrors `mdm_android_policy_requests.payload`).
  - `created_at DATETIME(6) NOT NULL DEFAULT NOW(6)`.
  - `updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6)`.
  - Indexes on `host_uuid` and `operation_name`.
- **Decision:** `host_mdm_actions.{lock_ref, wipe_ref}` continues to hold the Fleet-generated `command_uuid` (36 chars, fits the
  existing VARCHAR(36) column). The longer `operation_name` lives in the sibling table.
- `GetHostLockWipeStatus` gains a `case "android":` branch that reads the referenced row from `mdm_android_commands` and computes
  the `IsPendingLock` / `IsPendingWipe` / `IsLocked` / `IsWiped` flags analogously to the Apple branch.
- **Alternative considered:** Widening `host_mdm_actions.lock_ref` to VARCHAR(255) and storing the operation name directly.
  Rejected — it pollutes the cross-platform table with AMAPI specifics, and we still need a row to track command_type,
  status, error_code, etc. Centralizing in a sibling table mirrors Apple's `nano_commands` precisely.
- **Alternative considered:** Reusing `mdm_android_policy_requests`. Rejected — that table is purposefully scoped to policy
  mutations (has `policy_id`, `policy_version`, `applied_policy_version`); commands have a different lifecycle.

### Pub/Sub `COMMAND` is the only command-status path

AMAPI offers two ways to learn a command's outcome: (a) polling `operations.get(name)` until `Done=true`, or (b) consuming the
Pub/Sub `COMMAND` notification, which fires when the device acknowledges the command and carries the populated `Command`
resource (with `errorCode`, status fields, etc.).

- **Decision:** Implement `handlePubSubCommand` in `server/mdm/android/service/pubsub.go`. Dispatch from `ProcessPubSubPush`:
  ```go
  case android.PubSubCommand:
      return svc.handlePubSubCommand(ctx, token, rawData)
  ```
- The handler decodes the payload, looks up the matching `mdm_android_commands` row by `operation_name`, and updates `status`
  (to `acknowledged` or `error`) plus `error_code` / `error_message` when present. The handler **does NOT** clear
  `host_mdm_actions.{lock_ref, wipe_ref}`. The refs stay set; `GetHostLockWipeStatus` joins to `mdm_android_commands` and
  computes `IsLocked()` / `IsWiped()` from the joined `status` (`acknowledged` ⇒ locked or wiped). This mirrors the Apple
  state model: `host_mdm_actions.lock_ref` for Apple stays set after ack, and the joined `nano_commands` row carries the
  `Acknowledged` status. Refs are only cleared on transitions like Unlock (Apple does the same in
  `server/datastore/mysql/scripts.go` `buildHostLockWipeStatusUpdateStmt`).
- **HostLockWipeStatus methods must learn Android.** `fleet.HostLockWipeStatus.IsPendingLock()`, `IsLocked()`,
  `IsPendingWipe()`, `IsWiped()` (`server/fleet/scripts.go:724-800`) currently switch on `HostFleetPlatform` and handle only
  `darwin|ios|ipados` (MDM-command-based) and `windows|linux` (script-based). For Android they fall through to the
  script-based / `default:false` paths, which would always return false even with the datastore populated. The pending-state
  guards on `LockHost` / `WipeHost` would not block double-issue. Add an explicit `case "android":` branch in each of these
  four methods that checks the joined `mdm_android_commands.status` (`pending` ⇒ IsPending*, `acknowledged` ⇒
  IsLocked/IsWiped, anything else ⇒ false). This is a task in `tasks.md` Phase 4 (datastore + struct method extension);
  not just a datastore change.
- **No polling, no janitor in v1.** Pub/Sub is already a hard dependency of Android MDM (enrollment and status reports use it
  exclusively); commands inherit the same dependency. If a customer's Pub/Sub is broken, the rest of Android MDM is already
  broken.
- **Spike output may revise the v1-no-janitor decision** — see `ai/android-commands/spike-plan.md` step 5. Specifically: if
  AMAPI does NOT emit a COMMAND notification when a command expires (default duration 10 min), a janitor cron is needed to
  clear stale rows. The spike answers this with real-device data.
- **Alternative considered:** Polling `operations.get` from a worker. Rejected — higher latency, more infrastructure for no
  reliability gain over what Pub/Sub already provides for the rest of the integration.

### BYO Unenroll switches from `device.delete` to `IssueCommand(WIPE)` (internal mechanism, activity unchanged)

Per the Figma dev note on the BYO actions dropdown:

> "Unenroll" on BYO hosts should send "wipe" Android command, because current unenroll (`device.delete`) command expires if host
> is offline more than 30 days.

AMAPI's `WIPE` command, sent to a BYO device, removes the work profile (not a factory reset — that scope only applies to COBO).
It is queued at AMAPI and remains queued until the device comes online; it does not expire server-side the way `device.delete`
does.

- **Decision:** `UnenrollAndroidHost(ctx, hostID)` in `server/mdm/android/service/service.go:823` branches on `companyOwned`:
  - **BYO** (`companyOwned == false`): generate a `command_uuid`, call `IssueCommand(WIPE)`, insert `mdm_android_commands` row.
    Do NOT write `host_mdm_actions.wipe_ref` — see "no Android badges" decision below; without a UI badge to drive, there's
    no user-facing reason to occupy the cross-platform wipe state spine, and the user-facing operation remains "Unenroll" not
    "Wipe."
  - **COBO** (`companyOwned == true`): unchanged — `EnterprisesDevicesDelete`. This matches the Figma's separate "Delete" menu
    item for COBO. (Note: the COBO dropdown has both "Wipe" — which sends WIPE — and "Delete" — which calls device.delete.
    These are intentionally different.)
- **Activity logging (resolved 2026-05-20 by product):** BYO Android Unenroll continues to emit `ActivityTypeMDMUnenrolled`
  (the existing behavior — emitted by `server/service/mdm.go:MDMUnenroll` at line 3749). The Figma activity feed showing
  `Tress wiped Huck's Pixel 10.` on a BYO host is superseded by this product decision. **Rationale:** the WIPE command is an
  internal implementation detail to survive >30 days offline; the user-facing operation is still "Unenroll" so the activity
  reflects that. No code change needed in the activity emit path — `MDMUnenroll` continues to fire `MDMUnenrolled` for all
  platforms uniformly.
- **Pub/Sub COMMAND handler behavior for BYO Unenroll's WIPE:** since there's no `host_mdm_actions.wipe_ref` for this case,
  the handler updates `mdm_android_commands.status` to `acknowledged` (or `error`) but does not touch `host_mdm_actions`.
  The host's MDM-off transition continues to be driven by the existing `DELETED` Pub/Sub notification path
  (`server/mdm/android/service/pubsub.go` `handlePubSubStatusReport`), which fires after AMAPI removes the device record
  following a successful BYO WIPE.
- **Alternative considered:** Keep both `device.delete` and add a parallel WIPE for resilience. Rejected — sending two
  commands conflates two state machines and confuses the admin (which activity wins?).
- **Alternative considered:** Write `host_mdm_actions.wipe_ref` for BYO Unenroll to gate against double-issue. Rejected —
  the Unenroll endpoint has no existing pending-state guard for any platform; double-Unenroll on Android is rare and
  benign (AMAPI accepts the second WIPE as a no-op since the work profile is already removed).

### Clear passcode does NOT generate or return a password

The Figma modal copy is unambiguous:

- **COBO Clear passcode**: "This will clear the host passcode. The user can unlock the device without entering a passcode."
- **BYO Clear passcode**: "This only clears the work profile passcode."

Neither flow displays, generates, or stores a password. The issue body's line "fleetctl mdm clear-passcode ... returns the
generated password" predates the Figma and is superseded.

- **Decision:** Server sends `IssueCommand(RESET_PASSWORD)` with `newPassword=""` (empty). AMAPI clears the passcode to nothing,
  matching the modal copy. No password generation, no `new_password` column, no encryption pipeline.
- `POST /hosts/:id/clear_passcode` response shape for Android matches the existing Apple response shape:
  `{command_uuid, request_type, platform}` — populated with `request_type="RESET_PASSWORD"` and `platform="android"`.
- `fleetctl mdm clear-passcode --host=X` prints the success/failure status — no password output.
- **Resolved 2026-05-20 by product:** clear passcode clears the passcode, does NOT generate a new one. Confirms the
  recommended default below.
- **Alternative considered:** Implementing per-issue-body (generate, store encrypted, return). Rejected for v1 — adds
  encryption complexity, requires a new password-display UX (not in Figma), and contradicts the modal copy. Can be added in
  a follow-up if product reverses.

### Activity types reused, no new types in this change

The Figma activity strings match the existing templates:

| Figma string | Existing type |
|---|---|
| `Tress locked Huck's Pixel 10.` (dashboard) / `Tress locked this host.` (host detail) | `ActivityTypeLockedHost` |
| `Tress wiped Huck's Pixel 10.` / `Tress wiped this host.` | `ActivityTypeWipedHost` |
| `Tress cleared the passcode for Huck's Pixel 10.` / `Tress cleared the passcode for this host.` | `ActivityTypeClearedPasscode` |

The dev note on the dashboard activity reinforces this: "Actions already exist for 'cleared passcode', 'locked host' and
'wiped host'. Confirm that Android activities show up, too."

- **Decision:** Emit existing activity types at command-issue time (not at ack), matching the existing macOS/iOS/Windows
  behavior. The `ActivityTypeLockedHost` struct has a `ViewPIN bool` field used by macOS; for Android, set `ViewPIN=false`
  (no PIN to view).
- **Alternative considered:** New `ActivityTypeAndroidCommandIssued` / `...Failed` activities. Rejected — Fleet's activity
  feed is platform-agnostic by design; the existing types already accept a `HostPlatform` field where relevant.

### Pending-state machine reuses `host_mdm_actions` unchanged

`host_mdm_actions` already has `fleet_platform` (added by `20240301173035_AddFleetPlatformToHostMDMActions.go`). Writing
`fleet_platform = 'android'` and a UUID into `lock_ref` / `wipe_ref` is enough for `lockWipe.IsPendingLock()`,
`IsPendingWipe()`, etc. to return correct values.

- **Decision:** No schema change to `host_mdm_actions`. The new `mdm_android_commands` table holds all Android-specific state.
- **GetHostLockWipeStatus** is the only datastore method that needs an `case "android":` branch. The branch reads the row
  from `mdm_android_commands` keyed by the `lock_ref` / `wipe_ref` UUID and populates a status object analogous to the
  Apple branch.

### No host-header badges for Android (product decision 2026-05-20)

**Resolved by product:** Android does NOT get any host-header badges (no "Wipe pending", "Wiped", "Lock pending", "Locked",
or "Clear passcode pending"). The Figma badge screenshots showing `Wipe pending` and `Wiped` on a BYO host are superseded by
this decision. The issue body's test plan line — "If host is pending wipe, show 'Wipe pending' badge ... like we do for
iOS/iPadOS" — is also superseded.

- **Decision:** All four Android commands (Lock, Wipe, Clear passcode, BYO Unenroll-as-Wipe) are fire-and-flash. Admin sees a
  flash message ("Successfully sent request to ...") and the activity-feed entry; no host-header badge state appears at any
  point in the lifecycle.
- **Frontend impact:** the badge-rendering code on the host details page header that currently checks
  `lockWipe.IsPendingWipe()` / `IsWiped()` for iOS/iPadOS hosts must be gated to exclude Android. The badge components
  themselves stay (Apple hosts still use them); only the platform predicate changes.
- **Server-side state machine impact:** explicit user Lock / Wipe still writes `host_mdm_actions.{lock_ref, wipe_ref}` —
  this is needed for the pending-state guards in `LockHost` / `WipeHost` to block double-issue (matching every other
  platform). The guards are server-side and invisible to the user; they only surface if the admin tries a second action
  before the first completes, where they return a 422 error. BYO Unenroll's WIPE does NOT write `host_mdm_actions.wipe_ref`
  (see the BYO Unenroll decision above).
- **Alternative considered:** Keep badge parity with iOS/iPadOS. Rejected by product without further documented rationale;
  proceed with no-badge implementation.

### Premium-gated end-to-end

The existing Lock/Wipe/Clear-passcode endpoints are Premium-only (`ee/server/service/hosts.go`, `ee/server/service/mdm.go`).

- **Decision:** No tier-check changes — the existing Premium gate in the EE service automatically extends to Android because
  dispatch happens within the same `LockHost` / `WipeHost` / `ClearPasscode` method.
- Frontend dropdown: the existing `canLockHost` / `canWipeHost` / `canClearPasscode` already check `isPremiumTier`. The
  Android-branch rewrites preserve that gate.

### Dropdown visibility per ownership mode

From the Figma BYO and COBO dropdown screenshots and the dev notes:

| State | Visible items |
|---|---|
| BYO Android, MDM on | Transfer, **Clear passcode**, **Unenroll**, **Lock**, Delete |
| COBO Android, MDM on | Transfer, **Clear passcode**, **Lock**, **Wipe**, Delete |
| Any Android, MDM off | Transfer, Delete (only) |

- **Decision:** Modify `helpers.tsx` such that:
  - `canLockHost` allows Android when MDM is configured and Premium.
  - `canWipeHost` allows Android only when `IsCompanyOwned` (which Fleet derives from the existing `host_mdm.is_personal_enrollment` column inverted).
  - `canClearPasscode` allows Android always when MDM is configured (BYO clears work profile, COBO clears device).
  - "Unenroll" is always shown for BYO Android with MDM on. Its onClick sends the existing `POST /hosts/:id/mdm` request,
    which now triggers the wipe path server-side.
- **Conditional "Clear passcode" visibility on BYO** (Figma dev note: "Only show if host has a work profile password"). The
  Android Management API exposes `passwordCompliant` / `requiresPasswordPolicy` signals on the device status — see
  `ai/android-commands/open-product-questions.md` for whether Fleet has this data today or needs to add it. **For v1, show
  always** unless product clarifies; over-showing is safer than hiding a working command.

## Open Questions

Most product questions are resolved as of 2026-05-20. Remaining items are spike-driven (empirical), not product-driven.

1. ~~`fleetctl mdm clear-passcode` response shape~~ **— RESOLVED 2026-05-20:** clear passcode clears the passcode and does
   NOT generate a new one. Server sends `RESET_PASSWORD` with empty `newPassword`. fleetctl returns
   `{command_uuid, request_type, platform}` with no password field. Issue body line is superseded.

2. ~~BYO Unenroll activity type~~ **— RESOLVED 2026-05-20:** continue emitting `ActivityTypeMDMUnenrolled` (existing
   behavior). The WIPE command is an internal mechanism; the user-facing operation remains "Unenroll." No code change in
   activity emit path; `MDMUnenroll` continues to fire `MDMUnenrolled` for all platforms uniformly.

3. ~~BYO Unenroll modal copy~~ **— RESOLVED 2026-05-20:** the `UnenrollMdmModal` Android branch needs no change. Without a
   "Wipe pending" badge and with the activity still saying "unenrolled," the existing modal copy ("Company data and OS
   settings (work profile) will be deleted.") is consistent with what the admin sees.

4. **"Clear passcode" conditional visibility on BYO** ("only show if host has a work profile password") — _v1 default:_
   show always. Track work-profile password state in a follow-up if confused users surface in QA. (Tracked in
   `open-product-questions.md` Q4.)

5. **AMAPI command expiry behavior** — _spike-driven (Q5 in spike-plan.md)._ Determines whether a janitor cron is needed.
   Resolves before the real implementation phase begins.

6. ~~Wipe pending / Wiped badge behavior on BYO~~ **— RESOLVED 2026-05-20:** no badges shown for Android at all. The host
   record stays in Fleet with `MDM status = Off` after the existing `DELETED` Pub/Sub notification fires (same path as
   today for `device.delete`); the WIPE command's COMMAND notification updates `mdm_android_commands.status` only.

## Migration Plan

This is a single-server feature with no agent-side changes (unlike `add-android-cert-san-attributes`). Rollout is one PR
(or a small sequence), one server restart, and the feature lights up.

- **Step 1**: Land the migration creating `mdm_android_commands`. Safe — additive, no FKs, indexes on `host_uuid` and
  `operation_name`.
- **Step 2**: Land the AMAPI client capability (`IssueCommand` on `androidmgmt.Client` + mock). Pure addition, no behavior
  change.
- **Step 3**: Land `android.Service` new methods and the Pub/Sub COMMAND handler. The handler is gated by the previously
  declared but unhandled `android.PubSubCommand` notification type — adding it doesn't affect existing flows.
- **Step 4**: Land the BYO Unenroll behavior change (`UnenrollAndroidHost` branches). **High-risk step** — pre-flight on the
  spike's BYO test device before merging. Activity logging change ships with this step.
- **Step 5**: Land the EE service dispatch (`case "android":` branches in `enqueueLockHostRequest`, `enqueueWipeHostRequest`,
  `ClearPasscode`). Frontend dropdown rewrite ships in parallel.
- **Step 6**: Land the modals and `fleetctl mdm clear-passcode` subcommand.
- **Step 7**: Update feature guide at https://fleetdm.com/guides/lock-wipe-hosts.

No agent-side rollout coordination needed.
