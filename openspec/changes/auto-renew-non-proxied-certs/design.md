## Context

Fleet's existing renewal pipeline assumes the server is in the cert-issuance path. For Fleet-issued MDM enrollment certs, `RenewSCEPCertificates` reads `nano_cert_auth_associations`. For profile-delivered certs from CAs that Fleet proxies (NDES, Custom SCEP Proxy, DigiCert, Smallstep), `RenewMDMManagedCertificates` reads `host_mdm_managed_certificates` rows that the proxy step populates at issuance.

For Hydrant ACME and non-proxied SCEP (Okta conditional access, Okta Verify), Fleet is not in the issuance path. The device performs the cert exchange directly with the CA. Fleet first sees the cert when it's reported via osquery (software certs) or the MDM `CertificateList` command (hardware-bound certs on Apple platforms). Today on macOS the `CertificateList` command is not used at all — only iOS/iPadOS use it via `IOSiPadOSRefetch`. So hardware-bound ACME certs on macOS are effectively invisible to Fleet, and even for software certs no `host_mdm_managed_certificates` row exists for the non-proxied flows so the renewal cron has nothing to act on.

This change addresses both halves in two phases that ship as independent PR sequences:

- **Phase 1 (#42827)** — extend MDM `CertificateList` ingestion to macOS. Self-contained, customer-visible (certs appear on host details page), independently shippable.
- **Phase 2 (#40639)** — extend cert ingestion to also create `host_mdm_managed_certificates` rows so the existing renewal cron activates for non-proxied flows. Depends on Phase 1 for the macOS leg, otherwise independent (iOS/iPadOS already have `CertificateList` cadence; software-cert SCEP/ACME on macOS/Windows uses osquery ingestion).

## Goals / Non-Goals

**Goals:**
- Make MDM-delivered certs visible on the host details page on macOS (Phase 1).
- Auto-renew Hydrant ACME and non-proxied SCEP profile-delivered certs on Apple platforms (macOS post-Phase 1, iOS, iPadOS today) and Windows (Phase 2).
- Reuse existing renewal cron unchanged. Reuse existing renewal threshold logic.
- Use a single mechanism (extracting `fleet-<profile_uuid>` marker from cert Subject) for all platforms and all CA types in scope.
- Validate at profile upload that renewable certs include the marker, so silent non-renewal becomes impossible to misconfigure.
- Keep Phase 1 and Phase 2 independently mergeable so each is reviewable in isolation.

**Non-Goals:**
- Custom EST Proxy renewal (deliberately deferred; no customer driver).
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

**Implementation refinement (PR 1.2 final shape):** The InstallProfile-ack handler runs on every MDM command result, so the trigger gate lives in the hot path. Final implementation pushes all gating server-side: a single indexed query (`ProfileHasACMEPayloadForCommand`) returns host platform, profile UUID, ACME-payload presence (computed via `LOCATE` on the `mobileconfig` blob), and pending-refetch state (computed via `EXISTS` on the `host_mdm_commands` primary key). The common case (non-darwin or non-ACME or already-pending) early-returns without parsing the profile or transferring the blob to Go. Tracking row insertion happens AFTER `commander.CertificateList` succeeds — matching the iOS/iPadOS `IOSiPadOSRefetch` pattern — so a commander failure doesn't leave a stale tracking row that would suppress future triggers.

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

### Decision 2.2: `host_mdm_managed_certificates.type` becomes nullable (or carries a sentinel)

For ingestion-created rows we don't know the CA type — Fleet wasn't in the issuance path. Either allow `type` to be NULL or add a `non_proxied` enum value. The renewal cron's `WHERE hmmc.type = ?` clause iterates `ListCATypesWithRenewalSupport()` and would skip these rows; either include the sentinel in the supported list, or change the cron to `WHERE (hmmc.type IN (...) OR hmmc.type IS NULL)`.

**Alternatives considered:**
- *Synthesize a type at insert time by inspecting the profile content* (e.g., is it `com.apple.security.acme`? a SCEP payload pointing at a registered Hydrant URL?). Rejected: brittle, leaks profile-content awareness into the datastore layer, and gains nothing — the renewal cron only cares about whether a row should be considered, not what produced it.

### Decision 2.3: Marker extraction via regex on Subject CN/OU

Inverted matching loop. Today: for each row, check ingested certs for `fleet-<row.profile_uuid>`. New: for each ingested cert, regex-extract `fleet-<uuid>` from Subject CN/OU and look up the profile by UUID.

Constraints on the extracted UUID:
- Must match Fleet's `profile_uuid` format (varchar(37), typically standard UUID with dashes — verify before regex finalization).
- Looked-up profile must exist in `host_mdm_apple_profiles` or `host_mdm_windows_profiles` for this host. Otherwise the cert is from a stale or copied profile UUID and is ignored.

**Alternatives considered:**
- *Match by Issuer + Serial recorded at issuance.* Rejected: only works for proxied flows where Fleet saw the issuance. Useless here.
- *Match by Subject == host identifier.* Rejected: breaks when one host has multiple profiles using the same CA.

### Decision 2.4: Add `hydrant` to renewal-supported CA list

`ListCATypesWithRenewalSupport()` and `ListCATypesWithRenewalIDSupport()` in `server/fleet/certificate_authorities.go` use legacy `CAConfigAssetType`. Hydrant is currently only in the newer `CAType` enum. Either add a `CAConfigHydrant` legacy constant, or migrate the renewal lists to use `CAType` directly. Decision: add the legacy constant for symmetry, leave the `TODO HCA` rewrite for later — minimum-change approach.

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

## Phase Independence

The expected ship cadence is "both phases together" — both deliver value to the same customer-cisneros-a use case and the redeploy step that activates Phase 2 is also what surfaces certs into Phase 1's ingestion. Each phase is still independently mergeable to keep PRs reviewable in isolation.

Phase 1 alone (if Phase 2 is delayed): macOS ACME certs become visible after the customer reinstalls the delivering profile. No renewal yet.

Phase 2 alone (if Phase 1 is delayed): iOS/iPadOS Hydrant ACME renewal works (existing `IOSiPadOSRefetch` cadence ingests the certs after redeploy), Windows Okta SCEP renewal works (osquery ingests), macOS Okta SCEP works (osquery ingests software certs after redeploy). Only macOS hardware-bound Hydrant ACME requires Phase 1.

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

- **Profile UUID format** (Phase 2): confirm by sampling that all profile UUIDs across `host_mdm_apple_profiles` / `host_mdm_windows_profiles` use the same shape, to finalize the extraction regex.

- **Validation surface** (Phase 2): GitOps profile uploads (`fleetctl gitops`) should also enforce the marker requirement. Confirm `fleetctl` profile validation shares the same code path as UI upload.

- **Customer-facing redeploy guidance** (Phase 2): documentation must clearly explain that existing profiles need to be re-uploaded with the marker. Should the release-notes / guide include a `fleetctl` snippet customers can run to identify which existing profiles need updating?

- **Observability** (deferred): activity log entry on successful renewal vs. log-only? Out of scope per Non-Goals but worth tracking as a follow-up improvement.

- **Flow B — enrollment-time ACME cert detection** (gap in Phase 1 as currently implemented): PR 1.2 covers Flow A — profile-install ack triggers `CertificateList`. It does NOT cover Flow B — silicon Macs enrolling via DEP with `AppleRequireHardwareAttestation=true` get an ACME-issued enrollment cert that's invisible to osquery and never lands in `host_certificates` because no `InstallProfile` ack is involved in the enrollment ceremony (the cert is part of the enrollment profile itself). Closing this requires hooking `CertificateList` into either `mdmlifecycle.turnOnApple` (single-fire on first enrollment) or the `TokenUpdate` handler. Tracked as a follow-up sub-task; required to fully deliver #42827's customer promise for the cisneros-a use case.

- **Coordination with #44691** (Phase 2): PR #44691 restructures the matcher in `UpdateHostCertificates` (`toInsertBySHA1` map, pool-selection per hmmc row, best-match-wins, monotonic-forward predicate, `hmmcBackfillGrace`). It targets `main` and is expected to merge before our Phase 2 work lands. Our INSERT path (Decision 2.1) is conceptually orthogonal — `#44691` updates existing rows that got stuck NULL; we insert rows that don't exist yet. Implementation plan: land #44691 first, rebase Phase 2 work on top, add the INSERT loop alongside the restructured matcher rather than weaving into it. Reuse `toInsertBySHA1` / `incomingBySHA1` maps it already builds.
