package cached_mysql

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/patrickmn/go-cache"
)

type cachedMysql struct {
	fleet.Datastore

	c *cache.Cache
}

const (
	appConfigKey               = "AppConfig"
	defaultAppConfigExpiration = 1 * time.Second
)

func New(ds fleet.Datastore) fleet.Datastore {
	return &cachedMysql{
		Datastore: ds,
		c:         cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (ds *cachedMysql) NewAppConfig(ctx context.Context, info *fleet.AppConfig) (*fleet.AppConfig, error) {
	ac, err := ds.Datastore.NewAppConfig(ctx, info)
	if err != nil {
		return nil, err
	}

	ds.c.Set(appConfigKey, ac, defaultAppConfigExpiration)

	return ac.Clone()
}

func (ds *cachedMysql) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	cachedAc, found := ds.c.Get(appConfigKey)
	if found {
		return cachedAc.(*fleet.AppConfig).Clone()
	}

	ac, err := ds.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	ds.c.Set(appConfigKey, ac, defaultAppConfigExpiration)

	return ac.Clone()
}

func (ds *cachedMysql) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	err := ds.Datastore.SaveAppConfig(ctx, info)
	if err != nil {
		return err
	}

	ds.c.Set(appConfigKey, info, defaultAppConfigExpiration)

	return nil
}
