// Package etag_invalidate provides a fleet.Datastore decorator that fires
// osquery config ETag invalidations (see server/service/redis_config_etag)
// after every successful write that can change a host's rendered config.
//
// WHY A DATASTORE DECORATOR, AND WHAT IT PROTECTS //
// The Redis-backed config ETag SHORT CIRCUIT (see
// fleet.OsqueryService.GetClientConfigWithETag) serves "unchanged" responses without
// building the config. Its correctness therefore depends ENTIRELY on this
// decorator seeing every write that can change the rendered osquery config.
// Hooking here — rather than in service methods — means core service,
// ee/server/service, GitOps batch endpoints, and the async host-processing
// collector are all covered by construction, because they all funnel through
// the fleet.Datastore interface.
//
// Two invalidation kinds exist, with very different blast radii:
//
//   - DEPLOYMENT-WIDE (Invalidate: gen bump + 3m write fence): config/report
//     mutations, label deletion, and manual membership edits whose affected
//     host set is unknown. These are INFREQUENT administrative operations.
//   - PER-HOST (InvalidateHost: record DEL + 30s publish quarantine): label
//     result persistence and manual membership changes with known host IDs.
//     These are CONTINUOUS at fleet scale.
//
// HARD RULE The deployment-wide invalidation must be UNREACHABLE from
// the routine label evaluation paths (RecordLabelQueryExecutions and the
// async collector batches). Those run continuously across the fleet; routing
// them through the deployment generation would re-arm the write fence
// forever — the optimization would never warm while still paying Redis
// overhead on every request. Both routine paths know their host IDs, so they
// never need the fallback. Each deployment-wide arming logs its source
// (bounded, since arming is infrequent by design) so a violation is visible
// in production logs.
//
// MAINTENANCE CONTRACT If you add or change a datastore method that
// affects any of the following, you MUST add it to this decorator:
//
//   - global agent options, features, or server settings that alter the
//     osquery config (queries feeding SaveAppConfig — including
//     queryReportsDisabled, which changes report eligibility),
//   - team agent options / team features (NewTeam/SaveTeam/DeleteTeam),
//   - report (query) schedules or their label scoping, global or team (the
//     query CRUD methods, which feed ListScheduledQueriesForAgents; these
//     also reset the cached label-scope mode state),
//   - 2017 "legacy" packs and their scheduled queries (the pack methods;
//     these also reset the cached legacy-packs gate flag),
//   - labels: deletion (cascades query_labels away, instantly UNSCOPING
//     reports with no query CRUD and no membership persistence — a
//     systematic gap unless hooked), spec application (platform changes
//     delete membership; manual-label specs replace it), and manual
//     membership edits,
//   - host label membership persistence (per-host invalidation): the
//     synchronous path, the async collector path, AND the host-vitals label
//     cron (UpdateLabelMembershipByHostCriteria, which returns the changed
//     host IDs for exactly this purpose).
//
// A missed hook does NOT poison ETags forever — every stored record carries
// a backstop TTL — but hosts would be told "unchanged" against stale configs for up to that TTL.
// When in doubt, hook the method: over-invalidating costs minutes of the
// optimization, never correctness.
//
// Invalidation failures are logged and swallowed: an unreachable Redis must
// never fail a datastore write. That is safe because the ETag fast path
// itself fails open (it degrades to full config builds when Redis is
// unavailable), and stored records expire via their backstop TTLs.
package etag_invalidate
