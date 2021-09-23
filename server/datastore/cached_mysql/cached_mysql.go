package cached_mysql

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/datastore"
	"github.com/fleetdm/fleet/v4/server/fleet"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
)

type cachedMysql struct {
	fleet.Datastore
	datastore.Locker

	redisPool fleet.RedisPool

	hashes    map[string][32]byte
	appConfig *fleet.AppConfig
}

const CacheKeyAppConfig = "AppConfig"

func New(ds fleet.Datastore, locker datastore.Locker, redisPool fleet.RedisPool) fleet.Datastore {
	return &cachedMysql{
		Datastore: ds,
		Locker:    locker,
		redisPool: redisPool,
		hashes:    make(map[string][32]byte),
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

	ds.hashes[key] = blake2b.Sum256(b)

	return nil
}

var errAlreadyGotIt = errors.New("already have it")

func (ds *cachedMysql) getFromRedis(key string, v interface{}) error {
	conn := ds.redisPool.ConfigureDoer(ds.redisPool.Get())
	defer conn.Close()

	data, err := redigo.Bytes(conn.Do("GET", key))
	if err != nil {
		return errors.Wrap(err, "getting value from cache")
	}

	gotHash := blake2b.Sum256(data)
	if ds.hashes[key] == gotHash {
		return errAlreadyGotIt
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
	ds.appConfig = ac

	return ac, err
}
func (ds *cachedMysql) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	ac := &fleet.AppConfig{}
	ac.ApplyDefaults()

	err := ds.getFromRedis(CacheKeyAppConfig, ac)
	if err == errAlreadyGotIt && ds.appConfig != nil {
		return ds.appConfig, nil
	}
	if err == nil {
		return ac, nil
	}

	ac, err = ds.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "calling app config")
	}

	err = ds.storeInRedis(CacheKeyAppConfig, ac)
	ds.appConfig = ac

	return ac, err
}

func (ds *cachedMysql) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	err := ds.Datastore.SaveAppConfig(ctx, info)
	if err != nil {
		return errors.Wrap(err, "calling save app config")
	}

	err = ds.storeInRedis(CacheKeyAppConfig, info)
	ds.appConfig = info
	return err
}
