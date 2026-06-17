## Context

Fleet already collects, stores, and serves host certificates for macOS, and the Host details page shows a
"Certificates" card for Apple hosts. The data path is shared infrastructure:

- Detail queries `certificates_darwin` and `certificates_windows` in `server/service/osquery_utils/queries.go`
  read osquery's `certificates` table and call `directIngestHostCertificates{Darwin,Windows}`.
- Storage is `host_certificates` (one row per host + SHA-1) plus `host_certificate_sources` (one row per
  `(host_certificate_id, source, username)`, `source ENUM('system','user')`).
- `GET /api/_version_/fleet/hosts/{id}/certificates` returns `HostCertificatePayload` (`source`, `username`, etc.).
- The frontend already fetches certificates for every host (`HostDetailsPage.tsx`, query `enabled` for all hosts).
  Only the card's render condition is gated to Apple, and the table column is hardcoded to "Keychain".

So the Windows backend ingestion exists but has never been surfaced in the UI, and a spike showed its scope mapping
is incorrect.

### Spike (real Windows host, osquery run as SYSTEM = Fleet's service context)

Running `SELECT store_location, sid, username, count(*) FROM certificates WHERE store='Personal' GROUP BY ...`
returned (94 rows, 27 distinct certs by SHA-1):

| store_location | sid | username | meaning |
|---|---|---|---|
| `LocalMachine` | `""` | `""` | machine-wide store (macOS `System.keychain` analog) |
| `Users` | `S-1-5-21-…-500` | `fleetadmin` | real interactive user (macOS `login.keychain` analog) |
| `CurrentUser` / `Services` / `Users` | `S-1-5-18` | `SYSTEM` | LocalSystem account store, enumerated three times |

Findings:

1. osquery returns **both** machine and user certificates from the Personal store in one query.
2. The current scope rule (`username == "SYSTEM"` → system, else user) is **wrong**: real `LocalMachine\Personal`
   certs report `username == ""`, so they are mislabeled `user` with a blank owner. Meanwhile the LocalSystem
   account's certs (a separate store, often holding device/enrollment certs) are the only ones tagged `system`.
3. The same certificate is reported up to ~3.5× across redundant hive views (`CurrentUser`, `Services`,
   `Users\S-1-5-18`, and `_Classes` sub-hives).
4. osquery (running as SYSTEM) can only see a user's Personal certs while that user's `HKEY_USERS` hive is loaded,
   i.e. while the user is logged in. macOS does not have this limitation because it reads keychain **files** on disk.

## Goals / Non-Goals

**Goals:**
- Show the Certificates card on the Host details page for Windows hosts, matching the macOS experience.
- Make the Windows System/User scope accurate and stable.
- Preserve a user's certificates across logoff (do not flap them in and out of the UI).
- Support multiple users on one host, each labeled with its own username.
- Rename "Keychain" → "Scope" across platforms; make help text platform-aware.

**Non-Goals:**
- No change to the REST API response shape (reuse `source` / `username`).
- No new database schema / migration.
- No fleetd, GitOps, or activity changes.
- Not surfacing non-Personal stores (Root/CA/Trust, etc.) — same as macOS, which shows identity certs only. The
  help text continues to point users at the raw `certificates` table for everything else.
- Not building a way to enumerate certificates of users who have never logged in (not possible via osquery).

## Decisions

### Decision 1: Derive scope from the registry hive (`sid` / `store_location`), not the owner string

Add `store_location` and `sid` to the `certificates_windows` query (keep `WHERE store='Personal'`). Classify:

- `sid` matches `S-1-5-21-*` (real domain/local accounts, including the `_Classes` sub-hive, which carries the same
  base SID) → **User** scope, `username` = the reported owner.
- everything else — `LocalMachine` (empty sid), and the well-known accounts `S-1-5-18` (SYSTEM), `S-1-5-19`
  (LOCAL SERVICE), `S-1-5-20` (NETWORK SERVICE), `.DEFAULT`, `S-1-5-80-*` (service SIDs) → **System** scope,
  `username` = `""`.

Rationale: the `S-1-5-21-*` prefix is the canonical marker of a real user principal; everything else is machine or
service context. This is robust to the fact that osquery runs as SYSTEM (so `CurrentUser` is always SYSTEM here).

_Alternative considered — keep matching `username == "SYSTEM"`:_ rejected; the spike proved it mislabels real
machine certs (blank username) and is the root bug.

_Alternative considered — map purely on `store_location` (LocalMachine=system, Users/CurrentUser=user):_ close, but
`Users` also contains `S-1-5-18/19/20` service hives that are not real users, so we still need the SID check. Using
the SID prefix as the primary signal subsumes both.

### Decision 2: Merge LocalMachine + system/service accounts into a single "System" scope

All System-classified rows for the same certificate (SHA-1) collapse to one `(source=system, username="")` source.
Rationale: this mirrors macOS's single System bucket, and LocalMachine vs LocalSystem-account is an implementation
detail an admin does not need. Crucially, the LocalSystem-account certs are **not** duplicates of LocalMachine certs
— they are a distinct store that commonly holds SCEP/NDES/autoenrolled device certs and Fleet-managed certs, so they
must be retained (folding, not dropping).

### Decision 3: Dedup source tuples by `(SHA-1, scope, username)`

After classification, dedup so the redundant `CurrentUser` / `Services` / `Users\<service>` / `_Classes` views
collapse. A certificate present in both System and a user, or in two users' stores, yields one source row per
distinct `(scope, username)` — exactly what `host_certificate_sources` already models. Keep the existing guard that
ignores an empty osquery result (do not wipe data on a transient empty report).

### Decision 4: User-aware reconciliation in `UpdateHostCertificates`

Today reconciliation (`server/datastore/mysql/host_certificates.go`) soft-deletes any existing osquery-origin
certificate whose SHA-1 is absent from the incoming batch, scoped only by `origin`. On Windows that would delete a
logged-off user's certs (their hive is not loaded, so they are simply absent), making certs disappear from the UI
even though they are still installed.

New rule: compute the set of **observed users** = usernames present in the incoming batch (System is always
observed because LocalMachine is always visible). Reconcile (i.e. allow soft-delete of) only within observed scope
groups; leave certificates belonging to non-observed users untouched. A genuine removal for a user is detected the
next time that user is logged in and reporting.

Because a certificate can have several sources, reconciliation must operate at `host_certificate_sources`
granularity: remove only the source rows whose `(scope, username)` group is observed-but-no-longer-reported, and
soft-delete a `host_certificates` row only once it has no remaining live sources. This must stay correct for macOS,
where every user's keychain is always present on disk, so every scope is "observed" on every run and behavior is
unchanged.

Watch the existing `FIXME` in `replaceHostCertsSourcesDB` about duplicate source tuples causing unique-constraint
violations — Decision 3's dedup must run before insert.

### Decision 5: Frontend reuse, minimal gating change

- Un-gate the card: `showCertificatesCard = (isAppleDeviceHost || isWindowsHost) && !!hostCertificates?.certificates.length`.
- Rename the column header "Keychain" → "Scope" (applies to macOS too); keep the existing cell logic (System, or
  User with the username in a tooltip).
- Make the table help text platform-aware (macOS: system + login keychains; Windows: the Personal certificate
  store); show it for Windows as well.
- The details modal reuses the same payload — verify it renders for Windows (no Windows-specific fields).

## Risks / Trade-offs

- **Reconciliation regression / cert flapping** → Mitigation: drive unit + datastore tests from the real spike data
  shapes (LocalMachine blank-username, S-1-5-18 SYSTEM triple-listed, S-1-5-21 real user, `_Classes` duplicate);
  explicitly test "user logs off → certs preserved", "user logs back in and cert removed → soft-deleted", and
  "macOS full report → unchanged".
- **Stale user certs after a real removal while logged off** → Accepted: Fleet shows last-known state until the user
  is next seen logged in. This matches the data the agent can actually observe and is better than hiding installed
  certs.
- **Users who never log in are invisible** → Accepted and documented in the spec; not solvable via osquery.
- **Performance on hosts with many users/large stores** → Same ingestion/list path as macOS, already in production;
  the only additions are two SELECT columns and per-user grouping. Note for load testing: the osquery-perf simulator
  does not emit Windows cert rows today (possible follow-up, flagged in the issue).
- **Shared datastore change touches macOS** → Mitigation: macOS always observes all scopes, so the user-aware
  reconciliation is a no-op for it; guard with a macOS regression test.

## Migration Plan

No database migration. Deploy is backend + frontend only. Rollback is a straight revert; because reconciliation only
becomes more conservative (it deletes strictly less), a rollback cannot leave orphaned/incorrect rows beyond what the
prior behavior already produced. On first ingestion after deploy, previously mislabeled Windows scopes are corrected
in place.

## Open Questions

- None blocking. Confirm during QA on the Azure Windows VM that built-in Administrator (`S-1-5-21-…-500`) showing as
  a User-scope row is the desired product presentation (current decision: yes, it is a real interactive account).
