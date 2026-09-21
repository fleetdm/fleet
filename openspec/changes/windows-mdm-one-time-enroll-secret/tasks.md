## 1. Close blocking decisions

- [x] 1.1 Expiry for an unconsumed Windows secret: **decided, no TTL, matching macOS**. Recorded in design.md decision 8 and in security#68.
- [ ] 1.2 Decide whether recovery reuses `resendHostMDMProfileEndpoint` (already `mdmAnyMW`, already admin-only with the resend-while-verifying carve-out) or gets a new endpoint.
- [ ] 1.3 Decide the intended behavior when orbit loses its node key and re-enrolls with a spent secret: secret stays valid for its bound host on the same plane, or recovery is the path. Hosts going permanently silent is the outcome to avoid.
- [ ] 1.4 Decide the tier. The existing flag is Premium-only and force-disabled without a Premium license (`cmd/fleet/serve.go:355`), but the story says "Fleet Free and Fleet Premium… not a tiered feature". Reusing the flag inherits Premium-only.
- [ ] 1.5 Set the minimum fleetd version required for recovery to work, and decide whether to surface hosts below that floor using the orbit version the server already has.

## 2. Schema and minting (phase 1)

- [ ] 2.1 Write a migration adding a nullable Windows MDM enrollment reference to `host_one_time_enroll_secrets`, leaving `host_id` nullable so Apple rows are unaffected. Follow `.claude/rules/fleet-database.md`: `IF NOT EXISTS` guards, no-op `Down_`, migration test only if it changes data.
- [ ] 2.2 Relax the identity columns so a Windows row can be minted with `platform`, `hardware_uuid`, and `hardware_serial` unset or partially set from the enrollment.
- [ ] 2.3 Add an enrollment-bound minting function alongside `mintHostOneTimeEnrollSecret`, keyed on the MDM enrollment rather than `hosts.uuid`, sharing the reuse-vs-mint branch.
- [ ] 2.4 Implement the reuse predicate exactly as macOS does: resolve an existing row with `consumed_at IS NULL` to the same value; mint only when there is none.
- [ ] 2.5 Serialize concurrent mints for the same enrollment, using a lock on a row that exists at mint time (the enrollment row), mirroring the `FOR UPDATE` reasoning in `mintHostOneTimeEnrollSecret`.
- [ ] 2.6 Resolve the team at mint from the Windows MDM enrollment, via `GetWindowsEnrollmentDefaultFleet`, and store it on the row.
- [ ] 2.7 Extend `CleanupHostOneTimeEnrollSecrets` so Windows rows with no host linkage are swept on the same rules as orphaned Apple rows.

## 3. Wire minting into the fleetd install (phase 1)

- [ ] 3.1 Change `enqueueInstallFleetdCommand` (`server/service/microsoft_mdm.go:1507`) to obtain a per-enrollment secret instead of `GetEnrollSecrets(ctx, nil)[0].Secret`, gated on `auth.use_one_time_enroll_secrets`.
- [ ] 3.2 Keep the existing behavior byte-for-byte when the flag is off.
- [ ] 3.3 Handle the no-global-secret case: with the flag on, a Windows MDM enrollment must still issue a one-time secret rather than silently sending an empty `FLEET_SECRET`.
- [ ] 3.4 Verify that repeated session-start alerts (`processNewSessionAlert`) reuse the live secret rather than minting per alert, under the 1-minute fast poll.

## 4. Consumption and binding (phase 1)

- [ ] 4.1 Record the presented identifiers on first consumption for Windows rows, binding the secret to the first device that uses it.
- [ ] 4.2 Confirm the existing two-plane consumption and second-plane window apply unchanged to Windows rows, with no new code path.
- [ ] 4.3 Confirm rejection reasons (`one_time_secret_spent`, `one_time_secret_identifier_mismatch`) and the `host_enrollment_rejected` activity fire for Windows, with the existing 12-hour rate limit.
- [ ] 4.4 Confirm the enroll path adds no extra query for enrollments presenting an ordinary shared secret.

## 5. Host-scoped expansion on Windows delivery (phase 2)

- [ ] 5.1 Add `$FLEET_HOST_SECRET_*` expansion to `getPendingMDMCmds` (`server/service/microsoft_mdm.go:2073`), resolving per enrollment, alongside the existing `ExpandEmbeddedSecrets`.
- [ ] 5.2 Store the placeholder rather than the secret in `windows_mdm_commands.raw_command` for the fleetd install command.
- [ ] 5.3 Verify `GetMDMWindowsCommandResults` no longer discloses the secret in `payload`.
- [ ] 5.4 Confirm `validateWindowsProfileFleetVariables` still rejects user-supplied profiles carrying the placeholder, on both the single-upload and batch/GitOps paths.
- [ ] 5.5 Ensure the rollback path re-enqueues install commands built from the global secret rather than leaving unexpandable placeholders queued when the flag is turned off.

## 6. Recovery carrier, server side (phase 3)

- [ ] 6.1 Define the registry location the secret is delivered to, and the SyncML/CSP that writes it.
- [ ] 6.2 Add the Fleet-managed Windows profile (or equivalent command) that carries the placeholder and is re-deliverable independently of MSI install state.
- [ ] 6.3 Wire the admin-triggered recovery action per decision 1.2, keeping it admin-only and refusing the end-user My device path.
- [ ] 6.4 Ensure triggering recovery mints a fresh secret when the previous is consumed, and re-delivers the same value when it is not.
- [ ] 6.5 ACL the registry location to SYSTEM and Administrators, or record the explicit decision not to.

## 7. Recovery, agent side (phase 3)

- [ ] 7.1 Have orbit on Windows read the registry location on startup, as the analogue of the macOS `--use-system-configuration` loop. The read MUST be additive: fall back to `secret.txt` and the keystore when the registry value is absent, so new fleetd against an old server is a no-op.
- [ ] 7.2 Adopt a newer secret found there, replacing the stored credential, without re-enrolling an already-enrolled host that has no new secret waiting.
- [ ] 7.3 Keep orbit the single source of the secret for both planes: orbit continues to pass the secret to osqueryd rather than osquery sourcing it independently.
- [ ] 7.4 Confirm the existing mutual exclusion between `enroll-secret` and `enroll-secret-path` is not violated by the new path.

## 8. Tests

- [ ] 8.1 Fresh Windows MDM enrollment gets a secret that is not the global one, and the host appears in Fleet.
- [ ] 8.2 Secret is marked used after first orbit enroll; presenting it again is rejected.
- [ ] 8.3 Replay from a second machine with a different UUID is refused.
- [ ] 8.4 Replay from a second machine claiming the first host's UUID and serial is refused.
- [ ] 8.5 Un-enroll and re-enroll: the new install carries a different secret and the previous unused secret no longer works.
- [ ] 8.6 Two Windows hosts enrolling simultaneously get distinct secrets and neither consumes the other's.
- [ ] 8.7 Windows MDM enrollment into a non-global team: the minted secret carries that team and the host lands in it.
- [ ] 8.8 Minted secrets do not appear in `GET /api/v1/fleet/spec/enroll_secret`, `fleetctl get enroll_secrets`, `fleetctl generate-gitops`, or the enroll secrets UI.
- [ ] 8.9 Server with no global enroll secret configured still issues a one-time secret, or fails with a clear message.
- [ ] 8.10 **Wedge scenario, the one to prove hardest**: MSI installs, the secret is consumed or invalidated *before* fleetd enrolls, and the host recovers without device-side intervention and without MSI reinstall.
- [ ] 8.11 Repeatedly failing install does not mint an unbounded number of live secrets and the failure is observable.
- [ ] 8.12 A host that enrolled before the upgrade re-enrolls after it and transitions to a one-time secret.
- [ ] 8.13 **Old server, new fleetd**: a registry-reading fleetd against a server that still sends the global secret enrolls normally, falling back to `secret.txt`.
- [ ] 8.14 **New server with the flag on, old fleetd**: phases 1 and 2 still enroll the host, since the one-time secret arrives on the same `FLEET_SECRET` MSI property. Confirm recovery is inert rather than harmful on that host.

## 9. Documentation and rollout

- [ ] 9.1 Document that the secret delivered to Windows MDM hosts is single-use and re-issued on re-enrollment, in the Windows MDM setup guide.
- [ ] 9.2 Publish guidance to rotate the global enroll secret once the fleet has re-enrolled, since pre-change hosts still hold it.
- [ ] 9.3 Update `auth_use_one_time_enroll_secrets` configuration docs to cover Windows and state the Windows prerequisite from 1.4.
- [ ] 9.4 Add a changes/ entry.
- [ ] 9.5 Verify end to end on a real Windows host through Windows MDM, per the risk assessment, before enabling the flag anywhere.
