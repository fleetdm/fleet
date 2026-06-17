## ADDED Requirements

### Requirement: Collect Windows host certificates from the Personal store

Fleet SHALL collect certificates from the Personal certificate store of Windows hosts via osquery and ingest them
into host certificate storage, reusing the same storage, API, and payload fields as macOS host certificates.

#### Scenario: Personal store certificates are ingested

- **WHEN** a Windows host reports its osquery `certificates` table rows where `store = 'Personal'`
- **THEN** Fleet stores each distinct certificate (by SHA-1) for that host with its parsed fields (common name,
  issuer, subject, validity dates, key info, serial, signing algorithm)

#### Scenario: Non-Personal stores are not surfaced

- **WHEN** a Windows host has certificates in stores other than Personal (e.g. Trusted Root, Intermediate CA)
- **THEN** Fleet does not include them in the host's certificates, matching the macOS behavior of showing identity
  certificates only

#### Scenario: An empty report does not erase known certificates

- **WHEN** a Windows host returns zero certificate rows for a reporting cycle (e.g. a transient collection failure)
- **THEN** Fleet leaves the host's previously recorded certificates unchanged

### Requirement: Classify Windows certificate scope by registry hive

Fleet SHALL classify each Windows certificate as **System** or **User** scope based on the certificate's registry
hive (`sid` / `store_location`), not on the owner name string. A certificate whose `sid` identifies a real
interactive account (`S-1-5-21-*`) SHALL be User scope owned by that account's username; every other certificate
(machine `LocalMachine` store and built-in system/service accounts such as `S-1-5-18`, `S-1-5-19`, `S-1-5-20`,
`.DEFAULT`, and `S-1-5-80-*`) SHALL be System scope with no username.

#### Scenario: Machine-wide certificate is System scope

- **WHEN** a certificate is reported from the `LocalMachine` store (empty sid and empty username)
- **THEN** Fleet records it with System scope and no username

#### Scenario: Real user certificate is User scope

- **WHEN** a certificate is reported under a `S-1-5-21-*` user hive with username `jdoe`
- **THEN** Fleet records it with User scope and username `jdoe`

#### Scenario: Built-in system/service account certificates are System scope

- **WHEN** a certificate is reported under the LocalSystem account (`S-1-5-18`, username `SYSTEM`) or another
  built-in service account
- **THEN** Fleet records it with System scope and no username

### Requirement: Merge and de-duplicate Windows certificate scopes

Fleet SHALL fold all System-classified occurrences of the same certificate into a single System scope, and SHALL
de-duplicate scope records by `(certificate SHA-1, scope, username)` so that the same certificate enumerated across
multiple redundant registry hives (`CurrentUser`, `Services`, per-user `_Classes` sub-hives) is recorded once per
distinct scope/owner.

#### Scenario: Same certificate across redundant system hives is recorded once

- **WHEN** the same certificate (same SHA-1) is reported from `LocalMachine`, the LocalSystem `CurrentUser` view, and
  `Users\S-1-5-18`
- **THEN** Fleet records a single System-scope entry for that certificate

#### Scenario: A certificate held by two users yields one User row per user

- **WHEN** the same certificate is present in the Personal store of both user `alice` and user `bob`
- **THEN** Fleet records one User-scope entry for `alice` and one User-scope entry for `bob`

#### Scenario: A certificate in both System and a user store yields both scopes

- **WHEN** the same certificate is present in `LocalMachine` and in user `alice`'s store
- **THEN** Fleet records one System-scope entry and one User-scope entry for `alice`

### Requirement: Preserve certificates of users who are not currently logged in

Fleet SHALL reconcile (and therefore soft-delete) only certificates whose scope is present in the current report, and
SHALL preserve certificates belonging to a user who is not present in the current report. System scope is always
considered present. This is required because osquery can only enumerate a Windows user's Personal certificates while
that user is logged in and their registry hive is loaded.

#### Scenario: User logs off — their certificates are retained

- **WHEN** user `alice` was previously reporting certificates and a later report contains no rows for `alice`
  (she has logged off) but still contains System-scope rows
- **THEN** Fleet retains `alice`'s previously recorded certificates and does not soft-delete them

#### Scenario: Certificate removed while the user is logged in — it is soft-deleted

- **WHEN** user `alice` is present in the current report but one of her previously recorded certificates is no longer
  among her reported certificates
- **THEN** Fleet soft-deletes that certificate's User-scope entry for `alice`

#### Scenario: System certificate removed — it is soft-deleted

- **WHEN** a previously recorded System-scope certificate is absent from the current report
- **THEN** Fleet soft-deletes that System-scope certificate (System scope is always reconciled)

#### Scenario: macOS reconciliation is unchanged

- **WHEN** a macOS host reports its certificates (all keychains are always readable from disk)
- **THEN** every scope is considered present and reconciliation behaves exactly as before this change

### Requirement: Show the Certificates card on the Host details page for Windows hosts

Fleet SHALL display the Host details "Certificates" card for Windows hosts when the host has at least one
certificate, and SHALL hide the entire card when the host has no certificates, matching the macOS behavior.

#### Scenario: Windows host with certificates shows the card

- **WHEN** an admin views the Host details page for a Windows host that has certificates
- **THEN** the Certificates card is displayed with the host's certificates

#### Scenario: Windows host without certificates hides the card

- **WHEN** an admin views the Host details page for a Windows host that has no certificates
- **THEN** the Certificates card is not displayed

### Requirement: Certificates table shows a platform-agnostic Scope column

The certificates table SHALL label the certificate scope column "Scope" (replacing the previous "Keychain" label) on
all platforms, displaying "System" for system-scoped certificates and "User" for user-scoped certificates, with the
owning username available for user-scoped certificates. Table help text SHALL describe the certificate source
appropriately for the host platform.

#### Scenario: Scope column replaces Keychain on macOS and Windows

- **WHEN** an admin views the certificates table on either a macOS or a Windows host
- **THEN** the scope column header reads "Scope"

#### Scenario: User-scoped certificate shows its owner

- **WHEN** a user-scoped certificate is displayed
- **THEN** the row shows "User" scope and the owning username is shown (e.g. on hover)

#### Scenario: Help text matches the platform

- **WHEN** the certificates table is shown on a Windows host
- **THEN** the help text describes the Windows Personal certificate store (rather than macOS keychains)
