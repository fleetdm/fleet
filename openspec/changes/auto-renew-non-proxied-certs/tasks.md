> **Phase boundaries**: This change covers two stories. **Phase 1 (#42827)** is MDM cert ingestion for macOS — independently shippable. **Phase 2 (#40639)** is non-proxied cert renewal — depends on Phase 1 for the macOS hardware-bound leg. Expected to ship together.
>
> **No backfills**: Customers must redeploy ACME/SCEP profiles with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the cert Subject to enable renewal. Redeploy fires `InstallProfile` → on-demand `CertificateList` (Phase 1) → cert ingestion → managed-cert row insert (Phase 2). The natural redeploy flow replaces what would have been a deploy-time backfill in either phase.

---

# Phase 1 — #42827 cert ingestion for macOS

> Customer outcome: ACME and other MDM-delivered certs become visible on macOS host details *after the customer redeploys the delivering profile*. No automatic backfill.

## 1. PR 1.1 — Storage foundation (origin column, dedup)

Goal: prepare `host_certificates` to store certs from a second ingestion source. No new behavior triggered yet.

- [x] 1.1.1 Migration: add `host_certificates.origin` column (`enum('osquery','mdm')`, default `'osquery'` for existing rows) — `server/datastore/mysql/migrations/tables/20260505115111_AddOriginToHostCertificates.go` + test file (timestamp bumped from 20260428151210 to land after newer migrations on main; merged via PR #44339)
- [x] 1.1.2 Update `UpdateHostCertificates` in `server/datastore/mysql/host_certificates.go` to set `origin` based on the ingestion source parameter; default to `osquery` when called from existing osquery paths
- [x] 1.1.3 Update soft-delete logic so each ingestion source only deletes its own `origin`-matching rows (osquery sync doesn't delete `mdm` rows; MDM response doesn't delete `osquery` rows)
- [x] 1.1.4 Datastore tests for source-scoped deletion: pre-populate rows with mixed origin, run osquery sync omitting an `mdm` row, verify it survives; mirror for MDM-side
- [x] 1.1.5 Run `make generate-mock` if datastore interface signatures change

## 2. PR 1.2 — On-demand CertificateList for macOS

Goal: hook `CertificateList` into the ACME `InstallProfile` ack flow on macOS. Activates ingestion for new and renewed profile installs.

- [x] 2.1 In `server/service/apple_mdm.go` `CommandAndReportResults` `case "InstallProfile"`, detect when an acked profile contained a `com.apple.security.acme` payload on a macOS host (PR #44372)
- [x] 2.2 Enqueue `CertificateList` for the host using existing commander; reuse `RefetchCertsCommandUUIDPrefix` for tracking
- [x] 2.3 Ensure `handleRefetchCertsResults` continues to process the response and call `UpdateHostCertificates` with `origin='mdm'`
- [x] 2.4 Tests: simulate an ACME `InstallProfile` ack on a macOS host and verify `CertificateList` is queued; verify result handler ingests certs with `origin='mdm'`
- [x] 2.5 Verify iOS/iPadOS path (`server/mdm/apple/apple_mdm.go:1585-1605`) still works unchanged — no regressions
- [x] 2.6 Implementation note: gating happens server-side via a single indexed lookup (`ProfileHasACMEPayloadForCommand`) that bundles platform check and ACME-payload presence (`LOCATE` on `mobileconfig`). No plist parse on the hot path. Tracking row is added AFTER the commander succeeds (matching the iOS/iPadOS pattern) so a CertificateList enqueue failure doesn't leave a stale tracking row. Pending-refetch dedup was deliberately NOT included — see design.md Decision 1.1's "No pending-refetch deduplication" note. Duplicates collapse via the `host_mdm_commands` `(host_id, command_type)` PK and `handleRefetchCertsResults` is idempotent.

> **Known gap — Flow B (enrollment-time)**: PR 1.2 covers Flow A (profile-install ack triggers refetch) but not Flow B (DEP enrollment with hardware attestation issues an ACME enrollment cert that's invisible to osquery). Hooking into `mdmlifecycle.turnOnApple` or `TokenUpdate` for newly-enrolled silicon Macs is required to fully close #42827 for the customer-cisneros-a use case. Tracked as a follow-up sub-task; not in PR 1.2's scope.

## 3. PR 1.3 — Phase 1 documentation

- [ ] 3.1 Update host details documentation to mention macOS ACME / MDM-delivered cert visibility — explicitly note that visibility activates per-host on profile install/redeploy
- [ ] 3.2 Release notes entry for Phase 1

## 4. Phase 1 verification (QA, not a PR)

- [ ] 4.1 Real silicon Mac via ABM/DEP enrollment with hardware attestation enabled — install a fresh ACME profile, verify cert appears on host details after `InstallProfile` ack
- [ ] 4.2 Hardware-bound ACME cert visible only via `CertificateList` — confirm it appears in API response with `origin=mdm` recorded internally
- [ ] 4.3 Dedup verification: a cert visible to both osquery and `CertificateList` appears as exactly one row
- [ ] 4.4 Existing-profile scenario: install an ACME profile, then verify that without re-pushing the profile no new ingestion occurs (validates the no-backfill design)

---

# Phase 2 — #40639 non-proxied cert renewal

> Depends on Phase 1 for the macOS leg. iOS/iPadOS/Windows legs work independently. Customer outcome: non-proxied ACME (e.g. customer's Hydrant deployment) and Okta SCEP certs auto-renew before expiration *for profiles redeployed with the marker variable*.

## 5. PR 2.1 — Schema and renewal-cron foundation

Goal: prepare the data model for ingestion-created managed-cert rows. No behavior change yet. Independently mergeable.

- [x] 5.1 Verify `profile_uuid` format across `host_mdm_apple_profiles` / `host_mdm_windows_profiles` — empirically resolved by reading the substitution code (`server/mdm/microsoft/profile_variables.go:125`, `server/mdm/apple/profile_processor.go:401-404`): the marker is always literally `"fleet-" + profile_uuid`. PR 2.2 marker extraction can rely on substring matching against CN/OU; no regex needed.
- [x] 5.2 Migration: allow `host_mdm_managed_certificates.type` to be NULL — `server/datastore/mysql/migrations/tables/20260507160833_AllowNullTypeOnHostMDMManagedCertificates.go` + test file. NOTE: revise the stashed migration to drop the `'hydrant'` enum addition — Hydrant is not modeled as a CA type (see design.md Decision 2.4).
- [ ] 5.3 (REMOVED — see design.md Decision 2.4) ~~Add `CAConfigHydrant` legacy constant; include in renewal lists~~. Hydrant is not modeled as a CA type. The NULL-`type` mechanism (5.4) is the entire renewal path for non-proxied flows.
- [x] 5.4 Update `RenewMDMManagedCertificates` SELECT in `server/datastore/mysql/mdm.go` to iterate `ListCATypesWithRenewalSupport()` plus a single NULL bucket using null-safe equal (`hmmc.type <=> ?`).
- [ ] 5.5 Datastore tests: existing-type rows still selected, NULL-type rows now selected, non-renewable types still ignored. (Reduce the stashed test — drop the `hydrant` arm; keep the `ndes` and NULL arms.)

## 6. PR 2.2 — Ingestion-driven managed-cert row creation

Goal: extend `UpdateHostCertificates` to insert `host_mdm_managed_certificates` rows when ingested certs carry a renewal-ID marker. Core change.

- [ ] 6.1 Marker extraction strategy: use substring search for `"fleet-" + profile_uuid` against ingested certs' CN/OU, mirroring the existing matcher loop's approach (origin/main `host_certificates.go:188-189`). No new regex helper needed — the source-of-truth profile UUID list comes from the host's `host_mdm_apple_profiles` / `host_mdm_windows_profiles` rows and can be enumerated.
- [ ] 6.2 Extend `UpdateHostCertificates` in `server/datastore/mysql/host_certificates.go` to add an INSERT pass: for each profile installed on the host without a corresponding `host_mdm_managed_certificates` row, search the incoming/toInsert pool for a cert whose CN or OU contains `"fleet-" + profile_uuid`; if found, INSERT a row populated with the cert's `serial`, `not_valid_before`, `not_valid_after`, NULL `type`, NULL `ca_name`, NULL `challenge_retrieved_at`. Reuse `toInsertBySHA1` / `incomingBySHA1` already built by #44691.
- [ ] 6.3 **Matcher-guard fix (load-bearing for ingestion-created rows):** adjust the existing matcher's `if !hostMDMManagedCert.Type.SupportsRenewalID() { continue }` skip so empty/NULL `Type` rows are NOT excluded. Without this, ingestion-created NULL-`type` rows would never have their `not_valid_after` advanced after a renewal completes, producing a non-terminating renewal loop. See design.md Decision 2.2.
- [ ] 6.4 Datastore tests covering: first-time ingestion creates NULL-`type` row; subsequent ingestion updates existing NULL-`type` row's `not_valid_after` (validates 6.3); mismatched UUID ignored; marker present but profile not installed on this host; existing proxied-`type` rows continue to work as before (no regression)
- [ ] 6.5 Service-level integration test: simulate an osquery cert report with a marker-bearing cert and verify the managed-cert row materializes
- [ ] 6.6 Run `make generate-mock` if datastore interface signatures change

> **Coordination with #44691**: That PR restructures the matcher in `UpdateHostCertificates` (introduces `toInsertBySHA1` map, pool-selection per hmmc row, best-match-wins, monotonic-forward predicate, `hmmcBackfillGrace`). It targets `main` and is expected to land before this PR. Our INSERT path lands as a separate loop alongside that restructured matcher, reusing `toInsertBySHA1`/`incomingBySHA1` it already builds. Plan to rebase against main once #44691 merges.

## 7. PR 2.3 — Apple profile-upload validation + variable rename

Goal: hard-reject Apple SCEP/ACME profiles missing the renewal-ID marker, and add code support for the renamed customer-facing variable (`$FLEET_VAR_CERTIFICATE_RENEWAL_ID`, per design.md Decision 2.7). This validation is the customer-facing trigger that drives the redeploy step.

- [ ] 7.1 Add `FleetVarCertificateRenewalID = "CERTIFICATE_RENEWAL_ID"` constant in `server/fleet/mdm.go` alongside the existing `FleetVarSCEPRenewalID`. Both are recognized by `FindFleetVariables`.
- [ ] 7.2 Extend `FleetVarSCEPRenewalIDRegexp` (or add a sibling regex) to match either name, then expose a unified `FleetVarRenewalIDRegexp` used by all substitution sites. Both names substitute to identical output: `"fleet-" + profile_uuid`. Touch points: `server/mdm/apple/profile_processor.go:401-404`, `server/mdm/microsoft/profile_variables.go:124-125`.
- [ ] 7.3 Detect ACME payload types in Apple profile content during validation in `server/service/apple_mdm.go` (around the existing fleet-variable validation) and reject when the renewal-ID marker is missing from the cert Subject. The per-CA SCEP validators (NDES/Custom SCEP/Smallstep) already enforce this requirement for SCEP profiles that use Fleet proxy variables; coverage for raw-SCEP profiles (no proxy vars, e.g. Okta conditional access / Okta Verify with static challenge) is split out into PR 2.3b (§7b below).
- [ ] 7.4 If an ACME payload is present, require `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the cert Subject **OU only** (CN placement is rejected — net-new surface per design.md Decision 2.7); reject with `fleet.NewInvalidArgumentError` naming `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` and stating the OU requirement. The error message MUST NOT mention the legacy `$FLEET_VAR_SCEP_RENEWAL_ID` — net-new surface, preferred name only.
- [ ] 7.5 Confirm the same code path is hit by `fleetctl gitops` profile uploads; add an integration test if not already covered.
- [ ] 7.6 Tests: ACME payload + marker in OU → accepted; payload + marker in CN only → rejected (net-new surface, OU-only per Decision 2.7); payload + legacy variable name → rejected (net-new, preferred only); payload + no marker → rejected with expected error message; non-renewable payloads → unaffected; existing profiles in DB are not retroactively rejected; substitution produces identical output for both variable names (substitution is unchanged; the OU/legacy restrictions are validation-only).
- [ ] 7.7 Release-notes draft for the validation behavior change, including the redeploy guidance and the variable-name rename note (legacy still accepted, new name preferred for new authoring).

## 7b. PR 2.3b — Apple non-proxied SCEP validation

Goal: close the gap left by the three existing per-CA SCEP validators, which only fire when Fleet proxy variables are present in the profile. A raw-SCEP profile (literal Challenge/URL strings, no Fleet proxy vars — the Okta SCEP / Okta Verify-static-challenge flows named in #40639) currently bypasses all renewal-ID enforcement. Mirrors the ACME validator added in PR 2.3 but triggers on `com.apple.security.scep` payload type.

- [ ] 7b.1 In `server/service/apple_mdm.go`, add `additionalNonProxiedSCEPValidation` next to `additionalACMEValidation`. Trigger: profile contains a `com.apple.security.scep` payload AND the profile uses none of the Fleet proxy variables (`FleetVarNDESSCEPChallenge` / `FleetVarNDESSCEPProxyURL`, `FleetVarCustomSCEPChallengePrefix` / `FleetVarCustomSCEPProxyURLPrefix`, `FleetVarSmallstepSCEPChallengePrefix` / `FleetVarSmallstepSCEPProxyURLPrefix`). When the profile DOES use those vars, the existing per-CA validators already enforce the renewal-ID requirement and adding this validator on top would double-validate.
- [ ] 7b.2 Apply the same Subject **OU-only** requirement as ACME: accept ONLY `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`, only in OU. Both restrictions follow design.md Decision 2.7's net-new rule — raw-SCEP renewal-ID enforcement did not exist before this validator, so no back-compat is owed for legacy variable name OR CN placement. Reject with `fleet.NewInvalidArgumentError`; the error message names only the preferred variable and states the OU requirement.
- [ ] 7b.3 Wire the new validator into `validateConfigProfileFleetVariables` before the fleetVars early-return, matching the integration shape of `additionalACMEValidation`.
- [ ] 7b.4 Tests in `apple_mdm_test.go` covering: raw SCEP with new marker in OU → accepted; new marker in CN only → rejected (OU-only on this net-new surface); legacy marker → rejected (preferred-only on this net-new surface); no marker at all → rejected; profile that also uses Fleet proxy vars → falls through to the existing per-CA validator path (no double-validation, no behavioral regression).
- [ ] 7b.5 Release-notes alignment: this validator is the customer-facing trigger for the Okta SCEP / Okta Verify customers named in #40639. PR 2.5 docs should describe ACME and raw-SCEP redeploy paths together rather than treating them as separate stories.

## 8. PR 2.4 — Windows profile-upload validation

Goal: same as PR 2.3 for Windows — including the legacy/new variable name back-compat (Decision 2.7).

- [ ] 8.1 Resolve the Windows Subject substitution open question (see design.md): can `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` be reused in the Windows profile cert Subject, or does Windows need a new Subject-targeted variable?
- [ ] 8.2 If a new variable is required, add it to `server/fleet/mdm.go` and substitution logic in `server/datastore/mysql/microsoft_mdm.go` and `server/mdm/microsoft/profile_variables.go`. Reuse the unified `FleetVarRenewalIDRegexp` from PR 2.3 task 7.2 so Windows accepts both legacy and new names without duplicating regex logic.
- [ ] 8.3 Extend Windows profile validation in `server/service/windows_mdm_profiles.go` to detect SCEP cert configurations and require either renewal-ID variable in the Subject (back-compat per Decision 2.7). Validation error messages reference only `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`.
- [ ] 8.4 Tests mirroring PR 2.3's coverage but for Windows profiles, including legacy-name back-compat.

## 9. PR 2.5 — Phase 2 documentation

Can ship in parallel with PR 2.3 / 2.4.

- [ ] 9.1 Update Fleet's certificate renewal guide (or create a new guide — Product decision pending) to document non-proxied SCEP and ACME renewal
- [ ] 9.2 Add example profiles (non-proxied ACME — generic plus a customer's-Hydrant illustration, Okta conditional access SCEP, Okta Verify static challenge SCEP) showing correct marker placement in Subject CN/OU
- [ ] 9.3 **Customer-facing redeploy guidance**: explicit upgrade-step doc explaining that existing profiles must be re-uploaded with the marker for renewal to activate; ideally include a `fleetctl` snippet or query to identify which existing profiles need updating
- [ ] 9.4 Reference doc note that `host_mdm_managed_certificates.type` may be NULL for non-proxied rows
- [ ] 9.5 Variable rename note (Decision 2.7): release-notes entry calling out that `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` is the new preferred name (per docs PR #44069); legacy `$FLEET_VAR_SCEP_RENEWAL_ID` remains accepted on pre-existing validators (proxy SCEP — NDES/Custom/Smallstep, Windows non-proxied SCEP) for back-compat, but net-new validators (ACME, raw Apple SCEP) accept only the preferred name. New profiles should use the new name everywhere. Confirm rachaelshaw's customer-facing guide PR (#43293) lands alongside this release.
- [ ] 9.6 Final release-notes entry consolidating Phase 2 changes

## 10. Phase 2 verification (QA, not a PR)

- [ ] 10.1 Non-proxied ACME (customer-cisneros-a's Hydrant deployment) on real silicon Mac (Phase 1 fully shipped): redeploy a profile with the marker, force a cert near expiration, verify renewal cron triggers, profile re-pushes, new cert ingested via on-demand `CertificateList`, managed-cert row updated; row remains NULL `type` throughout
- [ ] 10.2 Non-proxied ACME on iOS / iPadOS: redeploy profile with marker, verify existing `IOSiPadOSRefetch` cron path picks up the new flow without regression
- [ ] 10.3 Okta conditional access SCEP on macOS via osquery ingestion path (after profile redeploy)
- [ ] 10.4 Okta Verify SCEP (static challenge) on macOS (after profile redeploy)
- [ ] 10.5 Okta SCEP on Windows (after profile redeploy)
- [ ] 10.6 Negative path: do NOT redeploy a profile; confirm no managed-cert row materializes and no renewal happens (validates the no-backfill design and consistency)
- [ ] 10.7 Profile-validation rejection: attempt to upload an ACME/SCEP profile without the marker; confirm rejection with the expected error message

---

## 11. Per-PR checklist (apply to each implementation PR)

- [ ] 11.1 `make lint-go` passes
- [ ] 11.2 Datastore tests pass: `MYSQL_TEST=1 FLEET_MYSQL_TEST_PORT=3308 go test ./server/datastore/mysql/...`
- [ ] 11.3 Service tests pass: `MYSQL_TEST=1 REDIS_TEST=1 FLEET_MYSQL_TEST_PORT=3308 go test ./server/service/...`
- [ ] 11.4 `make generate-mock` if datastore interface changed
- [ ] 11.5 PR description names the parent story (#42827 or #40639) and the phase/PR position (e.g., "Phase 1 PR 1.2 — on-demand CertificateList")

---

## PR sequencing summary

```
   PHASE 1 (#42827)
   ──────────────────────────────────────────────────────
   PR 1.1 (origin column, dedup)  ─── foundation
       │
       ▼
   PR 1.2 (CertificateList trigger) ── activates ingestion
       │
       ▼
   PR 1.3 (docs)                  ─── parallel-able

   ──────────────── PHASE 1 SHIPS ────────────────

   PHASE 2 (#40639)
   ──────────────────────────────────────────────────────
   PR 2.1 (schema + CA list)      ─── foundation
       │
       ▼
   PR 2.2 (UpdateHostCertificates) ── core change
       │
       ├──────────────────┬───────────────────┐
       ▼                  ▼                   ▼
   PR 2.3 (Apple        PR 2.4 (Windows    PR 2.5 (docs,
   validation)          validation,        parallel)
                        blocked on
                        Windows TODO)
```

Phase 1: 3 PRs (was 4 — backfill dropped).
Phase 2: 5 PRs (was 6 — linkage backfill dropped).
Total: 8 implementation/docs PRs (down from 10).
