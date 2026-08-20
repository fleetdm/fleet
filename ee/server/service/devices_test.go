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
	host := &fleet.Host{ID: 1, UUID: "host-uuid-1"}

	ssoSettings := fleet.SSOProviderSettings{
		EntityID:    "mdm.test.com",
		IDPName:     "SimpleSAML",
		MetadataURL: "https://idp.example.com/metadata",
	}

	cases := []struct {
		name      string
		appConfig func() fleet.AppConfig
		wantErr   string
	}{
		{
			name:      "sso disabled",
			appConfig: func() fleet.AppConfig { return fleet.AppConfig{} },
			wantErr:   "is not enabled",
		},
		{
			name: "sso enabled but no IdP configured",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				return ac
			},
			wantErr: "no IdP is configured",
		},
		{
			name: "alternative browser host differs from the ACS host",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.FleetDesktop.AlternativeBrowserHost = "proxy.example.com"
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				ac.MDM.EndUserAuthentication.SSOProviderSettings = ssoSettings
				return ac
			},
			wantErr: "same host",
		},
		{
			name: "custom Apple MDM URL differs from the server URL",
			appConfig: func() fleet.AppConfig {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				ac.MDM.AppleServerURL = "https://mdm.example.com"
				ac.MDM.EndUserAuthentication.SSOProviderSettings = ssoSettings
				return ac
			},
			wantErr: "same host",
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

			ctx := hostctx.NewContext(t.Context(), host)
			_, err := svc.InitiateDeviceSSO(ctx, "/device/some-token")
			require.Error(t, err)
			require.Contains(t, err.Error(), c.wantErr)

			var badReq *fleet.BadRequestError
			require.ErrorAs(t, err, &badReq)
		})
	}
}

func TestInitiateDeviceSSOAllowsMatchingHosts(t *testing.T) {
	idpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(idpSrv.Close)

	for _, c := range []struct {
		name                   string
		alternativeBrowserHost string
		appleServerURL         string
	}{
		{name: "no alternative browser host"},
		{name: "alternative browser host equals the server host", alternativeBrowserHost: "fleet.example.com"},
		{name: "alternative browser host differs only by port", alternativeBrowserHost: "fleet.example.com:8443"},
		{name: "apple server url equals the server url", appleServerURL: "https://fleet.example.com"},
	} {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				var ac fleet.AppConfig
				ac.FleetDesktop.SSOEnabled = true
				ac.FleetDesktop.AlternativeBrowserHost = c.alternativeBrowserHost
				ac.ServerSettings.ServerURL = "https://fleet.example.com"
				ac.MDM.AppleServerURL = c.appleServerURL
				ac.MDM.EndUserAuthentication.SSOProviderSettings = fleet.SSOProviderSettings{
					EntityID:    "mdm.test.com",
					IDPName:     "SimpleSAML",
					MetadataURL: idpSrv.URL,
				}
				return &ac, nil
			}

			svc, _, _ := newDeviceSSOTestService(t, ds, time.Hour)

			ctx := hostctx.NewContext(t.Context(), &fleet.Host{ID: 1, UUID: "host-uuid-1"})
			_, err := svc.InitiateDeviceSSO(ctx, "/device/some-token")

			require.Error(t, err)
			require.ErrorContains(t, err, "metadata")
		})
	}
}

func TestInitiateDeviceSSOMissingHostInContext(t *testing.T) {
	svc, _, _ := newDeviceSSOTestService(t, new(mock.Store), time.Hour)

	_, err := svc.InitiateDeviceSSO(t.Context(), "/device/some-token")
	require.Error(t, err)
}
