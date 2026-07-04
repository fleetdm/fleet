## Why

IT admins need a break-glass local admin account on Windows hosts for troubleshooting, matching what Fleet already provides on macOS (#37141). Fleet 4.90.0 commits to this for Windows hosts that enroll through the out-of-box experience (parent story #43488, customers easterwood and mozartia).

## What Changes

- New per-platform config surface: `managed_local_account_settings` object under `mdm.apple_settings` and `mdm.windows_settings` (global config and teams), with the existing `setup_experience.enable_managed_local_account` and `setup_experience.end_user_local_account_type` fields becoming deprecated aliases of the Apple values.
- New orbit notification (`create_windows_managed_local_account`) sent during Windows OOBE enrollment (Autopilot ESP or Entra-join-during-OOBE), gated on the setting, a new fleetd capability, and a premium license.
- New orbit endpoint `POST /api/fleet/orbit/managed_local_account`: fleetd escrows a device-generated password, stored encrypted in the existing `host_managed_local_account_passwords` table (one migration: `command_uuid` becomes nullable).
- fleetd (Windows only) creates the hidden `_fleetadmin` local admin account: netapi32 account creation, Administrators membership by SID, sign-in screen hiding via the `SpecialAccounts\UserList` registry value, crash-safe escrow-then-marker flow.
- `GET /api/v1/fleet/hosts/{id}/managed_account_password` opens to Windows hosts; password rotation stays macOS-only (Windows rotation is story #43489).
- Controls > Setup experience > Users page gains a macOS/Windows sub nav; Windows tab has only the Managed > "Create hidden admin" toggle. Host details "Show managed account" action and modal work for Windows hosts, without rotation UI.
- `fleetctl generate-gitops` emits the new nested fields instead of the shared setup-experience TODO placeholder.

No breaking changes: deprecated fields keep working as aliases.

## Capabilities

### New Capabilities

- `managed-local-account-config`: enabling and disabling the managed local account per platform through the UI, REST API, and GitOps, including deprecated-field aliasing, premium gating, and enable/disable activities.
- `windows-managed-local-account-provisioning`: the server-to-fleetd flow that creates the hidden admin account on Windows during OOBE enrollment and escrows its password (notification gates, device-side creation, idempotency, escrow storage).
- `windows-managed-account-password-access`: retrieving a Windows host's managed account password through the API and the Host details UI, including permissions and the no-rotation constraints for Windows.

### Modified Capabilities

None. `openspec/specs/` has no established capabilities yet; this change introduces its capabilities as new specs.

## Impact

- Backend: `server/fleet/app.go` (settings structs, validation, clone), `server/service/appconfig.go`, `ee/server/service/teams.go` and `mdm.go` (aliasing across all five write paths), `server/service/orbit.go` and `handler.go` (notification, escrow endpoint), `ee/server/service/hosts.go` and `server/service/hosts.go` (password access), `server/fleet/capabilities.go`, one MySQL migration.
- Agent: new `orbit/pkg/managedaccount/` package registered in `orbit/cmd/orbit/orbit.go` (ships on the fleetd release train).
- Frontend: `frontend/pages/ManageControlsPage/SetupExperience/cards/Users/`, `frontend/pages/hosts/details/HostDetailsPage/` (actions dropdown, ManagedAccountModal).
- CLI: `cmd/fleetctl/fleetctl/generate_gitops.go`, `pkg/spec/` GitOps parsing.
- Docs: contributor API reference, YAML and REST reference fixes, guide PR #47925.
- Premium only; no new activity types; no load profile change.

Full engineering spec with verified code anchors: `engineering-spec.md` in this change directory. Sub-issues #48720 to #48724.
