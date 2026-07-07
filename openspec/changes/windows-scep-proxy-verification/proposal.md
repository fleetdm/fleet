## Why

When Fleet proxies SCEP for a Windows MDM certificate profile, the device ACKs the SyncML `<Exec>` (`Install/Enroll`)
with a 2xx status immediately, then runs the SCEP exchange asynchronously. Fleet marks the profile `verified` off that
2xx ACK and never confirms a certificate actually landed. The result (GitHub issue #45550): hosts that never obtained a
certificate report `verified`, giving admins no way to tell a successful enrollment from a silent failure (upstream CA
error, expired challenge, or a device-side failure such as a DNS lookup error before the request even reaches Fleet).

Fleet already ingests Windows host certificates and already injects a renewal-ID marker (`fleet-<profile_uuid>`) into
each proxied SCEP certificate's OU, and already matches ingested certs back to the requesting profile. That match result
is not connected to profile delivery status. This change wires it up so Windows SCEP profile status reflects reality.

## What Changes

- Proxied Windows SCEP certificate profiles no longer go straight to `verified` on the device's 2xx ACK. They move to
  `verifying` and stay there until Fleet observes the matching certificate on the host.
- A proxied Windows SCEP profile transitions `verifying` -> `verified` when the host reports a certificate whose OU (or
  CN) contains the profile's renewal-ID marker. This happens inside the certificate ingestion path, so it takes effect
  as soon as the host reports (immediately on a manual Refetch).
- Unconfirmed profiles stay in `verifying` indefinitely rather than being failed on absence. This covers hosts that are
  offline, running osquery older than 5.23.1 (no `subject2`/`issuer2`, so no cert ingestion), reporting an empty
  Personal store, or (for `./User/...` profiles) waiting for that user to log in so their store becomes readable.
- Fleet surfaces certificate failures it directly observes: when the SCEP proxy sees an upstream error during
  `GetCACaps`, `GetCACert`, or `PKIOperation` for a `(host, profile)`, that profile is set to `failed` with a detail
  string describing the SCEP operation and upstream status. If the device's own SCEP retry later succeeds and Fleet
  observes the certificate, the profile self-heals to `verified`.
- Scope is limited to proxied SCEP CAs whose certificates carry a matchable renewal-ID marker
  (`custom_scep_proxy`, `ndes`, `smallstep`). Non-certificate Windows profiles and server-issued CAs (DigiCert) are
  unchanged.
- No transient-vs-permanent error classification and no new activity type: this matches how Apple profile verification
  already behaves (retry a fixed number of times, then fail; no verification-failure activity).

## Capabilities

### New Capabilities
- `windows-scep-cert-verification`: how a proxied Windows SCEP configuration profile's delivery status
  (`verifying`, `verified`, `failed`) is determined from observed host certificates and proxy-detected upstream errors,
  rather than from the SyncML `<Exec>` ACK alone.

### Modified Capabilities
<!-- No existing openspec specs; nothing to modify. -->

## Impact

- **Behavior**: proxied Windows SCEP profiles gain a real `verifying` state and only reach `verified` once the cert is
  observed. `GET /api/v1/fleet/hosts/{id}` and the host detail / profiles UI reflect `verifying`/`verified`/`failed`
  through the existing `status` + `detail` fields; no new API or screen.
- **Code**:
  - `server/datastore/mysql/microsoft_mdm.go` (`WindowsResponseToDeliveryStatus` / the response-to-status mapping): map
    a 2xx ACK to `verifying` for profiles backed by a renewal-ID managed-cert row.
  - `server/datastore/mysql/host_certificates.go` (`UpdateHostCertificates`): on a renewal-ID match, flip the matching
    Windows profile `verifying` -> `verified`; rework the `verified`-gated "stuck" backfill branch so `verifying`
    profiles are also matched.
  - `ee/server/service/scep/scep_proxy.go` (`validateIdentifier` and the upstream forward path): persist upstream SCEP
    errors for the `(host, profile)` and mark the profile `failed` with detail (replaces the existing
    `TODO: Early return for Windows profiles as they do not support resending yet`).
  - `server/fleet/microsoft_mdm.go`: status-mapping helpers and any new datastore interface method.
- **Dependencies**: verification requires osquery 5.23.1+ on the Windows host (for `subject2`/`issuer2` in the
  `certificates` table). Older agents leave profiles in `verifying` (documented, no fallback).
- **Premium**: SCEP proxy is already Fleet Premium; confirm gating stays intact. No migrations expected (reuses
  `host_mdm_windows_profiles.status`/`detail`/`retries` and `host_mdm_managed_certificates`).
- **Docs** (separate PR): `server/mdm/scep/SCEP.md` and the SCEP/cert-proxy feature guide.
