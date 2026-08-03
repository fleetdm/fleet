// Package etag_invalidate provides a fleet.Datastore decorator that fires
// osquery config ETag invalidations (see server/service/redis_config_etag)
// after every successful write that can change a host's rendered config.
//
// ██ WHY A DATASTORE DECORATOR, AND WHAT IT PROTECTS ████████████████████████
//
// The Redis-backed config ETag SHORT CIRCUIT (see
// fleet.OsqueryService.GetClientConfigWithETag) serves 304 responses without
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
// ██ HARD RULE ██ The deployment-wide invalidation must be UNREACHABLE from
// the routine label evaluation paths (RecordLabelQueryExecutions and the
// async collector batches). Those run continuously across the fleet; routing
// them through the deployment generation would re-arm the write fence
// forever — the optimization would never warm while still paying Redis
// overhead on every request. Both routine paths know their host IDs, so they
// never need the fallback. Each deployment-wide arming logs its source
// (bounded, since arming is infrequent by design) so a violation is visible
// in production logs.
//
// ██ MAINTENANCE CONTRACT ██ If you add or change a datastore method that
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
//     systematic gap unless hooked) and manual membership edits,
//   - host label membership persistence (per-host invalidation): the
//     synchronous path, the async collector path, AND the host-vitals label
//     cron (UpdateLabelMembershipByHostCriteria, which returns the changed
//     host IDs for exactly this purpose).
//
// A missed hook does NOT poison ETags forever — every stored record carries
// a backstop TTL — but hosts would receive stale 304s for up to that TTL.
// When in doubt, hook the method: over-invalidating costs minutes of the
// optimization, never correctness.
//
// Invalidation failures are logged and swallowed: an unreachable Redis must
// never fail a datastore write. That is safe because the ETag fast path
// itself fails open (it degrades to full config builds when Redis is
// unavailable), and stored records expire via their backstop TTLs.
//
// ████████████████████████████████████████████████████████████████████████████
package etag_invalidate

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Datastore decorates a fleet.Datastore so that every successful
// config-affecting write fires the appropriate ETag invalidation (see the
// package docs for the deployment-wide vs per-host split).
type Datastore struct {
	fleet.Datastore
	store  fleet.ConfigETagStore
	logger *slog.Logger
	// errLogLast rate-limits invalidation-failure error logging (unix
	// seconds of the last emitted line). Per-host invalidations run at
	// fleet scale on every label cycle; during a Redis outage, unlimited
	// error logging would emit at that same volume.
	errLogLast atomic.Int64
}

// New wraps ds. The decorator is intentionally always-on whenever Redis is
// configured (even when the osquery.redis_config_etags feature flag is off)
// so the generation counter stays coherent — flipping the flag on later is
// then immediately safe.
func New(ds fleet.Datastore, store fleet.ConfigETagStore, logger *slog.Logger) *Datastore {
	return &Datastore{
		Datastore: ds,
		store:     store,
		logger:    logger,
	}
}

// logErrRateLimited logs invalidation failures at most once per 30 seconds
// per Fleet instance.
func (d *Datastore) logErrRateLimited(ctx context.Context, msg string, args ...any) {
	const minInterval = 30 // seconds
	now := time.Now().Unix()
	last := d.errLogLast.Load()
	if now-last >= minInterval && d.errLogLast.CompareAndSwap(last, now) {
		d.logger.ErrorContext(ctx, msg, args...)
	}
}

// invalidate bumps the deployment ETag generation and arms the write fence.
// Errors are logged, never returned: config delivery must not depend on
// Redis health, and the ETag store fails open (see package docs).
//
// Every arming logs its source at Info — deployment-wide invalidations are
// infrequent administrative operations by design, so this logging is
// bounded, and it is the production tripwire for the package's HARD RULE
// (routine label traffic must never appear here).
func (d *Datastore) invalidate(ctx context.Context, reason string) {
	if err := d.store.Invalidate(ctx); err != nil {
		d.logger.ErrorContext(ctx, "config etag invalidation failed; stored etags will expire via backstop TTL",
			"reason", reason, "err", err)
		return
	}
	d.logger.InfoContext(ctx, "config etag deployment invalidation (write fence armed)",
		"source", reason)
}

// invalidateAndResetLegacyFlag additionally drops the cached deployment-wide
// "legacy packs present" gate flag — pack CRUD is what adds/removes 2017
// packs, so the answer may have just changed in either direction.
func (d *Datastore) invalidateAndResetLegacyFlag(ctx context.Context, reason string) {
	d.invalidate(ctx, reason)
	if err := d.store.ResetLegacyPacksFlag(ctx); err != nil {
		d.logger.ErrorContext(ctx, "config etag legacy packs flag reset failed; flag will expire via its TTL",
			"reason", reason, "err", err)
	}
}

// invalidateAndResetLabelScopes additionally drops the cached label-scope
// mode state — query CRUD can add/remove label-scoped reports (query_labels
// rows) and label deletion can cascade them away, either of which changes
// the shared-vs-per-host mode selection. Prompt reset matters: a stale
// "shared" answer combined with a newly host-specific config could let
// team-shared keys be trusted once the fence expires.
func (d *Datastore) invalidateAndResetLabelScopes(ctx context.Context, reason string) {
	d.invalidate(ctx, reason)
	if err := d.store.ResetLabelScopes(ctx); err != nil {
		d.logger.ErrorContext(ctx, "config etag label scopes reset failed; state will expire via its TTL",
			"reason", reason, "err", err)
	}
}

// invalidateHost deletes the host's per-host ETag record and arms its 30s
// publish quarantine. Errors are rate-limited-logged and swallowed: label
// persistence must never fail because Redis is unreachable — the record's
// jittered backstop TTL bounds the resulting staleness. Rate limiting
// matters here specifically because this path runs at fleet scale on every
// label cycle; a Redis outage must not log at that volume (failure counts
// are in the store's invalidations_total{result="error"} metric).
func (d *Datastore) invalidateHost(ctx context.Context, hostID uint, reason string) {
	if err := d.store.InvalidateHost(ctx, hostID); err != nil {
		d.logErrRateLimited(ctx, "config etag host invalidation failed; record will expire via backstop TTL",
			"reason", reason, "host_id", hostID, "err", err)
	}
}

// invalidateHosts fans a per-host invalidation across the distinct hosts of
// a committed batch.
func (d *Datastore) invalidateHosts(ctx context.Context, hostIDs map[uint]struct{}, reason string) {
	for hostID := range hostIDs {
		d.invalidateHost(ctx, hostID, reason)
	}
}

// ///////////////////////////////////////////////////////////////////////////
// App config: global agent options, features, server settings
// (queryReportsDisabled, which changes report eligibility), all of which
// alter the rendered osquery config.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	if err := d.Datastore.SaveAppConfig(ctx, info); err != nil {
		return err
	}
	d.invalidate(ctx, "SaveAppConfig")
	return nil
}

// ///////////////////////////////////////////////////////////////////////////
// Teams: team agent options and team features live on the team record.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) NewTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	t, err := d.Datastore.NewTeam(ctx, team)
	if err != nil {
		return nil, err
	}
	d.invalidate(ctx, "NewTeam")
	return t, nil
}

func (d *Datastore) SaveTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	t, err := d.Datastore.SaveTeam(ctx, team)
	if err != nil {
		return nil, err
	}
	d.invalidate(ctx, "SaveTeam")
	return t, nil
}

func (d *Datastore) DeleteTeam(ctx context.Context, tid uint) error {
	if err := d.Datastore.DeleteTeam(ctx, tid); err != nil {
		return err
	}
	d.invalidate(ctx, "DeleteTeam")
	return nil
}

// ///////////////////////////////////////////////////////////////////////////
// Queries (reports): their schedules feed ListScheduledQueriesForAgents,
// which produces the "Global" and "team-<id>" packs in the config. Query
// CRUD also resets the cached label-scope mode state because it can add or
// remove LABEL-SCOPED reports (query_labels rows), which flips scopes
// between shared and per-host mode.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) ApplyQueries(ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{}) error {
	if err := d.Datastore.ApplyQueries(ctx, authorID, queries, queriesToDiscardResults); err != nil {
		return err
	}
	d.invalidateAndResetLabelScopes(ctx, "ApplyQueries")
	return nil
}

func (d *Datastore) NewQuery(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
	q, err := d.Datastore.NewQuery(ctx, query, opts...)
	if err != nil {
		return nil, err
	}
	d.invalidateAndResetLabelScopes(ctx, "NewQuery")
	return q, nil
}

func (d *Datastore) SaveQuery(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
	if err := d.Datastore.SaveQuery(ctx, query, shouldDiscardResults, shouldDeleteStats); err != nil {
		return err
	}
	d.invalidateAndResetLabelScopes(ctx, "SaveQuery")
	return nil
}

func (d *Datastore) DeleteQuery(ctx context.Context, teamID *uint, name string) error {
	if err := d.Datastore.DeleteQuery(ctx, teamID, name); err != nil {
		return err
	}
	d.invalidateAndResetLabelScopes(ctx, "DeleteQuery")
	return nil
}

func (d *Datastore) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	n, err := d.Datastore.DeleteQueries(ctx, ids)
	if err != nil {
		return n, err
	}
	d.invalidateAndResetLabelScopes(ctx, "DeleteQueries")
	return n, nil
}

// ///////////////////////////////////////////////////////////////////////////
// 2017 "legacy" packs and their scheduled queries. Pack CRUD also resets
// the cached "legacy packs present" gate flag, since it is exactly what
// adds/removes the hard-bypass blocker.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) ApplyPackSpecs(ctx context.Context, specs []*fleet.PackSpec) error {
	if err := d.Datastore.ApplyPackSpecs(ctx, specs); err != nil {
		return err
	}
	d.invalidateAndResetLegacyFlag(ctx, "ApplyPackSpecs")
	return nil
}

func (d *Datastore) NewPack(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
	p, err := d.Datastore.NewPack(ctx, pack, opts...)
	if err != nil {
		return nil, err
	}
	d.invalidateAndResetLegacyFlag(ctx, "NewPack")
	return p, nil
}

func (d *Datastore) SavePack(ctx context.Context, pack *fleet.Pack) error {
	if err := d.Datastore.SavePack(ctx, pack); err != nil {
		return err
	}
	d.invalidateAndResetLegacyFlag(ctx, "SavePack")
	return nil
}

func (d *Datastore) DeletePack(ctx context.Context, name string) error {
	if err := d.Datastore.DeletePack(ctx, name); err != nil {
		return err
	}
	d.invalidateAndResetLegacyFlag(ctx, "DeletePack")
	return nil
}

func (d *Datastore) NewScheduledQuery(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
	s, err := d.Datastore.NewScheduledQuery(ctx, sq, opts...)
	if err != nil {
		return nil, err
	}
	d.invalidate(ctx, "NewScheduledQuery")
	return s, nil
}

func (d *Datastore) SaveScheduledQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	s, err := d.Datastore.SaveScheduledQuery(ctx, sq)
	if err != nil {
		return nil, err
	}
	d.invalidate(ctx, "SaveScheduledQuery")
	return s, nil
}

func (d *Datastore) DeleteScheduledQuery(ctx context.Context, id uint) error {
	if err := d.Datastore.DeleteScheduledQuery(ctx, id); err != nil {
		return err
	}
	d.invalidate(ctx, "DeleteScheduledQuery")
	return nil
}

// ///////////////////////////////////////////////////////////////////////////
// Labels: deletion and manual membership edits.
// ///////////////////////////////////////////////////////////////////////////

// DeleteLabel is a REQUIRED hook, not an optional one: deleting a label
// cascades away its query_labels rows, which can instantly UNSCOPE a report
// (it then applies to every host) with no query CRUD and no membership
// persistence — a systematic invalidation gap otherwise. Label deletion is a
// rare admin action, so the deployment-wide invalidation costs nothing; the
// label-scope mode state is reset too, since the last label-scoped report of
// a scope may just have become unscoped.
func (d *Datastore) DeleteLabel(ctx context.Context, name string, filter fleet.TeamFilter) error {
	if err := d.Datastore.DeleteLabel(ctx, name, filter); err != nil {
		return err
	}
	d.invalidateAndResetLabelScopes(ctx, "DeleteLabel")
	return nil
}

// SaveLabel uses the deployment-wide invalidation: for manual labels the
// supplied host list contains only the NEW membership — hosts REMOVED from
// the label (including clearing the list entirely) are absent from it, so
// per-host invalidation of the supplied IDs would miss exactly the hosts
// whose config shrank. Label edits are infrequent administrative operations
// (inside the hard-rule carve-out), and this avoids building an old-vs-new
// membership diff for a rare path. Label-QUERY edits alone would not need a
// hook (membership changes flow through subsequent evaluation cycles), but
// over-invalidating here is free.
func (d *Datastore) SaveLabel(ctx context.Context, label *fleet.Label, hostIDs []uint, teamFilter fleet.TeamFilter) (*fleet.LabelWithTeamName, []uint, error) {
	l, ids, err := d.Datastore.SaveLabel(ctx, label, hostIDs, teamFilter)
	if err != nil {
		return nil, nil, err
	}
	d.invalidate(ctx, "SaveLabel")
	return l, ids, nil
}

// UpdateLabelMembershipByHostIDs replaces a manual label's membership with
// the given host list — same removal blindness as SaveLabel, same
// deployment-wide answer.
func (d *Datastore) UpdateLabelMembershipByHostIDs(ctx context.Context, label fleet.Label, hostIds []uint, teamFilter fleet.TeamFilter) (*fleet.Label, []uint, error) {
	l, ids, err := d.Datastore.UpdateLabelMembershipByHostIDs(ctx, label, hostIds, teamFilter)
	if err != nil {
		return nil, nil, err
	}
	d.invalidate(ctx, "UpdateLabelMembershipByHostIDs")
	return l, ids, nil
}

// UpdateLabelMembershipByHostCriteria recomputes a host-vitals label's
// membership. It runs from a 5-MINUTE CRON for every host-vitals label, so
// it must follow the HARD RULE like the routine osquery paths: never the
// deployment-wide invalidation (the fence would be re-armed nearly
// continuously), and not all-members invalidation either (that would cold
// every member every 5 minutes). The datastore method computes and returns
// exactly the hosts whose membership VALUE changed, inside its own
// transaction — invalidate precisely those.
func (d *Datastore) UpdateLabelMembershipByHostCriteria(ctx context.Context, hvl fleet.HostVitalsLabel) (*fleet.Label, []uint, error) {
	label, changedHostIDs, err := d.Datastore.UpdateLabelMembershipByHostCriteria(ctx, hvl)
	if err != nil {
		return nil, nil, err
	}
	for _, hostID := range changedHostIDs {
		d.invalidateHost(ctx, hostID, "UpdateLabelMembershipByHostCriteria")
	}
	return label, changedHostIDs, nil
}

// AddLabelsToHost and RemoveLabelsFromHost carry the affected host directly:
// per-host invalidation, no deployment-wide cost.
func (d *Datastore) AddLabelsToHost(ctx context.Context, hostID uint, labelIDs []uint) error {
	if err := d.Datastore.AddLabelsToHost(ctx, hostID, labelIDs); err != nil {
		return err
	}
	d.invalidateHost(ctx, hostID, "AddLabelsToHost")
	return nil
}

func (d *Datastore) RemoveLabelsFromHost(ctx context.Context, hostID uint, labelIDs []uint) error {
	if err := d.Datastore.RemoveLabelsFromHost(ctx, hostID, labelIDs); err != nil {
		return err
	}
	d.invalidateHost(ctx, hostID, "RemoveLabelsFromHost")
	return nil
}

// ///////////////////////////////////////////////////////////////////////////
// Routine label result persistence — PER-HOST invalidation only (see the
// HARD RULE in the package docs). Conservative by design: every successful
// persistence invalidates, with no membership value diff.
// ///////////////////////////////////////////////////////////////////////////

// RecordLabelQueryExecutions is the synchronous label persistence path. The
// invalidation runs only after the underlying write succeeds — invalidating
// before commit would let an intervening build republish pre-change
// membership before the write completes.
func (d *Datastore) RecordLabelQueryExecutions(ctx context.Context, host *fleet.Host, results map[uint]*bool, t time.Time, deferredSaveHost bool) error {
	if err := d.Datastore.RecordLabelQueryExecutions(ctx, host, results, t, deferredSaveHost); err != nil {
		return err
	}
	d.invalidateHost(ctx, host.ID, "RecordLabelQueryExecutions")
	return nil
}

// AsyncBatchInsertLabelMembership / AsyncBatchDeleteLabelMembership are the
// async collector persistence path: invalidate every distinct host in the
// committed batch (batch tuples are [labelID, hostID]). Invalidating when
// the agent first stages results in Redis would be insufficient — the
// collector may not have committed yet. AsyncBatchUpdateLabelTimestamp is
// deliberately NOT hooked: it touches timestamps only, never membership.
func (d *Datastore) AsyncBatchInsertLabelMembership(ctx context.Context, batch [][2]uint) error {
	if err := d.Datastore.AsyncBatchInsertLabelMembership(ctx, batch); err != nil {
		return err
	}
	d.invalidateHosts(ctx, distinctBatchHosts(batch), "AsyncBatchInsertLabelMembership")
	return nil
}

func (d *Datastore) AsyncBatchDeleteLabelMembership(ctx context.Context, batch [][2]uint) error {
	if err := d.Datastore.AsyncBatchDeleteLabelMembership(ctx, batch); err != nil {
		return err
	}
	d.invalidateHosts(ctx, distinctBatchHosts(batch), "AsyncBatchDeleteLabelMembership")
	return nil
}

// distinctBatchHosts extracts the distinct host IDs from async label
// membership batch tuples ([labelID, hostID]).
func distinctBatchHosts(batch [][2]uint) map[uint]struct{} {
	hosts := make(map[uint]struct{}, len(batch))
	for _, tup := range batch {
		hosts[tup[1]] = struct{}{}
	}
	return hosts
}
