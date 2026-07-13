package service

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	gocache "github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rawMessagePtr(s string) *json.RawMessage {
	raw := json.RawMessage(s)
	return &raw
}

// setupPackConfigCacheTest creates a mock datastore and service configured for
// pack config cache testing. The returned callCounter tracks the number of
// times ListScheduledQueriesForAgents is invoked (the main DB call that the
// cache is intended to avoid).
func setupPackConfigCacheTest(t *testing.T) (
	svc *Service,
	ds *mock.Store,
	callCounter *atomic.Int64,
) {
	t.Helper()

	ds = new(mock.Store)
	callCounter = &atomic.Int64{}

	// Base agent options (minimal).
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			AgentOptions: rawMessagePtr(`{"config":{"options":{"pack_delimiter":"/"}}}`),
		}, nil
	}

	ds.TeamAgentOptionsFunc = func(ctx context.Context, teamID uint) (*json.RawMessage, error) {
		return nil, nil
	}

	// No legacy packs by default.
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return []*fleet.Pack{}, nil
	}

	// Default: no scheduled queries in packs.
	ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, packID uint) (fleet.ScheduledQueryList, error) {
		return []*fleet.ScheduledQuery{}, nil
	}

	// Scheduled queries for agents -- this is the main DB call we track.
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		callCounter.Add(1)
		if teamID == nil {
			return []*fleet.Query{
				{
					Name:     "global_query",
					Query:    "SELECT 1",
					Interval: 60,
					Logging:  "snapshot",
				},
			}, nil
		}
		return []*fleet.Query{
			{
				Name:     "team_query",
				Query:    "SELECT 2",
				Interval: 30,
				Logging:  "differential",
				TeamID:   teamID,
			},
		}, nil
	}

	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{ID: id}, nil
	}

	fleetSvc, _ := newTestService(t, ds, nil, nil)
	// newTestService returns a validationMiddleware that embeds the core
	// *Service. Unwrap it so we can access packConfigCache and
	// InvalidatePackConfigCache directly.
	svc = fleetSvc.(validationMiddleware).Service.(*Service)
	return svc, ds, callCounter
}

// TestPackConfigCacheHit verifies that two consecutive GetClientConfig calls
// for the same host return the same pack config and that the second call does
// not hit the DB for scheduled queries.
func TestPackConfigCacheHit(t *testing.T) {
	svc, _, callCounter := setupPackConfigCacheTest(t)

	host := &fleet.Host{ID: 1}
	ctx := hostctx.NewContext(t.Context(), host)

	// First call -- cache miss, should hit DB.
	conf1, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	require.Contains(t, conf1, "packs")
	callsBefore := callCounter.Load()
	require.Positive(t, callsBefore, "expected at least one DB call on cache miss")

	// Second call -- cache hit, should NOT call ListScheduledQueriesForAgents again.
	conf2, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	callsAfter := callCounter.Load()
	assert.Equal(t, callsBefore, callsAfter, "expected no additional DB calls on cache hit")

	// Verify the pack config content is identical.
	assert.JSONEq(t,
		string(conf1["packs"].(json.RawMessage)),
		string(conf2["packs"].(json.RawMessage)),
	)
}

// TestPackConfigCacheInvalidationOnQueryCreate verifies that after
// InvalidatePackConfigCache is called (simulating a new query being created),
// the next GetClientConfig rebuilds the config from the DB.
func TestPackConfigCacheInvalidationOnQueryCreate(t *testing.T) {
	svc, ds, callCounter := setupPackConfigCacheTest(t)

	host := &fleet.Host{ID: 1}
	ctx := hostctx.NewContext(t.Context(), host)

	// Warm the cache.
	_, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	callsAfterWarm := callCounter.Load()

	// "Create" a new scheduled query by changing the mock and invalidating cache.
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		callCounter.Add(1)
		if teamID == nil {
			return []*fleet.Query{
				{Name: "global_query", Query: "SELECT 1", Interval: 60, Logging: "snapshot"},
				{Name: "new_query", Query: "SELECT 'new'", Interval: 120, Logging: "snapshot"},
			}, nil
		}
		return nil, nil
	}
	svc.InvalidatePackConfigCache()

	// Next call should hit DB and include the new query.
	conf, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	assert.Greater(t, callCounter.Load(), callsAfterWarm, "expected DB call after cache invalidation")

	packJSON := string(conf["packs"].(json.RawMessage))
	assert.Contains(t, packJSON, "new_query")
	assert.Contains(t, packJSON, "SELECT 'new'")
}

// TestPackConfigCacheInvalidationOnQueryModify verifies that modifying a
// query's SQL and invalidating the cache causes the config to reflect the
// updated SQL.
func TestPackConfigCacheInvalidationOnQueryModify(t *testing.T) {
	svc, ds, callCounter := setupPackConfigCacheTest(t)

	host := &fleet.Host{ID: 1}
	ctx := hostctx.NewContext(t.Context(), host)

	// Warm the cache.
	conf1, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	assert.Contains(t, string(conf1["packs"].(json.RawMessage)), "SELECT 1")

	// "Modify" the query SQL.
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		callCounter.Add(1)
		if teamID == nil {
			return []*fleet.Query{
				{Name: "global_query", Query: "SELECT 'modified'", Interval: 60, Logging: "snapshot"},
			}, nil
		}
		return nil, nil
	}
	svc.InvalidatePackConfigCache()

	conf2, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	packJSON := string(conf2["packs"].(json.RawMessage))
	assert.Contains(t, packJSON, "SELECT 'modified'")
	assert.NotContains(t, packJSON, `"SELECT 1"`)
}

// TestPackConfigCacheInvalidationOnQueryDelete verifies that deleting a query
// and invalidating the cache removes it from the config.
func TestPackConfigCacheInvalidationOnQueryDelete(t *testing.T) {
	svc, ds, callCounter := setupPackConfigCacheTest(t)

	host := &fleet.Host{ID: 1}
	ctx := hostctx.NewContext(t.Context(), host)

	// Warm the cache -- global_query should be present.
	conf1, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	assert.Contains(t, string(conf1["packs"].(json.RawMessage)), "global_query")

	// "Delete" the query -- return empty list.
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		callCounter.Add(1)
		return nil, nil
	}
	svc.InvalidatePackConfigCache()

	conf2, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	// With no queries, packs should be absent from the config.
	_, hasPacks := conf2["packs"]
	assert.False(t, hasPacks, "expected no packs after all queries deleted")
}

// TestPackConfigCacheInvalidationOnApplyQuerySpecs verifies that after a
// GitOps-style ApplyQuerySpecs (simulated by changing the mock and
// invalidating), the config reflects the new spec.
func TestPackConfigCacheInvalidationOnApplyQuerySpecs(t *testing.T) {
	svc, ds, callCounter := setupPackConfigCacheTest(t)

	host := &fleet.Host{ID: 1}
	ctx := hostctx.NewContext(t.Context(), host)

	// Warm the cache.
	conf1, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	assert.Contains(t, string(conf1["packs"].(json.RawMessage)), "global_query")

	// Simulate ApplyQuerySpecs updating queries.
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		callCounter.Add(1)
		if teamID == nil {
			return []*fleet.Query{
				{Name: "spec_query_a", Query: "SELECT 'a'", Interval: 10, Logging: "snapshot"},
				{Name: "spec_query_b", Query: "SELECT 'b'", Interval: 20, Logging: "snapshot"},
			}, nil
		}
		return nil, nil
	}
	svc.InvalidatePackConfigCache()

	conf2, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	packJSON := string(conf2["packs"].(json.RawMessage))
	assert.Contains(t, packJSON, "spec_query_a")
	assert.Contains(t, packJSON, "spec_query_b")
	assert.NotContains(t, packJSON, "global_query")
}

// TestPackConfigCacheTTLExpiration verifies that after the cache TTL expires,
// a fresh config is built from the DB.
func TestPackConfigCacheTTLExpiration(t *testing.T) {
	svc, ds, callCounter := setupPackConfigCacheTest(t)

	// Replace the cache with a very short TTL so the test doesn't wait long.
	svc.packConfigCache = gocache.New(50*time.Millisecond, 25*time.Millisecond)

	host := &fleet.Host{ID: 1}
	ctx := hostctx.NewContext(t.Context(), host)

	// Warm the cache.
	_, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	callsAfterWarm := callCounter.Load()

	// Confirm cache hit.
	_, err = svc.GetClientConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, callsAfterWarm, callCounter.Load(), "expected cache hit before TTL expiry")

	// Wait for TTL to expire.
	time.Sleep(100 * time.Millisecond)

	// Update the mock so we can detect a fresh DB read.
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		callCounter.Add(1)
		if teamID == nil {
			return []*fleet.Query{
				{Name: "refreshed_query", Query: "SELECT 'refreshed'", Interval: 60, Logging: "snapshot"},
			}, nil
		}
		return nil, nil
	}

	conf, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	assert.Greater(t, callCounter.Load(), callsAfterWarm, "expected DB call after TTL expiry")
	assert.Contains(t, string(conf["packs"].(json.RawMessage)), "refreshed_query")
}

// TestPackConfigCacheTeamIsolation verifies that hosts in different teams get
// different cached configs and that caching one team's config does not affect
// another team's config.
func TestPackConfigCacheTeamIsolation(t *testing.T) {
	svc, _, callCounter := setupPackConfigCacheTest(t)

	globalHost := &fleet.Host{ID: 1}
	team1Host := &fleet.Host{ID: 2, TeamID: new(uint(1))}
	team2Host := &fleet.Host{ID: 3, TeamID: new(uint(2))}

	ctxGlobal := hostctx.NewContext(t.Context(), globalHost)
	ctxTeam1 := hostctx.NewContext(t.Context(), team1Host)
	ctxTeam2 := hostctx.NewContext(t.Context(), team2Host)

	// Fetch config for each.
	confGlobal, err := svc.GetClientConfig(ctxGlobal)
	require.NoError(t, err)
	confTeam1, err := svc.GetClientConfig(ctxTeam1)
	require.NoError(t, err)
	confTeam2, err := svc.GetClientConfig(ctxTeam2)
	require.NoError(t, err)

	callsAfterAllFetched := callCounter.Load()

	// Global config should have "Global" pack but no team pack.
	globalPacks := string(confGlobal["packs"].(json.RawMessage))
	assert.Contains(t, globalPacks, `"Global"`)
	assert.NotContains(t, globalPacks, `"team-1"`)
	assert.NotContains(t, globalPacks, `"team-2"`)

	// Team 1 should have both "Global" and "team-1" packs.
	team1Packs := string(confTeam1["packs"].(json.RawMessage))
	assert.Contains(t, team1Packs, `"Global"`)
	assert.Contains(t, team1Packs, `"team-1"`)
	assert.NotContains(t, team1Packs, `"team-2"`)

	// Team 2 should have both "Global" and "team-2" packs.
	team2Packs := string(confTeam2["packs"].(json.RawMessage))
	assert.Contains(t, team2Packs, `"Global"`)
	assert.Contains(t, team2Packs, `"team-2"`)
	assert.NotContains(t, team2Packs, `"team-1"`)

	// Now fetch all three again -- all should be cache hits.
	_, err = svc.GetClientConfig(ctxGlobal)
	require.NoError(t, err)
	_, err = svc.GetClientConfig(ctxTeam1)
	require.NoError(t, err)
	_, err = svc.GetClientConfig(ctxTeam2)
	require.NoError(t, err)

	assert.Equal(t, callsAfterAllFetched, callCounter.Load(),
		"expected no additional DB calls -- all three team configs should be cached independently")
}

// TestPackConfigCacheLegacyPacksBypass verifies that when a host has legacy
// packs assigned, the cache is bypassed entirely (every call hits the DB).
func TestPackConfigCacheLegacyPacksBypass(t *testing.T) {
	svc, ds, callCounter := setupPackConfigCacheTest(t)

	// Assign a legacy pack to host 1.
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		if hid == 1 {
			return []*fleet.Pack{{ID: 10, Name: "legacy_pack"}}, nil
		}
		return []*fleet.Pack{}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, packID uint) (fleet.ScheduledQueryList, error) {
		if packID == 10 {
			return []*fleet.ScheduledQuery{
				{Name: "legacy_q", Query: "SELECT 'legacy'", Interval: 30},
			}, nil
		}
		return []*fleet.ScheduledQuery{}, nil
	}

	legacyHost := &fleet.Host{ID: 1}
	ctx := hostctx.NewContext(t.Context(), legacyHost)

	// First call.
	conf1, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	callsAfterFirst := callCounter.Load()
	assert.Contains(t, string(conf1["packs"].(json.RawMessage)), "legacy_pack")

	// Second call -- should still hit DB because legacy packs bypass cache.
	_, err = svc.GetClientConfig(ctx)
	require.NoError(t, err)
	assert.Greater(t, callCounter.Load(), callsAfterFirst,
		"expected DB call even on second request when legacy packs are present")
}

// TestPackConfigCachePerformance measures the speedup from cache hits by
// comparing 1000 GetClientConfig calls against the warm cache versus the
// number of DB calls that would have occurred without it.
func TestPackConfigCachePerformance(t *testing.T) {
	svc, _, callCounter := setupPackConfigCacheTest(t)

	host := &fleet.Host{ID: 1}
	ctx := hostctx.NewContext(t.Context(), host)

	const iterations = 1000

	// Warm the cache with the first call.
	_, err := svc.GetClientConfig(ctx)
	require.NoError(t, err)
	callsAfterWarm := callCounter.Load()

	start := time.Now()
	for range iterations {
		_, err := svc.GetClientConfig(ctx)
		require.NoError(t, err)
	}
	cachedDuration := time.Since(start)

	dbCallsDuringCached := callCounter.Load() - callsAfterWarm
	assert.Equal(t, int64(0), dbCallsDuringCached,
		"expected zero DB calls for ListScheduledQueriesForAgents during cached iterations")

	// Measure uncached: invalidate cache on every call.
	svc.InvalidatePackConfigCache()
	callsBefore := callCounter.Load()

	start = time.Now()
	for range iterations {
		svc.InvalidatePackConfigCache()
		_, err := svc.GetClientConfig(ctx)
		require.NoError(t, err)
	}
	uncachedDuration := time.Since(start)

	dbCallsDuringUncached := callCounter.Load() - callsBefore
	assert.Equal(t, int64(iterations), dbCallsDuringUncached,
		"expected exactly %d DB calls when cache is invalidated each time", iterations)

	t.Logf("Performance: cached=%v, uncached=%v, speedup=%.1fx",
		cachedDuration, uncachedDuration,
		float64(uncachedDuration)/float64(cachedDuration),
	)
}
