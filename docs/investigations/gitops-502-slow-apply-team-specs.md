# Investigation: GitOps 502 — Slow `POST /api/latest/fleet/spec/teams`

## Context

A customer is getting a 502 error when running `fleetctl gitops`. The `POST /api/latest/fleet/spec/teams` request takes 107 seconds, gets cancelled by Go context, and the AWS load balancer returns 502. Deadlocks are detected in `hosts` and `label_membership` tables.

**Key data point**: One of the customer's teams has **70K hosts**. This team is likely the problematic one, as the bottleneck scales with total host count.

## Root Cause: `mdmAppleBatchSetHostDeclarationStateDB` processes ALL hosts

The call chain is:

```
ApplyTeamSpecs (loops per team)
  → editTeamFromSpec
    → mdmAppleEditedAppleOSUpdates (macOS/iOS/iPadOS if updates edited)
      → SetOrUpdateMDMAppleDeclaration
      → BulkSetPendingMDMHostProfiles    ← starts a withRetryTxx transaction
        → SELECT uuid, platform FROM hosts WHERE team_id IN (?)
        → bulkSetPendingMDMAppleHostProfilesDB (for apple hosts)
        → mdmAppleBatchSetHostDeclarationStateDB   ← THE BOTTLENECK
          → mdmAppleGetHostsWithChangedDeclarationsDB   ← full-fleet scan
```

### The expensive operation

`mdmAppleGetHostsWithChangedDeclarationsDB` (`apple_mdm.go:5866`) runs a massive query that computes the desired declaration state for **ALL Apple hosts in the entire fleet**, not just the team being edited. It calls `generateDesiredStateQuery("declaration")` (`apple_mdm.go:2973`) which is a 4-way UNION:

1. Non label-based declarations → JOINs `mdm_apple_declarations` × `hosts` × `nano_enrollments` × `nano_devices`
2. Include-all label declarations → same JOINs + `label_membership`
3. Exclude-any label declarations → same JOINs + `label_membership` + `labels`
4. Include-any label declarations → same JOINs + `label_membership`

Each sub-query JOINs declarations with hosts on `h.team_id = mae.team_id` (cross-team match). The host filter is `TRUE` (no filtering), so it scans every enrolled Apple host.

There's a known TODO at `mdm.go:910-917`:
```go
// TODO(roberto): this method currently sets the state of all
// declarations for all hosts. I don't see an immediate concern
// ... but this could be optimized to use only a provided
// set of host uuids.
```

### Why it causes deadlocks

This runs inside a `withRetryTxx` transaction (`mdm.go:697`), which:
1. **Reads** from `hosts` and `label_membership` (via the desired state query)
2. **Writes** to `host_mdm_apple_declarations` (via `mdmAppleBatchSetPendingHostDeclarationsDB`)

Concurrently, osquery operations from checking-in hosts:
1. **Write** to `hosts.label_updated_at` (via `AsyncBatchUpdateLabelTimestamp` in `labels.go:1530`)
2. **Write** to `label_membership` (via `AsyncBatchInsertLabelMembership` in `labels.go:1483`)

The conflicting lock ordering on `hosts` and `label_membership` causes deadlocks.

### Multiplicative effect

If the customer has N teams with macOS update settings, `BulkSetPendingMDMHostProfiles` runs N times (once per team in the `ApplyTeamSpecs` loop). Each invocation triggers the full-fleet declaration scan. With iOS/iPadOS updates too, it could be up to 3N times.

## Key Files

| File | Line | Function |
|------|------|----------|
| `ee/server/service/teams.go` | 1010 | `ApplyTeamSpecs` — loops through team specs |
| `ee/server/service/teams.go` | 1356 | `editTeamFromSpec` — per-team edit logic |
| `ee/server/service/mdm.go` | 1258 | `mdmAppleEditedAppleOSUpdates` — calls BulkSetPending |
| `server/datastore/mysql/mdm.go` | 692 | `BulkSetPendingMDMHostProfiles` — transaction wrapper |
| `server/datastore/mysql/mdm.go` | 707 | `bulkSetPendingMDMHostProfilesDB` — orchestrator |
| `server/datastore/mysql/mdm.go` | 919 | call to `mdmAppleBatchSetHostDeclarationStateDB` — full-fleet scan |
| `server/datastore/mysql/apple_mdm.go` | 5866 | `mdmAppleGetHostsWithChangedDeclarationsDB` — the expensive query |
| `server/datastore/mysql/apple_mdm.go` | 2973 | `generateDesiredStateQuery` — 4-UNION desired state |
| `server/datastore/mysql/apple_mdm.go` | 5636 | `mdmAppleBatchSetHostDeclarationStateDB` — processes results |

## Fix: Scope declaration state queries to affected hosts only

The profiles version (`bulkSetPendingMDMAppleHostProfilesDB`) already correctly scopes to specific hosts via `h.uuid IN (?)`. The declarations version does not. The fix mirrors the profiles pattern.

### Current behavior (broken)

In `bulkSetPendingMDMHostProfilesDB` (`mdm.go:860`):
```go
if len(hosts) == 0 && !hasAppleDecls {
    // fetch hosts from DB
}
```
When `hasAppleDecls` is true, host fetching is **skipped**, and `appleHosts` is empty. Then `mdmAppleBatchSetHostDeclarationStateDB` (line 919) runs the full-fleet desired state query with `"TRUE"` as the host condition.

### Changes required

#### 1. Fetch hosts for declarations (`mdm.go`)

In `bulkSetPendingMDMHostProfilesDB`, add a case for `hasAppleDecls` in the host-fetching switch, and remove the `!hasAppleDecls` guard:

```go
case hasAppleDecls:
    uuidStmt = `
        SELECT DISTINCT h.uuid, h.platform
        FROM hosts h
        JOIN mdm_apple_declarations mad
            ON h.team_id = mad.team_id OR (h.team_id IS NULL AND mad.team_id = 0)
        WHERE
            mad.declaration_uuid IN (?)
            AND (h.platform = 'darwin' OR h.platform = 'ios' OR h.platform = 'ipados')`
    args = append(args, profileUUIDs)
```

Change line 860 from `if len(hosts) == 0 && !hasAppleDecls` to `if len(hosts) == 0`.

#### 2. Pass host UUIDs to declaration state function (`mdm.go`)

Change the call at line 919 to pass `appleHosts`:
```go
_, updates.AppleDeclaration, err = mdmAppleBatchSetHostDeclarationStateDB(ctx, tx, batchSize, nil, appleHosts)
```

#### 3. Parameterize declaration state queries (`apple_mdm.go`)

**`mdmAppleBatchSetHostDeclarationStateDB`** (line 5636): Add `hostUUIDs []string` parameter, forward to `mdmAppleGetHostsWithChangedDeclarationsDB`.

**`mdmAppleGetHostsWithChangedDeclarationsDB`** (line 5866): Add `hostUUIDs []string` parameter. When provided, build `hostCondition = "h.uuid IN (?)"` with args; when empty, use `"TRUE"`. Pass the condition to both `generateEntitiesToInstallQuery` and `generateEntitiesToRemoveQuery` for declarations, and use `sqlx.In` for the host UUIDs. **Critical**: wrap the UNION ALL (install + remove) in an outer `SELECT * FROM (...) WHERE host_uuid IN (?)` filter — this prevents the remove query's RIGHT JOIN from picking up declarations for hosts outside the specified set.

**`generateEntitiesToInstallQuery`** (line 3188): Refactor the `hostUUID string` parameter to `hostCondition string`. Currently it builds the condition internally:
```go
hostCondition := "TRUE"
if hostUUID != "" {
    hostCondition = fmt.Sprintf("h.uuid = '%s'", hostUUID)
}
```
Instead, accept the condition directly. Update the two callers:
- `listMDMAppleProfilesToInstallTransaction` (line 3300): pass `fmt.Sprintf("h.uuid = '%s'", hostUUID)` or `"TRUE"`.
- `mdmAppleGetHostsWithChangedDeclarationsDB` (line 5894): pass the condition built from `hostUUIDs`.

**`generateEntitiesToRemoveQuery`** (line 3243): Add a `hostCondition string` parameter (currently hardcodes `"TRUE"`). Update callers similarly.

#### 4. Batch processing for large teams

Since one team has 70K hosts, the declaration state query with `h.uuid IN (70K values)` could still be large. Use the same batching pattern as profiles (`selectProfilesBatchSize = 10_000`). Process declaration state in batches of 10K host UUIDs.

#### 5. Update all callers

The following callers need to be updated to match new signatures:
- `bulkSetPendingMDMHostProfilesDB` → `mdmAppleBatchSetHostDeclarationStateDB` (pass `appleHosts`)
- `listMDMAppleProfilesToInstallTransaction` → `generateEntitiesToInstallQuery` (pass condition)
- `listMDMAppleProfilesToRemoveTransaction` → `generateEntitiesToRemoveQuery` (pass condition)
- Any other callers of `mdmAppleBatchSetHostDeclarationStateDB` (check for `"TRUE"` as default)

### Files to modify

| File | Changes |
|------|---------|
| `server/datastore/mysql/mdm.go` | Add declaration host lookup, remove `!hasAppleDecls` guard, pass `appleHosts` to decl function |
| `server/datastore/mysql/apple_mdm.go` | Parameterize `generateEntitiesToInstallQuery`, `generateEntitiesToRemoveQuery`, `mdmAppleGetHostsWithChangedDeclarationsDB`, `mdmAppleBatchSetHostDeclarationStateDB` with host UUIDs |

### Verification

1. **Unit tests**: `make run-go-tests PKG_TO_TEST="server/datastore/mysql" TESTS_TO_RUN="TestBulkSetPendingMDMHostProfiles"`
2. **Integration tests**: Run `make test-go CI_TEST_PKG=mysql` focusing on MDM declaration tests
3. **Manual test**: Set up a multi-team environment, edit OS update settings via `fleetctl gitops`, verify the declaration state query now includes `h.uuid IN (...)` instead of `TRUE`
4. **Performance**: With the fix, a gitops run for a customer with 70K hosts should complete in seconds instead of 107+ seconds, because only the affected team's hosts are queried
