## Context

Fleet creates a hidden `_fleetadmin` break-glass admin on macOS during ADE enrollment (#37141): the server generates a password, escrows it in `host_managed_local_account_passwords`, and sends the Apple MDM `AccountConfiguration` command with a salted hash. Story #43488 extends the feature to Windows hosts enrolling through the out-of-box experience. Windows MDM cannot replicate the macOS approach, so the mechanics differ while the product behavior stays parallel. A full engineering spec with verified code anchors lives at `engineering-spec.md` in this change directory; sub-issues are #48720 to #48724.

## Goals / Non-Goals

**Goals:**
- Hidden `_fleetadmin` local admin with a unique escrowed password on Windows hosts that enroll via Autopilot ESP or Entra-join-during-OOBE
- Per-platform enable toggle (`managed_local_account_settings`) across UI, REST API, and GitOps, with the legacy `setup_experience` fields as compatible aliases
- Password retrieval through the existing Host details flow, premium-gated, permission-checked

**Non-Goals:**
- Password rotation for Windows (story #43489, 4.91.0); this design only keeps the door open
- Manually enrolled Windows hosts or hosts enrolled before the feature is turned on
- End-user account type control on Windows (macOS-only concept)
- Custom account name or password policy (future fields on `managed_local_account_settings`)

## Decisions

1. **fleetd creates the account, not the Accounts CSP.** The CSP sets a password only at creation (no rotation path for #43489), no CSP hides an account from the sign-in screen, and SyncML bodies leak plaintext into MDM command logs. fleetd runs as SYSTEM during the ESP and can do all of it: netapi32 (`NetUserAdd`/`NetUserSetInfo`), Administrators membership by well-known SID `S-1-5-32-544` (locale independent), and the `Winlogon\SpecialAccounts\UserList` registry value for hiding. Alternative considered: Accounts CSP plus a separate hide profile; rejected for the rotation dead end.
2. **Password is generated on the device and escrowed up.** fleetd generates 32 chars from `crypto/rand` and POSTs to a new orbit endpoint (`POST /api/fleet/orbit/managed_local_account`), following the LUKS escrow pattern (`EscrowLUKSData`). Alternative: server-generated password pushed down in the orbit config response (closer to macOS); rejected because plaintext would ride in config responses that are easier to log and cache.
3. **Reuse `host_managed_local_account_passwords` with one migration** (`command_uuid` nullable). The table is keyed by `host_uuid`, its read paths are platform-agnostic, and the 4.91 rotation reuses the pending columns. Windows rows: `status='verified'` on escrow, `command_uuid`/`account_uuid`/`auto_rotate_at` NULL, which structurally excludes them from the macOS rotation cron (its query requires `account_uuid IS NOT NULL` and a set `auto_rotate_at`). Alternative: a new Windows table; rejected as duplicate encryption/read plumbing.
4. **Legacy `setup_experience` fields stay the canonical macOS storage; the new fields are aliases.** The worker, EE PATCH, and team-spec paths all read/write `MacOSSetup.EnableManagedLocalAccount`; moving storage touches every one for zero user benefit. Writes on either surface converge; conflicting values in one payload return 422. Windows has no legacy alias: `WindowsSettings.ManagedLocalAccountSettings` is the storage. Alternative: migrate storage to the new shape; rejected for churn and back-compat risk.
5. **The notification is never gated on an existing escrow row; the device owns idempotency.** A wiped and re-enrolled host still has its old row but no account. fleetd resets the password if the account exists and writes a local marker file (timestamp only, never the password) only after a confirmed escrow; the marker stops reprocessing, and a wiped disk wipes the marker exactly when re-creation is wanted. Alternative: server-side dedupe; rejected because the server cannot distinguish wipe-and-re-enroll from a crash retry.
6. **All OOBE enrollments qualify**, gated on `mdm_windows_enrollments.awaiting_configuration` Pending or Active, the same state the existing Windows setup-experience notification uses. Plus three more gates: setting enabled for the host's team, fleetd capability `windows_managed_local_account` (advertised only on `runtime.GOOS == "windows"`), premium license.
7. **The Windows toggle saves via config/teams PATCH**, not `PATCH /setup_experience`. The documented API (#47915) puts the new object under `mdm.windows_settings`; the setup-experience endpoint has no Windows field. The macOS tab keeps its existing save path.

## Risks / Trade-offs

- [Team GitOps silently drops the Windows setting] `editTeamFromSpec` copies `WindowsSettings` selectively (only `CustomSettings` today, `ee/server/service/teams.go:1882-1883`) → explicit copy of `ManagedLocalAccountSettings` plus a regression test in the CoS.
- [No premium check exists on the config PATCH path] The macOS gate lives in the CE stub of `PATCH /setup_experience` only → add an explicit `ErrMissingLicense` check in `validateMDM` for the new fields.
- [Any local admin can reset `_fleetadmin`'s password, diverging from escrow] Accepted, same posture as macOS (same gap exists there per QA); rotation in 4.91 is the mitigation.
- [Crash between account creation and escrow] The account briefly holds a password nobody knows → next config fetch during OOBE resets and re-escrows; converges within one poll cycle. Residual risk: a crash in the final seconds of OOBE leaves an account without escrow; visible as a missing row, recoverable by wipe or by #43489 rotation.
- [Docs drift] The REST reference documents a `POST /api/v1/fleet/managed_local_account` endpoint that exists nowhere in code, and #48110 has a list-vs-object typo and wrong deprecated-field scope → docs follow-ups tracked in #48724.

## Migration Plan

- One additive-safe migration (widen `command_uuid` to nullable); no data backfill; forward-only, standard Fleet migration flow.
- Feature defaults off on all platforms. Server ships first; fleetd with the receiver ships on the fleetd release train (old fleetd never receives the notification because it lacks the capability).
- Rollback: disable the setting (stops new account creation); already-created accounts and escrowed passwords remain, matching macOS behavior.

## Open Questions

- Parent story t-shirt size is unset (product to set; does not block).
- Phantom `POST /api/v1/fleet/managed_local_account` endpoint: remove from docs or implement (Mel's call, tracked in #48724).
- Windows tab visibility when Windows MDM is off: recommend hiding, matching `PlatformTabs`; confirm with product during frontend review.
