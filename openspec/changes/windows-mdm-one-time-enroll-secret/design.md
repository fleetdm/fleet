## Context

Fleet ships a per-host, single-use enroll secret for Apple/macOS today (story #53063; PRs #53158, #53166, #53167), gated behind `auth.use_one_time_enroll_secrets`. Windows still hands every MDM-enrolled host the same global secret on the fleetd MSI command line.

The macOS design has a shape worth naming, because the parity work follows from it. The stored profile never contains a credential: `ensureFleetProfiles` writes the literal placeholder `$FLEET_HOST_SECRET_ENROLL_SECRET`, and the real value is minted lazily at command delivery, when nanomdm calls `ExpandHostSecrets`. Recovery from a wedged device falls out of the same property: an admin resends the Fleetd configuration profile, the reconciler re-enqueues the placeholder, delivery mints fresh, and orbit adopts it because it re-reads the profile on every start of its `--use-system-configuration` loop.

Windows has neither half of that. `getPendingMDMCmds` expands only server-scoped `$FLEET_SECRET_*`; the host-scoped `ExpandHostSecrets` hook is declared on the nanomdm storage interface and exists only on the Apple path. And orbit on Windows reads `secret.txt` exactly once, moves the value into Windows Credential Manager, and deletes the file — there is no re-read.

Three further constraints shape the design:

- **No host row at the moment of need.** `authBinarySecurityToken` returns an empty host UUID for both automatic (Entra, Autopilot) flows, and the enrollment is inserted unlinked with `host_uuid = ''`. Linkage happens later, via DevDetail SMBIOS serial, orbit's reverse link, or osquery ingest. But the fleetd install is enqueued *by MDM device ID* precisely so it works for unlinked enrollments. `mintHostOneTimeEnrollSecret` keys on `hosts.uuid` and returns `notFound("Host")`, so it cannot be reused unchanged.
- **No team on the enrollment.** `mdm_windows_enrollments` has no team column; team comes from the `hosts` row, or for MDM-first enrollments from `GetWindowsEnrollmentDefaultFleet`.
- **The MSI will not reinstall.** The install command targets a fixed product GUID, and the code says why: *"keeping the same GUID will prevent the MSI to be installed multiple times."*

## Goals / Non-Goals

**Goals:**
- Remove the fleet-wide blast radius: a secret recovered from one Windows device must not enroll any other host.
- Keep the plaintext secret out of `windows_mdm_commands.raw_command` and out of the MDM command-results API.
- Give an administrator a way to recover a Windows host whose secret is spent or was never applied, without touching the device.
- Reuse the macOS storage, consumption, and rejection-reporting machinery rather than building a parallel one.
- Match macOS resolution semantics exactly, so the two platforms do not drift.

**Non-Goals:**
- Fixing the underlying enrollment-matching weakness in security#7 (`matchHostDuringEnrollment` has no proof of possession; `VerifyEnrollSecret` is an unscoped global lookup). Complementary, independently shippable.
- Extending one-time issuance to Apple manual installers, Android, or `fleetctl package` output.
- Changing the two-plane consumption model or the second-plane window.
- Any administrator-facing surface for viewing or revoking outstanding one-time secrets. Re-enrollment and recovery are the revocation story.

## Decisions

### 1. Bind the secret to the Windows MDM enrollment, not to a `hosts` row

The case that needs the fleetd install is exactly the case with no host row. Binding to `mdm_windows_enrollments` (by MDM device ID, with MDM hardware ID as the stable identity) lets minting succeed at the only moment it is useful.

*Alternatives considered.* **Wait for linkage before minting** — would delay the install until DevDetail returns a serial, which can take a full poll interval and in the relaxed 480-minute schedule is hours; it also breaks the existing design where the install is deliberately keyed on device ID. **Create a placeholder `hosts` row at MDM enroll** — invents a host that may never materialize and collides with the anti-takeover guard in `MDMWindowsConflictingEnrollmentHardwareID`.

### 2. Extend `host_one_time_enroll_secrets` with a nullable enrollment reference

Keep one table, one consumption path, one cleanup cron, one rejection-reporting path. Apple rows keep `host_id` populated and the new column NULL; Windows rows start with the enrollment reference and gain `host_id` when linkage happens.

*Alternatives considered.* **A separate Windows table** — duplicates consumption, cleanup, and the enroll-path lookup, and the enroll path is the hottest path Fleet has; two lookups instead of one is the wrong trade. **A column on `mdm_windows_enrollments`** — that table already holds per-enrollment material (`credentials_hash`), but it has no room for the consumption state (`consumed_at`, per-plane timestamps) without duplicating the macOS columns.

### 3. Relax the identity binding for Windows, and populate it on first use

`MatchesHost` compares platform, hardware UUID, and hardware serial. Fleet knows none of them reliably when it enqueues the Windows install; the serial only arrives later over DevDetail. Windows rows therefore mint with the identifiers unset (or with only what the enrollment carries, such as a ZTDID or device-reported serial) and record the presented identifiers on first consumption, binding the secret to the first device that uses it. Replay from a second machine is then rejected by the recorded binding, which is the property the story actually needs.

*Alternatives considered.* **Require full identifiers at mint** — not available, would gate the install on DevDetail. **Skip the binding check on Windows entirely** — would leave a single-use secret replayable within its unconsumed window by anyone who scraped it, which is most of the value gone.

### 4. Add host-scoped expansion to the Windows delivery path

Teach `getPendingMDMCmds` to expand `$FLEET_HOST_SECRET_*` per enrollment and store the placeholder in `raw_command`. This is the change that closes the command-results API disclosure, and it is the mechanism that makes re-delivery mint naturally rather than requiring a separate re-mint call.

*Alternatives considered.* **Mint eagerly at enqueue and store the value** — smaller diff, but leaves a live credential in `windows_mdm_commands` readable through the API. The value is now single-use and device-bound so the exposure is much smaller, but it is avoidable, and keeping the two platforms structurally identical is worth more than the saved effort.

### 5. Recovery rides a re-deliverable carrier plus an agent re-read

Because the MSI will not reinstall, the recovery path cannot be the fleetd install command. It needs a carrier Fleet can rewrite independently of MSI state, and an agent that consults it. Concretely: a Fleet-managed Windows configuration profile that writes the secret to a known registry location, and orbit reading that location on each start and refreshing its stored secret — the direct analogue of `--use-system-configuration` on macOS. Fleet's Windows profiles are raw SyncML with arbitrary LocURIs, so a registry-writing CSP is expressible.

*Alternatives considered.* **Put the secret in the MSI's service `Environment` REG_MULTI_SZ** (which already carries the per-enrollment `ORBIT_EUA_TOKEN`, and which orbit would read with zero agent changes via `ORBIT_ENROLL_SECRET`) — attractive and cheap, but it still rides the MSI, so it does nothing for recovery; it is also world-readable by default where `secret.txt` is ACL'd to SYSTEM and Administrators. **Vary the MSI product GUID to force reinstall** — turns every recovery into a full reinstall and risks repeated install churn. **Require a manual device-side repair** — that is the status quo we are trying to remove.

### 6. Keep orbit the single source of the secret for both planes

macOS relies on this quietly: the reuse predicate is `consumed_at IS NULL`, and `consumed_at` is stamped on first use by *either* plane, so once orbit enrolls, the next delivery mints a fresh secret even though osquery may still be inside its second-plane window. That is safe only because osquery never reads the profile — orbit hands osqueryd the secret it already holds. Windows must preserve that: the registry carrier feeds orbit, and orbit continues to pass the secret to osqueryd. If osquery ever sourced the secret independently from a location MDM can rewrite, a re-delivery landing between the two enrollments would hand osquery a secret whose osquery slot belongs to a different row.

### 7. Reuse the existing feature flag

`auth.use_one_time_enroll_secrets` already gates the macOS behavior, is Premium-only, defaults off, and is force-disabled on license downgrade. Windows reuses it rather than adding a second key. The macOS documentation states a hard prerequisite (every Mac must run fleetd installed by Fleet MDM); the Windows equivalent needs stating before one flag governs both platforms.

### 8. Ship in three phases

Phase 1: mint per enrollment on the existing MSI carrier. Removes the fleet-wide blast radius, which is the substance of the issue. Phase 2: host-scoped expansion, closing the at-rest and API disclosure. Phase 3: the recovery channel. **Single-use enforcement stays off or lenient until phase 3 lands**, because only then does a wedged host have a way back.

## Risks / Trade-offs

- **[Permanent wedge from single-use semantics]** A host whose MSI installed but never enrolled looks fleetd-less forever: `isFleetdPresentOnDevice` requires both an orbit version and a recent check-in, so Fleet re-enqueues the install on every session alert, the device ignores it because the GUID is already installed, and `secret.txt` was deleted after first read. This failure is *created by* making the secret single-use; it cannot happen today. → Do not enforce single-use until the phase 3 recovery channel exists. Make this the first thing QA proves, not the last.
- **[Registry disclosure]** A secret in the registry is world-readable by default, unlike `secret.txt`. → ACL the key to SYSTEM and Administrators, or make an explicit decision that a single-use, device-bound secret does not warrant it. Do not leave this implicit.
- **[Minting churn from repeated session alerts]** There is no dedupe on the fleetd install; every session-start alert re-enqueues while fleetd looks absent. → The `consumed_at IS NULL` reuse branch is what bounds this; confirm it holds under the 1-minute fast poll, and that a repeatedly failing install does not mint per alert.
- **[Team resolution differs from macOS]** Team comes from `GetWindowsEnrollmentDefaultFleet`, not from a `hosts` row, and `maybeAssignWindowsEnrollmentDefaultFleet` applies only under specific conditions. → A secret minted before the default team is resolvable could carry the wrong team; resolve team at mint from the enrollment, and verify the non-global-team case explicitly.
- **[Primary database read on every enroll]** `GetHostOneTimeEnrollSecret` deliberately reads the primary, because secrets are minted moments before they are presented. Windows inherits this on the hottest path Fleet has. → Confirm the lookup adds no per-enroll cost for enrollments using an ordinary shared secret.
- **[Legitimate re-enroll after node key loss]** Orbit that loses its node key re-enrolls with the secret it still holds. Under single-use that secret is spent. → Decide explicitly whether the secret stays valid for its bound host on the same plane, or whether recovery is the intended path; hosts going permanently silent is the failure to avoid.
- **[Hosts holding the old global secret]** Every host enrolled before the change holds a copy of the fleet-wide secret, which stays live until rotated. → Ship with guidance to rotate the global secret once the fleet has re-enrolled.

## Migration Plan

1. Land phases 1 and 2 with `auth.use_one_time_enroll_secrets` off. No behavior change for any existing deployment.
2. Verify on a real Windows host through Windows MDM end to end: enrollment, install, orbit enroll, osquery enroll, re-enrollment.
3. Land phase 3 (recovery channel). Verify the wedge scenario specifically: consume or invalidate the secret *before* fleetd enrolls, then recover without device-side intervention.
4. Enable the flag on dogfood. Watch for hosts enrolled in MDM but never reporting, which is what a too-aggressive secret lifetime looks like.
5. Document that the secret delivered to Windows MDM hosts is single-use and re-issued, and publish the global-secret rotation guidance.

**Rollback.** Disable `auth.use_one_time_enroll_secrets`. Windows returns to the global secret on the MSI command line immediately. Rows in `host_one_time_enroll_secrets` become inert rather than harmful, since the enroll path stops consulting them. Phase 2 needs care here: a stored placeholder is only meaningful while expansion is active, so the rollback path must re-enqueue install commands built from the global secret rather than leaving placeholders queued.

## Open Questions

- **Expiry for an unconsumed secret.** macOS chose *no* TTL; the security issue and the local draft both assume one. Recommendation: match macOS. A TTL adds a failure class (expired before a slow MSI download finished) that phase 3 recovery would then have to absorb, and the self-heal should be proven before adding a second clock.
- **Where the recovery carrier lives.** A dedicated Fleet-managed Windows profile is the cleanest analogue of the Fleetd configuration profile, but it introduces a Fleet-managed Windows profile where none exists today. Worth confirming against how Fleet-managed Windows profiles are reconciled.
- **Whether recovery is a new endpoint or reuses `resendHostMDMProfileEndpoint`.** The existing endpoint is registered under `mdmAnyMW` and already has the admin-only and resend-while-verifying carve-outs macOS needed. Reusing it would keep the two platforms' operator experience identical.
- **Whether `host_enrollment_rejected` activities are wanted at Windows volume**, or whether the existing 12-hour per-host-per-reason rate limit is sufficient.
