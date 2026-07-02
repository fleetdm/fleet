## 1. Backend — osquery query and ingestion

- [ ] 1.1 Add `store_location` and `sid` to the `SELECT` of the `certificates_windows` detail query in
  `server/service/osquery_utils/queries.go` (keep `WHERE store = 'Personal'`).
- [ ] 1.2 In `directIngestHostCertificatesWindows`, replace the `username == "SYSTEM"` scope heuristic with hive-based
  classification: `sid` matching `S-1-5-21-*` → User scope (username = reported owner); everything else
  (`LocalMachine`/empty sid, `S-1-5-18/19/20`, `.DEFAULT`, `S-1-5-80-*`) → System scope, username = `""`.
- [ ] 1.3 De-duplicate the ingested source tuples by `(SHA-1, scope, username)` so redundant `CurrentUser` /
  `Services` / `_Classes` / `Users\<service>` views collapse; ensure dedup runs before the records reach
  `UpdateHostCertificates`/`replaceHostCertsSourcesDB` (avoid the unique-constraint FIXME at
  `server/datastore/mysql/host_certificates.go:521-530`).
- [ ] 1.4 Keep the existing empty-result guard (do not overwrite stored certs when osquery returns zero rows).
- [ ] 1.5 Rewrite `TestDirectIngestHostCertificatesWindows` using the spike data shapes: `LocalMachine\Personal`
  (blank username → System), `S-1-5-18`/SYSTEM triple-listed (→ one System entry), `S-1-5-21-*` real user (→ User),
  and a `_Classes` duplicate (→ collapses into the user's entry).

## 2. Backend — user-aware reconciliation (datastore)

- [ ] 2.1 In `server/datastore/mysql/host_certificates.go`, change `UpdateHostCertificates` reconciliation so it only
  soft-deletes within scope groups present in the incoming batch: compute observed usernames from the batch, treat
  System scope as always observed, and skip deletion for certificates/sources of users not in the batch.
- [ ] 2.2 Reconcile at `host_certificate_sources` granularity: remove only the source rows whose `(scope, username)`
  is observed-but-no-longer-reported, and soft-delete a `host_certificates` row only when it has no remaining live
  sources.
- [ ] 2.3 Verify macOS behavior is unchanged (every keychain is always present on disk → every scope observed →
  identical reconciliation).
- [ ] 2.4 Add datastore tests (`MYSQL_TEST=1`): user logs off → certs preserved; cert removed while user logged in →
  soft-deleted; System cert removed → soft-deleted; multi-user host → one User entry per user; macOS regression.
- [ ] 2.5 Run `go test ./server/service/` to confirm no mock/interface breakage after datastore changes.

## 3. Frontend — show and label Windows certificates

- [ ] 3.1 Un-gate the card in `HostDetailsPage.tsx`:
  `showCertificatesCard = (isAppleDeviceHost || isWindowsHost) && !!hostCertificates?.certificates.length`.
- [ ] 3.2 Rename the certificates table column header "Keychain" → "Scope" in `CertificatesTableConfig.tsx` (applies
  to macOS too); keep the cell logic (System; User with username tooltip).
- [ ] 3.3 Make the table help text platform-aware in `CertificatesTable.tsx` (macOS: system + login keychains;
  Windows: the Personal certificate store) and show it for Windows hosts.
- [ ] 3.4 Verify the certificate details modal renders correctly for Windows certificates (same payload, no
  Windows-specific fields).
- [ ] 3.5 Update/extend frontend tests and the certificates mock (`frontend/__mocks__/certificatesMock.ts`) to cover
  Windows hosts (System and User rows) and the renamed column.

## 4. Verification on a real Windows host

- [ ] 4.1 On the Azure Windows VM, enroll with orbit built from this branch and confirm the Certificates card appears
  with correct System/User scopes (System for LocalMachine + SYSTEM-account certs, User per logged-in user).
- [ ] 4.2 Confirm a user logging off does not remove their certificates from Host details, and that a certificate
  actually removed (while the user is logged in) disappears on the next report.
- [ ] 4.3 Confirm the macOS Certificates card still shows the renamed "Scope" column and unchanged data.

## 5. Lint, test plan, and follow-ups

- [ ] 5.1 Run `make lint-go-incremental` and `make lint-js`; add a `changes/` changelog entry.
- [ ] 5.2 Fill in the issue test plan / QA confirmation and capture results.
- [ ] 5.3 (Separate PR) Add any Contributor-API doc note about Windows certificate scope semantics.
- [ ] 5.4 (Optional follow-up) Note osquery-perf does not emit Windows cert rows; file/track if load testing needs it.
