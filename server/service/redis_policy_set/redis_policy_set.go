package redis_policy_set

import (
	"strconv"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	redigo "github.com/gomodule/redigo/redis"
)

type redisFailingPolicySet struct {
	pool fleet.RedisPool
}

var _ service.FailingPolicySet = (*redisFailingPolicySet)(nil)

// NewFailing creates a redis policy set for failing policies.
func NewFailing(pool fleet.RedisPool) *redisFailingPolicySet {
	return &redisFailingPolicySet{
		pool: pool,
	}
}

func policySetKey(policyID uint) string {
	return "policies:failing:" + strconv.Itoa(int(policyID))
}

// AddFailingPoliciesForHost adds the given host to the policy sets.
func (r *redisFailingPolicySet) AddHost(policyID, hostID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	_, err := conn.Do("SADD", policySetKey(policyID), hostID)
	return err
}

// ListHosts returns the list of hosts present in the policy set.
func (r *redisFailingPolicySet) ListHosts(policyID uint) ([]uint, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	ids, err := redigo.Uint64s(conn.Do("SMEMBERS", policySetKey(policyID)))
	if err != nil && err != redigo.ErrNil {
		return nil, err
	}
	hostIDs := make([]uint, len(ids))
	for i := range ids {
		hostIDs[i] = uint(ids[i])
	}
	return hostIDs, nil
}

// RemoveHosts removes the hosts from the policy set.
func (r *redisFailingPolicySet) RemoveHosts(policyID uint, hostIDs []uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	var args redigo.Args
	args = args.Add(policySetKey(policyID))
	args = args.AddFlat(hostIDs)
	_, err := conn.Do("SREM", args...)
	return err
}
