package cached_mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jinzhu/copier"
	"github.com/patrickmn/go-cache"
)

const (
	appConfigKey                      = "AppConfig:%s"
	defaultAppConfigExpiration        = 1 * time.Second
	packsHostKey                      = "Packs:host:%d"
	defaultPacksExpiration            = 1 * time.Minute
	scheduledQueriesKey               = "ScheduledQueries:pack:%d"
	defaultScheduledQueriesExpiration = 1 * time.Minute
	teamAgentOptionsKey               = "TeamAgentOptions:team:%d"
	defaultTeamAgentOptionsExpiration = 1 * time.Minute
)

// cloner represents any type that can clone itself. Used by types to provide a more efficient clone method.
type cloner interface {
	Clone() (interface{}, error)
}

func clone(v interface{}) (interface{}, error) {
	if cloner, ok := v.(cloner); ok {
		return cloner.Clone()
	}

	vv := reflect.ValueOf(v)
	isPtr := false
	if vv.Kind() == reflect.Ptr {
		isPtr = true
		vv = vv.Elem()
	}

	clone := reflect.New(vv.Type())
	err := copier.Copy(clone.Interface(), v)
	if err != nil {
		return nil, err
	}

	if isPtr {
		return clone.Interface(), nil
	}

	return clone.Elem().Interface(), nil
}

// cloneCache wraps the in memory cache with one that clones items before returning them.
type cloneCache struct {
	*cache.Cache
}

func (w *cloneCache) Get(k string) (interface{}, bool) {
	v, found := w.Cache.Get(k)
	if !found {
		return v, false
	}

	clone, err := clone(v)
	if err != nil {
		// Unfortunely, we can't return an error here. Let's return a cache miss instead of panic'ing.
		return nil, false
	}
	return clone, true
}

type cachedMysql struct {
	fleet.Datastore

	c *cloneCache

	packsExp            time.Duration
	scheduledQueriesExp time.Duration
	teamAgentOptionsExp time.Duration
}

type Option func(*cachedMysql)

func WithPacksExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.packsExp = d
	}
}

func WithScheduledQueriesExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.scheduledQueriesExp = d
	}
}

func WithTeamAgentOptionsExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.teamAgentOptionsExp = d
	}
}

func New(ds fleet.Datastore, opts ...Option) fleet.Datastore {
	c := &cachedMysql{
		Datastore:           ds,
		c:                   &cloneCache{cache.New(5*time.Minute, 10*time.Minute)},
		packsExp:            defaultPacksExpiration,
		scheduledQueriesExp: defaultScheduledQueriesExpiration,
		teamAgentOptionsExp: defaultTeamAgentOptionsExpiration,
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

	return ac, nil
}

func (ds *cachedMysql) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	if x, found := ds.c.Get(appConfigKey); found {
		ac, ok := x.(*fleet.AppConfig)
		if ok {
			return ac, nil
		}
	}

	ac, err := ds.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	ds.c.Set(appConfigKey, ac, defaultAppConfigExpiration)

	return ac, nil
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
	key := fmt.Sprintf(packsHostKey, hid)
	if x, found := ds.c.Get(key); found {
		cachedPacks, ok := x.([]*fleet.Pack)
		if ok {
			return cachedPacks, nil
		}
	}

	packs, err := ds.Datastore.ListPacksForHost(ctx, hid)
	if err != nil {
		return nil, err
	}

	ds.c.Set(key, packs, ds.packsExp)

	return packs, nil
}

func (ds *cachedMysql) ListScheduledQueriesInPack(ctx context.Context, packID uint) ([]*fleet.ScheduledQuery, error) {
	key := fmt.Sprintf(scheduledQueriesKey, packID)
	if x, found := ds.c.Get(key); found {
		scheduledQueries, ok := x.([]*fleet.ScheduledQuery)
		if ok {
			return scheduledQueries, nil
		}
	}

	scheduledQueries, err := ds.Datastore.ListScheduledQueriesInPack(ctx, packID)
	if err != nil {
		return nil, err
	}

	ds.c.Set(key, scheduledQueries, ds.scheduledQueriesExp)

	return scheduledQueries, nil
}

func (ds *cachedMysql) TeamAgentOptions(ctx context.Context, teamID uint) (*json.RawMessage, error) {
	key := fmt.Sprintf(teamAgentOptionsKey, teamID)
	if x, found := ds.c.Get(key); found {
		if agentOptions, ok := x.(*json.RawMessage); ok {
			return agentOptions, nil
		}
	}

	agentOptions, err := ds.Datastore.TeamAgentOptions(ctx, teamID)
	if err != nil {
		return nil, err
	}

	ds.c.Set(key, agentOptions, ds.scheduledQueriesExp)

	return agentOptions, nil
}

func (ds *cachedMysql) SaveTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	team, err := ds.Datastore.SaveTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	key := fmt.Sprintf(teamAgentOptionsKey, team.ID)

	ds.c.Set(key, team.AgentOptions, ds.teamAgentOptionsExp)

	return team, nil
}

func (ds *cachedMysql) DeleteTeam(ctx context.Context, teamID uint) error {
	err := ds.Datastore.DeleteTeam(ctx, teamID)
	if err != nil {
		return err
	}

	key := fmt.Sprintf(teamAgentOptionsKey, teamID)

	ds.c.Delete(key)

	return nil
}
