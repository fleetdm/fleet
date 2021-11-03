package cached_mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
)

type cachedMysql struct {
	fleet.Datastore

	redisPool fleet.RedisPool
}

const (
	CacheKeyAppConfig = "cache:AppConfig"
)

func New(ds fleet.Datastore, redisPool fleet.RedisPool) fleet.Datastore {
	return &cachedMysql{
		Datastore: ds,
		redisPool: redisPool,
	}
}

func (ds *cachedMysql) storeInRedis(key string, v interface{}) error {
	conn := redis.ConfigureDoer(ds.redisPool, ds.redisPool.Get())
	defer conn.Close()

	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshaling object to cache in redis: %w", err)
	}

	if _, err := conn.Do("SET", key, b, "EX", (24 * time.Hour).Seconds()); err != nil {
		return fmt.Errorf("caching object in redis: %w", err)
	}

	return nil
}

func (ds *cachedMysql) getFromRedis(key string, v interface{}) error {
	conn := redis.ReadOnlyConn(ds.redisPool,
		redis.ConfigureDoer(ds.redisPool, ds.redisPool.Get()))
	defer conn.Close()

	data, err := redigo.Bytes(conn.Do("GET", key))
	if err != nil {
		return fmt.Errorf("getting value from cache: %w", err)
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		return fmt.Errorf("unmarshaling object from cache: %w", err)
	}

	return nil
}

func (ds *cachedMysql) NewAppConfig(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
	ac, err := ds.Datastore.NewAppConfig(ctx, info)
	if err != nil {
		return nil, err
	}

	err = ds.storeInRedis(CacheKeyAppConfig, ac)

	return ac, err
}

func (ds *cachedMysql) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	ac := &fleet.AppConfig{}
	ac.ApplyDefaults()

	err := ds.getFromRedis(CacheKeyAppConfig, ac)
	if err == nil {
		return ac, nil
	}

	ac, err = ds.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	err = ds.storeInRedis(CacheKeyAppConfig, ac)

	return ac, err
}

func (ds *cachedMysql) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	err := ds.Datastore.SaveAppConfig(ctx, info)
	if err != nil {
		return err
	}

	return ds.storeInRedis(CacheKeyAppConfig, info)
}
