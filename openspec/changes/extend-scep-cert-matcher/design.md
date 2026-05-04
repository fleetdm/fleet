## Context

`UpdateHostCertificates` (`server/datastore/mysql/host_certificates.go`) ingests cert data from a host (osquery cert refetch on macOS / Windows, MDM `CertificateList` response on iOS). When the cert list has changed, it loads the host's `host_mdm_managed_certificates` (hmmc) rows via `ListHostMDMManagedCertificates`, finds matches in the newly-inserted certs (`toInsert`) by `"fleet-" + profile_uuid` substring on cert subject CN/OU, and queues updates via `updateHostMDMManagedCertDetailsDB`.

The matcher only iterates `toInsert`. It also takes the first match it finds and breaks. These two limitations cause a class of silent failures: when the original toInsert match was missed for any reason — replica lag preventing the hmmc row from being visible to the matcher's read, a transaction race, or the renewed cert landing in `existingBySHA1` rather than `toInsert` — the row stays NULL forever. The renewal cron's `HAVING validity_period IS NOT NULL` lock then permanently excludes it, so the cron silently never re-attempts. Manual re-push is the only recovery, which motivated customer issue #44111.

The data needed to recover is already in memory: `ListHostMDMManagedCertificates` is already loaded, and `incomingBySHA1` (the merged "what the host currently has") is already constructed at lines 38-67 by the time the matcher runs. Recovery can happen in-place with no new SELECTs.

## Goals / Non-Goals

**Goals:**

- Recover stuck-NULL `hmmc` rows when the original toInsert match was missed (replica lag, race, cert was already in `host_certificates` so it didn't land in `toInsert`).
- Preserve the in-flight synchronization the renewal cron relies on — do not regress an `hmmc` row by writing values from an old cert that's still present in `host_certificates` but predates the latest renewal.
- Add no new database queries on the hot path. All recovery work uses data already loaded by `UpdateHostCertificates`.
- Keep the change additive in observable behavior: callers and tests that don't exercise stuck-NULL recovery should see the same outcomes as today.

**Non-Goals:**

- Solve every renewal-recovery scenario. Hosts that never call `UpdateHostCertificates` (truly offline / no fleetd) remain unrecoverable until they come back online — same as today. A periodic cron approach was considered and rejected because it has no "host is currently reachable" signal and would risk acting on stale `host_certificates` rows.
- Touch the renewal cron's `WHERE` / `HAVING` clauses, the challenge TTL, the `BulkUpsertMDMManagedCertificates` upsert, or the per-platform profile state machines. Those concerns are independent.
- Strengthen the matcher's renewal-ID matching itself (it remains a substring search of `"fleet-" + profile_uuid` against `subject_common_name` / `subject_org_unit`). Moving to a first-class link is out of scope.

## Decisions

### 1. Recovery happens inside the existing matcher, not in a new function

The matcher already runs `ListHostMDMManagedCertificates(ctx, hostUUID)` for every `UpdateHostCertificates` call where `len(toInsert) > 0`. Both the hmmc rows and the full incoming cert set (`incomingBySHA1`, built at lines 38-67) are in memory by the time the matcher starts. Adding recovery as a different code branch reuses both — zero new SELECTs.

**Alternative considered:** add a separate per-call backfill function that runs after the matcher, with its own eligibility SELECT joining `hmmc` to the per-platform profile tables and a per-row SELECT against `host_certificates`. Rejected because it adds 1+N SELECTs per cert-changing call and creates a second mechanism doing similar work in the same call — the kind of overlap that seeds the next bug.

**Alternative considered:** move recovery to a periodic cron. Rejected because a cron has no per-host "online and reporting fresh state" signal, so it can match against the OLD pre-renewal cert that's still in `host_certificates` and trigger spurious resends to an offline host. The act of `UpdateHostCertificates` being called is itself the freshness signal we want to gate on.

### 2. Two cert pools, picked per-hmmc-row

For each hmmc row the matcher iterates, the cert pool to search depends on the row's state:

```
hostMDMManagedCert.NotValidAfter == nil
    && time.Since(hostMDMManagedCert.UpdatedAt) > hmmcBackfillGrace
        ↓
    pool = incomingBySHA1   (recovery — full reported inventory)

otherwise
        ↓
    pool = toInsertBySHA1   (steady state — same scope as today's matcher)
```

This is the load-bearing decision for the offline-after-renewal safety. Steady-state hmmc rows (already populated, or freshly NULL'd within the grace window) keep today's matcher semantics: react only to NEW certs in this report. They never get matched against a cert that was already present in `host_certificates` from a prior cycle, so the in-flight blank-out can't be undone by an old cert.

Stuck rows widen the search. They've been NULL longer than the grace, the device is currently online and reporting (we wouldn't be in `UpdateHostCertificates` otherwise), so we trust the full reported inventory.

**Alternative considered:** drop the gate entirely, always iterate `incomingBySHA1`. Rejected — exposes the offline-after-renewal vulnerability described in §1.

**Alternative considered:** keep the gate but iterate `incomingBySHA1` always (any stuck rows opportunistically match). Rejected — when an in-flight row sees `incomingBySHA1` (because some unrelated cert changed at the same time), it could match against the OLD cert and clobber the lock.

### 3. "Best match wins" instead of "first match wins"

Today's matcher takes the first cert in `toInsert` whose subject matches the renewal-ID substring and `break`s. With the wider `incomingBySHA1` pool, both the OLD pre-renewal cert and the NEW renewed cert may be present at the same time — they share the same renewal ID. We pick the one with the latest `not_valid_before` among currently-valid certs (`not_valid_before <= NOW < not_valid_after`), so a freshly-issued cert always wins over the older one regardless of iteration order.

### 4. Monotonic-forward predicate

After picking the best match, before queuing an update, check:

```
hostMDMManagedCert.NotValidAfter != nil
    && !hostMDMManagedCert.NotValidAfter.Before(bestMatch.NotValidAfter)
        ↓ skip — would regress
```

This is belt-and-suspenders: the pool selection in §2 should already prevent regression in normal cases, but the predicate is cheap and closes any gap (e.g., a custom CA that issued an out-of-order cert).

### 5. Symmetric in-memory access via `toInsertBySHA1`

The existing matcher's loop is `for _, certToInsert := range toInsert`. To pick a pool dynamically per hmmc row, both `toInsert` and `incomingBySHA1` need the same shape. `incomingBySHA1` is already a `map[string]*fleet.HostCertificateRecord` (lines 38-67). Build a `toInsertBySHA1` map alongside `toInsert` in the diff loop at lines 88-116. Trivial — same memory, same population step.

## Risks / Trade-offs

- **[Coverage gap: hosts with stable cert lists]** The matcher only fires when `len(toInsert) > 0`. A host whose cert list never changes (rare in practice — system certs rotate, app installs add/remove certs) won't trigger recovery. → Document as a known limitation; if it bites, widen the trigger to also include `len(toDelete) > 0`.
- **[Recovery latency tied to osquery cycle]** Stuck rows recover on the next cert-changing osquery report from that host, which can be up to the host's osquery interval (default ~30 min). For renewal — which operates on day/week timescales — this is fine. → Accept.
- **[Pool selection logic obscures intent]** The "two pools, choose by row state" pattern is more nuanced than today's single-pool matcher. Reviewers must understand the in-flight vs stuck distinction to trust the safety. → Mitigated by inline comments explaining the gate semantics; the design.md (this doc) carries the rationale.

## Open Questions

- Should the gate also include `len(toDelete) > 0`? The cost is zero (just a wider trigger condition) and it covers the rare "only deletion in this report" case. Lean: include it.
- Is there value in logging when recovery actually fires (matcher takes the wide pool path)? Useful for operators investigating stuck-renewal incidents. Lean: add a `DebugContext` log keyed by `host_uuid` and `profile_uuid`.
