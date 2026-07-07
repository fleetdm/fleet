## ADDED Requirements

### Requirement: Proxied Windows SCEP profiles do not verify on ACK alone

A proxied Windows SCEP profile SHALL be set to `verifying`, not `verified`, on a successful (2xx) ACK of its SyncML
`<Exec>` (`Install/Enroll`) command. This applies to profiles that request a certificate through a Fleet SCEP proxy CA
whose certificates carry a matchable renewal-ID marker (CA type `custom_scep_proxy`, `ndes`, or `smallstep`).

Profiles that do not request a proxied certificate with a renewal-ID marker (non-certificate Windows profiles, and
server-issued CAs such as DigiCert) SHALL retain their existing behavior and MAY move directly to `verified` on ACK.

#### Scenario: Proxied SCEP profile ACKed by device
- **WHEN** a Windows host acknowledges the SyncML `<Exec>` for a SCEP profile backed by a `custom_scep_proxy`, `ndes`, or
  `smallstep` CA with a 2xx status
- **THEN** the profile's `host_mdm_windows_profiles.status` is set to `verifying`, not `verified`

#### Scenario: Non-certificate Windows profile ACKed by device
- **WHEN** a Windows host acknowledges the SyncML command for a profile that is not a proxied SCEP certificate profile
  with a 2xx status
- **THEN** the profile's status is unchanged from today and reaches `verified`

### Requirement: Profile verifies when the matching certificate is observed

The system SHALL transition a `verifying` proxied Windows SCEP profile to `verified` when it ingests a host certificate
whose Subject common name (CN) or organizational unit (OU) contains the profile's renewal-ID marker `fleet-<profile_uuid>`
and whose validity window includes the current time. This transition SHALL occur within the host-certificate ingestion
path so it takes effect on the host's next detail report, including an operator-triggered Refetch.

#### Scenario: Matching certificate reported by host
- **WHEN** a host in `verifying` for a proxied SCEP profile reports a certificate whose OU or CN contains
  `fleet-<profile_uuid>` and the certificate is currently valid
- **THEN** the profile transitions to `verified`

#### Scenario: Operator refetches a host after certificate issuance
- **WHEN** an operator triggers Refetch on the host and the forced detail query returns the matching certificate
- **THEN** the profile transitions to `verified` on that report without waiting for the next scheduled detail interval

### Requirement: Unconfirmed profiles remain verifying and are never failed on absence

The system SHALL leave a proxied Windows SCEP profile in `verifying` for as long as the matching certificate has not been
observed. Absence of an observed certificate SHALL NOT, by itself, cause the profile to be marked `failed`, because
absence is ambiguous (not yet issued, host offline, certificate store not readable, or agent unable to enumerate
certificates). No time-based verification timeout SHALL fail a proxied Windows SCEP profile.

#### Scenario: Host offline for an extended period
- **WHEN** a host does not report host details for multiple days after receiving a proxied SCEP profile
- **THEN** the profile remains `verifying` and is not marked `failed`, and it verifies on the host's next report if the
  certificate is present

#### Scenario: Agent cannot enumerate certificates
- **WHEN** the host runs an osquery version that cannot report the certificate subject/issuer detail fields required for
  matching, so no certificate is ingested
- **THEN** the profile remains `verifying` indefinitely and is not marked `failed`

#### Scenario: Host reports no personal certificates
- **WHEN** the host's certificate report contains no certificates to match against
- **THEN** the profile remains `verifying` and no status change occurs

### Requirement: User-scoped profiles verify only after the target user logs in

A user-scoped Windows SCEP profile (`./User/...`) SHALL remain `verifying` until the target user logs in and the matching
certificate is reported, because the agent can enumerate that user's certificate store only while the user is logged in.
If the user never logs in, the profile SHALL remain `verifying` indefinitely. A certificate previously observed and
verified for a user SHALL NOT be removed while that user is logged off.

#### Scenario: Target user not logged in
- **WHEN** a `./User/...` SCEP profile has been delivered but the target user has not logged in, so their certificate
  store cannot be read
- **THEN** the profile remains `verifying`

#### Scenario: Target user logs in and certificate is present
- **WHEN** the target user logs in and the host reports the matching certificate from that user's store
- **THEN** the profile transitions to `verified`

### Requirement: Proxy-observed upstream errors mark the profile failed

Fleet SHALL set a proxied Windows SCEP profile to `failed` when its SCEP proxy directly observes an upstream
certificate-authority error while handling `GetCACaps`, `GetCACert`, or `PKIOperation` for that `(host, profile)`, and
SHALL record a `detail` string identifying the SCEP operation and the upstream status. If the device subsequently retries
the SCEP exchange, obtains a certificate, and Fleet observes the matching certificate, the profile SHALL transition from
`failed` to `verified`. The system SHALL NOT classify these failures as transient versus permanent.

#### Scenario: Upstream CA returns an error during PKIOperation
- **WHEN** the SCEP proxy receives an upstream error during `PKIOperation` for a `(host, profile)`
- **THEN** the profile is set to `failed` with a `detail` naming the operation (`PKIOperation`) and the upstream status

#### Scenario: Device retry succeeds after a recorded failure
- **WHEN** a profile is `failed` from a proxy-observed error and the host later reports the matching certificate
- **THEN** the profile transitions to `verified`

#### Scenario: Device-side failure the proxy never sees
- **WHEN** the SCEP exchange fails on the device before reaching Fleet's proxy (for example a DNS resolution error), so
  the proxy observes no request
- **THEN** the profile remains `verifying` (Fleet has no evidence of failure) rather than being marked `failed`

### Requirement: Verification is scoped to proxied CAs with a matchable identifier

The verifying-until-observed and proxy-error-surfacing behaviors SHALL apply only to Windows SCEP profiles whose CA type
supports a renewal-ID marker (`custom_scep_proxy`, `ndes`, `smallstep`). Server-issued CAs (DigiCert) and non-certificate
profiles SHALL be unaffected.

#### Scenario: DigiCert or non-certificate profile
- **WHEN** a Windows profile is backed by DigiCert or is not a certificate profile
- **THEN** none of the verifying-until-observed, cert-match verification, or proxy-error-failure behaviors apply to it
