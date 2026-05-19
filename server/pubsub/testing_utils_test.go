package pubsub

import (
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
)

// setupRedisForTest builds a redisQueryResults backed by a freshly-created
// redistest pool. It is used by pubsub's own tests; external callers should
// build the store directly via NewRedisQueryResults so this helper doesn't
// have to live in a regular .go file (and pull "testing" into the production
// binary).
func setupRedisForTest(t *testing.T, cluster, readReplica bool) *redisQueryResults {
	const dupResults = false
	pool := redistest.SetupRedis(t, "zz", cluster, false, readReplica)
	return NewRedisQueryResults(pool, dupResults, slog.New(slog.DiscardHandler))
}
