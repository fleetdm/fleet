package mysqlredis

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

const enrolledHostsSetKey = "enrolled_hosts:host_ids"

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

func (d *datastore) addHosts(ctx context.Context, hostIDs ...uint) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	args := redigo.Args{enrolledHostsSetKey}
	args = args.AddFlat(hostIDs)
	_, err := conn.Do("SADD", args...)
	return ctxerr.Wrap(ctx, err, "enrolled limits: add hosts")
}

func (d *datastore) removeHosts(ctx context.Context, hostIDs ...uint) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	args := redigo.Args{enrolledHostsSetKey}
	args = args.AddFlat(hostIDs)
	_, err := conn.Do("SREM", args...)
	return ctxerr.Wrap(ctx, err, "enrolled limits: remove hosts")
}

func (d *datastore) NewHost(ctx context.Context, host *fleet.Host) (*fleet.Host, error) {
	if d.enforceHostLimit > 0 {
		// TODO: check if limit is exceeded
	}

	h, err := d.Datastore.NewHost(ctx, host)
	if err == nil && d.enforceHostLimit > 0 {
		if err := d.addHosts(ctx, h.ID); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return h, err
}

func (d *datastore) EnrollHost(ctx context.Context, osqueryHostID, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
	if d.enforceHostLimit > 0 {
		// TODO: check if limit is exceeded
	}

	h, err := d.Datastore.EnrollHost(ctx, osqueryHostID, nodeKey, teamID, cooldown)
	if err == nil && d.enforceHostLimit > 0 {
		if err := d.addHosts(ctx, h.ID); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return h, err
}

func (d *datastore) DeleteHost(ctx context.Context, hid uint) error {
	err := d.Datastore.DeleteHost(ctx, hid)
	if err == nil && d.enforceHostLimit > 0 {
		if err := d.removeHosts(ctx, hid); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return err
}

func (d *datastore) DeleteHosts(ctx context.Context, ids []uint) error {
	err := d.Datastore.DeleteHosts(ctx, ids)
	if err == nil && d.enforceHostLimit > 0 {
		if err := d.removeHosts(ctx, ids...); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return err
}

func (d *datastore) CleanupExpiredHosts(ctx context.Context) error {
	if d.enforceHostLimit > 0 {
	}
	// TODO: change the signature to return IDs of deleted hosts.
	// TODO: SREM host ids after successful call
	return d.Datastore.CleanupExpiredHosts(ctx)
}

func (d *datastore) CleanupIncomingHosts(ctx context.Context, now time.Time) error {
	if d.enforceHostLimit > 0 {
	}
	// TODO: change the signature to return IDs of deleted hosts.
	// TODO: SREM host ids after successful call
	return d.Datastore.CleanupIncomingHosts(ctx, now)
}
