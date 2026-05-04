## ADDED Requirements

### Requirement: Matcher updates hmmc when host reports new SCEP cert

When `UpdateHostCertificates` ingests cert data from a host (osquery cert refetch on macOS / Windows, MDM `CertificateList` response on iOS), the matcher SHALL update `host_mdm_managed_certificates` (hmmc) for any managed-cert profile whose renewal-ID substring (`"fleet-" + profile_uuid`) appears in a newly-inserted cert's `subject_common_name` or `subject_org_unit`.

The matcher SHALL only consider hmmc rows whose `type` supports renewal IDs (custom SCEP proxy, NDES, Smallstep — DigiCert is excluded because Fleet writes its hmmc directly at issuance).

#### Scenario: New SCEP cert reported, matcher writes fresh hmmc values

- **WHEN** the host's cert ingest call inserts a new cert into `host_certificates` whose subject contains `"fleet-" + profile_uuid` for an existing hmmc row
- **THEN** the matcher SHALL queue an UPDATE on that hmmc row setting `not_valid_before`, `not_valid_after`, and `serial` from the matching cert
- **AND** the UPDATE SHALL run inside the same transaction as the cert insert

#### Scenario: DigiCert hmmc rows are not touched by the matcher

- **WHEN** an hmmc row has `type = digicert`
- **THEN** the matcher SHALL skip it regardless of whether any reported cert matches the renewal-ID substring

### Requirement: Matcher recovers stuck-NULL hmmc rows from prior reports

The matcher SHALL widen its cert-pool search beyond newly-inserted certs and use the full reported inventory whenever an hmmc row has been NULL for longer than the in-flight grace window (`hmmcBackfillGrace`). This recovers rows where an earlier matcher run failed to link the new cert to hmmc (replica lag, a transaction race, or the cert landing in `existingBySHA1` rather than `toInsert`). Without this widening, the renewal cron's `HAVING validity_period IS NOT NULL` lock permanently excludes the row and the only recovery is admin re-push.

#### Scenario: hmmc stuck NULL beyond grace window recovers from full inventory

- **WHEN** an hmmc row has `not_valid_after IS NULL` and `updated_at < NOW() - hmmcBackfillGrace`
- **AND** a cert in the host's currently-reported inventory (whether newly inserted or already in `host_certificates` from a prior call) matches the renewal-ID substring and is currently valid (`not_valid_before <= NOW <= not_valid_after`)
- **THEN** the matcher SHALL queue an UPDATE on the hmmc row from the matching cert

#### Scenario: hmmc still in flight (NULL but recently updated) is not touched

- **WHEN** an hmmc row has `not_valid_after IS NULL` and `updated_at >= NOW() - hmmcBackfillGrace`
- **THEN** the matcher SHALL only consider newly-inserted certs (`toInsert` scope), preserving today's "react to new cert" semantics
- **AND** the matcher SHALL NOT match against pre-existing certs in `host_certificates` from prior reports — preventing the OLD pre-renewal cert from clobbering the in-flight blank-out

### Requirement: Matcher picks the most recently issued matching cert when multiple match

When the matcher's selected cert pool contains more than one cert whose subject matches the renewal-ID substring (e.g., the OLD pre-renewal cert and the NEW renewed cert are both reported in the same osquery cycle), the matcher SHALL choose the cert with the latest `not_valid_before` among currently-valid candidates.

#### Scenario: Tie-breaker prefers the freshest cert

- **WHEN** two certs in the host's reported inventory both match the renewal-ID substring for an hmmc row
- **AND** both are currently valid
- **THEN** the matcher SHALL update hmmc with values from the cert whose `not_valid_before` is the latest

### Requirement: Matcher never regresses hmmc with an older cert

The matcher SHALL apply a monotonic-forward check before queuing any UPDATE: if the chosen cert's `not_valid_after` is not strictly later than the existing hmmc `not_valid_after`, the matcher SHALL skip the UPDATE.

#### Scenario: Already-fresh hmmc row is not regressed by an older matching cert

- **WHEN** an hmmc row already has `not_valid_after` populated
- **AND** the cert the matcher would otherwise pick has `not_valid_after` not strictly later than the hmmc value
- **THEN** the matcher SHALL skip the UPDATE for this row

### Requirement: Matcher reuses the existing per-host cert listing

The matcher SHALL NOT issue any new SELECTs beyond `ListHostMDMManagedCertificates(ctx, hostUUID)`, which is already executed by `UpdateHostCertificates` when the host has cert changes to ingest. All cert-pool selection and matching SHALL use in-memory data already loaded by `UpdateHostCertificates` (the host's existing certs and the incoming reported certs).

#### Scenario: Recovery path issues no extra database queries

- **WHEN** the matcher takes the wide-pool (recovery) path for a stuck hmmc row
- **THEN** no SELECTs are issued beyond what the steady-state matcher path already issues
