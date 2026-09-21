## Why

Every Windows host that enrolls through Fleet MDM receives the same fleet-wide enroll secret. `enqueueInstallFleetdCommand` reads `GetEnrollSecrets(ctx, nil)`, takes `secrets[0].Secret`, and interpolates it verbatim into the fleetd MSI command line as `FLEET_SECRET`. The value never expires and is never consumed, it is stored in plaintext in `windows_mdm_commands.raw_command` and returned by the MDM command-results API, and it comes to rest on the device at `[ORBITROOT]secret.txt`. One local administrator, one support bundle, or one API reader with MDM access yields a credential that works against every host in the fleet, from the internet, forever. This is the Windows half of [fleetdm/security#68](https://github.com/fleetdm/security/issues/68), scoped down from security#7.

Apple/macOS already shipped the fix (story #53063; PRs #53158, #53166, #53167). Windows is the remaining half, and it is the weaker one: TPM-backed host identity, the complementary proof-of-possession mechanism, is Linux-only today, so Windows has no fallback if this path stays weak.

## What Changes

- `enqueueInstallFleetdCommand` mints a fresh per-enrollment secret instead of reading the global one. The secret is bound to the Windows MDM enrollment and carries that enrollment's team.
- Minting reuses the existing `host_one_time_enroll_secrets` storage and the existing consumption path, extended to bind to a Windows MDM enrollment rather than to a `hosts` row. This is required because the exact case that needs the fleetd install is the one where no host row exists yet and `mdm_windows_enrollments.host_uuid` is still `''`.
- Resolution semantics match macOS exactly: an existing secret with `consumed_at IS NULL` resolves to itself, and a new secret is minted only once the old one is consumed. Re-delivery stays idempotent while a secret is live.
- The Windows MDM delivery path gains host-scoped secret expansion (`$FLEET_HOST_SECRET_*`), which `getPendingMDMCmds` does not do today. The stored SyncML holds a placeholder, so the plaintext secret stops being at rest in `windows_mdm_commands` and stops being readable through the command-results API.
- A recovery channel is added so an administrator can re-deliver a fresh secret to a host whose secret was consumed or lost, **without** reinstalling the MSI. This has no macOS analogue in mechanism, only in outcome, and is required on Windows for a reason set out in `design.md`: the fleetd install command uses a fixed MSI GUID, so re-sending it to a device that already has fleetd installed is a device-side no-op and the new secret never lands.
- Consumption keeps the existing two-plane model (orbit and osquery, each usable once, second plane within `HostOneTimeEnrollSecretSecondPlaneWindow`) unchanged.
- Gated by the existing `auth.use_one_time_enroll_secrets` config key (Premium, default off). No new flag. **BREAKING** when that flag is enabled: a Windows host that still holds only the global secret and cannot re-enroll through MDM will not be able to enroll fleetd once the shared-secret path is closed for MDM-managed hosts. Behind a default-off flag, so not breaking on upgrade.

## Capabilities

### New Capabilities

- `windows-one-time-enroll-secret`: Minting a per-enrollment, single-use enroll secret for Windows MDM hosts; binding it to the MDM enrollment and team; consuming it across the orbit and osquery planes; re-issuing it once consumed; and keeping the plaintext value out of stored MDM commands.
- `windows-enroll-secret-recovery`: Administrator-triggered re-delivery of a fresh enroll secret to a Windows host whose secret was consumed, lost, or never applied, through a channel that does not depend on reinstalling the fleetd MSI.

### Modified Capabilities

<!-- None. openspec/specs/ is empty; there are no accepted specs whose requirements change. -->

## Impact

**Server — minting and delivery**
- `server/service/microsoft_mdm.go`: `enqueueInstallFleetdCommand` (:1507) stops reading the global secret; `getPendingMDMCmds` (:2073) gains host-scoped expansion; `processNewSessionAlert` (:1611) and `isFleetdPresentOnDevice` (:1448) govern re-enqueue frequency and therefore how often minting is reached.
- `server/datastore/mysql/host_one_time_enroll_secrets.go`: `mintHostOneTimeEnrollSecret` (:103) currently keys on `hosts.uuid` and returns `notFound("Host")` when there is no host row; needs an enrollment-bound path.
- `server/fleet/host_one_time_enroll_secrets.go`: `MatchesHost` (:45) compares platform, hardware UUID, and serial, none of which Fleet reliably knows at Windows MDM enroll time.

**Database**
- `host_one_time_enroll_secrets` (migration `20260901200000`) needs an enrollment reference, or the binding needs to live on `mdm_windows_enrollments`, which already carries per-enrollment material in `credentials_hash`.

**Agent**
- `orbit/pkg/packaging/windows_templates.go`: the MSI writes orbit's service environment to the registry as REG_MULTI_SZ (:104) and already carries a per-enrollment server-minted token precedent in `ORBIT_EUA_TOKEN` (:124).
- `orbit/cmd/orbit/orbit.go`: secret resolution (:470) reads `secret.txt` once, moves the value into Windows Credential Manager, and deletes the file. The recovery capability requires orbit to consult a re-deliverable source on each start, which is the Windows analogue of `--use-system-configuration` on macOS.

**Security posture**
- Removes the fleet-wide blast radius on this path and closes the command-results API disclosure. Does not address the underlying enrollment-matching weakness in security#7 (`matchHostDuringEnrollment`, `VerifyEnrollSecret`), which remains complementary and independently shippable.

**Not affected**
- No UI, CLI, REST API, GitOps, or activity surface for the secrets themselves. They are machine-issued and must not appear in Settings > Enroll secrets, `fleetctl get enroll_secrets`, or GitOps output. The `$FLEET_HOST_SECRET_*` prefix is already rejected on Windows profile upload by `validateWindowsProfileFleetVariables`, and that stays as is.
