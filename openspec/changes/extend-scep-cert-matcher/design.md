## Context

`UpdateHostCertificates` (`server/datastore/mysql/host_certificates.go`) ingests cert data from a host (osquery cert refetch on macOS / Windows, MDM `CertificateList` response on iOS). Pre-change, the matcher was gated on `len(toInsert) > 0`: it loaded the host's `host_mdm_managed_certificates` (hmmc) rows via `ListHostMDMManagedCertificates`, scanned newly-inserted certs for the `"fleet-" + profile_uuid` substring on cert subject CN/OU, and queued updates via `updateHostMDMManagedCertDetailsDB`.

Two limitations cause a class of silent failures:

1. The matcher only iterates `toInsert`. If the renewed cert was inserted on a prior call but the matcher missed updating hmmc (replica lag, transaction race, the matcher's read hit a stale replica that didn't yet have the hmmc row), subsequent calls see the cert as already-existing — `toInsert` is empty for it — and the matcher never gets another chance.
2. The matcher takes the first match it finds and breaks. With both an old pre-renewal cert and a new renewed cert reported in the same osquery cycle, iteration order can pick either.

The renewal cron's `HAVING validity_period IS NOT NULL` lock then permanently excludes any row stuck NULL, so the cron silently never re-attempts. Manual re-push is the only recovery — issue #44111.

## Goals / Non-Goals

**Goals:**

- Recover stuck-NULL `hmmc` rows from a renewed cert already in `host_certificates`, regardless of whether this `UpdateHostCertificates` call has any cert changes (`toInsert`/`toDelete` empty included).
- Preserve the in-flight synchronization the renewal cron relies on — do not regress an `hmmc` row by writing values from an old cert that's still present in `host_certificates` but predates the latest renewal.
- Keep the change additive in observable behavior: callers and tests that don't exercise stuck-NULL recovery should see the same outcomes as today.

**Non-Goals:**

- Solve every renewal-recovery scenario. Hosts that never call `UpdateHostCertificates` (truly offline / no fleetd) remain unrecoverable until they come back online — same as today. A periodic cron approach was considered and rejected because it has no "host is currently reachable" signal and would risk acting on stale `host_certificates` rows.
- Touch the renewal cron's `WHERE` / `HAVING` clauses, the challenge TTL, the `BulkUpsertMDMManagedCertificates` upsert, or the per-platform profile state machines. Those concerns are independent.
- Strengthen the matcher's renewal-ID matching itself (it remains a substring search of `"fleet-" + profile_uuid` against `subject_common_name` / `subject_org_unit`). Moving to a first-class link is out of scope.

## Decisions

### 1. Recovery happens inside the existing matcher, not in a new function

Adding recovery as a different code branch inside `UpdateHostCertificates` reuses the same data the matcher already needs (the host's hmmc rows and the full incoming cert set). A separate function would re-read both via additional queries.

**Alternative considered:** add a separate per-call backfill function with its own eligibility SELECT joining `hmmc` to the per-platform profile tables and a per-row SELECT against `host_certificates`. Rejected because it adds 1+N SELECTs per call and creates a second mechanism doing similar work — the kind of overlap that seeds the next bug.

**Alternative considered:** move recovery to a periodic cron. Rejected because a cron has no per-host "online and reporting fresh state" signal, so it can match against the OLD pre-renewal cert that's still in `host_certificates` and trigger spurious resends to an offline host. The act of `UpdateHostCertificates` being called is itself the freshness signal.

### 2. The matcher runs on every `UpdateHostCertificates` call

Previously gated on `len(toInsert) > 0`. That gate prevented stable-cert-list hosts from ever recovering: if a renewed cert was inserted on an earlier call but the hmmc row never got linked, no later call would surface that cert in `toInsert` again, and the row would stay stuck forever.

Dropping the gate means the matcher's load (one joined SELECT) hits every cert-refetch cycle on every host. The cost is documented under Risks below.

### 3. Two cert pools, picked per-`hmmc`-row

Per row, the cert pool to search depends on the row's state:

```
hostMDMManagedCert.NotValidAfter == nil
    && time.Since(hostMDMManagedCert.UpdatedAt) > hmmcBackfillGrace
    && (apple_status OR windows_status is 'verified')
        ↓
    pool = incomingBySHA1   (recovery — full reported inventory)

len(toInsertBySHA1) > 0
        ↓
    pool = toInsertBySHA1   (steady state — same scope as today's matcher)

otherwise
        ↓
    skip (no work for this row this call)
```

This is the load-bearing decision for the offline-after-renewal safety. Steady-state hmmc rows (already populated, or freshly NULL'd within the grace window, or in-flight pending/verifying, or terminally failed) keep today's "react only to NEW certs" semantics. They never get matched against a cert that was already present in `host_certificates` from a prior cycle, so the in-flight blank-out can't be undone by an old cert.

Stuck rows widen the search. They've been NULL longer than the grace AND the profile reached `'verified'`, so the renewal completed successfully on the platform's terms; the device is currently online and reporting (we wouldn't be in `UpdateHostCertificates` otherwise), so we trust the full reported inventory.

**Alternative considered:** drop all gating and always iterate `incomingBySHA1`. Rejected — when an in-flight row sees `incomingBySHA1` (because some unrelated cert changed at the same time), it could match against the OLD cert and clobber the lock.

**Alternative considered:** authorize pool-widening on `'failed'` too ("renewal is no longer legitimately in flight either way"). Rejected — see §8; it would re-introduce the loop that pre-PR behavior implicitly avoided.

### 4. "Best match wins" instead of "first match wins"

Today's matcher takes the first cert in `toInsert` whose subject matches the renewal-ID substring and `break`s. With the wider `incomingBySHA1` pool, both the OLD pre-renewal cert and the NEW renewed cert may be present at the same time — they share the same renewal ID. We pick the one with the latest `not_valid_before` among currently-valid certs (`not_valid_before <= NOW < not_valid_after`), so a freshly-issued cert always wins over the older one regardless of iteration order.

### 5. Monotonic-forward predicate

Before queuing an update, check:

```
hostMDMManagedCert.NotValidAfter != nil
    && !hostMDMManagedCert.NotValidAfter.Before(bestMatch.NotValidAfter)
        ↓ skip — would regress
```

Belt-and-suspenders: the pool selection in §3 should already prevent regression in normal cases, but the predicate is cheap and closes any gap (e.g., a custom CA that issued an out-of-order cert).

### 6. Symmetric in-memory access via `toInsertBySHA1`

`incomingBySHA1` is already a `map[string]*fleet.HostCertificateRecord` (lines 38-67 in `UpdateHostCertificates`). A parallel `toInsertBySHA1` map is built alongside `toInsert` in the diff loop so the two pools share the same access pattern.

### 7. Joined SELECT replaces `ListHostMDMManagedCertificates`

The stuck-row check in §3 needs the related profile's delivery status. The matcher inlines a SELECT that `LEFT JOIN`s `host_mdm_managed_certificates` to `host_mdm_apple_profiles` and `host_mdm_windows_profiles` (filtered by `operation_type = 'install'`) and returns both the hmmc fields and the per-platform status. Both joins target indexed PKs — same query count as before for cert-changing calls, and one extra SELECT for cert-stable calls (the new always-run cost).

### 8. Recovery aligns with the platform's terminal-on-failure contract

SCEP failures are terminal across the rest of the platform: initial-delivery failures sit at `hp.status='failed'` until an admin POSTs to `/configuration_profiles/resend/`, and the resend endpoint itself only allows `'failed'` or `'verified'` as the starting state — implicit acknowledgment that those are the two admin-actionable states, not interchangeable safe-to-auto-recover states.

Pre-PR renewal cron behavior on a permanently-broken SCEP server was self-quieting: one cycle blanked hmmc, SCEP failed, hmmc stayed NULL, the cron's `HAVING validity_period IS NOT NULL` excluded the row from every subsequent tick. Admin resend was the recovery path, by design.

If the matcher widens its pool when `hp.status='failed'`, the OLD cert (still in inventory) becomes `bestMatch`, hmmc gets re-populated from the OLD cert's validity, the cron's HAVING gate stops blocking the row, and the next cron tick re-pushes the profile — every hour, until SCEP is fixed. That converts a designed silent fail into a noisy MDM-push hot loop and breaks the platform's "failure means wait for admin" contract.

Therefore `'verified'` is the only profile state that authorizes pool-widening: it's the platform's signal that the renewal cycle finished successfully, so a missed link in `host_certificates` is the right thing to recover from. `'failed'` is excluded.

**Alternative considered:** anchor recovery on `bestMatch.NotValidBefore > hmmc.UpdatedAt` ("only recover if the candidate cert was issued after the last reconciliation pulse"). Rejected — closes the same loop and is technically tighter, but introduces a "we know better than the rest of the platform" mechanism instead of mirroring the existing terminal-on-failure contract. Not worth the cleverness when status-based gating is already correct and simpler to reason about.

## Risks / Trade-offs

- **[Hot-path cost]** The matcher's joined SELECT now runs on every `UpdateHostCertificates` call. At 100K hosts with default cert-refetch cadence this is ~28 SELECTs/sec sustained on the read pool. Each is host-uuid-keyed against indexed PKs and returns 0+ small rows. → To be load-tested before merging. If load shows this is unacceptable, fallback is to gate on `len(toInsert) > 0 || len(toDelete) > 0`, which loses scenario "stable cert list with stuck row" but preserves the rest.
- **[Pool selection logic complexity]** Three branches (stuck → recovery, has new certs → steady, else → skip) is more nuanced than today's single-pool matcher. Reviewers must understand the in-flight vs stuck distinction to trust the safety. → Mitigated by inline comments and this design doc.

## Open Questions

(resolved during implementation; all leftover items moved to PR description)
