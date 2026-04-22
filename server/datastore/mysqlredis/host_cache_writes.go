package mysqlredis

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// This file holds the mysqlredis overrides for write-path methods that mutate
// cached host fields. Each wrapper delegates to the inner Datastore first, then
// invalidates the Redis-backed host cache on success. Errors from the inner
// call short-circuit invalidation — we must not poison the cache on transient
// failures.
//
// Invalidation reason labels must be one of the low-cardinality values the
// metric in metrics.go expects: update | enroll | team | delete | cert.
//
// NOTE: If hostCacheEnabled is false, every helper used here is a no-op and
// these wrappers behave identically to the inner methods.

// UpdateHost wraps the inner UpdateHost to invalidate the cache entry for the
// host's current node_key after a successful write. The old node_key entry (if
// re-enrollment changed it) becomes unreachable and expires via TTL.
func (d *Datastore) UpdateHost(ctx context.Context, host *fleet.Host) error {
	if err := d.Datastore.UpdateHost(ctx, host); err != nil {
		return err
	}
	d.invalidateAfterHostWrite(ctx, host, "update")
	return nil
}

// SerialUpdateHost wraps the inner SerialUpdateHost. SerialUpdateHost enqueues
// to the datastore's writeCh and blocks until the write completes, so on
// return the UPDATE has already hit MySQL and we can safely invalidate. This
// wrap is essential because the writeCh loop in mysql.go invokes the inner
// UpdateHost directly and bypasses any wrapper on that method.
func (d *Datastore) SerialUpdateHost(ctx context.Context, host *fleet.Host) error {
	if err := d.Datastore.SerialUpdateHost(ctx, host); err != nil {
		return err
	}
	d.invalidateAfterHostWrite(ctx, host, "update")
	return nil
}

// UpdateHostOsqueryIntervals invalidates by host ID after the inner write.
// Affected fields: distributed_interval, config_tls_refresh, logger_tls_period.
func (d *Datastore) UpdateHostOsqueryIntervals(ctx context.Context, hostID uint, intervals fleet.HostOsqueryIntervals) error {
	if err := d.Datastore.UpdateHostOsqueryIntervals(ctx, hostID, intervals); err != nil {
		return err
	}
	d.hostCacheDeleteByID(ctx, hostID, "update")
	return nil
}

// UpdateHostRefetchRequested invalidates by host ID after the inner write.
// Affected field: refetch_requested. Staleness would cause admin-triggered
// refetches to be delayed by up to TTL; invalidation keeps that latency low.
func (d *Datastore) UpdateHostRefetchRequested(ctx context.Context, hostID uint, value bool) error {
	if err := d.Datastore.UpdateHostRefetchRequested(ctx, hostID, value); err != nil {
		return err
	}
	d.hostCacheDeleteByID(ctx, hostID, "update")
	return nil
}

// UpdateHostRefetchCriticalQueriesUntil invalidates by host ID after the inner
// write. Affected field: refetch_critical_queries_until.
func (d *Datastore) UpdateHostRefetchCriticalQueriesUntil(ctx context.Context, hostID uint, until *time.Time) error {
	if err := d.Datastore.UpdateHostRefetchCriticalQueriesUntil(ctx, hostID, until); err != nil {
		return err
	}
	d.hostCacheDeleteByID(ctx, hostID, "update")
	return nil
}

// EnrollOrbit invalidates for the returned host on successful enrollment. Orbit
// enrollment may create a new hosts row or update an existing one's
// orbit_node_key + team_id. In either case the cached snapshot is stale after
// the call.
func (d *Datastore) EnrollOrbit(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
	host, err := d.Datastore.EnrollOrbit(ctx, opts...)
	if err != nil {
		return nil, err
	}
	d.invalidateAfterHostWrite(ctx, host, "enroll")
	return host, nil
}

// AddHostsToTeam invalidates every host in the batch after a successful team
// reassignment. The invalidation is intentionally synchronous so test harnesses
// observe a stable cache state on return; operators running very large batches
// (> ~1000 hosts) will see proportionally longer write latency today. A later
// optimization could pipeline the Redis invalidations to reduce that latency
// without changing the synchronous semantics of this method.
func (d *Datastore) AddHostsToTeam(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
	if err := d.Datastore.AddHostsToTeam(ctx, params); err != nil {
		return err
	}
	if !d.hostCacheEnabled || params == nil {
		return nil
	}
	for _, id := range params.HostIDs {
		d.hostCacheDeleteByID(ctx, id, "team")
	}
	return nil
}

// UpdateHostIdentityCertHostIDBySerial invalidates for the host whose
// has_host_identity_cert just flipped. This is the security-critical path: a
// stale `false` would cause AuthenticateHost to skip the httpsig verification
// for up to TTL. Explicit invalidation closes that window to a round-trip.
func (d *Datastore) UpdateHostIdentityCertHostIDBySerial(ctx context.Context, serialNumber uint64, hostID uint) error {
	if err := d.Datastore.UpdateHostIdentityCertHostIDBySerial(ctx, serialNumber, hostID); err != nil {
		return err
	}
	d.hostCacheDeleteByID(ctx, hostID, "cert")
	return nil
}

// invalidateAfterHostWrite is the common tail for write paths that hand us a
// *fleet.Host. Invalidates via the reverse indices on the host's ID — the
// only path that's guaranteed to clear BOTH cache families (osquery and
// orbit) regardless of which keys are populated on the caller-supplied Host
// struct. Some write paths hand us only a partial Host (e.g., the returned
// *Host from mysql.EnrollOrbit has ID but not NodeKey/OrbitNodeKey), and
// relying on direct-key DEL in those cases would silently miss the other
// family and leave it stale until TTL.
//
// Additionally clears direct-key entries for any keys the caller DOES have
// on the struct. This covers the case where a NotFound was negatively-cached
// under a key before the host existed — the reverse index can't find that
// entry (the host has no id-to-key mapping yet), so it must be cleared by
// the caller-supplied key. Matters most for NewHost / EnrollOsquery which
// can race with pre-enrollment probes.
func (d *Datastore) invalidateAfterHostWrite(ctx context.Context, host *fleet.Host, reason string) {
	if host == nil || host.ID == 0 {
		return
	}
	d.hostCacheDeleteByID(ctx, host.ID, reason)

	// Belt-and-braces clear of direct keys. Does not record a second
	// invalidation — hostCacheDeleteByID already did.
	nk := ""
	if host.NodeKey != nil {
		nk = *host.NodeKey
	}
	onk := ""
	if host.OrbitNodeKey != nil {
		onk = *host.OrbitNodeKey
	}
	d.hostCacheClearDirectEntries(ctx, nk, onk)
}
