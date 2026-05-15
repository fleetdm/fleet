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
- Auto-renew non-proxied ACME (including the customer-cisneros-a Hydrant deployment) and non-proxied SCEP profile-delivered certs on Apple platforms (macOS post-Phase 1, iOS, iPadOS today) and Windows (Phase 2) — **for profiles that opt in by including the marker variable**.
- Reuse existing renewal cron unchanged. Reuse existing renewal threshold logic.
- Use a single mechanism (extracting `fleet-<profile_uuid>` marker from cert Subject) for all platforms and all CA types in scope.
- Make the marker opt-in: profiles that don't include it continue to work exactly as in 4.85 (manual redeployment for renewal). No upload-time enforcement, no GitOps breakage, no upgrade-day surprise.
- Keep Phase 1 and Phase 2 independently mergeable so each is reviewable in isolation.

**Non-Goals:**
- Custom EST Proxy renewal (deliberately deferred; no customer driver).
- New first-class CA type for Hydrant ACME (Hydrant does not officially support ACME yet; customer's private fork is the only deployment in scope; the non-proxied mechanism handles it without a CA registration).
- New first-class CA type for Okta SCEP (renewal works without one).
- **Forcing customers to add the marker.** Marker is opt-in; missing marker means no auto-renewal for that profile, not an error.
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

### Decision 1.2: No deploy-time backfill — profile activity is the trigger

We do NOT run a one-time `CertificateList` backfill across the macOS fleet at deploy. Rationale: ingestion is tied to profile activity. New ACME profile installs fire `CertificateList` via Decision 1.1. Customers who want auto-renewal opt in by redeploying a profile with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the cert Subject; that redeploy naturally fires the trigger and surfaces the marker-bearing cert. Customers who don't need auto-renewal can leave profiles untouched — those certs simply don't appear in `host_certificates` until something else surfaces them.

**Implications:**
- Customers who do not redeploy a particular ACME/SCEP profile see no certs from it in the host details page on macOS. Same outcome as today (unchanged behavior). No regression.
- Visibility on macOS becomes conditional on profile activity. Customers who want immediate visibility for compliance/audit reasons must take an explicit action (redeploy).
- Eliminates the rate-limiting / resumption / runbook complexity that a fleet-wide backfill would introduce.

**Alternatives considered:**
- *One-time `CertificateList` backfill across the fleet.* Rejected: the redeploy flow (when customers opt into auto-renewal) already surfaces the relevant certs. Customers who don't redeploy explicitly opted out of auto-renewal; backfilling their certs doesn't help them. Adds engineering and operational complexity (rate-limiting, completion tracking, resume-on-restart, monitoring) for a transparency benefit that is shorter-lived than the redeploy timeline.
- *Synchronously backfill all hosts at first deploy.* Rejected for the same reason plus the thundering-herd problem on APNs / the MDM command queue.

### Decision 1.3: Unified storage with `origin` column

Store certs from both osquery and MDM `CertificateList` ingestion in `host_certificates`, deduplicated by `sha1_sum`. Add an `origin` column (`osquery` / `mdm`) so each ingestion source only soft-deletes rows it owns. The column is internal — not exposed in the public API.

**Alternatives considered:**
- *Separate `host_mdm_certificates` table.* Rejected: complicates the host details API by requiring a UNION across two tables. Single-table-with-origin is simpler.
- *Expose `origin` in the API.* Rejected per discussion — clients only care about the unified deduped list.

### Decision 1.4: Osquery sync downgrades existing `mdm` rows to `osquery`

Refinement of Decision 1.3 surfaced during 2026-05-14 local validation. Today the `origin` column reflects "first ingestion source to see this cert." That's misleading: MDM `CertificateList` returns the entire device keychain on every InstallProfile ack, including Root CAs and other system certs the user installed manually. If MDM observes a non-MDM-delivered cert before the next osquery cert sync, the row gets `origin='mdm'` even though Fleet did not deliver it.

**Decision:** when osquery sync UPSERTs a cert that already exists with `origin='mdm'`, downgrade `origin` to `'osquery'`. One-way only (mdm → osquery, never the reverse).

**Why this is correct:**
- Osquery sees a strict superset of what MDM `CertificateList` sees (the entire keychain, unfiltered). If osquery observes a cert, the device has it in the keychain by whatever path; the cert's presence is not evidence of MDM delivery.
- MDM `CertificateList` observation is *not* sufficient evidence of MDM delivery, because the protocol returns every keychain entry, not just Fleet-delivered ones. The current MDM-only path remains the right one for first-seen MDM-delivered certs (ACME/SCEP) because they appear in the keychain only after the MDM-driven exchange.
- The matcher path (PR 2.2) keys on the `fleet-<profile_uuid>` marker in the cert Subject, not on `origin`, so renewal correctness is unaffected. `origin` is purely for visibility / audit semantics.

**Concretely:** the live row for a system cert observed by both sources lands as `origin='osquery'`. The live row for an ACME cert observed only by MDM (because hardware-bound certs are invisible to osquery on macOS) stays `origin='mdm'`. That matches intent: "did Fleet deliver this cert?"

**Alternatives considered:**
- *Filter what `CertificateList` ingests* — only persist certs that match an installed cert-issuing profile on the host. Rejected: requires inspecting profile contents during ingestion (currently we don't), creates a tight coupling between the ingestion path and the profile-matching logic, and loses visibility of MDM-delivered certs whose profile got removed but cert was retained. The downgrade approach keeps the ingestion path simple and lets matcher logic handle linkage separately.
- *Three-valued `origin` column (`osquery` / `mdm` / `both`).* Rejected: doesn't add information — `both` would mean "observed by both," which is equivalent to "osquery saw it" since osquery is the superset. And it'd require enum migration plus a third soft-delete branch.
- *Backfill existing rows on deploy.* Rejected: the next natural osquery sync downgrades affected rows. Adding a deploy-time scan is unnecessary churn for a slowly-self-correcting issue.

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

### Decision 2.5: No DB-side linkage backfill — opt-in redeploy provides the marker-bearing certs

We do NOT run a DB-side linkage backfill at Phase 2 deploy. Rationale: marker-bearing certs do not exist on devices until a customer opts into auto-renewal by redeploying a profile with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the Subject. Before that opt-in, existing certs lack the marker — there is nothing to link.

When the customer redeploys, the new ACME/SCEP exchange produces a cert with the marker in its Subject. Phase 1's on-demand `CertificateList` ingests it. Phase 2's `UpdateHostCertificates` insert path (Decision 2.1) creates the `host_mdm_managed_certificates` row. The renewal cron then picks it up at threshold.

Customers who don't include the marker in their profiles simply don't get auto-renewal — same behavior as 4.85 and earlier. No linkage row gets created; the renewal cron has nothing to act on.

**Alternatives considered:**
- *Backfill scan over `host_certificates`.* Rejected: pre-opt-in certs don't carry the marker, so scanning them yields nothing useful. Post-opt-in certs are linked by the natural insert path. The backfill would be a no-op in both regimes.
- *Re-run Phase 1's `CertificateList` backfill after Phase 2 ships.* Same problem — pre-opt-in certs lack the marker. Even if visible in `host_certificates`, they wouldn't link. And Phase 1's backfill itself was dropped (Decision 1.2).

### Decision 2.6: The marker is optional — no profile-upload validation for missing marker

`$FLEET_VAR_CERTIFICATE_RENEWAL_ID` is an **opt-in renewal activator**, not a required field. Profile uploads succeed regardless of whether the marker is present. Omitting it just means auto-renewal doesn't activate for that profile — Fleet falls back to today's manual-redeployment workflow for cert lifecycle, identical to 4.85 behavior.

**Net-new validators (Apple ACME, Apple raw SCEP) are removed entirely.** PR 2.3 and PR 2.3b initially shipped validators that rejected profiles missing the marker; those validators are reverted under this scope. The marker is now purely advisory.

**Pre-existing validators (proxy SCEP — NDES / Custom SCEP / Smallstep, Windows non-proxied SCEP) keep their existing renewal-ID enforcement.** Those validators predate this story (since 4.65) and customers have authored profiles against them for years. Loosening them would have no benefit and might silently flip working renewal off.

The matcher (`host_certificates.go:195`) keeps its defensive CN-or-OU search for `fleet-<profile_uuid>`. If a customer puts the marker in CN instead of OU, the matcher may still find it depending on CA cooperation — no upload-time enforcement gets in the way.

**Why this changed (scope reversal, 2026-05-15):**

Validators initially rejected missing marker on the theory that "silent non-renewal is the failure mode this story exists to prevent." But that framing conflated two different customer scenarios:

1. **Customer wants auto-renewal.** They add the marker. Today's behavior (with or without validators) gives them auto-renewal.
2. **Customer doesn't want auto-renewal, or doesn't know it's a feature.** They don't add the marker. Today's behavior continues to work — they renew manually as they always have.

Hard-rejecting case 2 breaks existing deployments. Specifically:

- **GitOps trap:** A customer with existing Conditional Access / Okta Verify / ACME profiles in their GitOps spec runs `fleetctl gitops apply` after upgrading to 4.86. The profiles haven't changed. But upload now fails because the validators reject missing markers. Their next sync breaks.
- **UI-edit trap:** A customer opens an existing profile in the UI, makes an unrelated edit (rename, label change), saves. The validator now rejects, breaking the edit.
- **Conditional Access:** Fleet's own generated profile lacks the marker (#45580 surfaced this gap). Customers following the published guide can't deploy.

The cost of hard rejection — broken upgrades for the customer base that doesn't even use auto-renewal — exceeded the benefit of catching the misconfiguration that's only relevant to customers who opted in. Marker stays as a documented variable; renewal activates when present; everything else continues unchanged.

**Alternatives considered:**
- *Reject missing marker (original design).* Rejected: GitOps continuity and upgrade-day breakage outweigh the silent-failure protection. Most customers don't even use auto-renewal; rejecting their existing profiles is a regression.
- *Accept missing marker but reject misplaced marker (e.g., marker in CN instead of OU).* Considered. Modest benefit (catches typos for opted-in customers) at the cost of a partial validator that's harder to explain. Rejected for simplicity. The matcher's CN-or-OU search means misplaced markers may actually still work in practice; upload-time validation isn't necessary.
- *Soft warning at upload time.* Considered. No UI surface to display a non-blocking warning today; activity log entries get lost. Discoverability worse than just documenting the requirement in the guide.

### Decision 2.7: Variable rename — accept both `SCEP_RENEWAL_ID` and `CERTIFICATE_RENEWAL_ID`

The customer-facing variable name was renamed from `$FLEET_VAR_SCEP_RENEWAL_ID` to `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` per PR #44069 (merged 2026-05-01 into `docs-v4.86.0`), referenced by #40639 as "[New variable name]". Reasoning: the variable is no longer SCEP-specific — it must apply to ACME profiles too for the customer-cisneros-a use case (and any future non-proxied flow), so the SCEP-prefixed name reads as a bug to anyone authoring an ACME profile.

Today (4.85 and earlier) the only valid name is `SCEP_RENEWAL_ID`. The rename is half-shipped: docs use the new name, code still defines and substitutes only the old name. Phase 2 closes that gap.

**Implementation approach:** add `FleetVarCertificateRenewalID = "CERTIFICATE_RENEWAL_ID"` alongside the existing `FleetVarSCEPRenewalID`. Both are recognized by `FindFleetVariables`. Both substitute to the same value (`"fleet-" + ProfileUUID`) via the same substitution helper — implemented as a single regex matching either name.

**Substitution accepts both names everywhere.** The substitution code (`server/mdm/apple/profile_processor.go`, `server/mdm/microsoft/profile_variables.go`) uses the unified `FleetVarRenewalIDRegexp` so a profile authored with either name produces identical `fleet-<profile_uuid>` output at delivery time. New authoring should use the preferred name per documentation, but legacy name continues to work — no validation enforcement either way (see Decision 2.6).

**Pre-existing proxy-SCEP validators retain their existing behavior** (NDES / Custom SCEP / Smallstep, Windows non-proxied SCEP). Those validators predate this story (since 4.65) and continue to require the renewal-ID variable in their authoring patterns. The variable rename adds preferred-name acceptance on those surfaces (PR 2.3 / PR 2.4 / PR 2.5) without dropping legacy-name back-compat. Customers running 4.85 SCEP profiles continue to work; new authoring can use either name.

**Why accept both rather than hard-rename:**
- Customers running 4.85 likely have `$FLEET_VAR_SCEP_RENEWAL_ID` in deployed SCEP profile Subjects. Hard-renaming the substitution constant would break those profiles on the next upload/edit cycle (substitution would leave the literal `$FLEET_VAR_SCEP_RENEWAL_ID` string in the profile, which the device CA rejects).
- Backwards-compat cost is small: one extra regex alternation, one extra constant. No new datastore work, no migration.
- We can deprecate `SCEP_RENEWAL_ID` in a later release once telemetry shows the long tail has migrated.

**Alternatives considered:**
- *Hard-rename — only `CERTIFICATE_RENEWAL_ID` works.* Rejected per the customer-impact reasoning above.
- *Hard-rename with a deprecation warning on uploads using the old name.* Considered. Same back-compat issue for already-deployed profiles that don't get re-uploaded — they'd silently stop substituting. Reject for the same reason.
- *Keep `SCEP_RENEWAL_ID` as the only name and revert the docs PR.* Rejected: docs decision is product-led and reflects the variable's actual scope (any cert, not just SCEP). The mismatch is a code-side gap to close, not a docs error to revert.

**Historical note (2026-05-15 scope change):** earlier drafts of this Decision included a "net-new surfaces accept preferred-name-only and require OU placement" rule for the Apple ACME validator (PR 2.3) and Apple raw-SCEP validator (PR 2.3b). Those validators were removed entirely under Decision 2.6's scope reversal, so the net-new-vs-pre-existing distinction no longer has enforcement behavior to mediate. The matcher accepts the marker in CN or OU regardless. Customers who include the marker should use the preferred name in OU per documentation, but no validator enforces either choice.

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
