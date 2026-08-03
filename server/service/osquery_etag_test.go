package service

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
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

func (s *stubConfigETagStore) GetValid(ctx context.Context, scope, platform string) (string, bool, error) {
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

func (s *stubConfigETagStore) GetValidHost(ctx context.Context, hostID uint, scope, platform string) (string, bool, error) {
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
// a valid stored ETag matching the client validator must produce a 304 with
// ZERO datastore reads and no ETag write.
func TestConfigETagSharedShortCircuitHit(t *testing.T) {
	ds := newETagTestDS()
	svc, ctx := newTestService(t, ds, nil, nil)
	store := &stubConfigETagStore{etag: `"stored"`, valid: true}
	svc.SetConfigETagStore(store)

	ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
	result, err := svc.GetClientConfigWithETag(ctx, `"stored"`)
	require.NoError(t, err)

	assert.True(t, result.NotModified)
	assert.Nil(t, result.Body, "304 must carry no body")
	assert.Equal(t, `"stored"`, result.ETag)
	assert.Equal(t, "redis_not_modified", result.CacheStatus)
	assert.Equal(t, "shared", result.Mode)
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
// must produce a 304 with ZERO datastore reads, and the shared record must
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
	result, err := svc.GetClientConfigWithETag(ctx, `"host-etag"`)
	require.NoError(t, err)

	assert.True(t, result.NotModified)
	assert.Nil(t, result.Body)
	assert.Equal(t, `"host-etag"`, result.ETag)
	assert.Equal(t, "redis_host_not_modified", result.CacheStatus)
	assert.Equal(t, "host", result.Mode)
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
		wantMode string
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
			result, err := svc.GetClientConfigWithETag(ctx, `"whatever"`)
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
				result, err := svc.GetClientConfigWithETag(ctx, `"whatever-stale"`)
				require.NoError(t, err)

				assert.False(t, result.NotModified)
				assert.NotEmpty(t, result.Body)
				assert.Equal(t, clientConfigETag(result.Body), result.ETag)
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
		result, err := svc.GetClientConfigWithETag(ctx, `"whatever-stale"`)
		require.NoError(t, err)

		assert.False(t, result.NotModified)
		assert.NotEmpty(t, result.Body)
		assert.Equal(t, "host", result.Mode)
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
	first, err := svc.GetClientConfigWithETag(ctx, "")
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
	second, err := svc.GetClientConfigWithETag(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "host", second.Mode)
	assert.Contains(t, string(second.Body), "report-v2",
		"per-host mode must bypass the team pack cache")
	assert.NotContains(t, string(second.Body), "report-v1")

	// 4. Control: back in shared mode, the (still unexpired) cached v1
	// render is what a shared build serves — proving step 3 really did
	// bypass the cache rather than the cache having expired.
	store.scopes = fleet.ConfigETagLabelScopes{}
	third, err := svc.GetClientConfigWithETag(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, "shared", third.Mode)
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
			first, err := svc.GetClientConfigWithETag(ctx, "")
			require.NoError(t, err)
			assert.Equal(t, "shared", first.Mode)
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
			second, err := svc.GetClientConfigWithETag(ctx, "")
			require.NoError(t, err)
			assert.Equal(t, "bypass", second.Mode)
			assert.Contains(t, string(second.Body), "report-v2",
				"unknown label-scope state must bypass the team pack cache")
			assert.NotContains(t, string(second.Body), "report-v1")
			assert.Equal(t, getBefore, store.getCalls, "bypass must not read Redis records")
			assert.Equal(t, setBefore, store.setCalls, "bypass must not write Redis records")
			assert.Equal(t, hostSetBefore, store.hostSetCalls, "bypass must not write Redis records")

			// 4. Control: scope state recovers → shared mode serves the
			// still-warm cached v1 render, proving step 3 truly bypassed.
			store.scopesErr = nil
			third, err := svc.GetClientConfigWithETag(ctx, "")
			require.NoError(t, err)
			assert.Equal(t, "shared", third.Mode)
			assert.Contains(t, string(third.Body), "report-v1")
		})
	}
}

// TestConfigETagNaive304StillWorks: with a Redis miss (and with no store at
// all), the pre-existing bandwidth-only 304 — validator vs freshly built
// body — must keep working.
func TestConfigETagNaive304StillWorks(t *testing.T) {
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
			first, err := svc.GetClientConfigWithETag(ctx, "")
			require.NoError(t, err)
			assert.False(t, first.NotModified)
			assert.Equal(t, "full_no_validator", first.CacheStatus)
			require.NotEmpty(t, first.ETag)
			if !withStore {
				assert.Equal(t, "off", first.Mode)
			}

			// second fetch presents the etag: full build, then naive 304
			second, err := svc.GetClientConfigWithETag(ctx, first.ETag)
			require.NoError(t, err)
			assert.True(t, second.NotModified)
			assert.Nil(t, second.Body)
			assert.Equal(t, first.ETag, second.ETag)
			assert.Equal(t, "not_modified", second.CacheStatus)

			if !withStore {
				assert.Equal(t, 0, store.getCalls)
				assert.Equal(t, 0, store.setCalls)
			}
		})
	}
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
		wantMode    string
		wantHostGet int
		wantGet     int
	}{
		{"no blockers -> shared", nil, fleet.ConfigETagLabelScopes{}, "shared", 0, 1},
		{"2017 packs -> bypass", []*fleet.Pack{{ID: 1, Name: "p"}}, fleet.ConfigETagLabelScopes{}, "bypass", 0, 0},
		{"global label scope -> host", nil, fleet.ConfigETagLabelScopes{Global: true}, "host", 1, 0},
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
			result, err := svc.GetClientConfigWithETag(ctx, `"anything"`)
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
			result, err := svc.GetClientConfigWithETag(ctx, `"anything"`)
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
			result, err := svc.GetClientConfigWithETag(ctx, `"anything"`)
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
		_, err := svc.GetClientConfigWithETag(ctx, `"whatever"`)
		require.NoError(t, err)
	}

	const stateMsg = "config etag optimization state first observed"
	assert.Equal(t, 1, strings.Count(buf.String(), stateMsg),
		"the state log must be emitted exactly once per container")
}
