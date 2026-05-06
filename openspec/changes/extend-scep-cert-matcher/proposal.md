## Why

When the matcher inside `UpdateHostCertificates` (`server/datastore/mysql/host_certificates.go`) misses linking a renewed SCEP cert to its `host_mdm_managed_certificates` row — because of replica lag, a transaction race, or because the cert lands as `existingBySHA1` rather than `toInsert` — the row stays NULL forever. The renewal cron's `HAVING validity_period IS NOT NULL` lock then excludes it, so the cron silently never re-attempts. Manual re-push is the only recovery, which is what motivated customer issue #44111.

## What Changes

- Run the matcher on every `UpdateHostCertificates` call instead of gating on `len(toInsert) > 0`. Hosts whose cert inventory is stable (no `toInsert` this cycle) can now recover stuck `hmmc` rows from a renewed cert that's already in `host_certificates`.
- Per-`hmmc`-row pool selection: when the row is **stuck** (`not_valid_*` NULL AND `updated_at` older than the in-flight grace window AND profile in `'verified'`) the matcher widens its search to the full reported inventory; otherwise it iterates only newly-inserted certs (preserves today's "react to new certs" semantics so an in-flight renewal can't be clobbered by the pre-renewal cert).
- Switch "first match wins" → "best match wins" (latest `not_valid_before` among currently-valid candidates), so when both old and new renewal-ID certs are reported the freshest wins regardless of iteration order.
- Add a monotonic-forward predicate so the matcher cannot regress an already-fresh `hmmc` row with an older cert.
- Replace `ListHostMDMManagedCertificates` with a SELECT that also `LEFT JOIN`s the per-platform profile tables for delivery status (needed to gate the stuck-check on `'verified'` profiles).
- Introduce `hmmcBackfillGrace` (a duration constant) used by the stuck-row check.

No changes to `BulkUpsertMDMManagedCertificates`, the renewal cron, or the per-platform profile state machines. No new datastore methods, no new mocks, no migrations.

## Capabilities

### New Capabilities

- `mdm-cert-state-sync`: Keeping `host_mdm_managed_certificates` (hmmc) in sync with the cert state the device actually reports — including recovery for rows that get stuck NULL after a renewal trigger when the original toInsert match was missed.

### Modified Capabilities

(none)

## Impact

- Code: `server/datastore/mysql/host_certificates.go` — matcher extended (~30 LOC delta) plus a new `hmmcBackfillGrace` constant.
- Tests: `server/datastore/mysql/host_certificates_test.go` — adds subtests covering missed-ingest recovery, in-flight protection, monotonic forward, tie-breaker, DigiCert-not-clobbered, pending-profile-skipped, and stable-cert-list recovery.
- No public datastore-interface changes; no `make generate-mock` needed.
- Hot-path cost: one additional SELECT per `UpdateHostCertificates` call (the joined hmmc + per-platform-profile query). Host-uuid-keyed against indexed PKs. To be load-tested before merging.
- Behavior changes are observable only in renewal-recovery scenarios; steady-state cert ingest is unchanged.
