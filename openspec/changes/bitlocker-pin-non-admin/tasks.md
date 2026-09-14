## 1. Prerequisites and scaffolding

- [ ] 1.1 Confirm the #52159 F1 fix (rotation must not add a TPM-only protector when a TPM+PIN protector exists) is merged or scheduled in the same release; add an orbit unit test covering a type 4 present before rotation
- [ ] 1.2 Add `CapabilityWindowsBitLockerPIN = "windows_bitlocker_pin"` to `server/fleet/capabilities.go` and include it in `GetOrbitClientCapabilities` on Windows
- [ ] 1.3 Add `fleet.ActivityTypeCreatedDiskEncryptionPIN{HostID, HostDisplayName}` (`created_disk_encryption_pin`) to `server/fleet/activities.go` with `HostIDs()`, and register it wherever activity types are enumerated (activity type list, `docs/Contributing/reference/audit-logs.md` on main)

## 2. Server: data model

- [ ] 2.1 Migration: create `host_bitlocker_pin_requests` (`host_id` PK, `pin_encrypted` nullable, `status` ENUM(`pending`,`delivered`,`set`,`failed`), `client_error` VARCHAR(255) default empty, `created_at`, `updated_at` TIMESTAMP(6)) with a test file (`/new-migration`)
- [ ] 2.2 Migration: add `fleetd_bitlocker_pin_capable` TINYINT(1) NOT NULL DEFAULT 0 to `mdm_windows_enrollments`, guarded with `columnExists`, with a test file
- [ ] 2.3 Datastore: `QueueBitLockerPINRequest(ctx, hostID, pin)` encrypting with the server private key (reuse the AES-GCM helper used by `mdm_config_assets`), replacing any existing row
- [ ] 2.4 Datastore: `GetBitLockerPINRequestState(ctx, hostID)` returning status, sanitized error, age, and a pending-and-fresh flag (5 minute TTL)
- [ ] 2.5 Datastore: `TakeBitLockerPINRequest(ctx, hostID)` returning the decrypted PIN for a fresh `pending` row exactly once. MySQL has no `UPDATE ... RETURNING`, so a plain read-then-write can hand the same PIN to two concurrent orbit polls: run it in a writer transaction (`ctxdb.RequirePrimary`) that does `SELECT ... FOR UPDATE` on the row, checks it is still `pending` and unexpired, then sets `delivered` and NULLs the ciphertext before committing. A second caller blocks on the lock and then sees `delivered`, so it gets nothing
- [ ] 2.6 Datastore: `SetBitLockerPINRequestOutcome(ctx, hostID, outcome, clientError)` (delete on `set`, store `failed` + reason otherwise) and `DeleteBitLockerPINRequest`
- [ ] 2.7 Datastore: `SetMDMWindowsEnrollmentFleetdBitLockerPINCapable(ctx, hostUUID, capable)` and add `FleetdBitLockerPINCapable` to `GetMDMWindowsHostConfigState`
- [ ] 2.8 Add the new methods to the `fleet.Datastore` interface, regenerate `server/mock/datastore_mock.go`, run `go test ./server/service/` to catch uninitialized mocks
- [ ] 2.9 MySQL tests for 2.3 to 2.7 covering replace-on-resubmit, TTL expiry, single-fetch semantics, and default-false capability

## 3. Server: endpoints and notifications

- [ ] 3.1 Device endpoint `POST /api/_version_/fleet/device/{token}/disk_encryption_pin` (request/response structs, `deviceAuthToken()`, registered with `errorLimiter`), validating 6 to 20 ASCII digits and eligibility (Windows, not server, MDM-connected, fleet requires PIN, `tpm_pin_set` false, capable fleetd). Place the implementation according to the tier decision from the design's open question: `ee/server/service` with `ErrMissingLicense` in core if Premium, core service if Free. Do not write the license gate before that is answered
- [ ] 3.2 Orbit endpoint `POST /api/fleet/orbit/disk_encryption_pin/request` under `oeWindowsMDM` returning `{pin}` once via 2.5, `404` otherwise; never log the value
- [ ] 3.3 Orbit endpoint `POST /api/fleet/orbit/disk_encryption_pin` under `oeWindowsMDM` with `{outcome, client_error}`: on `set` call `SetOrUpdateHostDiskTpmPIN(true)`, delete the request, create the activity with a nil user, `UpdateHostRefetchRequested(true)`; on `failed` store the sanitized reason; reject unknown outcomes and empty reasons
- [ ] 3.4 Orbit config: add `BitLockerPINRequestPending bool` to `OrbitConfigNotifications` and set it in the Windows branch of the config handler only while the request is fresh and eligibility still holds; persist the `windows_bitlocker_pin` capability next to `fleetd_sync_capable`
- [ ] 3.5 Device host response: add `fleetd_can_set_pin` and `pin_request{status,error}` under `mdm.os_settings.disk_encryption` (server type plus `hostDetailResponseForHost` for the device path only)
- [ ] 3.6 Fleet Desktop summary: add `NeedsBitLockerPIN bool` to `DesktopNotifications` and set it in `ee/server/service/devices.go` when `action_required == create_pin` and the enrollment is PIN-capable
- [ ] 3.7 Discard pending requests when `require_bitlocker_pin` is turned off for the fleet or the host changes fleet (hook into the existing disk encryption settings update and host transfer paths)
- [ ] 3.8 Unit tests in `server/service` for 3.1 to 3.6 (eligibility matrix, validation, once-only fetch, outcome handling, notification gating) and an `integration-mdm` test driving submit, config poll, fetch, report, activity and refetch end to end
- [ ] 3.9 Client: `client/orbit_client.go` methods `GetBitLockerPINRequest()` and `SetOrUpdateBitLockerPINOutcome(outcome, clientError)`; `client/device_client.go` unchanged (Fleet Desktop reads the summary field)

## 4. Orbit: apply the PIN

- [ ] 4.1 `orbit/pkg/bitlocker`: `protectWithTPMAndPIN(pin)` on `Volume` (calls `ProtectKeyWithTPMAndPIN` with nil name and nil profile), error mapping for 0x80310068, 0x8031009A, 0x80310031, 0x80284008, 0x80310000, 0x80310030, 0x80310023 in `encryptErrHandler`
- [ ] 4.2 `setTPMAndPINProtectorOnCOMThread(volume, pin)` implementing preconditions (not server, FullyEncrypted, protection on, no type 4/6), add, verify type 4 present, delete all type 1 by ID, verify none remain, rollback (delete the new type 4) on any post-add failure; expose as `COMWorker.SetTPMAndPINProtector`
- [ ] 4.3 `orbit/pkg/update`: extend `windowsMDMBitlockerConfigReceiver.Run` to handle `BitLockerPINRequestPending` under the existing `TryLock`, fetch via 3.9, apply via 4.2, report outcome, zero the PIN; inject fetch/apply/report as function fields for tests
- [ ] 4.4 Table-driven tests for 4.2 (fake protector lists) and 4.3 (pending flag, mutex held, fetch 404, apply failure, report failure) with no COM
- [ ] 4.5 Orbit startup on Windows: idempotently register `HKLM\Software\Classes\AppUserModelId\FleetDM.FleetDesktop` with `DisplayName = "Fleet Desktop"` and `IconUri` to the Fleet Desktop icon; warn and continue on failure
- [ ] 4.6 `orbit/changes/49133-bitlocker-pin-non-admin` entry (no extension)

## 5. Fleet Desktop: Windows toast

- [ ] 5.1 New Windows-only package `orbit/pkg/toast`: XML builder for a ToastGeneric toast with heading, body, protocol-activation button, tag, group, expiration; PowerShell script renderer; runner using `powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass` with `HideWindow`; AUMID selection (Fleet Desktop key present, else Windows PowerShell AUMID); `Remove(tag, group)` helper
- [ ] 5.2 Table-driven tests for the XML and script rendering (escaping of URL and copy, tag/group presence, expiration)
- [ ] 5.3 `orbit/cmd/desktop`: on Windows, when the summary has `NeedsBitLockerPIN`, post the toast once per process with the current My device URL plus `?create_pin=1`; re-post on token change while still needed; remove the toast when the flag clears; log and swallow failures
- [ ] 5.4 Manual verification matrix: Windows 10 and 11, Focus Assist on, toast disabled by policy, Constrained Language Mode, AUMID key present and absent, token rotation while pending

## 6. Frontend

- [ ] 6.1 `frontend/interfaces/host.ts`: add `fleetd_can_set_pin` and `pin_request` to the disk encryption os setting type; `frontend/services/entities/disk_encryption.ts`: `submitBitLockerPin(token, pin)`; add the device endpoint to `utilities/endpoints`
- [ ] 6.2 Rewrite `BitLockerPinModal` as the form from the Figma (title, intro, two masked `InputField`s with helper text, Cancel/Save), validation (6 to 20 digits, match), Save disabled until valid
- [ ] 6.3 Submit flow: waiting state, poll the device host query every 3 s for up to 90 s, success toast "Successfully set PIN." and close, failure toast "Couldn't set PIN. {error}. Try again or contact your IT admin." and re-enable, timeout copy
- [ ] 6.4 Keep the existing instructions modal as `BitLockerPinInstructionsModal` and render it when `fleetd_can_set_pin` is false
- [ ] 6.5 `DeviceUserPage`: open the modal on load when `location.query.create_pin` is set and `action_required === "create_pin"`
- [ ] 6.6 `DeviceUserBanners`: new end-user copy with Create PIN link; `HostDetailsBanners`: admin copy without a link; update both `.tests.tsx`
- [ ] 6.7 Activity rendering for `created_disk_encryption_pin` in `GlobalActivityItem` and the host activity item ("End user created a disk encryption PIN for {host}.") plus the activity type filter label
- [ ] 6.8 Jest tests for the modal (validation, waiting, success, failure, timeout) and the deep link; run `make lint-js`

## 7. Docs and release notes

- [ ] 7.1 Retarget #50463 (open) and re-apply #50335 (merged into `docs-v4.92.0`) to the 4.93.0 docs branch using `push-reference-docs`
- [ ] 7.2 `docs/Contributing/reference/api-for-contributors.md`: document the device submit endpoint, the two orbit endpoints, the new orbit config notification, and the new device host and desktop summary fields
- [ ] 7.3 Update `articles/enforce-disk-encryption.md` (remove the Refetch step, mention the login toast, note PIN changes go through Windows) and `docs/Contributing/architecture/mdm/disk-encryption.md` sequence to match the relay design
- [ ] 7.4 `changes/49133-bitlocker-pin-non-admin` entry (no extension) for the server and frontend

## 8. Verification

- [ ] 8.1 Run affected Go packages (`server/service`, `server/datastore/mysql`, `orbit/pkg/bitlocker`, `orbit/pkg/update`, `client`) and `make lint-go-incremental`; `yarn test` for touched frontend paths
- [ ] 8.2 On `bl-fix-test` (Azure, vTPM): end to end as a standard user through the My device page, confirm protector list becomes type 4 + type 3 with no type 1, activity recorded, banner clears without Refetch, then remove the PIN protector before any reboot
- [ ] 8.3 On a physical Windows 11 machine or Hyper-V VM with a vTPM: same flow, then reboot and confirm the pre-boot PIN prompt accepts the PIN; repeat as a local admin
- [ ] 8.4 Negative cases: PIN already set, protection suspended, requirement turned off mid-flight, old fleetd (legacy modal, no toast), network loss between submit and poll
- [ ] 8.5 Post the completed test plan on GitHub #49133 and hand off to QA
