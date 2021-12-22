// Package redis_policy_set provides a Redis implementation of service.FailingPolicySet.
package redis_policy_set

import (
	"fmt"
	"strconv"
	"strings"

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

const (
	policySetKeyPrefix = "policies:failing:"
	// policySetsSetKey is used to avoid a SCAN command when listing policy sets.
	policySetsSetKey = "policies:failing_sets"
)

// ListSets lists all the policy sets.
func (r *redisFailingPolicySet) ListSets() ([]uint, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	ids, err := redigo.Ints(conn.Do("SMEMBERS", policySetsSetKey))
	if err != nil && err != redigo.ErrNil {
		return nil, err
	}
	policyIDs := make([]uint, len(ids))
	for i := range ids {
		policyIDs[i] = uint(ids[i])
	}
	return policyIDs, nil
}

// AddHost adds the given host to the policy sets.
func (r *redisFailingPolicySet) AddHost(policyID uint, host service.PolicySetHost) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// The order of the following two operations is important.
	//
	// The ordering of operations in AddHost and RemoveSet has been chosen to avoid
	// ending up with a policySetKey with a host entry, but without its corresponding entry
	// in the set of sets.
	if _, err := conn.Do("SADD", policySetKey(policyID), hostEntry(host)); err != nil {
		return err
	}
	if _, err := conn.Do("SADD", policySetsSetKey, policyID); err != nil {
		return err
	}
	return nil
}

// ListHosts returns the list of hosts present in the policy set.
func (r *redisFailingPolicySet) ListHosts(policyID uint) ([]service.PolicySetHost, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	hostEntries, err := redigo.Strings(conn.Do("SMEMBERS", policySetKey(policyID)))
	if err != nil && err != redigo.ErrNil {
		return nil, err
	}
	hosts := make([]service.PolicySetHost, len(hostEntries))
	for i := range hostEntries {
		policySetHost, err := parseHostEntry(hostEntries[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse host entry: %w", err)
		}
		hosts[i] = *policySetHost
	}
	return hosts, nil
}

// RemoveHosts removes the hosts from the policy set.
func (r *redisFailingPolicySet) RemoveHosts(policyID uint, hosts []service.PolicySetHost) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	var args redigo.Args
	args = args.Add(policySetKey(policyID))
	for _, host := range hosts {
		args = args.Add(hostEntry(host))
	}
	_, err := conn.Do("SREM", args...)
	return err
}

// RemoveSet removes a policy set.
func (r *redisFailingPolicySet) RemoveSet(policyID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	// The order of the following two operations is important.
	//
	// See comment in AddHost.
	if _, err := conn.Do("SREM", policySetsSetKey, policyID); err != nil {
		return err
	}
	if _, err := conn.Do("DEL", policySetKey(policyID)); err != nil {
		return err
	}
	return nil
}

func policySetKey(policyID uint) string {
	return policySetKeyPrefix + strconv.Itoa(int(policyID))
}

func hostEntry(host service.PolicySetHost) string {
	return strconv.Itoa(int(host.ID)) + "," + host.Hostname
}

func parseHostEntry(v string) (*service.PolicySetHost, error) {
	parts := strings.SplitN(v, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format: %s", v)
	}
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid id: %s", v)
	}
	return &service.PolicySetHost{
		ID:       uint(id),
		Hostname: parts[1],
	}, nil
}
