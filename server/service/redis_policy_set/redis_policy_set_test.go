package redis_policy_set

import (
	"sort"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestRedisFailingPolicySet(t *testing.T) {
	for _, f := range []func(*testing.T, *redisFailingPolicySet){
		testRedisFailingPolicySetBasic,
	} {
		t.Run(test.FunctionName(f), func(t *testing.T) {
			t.Run("standalone", func(t *testing.T) {
				store := setupRedis(t, false)
				f(t, store)
			})

			t.Run("cluster", func(t *testing.T) {
				store := setupRedis(t, true)
				f(t, store)
			})
		})
	}
}

func setupRedis(t *testing.T, cluster bool) *redisFailingPolicySet {
	pool := redistest.SetupRedis(t, cluster, true, true)
	return NewFailing(pool)
}

func testRedisFailingPolicySetBasic(t *testing.T, r *redisFailingPolicySet) {
	policyID1 := uint(1)

	// Test listing if the policy set doesn't exist.
	hostIDs, err := r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	// Test removing hosts if the set doesn't exist.
	hostx := service.PolicySetHost{
		ID:       uint(999),
		Hostname: "hostx.example",
	}
	err = r.RemoveHosts(policyID1, []service.PolicySetHost{hostx})
	require.NoError(t, err)

	// Test adding a new host to a policy set.
	host2 := service.PolicySetHost{
		ID:       uint(2),
		Hostname: "host2.example",
	}
	err = r.AddHost(policyID1, host2)
	require.NoError(t, err)

	// Test listing the policy set.
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	// Test adding a second host to the policy set.
	host3 := service.PolicySetHost{
		ID:       uint(3),
		Hostname: "host3.example",
	}
	err = r.AddHost(policyID1, host3)
	require.NoError(t, err)

	// Test adding a shared host on a different policy set.
	policyID2 := uint(2)
	err = r.AddHost(policyID2, host2)
	require.NoError(t, err)

	// Test listing of first and second policy set.
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 2)
	sort.Slice(hostIDs, func(i, j int) bool {
		return hostIDs[i].ID < hostIDs[j].ID
	})
	require.Equal(t, host2, hostIDs[0])
	require.Equal(t, host3, hostIDs[1])
	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	// Test removing all hosts from the first policy set.
	err = r.RemoveHosts(policyID1, []service.PolicySetHost{host2, host3})
	require.NoError(t, err)

	// Test listing of first and second policy set (after some removal).
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)
	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	// Test removal of a host on the second policy set.
	err = r.RemoveHosts(policyID2, []service.PolicySetHost{host2})
	require.NoError(t, err)

	// Test listing of the second policy set.
	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	// Add another host to the first policy set (after removal of the other two).
	host4 := service.PolicySetHost{
		ID:       uint(4),
		Hostname: "host4.example",
	}
	err = r.AddHost(policyID1, host4)
	require.NoError(t, err)

	// Test listing of the first policy set.
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host4, hostIDs[0])

	// Test removal of the first policy set.
	err = r.RemoveSet(policyID1)
	require.NoError(t, err)

	// Test listing of a removed policy set.
	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	// Test a second removal of the first policy set.
	err = r.RemoveSet(policyID1)
	require.NoError(t, err)

	// Test removing hosts if the set doesn't exist anymore.
	err = r.RemoveHosts(policyID1, []service.PolicySetHost{host4})
	require.NoError(t, err)

	// Test removal of an unexisting policy set.
	policyIDX := uint(999)
	err = r.RemoveSet(policyIDX)
	require.NoError(t, err)
}
