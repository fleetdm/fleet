package fleet

import "github.com/gomodule/redigo/redis"

// RedisPool is the common interface for redigo's Pool for standalone Redis
// and redisc's Cluster for Redis Cluster.
type RedisPool interface {
	// Get returns a redis connection. It must always be closed after use.
	Get() redis.Conn

	// Close closes the redis connection.
	Close() error

	// Stats returns a map of redis pool statistics for each server address.
	Stats() map[string]redis.PoolStats

	// ConfigureDoer returns a redis connection that is properly configured
	// to execute Do commands. This should only be called when the actions
	// to execute are all done with conn.Do.
	ConfigureDoer(redis.Conn) redis.Conn
}
