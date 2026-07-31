// Package etag_invalidate provides a fleet.Datastore decorator that fires
// osquery config ETag invalidations (see server/service/redis_config_etag)
// after every successful config-affecting write.
//
// ██ WHY A DATASTORE DECORATOR, AND WHAT IT PROTECTS ████████████████████████
//
// The Redis-backed config ETag SHORT CIRCUIT (see
// fleet.OsqueryService.GetClientConfigWithETag) serves 304 responses without
// building the config. Its correctness therefore depends ENTIRELY on this
// decorator seeing every write that can change the rendered osquery config.
// Hooking here — rather than in service methods — means core service,
// ee/server/service, and GitOps batch endpoints are all covered by
// construction, because they all funnel through the fleet.Datastore
// interface.
//
// ██ MAINTENANCE CONTRACT ██ If you add or change a datastore method that
// affects any of the following, you MUST add it to this decorator:
//
//   - global agent options, features, or server settings that alter the
//     osquery config (queries feeding SaveAppConfig),
//   - team agent options / team features (NewTeam/SaveTeam/DeleteTeam),
//   - report (query) schedules or their label scoping, global or team (the
//     query CRUD methods, which feed ListScheduledQueriesForAgents),
//   - 2017 "legacy" packs and their scheduled queries (the pack methods).
//
// A missed hook does NOT poison ETags forever — the stored records carry a
// backstop TTL (redis_config_etag.DefaultETagTTL) — but hosts would receive
// stale 304s for up to that TTL. When in doubt, hook the method:
// over-invalidating costs a few minutes of the optimization, never
// correctness.
//
// Invalidation failures are logged and swallowed: an unreachable Redis must
// never fail a datastore write. That is safe because the ETag fast path
// itself fails open (it degrades to full config builds when Redis is
// unavailable), and stored records expire via the backstop TTL.
//
// ████████████████████████████████████████████████████████████████████████████
package etag_invalidate

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Datastore decorates a fleet.Datastore so that every successful
// config-affecting write bumps the config ETag generation and arms the write
// fence (fleet.ConfigETagStore.Invalidate). Pack and query writes
// additionally reset the cached "short circuit blocked" gate flag, because
// they are what can create or remove the gate's blockers (2017 packs and
// label-scoped reports, respectively).
type Datastore struct {
	fleet.Datastore
	store  fleet.ConfigETagStore
	logger *slog.Logger
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

// invalidate bumps the ETag generation and arms the write fence. Errors are
// logged, never returned: config delivery must not depend on Redis health,
// and the ETag store fails open (see package docs).
func (d *Datastore) invalidate(ctx context.Context, reason string) {
	if err := d.store.Invalidate(ctx); err != nil {
		d.logger.ErrorContext(ctx, "config etag invalidation failed; stored etags will expire via backstop TTL",
			"reason", reason, "err", err)
	}
}

// invalidateAndResetBlockedFlag additionally drops the cached deployment-wide
// "short circuit blocked" answer — pack CRUD can add/remove 2017 packs, and
// query CRUD can add/remove label-scoped reports (query_labels), either of
// which changes the answer. Resetting promptly (rather than waiting out the
// flag TTL) matters: the flag lives longer than the write fence, and a stale
// "not blocked" answer combined with a newly host-specific config could
// poison team-shared ETag keys once the fence expires.
func (d *Datastore) invalidateAndResetBlockedFlag(ctx context.Context, reason string) {
	d.invalidate(ctx, reason)
	if err := d.store.ResetShortCircuitBlockedFlag(ctx); err != nil {
		d.logger.ErrorContext(ctx, "config etag blocked flag reset failed; flag will expire via its TTL",
			"reason", reason, "err", err)
	}
}

// ///////////////////////////////////////////////////////////////////////////
// App config: global agent options, features, server settings
// (queryReportsDisabled), all of which alter the rendered osquery config.
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
// CRUD also resets the gate flag because it can add or remove LABEL-SCOPED
// reports (query_labels rows), which block the short circuit entirely.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) ApplyQueries(ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{}) error {
	if err := d.Datastore.ApplyQueries(ctx, authorID, queries, queriesToDiscardResults); err != nil {
		return err
	}
	d.invalidateAndResetBlockedFlag(ctx, "ApplyQueries")
	return nil
}

func (d *Datastore) NewQuery(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
	q, err := d.Datastore.NewQuery(ctx, query, opts...)
	if err != nil {
		return nil, err
	}
	d.invalidateAndResetBlockedFlag(ctx, "NewQuery")
	return q, nil
}

func (d *Datastore) SaveQuery(ctx context.Context, query *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) error {
	if err := d.Datastore.SaveQuery(ctx, query, shouldDiscardResults, shouldDeleteStats); err != nil {
		return err
	}
	d.invalidateAndResetBlockedFlag(ctx, "SaveQuery")
	return nil
}

func (d *Datastore) DeleteQuery(ctx context.Context, teamID *uint, name string) error {
	if err := d.Datastore.DeleteQuery(ctx, teamID, name); err != nil {
		return err
	}
	d.invalidateAndResetBlockedFlag(ctx, "DeleteQuery")
	return nil
}

func (d *Datastore) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	n, err := d.Datastore.DeleteQueries(ctx, ids)
	if err != nil {
		return n, err
	}
	d.invalidateAndResetBlockedFlag(ctx, "DeleteQueries")
	return n, nil
}

// ///////////////////////////////////////////////////////////////////////////
// 2017 "legacy" packs and their scheduled queries. Pack CRUD also resets
// the cached "short circuit blocked" gate flag, since it is exactly what
// adds/removes the 2017-pack blocker.
// ///////////////////////////////////////////////////////////////////////////

func (d *Datastore) ApplyPackSpecs(ctx context.Context, specs []*fleet.PackSpec) error {
	if err := d.Datastore.ApplyPackSpecs(ctx, specs); err != nil {
		return err
	}
	d.invalidateAndResetBlockedFlag(ctx, "ApplyPackSpecs")
	return nil
}

func (d *Datastore) NewPack(ctx context.Context, pack *fleet.Pack, opts ...fleet.OptionalArg) (*fleet.Pack, error) {
	p, err := d.Datastore.NewPack(ctx, pack, opts...)
	if err != nil {
		return nil, err
	}
	d.invalidateAndResetBlockedFlag(ctx, "NewPack")
	return p, nil
}

func (d *Datastore) SavePack(ctx context.Context, pack *fleet.Pack) error {
	if err := d.Datastore.SavePack(ctx, pack); err != nil {
		return err
	}
	d.invalidateAndResetBlockedFlag(ctx, "SavePack")
	return nil
}

func (d *Datastore) DeletePack(ctx context.Context, name string) error {
	if err := d.Datastore.DeletePack(ctx, name); err != nil {
		return err
	}
	d.invalidateAndResetBlockedFlag(ctx, "DeletePack")
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
