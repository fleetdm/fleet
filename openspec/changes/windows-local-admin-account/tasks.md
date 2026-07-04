## 1. Config surface and GitOps (#48720)

- [ ] 1.1 Add `ManagedLocalAccountSettings` type; embed in `MacOSSettings` (plus `EndUserLocalAccountType`) and `WindowsSettings` in `server/fleet/app.go`; update `ToMap`/`FromMap` and `AppConfig.Clone`
- [ ] 1.2 Wire aliasing between the deprecated `setup_experience` fields and the Apple values across all write paths: `ModifyAppConfig`, `ModifyTeam`, `editTeamFromSpec` (including the selective `WindowsSettings` copy fix), `updateTeamMDMAppleSetup`, and the setup-experience PATCH; 422 on conflicting values in one payload
- [ ] 1.3 Add validation (Apple account type rules, no Windows account type) and the explicit premium check in `validateMDM` for the new fields
- [ ] 1.4 Fire `enabled/disabled_managed_local_account` activities on Windows toggle changes from config PATCH and team apply paths
- [ ] 1.5 GitOps: accept new nested keys on input; make `fleetctl generate-gitops` emit them and drop the managed-local-account conditions from the shared setup-experience TODO placeholder
- [ ] 1.6 Tests: struct/clone/validation units, appconfig and teams service tests, team-scoped gitops persistence regression, generate/apply round trip

## 2. Server orchestration and escrow (#48721)

- [ ] 2.1 Migration: make `host_managed_local_account_passwords.command_uuid` nullable; regenerate `schema.sql`
- [ ] 2.2 Add `CapabilityWindowsManagedLocalAccount` and advertise it from `GetOrbitClientCapabilities` under the `runtime.GOOS == "windows"` guard
- [ ] 2.3 Add `CreateWindowsManagedLocalAccount` notification in `ReadOrbitConfig` with the four gates (OOBE awaiting-configuration, team setting, capability, premium)
- [ ] 2.4 Add `POST /api/fleet/orbit/managed_local_account` endpoint, `EscrowWindowsManagedLocalAccountPassword` service method, and `SaveHostManagedLocalAccountFromEscrow` datastore method (status verified, NULL command_uuid); run `make generate-mock`
- [ ] 2.5 Open `GetHostManagedAccountPassword` to Windows without arming auto-rotate; keep rotate endpoint macOS-only with a Windows-specific message; add the `case "windows":` host-detail population gated on Windows MDM
- [ ] 2.6 Tests: migration, notification gate table test, escrow endpoint (store/replace/reject), cron exclusion of Windows-shaped rows, end-to-end enrollment-to-retrieval integration test

## 3. fleetd Windows receiver (#48723)

- [ ] 3.1 Create `orbit/pkg/managedaccount/` (windows impl plus stub) and register the receiver in `orbit/cmd/orbit/orbit.go`
- [ ] 3.2 Implement the create-and-escrow flow: password generation, `NetUserAdd`/`NetUserSetInfo`, Administrators by SID, `SpecialAccounts\UserList` registry hide, escrow POST via new `OrbitClient.SendManagedLocalAccountPassword`, marker file after confirmed escrow, `client_error` reporting
- [ ] 3.3 Move or re-export `fleet.ManagedLocalAccountUsername` from a platform-neutral location
- [ ] 3.4 Unit tests (generator classes, no-op guards, marker ordering with mock client); manual Windows VM verification steps documented in the PR; orbit changelog entry; fleetd release checklist

## 4. Frontend (#48722)

- [ ] 4.1 Restructure the Users card: End user authentication above a new macOS/Windows sub nav (follow `PlatformTabs`), platform icons, Figma copy updates
- [ ] 4.2 Windows tab with Managed > "Create hidden admin" only; save via config/teams PATCH with `mdm.windows_settings.managed_local_account_settings`; keep macOS save path; tier and GitOps-mode gating
- [ ] 4.3 Enable "Show managed account" for Windows in `canShowManagedAccount` (skip the ADE check, use the status fallback); modal with `canRotatePassword={false}` for Windows; pending tooltips
- [ ] 4.4 Tests: `canShowManagedAccount` Windows cases, Users form save paths; `yarn test` and `yarn lint` clean

## 5. Documentation and QA (#48724)

- [ ] 5.1 Document `POST /api/fleet/orbit/managed_local_account` in the contributor API reference
- [ ] 5.2 Docs fixes: list-vs-object YAML example, deprecated field scope (macOS-only aliasing), phantom `POST /api/v1/fleet/managed_local_account` endpoint resolution; confirm and merge guide PR #47925
- [ ] 5.3 Run the ten QA scenarios on real Autopilot and Entra-OOBE enrollments; record results and complete the parent story test plan and Engineering checkboxes
