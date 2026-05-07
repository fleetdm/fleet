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
- [x] 2.6 Implementation note: gating happens server-side via a single indexed lookup (`ProfileHasACMEPayloadForCommand`) that bundles platform check, ACME-payload presence (`LOCATE` on `mobileconfig`), and pending-refetch dedup (EXISTS on `host_mdm_commands` PK). No plist parse on the hot path. Tracking row is added AFTER the commander succeeds (matching the iOS/iPadOS pattern) so a CertificateList enqueue failure doesn't leave a stale tracking row.

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

> Depends on Phase 1 for the macOS leg. iOS/iPadOS/Windows legs work independently. Customer outcome: Hydrant ACME and Okta SCEP certs auto-renew before expiration *for profiles redeployed with the marker variable*.

## 5. PR 2.1 — Schema and CA-list foundation

Goal: prepare the data model for ingestion-created managed-cert rows. No behavior change yet. Independently mergeable.

- [ ] 5.1 Verify `profile_uuid` format across `host_mdm_apple_profiles` / `host_mdm_windows_profiles` (sample existing rows) to finalize the marker-extraction regex used in PR 2.2
- [ ] 5.2 Migration: allow `host_mdm_managed_certificates.type` to be NULL (or add `non_proxied` enum value) — `server/datastore/mysql/migrations/tables/<timestamp>_AllowNullTypeOnHostMDMManagedCertificates.go` + test file
- [ ] 5.3 Add `CAConfigHydrant` legacy constant in `server/fleet/certificate_authorities.go`; include it in `ListCATypesWithRenewalSupport()` and `ListCATypesWithRenewalIDSupport()`
- [ ] 5.4 Update `RenewMDMManagedCertificates` SELECT in `server/datastore/mysql/mdm.go:2861-2960` to include rows where `type IS NULL` (or non-proxied sentinel)
- [ ] 5.5 Datastore tests: existing-type rows still selected, NULL-type rows now selected, non-renewable types still ignored

## 6. PR 2.2 — Ingestion-driven managed-cert row creation

Goal: extend `UpdateHostCertificates` to insert `host_mdm_managed_certificates` rows when ingested certs carry a renewal-ID marker. Core change.

- [ ] 6.1 Define renewal-ID extraction helper (regex on cert Subject CN/OU) — new file under `server/fleet/` or shared utility
- [ ] 6.2 Extend `UpdateHostCertificates` in `server/datastore/mysql/host_certificates.go` to scan newly inserted certs for the marker and INSERT missing `host_mdm_managed_certificates` rows
- [ ] 6.3 Verify the extracted profile_uuid resolves to a profile installed on the host before insert; log mismatches at debug level; do not insert otherwise
- [ ] 6.4 Datastore tests covering: first-time ingestion creates row; subsequent ingestion updates existing row; mismatched UUID ignored; multiple markers in same Subject (edge case); marker present but profile not installed on this host
- [ ] 6.5 Service-level integration test: simulate an osquery cert report with a marker-bearing cert and verify the managed-cert row materializes
- [ ] 6.6 Run `make generate-mock` if datastore interface signatures change

> **Coordination with #44691**: That PR restructures the matcher in `UpdateHostCertificates` (introduces `toInsertBySHA1` map, pool-selection per hmmc row, best-match-wins, monotonic-forward predicate, `hmmcBackfillGrace`). It targets `main` and is expected to land before this PR. Our INSERT path lands as a separate loop alongside that restructured matcher, reusing `toInsertBySHA1`/`incomingBySHA1` it already builds. Plan to rebase against main once #44691 merges.

## 7. PR 2.3 — Apple profile-upload validation

Goal: hard-reject Apple SCEP/ACME profiles missing the renewal-ID marker. This validation is the customer-facing trigger that drives the redeploy step.

- [ ] 7.1 Detect SCEP and ACME payload types in Apple profile content during validation in `server/service/apple_mdm.go` (around the existing fleet-variable validation at line 71+)
- [ ] 7.2 If a renewable payload is present, require `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` substring in the cert Subject (CN or OU); reject with `fleet.NewInvalidArgumentError` naming the variable and explaining the requirement
- [ ] 7.3 Confirm the same code path is hit by `fleetctl gitops` profile uploads; add an integration test if not already covered
- [ ] 7.4 Tests: payload + marker → accepted; payload + no marker → rejected with expected error message; non-renewable payloads → unaffected; existing profiles in DB are not retroactively rejected
- [ ] 7.5 Release-notes draft for the validation behavior change, including the redeploy guidance

## 8. PR 2.4 — Windows profile-upload validation

Goal: same as PR 2.3 for Windows. Separated because the Windows path has a pending design question.

- [ ] 8.1 Resolve the Windows Subject substitution open question (see design.md): can `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` be reused in the Windows profile cert Subject, or does Windows need a new Subject-targeted variable?
- [ ] 8.2 If a new variable is required, add it to `server/fleet/mdm.go` and substitution logic in `server/datastore/mysql/microsoft_mdm.go` and `server/mdm/microsoft/profile_variables.go`
- [ ] 8.3 Extend Windows profile validation in `server/service/windows_mdm_profiles.go:143-159` to detect SCEP cert configurations and require the renewal-ID variable in the Subject
- [ ] 8.4 Tests mirroring PR 2.3's coverage but for Windows profiles

## 9. PR 2.5 — Phase 2 documentation

Can ship in parallel with PR 2.3 / 2.4.

- [ ] 9.1 Update Fleet's certificate renewal guide (or create a new guide — Product decision pending) to document non-proxied SCEP and ACME renewal
- [ ] 9.2 Add example profiles (Hydrant ACME, Okta conditional access SCEP, Okta Verify static challenge SCEP) showing correct marker placement in Subject CN/OU
- [ ] 9.3 **Customer-facing redeploy guidance**: explicit upgrade-step doc explaining that existing profiles must be re-uploaded with the marker for renewal to activate; ideally include a `fleetctl` snippet or query to identify which existing profiles need updating
- [ ] 9.4 Reference doc note that `host_mdm_managed_certificates.type` may be NULL for non-proxied rows
- [ ] 9.5 Final release-notes entry consolidating Phase 2 changes

## 10. Phase 2 verification (QA, not a PR)

- [ ] 10.1 Hydrant ACME on real silicon Mac (Phase 1 fully shipped): redeploy a profile with the marker, force a cert near expiration, verify renewal cron triggers, profile re-pushes, new cert ingested via on-demand `CertificateList`, managed-cert row updated
- [ ] 10.2 Hydrant ACME on iOS / iPadOS: redeploy profile with marker, verify existing `IOSiPadOSRefetch` cron path picks up the new flow without regression
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
