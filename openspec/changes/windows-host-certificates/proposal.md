## Why

IT admins can see the certificates installed on macOS hosts (Host details page), which helps them troubleshoot
problems like "this user can't reach the corporate network, do they actually have the right certificate?" Windows
admins have the same need, but the Host details page does not show certificates for Windows hosts today, even though
Fleet already collects them. This closes that gap so Windows reaches parity with macOS (GitHub issue #31294).

A spike on a real Windows host also found that the existing Windows certificate scope mapping is wrong: machine-wide
certificates installed in the `LocalMachine` store are mislabeled as user certificates with a blank owner, and the
"scope" cannot be trusted. This change corrects that so the System / User distinction is accurate.

## What Changes

- Show the existing "Certificates" card on the Host details page for Windows hosts (it is already shown for Apple
  hosts; the data is already fetched for all hosts, only the display is gated off for Windows).
- Rename the certificates table column "Keychain" to "Scope" for all platforms (macOS and Windows), and make the
  table help text platform-aware (keychains on macOS, the Personal certificate store on Windows).
- Fix Windows certificate scope mapping: derive scope from the certificate's registry hive (`store_location` / `sid`)
  instead of matching the owner string against `"SYSTEM"`. Machine-wide (`LocalMachine`) and built-in system/service
  accounts collapse into a single **System** scope; real interactive users (`S-1-5-21-*`) become **User** scope keyed
  by username. This matches the two-bucket System / User model already used for macOS.
- Make Windows certificate reconciliation user-aware: a user's certificates are only visible to osquery while that
  user is logged in, so Fleet must not delete a user's certificates just because the user logged off. Only
  certificates for users present in the current report (plus the always-visible System scope) are reconciled;
  certificates for absent users are preserved until a genuine removal is observed while that user is logged in.
- Support multiple users on one Windows host: each user that holds a certificate appears as its own User-scope row
  labeled with that username (this already works through the shared data model; this change makes it correct for
  Windows).

Not changing: the REST API response shape (reuses the existing macOS `source` / `username` fields), the database
schema (the `source` enum and `username` column already exist), fleetd, GitOps, and activities.

## Capabilities

### New Capabilities
- `windows-host-certificates`: Collecting, scoping (System vs User), reconciling, and displaying Windows host
  certificates on the Host details page, with parity to the macOS certificates experience.

### Modified Capabilities
<!-- No existing spec files under openspec/specs/ to modify; the macOS certificates behavior has never been captured
     as an OpenSpec spec. The shared System/User display and column rename are documented in the new capability. -->

## Impact

- **Backend (osquery ingestion)**: `server/service/osquery_utils/queries.go` — the `certificates_windows` detail
  query gains `store_location` and `sid`; `directIngestHostCertificatesWindows` reworks scope derivation and dedup.
  Test: `TestDirectIngestHostCertificatesWindows`.
- **Backend (datastore)**: `server/datastore/mysql/host_certificates.go` — `UpdateHostCertificates` reconciliation
  becomes user/scope-aware so logged-off users' certificates are preserved. Shared by macOS and Windows; macOS
  behavior must be unchanged.
- **Frontend**: `frontend/pages/hosts/details/HostDetailsPage/HostDetailsPage.tsx` (un-gate the card for Windows),
  `.../cards/Certificates/CertificatesTable/CertificatesTableConfig.tsx` ("Keychain" → "Scope"),
  `.../CertificatesTable/CertificatesTable.tsx` (platform-aware help text), and the certificate details modal
  (verify it renders for Windows).
- **No database migration.** **No REST API contract change.** **No fleetd change.**
- **Tiers**: available in Fleet Free and Fleet Premium.
- **Docs**: any Contributor-API note about Windows scope semantics ships in a separate docs PR, per team convention.
