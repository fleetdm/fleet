package redis_policy_set

import (
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
	hostIDs, err := r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	host2 := service.PolicySetHost{
		ID:       uint(2),
		Hostname: "host2.example",
	}
	err = r.AddHost(policyID1, host2)
	require.NoError(t, err)

	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	host3 := service.PolicySetHost{
		ID:       uint(3),
		Hostname: "host3.example",
	}
	err = r.AddHost(policyID1, host3)
	require.NoError(t, err)

	policyID2 := uint(2)
	err = r.AddHost(policyID2, host2)
	require.NoError(t, err)

	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 2)
	require.Equal(t, host2, hostIDs[0])
	require.Equal(t, host3, hostIDs[1])

	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	err = r.RemoveHosts(policyID1, []service.PolicySetHost{host2, host3})
	require.NoError(t, err)

	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host2, hostIDs[0])

	err = r.RemoveHosts(policyID2, []service.PolicySetHost{host2})
	require.NoError(t, err)

	hostIDs, err = r.ListHosts(policyID2)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	host4 := service.PolicySetHost{
		ID:       uint(4),
		Hostname: "host4.example",
	}
	err = r.AddHost(policyID1, host4)
	require.NoError(t, err)

	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Len(t, hostIDs, 1)
	require.Equal(t, host4, hostIDs[0])

	err = r.RemoveSet(policyID1)
	require.NoError(t, err)

	hostIDs, err = r.ListHosts(policyID1)
	require.NoError(t, err)
	require.Empty(t, hostIDs)

	err = r.RemoveSet(policyID1)
	require.NoError(t, err)
}
