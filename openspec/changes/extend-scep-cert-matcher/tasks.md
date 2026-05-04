## 1. Extend the matcher in place

- [x] 1.1 In `server/datastore/mysql/host_certificates.go`, add a top-level `hmmcBackfillGrace` duration constant (default `4 * time.Hour`) next to `hostCertificateAllowedOrderKeys`. Include a comment explaining its role: the in-flight grace window after which a NULL hmmc row is considered stuck and eligible for wide-pool recovery.
- [x] 1.2 In the diff loop at lines 88-116, build a `toInsertBySHA1 map[string]*fleet.HostCertificateRecord` alongside `toInsert`. Populate in the same `else` branch that appends to `toInsert`.
- [x] 1.3 Replace the matcher block at lines 115-152 with the new shape:
    - Keep the `if len(toInsert) > 0` gate (decision §2 in design.md).
    - For each hmmc row returned by `ListHostMDMManagedCertificates`, skip if `Type.SupportsRenewalID()` is false.
    - Compute `stuck := hostMDMManagedCert.NotValidAfter == nil && time.Since(hostMDMManagedCert.UpdatedAt) > hmmcBackfillGrace`.
    - Choose pool: `incomingBySHA1` when stuck, else `toInsertBySHA1`.
    - Iterate the pool and pick the cert with the latest `NotValidBefore` among currently-valid candidates (`NotValidBefore <= now <= NotValidAfter`).
    - If no match, continue.
    - Apply monotonic-forward predicate: if `hostMDMManagedCert.NotValidAfter != nil && !hostMDMManagedCert.NotValidAfter.Before(bestMatch.NotValidAfter)`, continue.
    - Build the `MDMManagedCertificate` update record (same shape as today's matcher).
    - If `!hostMDMManagedCert.Equal(*managedCertToUpdate)`, append to `hostMDMManagedCertsToUpdate`.
- [x] 1.4 Verify the `MDMManagedCertificate` struct (in `server/fleet/apple_mdm.go`) exposes the field needed for the stuck check (`UpdatedAt time.Time`), and that `ListHostMDMManagedCertificates`'s SELECT loads it. If `UpdatedAt` is missing, add it to the struct's `db` tags and the SELECT column list. Ensure no breaking change to other consumers.
- [x] 1.5 Add an inline comment on the pool-selection block referencing the in-flight vs stuck distinction, calling out that the steady-state path matches today's behavior.

## 2. Tests

- [x] 2.1 In `server/datastore/mysql/host_certificates_test.go`, add a top-level entry `{"Matcher recovers stuck hmmc rows", testMatcherRecoversStuckHMMCRows}` to the `TestHostCertificates` cases slice.
- [x] 2.2 Implement `testMatcherRecoversStuckHMMCRows(t *testing.T, ds *Datastore)` with subtests covering:
    - **MissedIngestRecovered** — hp.status='verified' (Apple or Windows), hmmc.not_valid_after IS NULL, hmmc.updated_at backdated past grace, a matching cert already present in `host_certificates` from a prior call. Drive a new `UpdateHostCertificates` call where toInsert is non-empty (e.g., add an unrelated cert) and assert the stuck hmmc row gets populated with the matching cert's values.
    - **GraceBoundary** — same setup but with hmmc.updated_at within the grace window; assert no update fires.
    - **TieBreaker** — two valid matching certs for the same renewal-ID; assert the cert with the latest `not_valid_before` wins.
    - **DigiCertNotClobbered** — hmmc.type='digicert' with manually populated values; matching cert in host_certificates; assert no update.
    - **PendingProfileSkipped** — hp.status='pending' (in flight); even with a matching cert and old updated_at, assert no recovery (the in-flight gate keeps the row out of the wide-pool branch).
    - **MonotonicForward** — hmmc populated with a fresher `not_valid_after`; only an older matching cert in the pool; assert no regression.
- [x] 2.3 Reuse `BulkUpsertMDMAppleHostProfiles` / `BulkUpsertMDMWindowsHostProfiles` for profile state setup, and the existing `generateTestHostCertificateRecord` helper for cert payloads. Backdate `hmmc.updated_at` via raw SQL.

## 3. Verification

- [x] 3.1 `go build ./...` clean.
- [x] 3.2 `make lint-go-incremental` reports 0 issues against changes since `origin/main`.
- [x] 3.3 `MYSQL_TEST=1 REDIS_TEST=1 FLEET_MYSQL_TEST_PORT=3309 go test ./server/datastore/mysql/ -count=1 -run "TestHostCertificates|TestMDMApple/MDMManagedSCEPCertificates" -timeout 5m` passes.
- [x] 3.4 `MYSQL_TEST=1 REDIS_TEST=1 FLEET_MYSQL_TEST_PORT=3309 go test ./server/service/...` — targeted SCEP/MDM tests pass (TestIntegrationsMDM/TestWindowsUserSCEPProfile, TestHostCertificates, TestMDMApple/MDMManagedSCEPCertificates). Broader sweep had three classes of failure, all unrelated to this change: (1) Redis-cluster packages (`async`, `redis_key_value`, `redis_lock`, `redis_policy_set`) fail with `dial 127.0.0.1:7001: connection refused` because the macOS-specific local redis cluster isn't running; (2) several `TestIntegrations*` tests (TestIntegrationsMDM/TestWindowsProfileRetry, TestWindowsProfilesFleetVariableSubstitution, TestWindowsUserSCEPProfile) fail in the broader run but pass when re-run in isolation — pre-existing test-pollution / order-dependence in the integration suite. CI will run with full infrastructure.
- [ ] 3.5 Manual smoke per `docs/Contributing/product-groups/mdm/custom-scep-integration.md`: backdate an `hmmc.not_valid_after` and `updated_at` on a host with a matching cert in `host_certificates`, trigger a cert-changing `UpdateHostCertificates` call, verify hmmc gets populated and that the renewal cron does not re-mark the row.

## 4. Changelog & PR

- [x] 4.1 `changes/44111-scep-autorenew-fail` already exists with the customer-facing wording approved earlier: `Fixed SCEP certificates failing to auto-renew, including on devices that go offline between profile push and SCEP request.` Kept as-is.
- [ ] 4.2 In the PR description, briefly explain the matcher-extension shape (one paragraph) and link to this OpenSpec change for the full design rationale.
