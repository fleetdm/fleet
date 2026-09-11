## Why

When a fleet requires a BitLocker startup PIN (`require_bitlocker_pin`), the only way for an end user to create one is Windows' own "Change how the drive is unlocked at startup", which needs UAC elevation. Organizations that run their users as standard (non-admin) accounts, such as Flywire (`customer-flavia`, GitHub #49133 and #46494), cannot get a PIN onto the disk without granting admin rights or sending a technician. Competing agents (Workspace ONE Intelligent Hub) prompt the user for the PIN and apply it themselves. Fleet already runs orbit as SYSTEM on every Windows host, so it can do the same. Milestone 4.93.0.

## What Changes

- Fleet Desktop (Windows) shows a toast at each login while the host needs a PIN: "Fleet: Action needed" / "Set your BitLocker PIN to protect this device", with a **Create PIN** button that opens the My device page at the Create PIN modal.
- The My device page replaces today's instructions-only "Create PIN" modal with a form (BitLocker PIN, Confirm PIN, client-side validation, Save). The same modal serves every user, admin or not; the old instructions modal remains only for hosts whose fleetd predates the new capability.
- The PIN travels browser to Fleet server to orbit: a new device-token endpoint queues an encrypted, short-lived PIN request; the orbit config response flags the pending request; orbit fetches the PIN once, applies it, and reports the outcome. The server never returns the PIN on any user-authenticated API and deletes it on delivery.
- Orbit gains a privileged TPM+PIN path: `ProtectKeyWithTPMAndPIN` on the OS volume, verification of the resulting protector, removal of the TPM-only protector that would otherwise negate the PIN, rollback on partial failure, and a new `windows_bitlocker_pin` fleetd capability the server persists per Windows MDM enrollment.
- On success the server marks the host's PIN as set, records a new `created_disk_encryption_pin` activity ("End user created a disk encryption PIN for {host}"), and requests a refetch so the banner clears without the user pressing Refetch. On failure the modal shows "Couldn't set PIN. {error}. Try again or contact your IT admin."
- The disk encryption banners on My device (end user) and Host details (admin) adopt the new copy.
- Orbit registers a "Fleet Desktop" AppUserModelID in the Windows registry so the toast displays under the Fleet Desktop name and icon.
- Fleet continues to leave `SystemDrivesDisallowStandardUsersCanChangePIN` unset, so PIN changes stay in the Windows UI. The Fleet path only creates a PIN where none exists and refuses to overwrite one.

No breaking changes. No new admin setting, no fleetctl or GitOps changes.

**Tier is unresolved and must be settled before implementation.** GitHub #49133 states "Changes to paid features or tiers: Free", but its own test plan has a "Premium gating (BitLocker management is a premium feature)" section asserting the opposite, and the REST API reference documents `windows_require_bitlocker_pin` as "Available in Fleet Premium". In practice the question is close to moot, because a PIN is only ever demanded when that Premium setting is on, so a Fleet Free host never reaches this flow. The proposal therefore does not assert a tier: see the open question in the design.

## Capabilities

### New Capabilities
- `bitlocker-pin-relay`: Server-side lifecycle of a PIN request: device-token submit endpoint, encrypted short-TTL storage, eligibility rules, orbit fetch and result endpoints, orbit config and Fleet Desktop notifications, `tpm_pin_set` update, `created_disk_encryption_pin` activity, refetch, and fleetd capability gating.
- `bitlocker-pin-agent`: Orbit (Windows) fetching a pending PIN, applying a TPM+PIN protector under the BitLocker receiver mutex with preconditions, protector verification, TPM-only protector removal, rollback, error mapping, and outcome reporting; plus advertising the `windows_bitlocker_pin` capability.
- `bitlocker-pin-end-user-ui`: The My device Create PIN modal (form, validation, waiting state, polling, success and failure messages, deep link), the legacy instructions fallback for old fleetd, and the updated banners on My device and Host details, plus the activity feed rendering.
- `fleet-desktop-windows-toast`: Fleet Desktop showing a Windows toast once per login while the server reports a needed PIN, re-posting it when the device token rotates, and orbit registering the Fleet Desktop AppUserModelID.

### Modified Capabilities
<!-- None. openspec/specs/ has no existing capabilities for this area. -->

## Impact

- **Server** (`server/service`, `server/datastore/mysql`, `server/fleet`, `ee/server/service`): two additive migrations (PIN request table, `fleetd_bitlocker_pin_capable` on `mdm_windows_enrollments`), datastore methods and mocks, new device endpoint `POST /api/_version_/fleet/device/{token}/disk_encryption_pin`, new orbit endpoints `POST /api/fleet/orbit/disk_encryption_pin/request` and `POST /api/fleet/orbit/disk_encryption_pin`, new fields on the orbit config notifications, the device host response and the Fleet Desktop summary, new activity type and capability constant.
- **Agent** (`orbit/pkg/bitlocker`, `orbit/pkg/update`, `client/orbit_client.go`, `server/fleet/capabilities.go`): new COM worker method and receiver branch, new capability, AUMID registration at startup. Requires a fleetd release (1.61.0 or the next after 1.60.0); older fleetd keeps today's behavior.
- **Fleet Desktop** (`orbit/cmd/desktop`, new `orbit/pkg/toast`): first Windows notification code, PowerShell-driven WinRT toast, token-rotation re-post.
- **Frontend** (`frontend/pages/hosts/details/DeviceUserPage`, `HostDetailsPage`, `DashboardPage/cards/ActivityFeed`, `frontend/interfaces`): modal rewrite, banner copy, deep link handling, activity rendering.
- **Docs**: contributor API reference for the three endpoints and new response fields, audit log entry (already merged to the 4.92.0 docs branch by #50335, must move to 4.93.0), `articles/enforce-disk-encryption.md` and the disk encryption architecture doc (#50463, also needs retargeting), `changes/` and `orbit/changes/` entries.
- **Prerequisite**: the #52159 fix that stops key rotation from re-adding a TPM-only protector, otherwise a PIN set through this flow is silently negated at the next rotation.
- **Security posture**: the server holds a user-chosen secret encrypted for at most minutes; anyone with code in the user session can already read the device token and could submit a PIN, whose worst case is a lockout the escrowed recovery key can undo. Details in the design.
