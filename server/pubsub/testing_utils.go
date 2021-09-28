package pubsub

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
)

func SetupRedisForTest(t *testing.T, cluster bool) *redisQueryResults {
	const dupResults = false
	pool := redis.SetupRedis(t, cluster, false)
	store := NewRedisQueryResults(pool, dupResults)
	return store
}
