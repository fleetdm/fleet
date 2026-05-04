## Why

When the matcher inside `UpdateHostCertificates` (`server/datastore/mysql/host_certificates.go`) misses linking a newly-issued SCEP cert to its `host_mdm_managed_certificates` row — because of replica lag, a transaction race, or because the cert lands as `existingBySHA1` rather than `toInsert` — the row stays NULL forever. The renewal cron's `HAVING validity_period IS NOT NULL` lock then excludes it, so the cron silently never re-attempts. Manual re-push is the only recovery, which is what motivated customer issue #44111.

The matcher already loads `ListHostMDMManagedCertificates` for the host whenever the cert inventory has changed, and the full incoming cert set is already in memory by the time the matcher runs. We can extend the matcher itself to recover stuck-NULL rows from this already-loaded data — no new database queries on the hot path.

## What Changes

- Extend the existing matcher in `UpdateHostCertificates` to recover stuck-NULL `hmmc` rows by widening the cert pool it searches when (and only when) the row's `not_valid_*` are NULL and `updated_at` is older than the in-flight grace window.
- Switch the matcher's "first match wins" loop to "best match wins" (most recently issued currently-valid cert), so a stale cert in `host_certificates` can't win over a fresh one when both match the renewal-ID substring.
- Add a monotonic-forward predicate so the matcher cannot regress an already-fresh `hmmc` row with an older cert.
- Build a `toInsertBySHA1` map alongside the existing `incomingBySHA1` so the matcher's two cert pools (steady-state vs recovery-mode) share the same access pattern.
- Introduce `hmmcBackfillGrace` (a duration constant) used by the matcher's stuck-row check.

No changes to `BulkUpsertMDMManagedCertificates`, the renewal cron, or the per-platform profile state machines. No new datastore methods, no new mocks, no migrations.

## Capabilities

### New Capabilities

- `mdm-cert-state-sync`: Keeping `host_mdm_managed_certificates` (hmmc) in sync with the cert state the device actually reports — including recovery for rows that get stuck NULL after a renewal trigger when the original toInsert match was missed.

### Modified Capabilities

(none)

## Impact

- Code: `server/datastore/mysql/host_certificates.go` — matcher extended (~30 LOC delta) plus a new `hmmcBackfillGrace` constant.
- Tests: `server/datastore/mysql/host_certificates_test.go` — add subtests covering missed-ingest recovery, in-flight protection, monotonic forward, tie-breaker, DigiCert-not-clobbered, and pending-profile-skipped.
- No public datastore-interface changes; no `make generate-mock` needed.
- Hot-path cost: zero new SELECTs over the existing matcher path.
- Behavior changes are observable only in renewal-recovery scenarios; steady-state cert ingest is unchanged.
