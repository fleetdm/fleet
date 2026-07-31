package service

import (
	"context"
	"encoding/json"
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
	etag       string
	valid      bool
	getErr     error
	blocked    bool
	blockedErr error
	callLoader bool // when true, ShortCircuitBlocked defers to the load callback (exercises the real loader)
	setErr     error

	getCalls        int
	setCalls        int
	blockedCalls    int
	lastSetScope    string
	lastSetPlatform string
	lastSetETag     string
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

func (s *stubConfigETagStore) ShortCircuitBlocked(ctx context.Context, load func(ctx context.Context) (bool, error)) (bool, error) {
	s.blockedCalls++
	if s.blockedErr != nil {
		return true, s.blockedErr
	}
	if s.callLoader {
		blocked, err := load(ctx)
		if err != nil {
			return true, err
		}
		return blocked, nil
	}
	return s.blocked, nil
}

func (s *stubConfigETagStore) ResetShortCircuitBlockedFlag(ctx context.Context) error { return nil }

// newETagTestDS returns a mock datastore sufficient for a minimal full
// config build (no agent options, no packs, no schedules).
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
	ds.HasLabelScopedScheduledQueriesFunc = func(ctx context.Context) (bool, error) {
		return false, nil
	}
	return ds
}

// TestConfigETagShortCircuitHit is THE short circuit test: a valid stored
// ETag matching the client validator must produce a 304 with ZERO datastore
// reads and no ETag write.
func TestConfigETagShortCircuitHit(t *testing.T) {
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
	assert.Equal(t, 1, store.getCalls)
	assert.Equal(t, 0, store.setCalls)

	// The whole point: nothing was read from the database.
	assert.False(t, ds.AppConfigFuncInvoked, "short circuit must not read app config")
	assert.False(t, ds.ListPacksForHostFuncInvoked, "short circuit must not list packs")
	assert.False(t, ds.ListScheduledQueriesForAgentsFuncInvoked, "short circuit must not list schedules")
}

// TestConfigETagMissBuildsAndPersists: a Redis miss (or stale generation)
// falls back to a full build, and the fresh ETag is offered to the store
// under the host's (scope, platform).
func TestConfigETagMissBuildsAndPersists(t *testing.T) {
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
			assert.Equal(t, tc.wantScope, store.lastSetScope)
			assert.Equal(t, tc.host.Platform, store.lastSetPlatform)
			assert.Equal(t, result.ETag, store.lastSetETag)
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

// TestConfigETagBlockedGate: if the deployment has any short circuit blocker
// (2017 packs, label-scoped reports) or the gate errors, the short circuit
// must be fully bypassed — no Redis reads, no Redis writes — while config
// delivery proceeds normally.
func TestConfigETagBlockedGate(t *testing.T) {
	for _, tc := range []struct {
		name  string
		store *stubConfigETagStore
	}{
		{"blocker present", &stubConfigETagStore{blocked: true, etag: `"stored"`, valid: true}},
		{"gate error", &stubConfigETagStore{blockedErr: assert.AnError, etag: `"stored"`, valid: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := newETagTestDS()
			svc, ctx := newTestService(t, ds, nil, nil)
			svc.SetConfigETagStore(tc.store)

			ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
			result, err := svc.GetClientConfigWithETag(ctx, `"stored"`)
			require.NoError(t, err)

			// The stored ETag would have matched — but the gate must keep
			// Redis out of the request entirely.
			assert.NotEqual(t, "redis_not_modified", result.CacheStatus)
			assert.Equal(t, 0, tc.store.getCalls)
			assert.Equal(t, 0, tc.store.setCalls)
			assert.True(t, ds.ListPacksForHostFuncInvoked, "must fall back to a full build")
		})
	}
}

// TestConfigETagLegacyPackHostNeverPersists: a host that has 2017 packs has
// a HOST-SPECIFIC config; persisting its ETag under the team-shared key
// would poison every teammate. The write must be skipped even though the
// deployment-wide gate is open.
func TestConfigETagLegacyPackHostNeverPersists(t *testing.T) {
	ds := newETagTestDS()
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return []*fleet.Pack{{ID: 1, Name: "legacy"}}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, pid uint) (fleet.ScheduledQueryList, error) {
		return fleet.ScheduledQueryList{}, nil
	}

	svc, ctx := newTestService(t, ds, nil, nil)
	store := &stubConfigETagStore{valid: false} // gate open: blocked=false (e.g. stale gate cache)
	svc.SetConfigETagStore(store)

	ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
	result, err := svc.GetClientConfigWithETag(ctx, `"anything"`)
	require.NoError(t, err)

	assert.False(t, result.NotModified)
	assert.NotEmpty(t, result.Body)
	assert.Equal(t, 0, store.setCalls, "legacy-pack host ETag must never be persisted")
}

// TestConfigETagFailsOpen: Redis errors on read or write must never fail the
// request nor change the served config.
func TestConfigETagFailsOpen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		store *stubConfigETagStore
	}{
		{"read error", &stubConfigETagStore{getErr: assert.AnError}},
		{"write error", &stubConfigETagStore{valid: false, setErr: assert.AnError}},
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

// TestConfigETagBlockerLoader exercises the real loader
// (Service.configShortCircuitBlockers) through a store stub that defers to
// the load callback — proving that EITHER 2017 packs OR label-scoped reports
// (query_labels) close the gate, and that neither existing leaves it open.
// Label-scoped reports make ListScheduledQueriesForAgents filter per host,
// so a team-shared ETag would be wrong for them AND would drift with label
// membership (which fires no invalidation event).
func TestConfigETagBlockerLoader(t *testing.T) {
	for _, tc := range []struct {
		name        string
		packs       []*fleet.Pack
		labelScoped bool
		wantBlocked bool
	}{
		{"no blockers", nil, false, false},
		{"2017 packs exist", []*fleet.Pack{{ID: 1, Name: "p"}}, false, true},
		{"label-scoped reports exist", nil, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ds := newETagTestDS()
			ds.ListPacksFunc = func(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
				return tc.packs, nil
			}
			ds.HasLabelScopedScheduledQueriesFunc = func(ctx context.Context) (bool, error) {
				return tc.labelScoped, nil
			}

			svc, ctx := newTestService(t, ds, nil, nil)
			store := &stubConfigETagStore{callLoader: true, etag: `"stored"`, valid: true}
			svc.SetConfigETagStore(store)

			ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, Platform: "darwin"})
			result, err := svc.GetClientConfigWithETag(ctx, `"stored"`)
			require.NoError(t, err)

			if tc.wantBlocked {
				assert.NotEqual(t, "redis_not_modified", result.CacheStatus)
				assert.Equal(t, 0, store.getCalls, "blocked gate must keep Redis reads out")
				assert.Equal(t, 0, store.setCalls, "blocked gate must keep Redis writes out")
			} else {
				assert.Equal(t, "redis_not_modified", result.CacheStatus)
				assert.Equal(t, 1, store.getCalls)
			}
		})
	}
}
