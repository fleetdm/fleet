## 1. Detect proxied SCEP profiles at response time

- [ ] 1.1 Add a datastore helper (or reuse an hmmc lookup) to determine, for a `(host_uuid, profile_uuid)`, whether an
  `host_mdm_managed_certificates` row exists with `type` satisfying `SupportsRenewalID()` (`custom_scep_proxy`, `ndes`,
  `smallstep`). Prefer folding it into the existing status-save query in `updateMDMWindowsHostProfileStatusFromResponseDB`
  rather than a separate round trip.
- [ ] 1.2 If a new `Datastore` interface method is added, regenerate/extend the mock and run `go test ./server/service/`
  so uninitialized mocks do not crash other tests.

## 2. Map ACK to `verifying` for proxied SCEP profiles

- [ ] 2.1 In `server/datastore/mysql/microsoft_mdm.go` (`updateMDMWindowsHostProfileStatusFromResponseDB` /
  `MDMWindowsSaveResponse`), when a 2xx Install ACK is for a proxied SCEP profile (from task 1.1), persist `verifying`
  instead of `verified`. Leave non-proxied profiles on today's path (via `WindowsResponseToDeliveryStatus`).
- [ ] 2.2 Keep `WindowsResponseToDeliveryStatus` a pure helper; do the proxied-SCEP branch at the datastore layer where
  hmmc context is available. Do not change Remove-operation mapping.
- [ ] 2.3 Unit-test the mapping: proxied SCEP profile 2xx ACK -> `verifying`; non-cert profile 2xx ACK -> `verified`;
  4xx/5xx ACK -> `failed` (unchanged).

## 3. Flip `verifying` -> `verified` on certificate observation

- [ ] 3.1 In `server/datastore/mysql/host_certificates.go` (`UpdateHostCertificates`), when a renewal-ID match is found
  for an hmmc row, also update the matching `host_mdm_windows_profiles` row (Install) to `verified`: set
  `status=verified`, clear `detail`, and preserve `retries` (matches the "verified preserves retries" convention). This
  covers both `verifying` -> `verified` and the `failed` -> `verified` self-heal.
- [ ] 3.2 Rework the `verified`-gated "stuck" backfill branch (currently `verified := isVerifiedStatus(...)`) so a
  first-observation match flips `verifying` -> `verified`, while preserving the existing stale-renewal recovery for
  already-`verified` rows (issue #44111).
- [ ] 3.3 Confirm the flip is scoped to Windows profiles and to CA types with renewal-ID support; do not touch Apple
  profile status here.
- [ ] 3.4 Decide and implement whether the status update batches with the hmmc update in one transaction or runs as a
  follow-on update keyed by matched profile UUIDs (design Open Question).

## 4. Surface proxy-observed upstream errors as `failed` (Option A)

- [ ] 4.1 In `ee/server/service/scep/scep_proxy.go`, replace the Windows early-return
  (`TODO: Early return for Windows profiles as they do not support resending yet`) so upstream errors during
  `GetCACaps`/`GetCACert`/`PKIOperation` for a `(host, profile)` are captured.
- [ ] 4.2 Add a datastore method to set `host_mdm_windows_profiles.status = failed` with a `detail` for the
  `(host_uuid, profile_uuid)` parsed from the proxy identifier. Format the detail as `SCEP <operation> failed: <reason>`
  (operation = `GetCACaps`/`GetCACert`/`PKIOperation`; reason = `HTTP <code>` or a short class such as `timeout` /
  `malformed PKCS#7 response`). Never include the SCEP challenge or full proxy URL. Do NOT touch `retries` (this is not
  the SyncML auto-retry path; the device's own CSP retry handles transient blips). Guard against resurrecting a row for a
  profile that was removed while the exchange was in flight.
- [ ] 4.3 Ensure no transient/permanent classification and no new activity type are introduced.
- [ ] 4.4 Verify self-heal: a `failed` profile whose certificate is later observed transitions to `verified` via task 3
  (add a test covering `failed` -> `verified`).

## 5. Confirm no-fail-on-absence behavior

- [ ] 5.1 Add tests proving a proxied SCEP profile stays `verifying` when: the host is offline (no report), the agent
  cannot enumerate cert subjects (query discovery-gated off), the cert report is empty, and a `./User/...` profile's
  target user is not logged in.
- [ ] 5.2 Add a test proving a `./User/...` profile flips to `verified` after the target user logs in and reports the
  matching certificate from their store, and that a logged-off user's previously verified cert is not soft-deleted.

## 6. Integration and regression tests

- [ ] 6.1 Add an integration test (`MYSQL_TEST=1 REDIS_TEST=1`) for the full happy path: send proxied SCEP profile ->
  ACK -> `verifying` -> ingest matching cert -> `verified`, asserting `GET /api/v1/fleet/hosts/{id}` reports the
  transitions via `status`/`detail`.
- [ ] 6.2 Add an integration test for the failure path: upstream error at the proxy -> `failed` with detail.
- [ ] 6.3 Run `find-related-tests` and the `mysql`, `service`, and `integration-mdm` bundles; run
  `make lint-go-incremental`.

## 7. Premium gating and docs (docs in a separate PR)

- [ ] 7.1 Confirm SCEP proxy Premium gating is intact on backend and frontend; add a gating test if missing.
- [ ] 7.2 (Separate branch/PR) Update `server/mdm/scep/SCEP.md` and the SCEP/cert-proxy feature guide to describe the
  verifying-until-observed flow, the osquery 5.23.1+ requirement, the `./User/...` login dependency, and the
  proxy-error `detail` format. Add the `detail` format to the REST API reference.
