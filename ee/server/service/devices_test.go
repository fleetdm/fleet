package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
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

func TestInitiateDeviceSSOGuards(t *testing.T) {
	idpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(idpSrv.Close)

	idp := func(ac *fleet.AppConfig) {
		ac.MDM.EndUserAuthentication.SSOProviderSettings = fleet.SSOProviderSettings{
			EntityID:    "mdm.test.com",
			IDPName:     "SimpleSAML",
			MetadataURL: idpSrv.URL,
		}
	}

	cases := []struct {
		name      string
		wantErr   string
		appConfig func() fleet.AppConfig
	}{
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
			name:    "no alternative browser host",
			wantErr: "metadata",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				idp(&ac)
				return ac
			},
		},
		{
			name:    "alternative browser host equals the server host",
			wantErr: "metadata",
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
			name:    "alternative browser host differs only by port",
			wantErr: "metadata",
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
			name:    "apple server url equals the server url",
			wantErr: "metadata",
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

			ctx := hostctx.NewContext(t.Context(), &fleet.Host{ID: 1, UUID: "host-uuid-1"})
			_, err := svc.InitiateDeviceSSO(ctx, "/device/some-token")
			require.Error(t, err)
			require.ErrorContains(t, err, c.wantErr)

			// Guard refusals are client errors; reaching the metadata fetch is not.
			var badReq *fleet.BadRequestError
			if c.wantErr != "metadata" {
				require.ErrorAs(t, err, &badReq)
			}
		})
	}
}

func TestInitiateDeviceSSOMissingHostInContext(t *testing.T) {
	svc, _, _ := newDeviceSSOTestService(t, new(mock.Store), time.Hour)

	_, err := svc.InitiateDeviceSSO(t.Context(), "/device/some-token")
	require.Error(t, err)
}
