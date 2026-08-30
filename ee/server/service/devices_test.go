package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redismock "github.com/fleetdm/fleet/v4/server/mock/redis"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/stretchr/testify/require"
)

func newDeviceSSOTestService(t *testing.T, ds fleet.Datastore, sessionDuration time.Duration) (*Service, func(key string) *string, *clock.MockClock) {
	t.Helper()
	svc := newTestService(t, ds)
	mockClock := clock.NewMockClock()
	kvs, getKey := inMemoryKeyValueStore()
	svc.clock = mockClock
	svc.config = config.FleetConfig{Session: config.SessionConfig{Duration: sessionDuration}}
	svc.keyValueStore = kvs
	return svc, getKey, mockClock
}

func TestCreateDeviceSSOSession(t *testing.T) {
	svc, getKey, mockClock := newDeviceSSOTestService(t, new(mock.Store), time.Hour)

	ctx := t.Context()
	host := &fleet.Host{ID: 42, UUID: "host-uuid-1"}

	sessionID, ttl, err := svc.createDeviceSSOSession(ctx, host, "idp-acct-uuid")
	require.NoError(t, err)
	require.NotEmpty(t, sessionID)
	require.Equal(t, time.Hour, ttl)

	// Stored under the namespaced key, so it cannot collide with the other users
	// of the shared key/value store. Read back through the store rather than a
	// service method: the gate that consumes these sessions arrives with the
	// enforcement sub-issue.
	stored := getKey(deviceSSOSessionKeyPrefix + sessionID)
	require.NotNil(t, stored)

	var session fleet.DeviceSSOSession
	require.NoError(t, json.Unmarshal([]byte(*stored), &session))
	require.Equal(t, host.ID, session.HostID)
	require.Equal(t, "idp-acct-uuid", session.IdPAccountUUID)
	require.Equal(t, mockClock.Now().Add(time.Hour), session.ExpiresAt.UTC())
}

func TestDeviceSSOSessionExpires(t *testing.T) {
	svc, _, mockClock := newDeviceSSOTestService(t, new(mock.Store), time.Hour)

	ctx := t.Context()
	sessionID, _, err := svc.createDeviceSSOSession(ctx, &fleet.Host{ID: 42, UUID: "host-uuid-1"}, "idp-acct-uuid")
	require.NoError(t, err)

	// still valid just before the deadline
	mockClock.AddTime(time.Hour - time.Second)
	_, err = svc.validateDeviceSSOSession(ctx, sessionID)
	require.NoError(t, err)

	// the absolute deadline is enforced on read even though this store never
	// drops the key, so a store that stops honoring TTLs can't keep the session
	// alive.
	mockClock.AddTime(2 * time.Second)
	_, err = svc.validateDeviceSSOSession(ctx, sessionID)
	require.Error(t, err)
}

func TestDeviceSSOSessionUnknownIDNotFound(t *testing.T) {
	svc, _, _ := newDeviceSSOTestService(t, new(mock.Store), time.Hour)

	for _, sessionID := range []string{"", "does-not-exist"} {
		_, err := svc.validateDeviceSSOSession(t.Context(), sessionID)
		require.Error(t, err)
	}
}

func TestInitiateDeviceSSO(t *testing.T) {
	const devTok = "device-auth-token-abc123"

	idp := func(ac *fleet.AppConfig) {
		ac.MDM.EndUserAuthentication.SSOProviderSettings = fleet.SSOProviderSettings{
			EntityID: "fleet",
			IDPName:  "TestIDP",
			Metadata: mdmSSOTestMetadata,
		}
	}

	cases := []struct {
		name      string
		noHost    bool
		appConfig func() fleet.AppConfig
		wantErr   string
	}{
		{
			name:      "no host resolved from the device token",
			noHost:    true,
			wantErr:   "Authentication required",
			appConfig: func() fleet.AppConfig { return fleet.AppConfig{} },
		},
		{
			name:      "sso disabled",
			wantErr:   "is not enabled",
			appConfig: func() fleet.AppConfig { return fleet.AppConfig{} },
		},
		{
			name:    "sso enabled but no IdP configured",
			wantErr: "no IdP is configured",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				return ac
			},
		},
		{
			name:    "alternative browser host differs from the ACS host",
			wantErr: "same host",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.FleetDesktop.AlternativeBrowserHost = "proxy.example.com"
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				idp(&ac)
				return ac
			},
		},
		{
			name:    "custom Apple MDM URL differs from the server URL",
			wantErr: "same host",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				ac.MDM.AppleServerURL = "https://mdm.example.com"
				idp(&ac)
				return ac
			},
		},
		{
			name: "no alternative browser host",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				idp(&ac)
				return ac
			},
		},
		{
			name: "alternative browser host equals the server host",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.FleetDesktop.AlternativeBrowserHost = "fleet.example.com"
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				idp(&ac)
				return ac
			},
		},
		{
			name: "alternative browser host differs only by port",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.FleetDesktop.AlternativeBrowserHost = "fleet.example.com:8443"
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				idp(&ac)
				return ac
			},
		},
		{
			name: "apple server url equals the server url",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				ac.MDM.AppleServerURL = "https://fleet.example.com"
				idp(&ac)
				return ac
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				ac := c.appConfig()
				return &ac, nil
			}

			svc, _, _ := newDeviceSSOTestService(t, ds, time.Hour)
			svc.ssoSessionStore = sso.NewSessionStore(redistest.NopRedis())

			ctx := t.Context()
			if !c.noHost {
				ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 1, UUID: "host-uuid-1"})
			}

			initiation, err := svc.InitiateDeviceSSO(ctx, "/device/"+devTok)

			if c.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, c.wantErr)
				if c.noHost {
					var authRequired *fleet.AuthRequiredError
					require.ErrorAs(t, err, &authRequired)
				} else {
					var badReq *fleet.BadRequestError
					require.ErrorAs(t, err, &badReq)
				}
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, initiation.IdPURL)

			idpURL, err := url.Parse(initiation.IdPURL)
			require.NoError(t, err)

			require.Equal(t,
				string(fleet.SSORelayState(fleet.SSOInitiatorFleetDesktop)),
				idpURL.Query().Get("RelayState"))
			require.NotContains(t, idpURL.RawQuery, devTok)
		})
	}
}

func TestRequireDeviceSSOSession(t *testing.T) {
	host := &fleet.Host{ID: 1, UUID: "host-uuid-1", Platform: "darwin"}
	otherHost := &fleet.Host{ID: 2, UUID: "host-uuid-2", Platform: "darwin"}

	// mintFor returns a session ID for h, or "" for the callers that stand in for
	// a browser with no cookie yet.
	type sessionFor func(t *testing.T, svc *Service, ctx context.Context) string
	noSession := func(*testing.T, *Service, context.Context) string { return "" }
	sessionOf := func(h *fleet.Host) sessionFor {
		return func(t *testing.T, svc *Service, ctx context.Context) string {
			sessionID, _, err := svc.createDeviceSSOSession(ctx, h, "idp-acct-uuid")
			require.NoError(t, err)
			return sessionID
		}
	}

	cases := []struct {
		name                  string
		ssoEnabled            bool
		platform              string // defaults to darwin
		awaitingConfiguration bool   // darwin only: the Setup Assistant flag
		setupExperienceStatus fleet.SetupExperienceStatusResultStatus
		session               sessionFor
		advanceClock          time.Duration
		wantSSORequired       bool
		wantSessionLookup     bool
	}{
		{
			name:       "setting off allows the request",
			ssoEnabled: false,
			session:    noSession,
		},
		{
			name:              "session bound to the host allows the request",
			ssoEnabled:        true,
			session:           sessionOf(host),
			wantSessionLookup: true,
		},
		{
			name:              "no session requires sso",
			ssoEnabled:        true,
			session:           noSession,
			wantSSORequired:   true,
			wantSessionLookup: true,
		},
		{
			name:              "expired session requires sso",
			ssoEnabled:        true,
			session:           sessionOf(host),
			advanceClock:      2 * time.Hour,
			wantSSORequired:   true,
			wantSessionLookup: true,
		},
		{
			name:              "session minted for another host requires sso",
			ssoEnabled:        true,
			session:           sessionOf(otherHost),
			wantSSORequired:   true,
			wantSessionLookup: true,
		},
		{
			name:                  "host in setup experience is exempt",
			ssoEnabled:            true,
			awaitingConfiguration: true,
			session:               noSession,
			wantSessionLookup:     true,
		},
		{
			name:                  "linux host mid setup experience is exempt",
			ssoEnabled:            true,
			platform:              "ubuntu",
			setupExperienceStatus: fleet.SetupExperienceStatusRunning,
			session:               noSession,
			wantSessionLookup:     true,
		},
		{
			name:                  "windows host mid setup experience is exempt",
			ssoEnabled:            true,
			platform:              "windows",
			setupExperienceStatus: fleet.SetupExperienceStatusPending,
			session:               noSession,
			wantSessionLookup:     true,
		},
		{
			name:                  "linux host past setup experience requires sso",
			ssoEnabled:            true,
			platform:              "ubuntu",
			setupExperienceStatus: fleet.SetupExperienceStatusSuccess,
			session:               noSession,
			wantSSORequired:       true,
			wantSessionLookup:     true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = c.ssoEnabled
				return &ac, nil
			}
			ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
				require.Equal(t, host.UUID, hostUUID)
				return c.awaitingConfiguration, nil
			}
			ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hostUUID string, teamID uint) ([]*fleet.SetupExperienceStatusResult, error) {
				return []*fleet.SetupExperienceStatusResult{
					{Name: "install something", Status: c.setupExperienceStatus, HostUUID: hostUUID, SoftwareInstallerID: new(uint(1))},
				}, nil
			}

			svc, getKey, mockClock := newDeviceSSOTestService(t, ds, time.Hour)
			ctx := t.Context()

			caseHost := *host
			if c.platform != "" {
				caseHost.Platform = c.platform
				caseHost.OsqueryHostID = new("osquery-id")
			}

			sessionID := c.session(t, svc, ctx)
			mockClock.AddTime(c.advanceClock)

			err := svc.RequireDeviceSSOSession(ctx, &caseHost, sessionID)

			if c.wantSSORequired {
				var ssoRequired *fleet.DeviceSSORequiredError
				require.ErrorAs(t, err, &ssoRequired)
			} else {
				require.NoError(t, err)
			}

			// With the setting off nothing is looked up at all: no session read and
			// no setup experience query, so a Fleet running without the feature
			// pays nothing for it.
			if !c.wantSessionLookup {
				require.Nil(t, getKey(deviceSSOSessionKeyPrefix+sessionID))
				require.False(t, ds.GetHostAwaitingConfigurationFuncInvoked)
			}
		})
	}
}

func TestRequireDeviceSSOSessionSetupExperienceSurvivesStoreFailure(t *testing.T) {
	host := &fleet.Host{ID: 1, UUID: "host-uuid-1", Platform: "darwin"}

	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		var ac fleet.AppConfig
		ac.FleetDesktop.SSOEnabled = true
		return &ac, nil
	}
	ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
		return true, nil
	}

	svc, _, _ := newDeviceSSOTestService(t, ds, time.Hour)
	svc.keyValueStore = &redismock.KeyValueStore{
		GetFunc: func(ctx context.Context, key string) (*string, error) {
			return nil, errors.New("redis is down")
		},
	}

	require.NoError(t, svc.RequireDeviceSSOSession(t.Context(), host, ""))
}

func TestRequireDeviceSSOSessionMissingSetupExperienceRow(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		var ac fleet.AppConfig
		ac.FleetDesktop.SSOEnabled = true
		return &ac, nil
	}
	// Hosts that never ran setup experience have no row at all.
	ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
		return false, &notFoundError{}
	}

	svc, _, _ := newDeviceSSOTestService(t, ds, time.Hour)

	var ssoRequired *fleet.DeviceSSORequiredError
	require.ErrorAs(t, svc.RequireDeviceSSOSession(t.Context(), &fleet.Host{ID: 1, UUID: "host-uuid-1", Platform: "darwin"}, ""), &ssoRequired)
}

func TestRequireDeviceSSOSessionStoreFailure(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		var ac fleet.AppConfig
		ac.FleetDesktop.SSOEnabled = true
		return &ac, nil
	}
	ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
		return false, nil
	}

	svc, _, _ := newDeviceSSOTestService(t, ds, time.Hour)
	svc.keyValueStore = &redismock.KeyValueStore{
		GetFunc: func(ctx context.Context, key string) (*string, error) {
			return nil, errors.New("redis is down")
		},
	}

	err := svc.RequireDeviceSSOSession(t.Context(), &fleet.Host{ID: 1, UUID: "host-uuid-1", Platform: "darwin"}, "some-session")

	require.ErrorContains(t, err, "redis is down")
	var ssoRequired *fleet.DeviceSSORequiredError
	require.NotErrorAs(t, err, &ssoRequired, "a broken session store must not read as a prompt to sign in")
}
