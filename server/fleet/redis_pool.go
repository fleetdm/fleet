package fleet

import "github.com/gomodule/redigo/redis"

type RedisPool interface {
	Get() redis.Conn
	Close() error
	Stats() map[string]redis.PoolStats
}
