## Why

Fleet's renewal cron (`RenewMDMManagedCertificates`) automatically re-pushes profiles to renew expiring certs for proxied CAs (NDES SCEP, Custom SCEP Proxy, DigiCert, Smallstep) but does not work for non-proxied flows where the device talks directly to an external CA — non-proxied ACME and non-proxied SCEP (including Okta conditional access and Okta Verify w/ static challenge). For these, Fleet has no view of issuance, so no `host_mdm_managed_certificates` row gets created, so the renewal cron has nothing to act on. This blocks the 4.86 customer promise (#40639, customer-cisneros-a) where the customer deploys an ACME profile against their own Hydrant deployment (a private fork that supports ACME — Hydrant does not officially support ACME at the time of this writing) and needs those device certs to auto-renew before expiry to prevent service disruption (Wi-Fi, MDM identity).

For the customer-cisneros-a target on macOS, the ACME certs are hardware-bound and entirely invisible to osquery — they only appear via the MDM `CertificateList` command. Fleet uses this command on iOS/iPadOS today but not on macOS. So delivering the customer promise also requires extending MDM cert ingestion to macOS (#42827, also assigned to this work).

Non-goal of this change: model "Hydrant ACME" as a registerable CA type. The customer's Hydrant fork is private and the upstream Hydrant product does not yet officially support ACME, so a `hydrant` CA registration would not generalize to other customers. The non-proxied mechanism described below works for any ACME or SCEP server the customer's profile points at, regardless of vendor — Hydrant in this engagement, something else for the next customer. Once Hydrant ships official ACME support, a first-class Hydrant ACME CA registration may be added separately and is out of scope here.

This change covers both #42827 (cert ingestion) and #40639 (renewal). The two stories ship as independent phases because #42827 is independently valuable (admins can see ACME certs on the host details page even before renewal lands) and the dependency is one-directional: renewal of macOS hardware-bound certs cannot work until ingestion is in place.

## What Changes

### Phase 1 — #42827 cert ingestion (macOS)

- Extend MDM `CertificateList` command usage to macOS hosts. iOS/iPadOS already use it via `IOSiPadOSRefetch`.
- Trigger model: on-demand `CertificateList` when an ACME `InstallProfile` ack is received. NOT a recurring cadence — would generate too much MDM traffic across a full fleet.
- Ingest *all* MDM-delivered certs returned (not only ACME), deduplicating against osquery-ingested certs by `host_certificates.sha1_sum`.
- Add `host_certificates.origin` column (`osquery` / `mdm`) to scope deletion semantics so each ingestion source only soft-deletes rows it owns. Internal-only — not exposed in API.
- No deploy-time backfill: customers must re-deploy ACME/SCEP profiles to enable renewal (Phase 2 requires the marker variable in profile Subject). Redeploy → InstallProfile ack → on-demand `CertificateList` → cert ingested. The natural flow covers existing fleet without a separate backfill mechanism.

### Phase 2 — #40639 renewal extension

- Extend `UpdateHostCertificates` to *insert* `host_mdm_managed_certificates` rows when an ingested cert's Subject contains a `fleet-<profile_uuid>` marker but no matching managed-cert row exists. Today this function only updates existing rows.
- Extract the renewal ID marker from the cert Subject (CN or OU) and resolve it to a profile UUID. Verify the profile is installed on this host before inserting.
- Allow `host_mdm_managed_certificates.type` to be NULL for rows created from ingestion (no proxy step → no known CA type). The renewal cron then iterates the union of registered CA types plus a NULL bucket, so non-proxied rows are processed alongside proxied rows.
- Adjust the matcher in `UpdateHostCertificates` so the existing `SupportsRenewalID()` skip does not silently exclude NULL-`type` rows — without this, ingestion-created rows would never have their `not_valid_after` advanced after a renewal completes, producing a non-terminating renewal loop.
- Profile-authoring validation: reject SCEP/ACME profile uploads that lack `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the cert Subject. Without the marker, no renewal can ever fire. This validation also functions as the trigger for customers to redeploy existing profiles with the marker, which in turn surfaces certs into ingestion via the Phase 1 mechanism.
- No DB-side linkage backfill: marker-bearing certs only exist on devices after the customer redeploys profiles with `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the Subject. Redeploy triggers Phase 1 ingestion, which feeds Phase 2's natural insert path. No pre-existing marker-bearing certs to backfill.
- TODO (engineering confirmation): Windows scope — the current `$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID` substitutes into the OMA-URI container path, not the cert Subject. Either reuse `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in the Subject, or introduce a Windows-specific Subject variable.

## Capabilities

### New Capabilities

- `cert-ingestion-mdm`: MDM `CertificateList`-based ingestion of certs from Apple devices. Covers the trigger model (on-demand per `InstallProfile` ack, plus one-time backfill at deploy), unified storage in `host_certificates` with origin tracking, and dedup with osquery-ingested rows. Phase 1 of this change.
- `cert-renewal-non-proxied`: Auto-renewal of MDM-delivered SCEP and ACME certificates issued by external CAs that Fleet does not proxy (e.g. the customer's Hydrant deployment in customer-cisneros-a, Okta SCEP). Covers ingestion-driven managed-cert row creation, profile-authoring validation, platform coverage (macOS, iOS, iPadOS, Windows). Phase 2 of this change, depends on `cert-ingestion-mdm` being shipped for the macOS leg.

### Modified Capabilities

<!-- No existing specs in openspec/specs/ — all behavior is captured in the new capabilities above. -->

## Impact

### Phase 1 (#42827)

- **Code**:
  - `server/mdm/apple/apple_mdm.go` — extend on-demand `CertificateList` to macOS; today only iOS/iPadOS via `IOSiPadOSRefetch`.
  - `server/service/apple_mdm.go` — hook into the `InstallProfile` ack handler to trigger `CertificateList` for macOS ACME profile installs.
  - `server/datastore/mysql/host_certificates.go` — dedup by `sha1_sum`, source-aware soft-delete.
- **Schema**: add `host_certificates.origin` enum column (`osquery` / `mdm`); migration includes default value for existing rows.
- **API**: no surface changes — the API returns a single deduped cert list; `origin` is internal.
- **Customer-facing**: macOS ACME certs become visible on the host details page after the customer redeploys the delivering profile. No automatic backfill — visibility is conditional on profile activity, consistent with the redeploy requirement Phase 2 introduces.

### Phase 2 (#40639)

- **Code**:
  - `server/datastore/mysql/host_certificates.go` — `UpdateHostCertificates` extension (insert path); matcher adjustment for NULL-`type` rows.
  - `server/datastore/mysql/mdm.go` — extend `RenewMDMManagedCertificates` to iterate a NULL bucket alongside the registered CA types (null-safe equal `<=>`).
  - `server/service/apple_mdm.go` — profile fleet-variable validation (extends existing SCEP validation to ACME payloads).
  - `server/service/windows_mdm_profiles.go` — profile fleet-variable validation.
- **Schema**: `host_mdm_managed_certificates.type` becomes nullable (allow NULL, drop the `'ndes'` default).
- **API**: no surface changes — renewal happens in cron, no new endpoints.
- **Dependencies**: depends on Phase 1 for the macOS leg. iOS/iPadOS/Windows legs are independent.
- **Customer-facing**: Premium-only (consistent with existing renewal). Customers must include `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in profile Subject for renewal to function — documentation update required. Profile uploads missing the variable will be rejected.
