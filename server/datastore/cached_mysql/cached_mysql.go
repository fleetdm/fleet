package cached_mysql

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/datastore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
)

type cachedMysql struct {
	fleet.Datastore
	datastore.Locker

	redisPool fleet.RedisPool
}

const (
	CacheKeyAppConfig              = "AppConfig"
	CacheKeyAuthenticateHostPrefix = "AuthenticateHost"
)

func New(ds fleet.Datastore, locker datastore.Locker, redisPool fleet.RedisPool) fleet.Datastore {
	return &cachedMysql{
		Datastore: ds,
		Locker:    locker,
		redisPool: redisPool,
	}
}

func (ds *cachedMysql) storeInRedis(key string, v interface{}) error {
	conn := ds.redisPool.ConfigureDoer(ds.redisPool.Get())
	defer conn.Close()

	b, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "marshaling object to cache in redis")
	}

	if _, err := conn.Do("SET", key, b); err != nil {
		return errors.Wrap(err, "caching object in redis")
	}

	return nil
}

func (ds *cachedMysql) getFromRedis(key string, v interface{}) error {
	conn := ds.redisPool.ConfigureDoer(ds.redisPool.Get())
	defer conn.Close()

	data, err := redigo.Bytes(conn.Do("GET", key))
	if err != nil {
		return errors.Wrap(err, "getting value from cache")
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		return errors.Wrap(err, "unmarshaling object from cache")
	}

	return nil
}

func (ds *cachedMysql) NewAppConfig(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
	ac, err := ds.Datastore.NewAppConfig(ctx, info)
	if err != nil {
		return nil, errors.Wrap(err, "calling new app config")
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
		return nil, errors.Wrap(err, "calling app config")
	}

	err = ds.storeInRedis(CacheKeyAppConfig, ac)

	return ac, err
}

func (ds *cachedMysql) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	err := ds.Datastore.SaveAppConfig(ctx, info)
	if err != nil {
		return errors.Wrap(err, "calling save app config")
	}

	return ds.storeInRedis(CacheKeyAppConfig, info)
}

func (ds *cachedMysql) AuthenticateHost(ctx context.Context, nodeKey string) (*fleet.Host, error) {
	ac, err := ds.AppConfig(ctx)
	if err != nil || !ac.CacheHosts {
		return ds.Datastore.AuthenticateHost(ctx, nodeKey)
	}

	host := &fleet.Host{}
	err = ds.getFromRedis(CacheKeyAuthenticateHostPrefix+nodeKey, host)
	if err == nil {
		return host, nil
	}

	host, err = ds.Datastore.AuthenticateHost(ctx, nodeKey)
	if err != nil {
		return nil, err
	}

	err = ds.storeInRedis(CacheKeyAuthenticateHostPrefix+nodeKey, host)

	return host, err
}
