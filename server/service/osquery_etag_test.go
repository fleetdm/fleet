package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/redis_config_etag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubConfigETagStore is an in-memory fleet.ConfigETagStore for exercising
// the GetClientConfigWithETag short circuit without Redis.
type stubConfigETagStore struct {
	// shared records
	etag   string
	valid  bool
	getErr error
	setErr error

	// per-host records
	hostETag   string
	hostValid  bool
	hostGetErr error
	hostSetErr error

	// gate state
	legacyPresent bool
	legacyErr     error
	scopes        fleet.ConfigETagLabelScopes
	scopesErr     error
	callLoaders   bool // when true, gate methods defer to the load callbacks

	// recorded interactions
	getCalls        int
	hostGetCalls    int
	setCalls        int
	hostSetCalls    int
	lastSetScope    string
	lastSetPlatform string
	lastSetETag     string
	lastHostSetID   uint
	lastHostSetScope,
	lastHostSetPlatform,
	lastHostSetETag string
	hostInvalidations []uint
}

var _ fleet.ConfigETagStore = (*stubConfigETagStore)(nil)

func (s *stubConfigETagStore) GetETagIfCurrent(ctx context.Context, scope, platform string) (string, bool, error) {
	s.getCalls++
	if s.getErr != nil {
		return "", false, s.getErr
	}
	return s.etag, s.valid, nil
}

func (s *stubConfigETagStore) SetIfNoFence(ctx context.Context, scope, platform, etag string) (bool, error) {
	s.setCalls++
	s.lastSetScope, s.lastSetPlatform, s.lastSetETag = scope, platform, etag
	if s.setErr != nil {
		return false, s.setErr
	}
	return true, nil
}

func (s *stubConfigETagStore) Invalidate(ctx context.Context) error { return nil }

func (s *stubConfigETagStore) GetHostETagIfCurrent(ctx context.Context, hostID uint, scope, platform string) (string, bool, error) {
	s.hostGetCalls++
	if s.hostGetErr != nil {
		return "", false, s.hostGetErr
	}
	return s.hostETag, s.hostValid, nil
}

func (s *stubConfigETagStore) SetHostIfNoFence(ctx context.Context, hostID uint, scope, platform, etag string) (bool, error) {
	s.hostSetCalls++
	s.lastHostSetID, s.lastHostSetScope, s.lastHostSetPlatform, s.lastHostSetETag = hostID, scope, platform, etag
	if s.hostSetErr != nil {
		return false, s.hostSetErr
	}
	return true, nil
}

func (s *stubConfigETagStore) InvalidateHost(ctx context.Context, hostID uint) error {
	s.hostInvalidations = append(s.hostInvalidations, hostID)
	return nil
}

func (s *stubConfigETagStore) LegacyPacksPresent(ctx context.Context, load func(ctx context.Context) (bool, error)) (bool, error) {
	if s.legacyErr != nil {
		return true, s.legacyErr
	}
	if s.callLoaders {
		return load(ctx)
	}
	return s.legacyPresent, nil
}

func (s *stubConfigETagStore) ResetLegacyPacksFlag(ctx context.Context) error { return nil }

func (s *stubConfigETagStore) LabelScopes(ctx context.Context, load func(ctx context.Context) (fleet.ConfigETagLabelScopes, error)) (fleet.ConfigETagLabelScopes, error) {
	if s.scopesErr != nil {
		return fleet.ConfigETagLabelScopes{}, s.scopesErr
	}
	if s.callLoaders {
		return load(ctx)
	}
	return s.scopes, nil
}

func (s *stubConfigETagStore) ResetLabelScopes(ctx context.Context) error { return nil }

// newETagTestDS returns a mock datastore sufficient for a minimal full
// config build (no agent options, no packs, no schedules) plus the gate
// loaders.
func newETagTestDS() *mock.Store {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return nil, nil
	}
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		return nil, nil
	}
	ds.TeamAgentOptionsFunc = func(ctx context.Context, teamID uint) (*json.RawMessage, error) {
		return nil, nil
	}
	ds.ListPacksFunc = func(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
		return nil, nil
	}
	ds.LabelScopedScheduledQueryScopesFunc = func(ctx context.Context) (fleet.ConfigETagLabelScopes, error) {
		return fleet.ConfigETagLabelScopes{}, nil
	}
	return ds
}

// TestConfigETagSharedShortCircuitHit is THE shared-mode short circuit test:
// a valid stored ETag matching the client validator must produce a not-modified result with
// ZERO datastore reads and no ETag write.
func TestConfigETagSharedShortCircuitHit(t *testing.T) {
	ds := newETagTestDS()
	svc, ctx := newTestService(t, ds, nil, nil)
	store := &stubConfigETagStore{etag: `"stored"`, valid: true}
	svc.SetConfigETagStore(store)

	ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
	result, err := svc.GetClientConfigWithETag(ctx, new(`"stored"`))
	require.NoError(t, err)

	assert.True(t, result.NotModified)
	assert.Nil(t, result.Body, "a not-modified result must carry no body")
	assert.Equal(t, `"stored"`, result.ETag)
	assert.Equal(t, fleet.ConfigETagStatusRedisNotModified, result.CacheStatus)
	assert.Equal(t, fleet.ConfigETagModeShared, result.Mode)
	assert.Equal(t, 1, store.getCalls)
	assert.Equal(t, 0, store.hostGetCalls, "shared mode must not read per-host records")
	assert.Equal(t, 0, store.setCalls)

	// The whole point: nothing was read from the database.
	assert.False(t, ds.AppConfigFuncInvoked, "short circuit must not read app config")
	assert.False(t, ds.ListPacksForHostFuncInvoked, "short circuit must not list packs")
	assert.False(t, ds.ListScheduledQueriesForAgentsFuncInvoked, "short circuit must not list schedules")
}

// TestConfigETagPerHostShortCircuitHit is THE per-host-mode short circuit
// test: with label-scoped reports in scope, a valid stored per-host ETag
// must produce a not-modified result with ZERO datastore reads, and the shared record must
// never be consulted.
func TestConfigETagPerHostShortCircuitHit(t *testing.T) {
	ds := newETagTestDS()
	svc, ctx := newTestService(t, ds, nil, nil)
	store := &stubConfigETagStore{
		hostETag:  `"host-etag"`,
		hostValid: true,
		scopes:    fleet.ConfigETagLabelScopes{Global: true},
	}
	svc.SetConfigETagStore(store)

	ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
	result, err := svc.GetClientConfigWithETag(ctx, new(`"host-etag"`))
	require.NoError(t, err)

	assert.True(t, result.NotModified)
	assert.Nil(t, result.Body)
	assert.Equal(t, `"host-etag"`, result.ETag)
	assert.Equal(t, fleet.ConfigETagStatusRedisHostNotModified, result.CacheStatus)
	assert.Equal(t, fleet.ConfigETagModeHost, result.Mode)
	assert.Equal(t, 1, store.hostGetCalls)
	assert.Equal(t, 0, store.getCalls, "per-host mode must never read the shared record")
	assert.Equal(t, 0, store.hostSetCalls)

	assert.False(t, ds.AppConfigFuncInvoked, "short circuit must not read app config")
	assert.False(t, ds.ListPacksForHostFuncInvoked, "short circuit must not list packs")
	assert.False(t, ds.ListScheduledQueriesForAgentsFuncInvoked, "short circuit must not list schedules")
}

// TestConfigETagModeSelection: the label-scope state drives shared vs
// per-host mode per the host's effective scope (global ∪ team); legacy packs
// force bypass; gate errors force bypass.
func TestConfigETagModeSelection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		host     *fleet.Host
		store    *stubConfigETagStore
		wantMode fleet.ConfigETagMode
	}{
		{
			"no label scopes -> shared",
			&fleet.Host{ID: 1, Platform: "darwin"},
			&stubConfigETagStore{},
			"shared",
		},
		{
			"global label scope -> host mode for teamless host",
			&fleet.Host{ID: 1, Platform: "darwin"},
			&stubConfigETagStore{scopes: fleet.ConfigETagLabelScopes{Global: true}},
			"host",
		},
		{
			"global label scope -> host mode for team host too (inherited)",
			&fleet.Host{ID: 2, Platform: "darwin", TeamID: new(uint(9))},
			&stubConfigETagStore{scopes: fleet.ConfigETagLabelScopes{Global: true}},
			"host",
		},
		{
			"team label scope -> host mode for that team",
			&fleet.Host{ID: 3, Platform: "darwin", TeamID: new(uint(7))},
			&stubConfigETagStore{scopes: fleet.ConfigETagLabelScopes{TeamIDs: []uint{7}}},
			"host",
		},
		{
			"team label scope -> shared mode for other teams",
			&fleet.Host{ID: 4, Platform: "darwin", TeamID: new(uint(8))},
			&stubConfigETagStore{scopes: fleet.ConfigETagLabelScopes{TeamIDs: []uint{7}}},
			"shared",
		},
		{
			"team label scope -> shared mode for teamless host",
			&fleet.Host{ID: 5, Platform: "darwin"},
			&stubConfigETagStore{scopes: fleet.ConfigETagLabelScopes{TeamIDs: []uint{7}}},
			"shared",
		},
		{
			"legacy packs -> bypass",
			&fleet.Host{ID: 6, Platform: "darwin"},
			&stubConfigETagStore{legacyPresent: true, scopes: fleet.ConfigETagLabelScopes{Global: true}},
			"bypass",
		},
		{
			"legacy gate error -> bypass",
			&fleet.Host{ID: 7, Platform: "darwin"},
			&stubConfigETagStore{legacyErr: assert.AnError},
			"bypass",
		},
		{
			"label scope state error -> bypass (never guess shared)",
			&fleet.Host{ID: 8, Platform: "darwin"},
			&stubConfigETagStore{scopesErr: assert.AnError},
			"bypass",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := newETagTestDS()
			svc, ctx := newTestService(t, ds, nil, nil)
			svc.SetConfigETagStore(tc.store)

			ctx = hostctx.NewContext(ctx, tc.host)
			result, err := svc.GetClientConfigWithETag(ctx, new(`"whatever"`))
			require.NoError(t, err)
			assert.Equal(t, tc.wantMode, result.Mode)

			if tc.wantMode == "bypass" {
				assert.Equal(t, 0, tc.store.getCalls, "bypass must not read Redis records")
				assert.Equal(t, 0, tc.store.hostGetCalls, "bypass must not read Redis records")
				assert.Equal(t, 0, tc.store.setCalls, "bypass must not write Redis records")
				assert.Equal(t, 0, tc.store.hostSetCalls, "bypass must not write Redis records")
				assert.True(t, ds.ListPacksForHostFuncInvoked, "bypass must fall back to a full build")
			}
		})
	}
}

// assertBodyCarriesETag checks that an opted-in agent's Body carries the
// validator under the "etag" key. The hash-covers-canonical-body relationship
// is pinned separately in TestConfigETagBodyContract, which compares the
// opted-in and legacy responses directly rather than re-marshaling (a config
// containing json.RawMessage does not round-trip byte-identically through
// map[string]any).
func assertBodyCarriesETag(t *testing.T, result *fleet.ClientConfigResult) {
	t.Helper()
	var withKey map[string]any
	require.NoError(t, json.Unmarshal(result.Body, &withKey))
	assert.Equal(t, result.ETag, withKey["etag"], "body must carry the validator")
}

// TestConfigETagMissBuildsAndPersists: a miss falls back to a full build,
// and the fresh ETag is offered to the store under the right key family for
// the mode.
func TestConfigETagMissBuildsAndPersists(t *testing.T) {
	t.Run("shared mode", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			host      *fleet.Host
			wantScope string
		}{
			{"global host", &fleet.Host{ID: 1, Platform: "darwin"}, "global"},
			{"team host", &fleet.Host{ID: 2, Platform: "windows", TeamID: new(uint(7))}, "team:7"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				ds := newETagTestDS()
				svc, ctx := newTestService(t, ds, nil, nil)
				store := &stubConfigETagStore{valid: false}
				svc.SetConfigETagStore(store)

				ctx = hostctx.NewContext(ctx, tc.host)
				result, err := svc.GetClientConfigWithETag(ctx, new(`"whatever-stale"`))
				require.NoError(t, err)

				assert.False(t, result.NotModified)
				assert.NotEmpty(t, result.Body)
				// The agent opted in, so Body carries the validator; the
				// validator itself covers the canonical (etag-less) body.
				assertBodyCarriesETag(t, result)
				assert.True(t, ds.ListPacksForHostFuncInvoked, "miss must do a full build")

				require.Equal(t, 1, store.setCalls)
				assert.Equal(t, 0, store.hostSetCalls)
				assert.Equal(t, tc.wantScope, store.lastSetScope)
				assert.Equal(t, tc.host.Platform, store.lastSetPlatform)
				assert.Equal(t, result.ETag, store.lastSetETag)
			})
		}
	})

	t.Run("per-host mode", func(t *testing.T) {
		ds := newETagTestDS()
		svc, ctx := newTestService(t, ds, nil, nil)
		store := &stubConfigETagStore{
			hostValid: false,
			scopes:    fleet.ConfigETagLabelScopes{TeamIDs: []uint{7}},
		}
		svc.SetConfigETagStore(store)

		host := &fleet.Host{ID: 42, Platform: "linux", TeamID: new(uint(7))}
		ctx = hostctx.NewContext(ctx, host)
		result, err := svc.GetClientConfigWithETag(ctx, new(`"whatever-stale"`))
		require.NoError(t, err)

		assert.False(t, result.NotModified)
		assert.NotEmpty(t, result.Body)
		assertBodyCarriesETag(t, result)
		assert.Equal(t, fleet.ConfigETagModeHost, result.Mode)
		assert.True(t, ds.ListPacksForHostFuncInvoked, "miss must do a full build")

		require.Equal(t, 1, store.hostSetCalls)
		assert.Equal(t, 0, store.setCalls, "per-host mode must never publish the shared record")
		assert.Equal(t, uint(42), store.lastHostSetID)
		assert.Equal(t, "team:7", store.lastHostSetScope)
		assert.Equal(t, "linux", store.lastHostSetPlatform)
		assert.Equal(t, result.ETag, store.lastHostSetETag)
	})
}

// TestConfigETagPerHostBypassesTeamPackCache: per-host-mode full builds must
// not consume the team-keyed packConfigCache — its content is one host's
// label-filtered render served team-wide, which would poison per-host
// records with another host's config. Shared mode keeps using it.
func TestConfigETagPerHostBypassesTeamPackCache(t *testing.T) {
	ds := newETagTestDS()
	schedules := []*fleet.Query{{
		Name:     "report-v1",
		Query:    "SELECT 1",
		Interval: 30,
		Logging:  fleet.LoggingSnapshot,
	}}
	ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
		return schedules, nil
	}

	svc, baseCtx := newTestService(t, ds, nil, nil)
	store := &stubConfigETagStore{} // starts with NO label scopes → shared mode
	svc.SetConfigETagStore(store)

	host := &fleet.Host{ID: 1, Platform: "darwin"}
	ctx := hostctx.NewContext(baseCtx, host)

	// 1. Shared-mode request populates the team pack cache with v1.
	first, err := svc.GetClientConfigWithETag(ctx, new(""))
	require.NoError(t, err)
	assert.Contains(t, string(first.Body), "report-v1")

	// 2. The report set changes AND the scope becomes label-scoped (as a
	// query CRUD would cause). The team pack cache still holds v1 for up to
	// a minute.
	schedules = []*fleet.Query{{
		Name:     "report-v2",
		Query:    "SELECT 2",
		Interval: 30,
		Logging:  fleet.LoggingSnapshot,
	}}
	store.scopes = fleet.ConfigETagLabelScopes{Global: true}

	// 3. A per-host-mode build must bypass the cached v1 render and reflect
	// v2 immediately.
	second, err := svc.GetClientConfigWithETag(ctx, new(""))
	require.NoError(t, err)
	assert.Equal(t, fleet.ConfigETagModeHost, second.Mode)
	assert.Contains(t, string(second.Body), "report-v2",
		"per-host mode must bypass the team pack cache")
	assert.NotContains(t, string(second.Body), "report-v1")

	// 4. Control: back in shared mode, the (still unexpired) cached v1
	// render is what a shared build serves — proving step 3 really did
	// bypass the cache rather than the cache having expired.
	store.scopes = fleet.ConfigETagLabelScopes{}
	third, err := svc.GetClientConfigWithETag(ctx, new(""))
	require.NoError(t, err)
	assert.Equal(t, fleet.ConfigETagModeShared, third.Mode)
	assert.Contains(t, string(third.Body), "report-v1",
		"shared mode keeps using the team pack cache")
}

// TestConfigETagScopesUnknownBypassesTeamPackCache: when the deployment has
// no legacy packs but the label-scope state cannot be read (Redis error) or
// is being loaded by another request (leader-election contention), the
// request stays in bypass mode — no Redis record I/O — but its full build
// must still bypass the team-keyed pack cache: the deployment MAY have
// label-scoped reports, in which case the cached render can be another
// host's label-filtered config.
func TestConfigETagScopesUnknownBypassesTeamPackCache(t *testing.T) {
	for _, tc := range []struct {
		name      string
		scopesErr error
	}{
		{"scope state error", assert.AnError},
		{"leader-election contention", fleet.ErrConfigETagGateLoading},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := newETagTestDS()
			schedules := []*fleet.Query{{
				Name:     "report-v1",
				Query:    "SELECT 1",
				Interval: 30,
				Logging:  fleet.LoggingSnapshot,
			}}
			ds.ListScheduledQueriesForAgentsFunc = func(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
				return schedules, nil
			}

			svc, baseCtx := newTestService(t, ds, nil, nil)
			store := &stubConfigETagStore{} // shared mode initially
			svc.SetConfigETagStore(store)

			ctx := hostctx.NewContext(baseCtx, &fleet.Host{ID: 1, Platform: "darwin"})

			// 1. Shared-mode request warms the team pack cache with v1.
			first, err := svc.GetClientConfigWithETag(ctx, new(""))
			require.NoError(t, err)
			assert.Equal(t, fleet.ConfigETagModeShared, first.Mode)
			assert.Contains(t, string(first.Body), "report-v1")

			// 2. Schedules change while scope state becomes unavailable.
			schedules = []*fleet.Query{{
				Name:     "report-v2",
				Query:    "SELECT 2",
				Interval: 30,
				Logging:  fleet.LoggingSnapshot,
			}}
			store.scopesErr = tc.scopesErr

			// 3. Bypass mode with unknown scopes must NOT serve the cached
			// v1 render — and must not touch Redis records (delta-checked:
			// step 1's shared-mode publish legitimately counted one set).
			getBefore, setBefore, hostSetBefore := store.getCalls, store.setCalls, store.hostSetCalls
			second, err := svc.GetClientConfigWithETag(ctx, new(""))
			require.NoError(t, err)
			assert.Equal(t, fleet.ConfigETagModeBypass, second.Mode)
			assert.Contains(t, string(second.Body), "report-v2",
				"unknown label-scope state must bypass the team pack cache")
			assert.NotContains(t, string(second.Body), "report-v1")
			assert.Equal(t, getBefore, store.getCalls, "bypass must not read Redis records")
			assert.Equal(t, setBefore, store.setCalls, "bypass must not write Redis records")
			assert.Equal(t, hostSetBefore, store.hostSetCalls, "bypass must not write Redis records")

			// 4. Control: scope state recovers → shared mode serves the
			// still-warm cached v1 render, proving step 3 truly bypassed.
			store.scopesErr = nil
			third, err := svc.GetClientConfigWithETag(ctx, new(""))
			require.NoError(t, err)
			assert.Equal(t, fleet.ConfigETagModeShared, third.Mode)
			assert.Contains(t, string(third.Body), "report-v1")
		})
	}
}

// TestConfigETagNaiveNotModifiedStillWorks: with a Redis miss (and with no store at
// all), the pre-existing bandwidth-only not-modified path — validator vs freshly built
// body — must keep working.
func TestConfigETagNaiveNotModifiedStillWorks(t *testing.T) {
	for _, withStore := range []bool{true, false} {
		name := "no store (flag off)"
		if withStore {
			name = "store miss"
		}
		t.Run(name, func(t *testing.T) {
			ds := newETagTestDS()
			svc, ctx := newTestService(t, ds, nil, nil)
			store := &stubConfigETagStore{valid: false}
			if withStore {
				svc.SetConfigETagStore(store)
			}

			ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})

			// first fetch: no validator
			first, err := svc.GetClientConfigWithETag(ctx, new(""))
			require.NoError(t, err)
			assert.False(t, first.NotModified)
			assert.Equal(t, fleet.ConfigETagStatusFullNoValidator, first.CacheStatus)
			require.NotEmpty(t, first.ETag)
			if !withStore {
				assert.Equal(t, fleet.ConfigETagModeOff, first.Mode)
			}

			// second fetch presents the etag: full build, then naive not-modified
			second, err := svc.GetClientConfigWithETag(ctx, new(first.ETag))
			require.NoError(t, err)
			assert.True(t, second.NotModified)
			assert.Nil(t, second.Body)
			assert.Equal(t, first.ETag, second.ETag)
			assert.Equal(t, fleet.ConfigETagStatusNotModified, second.CacheStatus)

			if !withStore {
				assert.Equal(t, 0, store.getCalls)
				assert.Equal(t, 0, store.setCalls)
			}
		})
	}
}

// TestConfigETagBodyContract pins the two response bodies of the
// body-carried protocol: an opted-in agent's full response is the config
// with the validator added under the "etag" key, while an agent that did not
// send the field gets the canonical body with no "etag" key — and the
// validator is always computed over the canonical (etag-less) bytes, the
// representation the agent applies after stripping the key.
func TestConfigETagBodyContract(t *testing.T) {
	ds := newETagTestDS()
	svc, baseCtx := newTestService(t, ds, nil, nil)
	ctx := hostctx.NewContext(baseCtx, &fleet.Host{ID: 1, Platform: "darwin"})

	// Not opted in (nil): the canonical body, no etag key, and the validator
	// covers exactly those bytes.
	legacy, err := svc.GetClientConfigWithETag(ctx, nil)
	require.NoError(t, err)
	assert.False(t, legacy.NotModified)
	require.NotNil(t, legacy.Body)
	assert.NotContains(t, string(legacy.Body), `"etag"`)
	assert.Equal(t, clientConfigETag(legacy.Body), legacy.ETag,
		"the validator covers the etag-less body")

	// Opted in (empty etag, first request): the same config with the etag key
	// spliced in. Body is what goes on the wire in both cases.
	optedIn, err := svc.GetClientConfigWithETag(ctx, new(""))
	require.NoError(t, err)
	assert.False(t, optedIn.NotModified)
	require.NotNil(t, optedIn.Body)
	assert.Equal(t, legacy.ETag, optedIn.ETag, "opting in does not change the validator")

	var withKey map[string]any
	require.NoError(t, json.Unmarshal(optedIn.Body, &withKey))
	assert.Equal(t, optedIn.ETag, withKey["etag"])

	// Stripping the etag key yields exactly the bytes a legacy agent gets.
	delete(withKey, "etag")
	stripped, err := marshalClientConfig(withKey)
	require.NoError(t, err)
	assert.Equal(t, string(legacy.Body), string(stripped))

	// Unchanged: no body at all; the endpoint serves the constant.
	unchanged, err := svc.GetClientConfigWithETag(ctx, new(optedIn.ETag))
	require.NoError(t, err)
	assert.True(t, unchanged.NotModified)
	assert.Nil(t, unchanged.Body)
}

// TestConfigETagFeatureDisabled pins the osquery.config_etags escape hatch:
// with the flag off, the agent's etag is ignored entirely — no match is ever
// possible, the response is the legacy body with no etag key, and no etag
// store I/O happens, even with a warm store and a matching validator.
func TestConfigETagFeatureDisabled(t *testing.T) {
	ds := newETagTestDS()
	cfg := config.TestConfig()
	cfg.Osquery.ConfigETags = false
	svc, baseCtx := newTestServiceWithConfig(t, ds, cfg, nil, nil)
	store := &stubConfigETagStore{etag: "stored", valid: true}
	svc.SetConfigETagStore(store)

	ctx := hostctx.NewContext(baseCtx, &fleet.Host{ID: 1, Platform: "darwin"})

	// Even echoing the exact stored validator gets a full legacy build.
	result, err := svc.GetClientConfigWithETag(ctx, new("stored"))
	require.NoError(t, err)
	assert.False(t, result.NotModified)
	require.NotNil(t, result.Body)
	assert.NotContains(t, string(result.Body), `"etag"`,
		"disabled feature must never emit an etag key")
	assert.Equal(t, fleet.ConfigETagModeOff, result.Mode)
	assert.Equal(t, fleet.ConfigETagStatusFullNoValidator, result.CacheStatus)

	// The store is never consulted or written.
	assert.Equal(t, 0, store.getCalls)
	assert.Equal(t, 0, store.hostGetCalls)
	assert.Equal(t, 0, store.setCalls)
	assert.Equal(t, 0, store.hostSetCalls)
}

// TestConfigETagGateLoaders exercises the real loaders (userPacksExist,
// labelScopedReportScopes) through a store stub that defers to the load
// callbacks — proving 2017 packs force bypass and label-scope answers drive
// the mode, end to end through the datastore.
func TestConfigETagGateLoaders(t *testing.T) {
	for _, tc := range []struct {
		name        string
		packs       []*fleet.Pack
		scopes      fleet.ConfigETagLabelScopes
		wantMode    fleet.ConfigETagMode
		wantHostGet int
		wantGet     int
	}{
		{"no blockers -> shared", nil, fleet.ConfigETagLabelScopes{}, fleet.ConfigETagModeShared, 0, 1},
		{"2017 packs -> bypass", []*fleet.Pack{{ID: 1, Name: "p"}}, fleet.ConfigETagLabelScopes{}, fleet.ConfigETagModeBypass, 0, 0},
		{"global label scope -> host", nil, fleet.ConfigETagLabelScopes{Global: true}, fleet.ConfigETagModeHost, 1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := newETagTestDS()
			ds.ListPacksFunc = func(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
				return tc.packs, nil
			}
			ds.LabelScopedScheduledQueryScopesFunc = func(ctx context.Context) (fleet.ConfigETagLabelScopes, error) {
				return tc.scopes, nil
			}

			svc, ctx := newTestService(t, ds, nil, nil)
			store := &stubConfigETagStore{callLoaders: true}
			svc.SetConfigETagStore(store)

			ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
			result, err := svc.GetClientConfigWithETag(ctx, new(`"anything"`))
			require.NoError(t, err)

			assert.Equal(t, tc.wantMode, result.Mode)
			assert.Equal(t, tc.wantGet, store.getCalls)
			assert.Equal(t, tc.wantHostGet, store.hostGetCalls)
		})
	}
}

// TestConfigETagLegacyPackHostNeverPersists: a host that has 2017 packs has
// a HOST-SPECIFIC config beyond what per-host records model (and would
// poison teammates under a shared key). Even if the cached deployment gate
// is momentarily stale (says "no legacy packs"), a build that saw legacy
// packs must never publish.
func TestConfigETagLegacyPackHostNeverPersists(t *testing.T) {
	for _, tc := range []struct {
		name   string
		scopes fleet.ConfigETagLabelScopes
	}{
		{"shared mode", fleet.ConfigETagLabelScopes{}},
		{"per-host mode", fleet.ConfigETagLabelScopes{Global: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := newETagTestDS()
			ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
				return []*fleet.Pack{{ID: 1, Name: "legacy"}}, nil
			}
			ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, pid uint) (fleet.ScheduledQueryList, error) {
				return fleet.ScheduledQueryList{}, nil
			}

			svc, ctx := newTestService(t, ds, nil, nil)
			store := &stubConfigETagStore{scopes: tc.scopes} // stale gate: legacyPresent=false
			svc.SetConfigETagStore(store)

			ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
			result, err := svc.GetClientConfigWithETag(ctx, new(`"anything"`))
			require.NoError(t, err)

			assert.False(t, result.NotModified)
			assert.NotEmpty(t, result.Body)
			assert.Equal(t, 0, store.setCalls, "legacy-pack host ETag must never be persisted")
			assert.Equal(t, 0, store.hostSetCalls, "legacy-pack host ETag must never be persisted")
		})
	}
}

// TestConfigETagFailsOpen: Redis errors on read or write, in either mode,
// must never fail the request nor change the served config.
func TestConfigETagFailsOpen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		store *stubConfigETagStore
	}{
		{"shared read error", &stubConfigETagStore{getErr: assert.AnError}},
		{"shared write error", &stubConfigETagStore{valid: false, setErr: assert.AnError}},
		{"host read error", &stubConfigETagStore{hostGetErr: assert.AnError, scopes: fleet.ConfigETagLabelScopes{Global: true}}},
		{"host write error", &stubConfigETagStore{hostValid: false, hostSetErr: assert.AnError, scopes: fleet.ConfigETagLabelScopes{Global: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := newETagTestDS()
			svc, ctx := newTestService(t, ds, nil, nil)
			svc.SetConfigETagStore(tc.store)

			ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
			result, err := svc.GetClientConfigWithETag(ctx, new(`"anything"`))
			require.NoError(t, err, "redis failures must not fail config delivery")
			assert.False(t, result.NotModified)
			assert.NotEmpty(t, result.Body)
			assert.True(t, ds.ListPacksForHostFuncInvoked, "must fall back to a full build")
		})
	}
}

// TestConfigETagStateLoggedOncePerContainer: the optimization-state log is
// bounded to one line per Fleet container (sync.Once), no matter how many
// config requests are served — state visibility must not scale with request
// volume.
func TestConfigETagStateLoggedOncePerContainer(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ds := newETagTestDS()
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{Logger: logger})
	store := &stubConfigETagStore{}
	svc.SetConfigETagStore(store)

	ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
	for range 5 {
		_, err := svc.GetClientConfigWithETag(ctx, new(`"whatever"`))
		require.NoError(t, err)
	}

	const stateMsg = "config etag optimization state first observed"
	assert.Equal(t, 1, strings.Count(buf.String(), stateMsg),
		"the state log must be emitted exactly once per container")
}

// TestFenceTTLOutlivesInMemoryCaches turns the derivation documented on
// redis_config_etag.DefaultFenceTTL into a build failure.
//
// The fence suppresses ETag publication after a config-affecting write. It has
// to outlive every per-instance in-memory cache that feeds the config build,
// because an instance whose caches are still warm with pre-write data would
// otherwise compute the hash of a stale config and publish it as current —
// freezing every host in the scope on a config that no longer exists. The
// staleness is additive: packConfigCache is filled FROM cached_mysql reads.
//
// If any of those TTLs grows, DefaultFenceTTL must grow with it. Without this
// test that relationship lives only in a comment, and nothing notices when it
// stops holding.
func TestFenceTTLOutlivesInMemoryCaches(t *testing.T) {
	composedStaleness := PackConfigCacheTTL + cached_mysql.MaxConfigInputTTL()

	require.Greater(t, redis_config_etag.DefaultFenceTTL, composedStaleness,
		"DefaultFenceTTL (%s) must exceed packConfigCache TTL (%s) + max cached_mysql "+
			"config-input TTL (%s) = %s; raising either cache TTL requires raising the fence",
		redis_config_etag.DefaultFenceTTL, PackConfigCacheTTL,
		cached_mysql.MaxConfigInputTTL(), composedStaleness)

	// The backstop TTLs exist to bound a missed invalidation, so they must
	// outlive the window in which publication is suppressed.
	assert.Greater(t, redis_config_etag.DefaultETagTTL, redis_config_etag.DefaultFenceTTL,
		"a shared record must outlive the fence, or the cache can never warm")
	assert.Greater(t, redis_config_etag.DefaultHostETagMinTTL, redis_config_etag.DefaultFenceTTL,
		"a per-host record must outlive the fence")
}
