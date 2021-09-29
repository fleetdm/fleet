package fleet

import "github.com/gomodule/redigo/redis"

// RedisPool is the common interface for redigo's Pool for standalone Redis
// and redisc's Cluster for Redis Cluster.
type RedisPool interface {
	Get() redis.Conn
	Close() error
	Stats() map[string]redis.PoolStats
}
