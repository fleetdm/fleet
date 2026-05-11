## ADDED Requirements

### Requirement: MDM CertificateList ingestion on macOS

The system SHALL collect installed certificates from MDM-enrolled macOS hosts using the Apple `CertificateList` MDM command, in addition to the existing osquery-based ingestion, so that hardware-bound certificates invisible to osquery (notably hardware-bound ACME identities) are surfaced in Fleet.

#### Scenario: macOS ACME profile install triggers cert ingestion

- **WHEN** an MDM-enrolled macOS host installs a profile containing a `com.apple.security.acme` payload and acks the `InstallProfile` command
- **THEN** the system SHALL queue a `CertificateList` MDM command for that host
- **AND** the resulting cert list SHALL be ingested into `host_certificates`

#### Scenario: macOS does NOT use a recurring CertificateList cadence

- **WHEN** the system processes its scheduled MDM cron jobs on macOS hosts
- **THEN** the system SHALL NOT enqueue `CertificateList` on a recurring cadence
- **AND** SHALL rely on `InstallProfile`-ack triggers (per Decision 1.1) for macOS ingestion. There is no deploy-time backfill (per Decision 1.2 — the customer redeploy step that activates Phase 2 also produces the `InstallProfile` acks that drive Phase 1 ingestion).

#### Scenario: Multiple ACME profile acks in flight do not dedupe the trigger

- **WHEN** an `InstallProfile` ack arrives for an ACME profile on a macOS host while a previous `CertificateList` refetch for the same host is still pending
- **THEN** the system SHALL still enqueue a new `CertificateList` for the new ack, not skip it because of the in-flight refetch
- **AND** the duplicate enqueue SHALL collapse via the `host_mdm_commands` `(host_id, command_type)` primary key (`ON DUPLICATE KEY UPDATE`)
- **AND** when both refetch results eventually arrive, the result handler SHALL process each idempotently (the second is safe to receive after the first removes the tracking row)
- **AND** this guarantees the post-ACME-exchange cert state is captured even if the prior refetch returned before the device's ACME exchange completed

#### Scenario: iOS and iPadOS continue using existing refetch cadence

- **WHEN** the existing `IOSiPadOSRefetch` cron runs against iOS or iPadOS hosts
- **THEN** the system SHALL continue to enqueue `CertificateList` on its established hourly cadence without change

### Requirement: Unified cert storage with origin tracking

The system SHALL store certs from both osquery and MDM `CertificateList` ingestion in a single `host_certificates` table, deduplicated by `sha1_sum`, with an `origin` column scoping deletion semantics per ingestion source.

#### Scenario: Cert visible to both osquery and MDM appears once

- **WHEN** the same cert is reported by both osquery's `certificates` table and MDM `CertificateList` for the same host
- **THEN** the system SHALL store exactly one row keyed by `(host_id, sha1_sum)`

#### Scenario: Hardware-bound cert visible only via MDM is stored as origin=mdm

- **WHEN** an MDM `CertificateList` response includes a hardware-bound cert that osquery cannot see
- **THEN** the system SHALL insert a row with `origin = 'mdm'`
- **AND** subsequent osquery sync runs SHALL NOT delete that row

#### Scenario: Origin is not exposed in the public API

- **WHEN** a client requests the host certificates API
- **THEN** the response SHALL be a single deduped list of certs without exposing the `origin` column
- **AND** the column SHALL exist purely as an internal implementation concern

### Requirement: Source-scoped deletion

The system SHALL only soft-delete `host_certificates` rows whose `origin` matches the ingestion source that is reporting the latest cert state, so each source cannot remove rows owned by another source.

#### Scenario: Osquery sync does not delete MDM-only certs

- **WHEN** osquery reports a fresh cert list that omits a hardware-bound MDM-ingested cert
- **THEN** the system SHALL NOT mark the MDM-origin cert as deleted

#### Scenario: MDM CertificateList response does not delete osquery certs

- **WHEN** an MDM `CertificateList` response omits a software cert that osquery is reporting
- **THEN** the system SHALL NOT mark the osquery-origin cert as deleted
