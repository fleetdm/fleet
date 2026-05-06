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

### Requirement: Matcher runs on every UpdateHostCertificates call

The matcher SHALL execute on every `UpdateHostCertificates` call, regardless of whether the call has any new certs to insert. This is necessary so that an `hmmc` row stuck NULL from an earlier missed match (replica lag, transaction race, cert landing in `existingBySHA1` rather than `toInsert`) can recover from a renewed cert that's already present in `host_certificates`.

#### Scenario: Stable cert inventory still triggers recovery

- **WHEN** a host reports the same cert inventory it reported on a prior call (no `toInsert`, no `toDelete`)
- **AND** an hmmc row for that host is stuck NULL with a renewed cert already present in `host_certificates`
- **THEN** the matcher SHALL still execute and queue an UPDATE on the stuck hmmc row

### Requirement: Matcher recovers stuck-NULL hmmc rows from prior reports

The matcher SHALL widen its cert-pool search beyond newly-inserted certs and use the full reported inventory whenever an hmmc row has `not_valid_after IS NULL`, `updated_at` older than the in-flight grace window (`hmmcBackfillGrace`), AND the related per-platform profile is in the `'verified'` delivery state. The `'failed'` state is excluded: SCEP delivery failures are terminal across the platform (admin must resend), and widening the pool on `'failed'` would re-populate hmmc from the OLD cert and re-arm the renewal cron into an hourly push loop.

#### Scenario: hmmc stuck NULL beyond grace window with verified profile recovers from full inventory

- **WHEN** an hmmc row has `not_valid_after IS NULL` and `updated_at < NOW() - hmmcBackfillGrace`
- **AND** the related host_mdm_apple_profiles or host_mdm_windows_profiles row's `status` is `'verified'`
- **AND** a cert in the host's currently-reported inventory matches the renewal-ID substring and is currently valid (`not_valid_before <= NOW <= not_valid_after`)
- **THEN** the matcher SHALL queue an UPDATE on the hmmc row from the matching cert

#### Scenario: hmmc still in flight (NULL but recently updated) is not widened

- **WHEN** an hmmc row has `not_valid_after IS NULL` and `updated_at >= NOW() - hmmcBackfillGrace`
- **THEN** the matcher SHALL only consider newly-inserted certs (`toInsert` scope), preserving today's "react to new cert" semantics
- **AND** the matcher SHALL NOT match against pre-existing certs in `host_certificates` from prior reports — preventing the OLD pre-renewal cert from clobbering the in-flight blank-out

#### Scenario: Pending or verifying profile blocks recovery even when stuck-by-time

- **WHEN** an hmmc row has `not_valid_after IS NULL` and `updated_at < NOW() - hmmcBackfillGrace`
- **AND** the related profile's `status` is `'pending'` or `'verifying'` (renewal still legitimately in flight)
- **THEN** the matcher SHALL only consider newly-inserted certs (`toInsert` scope)
- **AND** the matcher SHALL NOT widen to the full reported inventory

#### Scenario: Failed profile blocks recovery (terminal-on-failure contract)

- **WHEN** an hmmc row has `not_valid_after IS NULL` and `updated_at < NOW() - hmmcBackfillGrace`
- **AND** the related profile's `status` is `'failed'`
- **THEN** the matcher SHALL only consider newly-inserted certs (`toInsert` scope)
- **AND** the matcher SHALL NOT widen to the full reported inventory, mirroring the platform's "SCEP failure is terminal — admin must resend" semantics and preventing the OLD pre-renewal cert from re-arming the renewal cron into a push loop

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

### Requirement: Matcher loads hmmc rows with related profile status in a single query

The matcher SHALL load the host's hmmc rows in one SELECT that `LEFT JOIN`s `host_mdm_apple_profiles` and `host_mdm_windows_profiles` (filtered by `operation_type = 'install'`) so the per-platform delivery status is available without an extra query. Cert-pool selection and matching SHALL otherwise use only in-memory data already loaded by `UpdateHostCertificates` (the host's existing certs and the incoming reported certs).

#### Scenario: Stuck-row recovery does not issue per-row queries

- **WHEN** the matcher takes the wide-pool (recovery) path for a stuck hmmc row
- **THEN** no SELECTs are issued beyond the single joined hmmc-with-status query that runs once per `UpdateHostCertificates` call
