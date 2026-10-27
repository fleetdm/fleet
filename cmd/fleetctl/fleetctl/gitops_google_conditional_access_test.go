package fleetctl

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google_proxy"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// fakeGoogleConditionalAccess stands in for fleetdm.com and Google, counting calls.
type fakeGoogleConditionalAccess struct {
	mu                       sync.Mutex
	checks, creates, deletes int
}

func (f *fakeGoogleConditionalAccess) Create(context.Context, string) (*conditional_access_google_proxy.CreateResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creates++
	return &conditional_access_google_proxy.CreateResponse{FleetServerSecret: "secret", OAuthClientID: "client"}, nil
}

func (f *fakeGoogleConditionalAccess) Delete(context.Context, string, string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletes++
	return nil
}

func (f *fakeGoogleConditionalAccess) CheckDelegation(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checks++
	return nil
}

func (f *fakeGoogleConditionalAccess) counts() [3]int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return [3]int{f.checks, f.creates, f.deletes}
}

func TestGitOpsGoogleConditionalAccess(t *testing.T) {
	// Cannot run t.Parallel() because it sets environment variables.
	fake := &fakeGoogleConditionalAccess{}
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := testing_utils.RunServerWithMockedDS(t, &service.TestServerOpts{
		License:                      license,
		KeyValueStore:                testing_utils.NewMemKeyValueStore(),
		ConditionalAccessGoogleProxy: fake,
		ConditionalAccessGoogleClientFactory: func(context.Context, []byte, string, string) (service.ConditionalAccessGoogleClient, error) {
			return fake, nil
		},
	})
	setupEmptyGitOpsMocks(ds)

	// Keep the saved config, the integration row, and the key between runs.
	var (
		mu        sync.Mutex
		appConfig = &fleet.AppConfig{}
		row       *fleet.ConditionalAccessGoogleIntegration
		key       []byte
	)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		mu.Lock()
		defer mu.Unlock()
		return appConfig.Copy(), nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		mu.Lock()
		defer mu.Unlock()
		appConfig = config.Copy()
		return nil
	}
	ds.ConditionalAccessGoogleGetFunc = func(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
		mu.Lock()
		defer mu.Unlock()
		if row == nil {
			return nil, &notFoundError{}
		}
		return row, nil
	}
	ds.ConditionalAccessGoogleCreateIntegrationFunc = func(ctx context.Context, customerID, secret, clientID string) error {
		mu.Lock()
		defer mu.Unlock()
		row = &fleet.ConditionalAccessGoogleIntegration{CustomerID: customerID, ProxyServerSecret: secret, OAuthClientID: clientID, SetupDone: true}
		return nil
	}
	ds.ConditionalAccessGoogleDeleteFunc = func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		row = nil
		return nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		mu.Lock()
		defer mu.Unlock()
		if key == nil {
			return nil, &notFoundError{}
		}
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetGoogleConditionalAccessServiceAccountKey: {Name: fleet.MDMAssetGoogleConditionalAccessServiceAccountKey, Value: key},
		}, nil
	}
	ds.InsertOrReplaceMDMConfigAssetFunc = func(ctx context.Context, asset fleet.MDMConfigAsset) error {
		mu.Lock()
		defer mu.Unlock()
		key = asset.Value
		return nil
	}
	ds.DeleteMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName) error {
		mu.Lock()
		defer mu.Unlock()
		key = nil
		return nil
	}

	t.Setenv("FLEET_SERVER_URL", fleetServerURL)
	t.Setenv("FLEET_GOOGLE_CA_KEY", `{"client_email": "sa@example.iam.gserviceaccount.com", "private_key": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n"}`)
	const base = `
controls:
queries:
policies:
labels:
agent_options:
software:
org_settings:
  server_settings:
    server_url: $FLEET_SERVER_URL
  org_info:
    contact_url: https://example.com/contact
    org_name: GitOps Google CA Test
  secrets:
`
	writeConfig := func(integrations string) string {
		f, err := os.CreateTemp(t.TempDir(), "*.yml")
		require.NoError(t, err)
		_, err = f.WriteString(base + integrations)
		require.NoError(t, err)
		return f.Name()
	}
	connected := writeConfig(`  integrations:
    google_conditional_access:
      - customer_id: C01234abc
        admin_email: admin@example.com
        api_key_json: $FLEET_GOOGLE_CA_KEY
`)

	// Connects on a fresh server; --dry-run checks the delegation but registers nothing.
	runAppForTest(t, []string{"gitops", "-f", connected, "--dry-run"})
	require.Equal(t, [3]int{1, 0, 0}, fake.counts())
	require.Nil(t, row)
	runAppForTest(t, []string{"gitops", "-f", connected})
	require.Equal(t, [3]int{2, 1, 0}, fake.counts())
	require.NotNil(t, row)
	require.JSONEq(t, `{"client_email": "sa@example.iam.gserviceaccount.com", "private_key": "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----"}`, string(key))
	require.Len(t, appConfig.Integrations.GoogleConditionalAccess, 1)
	require.True(t, appConfig.Integrations.GoogleConditionalAccess[0].ApiKey.IsEmpty(), "the key is never saved in the config")

	// A rerun is a no-op.
	runAppForTest(t, []string{"gitops", "-f", connected})
	require.Equal(t, [3]int{2, 1, 0}, fake.counts())

	// Removing it from YAML, or leaving only empty fields (unset variables), disconnects.
	runAppForTest(t, []string{"gitops", "-f", writeConfig("  integrations:\n")})
	require.Equal(t, [3]int{2, 1, 1}, fake.counts())
	require.Nil(t, row)
	require.Nil(t, key)
	require.Empty(t, appConfig.Integrations.GoogleConditionalAccess)

	runAppForTest(t, []string{"gitops", "-f", connected})
	require.Equal(t, [3]int{3, 2, 1}, fake.counts())
	runAppForTest(t, []string{"gitops", "-f", writeConfig(`  integrations:
    google_conditional_access:
      - customer_id: ""
        admin_email: ""
        api_key_json: ""
`)})
	require.Equal(t, [3]int{3, 2, 2}, fake.counts())
	require.Nil(t, row)
}
