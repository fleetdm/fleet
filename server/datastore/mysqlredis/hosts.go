package mysqlredis

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// TODO: additionally, we could have a cron job to re-sync the counts every so
// often, it would:
// a) SELECT COUNT(*) the hosts in the DB and compare with SCARD in redis
// b) only if different (could even be only if "significantly" different), load
// the IDs and replace the redis set with those.
//
// For additional safety, we could check the DB if we're about to reject an
// enrollment: if so, check the COUNT(*) in the DB, and if it matches or is
// above, reject, if it doesn't match load the IDs and replace the redis SET
// with those.

func (d *datastore) NewHost(ctx context.Context, host *fleet.Host) (*fleet.Host, error) {
	// TODO: SADD host id after successful call
	return d.Datastore.NewHost(ctx, host)
}

func (d *datastore) EnrollHost(ctx context.Context, osqueryHostID, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
	// TODO: SADD host id after successful call
	return d.Datastore.EnrollHost(ctx, osqueryHostID, nodeKey, teamID, cooldown)
}

func (d *datastore) DeleteHost(ctx context.Context, hid uint) error {
	// TODO: SREM host id after successful call
	return d.Datastore.DeleteHost(ctx, hid)
}

func (d *datastore) DeleteHosts(ctx context.Context, ids []uint) error {
	// TODO: SREM host ids after successful call
	return d.Datastore.DeleteHosts(ctx, ids)
}

func (d *datastore) CleanupExpiredHosts(ctx context.Context) error {
	// TODO: change the signature to return IDs of deleted hosts.
	// TODO: SREM host ids after successful call
	return d.Datastore.CleanupExpiredHosts(ctx)
}

func (d *datastore) CleanupIncomingHosts(ctx context.Context, now time.Time) error {
	// TODO: change the signature to return IDs of deleted hosts.
	// TODO: SREM host ids after successful call
	return d.Datastore.CleanupIncomingHosts(ctx, now)
}
