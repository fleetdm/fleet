## ADDED Requirements

### Requirement: Auto-renewal for non-proxied ACME certs on Apple platforms

The system SHALL automatically renew non-proxied ACME certificates (issued by an external CA that Fleet does not proxy — e.g. the customer-cisneros-a Hydrant deployment) on macOS, iOS, and iPadOS hosts before expiration by re-pushing the delivering profile, when the cert was issued in response to a profile installed by Fleet and the cert's Subject contains a Fleet-generated renewal-ID marker.

The mechanism SHALL NOT depend on Fleet having a registered CA for the issuing service. Eligibility is determined by the presence of a `host_mdm_managed_certificates` row (created from ingestion when the marker is present), regardless of whether the row's `type` is set or NULL.

#### Scenario: Non-proxied ACME cert nears expiration on iOS

- **WHEN** an iOS host has an ACME cert (issued by an external CA Fleet does not proxy) linked to a Fleet-installed profile, the cert's `not_valid_after` falls within the renewal threshold (validity_period > 30 days → 30 days; ≤ 30 days → validity_period/2)
- **THEN** the system SHALL set the profile's status to NULL so the next profile-manager cron run triggers a re-push
- **AND** SHALL select the row from the renewal cron's NULL-`type` bucket without requiring a registered CA for the issuing service

#### Scenario: Non-proxied ACME cert nears expiration on macOS

- **WHEN** a macOS host has a hardware-bound non-proxied ACME cert (visible only via MDM `CertificateList`) linked to a Fleet-installed profile and the cert nears expiration
- **THEN** the system SHALL re-push the profile to trigger a new ACME exchange between the device and the external CA
- **AND** the new cert SHALL be ingested via the on-demand `CertificateList` triggered by the resulting profile-install ack (provided by #42827)
- **AND** the linked managed-cert row's `not_valid_after` SHALL be advanced when the new cert is ingested
- **AND** the matcher SHALL update the row even when its `type` is NULL — the existing `SupportsRenewalID()` skip MUST NOT exclude NULL-`type` rows

### Requirement: Auto-renewal for non-proxied SCEP certs (Okta)

The system SHALL automatically renew non-proxied SCEP certificates (Okta conditional access, Okta Verify w/ static challenge) on macOS, iOS, iPadOS, and Windows hosts using the same profile re-push mechanism as non-proxied ACME, when the cert's Subject contains the renewal-ID marker.

#### Scenario: Okta SCEP cert renewal on Windows

- **WHEN** a Windows host has an Okta-issued SCEP cert linked to a Fleet-installed profile and the cert nears expiration
- **THEN** the system SHALL set the profile's status to NULL to trigger a re-push
- **AND** the device SHALL re-execute SCEP enrollment with Okta directly without any Fleet proxy step
- **AND** the new cert SHALL be ingested via the next osquery `certificates` table report

#### Scenario: Okta Verify SCEP cert with static challenge on macOS

- **WHEN** a macOS host has an Okta-issued SCEP cert (using a static challenge embedded in the profile) linked to a Fleet-installed profile and the cert nears expiration
- **THEN** the system SHALL re-push the profile so the device re-runs SCEP using the same static challenge
- **AND** the new cert SHALL be ingested via osquery and the linked managed-cert row updated

### Requirement: Managed-cert row creation from cert ingestion

The system SHALL create `host_mdm_managed_certificates` rows when a cert is first ingested whose Subject contains a `fleet-<profile_uuid>` marker matching a profile installed on the same host, instead of requiring the row to be created at proxy issuance time.

#### Scenario: First-time cert ingestion creates managed-cert row

- **WHEN** the system ingests a cert via osquery or `CertificateList` whose Subject CN or OU contains the substring `fleet-<uuid>`, AND `<uuid>` resolves to a `profile_uuid` present in `host_mdm_apple_profiles` or `host_mdm_windows_profiles` for the same host, AND no `host_mdm_managed_certificates` row exists for that (host_uuid, profile_uuid) pair
- **THEN** the system SHALL insert a new `host_mdm_managed_certificates` row populated with the cert's `serial`, `not_valid_before`, and `not_valid_after`
- **AND** SHALL set `type` to NULL or to a non-proxied sentinel value
- **AND** SHALL leave `challenge_retrieved_at` NULL since no proxy step occurred

#### Scenario: Subsequent cert ingestion updates existing row

- **WHEN** the system ingests a renewed cert whose Subject marker matches an existing `host_mdm_managed_certificates` row
- **THEN** the system SHALL update that row's `serial` and `not_valid_after` from the new cert
- **AND** SHALL NOT create a duplicate row

#### Scenario: Cert with marker that does not resolve to a profile is ignored

- **WHEN** the system ingests a cert whose Subject contains `fleet-<uuid>` but `<uuid>` does not match any profile installed on the host (stale, copied, or wrong tenant)
- **THEN** the system SHALL NOT insert a `host_mdm_managed_certificates` row
- **AND** SHALL log the mismatch at debug level for troubleshooting

### Requirement: Profile-upload validation enforces renewal-ID marker

The system SHALL reject profile uploads (via UI, API, or `fleetctl gitops`) when the profile contains a SCEP or ACME payload whose cert Subject does not include a renewal-ID marker variable in CN or OU. Both `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` (preferred, customer-facing name per PR #44069) and the legacy `$FLEET_VAR_SCEP_RENEWAL_ID` SHALL be accepted as valid markers. Validation error messages SHALL reference only `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` to steer new authoring toward the preferred name.

#### Scenario: Apple ACME profile missing renewal-ID variable is rejected

- **WHEN** a user uploads an Apple configuration profile containing a `com.apple.security.acme` payload whose `Subject` field does not contain `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`
- **THEN** the system SHALL respond with a 422 Invalid Argument error
- **AND** the error message SHALL name `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` and explain that it is required in the cert Subject for the cert to auto-renew

#### Scenario: Apple SCEP profile missing renewal-ID variable is rejected

- **WHEN** a user uploads an Apple configuration profile containing a `com.apple.security.scep` payload whose `Subject` field does not contain `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`
- **THEN** the system SHALL respond with a 422 Invalid Argument error
- **AND** the error message SHALL name `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` and explain the requirement

#### Scenario: Profile without renewable payloads is unaffected

- **WHEN** a user uploads a profile with no SCEP or ACME payloads (e.g., a Wi-Fi profile, a restrictions profile)
- **THEN** the system SHALL NOT require `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` and SHALL accept the profile

#### Scenario: Legacy `SCEP_RENEWAL_ID` variable is still accepted

- **WHEN** a user uploads a profile whose cert Subject contains `$FLEET_VAR_SCEP_RENEWAL_ID` (the pre-rename variable name) in CN or OU
- **THEN** the system SHALL accept the profile (the legacy name remains valid for back-compat with profiles authored against pre-4.86 docs)
- **AND** SHALL substitute the same value (`fleet-<profile_uuid>`) as it would for `$FLEET_VAR_CERTIFICATE_RENEWAL_ID`

### Requirement: Renewal-ID variable substitution recognizes both names

The system SHALL recognize both `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` (preferred) and `$FLEET_VAR_SCEP_RENEWAL_ID` (legacy) when substituting renewal-ID markers into profile content. Both names SHALL substitute to the identical string `"fleet-" + profile_uuid`. The variable name has no semantic effect on the resulting cert Subject — only on what the profile author types.

#### Scenario: New variable substitutes to fleet-<profile_uuid>

- **WHEN** a profile contains `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` in a cert Subject field
- **THEN** the system SHALL replace it with `"fleet-" + profile_uuid` before delivering the profile to the device

#### Scenario: Legacy variable substitutes identically

- **WHEN** a profile contains `$FLEET_VAR_SCEP_RENEWAL_ID` in a cert Subject field
- **THEN** the system SHALL replace it with `"fleet-" + profile_uuid` — the same value the new variable produces
- **AND** the resulting cert Subject SHALL be indistinguishable on the wire from one authored with the new variable name

### Requirement: Profile redeploy activates renewal for existing fleet

The system SHALL rely on customer profile redeploy as the activation step that surfaces marker-bearing certs into ingestion, instead of running a deploy-time backfill. Profile redeploy is required because Phase 2 introduces the marker-in-Subject requirement that pre-existing profiles do not satisfy.

#### Scenario: Customer redeploys ACME profile with marker

- **WHEN** a customer adds `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` to an existing ACME profile's Subject and redeploys it via Fleet UI or `fleetctl gitops`
- **THEN** the resulting `InstallProfile` ack on each affected host SHALL trigger a `CertificateList` (Phase 1 mechanism)
- **AND** the new cert (issued with the marker in Subject) SHALL be ingested
- **AND** the `UpdateHostCertificates` insert path SHALL create the corresponding `host_mdm_managed_certificates` row
- **AND** the renewal cron SHALL pick up the row at the configured threshold

#### Scenario: Customer does not redeploy a profile

- **WHEN** a customer upgrades to the version with these changes but does not redeploy a particular ACME/SCEP profile
- **THEN** the system SHALL NOT auto-renew certs from that profile
- **AND** the system SHALL NOT silently degrade or alter behavior for that profile beyond the existing baseline

### Requirement: Non-proxied managed-cert rows are eligible for renewal

The system SHALL include `host_mdm_managed_certificates` rows with NULL `type` in the renewal cron's selection set, so they are processed alongside rows from proxied CAs. Hydrant ACME and other non-proxied flows SHALL NOT be modeled as registered CA types — the NULL bucket is the canonical mechanism for any external CA Fleet does not proxy.

#### Scenario: Renewal cron selects non-proxied row near expiration

- **WHEN** `RenewMDMManagedCertificates` runs and a row exists with `type` IS NULL, `not_valid_after` within the renewal threshold, and a non-NULL profile status
- **THEN** the system SHALL include the row in the rows it processes
- **AND** SHALL set the corresponding profile's status to NULL to trigger re-push
- **AND** SHALL match the row using a null-safe equal predicate so the same SQL handles registered and NULL `type` values

#### Scenario: Matcher advances NULL-`type` row after renewal

- **WHEN** a renewal completes for a NULL-`type` row, a fresh cert is ingested, and `UpdateHostCertificates` runs the matcher
- **THEN** the system SHALL advance the row's `not_valid_after` from the new cert
- **AND** SHALL NOT skip the row solely because its `type` is NULL or empty
