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
	appConfig := &fleet.AppConfig{
		OrgInfo:        fleet.OrgInfo{OrgName: "Test"},
		ServerSettings: fleet.ServerSettings{ServerURL: "https://fleet.example.com"},
	}
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
	// Every app config update reads these.
	ds.ListVPPTokensFunc = func(context.Context) ([]*fleet.VPPTokenDB, error) { return nil, nil }
	ds.ListABMTokensFunc = func(context.Context) ([]*fleet.ABMToken, error) { return nil, nil }

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

	// The rest of an app config update runs on the canceled request, so these check the
	// integration's state rather than the update's result.
	const connectConfig = `{"integrations": {"google_conditional_access": {
		"customer_id": "C02cajzvg", "admin_email": "admin@example.com", "api_key_json": ` + googleCATestKey + `}}}`

	t.Run("a config update saves the registration", func(t *testing.T) {
		g := newGoogleCADisconnectTest(t, "")
		_, _ = g.svc.ModifyAppConfig(g.ctx, []byte(connectConfig), fleet.ApplySpecOptions{})
		require.Equal(t, connected("C02cajzvg"), g.row)
		_, registers, deregisters, _ := g.relay.calls()
		require.Equal(t, []string{"C02cajzvg"}, registers)
		require.Empty(t, deregisters)
		require.Equal(t, []string{fleet.ActivityTypeAddedConditionalAccessIntegrationGoogle{}.ActivityName()}, g.activities)
	})

	t.Run("a config update drops a registration that can't be saved", func(t *testing.T) {
		g := newGoogleCADisconnectTest(t, "")
		g.saveSecretErr = errors.New("save failed")
		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(connectConfig), fleet.ApplySpecOptions{})
		require.ErrorContains(t, err, "save failed")
		_, _, deregisters, _ := g.relay.calls()
		require.Equal(t, []string{"C02cajzvg"}, deregisters)
		require.False(t, g.row.Registered())
	})

	t.Run("a config update doesn't keep the secret of a customer it left", func(t *testing.T) {
		g := newGoogleCADisconnectTest(t, "C02cajzvg")
		_, _ = g.svc.ModifyAppConfig(g.ctx, []byte(`{"integrations": {"google_conditional_access": {"customer_id": "C03other"}}}`),
			fleet.ApplySpecOptions{})
		_, _, deregisters, _ := g.relay.calls()
		require.Equal(t, []string{"C02cajzvg"}, deregisters)
		require.False(t, g.row.Registered())
	})
}

type fakeGoogleCARelay struct {
	registers   []string // "customerID serverURL"
	deregisters []string // "customerID serverURL"
	// deregisterErr fails Deregister without recording the call.
	deregisterErr error
}

func (f *fakeGoogleCARelay) Register(_ context.Context, serverURL, customerID string) (*conditional_access_google.Registration, error) {
	f.registers = append(f.registers, customerID+" "+serverURL)
	return &conditional_access_google.Registration{FleetServerSecret: "secret-" + customerID}, nil
}

func (f *fakeGoogleCARelay) Deregister(_ context.Context, serverURL, customerID, _ string) error {
	if f.deregisterErr != nil {
		return f.deregisterErr
	}
	f.deregisters = append(f.deregisters, customerID+" "+serverURL)
	return nil
}

type fakeGoogleCACloudIdentity struct {
	unmanaged     *[]string
	delegationErr error
}

func (f fakeGoogleCACloudIdentity) CheckDelegation(context.Context) error { return f.delegationErr }

func (f fakeGoogleCACloudIdentity) SetClientState(_ context.Context, name string, state fleet.GoogleConditionalAccessClientState) error {
	if state.Managed != nil && !*state.Managed {
		*f.unmanaged = append(*f.unmanaged, name)
	}
	return nil
}

type googleCAAppConfigTest struct {
	svc       fleet.Service
	ctx       context.Context
	ds        *mock.Store
	relay     *fakeGoogleCARelay
	unmanaged []string
	rows      []fleet.ConditionalAccessGoogleIntegration // saved rows, in order
	config    *fleet.AppConfig                           // the stored app config
	saveErr   error                                      // returned by SaveAppConfig
	// storedKey is the stored service account key, nil when none is stored.
	storedKey []byte
	// checkedKeys are the keys the delegation check ran with, in order.
	checkedKeys   []string
	delegationErr error
}

const googleCATestKey = `{"client_email":"sa@example.com","private_key":"pk"}`

// newGoogleCAAppConfigTest sets up a Premium service whose integration is connected to
// C02cajzvg from https://old.example.com.
func newGoogleCAAppConfigTest(t *testing.T) *googleCAAppConfigTest {
	g := &googleCAAppConfigTest{ds: new(mock.Store), relay: &fakeGoogleCARelay{}, storedKey: []byte(googleCATestKey)}
	ds := g.ds

	dsAppConfig := &fleet.AppConfig{
		OrgInfo:        fleet.OrgInfo{OrgName: "Test"},
		ServerSettings: fleet.ServerSettings{ServerURL: "https://old.example.com"},
	}
	dsAppConfig.Integrations.GoogleConditionalAccess = &fleet.GoogleConditionalAccessIntegration{
		CustomerID: "C02cajzvg", AdminEmail: "admin@example.com",
	}
	g.config = dsAppConfig
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return dsAppConfig.Copy(), nil }
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		if g.saveErr != nil {
			return g.saveErr
		}
		*dsAppConfig = *conf
		return nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) { return []*fleet.VPPTokenDB{}, nil }
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) { return []*fleet.ABMToken{}, nil }
	// A server URL change queues a DEP profile sync.
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) { return job, nil }

	ds.GetGoogleConditionalAccessIntegrationFunc = func(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
		return &fleet.ConditionalAccessGoogleIntegration{
			ProxyServerSecret: "secret-" + "C02cajzvg", State: fleet.ConditionalAccessGoogleStateConnected,
		}, nil
	}
	ds.SaveGoogleConditionalAccessIntegrationFunc = func(ctx context.Context, row *fleet.ConditionalAccessGoogleIntegration) error {
		g.rows = append(g.rows, *row)
		return nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		if g.storedKey == nil {
			return nil, &notFoundError{}
		}
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetGoogleConditionalAccessServiceAccountKey: {Value: g.storedKey},
		}, nil
	}
	ds.ReplaceMDMConfigAssetsFunc = func(ctx context.Context, assets []fleet.MDMConfigAsset, _ sqlx.ExtContext) error {
		for _, a := range assets {
			if a.Name == fleet.MDMAssetGoogleConditionalAccessServiceAccountKey {
				g.storedKey = a.Value
			}
		}
		return nil
	}
	ds.DeleteGoogleConditionalAccessIntegrationFunc = func(ctx context.Context) error { return nil }
	ds.DeleteMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName) error {
		g.storedKey = nil
		return nil
	}

	opts := &TestServerOpts{
		License: &fleet.LicenseInfo{Tier: fleet.TierPremium},
		ConditionalAccessGoogleClient: &GoogleConditionalAccessClient{
			Relay: g.relay,
			CloudIdentity: func(_ context.Context, creds conditional_access_google.Credentials) (conditional_access_google.CloudIdentity, error) {
				g.checkedKeys = append(g.checkedKeys, string(creds.ServiceAccountKey))
				return fakeGoogleCACloudIdentity{unmanaged: &g.unmanaged, delegationErr: g.delegationErr}, nil
			},
		},
	}
	cfg := config.TestConfig()
	cfg.Server.PrivateKey = "72414F4A688151F75D032F5CDA095FC4"
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, opts)
	opts.ActivityMock.NewActivityFunc = func(context.Context, *activity_api.User, activity_api.ActivityDetails) error { return nil }
	g.svc = svc
	g.ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	return g
}

func TestModifyAppConfigGoogleConditionalAccessUnrelatedUpdate(t *testing.T) {
	g := newGoogleCAAppConfigTest(t)

	_, err := g.svc.ModifyAppConfig(g.ctx, []byte(`{"org_info": {"org_name": "Test2"}}`), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	require.False(t, g.ds.GetGoogleConditionalAccessIntegrationFuncInvoked)
	require.False(t, g.ds.GetAllMDMConfigAssetsByNameFuncInvoked)
}

func TestModifyAppConfigGoogleConditionalAccessGitOps(t *testing.T) {
	gitops := fleet.ApplySpecOptions{Overwrite: true}

	t.Run("a null setting deregisters and turns it off", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(`{"integrations": {"google_conditional_access": null}}`), gitops)
		require.NoError(t, err)
		require.Equal(t, []string{"C02cajzvg https://old.example.com"}, g.relay.deregisters)
		require.Empty(t, g.unmanaged)
		require.Nil(t, g.storedKey)
		require.False(t, g.ds.DeleteGoogleConditionalAccessIntegrationFuncInvoked)
		require.Equal(t, []fleet.ConditionalAccessGoogleIntegration{{State: fleet.ConditionalAccessGoogleStateDeleted}}, g.rows)
		require.Nil(t, g.config.Integrations.GoogleConditionalAccess)
	})

	t.Run("a missing setting keeps it, as older fleetctl versions never send it", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(`{"org_info": {"org_name": "Test2"}}`), gitops)
		require.NoError(t, err)
		require.Empty(t, g.relay.deregisters)
		require.False(t, g.ds.DeleteGoogleConditionalAccessIntegrationFuncInvoked)
		require.NotNil(t, g.config.Integrations.GoogleConditionalAccess)
	})

	t.Run("a null setting outside GitOps keeps it", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(`{"integrations": {"google_conditional_access": null}}`), fleet.ApplySpecOptions{})
		require.NoError(t, err)
		require.Empty(t, g.relay.deregisters)
		require.NotNil(t, g.config.Integrations.GoogleConditionalAccess)
	})
}

func TestModifyAppConfigGoogleConditionalAccessRemoval(t *testing.T) {
	const turnOff = `{"integrations": {"google_conditional_access": null}}`
	gitops := fleet.ApplySpecOptions{Overwrite: true}

	t.Run("failed deregistration changes nothing", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)
		g.relay.deregisterErr = &conditional_access_google.RelayStatusError{Code: 503}

		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(turnOff), gitops)
		var gwErr *fleet.GatewayError
		require.ErrorAs(t, err, &gwErr)
		require.False(t, g.ds.DeleteMDMConfigAssetsByNameFuncInvoked)
		require.Empty(t, g.rows)
		require.False(t, g.ds.SaveAppConfigFuncInvoked)
	})

	t.Run("a missing registration on fleetdm.com doesn't block it", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)
		g.relay.deregisterErr = &notFoundError{}

		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(turnOff), gitops)
		require.NoError(t, err)
		require.Equal(t, []fleet.ConditionalAccessGoogleIntegration{{State: fleet.ConditionalAccessGoogleStateDeleted}}, g.rows)
		require.Nil(t, g.storedKey)
		require.Nil(t, g.config.Integrations.GoogleConditionalAccess)
	})

	t.Run("failed config save keeps the key and the setting for a retry", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)
		g.saveErr = errors.New("save failed")

		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(turnOff), gitops)
		require.ErrorContains(t, err, "save failed")
		// The registration is gone, so the row can't keep its secret.
		require.Equal(t, []string{"C02cajzvg https://old.example.com"}, g.relay.deregisters)
		require.Equal(t, []fleet.ConditionalAccessGoogleIntegration{{State: fleet.ConditionalAccessGoogleStateDeleted}}, g.rows)
		require.False(t, g.ds.DeleteMDMConfigAssetsByNameFuncInvoked)
		require.NotNil(t, g.config.Integrations.GoogleConditionalAccess)
	})

	t.Run("customer change leaves with the old URL and registers with the new one", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(`{
			"server_settings": {"server_url": "https://new.example.com"},
			"integrations": {"google_conditional_access": {"customer_id": "C03other"}}
		}`), fleet.ApplySpecOptions{})
		require.NoError(t, err)
		require.Equal(t, []string{"C02cajzvg https://old.example.com"}, g.relay.deregisters)
		require.Equal(t, []string{"C03other https://new.example.com"}, g.relay.registers)
		require.Empty(t, g.unmanaged)
	})
}

func TestModifyAppConfigGoogleConditionalAccessKey(t *testing.T) {
	gitops := fleet.ApplySpecOptions{Overwrite: true}
	const newKey = `{"client_email":"sa2@example.com","private_key":"pk2"}`
	withKey := func(key string) []byte {
		return []byte(`{"integrations": {"google_conditional_access": {
			"customer_id": "C02cajzvg", "admin_email": "admin@example.com", "api_key_json": ` + key + `}}}`)
	}

	t.Run("a new key replaces the stored key and reconnects", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, withKey(newKey), gitops)
		require.NoError(t, err)
		require.Equal(t, []string{newKey}, g.checkedKeys)
		require.JSONEq(t, newKey, string(g.storedKey))
		require.Len(t, g.rows, 1)
		require.Equal(t, fleet.ConditionalAccessGoogleStateConnected, g.rows[0].State)
		require.Empty(t, g.relay.registers)
		require.Empty(t, g.relay.deregisters)
		require.Equal(t, &fleet.GoogleConditionalAccessIntegration{CustomerID: "C02cajzvg", AdminEmail: "admin@example.com"},
			g.config.Integrations.GoogleConditionalAccess)
	})

	t.Run("the stored key, formatted differently, is a no-op", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, withKey(`{"private_key": "pk", "client_email": "sa@example.com"}`), gitops)
		require.NoError(t, err)
		require.Empty(t, g.checkedKeys)
		require.False(t, g.ds.ReplaceMDMConfigAssetsFuncInvoked)
		require.Empty(t, g.rows)
	})

	t.Run("the masked key keeps the stored key", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, withKey(`"********"`), gitops)
		require.NoError(t, err)
		require.Empty(t, g.checkedKeys)
		require.False(t, g.ds.ReplaceMDMConfigAssetsFuncInvoked)
		require.Empty(t, g.rows)
		require.JSONEq(t, googleCATestKey, string(g.storedKey))
	})

	t.Run("a dry run checks the new key and stores nothing", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, withKey(newKey), fleet.ApplySpecOptions{Overwrite: true, DryRun: true})
		require.NoError(t, err)
		require.Equal(t, []string{newKey}, g.checkedKeys)
		require.False(t, g.ds.ReplaceMDMConfigAssetsFuncInvoked)
		require.Empty(t, g.rows)
	})

	t.Run("a failed delegation check with a new key stores nothing", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)
		g.delegationErr = errors.New("unauthorized_client")

		_, err := g.svc.ModifyAppConfig(g.ctx, withKey(newKey), gitops)
		require.ErrorContains(t, err, "unauthorized_client")
		var invalid *fleet.InvalidArgumentError
		require.ErrorAs(t, err, &invalid)
		require.False(t, g.ds.ReplaceMDMConfigAssetsFuncInvoked)
		require.Empty(t, g.rows)
		require.False(t, g.ds.SaveAppConfigFuncInvoked)
	})

	t.Run("a key missing its private key is rejected before any Google call", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)

		_, err := g.svc.ModifyAppConfig(g.ctx, withKey(`{"client_email": "sa2@example.com"}`), gitops)
		require.ErrorContains(t, err, "api_key_json must include client_email and private_key")
		require.Empty(t, g.checkedKeys)
	})

	t.Run("a new key without the server private key is rejected before any Google call", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)
		testSetEmptyPrivateKey = true
		t.Cleanup(func() { testSetEmptyPrivateKey = false })

		_, err := g.svc.ModifyAppConfig(g.ctx, withKey(newKey), gitops)
		require.ErrorContains(t, err, "Missing required private key")
		require.Empty(t, g.checkedKeys)
	})

	t.Run("turning it on with a key connects", func(t *testing.T) {
		g := newGoogleCAAppConfigTest(t)
		g.config.Integrations.GoogleConditionalAccess = nil
		g.storedKey = nil
		g.ds.GetGoogleConditionalAccessIntegrationFunc = func(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
			return nil, &notFoundError{}
		}

		_, err := g.svc.ModifyAppConfig(g.ctx, withKey(newKey), gitops)
		require.NoError(t, err)
		require.Equal(t, []string{newKey}, g.checkedKeys)
		require.JSONEq(t, newKey, string(g.storedKey))
		require.Equal(t, []string{"C02cajzvg https://old.example.com"}, g.relay.registers)
		require.Equal(t, fleet.ConditionalAccessGoogleStateConnected, g.rows[len(g.rows)-1].State)
		require.Empty(t, g.config.Integrations.GoogleConditionalAccess.APIKeyJSON)
	})
}

func TestModifyAppConfigGoogleConditionalAccessKeyRequired(t *testing.T) {
	const settings = `{"integrations": {"google_conditional_access": {"customer_id": "C02cajzvg", "admin_email": "admin@example.com"}}}`
	for _, opts := range []fleet.ApplySpecOptions{{}, {Overwrite: true}, {Overwrite: true, DryRun: true}} {
		g := newGoogleCAAppConfigTest(t)
		g.config.Integrations.GoogleConditionalAccess = nil
		g.storedKey = nil
		g.ds.GetGoogleConditionalAccessIntegrationFunc = func(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
			return &fleet.ConditionalAccessGoogleIntegration{State: fleet.ConditionalAccessGoogleStateDeleted}, nil
		}

		_, err := g.svc.ModifyAppConfig(g.ctx, []byte(settings), opts)
		var invalid *fleet.InvalidArgumentError
		require.ErrorAs(t, err, &invalid)
		require.Equal(t, []map[string]string{{
			"name":   "integrations.google_conditional_access.api_key_json",
			"reason": "api_key_json is required because no service account key is stored",
		}}, invalid.Invalid())
		require.Empty(t, g.checkedKeys)
		require.Empty(t, g.rows)
		require.False(t, g.ds.SaveAppConfigFuncInvoked)
	}
}
