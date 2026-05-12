## Context

Fleet's existing renewal pipeline assumes the server is in the cert-issuance path. For Fleet-issued MDM enrollment certs, `RenewSCEPCertificates` reads `nano_cert_auth_associations`. For profile-delivered certs from CAs that Fleet proxies (NDES, Custom SCEP Proxy, DigiCert, Smallstep), `RenewMDMManagedCertificates` reads `host_mdm_managed_certificates` rows that the proxy step populates at issuance.

For non-proxied ACME (e.g. the customer-cisneros-a engagement, where the customer deploys an ACME profile against a private Hydrant fork) and non-proxied SCEP (Okta conditional access, Okta Verify), Fleet is not in the issuance path. The device performs the cert exchange directly with the CA. Fleet first sees the cert when it's reported via osquery (software certs) or the MDM `CertificateList` command (hardware-bound certs on Apple platforms). Today on macOS the `CertificateList` command is not used at all — only iOS/iPadOS use it via `IOSiPadOSRefetch`. So hardware-bound ACME certs on macOS are effectively invisible to Fleet, and even for software certs no `host_mdm_managed_certificates` row exists for the non-proxied flows so the renewal cron has nothing to act on.

Importantly, this change does NOT add Hydrant ACME as a registerable CA type. Hydrant does not yet officially support ACME; the customer-cisneros-a use case relies on a private Hydrant fork. A `hydrant` CA type would not generalize to other customers, would bake a name into `host_mdm_managed_certificates.type` that may conflict with whatever the official Hydrant ACME integration looks like later, and is not necessary because the non-proxied mechanism described below works for any external ACME or SCEP server. When Hydrant ships official ACME support upstream, a first-class CA registration may be added separately.

This change addresses both halves in two phases that ship as independent PR sequences:

- **Phase 1 (#42827)** — extend MDM `CertificateList` ingestion to macOS. Self-contained, customer-visible (certs appear on host details page), independently shippable.
- **Phase 2 (#40639)** — extend cert ingestion to also create `host_mdm_managed_certificates` rows so the existing renewal cron activates for non-proxied flows. Depends on Phase 1 for the macOS leg, otherwise independent (iOS/iPadOS already have `CertificateList` cadence; software-cert SCEP/ACME on macOS/Windows uses osquery ingestion).

## Goals / Non-Goals

**Goals:**
- Make MDM-delivered certs visible on the host details page on macOS (Phase 1).
- Auto-renew non-proxied ACME (including the customer-cisneros-a Hydrant deployment) and non-proxied SCEP profile-delivered certs on Apple platforms (macOS post-Phase 1, iOS, iPadOS today) and Windows (Phase 2).
- Reuse existing renewal cron unchanged. Reuse existing renewal threshold logic.
- Use a single mechanism (extracting `fleet-<profile_uuid>` marker from cert Subject) for all platforms and all CA types in scope.
- Validate at profile upload that renewable certs include the marker, so silent non-renewal becomes impossible to misconfigure.
- Keep Phase 1 and Phase 2 independently mergeable so each is reviewable in isolation.

**Non-Goals:**
- Custom EST Proxy renewal (deliberately deferred; no customer driver).
- New first-class CA type for Hydrant ACME (Hydrant does not officially support ACME yet; customer's private fork is the only deployment in scope; the non-proxied mechanism handles it without a CA registration).
- New first-class CA type for Okta SCEP (renewal works without one).
- Renewal verification / silent-failure detection — orthogonal generic improvement that applies equally to existing proxied flows; out of scope here.
- Generic operational alerting for unbounded renewal-loop scenarios — same reasoning, generic concern.
- Recurring macOS `CertificateList` cadence (Phase 1 deliberately uses on-demand-only for load reasons).

## Phase 1 Decisions (#42827 cert ingestion)

### Decision 1.1: On-demand `CertificateList` per `InstallProfile` ack on macOS

iOS/iPadOS use a recurring hourly cron (`IOSiPadOSRefetch`). macOS instead triggers `CertificateList` on demand when an ACME `InstallProfile` ack is received. Rationale: a recurring hourly `CertificateList` against every macOS host in a large fleet would be a significant MDM traffic increase. The on-demand model couples cert visibility to the events that actually change cert state (profile installs and renewals).

**Alternatives considered:**
- *Recurring daily/weekly cadence on macOS.* Rejected for v1 to keep ingestion traffic minimal. Could be revisited later as a backstop for external state drift.
- *Use `IOSiPadOSRefetch` cron on macOS too.* Rejected — Apple's protocol semantics differ enough between the platforms that separate trigger paths are cleaner.

**Implementation refinement (PR 1.2 final shape):** The InstallProfile-ack handler runs on every MDM command result, so the trigger gate lives in the hot path. A single indexed query (`ProfileHasACMEPayloadForCommand`) returns host platform, profile UUID, and ACME-payload presence (computed via `LOCATE` on the `mobileconfig` blob). The common case (non-darwin or non-ACME) early-returns without parsing the profile or transferring the blob to Go. Tracking row insertion happens AFTER `commander.CertificateList` succeeds — matching the iOS/iPadOS `IOSiPadOSRefetch` pattern — so a commander failure doesn't leave a stale tracking row that would suppress future triggers.

**No pending-refetch deduplication.** The trigger deliberately does NOT skip enqueueing `CertificateList` when a previous refetch for the same host is still in flight. Reason: a refetch enqueued before the new `InstallProfile` ack can return BEFORE the device's ACME exchange has actually issued the new cert, capturing pre-renewal state. Skipping the new trigger because of that in-flight refetch would lose the renewed cert until something else surfaced it. Letting each ack queue its own refetch ensures the post-exchange state is always captured. Duplicate enqueues are tolerated because:
- `host_mdm_commands` has a `(host_id, command_type)` primary key, so duplicate INSERTs collapse via `ON DUPLICATE KEY UPDATE` rather than erroring.
- `handleRefetchCertsResults` is idempotent against an already-removed tracking row, so the second result lands safely after the first.

This was a course-correction during PR 1.2 review — an earlier draft included `EXISTS` on `host_mdm_commands` for dedup; the race-window concern surfaced in review and the dedup was removed.

### Decision 1.2: No deploy-time backfill — redeploy is the trigger

We do NOT run a one-time `CertificateList` backfill across the macOS fleet at deploy. Rationale: Phase 2 requires customers to redeploy SCEP/ACME profiles with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the cert Subject for renewal to function. Redeploy fires `InstallProfile`, which fires the on-demand `CertificateList` trigger from Decision 1.1, which surfaces the certs into `host_certificates`. The natural flow covers existing fleet without a separate mechanism.

**Implications:**
- Customers who do not redeploy a particular ACME/SCEP profile see no certs from it in the host details page on macOS. Same outcome as today (unchanged behavior). No regression.
- Visibility on macOS becomes conditional on profile activity. Customers who want immediate visibility for compliance/audit reasons must take an explicit action (redeploy).
- Eliminates the rate-limiting / resumption / runbook complexity that a fleet-wide backfill would introduce.

**Alternatives considered:**
- *One-time `CertificateList` backfill across the fleet.* Rejected: customers must redeploy anyway to enable renewal, so the backfill duplicates the work the redeploy already triggers. Adds engineering and operational complexity (rate-limiting, completion tracking, resume-on-restart, monitoring) for a transparency benefit that is shorter-lived than the redeploy timeline.
- *Synchronously backfill all hosts at first deploy.* Rejected for the same reason plus the thundering-herd problem on APNs / the MDM command queue.

### Decision 1.3: Unified storage with `origin` column

Store certs from both osquery and MDM `CertificateList` ingestion in `host_certificates`, deduplicated by `sha1_sum`. Add an `origin` column (`osquery` / `mdm`) so each ingestion source only soft-deletes rows it owns. The column is internal — not exposed in the public API.

**Alternatives considered:**
- *Separate `host_mdm_certificates` table.* Rejected: complicates the host details API by requiring a UNION across two tables. Single-table-with-origin is simpler.
- *Expose `origin` in the API.* Rejected per discussion — clients only care about the unified deduped list.

## Phase 2 Decisions (#40639 renewal extension)

### Decision 2.1: Extend `UpdateHostCertificates` to insert managed-cert rows

`UpdateHostCertificates` (`server/datastore/mysql/host_certificates.go:30-200`) today only *updates* existing `host_mdm_managed_certificates` rows by matching ingested certs against `"fleet-" + row.profile_uuid`. We extend it: for each ingested cert, scan its Subject CN/OU for a `fleet-<uuid>` pattern; if found and no managed-cert row exists for that (host, profile), insert one populated from the cert.

**Alternatives considered:**
- *Create the row at profile install time* (a "pending" state). Rejected: produces orphan rows when a profile installs but the cert is never issued (CA failure, attestation rejected). The existing model — row exists iff a cert has been observed — has cleaner semantics.
- *Add a separate "non-proxied managed cert" table*. Rejected: doubles the surface area of the renewal cron and the ingestion linkage logic. The existing table fits the new rows with one nullable column.

### Decision 2.2: `host_mdm_managed_certificates.type` becomes nullable

For ingestion-created rows we don't know the CA type — Fleet wasn't in the issuance path. The migration drops the `NOT NULL DEFAULT 'ndes'` and allows NULL. The renewal cron iterates `ListCATypesWithRenewalSupport()` plus a single NULL bucket, using null-safe equal (`hmmc.type <=> ?`) so the same parameterized query handles both registered types and NULL with no SQL branching.

We do NOT add a `non_proxied` enum sentinel. NULL has the right semantics (unknown — Fleet wasn't in the issuance path), avoids a fake CA-type value that has to be excluded from every CA-type-aware query, and matches how the column is read by code that expects the registered CA type when one is present.

**Knock-on requirement — matcher guard:** the existing `UpdateHostCertificates` matcher (#44691) skips rows where `!Type.SupportsRenewalID()`, which evaluates to `false` for the empty-string zero value that NULL scans into. Without an adjustment here, ingestion-created NULL-`type` rows would never have their `not_valid_after` advanced after a renewal completes, producing a non-terminating renewal loop. Phase 2 must treat empty/NULL `Type` as renewal-eligible in the matcher path.

**Alternatives considered:**
- *Add a `non_proxied` enum sentinel.* Rejected per above — extra value to maintain, no semantic gain.
- *Synthesize a type at insert time by inspecting the profile content* (e.g., is it `com.apple.security.acme`? a SCEP payload pointing at a registered URL?). Rejected: brittle, leaks profile-content awareness into the datastore layer, and gains nothing — the renewal cron only cares about whether a row should be considered, not what produced it.

### Decision 2.3: Marker extraction via regex on Subject CN/OU

Inverted matching loop. Today: for each row, check ingested certs for `fleet-<row.profile_uuid>`. New: for each ingested cert, regex-extract `fleet-<uuid>` from Subject CN/OU and look up the profile by UUID.

Constraints on the extracted UUID:
- Must match Fleet's `profile_uuid` format (varchar(37), typically standard UUID with dashes — verify before regex finalization).
- Looked-up profile must exist in `host_mdm_apple_profiles` or `host_mdm_windows_profiles` for this host. Otherwise the cert is from a stale or copied profile UUID and is ignored.

**Alternatives considered:**
- *Match by Issuer + Serial recorded at issuance.* Rejected: only works for proxied flows where Fleet saw the issuance. Useless here.
- *Match by Subject == host identifier.* Rejected: breaks when one host has multiple profiles using the same CA.

### Decision 2.4: Hydrant is not modeled as a CA type

We do NOT add a `CAConfigHydrant` constant or include a `hydrant` value in `host_mdm_managed_certificates.type`. Reasoning:

- Hydrant does not yet officially support ACME. The customer-cisneros-a deployment uses a private fork. A `hydrant` enum value or `CAConfigHydrant` constant would not generalize to other customers.
- The non-proxied mechanism (Decision 2.2: NULL `type`, plus the renewal cron's NULL bucket) handles the customer's certs without any Hydrant-specific code path. The mechanism applies equally to any external ACME or SCEP server the customer's profile points at.
- Once Hydrant ships official ACME support upstream, a first-class Hydrant ACME CA registration may be added separately. That work would include a CA configuration, an issuance proxy, and a real type value — the right model for it. Pre-baking the type now would force a name choice that may not match the eventual official integration.

**Alternatives considered:**
- *Add `CAConfigHydrant` now and use it for the customer.* Rejected: bakes a name into a shipped enum that may conflict with future official Hydrant ACME work; produces a CA type whose only meaning is "the customer-cisneros-a use case"; doesn't generalize to the next non-proxied customer.

### Decision 2.5: No DB-side linkage backfill — redeploy provides the marker-bearing certs

We do NOT run a DB-side linkage backfill at Phase 2 deploy. Rationale: marker-bearing certs do not exist on devices until the customer redeploys profiles with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the Subject. Before redeploy, existing certs lack the marker — there is nothing to link.

When the customer redeploys, the new ACME/SCEP exchange produces a cert with the marker in its Subject. Phase 1's on-demand `CertificateList` ingests it. Phase 2's `UpdateHostCertificates` insert path (Decision 2.1) creates the `host_mdm_managed_certificates` row. The renewal cron then picks it up at threshold.

**Alternatives considered:**
- *Backfill scan over `host_certificates`.* Rejected: pre-redeploy certs don't carry the marker, so scanning them yields nothing useful. Post-redeploy certs are linked by the natural insert path. The backfill would be a no-op in both regimes.
- *Re-run Phase 1's `CertificateList` backfill after Phase 2 ships.* Same problem — pre-redeploy certs lack the marker. Even if visible in `host_certificates`, they wouldn't link. And Phase 1's backfill itself was dropped (Decision 1.2).

### Decision 2.6: Profile-upload validation rejects missing marker

If a profile contains a SCEP or ACME payload (Apple) or a `ClientCertificateInstall/SCEP/...` configuration (Windows) and lacks `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the cert Subject, the upload fails with `fleet.NewInvalidArgumentError`. The error message names the variable and explains it's required for auto-renewal.

**Alternatives considered:**
- *Allow upload, surface a warning.* Rejected: silent non-renewal is the customer-promise failure mode this story exists to prevent. Hard rejection is worth the small UX cost.
- *Allow upload, document the requirement.* Same rejection rationale.

### Decision 2.7: Variable rename — accept both `SCEP_RENEWAL_ID` and `CERTIFICATE_RENEWAL_ID`

The customer-facing variable name was renamed from `$FLEET_VAR_SCEP_RENEWAL_ID` to `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` per PR #44069 (merged 2026-05-01 into `docs-v4.86.0`), referenced by #40639 as "[New variable name]". Reasoning: the variable is no longer SCEP-specific — it must apply to ACME profiles too for the customer-cisneros-a use case (and any future non-proxied flow), so the SCEP-prefixed name reads as a bug to anyone authoring an ACME profile.

Today (4.85 and earlier) the only valid name is `SCEP_RENEWAL_ID`. The rename is half-shipped: docs use the new name, code still defines and substitutes only the old name. Phase 2 closes that gap.

**Implementation approach:** add `FleetVarCertificateRenewalID = "CERTIFICATE_RENEWAL_ID"` alongside the existing `FleetVarSCEPRenewalID`. Both are recognized by `FindFleetVariables`. Both substitute to the same value (`"fleet-" + ProfileUUID`) via the same substitution helper — implemented as a single regex matching either name. Validation error messages reference only the new `CERTIFICATE_RENEWAL_ID` (we want new authoring to use the new name).

**Acceptance rule — pre-existing surfaces vs net-new surfaces:** the back-compat motivation below applies only to validation surfaces that customers have already authored against. Specifically:

- **Proxy SCEP validators (NDES / Custom SCEP / Smallstep)** — pre-existing since 4.65 (PR #34403). Accept BOTH the legacy `SCEP_RENEWAL_ID` and the preferred `CERTIFICATE_RENEWAL_ID`. Customers may have deployed 4.85-era SCEP profiles using the legacy name; hard-rejecting it would break those on next upload/edit.
- **ACME validator (PR 2.3)** — net-new. Accept ONLY `CERTIFICATE_RENEWAL_ID`. There are no pre-rename deployed ACME profiles to back-compat against — ACME validation didn't exist before this PR, and pre-rename ACME docs/examples (#40639) already used the new name. Accepting a SCEP-prefixed variable in an ACME context perpetuates the very confusion the rename was meant to fix.
- **Raw (non-proxied) SCEP validator (PR 2.3b)** — net-new. Accept ONLY `CERTIFICATE_RENEWAL_ID`. Raw-SCEP renewal-ID enforcement did not exist before PR 2.3b, so there are no deployed customer profiles relying on the legacy name being accepted by *this* validator. (Customers using raw SCEP today author the Subject by hand; the renewal-ID requirement is new.) Apply the same net-new rule as ACME — preferred-only, no perpetuation of the SCEP-prefixed name in fresh surfaces.
- **Windows non-proxied SCEP validator (PR 2.4)** — pre-existing surface. Windows already enforced renewal-ID via `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID` in 4.85; PR 2.4 swaps the variable name as part of the rename but the validation surface itself predates this story. Accept BOTH legacy and preferred names for the same back-compat reasoning as the proxy SCEP validators.

Substitution is unaffected by this distinction: both names continue to substitute identically across all platforms. The distinction is purely about what validation lets through at upload time.

**Acceptance rule — CN vs OU placement of the marker:** the same pre-existing-vs-net-new distinction applies to *where* the marker is allowed in the cert Subject. Every validator's error message has always read "must be in the SCEP/ACME certificate's organizational unit (OU)" — but the proxy SCEP validators silently accepted the marker in CN as well, an inherited message/behavior mismatch.

- **Pre-existing surfaces (proxy SCEP, Windows non-proxied SCEP)** — keep accepting the marker in CN or OU. The 4.85 test fixture `custom-scep-validation.mobileconfig` places `$FLEET_VAR_SCEP_RENEWAL_ID` in the CN (`WIFI $FLEET_VAR_SCEP_RENEWAL_ID`), proving real-world deployments authored against CN. Hard-rejecting CN would break those profiles on next upload/edit.
- **Net-new surfaces (ACME PR 2.3, raw Apple SCEP PR 2.3b)** — require the marker in OU only. No deployed profiles to back-compat against on these surfaces. OU is the semantically appropriate field (CN typically encodes principal identity, not metadata) and the error message already says "must be in the OU" — tightening here aligns behavior with the long-standing message and the docs. The ingestion matcher (`host_certificates.go:195`) keeps its defensive CN-or-OU search; tightening validation doesn't loosen the matcher.

**Why accept both rather than hard-rename:**
- Customers running 4.85 likely have `$FLEET_VAR_SCEP_RENEWAL_ID` in deployed SCEP profile Subjects. Hard-renaming the substitution constant would break those profiles on the next upload/edit cycle (validation passes, but substitution leaves the literal `$FLEET_VAR_SCEP_RENEWAL_ID` string in the profile, which the device CA rejects).
- Backwards-compat cost is small: one extra regex alternation, one extra constant. No new datastore work, no migration.
- We can deprecate `SCEP_RENEWAL_ID` in a later release once telemetry shows the long tail has migrated.

**Alternatives considered:**
- *Hard-rename — only `CERTIFICATE_RENEWAL_ID` works.* Rejected per the customer-impact reasoning above.
- *Hard-rename with a deprecation warning on uploads using the old name.* Considered. Same back-compat issue for already-deployed profiles that don't get re-uploaded — they'd silently stop substituting. Reject for the same reason.
- *Keep `SCEP_RENEWAL_ID` as the only name and revert the docs PR.* Rejected: docs decision is product-led and reflects the variable's actual scope (any cert, not just SCEP). The mismatch is a code-side gap to close, not a docs error to revert.

### Decision 2.8: INSERT path inherits the matcher's date-validity filter

The INSERT path added in Decision 2.1 reuses the same per-cert validity filter as the existing matcher (#44691):

```
if cert.NotValidBefore.After(now) || cert.NotValidAfter.Before(now) {
    continue
}
```

For the **matcher's UPDATE path** this filter is clearly correct: it prevents the matcher from regressing an existing `hmmc.not_valid_after` backward when the device is reporting both a stale (just-expired) cert and a freshly-issued one alongside it. Without the filter, best-match-by-NotValidBefore could latch onto the older cert.

For the **INSERT path** the filter is a deliberate design choice with a real trade-off:

```
   Cert in pool: marker matches profile, NotValidAfter < now (already expired)
   No existing hmmc row.

   With filter (chosen):
     INSERT skipped → no hmmc row → renewal cron has nothing to act on
     → cert stays unrenewed until device reports a fresh cert with the marker
     (i.e., until the device successfully re-ACMEs on its own)

   Without filter:
     INSERT with expired dates → cron immediately triggers re-push
     → device hopefully re-ACMEs → matcher updates row with fresh dates
     If device DOESN'T re-ACME (attestation failure, CA rejection, etc.):
        row sticks around with expired dates, cron loops forever
```

We keep the filter on the INSERT path for two reasons:

- **Symmetry with the matcher.** Same shape of pool iteration on both paths makes the code easier to reason about and avoids two subtly different "how do we evaluate a pool cert" rules in the same function.
- **Silent renewal-failure detection is a Non-Goal.** If a device's cert is already past expiry without a fresh one alongside it, that's a deeper failure mode (attestation rejection, CA outage, profile delivery problem) that re-pushing the profile won't necessarily fix. Inserting an expired-dates row would create a permanently-stuck row that the renewal cron loops on indefinitely — exactly the silent-failure pattern this story explicitly defers to a future change.

The trade-off: customers whose certs have already expired BEFORE Phase 2 deploys (and whose devices haven't re-ACMEd to produce a fresh cert) won't get auto-recovered by Phase 2 alone. They need to either redeploy the profile manually or wait for the eventual silent-failure-detection follow-up.

**Alternatives considered:**
- *Drop the filter on INSERT only.* Rejected per the silent-failure-loop concern. Re-push without verification of the renewal outcome is the exact pattern that the deferred renewal-verification story is meant to address; building it into INSERT now would short-circuit that future design.
- *Drop the filter on INSERT but require not-too-old (e.g., expired within last N days).* Considered. Adds a new tunable constant for one edge case; defers rather than resolves the silent-failure problem; rejected as over-engineering.

## Phase Independence

The expected ship cadence is "both phases together" — both deliver value to the same customer-cisneros-a use case and the redeploy step that activates Phase 2 is also what surfaces certs into Phase 1's ingestion. Each phase is still independently mergeable to keep PRs reviewable in isolation.

Phase 1 alone (if Phase 2 is delayed): macOS ACME certs become visible after the customer reinstalls the delivering profile. No renewal yet.

Phase 2 alone (if Phase 1 is delayed): iOS/iPadOS non-proxied ACME renewal works (existing `IOSiPadOSRefetch` cadence ingests the certs after redeploy), Windows Okta SCEP renewal works (osquery ingests), macOS Okta SCEP works (osquery ingests software certs after redeploy). Only macOS hardware-bound non-proxied ACME (the customer-cisneros-a target) requires Phase 1.

## Risks / Trade-offs

- **Risk**: Existing customer profiles in production may be missing the marker. Hard validation breaks them on next edit.
  → Mitigation: validation only fires on new uploads/edits, not retroactively on existing profiles. Document the requirement in release notes. Existing-fleet behavior unchanged for profiles already in place — they simply won't auto-renew, which matches today's behavior.

- **Risk**: Profile UUID format assumption — the regex relies on `varchar(37)` standard UUID shape. If Fleet ever changes profile UUID format, the regex misses certs.
  → Mitigation: source-of-truth is `host_mdm_*_profiles.profile_uuid`. Sample existing rows to confirm format before finalizing regex. Make the regex permissive within reasonable bounds and reject captures that don't resolve to an existing profile.

- **Risk**: Customer copies a profile from one Fleet instance to another. Old profile UUID is baked into the cert Subject. New Fleet instance can't resolve it.
  → Mitigation: cert is simply not linked, no harm done. Renewal won't happen until a new cert with a current-instance profile UUID lands. Document this as a known constraint of cert portability.

- **Risk**: Type B silent failure (profile installs, cert exchange silently fails) leads to unbounded retry loop.
  → Accepted: identical to existing proxied-CA behavior in production. Out of scope per Non-Goals.

- **Risk**: Customers fail to redeploy ACME/SCEP profiles after upgrade and assume renewal is working when it isn't.
  → Mitigation: profile-upload validation (Decision 2.6) rejects new uploads missing the marker, surfacing the requirement actively. Release notes call out the redeploy step explicitly. Documentation includes a checklist customers can run to verify renewal is active for each profile.

- **Trade-off**: Reusing `host_mdm_managed_certificates` for non-proxied rows pollutes its semantics (the column `type` no longer reliably identifies the CA). Cleaner alternative would be a separate table, but the duplication cost outweighs the conceptual purity.

## Migration Plan

**Phase 1 deploy:**
1. Schema migration: add `host_certificates.origin` column with default `osquery` for existing rows.
2. Code deploy: `CertificateList` ack-trigger handler for macOS, source-aware deletion logic, dedup by `sha1_sum`.
3. No backfill job. Customer redeploys of ACME/SCEP profiles activate ingestion per-host.

**Phase 2 deploy:**
1. Schema migration: allow `host_mdm_managed_certificates.type` to be NULL.
2. Code deploy: `UpdateHostCertificates` insert path, renewal-list update, profile validation.
3. Customer action required: redeploy ACME/SCEP profiles with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in cert Subject. Until they do, renewal does not activate for that profile.
4. Verification: confirm renewal cron picks up rows created by post-redeploy ingestion as certs near threshold.

**Rollback**: per phase. Phase 1 code rollback returns to existing iOS/iPadOS-only `CertificateList` behavior; the `origin` column persists harmlessly. Phase 2 code rollback restores update-only behavior on `UpdateHostCertificates`; the NULL `type` rows remain but won't be selected by the rolled-back renewal cron's strict `type` clause.

## Open Questions

- **Windows Subject substitution** (Phase 2): today's `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID` substitutes into the OMA-URI container path, not the cert Subject. For Windows renewal to use the same matching mechanism, either the customer profile must reuse `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the cert Subject (CertificateRequestBody/SubjectName CSP node), or a Windows-specific Subject variable is introduced. Engineering to confirm Windows profile authoring conventions.

- **Profile UUID format** (Phase 2): RESOLVED — empirically the marker is always literally `"fleet-" + profile_uuid` (substituted by `server/mdm/microsoft/profile_variables.go:125` and `server/mdm/apple/profile_processor.go:401-404`). No regex needed; substring search suffices.

- **Validation surface** (Phase 2): GitOps profile uploads (`fleetctl gitops`) should also enforce the marker requirement. Confirm `fleetctl` profile validation shares the same code path as UI upload.

- **Customer-facing redeploy guidance** (Phase 2): documentation must clearly explain that existing profiles need to be re-uploaded with the marker. Should the release-notes / guide include a `fleetctl` snippet customers can run to identify which existing profiles need updating?

- **Observability** (deferred): activity log entry on successful renewal vs. log-only? Out of scope per Non-Goals but worth tracking as a follow-up improvement.

- **Flow B — enrollment-time ACME cert detection** (gap in Phase 1 as currently implemented): PR 1.2 covers Flow A — profile-install ack triggers `CertificateList`. It does NOT cover Flow B — silicon Macs enrolling via DEP with `AppleRequireHardwareAttestation=true` get an ACME-issued enrollment cert that's invisible to osquery and never lands in `host_certificates` because no `InstallProfile` ack is involved in the enrollment ceremony (the cert is part of the enrollment profile itself). Closing this requires hooking `CertificateList` into either `mdmlifecycle.turnOnApple` (single-fire on first enrollment) or the `TokenUpdate` handler. Tracked as a follow-up sub-task; required to fully deliver #42827's customer promise for the cisneros-a use case.

- **Coordination with #44691** (Phase 2): PR #44691 restructures the matcher in `UpdateHostCertificates` (`toInsertBySHA1` map, pool-selection per hmmc row, best-match-wins, monotonic-forward predicate, `hmmcBackfillGrace`). It targets `main` and is expected to merge before our Phase 2 work lands. Our INSERT path (Decision 2.1) is conceptually orthogonal — `#44691` updates existing rows that got stuck NULL; we insert rows that don't exist yet. Implementation plan: land #44691 first, rebase Phase 2 work on top, add the INSERT loop alongside the restructured matcher rather than weaving into it. Reuse `toInsertBySHA1` / `incomingBySHA1` maps it already builds.
