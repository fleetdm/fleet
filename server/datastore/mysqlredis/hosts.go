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

var redisSetMembersBatchSize = 10000 // var so it can be changed in tests

// SyncEnrolledHostIDs forces synchronisation of host IDs between the DB and
// the Redis set. To optimize for the common case, it first checks if the
// counts are the same in the database and the redis set, and if so it does
// nothing else. Otherwise, it loads the current list of IDs from the database,
// clears the Redis set, and stores the IDs in the Redis set. This is called
// regularly (via a cron job) so that if the Redis set gets out of sync, it
// eventually fixes itself automatically.
func SyncEnrolledHostIDs(ctx context.Context, ds fleet.Datastore, pool fleet.RedisPool) error {
	dbCount, err := ds.CountEnrolledHosts(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "count enrolled hosts from the database")
	}

	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()

	redisCount, err := redigo.Int(conn.Do("SCARD", enrolledHostsSetKey))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "count enrolled hosts from redis")
	}

	if redisCount == dbCount {
		return nil
	}

	// counts differ, replace the redis set with ids from the database
	ids, err := ds.EnrolledHostIDs(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get enrolled host IDs from the database")
	}

	if _, err := conn.Do("DEL", enrolledHostsSetKey); err != nil {
		return ctxerr.Wrap(ctx, err, "clear redis enrolled hosts set")
	}

	// return the connection to the pool so it can be reused in addHosts
	conn.Close()

	if err := addHosts(ctx, pool, ids...); err != nil {
		return ctxerr.Wrap(ctx, err, "add database host IDs to the redis set")
	}
	return nil
}

func addHosts(ctx context.Context, pool fleet.RedisPool, hostIDs ...uint) error {
	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()

	for len(hostIDs) > 0 {
		max := len(hostIDs)
		if max > redisSetMembersBatchSize {
			max = redisSetMembersBatchSize
		}

		args := redigo.Args{enrolledHostsSetKey}
		args = args.AddFlat(hostIDs[:max])
		if _, err := conn.Do("SADD", args...); err != nil {
			return ctxerr.Wrap(ctx, err, "enrolled limits: add hosts")
		}
		hostIDs = hostIDs[max:]
	}
	return nil
}

func removeHosts(ctx context.Context, pool fleet.RedisPool, hostIDs ...uint) error {
	conn := redis.ConfigureDoer(pool, pool.Get())
	defer conn.Close()

	for len(hostIDs) > 0 {
		max := len(hostIDs)
		if max > redisSetMembersBatchSize {
			max = redisSetMembersBatchSize
		}

		args := redigo.Args{enrolledHostsSetKey}
		args = args.AddFlat(hostIDs)
		if _, err := conn.Do("SREM", args...); err != nil {
			return ctxerr.Wrap(ctx, err, "enrolled limits: remove hosts")
		}
		hostIDs = hostIDs[max:]
	}
	return nil
}

func (d *datastore) checkCanAddHost(ctx context.Context) (bool, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	n, err := redigo.Int(conn.Do("SCARD", enrolledHostsSetKey))
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "enrolled limits: check can add host")
	}
	if n >= d.enforceHostLimit {
		return false, nil
	}
	return true, nil
}

func (d *datastore) NewHost(ctx context.Context, host *fleet.Host) (*fleet.Host, error) {
	if d.enforceHostLimit > 0 {
		ok, err := d.checkCanAddHost(ctx)
		if !ok && err == nil {
			return nil, ctxerr.Errorf(ctx, "maximum number of hosts reached: %d", d.enforceHostLimit)
		}
		if err != nil {
			logging.WithErr(ctx, err)
		}
	}

	h, err := d.Datastore.NewHost(ctx, host)
	if err == nil && d.enforceHostLimit > 0 {
		if err := addHosts(ctx, d.pool, h.ID); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return h, err
}

func (d *datastore) EnrollHost(ctx context.Context, osqueryHostID, nodeKey string, teamID *uint, cooldown time.Duration) (*fleet.Host, error) {
	if d.enforceHostLimit > 0 {
		ok, err := d.checkCanAddHost(ctx)
		if !ok && err == nil {
			return nil, ctxerr.Errorf(ctx, "maximum number of hosts reached: %d", d.enforceHostLimit)
		}
		if err != nil {
			logging.WithErr(ctx, err)
		}
	}

	h, err := d.Datastore.EnrollHost(ctx, osqueryHostID, nodeKey, teamID, cooldown)
	if err == nil && d.enforceHostLimit > 0 {
		if err := addHosts(ctx, d.pool, h.ID); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return h, err
}

func (d *datastore) DeleteHost(ctx context.Context, hid uint) error {
	err := d.Datastore.DeleteHost(ctx, hid)
	if err == nil && d.enforceHostLimit > 0 {
		if err := removeHosts(ctx, d.pool, hid); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return err
}

func (d *datastore) DeleteHosts(ctx context.Context, ids []uint) error {
	err := d.Datastore.DeleteHosts(ctx, ids)
	if err == nil && d.enforceHostLimit > 0 {
		if err := removeHosts(ctx, d.pool, ids...); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return err
}

func (d *datastore) CleanupExpiredHosts(ctx context.Context) ([]uint, error) {
	ids, err := d.Datastore.CleanupExpiredHosts(ctx)
	if err == nil && d.enforceHostLimit > 0 {
		if err := removeHosts(ctx, d.pool, ids...); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return ids, err
}

func (d *datastore) CleanupIncomingHosts(ctx context.Context, now time.Time) ([]uint, error) {
	ids, err := d.Datastore.CleanupIncomingHosts(ctx, now)
	if err == nil && d.enforceHostLimit > 0 {
		if err := removeHosts(ctx, d.pool, ids...); err != nil {
			logging.WithErr(ctx, err)
		}
	}
	return ids, err
}
