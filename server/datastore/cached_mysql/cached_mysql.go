package cached_mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/patrickmn/go-cache"
)

// NOTE: To add a new cached item, make sure you know how/when to invalidate it
// and how long it can safely be cached. Consider the case where it is read to
// be updated - those cases need to bypass the cache and read directly from the
// DB to always use fresh data (see the ctxdb.BypassCachedMysql method). Follow
// all of these steps:
//
//  1. Add a unique key name and a default expiration duration, which will be
//     used in production.
//  2. Define an expiration duration field in the cachedMysql struct,
//     initialize it with the default expiration duration in New, and add a
//     WithXXXExpiration option to customize it.
//  3. Implement the cloner interface for the type of the cached item. If the
//     type is a slice, you will need to define a type for the slice (see
//     fleet.ScheduledQueryList for an example, or packsList in this package
//     for an alternative approach).
//  4. Add the cached item to fleet/tools/cloner-check/main.go (in the
//     cacheableItems slice variable) to ensure it gets properly checked in CI
//     when fields are added/modified. Run the tool to update the generated files
//     once you're confident that the Clone implementation covers all fields that
//     need special care (usually pointers, slices, maps).
//  5. Add the required Datastore methods to get the cached item and to set it,
//     and add tests in cached_mysql_test.go to ensure it works as expected.
const (
	appConfigKey                       = "AppConfig:%s"
	defaultAppConfigExpiration         = 1 * time.Second
	packsHostKey                       = "Packs:host:%d"
	defaultPacksExpiration             = 1 * time.Minute
	scheduledQueriesKey                = "ScheduledQueries:pack:%d"
	defaultScheduledQueriesExpiration  = 1 * time.Minute
	teamAgentOptionsKey                = "TeamAgentOptions:team:%d"
	defaultTeamAgentOptionsExpiration  = 1 * time.Minute
	teamFeaturesKey                    = "TeamFeatures:team:%d"
	defaultTeamFeaturesExpiration      = 1 * time.Minute
	teamMDMConfigKey                   = "TeamMDMConfig:team:%d"
	defaultTeamMDMConfigExpiration     = 1 * time.Minute
	queryByNameKey                     = "QueryByName:team:%d:%s"
	defaultQueryByNameExpiration       = 1 * time.Second
	queryResultsCountKey               = "QueryResultsCount:%d"
	defaultQueryResultsCountExpiration = 1 * time.Second
	// NOTE: MDM assets are cached using their checksum as well, as it's
	// important for them to always be fresh if they changed (see cachedi
	// mplementation below for details)
	mdmConfigAssetKey = "MDMConfigAsset:%s:%s"
	// NOTE: given how mdmConfigAssetKey works, it means that once an asset
	// changes, it'll linger for this amount of time. The curent
	// implementation assumes infrequent asset changes.
	defaultMDMConfigAssetExpiration = 15 * time.Minute
)

// cloneCache wraps the in memory cache with one that clones items before returning them.
type cloneCache struct {
	*cache.Cache
}

func (c *cloneCache) Get(ctx context.Context, k string) (fleet.Cloner, bool) {
	if ctxdb.IsCachedMysqlBypassed(ctx) {
		// cache miss if the caller explicitly asked to bypass the cache
		return nil, false
	}

	x, found := c.Cache.Get(k)
	if !found {
		return nil, false
	}
	xc, ok := x.(fleet.Cloner)
	if !ok {
		// should never happen, cached item is not a cloner
		return nil, false
	}

	clone, err := xc.Clone()
	if err != nil {
		// Unfortunely, we can't return an error here. Return a cache miss instead of panic'ing.
		return nil, false
	}
	return clone, true
}

func (c *cloneCache) Set(ctx context.Context, k string, x fleet.Cloner, d time.Duration) {
	clone, err := x.Clone()
	if err != nil {
		// Unfortunately, we can't return an error here. Skip caching it if clone
		// fails, but ensure that we clear any existing cached item for this key,
		// as the call to Set indicates the cache is now stale.
		c.Cache.Delete(k)
		return
	}

	c.Cache.Set(k, clone, d)
}

type cachedMysql struct {
	fleet.Datastore

	c *cloneCache

	appConfigExp         time.Duration
	packsExp             time.Duration
	scheduledQueriesExp  time.Duration
	teamAgentOptionsExp  time.Duration
	teamFeaturesExp      time.Duration
	teamMDMConfigExp     time.Duration
	queryByNameExp       time.Duration
	queryResultsCountExp time.Duration
	mdmConfigAssetExp    time.Duration
}

type Option func(*cachedMysql)

func WithAppConfigExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.appConfigExp = d
	}
}

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

func WithTeamFeaturesExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.teamFeaturesExp = d
	}
}

func WithTeamMDMConfigExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.teamMDMConfigExp = d
	}
}

func WithQueryByNameExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.queryByNameExp = d
	}
}

func WithQueryResultsCountExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.queryResultsCountExp = d
	}
}

func WithMDMConfigAssetExpiration(d time.Duration) Option {
	return func(o *cachedMysql) {
		o.mdmConfigAssetExp = d
	}
}

func New(ds fleet.Datastore, opts ...Option) fleet.Datastore {
	c := &cachedMysql{
		Datastore:            ds,
		c:                    &cloneCache{cache.New(5*time.Minute, 10*time.Minute)},
		appConfigExp:         defaultAppConfigExpiration,
		packsExp:             defaultPacksExpiration,
		scheduledQueriesExp:  defaultScheduledQueriesExpiration,
		teamAgentOptionsExp:  defaultTeamAgentOptionsExpiration,
		teamFeaturesExp:      defaultTeamFeaturesExpiration,
		teamMDMConfigExp:     defaultTeamMDMConfigExpiration,
		queryByNameExp:       defaultQueryByNameExpiration,
		queryResultsCountExp: defaultQueryResultsCountExpiration,
		mdmConfigAssetExp:    defaultMDMConfigAssetExpiration,
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

	ds.c.Set(ctx, appConfigKey, ac, ds.appConfigExp)

	return ac, nil
}

func (ds *cachedMysql) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	if x, found := ds.c.Get(ctx, appConfigKey); found {
		ac, ok := x.(*fleet.AppConfig)
		if ok {
			return ac, nil
		}
	}

	ac, err := ds.Datastore.AppConfig(ctx)
	if err != nil {
		return nil, err
	}

	ds.c.Set(ctx, appConfigKey, ac, ds.appConfigExp)

	return ac, nil
}

func (ds *cachedMysql) SaveAppConfig(ctx context.Context, info *fleet.AppConfig) error {
	err := ds.Datastore.SaveAppConfig(ctx, info)
	if err != nil {
		return err
	}

	ds.c.Set(ctx, appConfigKey, info, ds.appConfigExp)

	return nil
}

func (ds *cachedMysql) ListPacksForHost(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
	key := fmt.Sprintf(packsHostKey, hid)
	if x, found := ds.c.Get(ctx, key); found {
		cachedPacks, ok := x.(packsList)
		if ok {
			return cachedPacks, nil
		}
	}

	packs, err := ds.Datastore.ListPacksForHost(ctx, hid)
	if err != nil {
		return nil, err
	}

	ds.c.Set(ctx, key, packsList(packs), ds.packsExp)

	return packs, nil
}

func (ds *cachedMysql) ListScheduledQueriesInPack(ctx context.Context, packID uint) (fleet.ScheduledQueryList, error) {
	key := fmt.Sprintf(scheduledQueriesKey, packID)
	if x, found := ds.c.Get(ctx, key); found {
		scheduledQueries, ok := x.(fleet.ScheduledQueryList)
		if ok {
			return scheduledQueries, nil
		}
	}

	scheduledQueries, err := ds.Datastore.ListScheduledQueriesInPack(ctx, packID)
	if err != nil {
		return nil, err
	}

	ds.c.Set(ctx, key, scheduledQueries, ds.scheduledQueriesExp)

	return scheduledQueries, nil
}

func (ds *cachedMysql) TeamAgentOptions(ctx context.Context, teamID uint) (*json.RawMessage, error) {
	key := fmt.Sprintf(teamAgentOptionsKey, teamID)
	if x, found := ds.c.Get(ctx, key); found {
		if agentOptions, ok := x.(*rawJSONMessage); ok {
			return (*json.RawMessage)(agentOptions), nil
		}
	}

	agentOptions, err := ds.Datastore.TeamAgentOptions(ctx, teamID)
	if err != nil {
		return nil, err
	}

	ds.c.Set(ctx, key, (*rawJSONMessage)(agentOptions), ds.teamAgentOptionsExp)

	return agentOptions, nil
}

func (ds *cachedMysql) TeamFeatures(ctx context.Context, teamID uint) (*fleet.Features, error) {
	key := fmt.Sprintf(teamFeaturesKey, teamID)
	if x, found := ds.c.Get(ctx, key); found {
		if features, ok := x.(*fleet.Features); ok {
			return features, nil
		}
	}

	features, err := ds.Datastore.TeamFeatures(ctx, teamID)
	if err != nil {
		return nil, err
	}

	ds.c.Set(ctx, key, features, ds.teamFeaturesExp)

	return features, nil
}

func (ds *cachedMysql) TeamMDMConfig(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
	key := fmt.Sprintf(teamMDMConfigKey, teamID)
	if x, found := ds.c.Get(ctx, key); found {
		if cfg, ok := x.(*fleet.TeamMDM); ok {
			return cfg, nil
		}
	}

	cfg, err := ds.Datastore.TeamMDMConfig(ctx, teamID)
	if err != nil {
		return nil, err
	}

	ds.c.Set(ctx, key, cfg, ds.teamMDMConfigExp)

	return cfg, nil
}

func (ds *cachedMysql) SaveTeam(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
	team, err := ds.Datastore.SaveTeam(ctx, team)
	if err != nil {
		return nil, err
	}

	agentOptionsKey := fmt.Sprintf(teamAgentOptionsKey, team.ID)
	featuresKey := fmt.Sprintf(teamFeaturesKey, team.ID)
	mdmConfigKey := fmt.Sprintf(teamMDMConfigKey, team.ID)

	ds.c.Set(ctx, agentOptionsKey, (*rawJSONMessage)(team.Config.AgentOptions), ds.teamAgentOptionsExp)
	ds.c.Set(ctx, featuresKey, &team.Config.Features, ds.teamFeaturesExp)
	ds.c.Set(ctx, mdmConfigKey, &team.Config.MDM, ds.teamMDMConfigExp)

	return team, nil
}

func (ds *cachedMysql) DeleteTeam(ctx context.Context, teamID uint) error {
	err := ds.Datastore.DeleteTeam(ctx, teamID)
	if err != nil {
		return err
	}

	agentOptionsKey := fmt.Sprintf(teamAgentOptionsKey, teamID)
	featuresKey := fmt.Sprintf(teamFeaturesKey, teamID)
	mdmConfigKey := fmt.Sprintf(teamMDMConfigKey, teamID)

	ds.c.Delete(agentOptionsKey)
	ds.c.Delete(featuresKey)
	ds.c.Delete(mdmConfigKey)

	return nil
}

// TODO: should we handle DeleteQuery/DeleteQueries/SaveQuery to invalidate that cache?
func (ds *cachedMysql) QueryByName(ctx context.Context, teamID *uint, name string) (*fleet.Query, error) {
	teamID_ := uint(0) // global team is 0
	if teamID != nil {
		teamID_ = *teamID
	}
	key := fmt.Sprintf(queryByNameKey, teamID_, name)

	if x, found := ds.c.Get(ctx, key); found {
		if query, ok := x.(*fleet.Query); ok {
			return query, nil
		}
	}

	query, err := ds.Datastore.QueryByName(ctx, teamID, name)
	if err != nil {
		return nil, err
	}

	ds.c.Set(ctx, key, query, ds.queryByNameExp)

	return query, nil
}

func (ds *cachedMysql) ResultCountForQuery(ctx context.Context, queryID uint) (int, error) {
	key := fmt.Sprintf(queryResultsCountKey, queryID)

	if x, found := ds.c.Get(ctx, key); found {
		if count, ok := x.(integer); ok {
			return int(count), nil
		}
	}

	count, err := ds.Datastore.ResultCountForQuery(ctx, queryID)
	if err != nil {
		return 0, err
	}

	ds.c.Set(ctx, key, integer(count), ds.queryResultsCountExp)

	return count, nil
}

func (ds *cachedMysql) GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []fleet.MDMAssetName,
	queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
	// always reach the database to get the latest hashes
	latestHashes, err := ds.Datastore.GetAllMDMConfigAssetsHashes(ctx, assetNames)
	if err != nil {
		return nil, err
	}

	cachedAssets := make(map[fleet.MDMAssetName]fleet.MDMConfigAsset)
	var missingAssets []fleet.MDMAssetName

	for _, name := range assetNames {
		key := fmt.Sprintf(mdmConfigAssetKey, name, latestHashes[name])

		if x, found := ds.c.Get(ctx, key); found {
			asset, ok := x.(fleet.MDMConfigAsset)
			if ok {
				cachedAssets[name] = asset
				continue
			}
		}

		missingAssets = append(missingAssets, name)
	}

	if len(missingAssets) == 0 {
		return cachedAssets, nil
	}

	// fetch missing assets from the database
	assetMap, err := ds.Datastore.GetAllMDMConfigAssetsByName(ctx, missingAssets, queryerContext)
	if err != nil {
		return nil, err
	}

	// update the cache with the fetched assets and their hashes
	for name, asset := range assetMap {
		key := fmt.Sprintf(mdmConfigAssetKey, name, latestHashes[name])
		ds.c.Set(ctx, key, asset, ds.mdmConfigAssetExp)
		cachedAssets[name] = asset
	}

	return cachedAssets, nil
}
