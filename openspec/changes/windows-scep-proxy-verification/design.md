## Context

Fleet proxies SCEP for Windows MDM certificate profiles via `$FLEET_VAR_*_SCEP_PROXY_URL_*` variables. The Windows
`ClientCertificateInstall/SCEP` CSP ACKs the `<Exec>` (`Install/Enroll`) with 2xx immediately, then runs the SCEP
exchange asynchronously. Today `WindowsResponseToDeliveryStatus` (`server/fleet/microsoft_mdm.go`) maps any 2xx to
`verified`; Windows has no `verifying` step and no independent verification, so a host that never obtained a certificate
still reports `verified` (issue #45550).

The pieces to fix this already exist and are only unwired:
- A Windows SCEP profile is validated to hold exactly one certificate with all required fields, and the renewal-ID
  variable `$FLEET_VAR_CERTIFICATE_RENEWAL_ID` is required in the SubjectName OU
  (`server/service/windows_mdm_profiles.go`, `server/fleet/windows_mdm.go`).
- The renewal-ID expands to `fleet-<profile_uuid>` (`server/mdm/microsoft/profile_variables.go`).
- A `host_mdm_managed_certificates` (hmmc) row is created at send time for proxied SCEP profiles via
  `BulkUpsertMDMManagedCertificates` (`server/service/microsoft_mdm.go`), with `type` in {`custom_scep_proxy`, `ndes`,
  `smallstep`} and `not_valid_after = NULL`.
- Windows certs are ingested via the osquery `certificates_windows` query and `UpdateHostCertificates`
  (`server/service/osquery_utils/queries.go`, `server/datastore/mysql/host_certificates.go`). That path already matches
  ingested certs to hmmc rows by testing whether `fleet-<profile_uuid>` is a substring of the cert's CN or OU, and fills
  in `not_valid_after`/`serial`. Nothing connects that match to `host_mdm_windows_profiles.status`.

Apple is the reference model: profiles go `verifying` on ACK and reach `verified` only when osquery confirms the payload
is installed (`server/mdm/apple/profile_verifier.go`). Apple verification is event-driven (runs on host-detail ingest,
never a wall-clock cron), uses a 1-hour grace period measured from the profile's `EarliestInstallDate` (its earliest
`updated_at`) against the host's last detail-report time (`hostDetailUpdatedAt`), per
`ExpectedMDMProfile.IsWithinGracePeriod`; retries a fixed number of times, then fails. Apple does not classify errors and
emits no activity for a verification failure.

## Goals / Non-Goals

**Goals:**
- Proxied Windows SCEP profiles reflect reality: `verifying` until the certificate is observed, then `verified`.
- Immediate transition on the host's next report (works with a manual Refetch), by doing the flip inside the ingestion
  path rather than a separate cron.
- Surface certificate failures Fleet directly observes at the proxy as `failed` with a meaningful detail.
- Never false-fail: offline hosts, old agents, empty stores, and logged-off users all stay `verifying`.
- Reuse existing tables and matching logic; no schema migration.

**Non-Goals:**
- No transient-vs-permanent error classification (Apple does not classify; issue #45550 speculated it but we are
  dropping it).
- No new activity type (Apple emits none for verification failures; only for enrollment-renewal command failures).
- No absence-based / time-based verification timeout that fails a profile for not reporting.
- No changes to Apple/iOS behavior, to non-proxied ACME/SCEP flows, to DigiCert, or to non-certificate Windows profiles.
- No new REST endpoint or UI screen; the existing `status`/`detail` fields carry the information.
- No fallback verification path for agents that cannot report certificate subjects.

## Decisions

### Decision 1: `verifying` on ACK for proxied SCEP profiles, keyed off the managed-cert row

On a 2xx ACK, map the profile to `verifying` instead of `verified` **only** when the profile is a proxied SCEP profile.
Detect this by the presence of an hmmc row for `(host_uuid, profile_uuid)` whose `type` satisfies `SupportsRenewalID()`
(`custom_scep_proxy`/`ndes`/`smallstep`). Non-matching profiles keep today's straight-to-`verified` behavior.

The mapping today lives in `WindowsResponseToDeliveryStatus` (a pure function of the response string) and is applied in
`updateMDMWindowsHostProfileStatusFromResponseDB` (`server/datastore/mysql/microsoft_mdm.go`). The "is this a proxied
SCEP profile" test needs datastore context, so the decision belongs in the datastore layer where the response is saved,
not in the pure helper. Alternatives considered: (a) a boolean column on `host_mdm_windows_profiles` set at send time —
rejected as an unnecessary migration when the hmmc row already encodes it; (b) re-parsing the profile SyncML at response
time to detect SCEP LocURIs — rejected as more expensive and redundant with the hmmc row.

### Decision 2: Flip `verifying` -> `verified` inside `UpdateHostCertificates`, on positive match only

When ingestion matches a cert to an hmmc row by renewal-ID, also update the corresponding `host_mdm_windows_profiles`
row from `verifying` to `verified` (Install operation). Doing this in the ingestion path (not a cron) gives immediate
feedback on Refetch, matching the operator workflow "refresh the device, then see the status update."

This requires reworking the existing "stuck" backfill branch in `host_certificates.go`. That branch currently gates on
the profile already being `verified` (`verified := isVerifiedStatus(...)`) to decide whether to widen the match pool for
a stale renewal. With profiles now sitting in `verifying`, the first observation must still be matched: the normal branch
(`len(toInsertBySHA1) > 0`) already fires whenever a new cert arrives regardless of status, so first-observation matching
works; the change is to also drive the profile-status update from that match, and to make sure the `verified`-gated
stuck-recovery logic still behaves for genuine renewals (already-verified rows) without blocking the initial
`verifying` -> `verified` flip.

Only a positive match changes status. Absence never does (see Decision 4). Alternative considered: a dedicated Windows
verification cron mirroring Apple's `VerifyHostMDMProfiles` — rejected because Apple's cron exists partly to fail missing
profiles after a grace period, which we explicitly do not want here, and because a cron delays the Refetch feedback loop.

### Decision 3: Option A — proxy-observed upstream errors mark the profile `failed`

In the SCEP proxy (`ee/server/service/scep/scep_proxy.go`), when an upstream error occurs for a `(host, profile)` during
`GetCACaps`/`GetCACert`/`PKIOperation`, persist a `failed` status and a `detail` naming the operation and upstream
status. This replaces the current `TODO: Early return for Windows profiles as they do not support resending yet`. The
identifier passed to the proxy already carries `hostUUID,profileUUID,caName,fleetChallenge`, so the target row is known.

Because the device's own SCEP CSP retry (`RetryCount`/`RetryDelay`, optional CSP nodes) can succeed after a transient
upstream blip, a `failed` profile self-heals: if the cert is later observed, Decision 2 flips it to `verified`. We accept
the brief `failed` state and the possibility of Fleet's existing Windows resend (`MaxWindowsProfileRetries = 1`)
re-enqueuing the profile. We do not attempt transient/permanent classification (Non-Goals). Alternative considered:
record the error into `detail` but keep `verifying` until a terminal 4xx (Option B) — rejected because the device does
not reliably self-retry hard failures, so holding at `verifying` would hide a real failure longer than surfacing it and
letting it self-heal.

### Decision 4: Absence is never a failure signal

Empty and unsupported reports already no-op the ingestion path: `directIngestHostCertificatesWindows` returns early when
it parses zero certs, and the `certificates_windows` query is discovery-gated on the `subject2` column so it never runs
on older agents. Therefore "no matching cert" produces no signal at all, not a "cert absent" signal, and we deliberately
never infer failure from absence. This makes offline hosts, old agents, empty stores, and logged-off users all resolve
to "stay `verifying`" for free, with no extra code.

### Decision 5: Detail string format matches Fleet's existing profile-failure style

Reuse the operation-scoped, status-bearing shape Fleet already uses. Device-reported Windows failures are formatted as
`<LocURI>: status <code>` joined by `, ` (`BuildMDMWindowsProfilePayloadFromMDMResponse`), and the proxy already wraps
upstream errors per operation (`Could not GetCACaps/GetCACert from SCEP server ...`). Proxy-observed SCEP failures use:

```
SCEP <operation> failed: <reason>
```

where `<operation>` is `GetCACaps`, `GetCACert`, or `PKIOperation`, and `<reason>` is `HTTP <code>` for an upstream HTTP
status, or a short classification otherwise (`timeout`, `malformed PKCS#7 response`, `connection refused`). Examples:
`SCEP PKIOperation failed: HTTP 500`, `SCEP GetCACert failed: timeout`. Best-practice constraints: human-readable, names
the failing operation and the upstream status, no secrets or PII (never include the SCEP challenge; do not dump the full
proxy URL). The string is stable and documented in the REST API reference. Rejected: a raw wrapped-error dump (leaks URL
and is not stable) and a numeric-only code (not actionable for admins).

### Decision 6: Retries follow the existing "verified preserves, only explicit resend resets" convention

Fleet's retry conventions: a SyncML failure auto-retries (`retries++`, capped at `MaxWindowsProfileRetries`, status set
NULL to re-enqueue, `microsoft_mdm.go`); an explicit user-initiated resend fully resets (`status=NULL, command_uuid='',
detail='', retries=0`, mirroring Apple `ResendHostMDMProfile`); a system-driven renewal/challenge resend deliberately
preserves retries (`retries = retries -- avoid endlessly resending`); and the `verified` transition never touches
retries. Applying that here:

- A proxy-observed failure sets `status=failed` + `detail` and does NOT increment retries. It is not the SyncML
  auto-retry path, and the device's own SCEP CSP retry handles transient upstream blips, so Fleet must not also
  re-enqueue on top of it.
- The `failed`/`verifying` -> `verified` self-heal on certificate observation sets `status=verified`, clears `detail`,
  and preserves `retries` (matches "verified preserves retries"; only an explicit user resend resets to 0).
- Manual resend from the UI keeps its existing behavior (full reset), unchanged by this change.

## Risks / Trade-offs

- [Profiles could sit in `verifying` forever when verification is impossible (old osquery, user never logs in)] → This
  is intended and documented; it is strictly more honest than today's false `verified`. Contributor docs and the feature
  guide will state the osquery 5.23.1+ requirement and the user-login dependency for `./User/...` profiles.
- [`failed` -> `verified` self-heal is an unusual transition and may briefly surface a failure that later succeeds] →
  Accepted per Option A; the alternative hides real failures. The `detail` makes the reason auditable.
- [Reworking the `verified`-gated stuck-recovery branch in `host_certificates.go` risks regressing the renewal-backfill
  behavior from issue #44111] → Cover with targeted tests for both first-observation (`verifying` -> `verified`) and the
  existing stale-renewal recovery (already `verified`) so neither path breaks.
- [Verification lags the SCEP exchange by up to the detail-query interval (~1h)] → Acceptable; Refetch forces an
  immediate report. No timeout depends on this interval, so lag cannot cause a false failure.
- [Adding a datastore call at response-save time to detect proxied SCEP profiles adds per-response cost] → Bounded to
  Windows profile responses; can be folded into the existing status-update query/transaction rather than a separate
  round trip.

## Migration Plan

No database migration. All state reuses `host_mdm_windows_profiles.status`/`detail`/`retries` and
`host_mdm_managed_certificates`. Rollout is behavioral: after deploy, newly ACKed proxied SCEP profiles enter
`verifying`; existing rows already at `verified` are left as-is and are re-evaluated naturally on the next cert
observation or renewal. Rollback is reverting the code; no data cleanup required. Confirm Fleet Premium gating for SCEP
proxy is intact on both backend and frontend.

## Open Questions

- Whether the profile-status flip in `UpdateHostCertificates` should be batched with the existing hmmc update in one
  statement/transaction, or run as a follow-on update keyed by the matched profile UUIDs. (Implementation detail;
  resolve during task 3.4.)
