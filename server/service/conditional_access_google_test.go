package service

import (
	"context"
	"errors"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// disconnectingGoogleCARelay cancels the request as fleetdm.com answers, as a client
// that disconnects mid-request does. Like the HTTP relay, it fails on a canceled context.
type disconnectingGoogleCARelay struct {
	*mockedGoogleConditionalAccessClient
	disconnect context.CancelFunc
}

func (r *disconnectingGoogleCARelay) Register(ctx context.Context, serverURL, customerID string) (*conditional_access_google.Registration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	defer r.disconnect()
	return r.mockedGoogleConditionalAccessClient.Register(ctx, serverURL, customerID)
}

func (r *disconnectingGoogleCARelay) Deregister(ctx context.Context, serverURL, customerID, secret string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	defer r.disconnect()
	return r.mockedGoogleConditionalAccessClient.Deregister(ctx, serverURL, customerID, secret)
}

// googleCADisconnectTest is a service whose client disconnects as fleetdm.com answers
// and whose datastore, like MySQL, fails writes on a canceled context.
type googleCADisconnectTest struct {
	svc   fleet.Service
	ctx   context.Context
	relay *disconnectingGoogleCARelay

	settings   *fleet.GoogleConditionalAccessIntegration
	row        *fleet.ConditionalAccessGoogleIntegration
	key        []byte
	activities []string
	// saveSecretErr fails saving a row that holds a secret.
	saveSecretErr error
}

// newGoogleCADisconnectTest starts from a connected integration when customerID is set.
func newGoogleCADisconnectTest(t *testing.T, customerID string) *googleCADisconnectTest {
	g := &googleCADisconnectTest{}
	appConfig := &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: "https://fleet.example.com"}}
	if customerID != "" {
		appConfig.Integrations.GoogleConditionalAccess = &fleet.GoogleConditionalAccessIntegration{
			CustomerID: customerID, AdminEmail: "admin@example.com",
		}
		g.row = &fleet.ConditionalAccessGoogleIntegration{
			ProxyServerSecret: "secret-" + customerID, State: fleet.ConditionalAccessGoogleStateConnected,
		}
		g.key = []byte(`{"client_email":"sa@example.com"}`)
	}
	g.settings = appConfig.Integrations.GoogleConditionalAccess

	ds := new(mock.Store)
	ds.AppConfigFunc = func(context.Context) (*fleet.AppConfig, error) { return appConfig.Copy(), nil }
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		*appConfig = *conf
		g.settings = conf.Integrations.GoogleConditionalAccess
		return nil
	}
	ds.GetGoogleConditionalAccessIntegrationFunc = func(context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
		if g.row == nil {
			return nil, &notFoundError{}
		}
		row := *g.row
		return &row, nil
	}
	ds.SaveGoogleConditionalAccessIntegrationFunc = func(ctx context.Context, row *fleet.ConditionalAccessGoogleIntegration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if g.saveSecretErr != nil && row.Registered() {
			return g.saveSecretErr
		}
		saved := *row
		g.row = &saved
		return nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(context.Context, []fleet.MDMAssetName, sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		if g.key == nil {
			return nil, &notFoundError{}
		}
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetGoogleConditionalAccessServiceAccountKey: {Value: g.key},
		}, nil
	}
	ds.GetAllMDMConfigAssetsHashesFunc = func(context.Context, []fleet.MDMAssetName) (map[fleet.MDMAssetName]string, error) {
		if g.key == nil {
			return nil, &notFoundError{}
		}
		return map[fleet.MDMAssetName]string{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey: "hash"}, nil
	}
	ds.ReplaceMDMConfigAssetsFunc = func(ctx context.Context, assets []fleet.MDMConfigAsset, _ sqlx.ExtContext) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		g.key = assets[0].Value
		return nil
	}
	ds.DeleteMDMConfigAssetsByNameFunc = func(ctx context.Context, _ []fleet.MDMAssetName) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		g.key = nil
		return nil
	}

	g.relay = &disconnectingGoogleCARelay{mockedGoogleConditionalAccessClient: &mockedGoogleConditionalAccessClient{}}
	opts := &TestServerOpts{
		License: &fleet.LicenseInfo{Tier: fleet.TierPremium},
		ConditionalAccessGoogleClient: &GoogleConditionalAccessClient{
			Relay: g.relay, CloudIdentity: g.relay.cloudIdentity,
		},
	}
	cfg := config.TestConfig()
	cfg.Server.PrivateKey = "test-private-key-not-used-by-mock"
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, opts)
	opts.ActivityMock.NewActivityFunc = func(ctx context.Context, _ *activity_api.User, act activity_api.ActivityDetails) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		g.activities = append(g.activities, act.ActivityName())
		return nil
	}
	g.svc = svc
	g.ctx, g.relay.disconnect = context.WithCancel(viewer.NewContext(ctx, viewer.Viewer{User: test.UserAdmin}))
	t.Cleanup(g.relay.disconnect)
	return g
}

func TestGoogleConditionalAccessClientDisconnects(t *testing.T) {
	const key = `{"client_email":"sa2@example.com"}`
	connected := func(customerID string) *fleet.ConditionalAccessGoogleIntegration {
		return &fleet.ConditionalAccessGoogleIntegration{
			ProxyServerSecret: "secret-" + customerID, State: fleet.ConditionalAccessGoogleStateConnected,
		}
	}

	t.Run("connecting saves the registration", func(t *testing.T) {
		g := newGoogleCADisconnectTest(t, "")
		require.NoError(t, g.svc.ConnectGoogleConditionalAccess(g.ctx, "C02cajzvg", "admin@example.com", []byte(key)))
		require.Equal(t, connected("C02cajzvg"), g.row)
		_, registers, deregisters, _ := g.relay.calls()
		require.Equal(t, []string{"C02cajzvg"}, registers)
		require.Empty(t, deregisters)
		require.Equal(t, []string{fleet.ActivityTypeAddedConditionalAccessIntegrationGoogle{}.ActivityName()}, g.activities)
	})

	t.Run("a registration that can't be saved is dropped", func(t *testing.T) {
		g := newGoogleCADisconnectTest(t, "")
		g.saveSecretErr = errors.New("save failed")
		err := g.svc.ConnectGoogleConditionalAccess(g.ctx, "C02cajzvg", "admin@example.com", []byte(key))
		require.ErrorContains(t, err, "save failed")
		_, _, deregisters, _ := g.relay.calls()
		require.Equal(t, []string{"C02cajzvg"}, deregisters)
		require.Equal(t, []string{"secret-C02cajzvg"}, g.relay.deregisterSecrets())
		require.False(t, g.row.Registered())
		require.Empty(t, g.activities)
	})

	t.Run("changing the customer replaces the registration", func(t *testing.T) {
		g := newGoogleCADisconnectTest(t, "C02cajzvg")
		require.NoError(t, g.svc.UpdateGoogleConditionalAccess(g.ctx, new("C03other"), nil, nil))
		require.Equal(t, connected("C03other"), g.row)
		require.Equal(t, "C03other", g.settings.CustomerID)
		_, registers, deregisters, _ := g.relay.calls()
		require.Equal(t, []string{"C03other"}, registers)
		require.Equal(t, []string{"C02cajzvg"}, deregisters)
		require.Equal(t, []string{fleet.ActivityTypeEditedConditionalAccessIntegrationGoogle{}.ActivityName()}, g.activities)
	})

	t.Run("deleting removes the integration", func(t *testing.T) {
		g := newGoogleCADisconnectTest(t, "C02cajzvg")
		require.NoError(t, g.svc.DeleteGoogleConditionalAccess(g.ctx))
		require.Equal(t, &fleet.ConditionalAccessGoogleIntegration{State: fleet.ConditionalAccessGoogleStateDeleted}, g.row)
		require.Nil(t, g.key)
		require.Nil(t, g.settings)
		_, _, deregisters, _ := g.relay.calls()
		require.Equal(t, []string{"C02cajzvg"}, deregisters)
		require.Equal(t, []string{fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}.ActivityName()}, g.activities)
	})
}
