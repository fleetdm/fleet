package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// orbitHost returns a host with an OrbitNodeKey set so the function doesn't
// short-circuit on the "vanilla osquery" check.
func orbitHost(id uint, teamID *uint) *fleet.Host {
	return &fleet.Host{
		ID:           id,
		Platform:     "darwin",
		TeamID:       teamID,
		OrbitNodeKey: ptr.String("orbit-node-key"),
	}
}

// testServiceForCloudIdentity builds a Service backed by a mock store, with
// the GoogleCloudIdentity config UNSET (so the syncer is nil and any SyncHost
// goroutine is skipped — tests can focus on the policy/label fan-out).
func testServiceForCloudIdentity(t *testing.T, ds *mock.Store) *Service {
	t.Helper()
	svc, _ := newTestService(t, ds, nil, nil)
	serv := ((svc.(validationMiddleware)).Service).(*Service)
	return serv
}

// shortCircuitSyncerOnce pre-consumes the syncer's sync.Once with a no-op so
// googleCloudIdentitySyncerOrNil returns (nil, nil) without trying to load
// credentials. Tests that exercise the policy/label fan-out can call this to
// have the function complete cleanly without needing a real SA-JSON key.
func shortCircuitSyncerOnce(svc *Service) {
	svc.googleCloudIdentitySyncerOnce.Do(func() {})
}

func TestProcessGoogleCloudIdentityForNewlyFailingPolicies_VanillaOsqueryShortCircuits(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := testServiceForCloudIdentity(t, ds)

	host := &fleet.Host{ID: 1, Platform: "darwin"} // no OrbitNodeKey
	err := svc.processGoogleCloudIdentityForNewlyFailingPolicies(t.Context(), host, nil)
	require.NoError(t, err)
	// No datastore calls should have happened.
	assert.False(t, ds.GetPoliciesForConditionalAccessFuncInvoked)
	assert.False(t, ds.PoliciesByIDFuncInvoked)
	assert.False(t, ds.ListLabelsForHostFuncInvoked)
	assert.False(t, ds.AppConfigFuncInvoked)
}

func TestProcessGoogleCloudIdentityForNewlyFailingPolicies_NotConfigured(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := testServiceForCloudIdentity(t, ds)

	host := orbitHost(1, nil) // No team -> would consult AppConfig, but config not set short-circuits first.
	err := svc.processGoogleCloudIdentityForNewlyFailingPolicies(t.Context(), host, nil)
	require.NoError(t, err)
	assert.False(t, ds.GetPoliciesForConditionalAccessFuncInvoked)
	assert.False(t, ds.AppConfigFuncInvoked)
}

func TestProcessGoogleCloudIdentityForNewlyFailingPolicies_DisabledOnTeam(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		require.Equal(t, uint(42), tid)
		return &fleet.TeamLite{ID: 42}, nil // GoogleCloudIdentityEnabled unset -> disabled.
	}
	svc := testServiceForCloudIdentity(t, ds)
	svc.config.GoogleCloudIdentity = configWithCredentialsSet()
	shortCircuitSyncerOnce(svc)

	host := orbitHost(1, ptr.Uint(42))
	err := svc.processGoogleCloudIdentityForNewlyFailingPolicies(t.Context(), host, nil)
	require.NoError(t, err)
	assert.True(t, ds.TeamLiteFuncInvoked)
	assert.False(t, ds.GetPoliciesForConditionalAccessFuncInvoked, "should not load CA policies when team disabled")
}

func TestProcessGoogleCloudIdentityForNewlyFailingPolicies_HappyPath_NoTeam(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{}
		ac.Integrations.GoogleCloudIdentityEnabled.Set = true
		ac.Integrations.GoogleCloudIdentityEnabled.Valid = true
		ac.Integrations.GoogleCloudIdentityEnabled.Value = true
		return ac, nil
	}
	ds.GetPoliciesForConditionalAccessFunc = func(ctx context.Context, teamID uint, platform string) ([]uint, error) {
		assert.Equal(t, fleet.PolicyNoTeamID, teamID, "host with nil TeamID should use PolicyNoTeamID")
		assert.Equal(t, "darwin", platform)
		return []uint{10, 20, 30}, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
		// Only failing IDs (20) get loaded.
		assert.ElementsMatch(t, []uint{20}, ids)
		return map[uint]*fleet.Policy{20: {PolicyData: fleet.PolicyData{ID: 20, Name: "Disk encryption"}}}, nil
	}
	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		return &fleet.HostMDM{HostID: hostID, Enrolled: true}, nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return []*fleet.Label{{Name: "engineering"}, {Name: ""}, nil, {Name: "production"}}, nil
	}

	svc := testServiceForCloudIdentity(t, ds)
	svc.config.GoogleCloudIdentity = configWithCredentialsSet()
	shortCircuitSyncerOnce(svc)

	host := orbitHost(7, nil)
	// 10 passing, 20 failing, 30 not in incoming results (skipped). 99 is not CA-flagged (skipped).
	incoming := map[uint]*bool{
		10: new(true),
		20: new(false),
		99: new(false),
	}
	err := svc.processGoogleCloudIdentityForNewlyFailingPolicies(t.Context(), host, incoming)
	require.NoError(t, err)

	assert.True(t, ds.AppConfigFuncInvoked)
	assert.True(t, ds.GetPoliciesForConditionalAccessFuncInvoked)
	assert.True(t, ds.PoliciesByIDFuncInvoked)
	assert.True(t, ds.GetHostMDMFuncInvoked)
	assert.True(t, ds.ListLabelsForHostFuncInvoked)
}

func TestProcessGoogleCloudIdentityForNewlyFailingPolicies_PolicyByIDFallbackToPolicyN(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		tl := &fleet.TeamLite{ID: tid}
		tl.Config.Integrations.GoogleCloudIdentityEnabled.Set = true
		tl.Config.Integrations.GoogleCloudIdentityEnabled.Valid = true
		tl.Config.Integrations.GoogleCloudIdentityEnabled.Value = true
		return tl, nil
	}
	ds.GetPoliciesForConditionalAccessFunc = func(ctx context.Context, teamID uint, platform string) ([]uint, error) {
		return []uint{1, 2}, nil
	}
	// PoliciesByID returns a map missing id=2 — exercise the "policy_%d" fallback branch.
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
		return map[uint]*fleet.Policy{1: {PolicyData: fleet.PolicyData{ID: 1, Name: "alpha"}}}, nil
	}
	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		return nil, newNotFoundError() // exercise the "not found is fine" branch
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return nil, nil
	}

	svc := testServiceForCloudIdentity(t, ds)
	svc.config.GoogleCloudIdentity = configWithCredentialsSet()
	shortCircuitSyncerOnce(svc)

	host := orbitHost(1, ptr.Uint(5))
	err := svc.processGoogleCloudIdentityForNewlyFailingPolicies(t.Context(), host, map[uint]*bool{
		1: new(false),
		2: new(false),
	})
	require.NoError(t, err)
}

func TestProcessGoogleCloudIdentityForNewlyFailingPolicies_ListLabelsForHostNonFatal(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{}
		ac.Integrations.GoogleCloudIdentityEnabled.Set = true
		ac.Integrations.GoogleCloudIdentityEnabled.Valid = true
		ac.Integrations.GoogleCloudIdentityEnabled.Value = true
		return ac, nil
	}
	ds.GetPoliciesForConditionalAccessFunc = func(ctx context.Context, teamID uint, platform string) ([]uint, error) {
		return nil, nil
	}
	ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
		return nil, nil
	}
	ds.ListLabelsForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Label, error) {
		return nil, errors.New("labels DB exploded")
	}

	svc := testServiceForCloudIdentity(t, ds)
	svc.config.GoogleCloudIdentity = configWithCredentialsSet()
	shortCircuitSyncerOnce(svc)

	host := orbitHost(1, nil)
	err := svc.processGoogleCloudIdentityForNewlyFailingPolicies(t.Context(), host, nil)
	require.NoError(t, err, "ListLabelsForHost failure is non-fatal — should not bubble up")
}

func TestProcessGoogleCloudIdentityForNewlyFailingPolicies_PoliciesByIDErrorPropagates(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{}
		ac.Integrations.GoogleCloudIdentityEnabled.Set = true
		ac.Integrations.GoogleCloudIdentityEnabled.Valid = true
		ac.Integrations.GoogleCloudIdentityEnabled.Value = true
		return ac, nil
	}
	ds.GetPoliciesForConditionalAccessFunc = func(ctx context.Context, teamID uint, platform string) ([]uint, error) {
		return []uint{1}, nil
	}
	ds.PoliciesByIDFunc = func(ctx context.Context, ids []uint) (map[uint]*fleet.Policy, error) {
		return nil, errors.New("policies DB on fire")
	}

	svc := testServiceForCloudIdentity(t, ds)
	svc.config.GoogleCloudIdentity = configWithCredentialsSet()
	shortCircuitSyncerOnce(svc)

	host := orbitHost(1, nil)
	err := svc.processGoogleCloudIdentityForNewlyFailingPolicies(t.Context(), host, map[uint]*bool{
		1: new(false),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "google cloud identity")
}

func TestGoogleCloudIdentityConfiguredAndEnabledForTeam(t *testing.T) {
	t.Parallel()

	t.Run("not configured", func(t *testing.T) {
		t.Parallel()
		ds := new(mock.Store)
		svc := testServiceForCloudIdentity(t, ds)
		// config left default (zero)
		configured, enabled, err := svc.googleCloudIdentityConfiguredAndEnabledForTeam(t.Context(), nil)
		require.NoError(t, err)
		assert.False(t, configured)
		assert.False(t, enabled)
	})

	t.Run("no team, AppConfig enabled=true", func(t *testing.T) {
		t.Parallel()
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.Integrations.GoogleCloudIdentityEnabled.Set = true
			ac.Integrations.GoogleCloudIdentityEnabled.Valid = true
			ac.Integrations.GoogleCloudIdentityEnabled.Value = true
			return ac, nil
		}
		svc := testServiceForCloudIdentity(t, ds)
		svc.config.GoogleCloudIdentity = configWithCredentialsSet()

		configured, enabled, err := svc.googleCloudIdentityConfiguredAndEnabledForTeam(t.Context(), nil)
		require.NoError(t, err)
		assert.True(t, configured)
		assert.True(t, enabled)
	})

	t.Run("no team, AppConfig enabled unset -> false", func(t *testing.T) {
		t.Parallel()
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil // GoogleCloudIdentityEnabled not Set
		}
		svc := testServiceForCloudIdentity(t, ds)
		svc.config.GoogleCloudIdentity = configWithCredentialsSet()

		configured, enabled, err := svc.googleCloudIdentityConfiguredAndEnabledForTeam(t.Context(), nil)
		require.NoError(t, err)
		assert.True(t, configured)
		assert.False(t, enabled)
	})

	t.Run("team, enabled=true", func(t *testing.T) {
		t.Parallel()
		ds := new(mock.Store)
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			tl := &fleet.TeamLite{ID: tid}
			tl.Config.Integrations.GoogleCloudIdentityEnabled.Set = true
			tl.Config.Integrations.GoogleCloudIdentityEnabled.Valid = true
			tl.Config.Integrations.GoogleCloudIdentityEnabled.Value = true
			return tl, nil
		}
		svc := testServiceForCloudIdentity(t, ds)
		svc.config.GoogleCloudIdentity = configWithCredentialsSet()

		configured, enabled, err := svc.googleCloudIdentityConfiguredAndEnabledForTeam(t.Context(), ptr.Uint(7))
		require.NoError(t, err)
		assert.True(t, configured)
		assert.True(t, enabled)
	})

	t.Run("team, enabled unset -> false", func(t *testing.T) {
		t.Parallel()
		ds := new(mock.Store)
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			return &fleet.TeamLite{ID: tid}, nil
		}
		svc := testServiceForCloudIdentity(t, ds)
		svc.config.GoogleCloudIdentity = configWithCredentialsSet()

		configured, enabled, err := svc.googleCloudIdentityConfiguredAndEnabledForTeam(t.Context(), ptr.Uint(7))
		require.NoError(t, err)
		assert.True(t, configured)
		assert.False(t, enabled)
	})

	t.Run("AppConfig error wraps", func(t *testing.T) {
		t.Parallel()
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return nil, errors.New("appconfig boom")
		}
		svc := testServiceForCloudIdentity(t, ds)
		svc.config.GoogleCloudIdentity = configWithCredentialsSet()

		_, _, err := svc.googleCloudIdentityConfiguredAndEnabledForTeam(t.Context(), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "google cloud identity")
	})

	t.Run("TeamLite error wraps", func(t *testing.T) {
		t.Parallel()
		ds := new(mock.Store)
		ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
			return nil, errors.New("team DB explosion")
		}
		svc := testServiceForCloudIdentity(t, ds)
		svc.config.GoogleCloudIdentity = configWithCredentialsSet()

		_, _, err := svc.googleCloudIdentityConfiguredAndEnabledForTeam(t.Context(), ptr.Uint(3))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "google cloud identity")
	})
}

func TestGoogleCloudIdentitySyncerOrNil_ConfigUnset(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := testServiceForCloudIdentity(t, ds)

	got, err := svc.googleCloudIdentitySyncerOrNil(t.Context())
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestGoogleCloudIdentitySyncerOrNil_BadCredsMemoizedError(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := testServiceForCloudIdentity(t, ds)
	// Set config with a bogus SA-JSON path so NewTokenSource fails predictably.
	svc.config.GoogleCloudIdentity = config.GoogleCloudIdentityConfig{
		ServiceAccountJSON: "/nonexistent/path/sa.json",
		ImpersonatedAdmin:  "admin@example.com",
		CustomerID:         "C0xxxxxxx",
		WorkspaceDomains:   "example.com",
	}

	got, err1 := svc.googleCloudIdentitySyncerOrNil(t.Context())
	require.Error(t, err1)
	assert.Nil(t, got)
	assert.Contains(t, err1.Error(), "google cloud identity")

	// Second call returns the SAME memoized error (sync.Once guard).
	got, err2 := svc.googleCloudIdentitySyncerOrNil(t.Context())
	require.Error(t, err2)
	assert.Nil(t, got)
	assert.Same(t, err1, err2, "Once-init error must be memoized; second call should return the same error pointer")
}

// configWithCredentialsSet returns a config block where IsSet() returns true.
// We pass an SA-JSON path so the auth layer COULD initialize; for syncer-or-nil
// tests we want config IsSet=true but credentials don't actually need to work
// (we never trigger the sync.Once init in those tests).
func configWithCredentialsSet() config.GoogleCloudIdentityConfig {
	return config.GoogleCloudIdentityConfig{
		ServiceAccountJSON: "/tmp/sa.json", // need non-empty so IsSet returns true
		ImpersonatedAdmin:  "admin@example.com",
		CustomerID:         "C0xxxxxxx",
		WorkspaceDomains:   "example.com",
	}
}
