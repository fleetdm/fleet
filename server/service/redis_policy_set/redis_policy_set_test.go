package redis_policy_set

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/fleet/policytest"
	"github.com/fleetdm/fleet/v4/server/test"
)

func TestRedisFailingPolicySet(t *testing.T) {
	for _, f := range []func(*testing.T, fleet.FailingPolicySet){
		policytest.RunFailingPolicySetTests,
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
