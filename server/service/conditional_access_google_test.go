package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestModifyAppConfigGoogleConditionalAccessRequiresPrivateKey(t *testing.T) {
	ds := new(mock.Store)
	cfg := config.TestConfig()
	cfg.Server.PrivateKey = ""
	var factoryCalled bool
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{
		License: &fleet.LicenseInfo{Tier: fleet.TierPremium},
		ConditionalAccessGoogleClientFactory: func(context.Context, []byte, string, string) (ConditionalAccessGoogleClient, error) {
			factoryCalled = true
			return noopConditionalAccessGoogleClient{}, nil
		},
	})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			OrgInfo:        fleet.OrgInfo{OrgName: "Test"},
			ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"},
		}, nil
	}
	ds.ConditionalAccessGoogleGetFunc = func(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
		return nil, &notFoundError{}
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return nil, &notFoundError{}
	}

	_, err := svc.ModifyAppConfig(ctx, []byte(`{"integrations": {"google_conditional_access": [{
		"customer_id": "C01234abc",
		"admin_email": "admin@example.com",
		"api_key_json": {"client_email": "sa@example.com", "private_key": "key"}
	}]}}`), fleet.ApplySpecOptions{})
	require.ErrorContains(t, err, "Missing required private key")
	require.False(t, factoryCalled, "Google must not be called without a private key")
	require.False(t, ds.SaveAppConfigFuncInvoked)
	require.False(t, ds.InsertOrReplaceMDMConfigAssetFuncInvoked)
}

func TestModifyAppConfigGoogleConditionalAccessRowFailureKeepsPreviousIntegration(t *testing.T) {
	ds := new(mock.Store)
	proxy := &mockedConditionalAccessGoogle{}
	svc, ctx := newTestServiceWithConfig(t, ds, config.TestConfig(), nil, nil, &TestServerOpts{
		License:                              &fleet.LicenseInfo{Tier: fleet.TierPremium},
		ConditionalAccessGoogleProxy:         proxy,
		ConditionalAccessGoogleClientFactory: proxy.clientFactory,
	})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)}})

	oldKey := []byte(`{"client_email":"old@example.com","private_key":"old"}`)
	storedKey := oldKey
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		ac := &fleet.AppConfig{
			OrgInfo:        fleet.OrgInfo{OrgName: "Test"},
			ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"},
		}
		ac.Integrations.GoogleConditionalAccess = []*fleet.GoogleConditionalAccessIntegration{{CustomerID: "C01234abc", AdminEmail: "admin@example.com"}}
		return ac, nil
	}
	ds.ConditionalAccessGoogleGetFunc = func(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
		return &fleet.ConditionalAccessGoogleIntegration{CustomerID: "C01234abc", ProxyServerSecret: "old-secret", SetupDone: true}, nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetGoogleConditionalAccessServiceAccountKey: {Value: storedKey},
		}, nil
	}
	ds.InsertOrReplaceMDMConfigAssetFunc = func(ctx context.Context, asset fleet.MDMConfigAsset) error {
		storedKey = asset.Value
		return nil
	}
	ds.ConditionalAccessGoogleCreateIntegrationFunc = func(ctx context.Context, customerID, secret, clientID string) error {
		return errors.New("db is down")
	}

	_, err := svc.ModifyAppConfig(ctx, []byte(`{"integrations": {"google_conditional_access": [{
		"customer_id": "C05678def",
		"admin_email": "admin@example.com",
		"api_key_json": {"client_email": "new@example.com", "private_key": "new"}
	}]}}`), fleet.ApplySpecOptions{})
	require.ErrorContains(t, err, "db is down")

	// The previous row keeps the key it was verified with, and only the new registration is rolled back.
	require.Equal(t, string(oldKey), string(storedKey))
	require.Equal(t, []string{"C05678def"}, proxy.creates)
	require.Equal(t, [][2]string{{"C05678def", "secret-1"}}, proxy.deletes)
	require.False(t, ds.SaveAppConfigFuncInvoked)
}
