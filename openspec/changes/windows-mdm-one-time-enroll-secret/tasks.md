## 1. Close blocking decisions

- [x] 1.1 Expiry for an unconsumed Windows secret: **decided, no TTL, matching macOS**. Recorded in design.md decision 8 and in security#68.
- [x] 1.2 Recovery endpoint: **decided, reuse `resendHostMDMProfileEndpoint`** with an added check so the end user cannot resend it, following what Apple did. Design decision 10.
- [x] 1.3 Spent secret on a legitimate re-enroll: **decided, the IT admin resends to recover.** No automatic re-issue, matching Apple. Recovery is the single path back for any host whose secret is spent or lost.
- [x] 1.4 Tier: **decided, Premium only for now**, reusing `auth.use_one_time_enroll_secrets` and its existing force-off unchanged. The gate may move to Free later, so nothing in the Windows path may depend on teams existing. Design decision 9.
- [x] 1.5 Minimum fleetd version for recovery: **decided, document it only.** No version-detection code, no per-host gating, no support for the endpoint degrading on older fleetd.
- [x] 1.6 Stuck-state carve-out: **not needed.** Windows profiles are verified by the device's SyncML ack and land in `verified`, which is already resendable; only proxied SCEP profiles are downgraded to `verifying`. Apple's `verifyingAllowed` hack has no Windows equivalent. Design decision 10.

## 2. Schema and minting (phase 1)

- [ ] 2.1 Write a migration adding a nullable Windows MDM enrollment reference to `host_one_time_enroll_secrets`, leaving `host_id` nullable so Apple rows are unaffected. Follow `.claude/rules/fleet-database.md`: `IF NOT EXISTS` guards, no-op `Down_`, migration test only if it changes data.
- [ ] 2.2 Relax the identity columns so a Windows row can be minted with `platform`, `hardware_uuid`, and `hardware_serial` unset or partially set from the enrollment.
- [ ] 2.3 Add an enrollment-bound minting function alongside `mintHostOneTimeEnrollSecret`, keyed on the MDM enrollment rather than `hosts.uuid`, sharing the reuse-vs-mint branch.
- [ ] 2.4 Implement the reuse predicate exactly as macOS does: resolve an existing row with `consumed_at IS NULL` to the same value; mint only when there is none.
- [ ] 2.5 Serialize concurrent mints for the same enrollment, using a lock on a row that exists at mint time (the enrollment row), mirroring the `FOR UPDATE` reasoning in `mintHostOneTimeEnrollSecret`.
- [ ] 2.6 Resolve the team at mint from the Windows MDM enrollment, via `GetWindowsEnrollmentDefaultFleet`, and store it on the row.
- [ ] 2.7 Extend `CleanupHostOneTimeEnrollSecrets` so Windows rows with no host linkage are swept on the same rules as orphaned Apple rows.

## 3. Wire minting into the fleetd install (phase 1)

- [ ] 3.1 Change `enqueueInstallFleetdCommand` (`server/service/microsoft_mdm.go:1507`) to stop putting any secret on the MSI command line, gated on `auth.use_one_time_enroll_secrets`. The `FLEET_SECRET` property is left at its `"dummy"` default, which the MSI already treats as "no secret".
- [ ] 3.2 Keep the existing behavior byte-for-byte when the flag is off.
- [ ] 3.3 Handle the no-global-secret case: with the flag on, a Windows MDM enrollment must still issue a one-time secret rather than silently sending an empty `FLEET_SECRET`.
- [ ] 3.4 Verify that repeated session-start alerts (`processNewSessionAlert`) reuse the live secret rather than minting per alert, under the 1-minute fast poll.

## 4. Consumption and binding (phase 1)

- [ ] 4.1 Bind Windows rows to the enrollment via `mdm_hardware_id` at mint, for provenance. Record presented identifiers on first consumption.
- [ ] 4.5 Gate host-to-enrollment linkage on the secret: in the orbit-enroll reverse link (`server/service/orbit.go:430-465`), link only when the presented secret was minted for that enrollment, resolving the enrollment from the secret rather than from `MDMWindowsGetUnlinkedEnrolledDeviceWithHardwareSerial`.
- [ ] 4.6 Refuse the reverse link entirely when a shared secret is presented and the capability is enabled, since a device-asserted serial is not authorization.
- [ ] 4.7 Confirm the DevDetail and osquery-ingest linkage paths are unaffected, since neither presents a secret at linkage time and both remain guarded by `MDMWindowsConflictingEnrollmentHardwareID`.
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
- [ ] 6.3 Extend `resendHostMDMProfileEndpoint` to cover the registry-carrying Windows profile, adding the discriminator that refuses the end-user My device path (`ResendDeviceHostMDMProfile`) the way Apple does for the fleetd configuration profile.
- [ ] 6.4 Confirm the registry profile lands in `verified` on a wedged-but-MDM-reachable host, so no `verifyingAllowed` carve-out is required. If it can reach `verifying`, revisit 1.6.
- [ ] 6.5 Ensure triggering recovery mints a fresh secret when the previous is consumed, and re-delivers the same value when it is not.
- [ ] 6.6 Have the MSI create the registry key at install time with a DACL matching `secret.txt` (`O:SYG:SYD:PAI(A;;FA;;;SY)(A;;FA;;;BA)` — SYSTEM and Administrators only, inheritance disabled), so the MDM channel writes only the value into a pre-hardened key.
- [ ] 6.7 Keep the Windows path free of `IsPremium()` checks so the tier gate stays a one-line change when it moves to Free.

## 7. Recovery, agent side (phase 3)

- [ ] 7.0 Have orbit tolerate starting with **no** secret and poll for one, rather than treating an unresolvable secret as fatal (`orbit/cmd/orbit/orbit.go:470`). This is the one genuinely new agent behavior the registry-primary carrier requires, since the profile may land after the MSI finishes.
- [ ] 7.1 Have orbit on Windows read the registry location on startup and while waiting, as the analogue of the macOS `--use-system-configuration` loop. The read MUST be additive: fall back to `secret.txt` and the keystore when the registry value is absent, so new fleetd against an old server is a no-op.
- [ ] 7.2 Adopt a newer secret found there, replacing the stored credential, without re-enrolling an already-enrolled host that has no new secret waiting.
- [ ] 7.3 Delete the registry value once consumed, mirroring `readEnrollSecretFromFile`'s move-then-delete of `secret.txt`, so presence of the value means "a new secret is waiting".
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
- [ ] 9.3 Update `auth_use_one_time_enroll_secrets` configuration docs to cover Windows, note it is Premium-only today, and state the minimum fleetd version required for recovery to work (per 1.5, documentation is the whole mechanism here).
- [ ] 9.6 Document that recovering a stuck Windows host is an IT-admin action: resend the profile that writes the enroll secret to the registry. End users cannot resend it.
- [ ] 9.4 Add a changes/ entry.
- [ ] 9.5 Verify end to end on a real Windows host through Windows MDM, per the risk assessment, before enabling the flag anywhere.
