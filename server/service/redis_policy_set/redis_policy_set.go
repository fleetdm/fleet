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

const policySetKeyPrefix = "policies:failing:"

func policySetKey(policyID uint) string {
	return policySetKeyPrefix + strconv.Itoa(int(policyID))
}

func policyIDFromKey(key string) (uint, error) {
	policyID, err := strconv.Atoi(strings.TrimPrefix(key, policySetKeyPrefix))
	if err != nil {
		return 0, err
	}
	return uint(policyID), nil
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

// ListSets lists all the policy sets.
func (r *redisFailingPolicySet) ListSets() ([]uint, error) {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	var policyIDs []uint
	err := redis.EachNode(r.pool, false, func(conn redigo.Conn) error {
		var cursor int
		for {
			res, err := redigo.Values(conn.Do("SCAN", cursor, "MATCH", policySetKeyPrefix+"*"))
			if err != nil {
				return err
			}
			var curKeys []string
			if _, err = redigo.Scan(res, &cursor, &curKeys); err != nil {
				return err
			}
			for _, curKey := range curKeys {
				policyID, err := policyIDFromKey(curKey)
				if err != nil {
					return fmt.Errorf("invalid policy set key %s: %w", curKey, err)
				}
				policyIDs = append(policyIDs, policyID)
			}
			if cursor == 0 {
				return nil
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return policyIDs, nil
}

// AddFailingPoliciesForHost adds the given host to the policy sets.
func (r *redisFailingPolicySet) AddHost(policyID uint, host service.PolicySetHost) error {
	conn := redis.ConfigureDoer(r.pool, r.pool.Get())
	defer conn.Close()

	_, err := conn.Do("SADD",
		policySetKey(policyID),
		hostEntry(host),
	)
	return err
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
// If after removal, the policy has no hosts then the set is removed.
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

	_, err := conn.Do("DEL", policySetKey(policyID))
	return err
}
