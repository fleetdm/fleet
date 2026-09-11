## Context

Fleet enforces BitLocker on Windows through orbit (runs as SYSTEM, drives `Win32_EncryptableVolume` over COM) and the Windows MDM channel (pushes `SystemDrivesRequireStartupAuthentication` so Manage BitLocker offers the PIN option). When `require_bitlocker_pin` is on, the server's `whereBitLockerStatus` keeps a host in Action required until `tpm_pin_set_verify` sees a type 4 or 6 protector, and the My device page shows a banner plus an instructions-only modal telling the user to use Manage BitLocker, which requires UAC elevation. Standard users cannot complete it (GitHub #49133, #46494, Flywire).

Two facts shape the design:

1. **Nothing on the device lets the browser or Fleet Desktop reach orbit.** Fleet Desktop and the My device page talk only to the Fleet server (device-token API). Orbit only polls the server (config every 30 s) and receives a stop signal through a named kernel event. There is no local IPC.
2. **The PIN is typed into a web page.** The Figma Ready page (`DXLYRxlD6di4vVzO3sZiel`, node `2:130`) draws the Create PIN modal with Fleet's web `Modal` and `InputField` components on the My device page, not a native window.

Windows constraints (Microsoft Learn, verified 2026-09-11): `ProtectKeyWithTPMAndPIN` creates a type 4 protector on the OS volume, needs no UAC when called as SYSTEM, accepts 6 to 20 digits (letters and symbols only when `SystemDrivesEnhancedPIN` is on, which Fleet never sets), returns `FVE_E_PROTECTOR_EXISTS` when one exists, and "the presence of the 'TPM' key protector type negates the effects of other TPM-based key protectors", so the TPM-only protector must be removed afterwards. Toasts from an unpackaged desktop app are silently dropped unless the AppUserModelID is registered (Start Menu shortcut or `Software\Classes\AppUserModelId\<AUMID>` registry key); the Fleet MSI installs no shortcut today.

Related in-flight work: #52159 F1 (key rotation re-adds a TPM-only protector next to TPM+PIN) must land first or with this change. The investigation with all sources is in `ai/bitlocker/49133-investigation.md`.

## Goals / Non-Goals

**Goals:**
- A standard or admin Windows user can create the required BitLocker PIN from the My device page with no UAC prompt, and Fleet reflects success without a manual Refetch.
- Users are told at each login that a PIN is needed, with a one-click path to the modal.
- The PIN is never logged, never returned by any user-authenticated API, and is stored only encrypted and only for the seconds to minutes between submission and delivery to orbit.
- The Fleet path can only create a PIN where none exists; it can never replace one.
- Hosts on older fleetd keep today's instructions modal with no regression.
- A `created_disk_encryption_pin` activity records every success.

**Non-Goals:**
- Changing an existing PIN through Fleet (stays in the Windows UI; Fleet does not set `SystemDrivesDisallowStandardUsersCanChangePIN`).
- An admin setting to allow or disallow non-admin PIN creation (`require_bitlocker_pin` already expresses intent; the story's test plan mentions such a setting but the product spec does not).
- Enhanced (alphanumeric) PINs or reading the host's `MinimumPIN` policy into the modal in v1 (the helper text stays "6–20 digits"; policy-tightened hosts surface the Windows error).
- Fleetctl, GitOps, YAML, REST API (user-authenticated) or license changes.
- A local IPC channel between Fleet Desktop or the browser and orbit.
- Pushing orbit config over WebSocket to cut the 30 s poll latency (follow-up).

## Decisions

### D1. Relay the PIN through the Fleet server (not a local channel)

**Chosen:** browser posts to a device-token endpoint, the server stores the PIN encrypted, orbit picks it up on its next config poll through an authenticated orbit endpoint, applies it, and reports back.

**Alternative rejected:** a named pipe or loopback listener on the device. The browser cannot open a named pipe, so it would need a loopback HTTPS listener with an `Origin` allowlist and a shared secret, and the only secret available to the page is the device token, which is world-readable on disk anyway. Moving the modal into a native Fleet Desktop window contradicts the design and needs a Windows UI toolkit Fleet Desktop lacks. The local-process threat (any user-mode code submitting a PIN) is identical in both options because both authenticate with the device token.

**Why the relay is acceptable:** it mirrors the LUKS "Create key" flow (device trigger, orbit notification, orbit result endpoint), reuses every existing building block, adds no device-side attack surface, and the server already holds the volume's recovery password, which is strictly more powerful than the PIN.

### D2. Storage and lifetime of the PIN on the server

One row per host in a new table (`host_bitlocker_pin_requests`: `host_id` PK, `pin_encrypted`, `status` enum `pending|delivered|set|failed`, `client_error`, timestamps). The PIN is encrypted with the server private key using the same AES-GCM helper `mdm_config_assets` uses, cleared to NULL the moment orbit fetches it, and a `pending` row older than 5 minutes is treated as expired (not delivered, not advertised). Resubmission replaces the row. The value is excluded from logs, `ctxerr` payloads and every response except the one orbit fetch. **Alternative rejected:** storing it in `host_disk_encryption_keys`, which is the recovery key's row and has different lifetime and decryptability semantics.

### D3. Eligibility is enforced server-side on submit and again on advertise

A submission is accepted only when: host platform is Windows and not a server; host is MDM-enrolled with Fleet; the host's fleet has disk encryption and `require_bitlocker_pin` on; `host_disks.tpm_pin_set` is false; the persisted fleetd capability says the agent can apply a PIN; the PIN is 6 to 20 ASCII digits. The orbit config advertises `bitlocker_pin_request_pending` only while the same conditions still hold, so turning the requirement off or moving the host to another fleet quietly drops the request. The device endpoint sits behind the existing `errorLimiter` middleware.

### D4. Two orbit endpoints, mirroring `scripts/request` and `scripts/result`

`POST /api/fleet/orbit/disk_encryption_pin/request` returns `{pin}` once and marks the row delivered. `POST /api/fleet/orbit/disk_encryption_pin` takes `{outcome: "set"|"failed", client_error}`. Both are registered under `oeWindowsMDM` like `disk_encryption_protection`. **Alternative rejected:** a single endpoint with a mode field; the request/result split is the established orbit convention and keeps the one response that carries the PIN isolated.

### D5. Server actions on outcome

On `set`: `SetOrUpdateHostDiskTpmPIN(host, true)`, delete the request row, create `ActivityTypeCreatedDiskEncryptionPIN{HostID, HostDisplayName}` with a nil user (rendered as "End user"), and `UpdateHostRefetchRequested(host, true)` so osquery confirms the protector list quickly. On `failed`: keep the row with `status=failed` and the sanitized `client_error` (truncated to 255 like `bitlocker_protection_error`) so the page can show it, and leave `tpm_pin_set` alone. The agent's report is treated as a claim; `tpm_pin_set_verify` remains the observation of record.

### D6. Gating on a new fleetd capability, persisted per enrollment

Orbit on Windows adds `CapabilityWindowsBitLockerPIN` (`windows_bitlocker_pin`) to `GetOrbitClientCapabilities`. The server persists it from the `X-Fleet-Capabilities` header on the orbit config request into a new `mdm_windows_enrollments.fleetd_bitlocker_pin_capable` column (pattern: `fleetd_sync_capable`), because the device and Fleet Desktop endpoints have no such header. The device host response gains `mdm.os_settings.disk_encryption.fleetd_can_set_pin`; when false the page renders the legacy instructions modal and Fleet Desktop gets no toast. **Alternative rejected:** comparing `host_orbit_info.version` semver on the server; Fleet's convention for agent feature negotiation is capabilities.

### D7. Agent apply sequence with verification and rollback

Inside the existing BitLocker receiver mutex (so it never races the encrypt or protection-restore paths), orbit: checks not Windows Server, volume `C:` fully encrypted, protection on, no type 4 or 6 protector; calls `ProtectKeyWithTPMAndPIN(nil, nil, pin)`; verifies a type 4 protector now exists via `GetKeyProtectors(4)`; deletes every type 1 protector by ID; verifies no type 1 remains. If anything after the add fails, it deletes the type 4 protector it just added so the volume is left exactly as found, then reports `failed`. `FVE_E_PROTECTOR_EXISTS` is reported as `failed` with "PIN already set" rather than treated as success, so the Fleet path can never be used to replace a PIN. The PIN is held in memory only for the call and zeroed afterwards; it never appears in logs. **Alternative rejected:** leaving both protectors and reporting failure; that silently leaves a PIN that is not enforced at boot, the exact blind spot of #52159 F1.

### D8. Fleet Desktop toast via PowerShell-driven WinRT, under a Fleet Desktop AUMID registered by orbit

Fleet Desktop (Windows only) renders a `<toast>` XML with the Figma copy and a `<action content="Create PIN" activationType="protocol" arguments="<My device URL>?create_pin=1"/>` button, and posts it through a hidden `powershell.exe -NoProfile -NonInteractive -ExecutionPolicy Bypass` script calling `ToastNotificationManager::CreateToastNotifier(<AUMID>)`. Protocol activation opens the default browser with no click handling in Fleet Desktop. Orbit (SYSTEM) registers the AUMID (`FleetDM.FleetDesktop`) once at startup under `HKLM\Software\Classes\AppUserModelId\FleetDM.FleetDesktop` with `DisplayName = "Fleet Desktop"` and `IconUri` pointing at the installed Fleet Desktop icon, idempotently. If the key is absent (registration failed), Fleet Desktop falls back to PowerShell's own AUMID so the toast still shows. The toast is posted at most once per Fleet Desktop process while `notifications.needs_bitlocker_pin` is true, carries a fixed `Tag`/`Group` so a re-post replaces it, is re-posted when Fleet Desktop observes a device-token rotation (the URL embeds the token, which rotates about hourly), and carries an `ExpirationTime` so an ignored toast leaves Action Center instead of opening a dead link. **Alternatives rejected:** `Shell_NotifyIcon` balloon (no button, needs a systray fork); pure-Go WinRT bindings (heavier dependency, kept as fallback if PowerShell is blocked); a `fleet-desktop:` protocol handler (adds single-instance plumbing for no v1 benefit).

### D9. My device modal behavior

Form with two `InputField type="password"` controls, helper text "Must be 6–20 digits. Keep it somewhere safe. This PIN isn't saved by Fleet or your IT team.", Save disabled until both fields are 6 to 20 digits and equal. Save posts the PIN, then the modal enters a waiting state and polls `GET /device/{token}` every 3 s reading `pin_request.status` for up to 90 s (covers one 30 s orbit poll plus WMI time with margin). `set` closes the modal, shows the "Successfully set PIN." toast, and the banner disappears because `action_required` is no longer `create_pin`. `failed` shows "Couldn't set PIN. {error}. Try again or contact your IT admin." and re-enables Save. Timeout shows the same failure copy with a generic reason. `?create_pin=1` opens the modal on load when `action_required === "create_pin"`. The legacy instructions modal is kept and rendered when `fleetd_can_set_pin` is false.

### D10. Banners

My device banner text becomes "Disk encryption: Create a BitLocker PIN to protect your data if this host is lost or stolen." with a **Create PIN** link (drops the "select Refetch" sentence, since success now clears it automatically). Host details (admin) banner adopts the same lead sentence but without a link the admin cannot act on: "Disk encryption: The end user needs to create a BitLocker PIN. Fleet Desktop prompts them at each login." (Figma dev note says update both; the link for admins is called out in Open Questions.)

## Risks / Trade-offs

- [Server briefly holds a user secret] → AES-GCM with the server private key, NULLed on delivery, 5 minute TTL, never logged or returned to users, one row per host. The server already holds the recovery password for the same volume.
- [Any local process can read the device token and submit a PIN, locking the device at boot] → only accepted when no PIN exists and the fleet requires one; the agent verifies the protector list; the activity log names the host and time; the escrowed recovery key still unlocks the volume. Same exposure exists for every device-token endpoint today.
- [Agent reports success but the PIN is not actually enforced] → success requires a type 4 protector present and no type 1 remaining, verified after the calls, with rollback otherwise. `tpm_pin_set_verify` re-confirms from osquery after the requested refetch.
- [Key rotation re-adds a TPM-only protector and negates the PIN (#52159 F1)] → land that fix first or in the same release; add an integration check that rotation on a host with a type 4 protector does not add type 1.
- [30 s orbit poll makes Save feel slow] → explicit waiting state with copy, 90 s ceiling, failure copy on timeout; WebSocket-driven config wake is a follow-up.
- [PowerShell Constrained Language Mode or AppLocker blocks the WinRT toast] → toast failure is logged and swallowed; the banner on My device still carries the flow. Pure-Go WinRT bindings are the fallback if this proves common.
- [Toast dropped because the AUMID is unregistered] → orbit registers it at startup; Fleet Desktop falls back to PowerShell's AUMID when the key is missing.
- [Stale token in a toast left in Action Center] → fixed Tag/Group re-post on token rotation plus `ExpirationTime`.
- [Concurrent encrypt or protection-restore work on the same volume] → the PIN path runs under the same receiver mutex and requires fully encrypted plus protection on.
- [Windows Server or non-MDM hosts] → excluded by the existing `is_server` and MDM-connected checks on both submit and advertise.
- [Controls tab lags the banner] → `tpm_pin_set` is set from the agent report so the banner clears immediately; the Verified status still waits for the next detail ingest, which the requested refetch shortens to seconds.
- [Azure test VMs cannot take pre-boot PIN input] → verify WMI calls and protector lists on `bl-fix-test`, verify the boot prompt on a physical machine or Hyper-V VM with a vTPM.

## Migration Plan

1. Ship the server change (migrations are additive; older fleetd never sees the new notification because the capability column defaults to false).
2. Ship fleetd with the capability, the receiver branch, and AUMID registration. Hosts pick it up through the normal update channel; the server flips `fleetd_bitlocker_pin_capable` on their next config poll.
3. Ship the frontend and Fleet Desktop changes in the same server and fleetd releases respectively; the frontend degrades to the legacy modal when `fleetd_can_set_pin` is false.
4. Rollback: reverting the server hides the new endpoints and notifications; fleetd ignores unknown fields, and an in-flight `pending` row simply expires. No data migration to undo.
5. Move docs PRs #50335 and #50463 from the 4.92.0 docs branch to the 4.93.0 docs branch (`push-reference-docs`).

## Open Questions

- Confirm with product that the Host details (admin) banner drops the **Create PIN** link (D10).
- Confirm "Fleet Desktop" as the toast header name (AUMID `DisplayName`); the Figma header row reads "Fleet: Action needed", which Windows draws from the AUMID, not from toast content. Victor approved "Fleet Desktop" on 2026-09-11.
- Should the request TTL be longer than 5 minutes for hosts on slow networks? 30 s poll makes 5 minutes generous; revisit with telemetry.
- Whether `IconUri` should reference the Fleet Desktop executable's embedded icon or a PNG the MSI drops next to it.
