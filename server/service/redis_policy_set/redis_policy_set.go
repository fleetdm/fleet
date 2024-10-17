// Package redis_policy_set provides a Redis implementation of service.FailingPolicySet.
package redis_policy_set

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

type redisFailingPolicySet struct {
	pool       fleet.RedisPool
	testPrefix string // for tests, the key prefix to use to avoid conflicts
}

var _ fleet.FailingPolicySet = (*redisFailingPolicySet)(nil)

// NewFailing creates a redis policy set for failing policies.
func NewFailing(pool fleet.RedisPool) *redisFailingPolicySet {
	return &redisFailingPolicySet{
		pool: pool,
	}
}

type TestNamer interface {
	Name() string
}

// NewFailingTest creates a redis policy set for failing policies to be used
// only in tests.
func NewFailingTest(t TestNamer, pool fleet.RedisPool) *redisFailingPolicySet {
	return &redisFailingPolicySet{
		pool:       pool,
		testPrefix: t.Name() + ":",
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

	ids, err := redigo.Uint64s(conn.Do("SMEMBERS", r.policySetOfSetsKey()))
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
func (r *redisFailingPolicySet) AddHost(policyID uint, host fleet.PolicySetHost) error {
	// The order of the following two operations is important.
	//
	// The ordering of operations in AddHost and RemoveSet has been chosen to avoid
	// ending up with a policySetKey with a host entry, but without its corresponding entry
	// in the set of sets.
	//
	// The commands are run in different functions with different connections in
	// case it is running in redis cluster mode - the keys may not live on the
	// same node.  There is no additional connection cost for standalone mode, as
	// the second connection will come from the pool (same for cluster mode if
	// both key happen to live on the same node).
	if err := r.addHostToPolicySet(policyID, host); err != nil {
		return err
	}
	if err := r.addPolicyToSetOfSets(policyID); err != nil {
		return err
	}
	return nil
}

// scanPolicySet uses SSCAN (instead of SMEMBERS) to fetch the hosts from a policy set with a cursor.
//
// SMEMBERS blocks the Redis instance for the duration of the call, so if the set is big
// it could impact performance overall.
func (r *redisFailingPolicySet) scanPolicySet(policyID uint) ([]string, error) {
	const hostsScanCount = 100
	var hosts []string

	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	cursor := 0
	for {
		res, err := redigo.Values(conn.Do("SSCAN", r.policySetKey(policyID), cursor, "COUNT", hostsScanCount))
		if err != nil {
			return nil, fmt.Errorf("scan keys: %w", err)
		}
		var curElems []string
		_, err = redigo.Scan(res, &cursor, &curElems)
		if err != nil {
			return nil, fmt.Errorf("convert scan results: %w", err)
		}
		hosts = append(hosts, curElems...)
		if cursor == 0 {
			break
		}
	}
	return hosts, nil
}

// ListHosts returns the list of hosts present in the policy set.
func (r *redisFailingPolicySet) ListHosts(policyID uint) ([]fleet.PolicySetHost, error) {
	hostEntries, err := r.scanPolicySet(policyID)
	if err != nil {
		return nil, err
	}
	hosts := make([]fleet.PolicySetHost, len(hostEntries))
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
func (r *redisFailingPolicySet) RemoveHosts(policyID uint, hosts []fleet.PolicySetHost) error {
	if len(hosts) == 0 {
		return nil
	}

	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	var args redigo.Args
	args = args.Add(r.policySetKey(policyID))
	for _, host := range hosts {
		args = args.Add(hostEntry(host))
	}
	_, err := conn.Do("SREM", args...)
	return err
}

// RemoveSet removes a policy set.
func (r *redisFailingPolicySet) RemoveSet(policyID uint) error {
	// The order of the following two operations is important.
	//
	// See comment in AddHost.
	if err := r.removePolicyFromSetOfSets(policyID); err != nil {
		return err
	}
	if err := r.removePolicySet(policyID); err != nil {
		return err
	}
	return nil
}

func (r *redisFailingPolicySet) addHostToPolicySet(policyID uint, host fleet.PolicySetHost) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SADD", r.policySetKey(policyID), hostEntry(host)); err != nil {
		return fmt.Errorf("add host entry to policy set: %w", err)
	}
	return nil
}

func (r *redisFailingPolicySet) removePolicySet(policyID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("DEL", r.policySetKey(policyID)); err != nil {
		return fmt.Errorf("remove policy set: %w", err)
	}
	return nil
}

func (r *redisFailingPolicySet) addPolicyToSetOfSets(policyID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SADD", r.policySetOfSetsKey(), policyID); err != nil {
		return fmt.Errorf("add policy id to set of failing sets: %w", err)
	}
	return nil
}

func (r *redisFailingPolicySet) removePolicyFromSetOfSets(policyID uint) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SREM", r.policySetOfSetsKey(), policyID); err != nil {
		return fmt.Errorf("remove policy id from set of failing sets: %w", err)
	}
	return nil
}

func (r *redisFailingPolicySet) policySetKey(policyID uint) string {
	return r.testPrefix + policySetKeyPrefix + fmt.Sprint(policyID)
}

func (r *redisFailingPolicySet) policySetOfSetsKey() string {
	return r.testPrefix + policySetsSetKey
}

func hostEntry(host fleet.PolicySetHost) string {
	return fmt.Sprint(host.ID) + "," + host.Hostname
}

func parseHostEntry(v string) (*fleet.PolicySetHost, error) {
	parts := strings.SplitN(v, ",", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid format: %s", v)
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid id: %s", v)
	}
	return &fleet.PolicySetHost{
		ID:       uint(id),
		Hostname: parts[1],
	}, nil
}
