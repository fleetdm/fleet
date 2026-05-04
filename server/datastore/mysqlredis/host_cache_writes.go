package mysqlredis

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// This file holds the mysqlredis overrides for write-path methods that mutate
// cached host fields. Each wrapper delegates to the inner Datastore first, then
// invalidates the Redis-backed host cache on success. Errors from the inner
// call short-circuit invalidation. We must not poison the cache on transient
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
	d.invalidateAfterHostUpdate(ctx, host, "update")
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
	d.invalidateAfterHostUpdate(ctx, host, "update")
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

// EnrollOrbit invalidates for the returned host on successful enrollment. Orbit enrollment may create a new
// hosts row or update an existing one's orbit_node_key + team_id; in either case the cached snapshot is stale.
//
// Asymmetry with EnrollOsquery: mysql.EnrollOsquery does a final SELECT that hydrates host.NodeKey and
// host.OrbitNodeKey on the returned struct, but mysql.EnrollOrbit does NOT, the *fleet.Host it returns has
// only ID/ComputerName/Hostname/HardwareModel/HardwareSerial/Platform/PlatformLike populated.
//
// Replay opts to recover the just-issued orbit_node_key so the negative-cache plug fires for orbit, matching
// the contract the helper is documented to provide. Without this, an /orbit/* probe arriving in the 0-5s
// window before EnrollOrbit committed leaves an onk_miss:<K> entry that survives the wrapper, and the
// freshly-enrolled host returns NotFound from the negative cache for up to hostCacheNegativeTTL on its
// first /orbit/* requests.
func (d *Datastore) EnrollOrbit(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
	host, err := d.Datastore.EnrollOrbit(ctx, opts...)
	if err != nil {
		return nil, err
	}
	cfg := &fleet.DatastoreEnrollOrbitConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if host != nil && host.ID != 0 {
		d.hostCacheDeleteByID(ctx, host.ID, "enroll")
		d.hostCacheClearDirectEntries(ctx, "", cfg.OrbitNodeKey)
	}
	return host, nil
}

// AddHostsToTeam invalidates every host in the batch after a successful team
// reassignment. Uses the pipelined batch invalidator (one MGET + one DEL per
// Redis slot, chunked) rather than calling hostCacheDeleteByID in a loop —
// that naive approach takes ~8 sequential Redis round-trips per host and at
// 10k hosts × ~1 ms RTT adds ~80 s to the API call. The batched version is
// O(slots × chunks) and stays synchronous on return.
func (d *Datastore) AddHostsToTeam(ctx context.Context, params *fleet.AddHostsToTeamParams) error {
	if err := d.Datastore.AddHostsToTeam(ctx, params); err != nil {
		return err
	}
	if params != nil {
		d.invalidateHostIDs(ctx, params.HostIDs, "team")
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

// invalidateAfterHostUpdate is the common tail for write paths that update an already-existing host
// (UpdateHost, SerialUpdateHost). Clears the cache via the reverse index on the host's ID, which covers
// both osquery and orbit families.
//
// This path does NOT clear by direct keys: an already-cached host's reverse index is populated, so the
// by-ID path finds and DELs every related key. The pre-enrollment-negative-cache race that motivates
// invalidateAfterHostEnroll's direct-keys clear cannot apply to UpdateHost callers (you cannot UPDATE a
// host that doesn't exist yet), so the extra DELs would be wasted Redis ops.
func (d *Datastore) invalidateAfterHostUpdate(ctx context.Context, host *fleet.Host, reason string) {
	if host == nil || host.ID == 0 {
		return
	}
	d.hostCacheDeleteByID(ctx, host.ID, reason)
}

// invalidateAfterHostEnroll is the common tail for write paths that may CREATE a host or rotate its
// node_key (NewHost, EnrollOsquery, EnrollOrbit). Does the by-ID invalidation that
// invalidateAfterHostUpdate does, plus a direct-keys clear using the caller-supplied NodeKey and
// OrbitNodeKey on the returned *fleet.Host.
//
// The direct-keys clear plugs the pre-enrollment-negative-cache race: if a poll arrived with the
// new node_key BEFORE the host row existed, LoadHostByNodeKey returned NotFound and wrote
// nk_miss:<new_key> with a 5s TTL but did NOT populate the reverse index (no host ID to point at).
// hostCacheDeleteByID's reverse-index walk cannot find that entry; only a direct DEL using the
// just-issued node_key can. Without the direct clear, the freshly-enrolled host's first 0-5s of
// auth attempts return NotFound from the negative cache.
//
// Does NOT record a second invalidation; hostCacheDeleteByID already bumped the counter.
func (d *Datastore) invalidateAfterHostEnroll(ctx context.Context, host *fleet.Host, reason string) {
	if host == nil || host.ID == 0 {
		return
	}
	d.hostCacheDeleteByID(ctx, host.ID, reason)

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
