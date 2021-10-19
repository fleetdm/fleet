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

	// Mode returns the mode in which Redis is running.
	Mode() RedisMode
}

// RedisMode indicates the mode in which Redis is running.
type RedisMode byte

// List of supported Redis modes.
const (
	RedisStandalone RedisMode = iota
	RedisCluster
)

// String returns the string representation of the Redis mode.
func (m RedisMode) String() string {
	switch m {
	case RedisStandalone:
		return "standalone"
	case RedisCluster:
		return "cluster"
	default:
		return "unknown"
	}
}
