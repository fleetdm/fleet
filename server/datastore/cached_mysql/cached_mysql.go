package cached_mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/patrickmn/go-cache"
)

type cachedMysql struct {
	fleet.Datastore

	c *cache.Cache

	packsExp            time.Duration
	scheduledQueriesExp time.Duration
}

const (
	appConfigKey                      = "AppConfig"
	packsKey                          = "Packs"
	scheduledQueriesKey               = "ScheduledQueries"
	defaultAppConfigExpiration        = 1 * time.Second
	defaultPacksExpiration            = 1 * time.Minute
	defaultScheduledQueriesExpiration = 1 * time.Minute
)

type Option func(*cachedMysql)

func WithPacksExpiration(t time.Duration) Option {
	return func(o *cachedMysql) {
		o.packsExp = t
	}
}

func WithScheduledQueriesExpiration(t time.Duration) Option {
	return func(o *cachedMysql) {
		o.scheduledQueriesExp = t
	}
}

func New(ds fleet.Datastore, opts ...Option) fleet.Datastore {
	c := &cachedMysql{
		Datastore:           ds,
		c:                   cache.New(5*time.Minute, 10*time.Minute),
		packsExp:            defaultPacksExpiration,
		scheduledQueriesExp: defaultScheduledQueriesExpiration,
	}
	for _, fn := range opts {
		fn(c)
	}
	return c
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

func (ds *cachedMysql) ListPacksForHost(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
	key := fmt.Sprintf("%s_%d", packsKey, hid)
	cachedPacks, found := ds.c.Get(key)
	if found && cachedPacks != nil {
		casted, ok := cachedPacks.([]*fleet.Pack)
		if ok {
			return casted, nil
		}
	}

	packs, err := ds.Datastore.ListPacksForHost(ctx, hid)
	if err != nil {
		return nil, err
	}

	ds.c.Set(key, packs, ds.packsExp)

	return packs, nil
}

func (ds *cachedMysql) ListScheduledQueriesInPack(ctx context.Context, id uint) ([]*fleet.ScheduledQuery, error) {
	key := fmt.Sprintf("%s_%d", scheduledQueriesKey, id)
	cachedScheduledQueries, found := ds.c.Get(key)
	if found && cachedScheduledQueries != nil {
		casted, ok := cachedScheduledQueries.([]*fleet.ScheduledQuery)
		if ok {
			return casted, nil
		}
	}

	scheduledQueries, err := ds.Datastore.ListScheduledQueriesInPack(ctx, id)
	if err != nil {
		return nil, err
	}

	ds.c.Set(key, scheduledQueries, ds.scheduledQueriesExp)

	return scheduledQueries, nil
}
