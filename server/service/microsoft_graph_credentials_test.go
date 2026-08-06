package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/microsoft/msgraph"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	graphTenantA = "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
	graphTenantB = "11111111-1111-1111-1111-111111111111"
	graphClientA = "7f6b1665-51f5-48de-a9b6-ac17539583fb"
)

// fakeGraphClient stands in for a real Graph client so config-write validation never reaches the network.
type fakeGraphClient struct {
	verifyErr error
}

func (f *fakeGraphClient) VerifyCredential(context.Context) error { return f.verifyErr }

func (f *fakeGraphClient) ListWindowsAutopilotDevices(context.Context) ([]fleet.WindowsAutopilotDevice, error) {
	return nil, nil
}

// countingGraphFactory returns a factory that records how many clients it built, so tests can assert that an
// unchanged credential is never re-verified.
func countingGraphFactory(verifyErr error) (msgraph.ClientFactory, *int) {
	calls := 0
	return func(cred *fleet.MicrosoftGraphCredential) (msgraph.Client, error) {
		calls++
		return &fakeGraphClient{verifyErr: verifyErr}, nil
	}, &calls
}

type graphCredsTestEnv struct {
	svc     fleet.Service
	ctx     context.Context
	ds      *mock.Store
	stored  map[string]*fleet.MicrosoftGraphCredential
	deleted []string
}

func setupGraphCredsTest(t *testing.T, tier string, privateKey string, verifyErr error) (*graphCredsTestEnv, *int) {
	t.Helper()

	factory, calls := countingGraphFactory(verifyErr)
	ds := new(mock.Store)
	adminRole := fleet.RoleAdmin
	admin := &fleet.User{GlobalRole: &adminRole}

	// Windows MDM must be on for the surrounding Entra config to validate, which needs a WSTEP cert/key pair.
	cfg := config.TestConfig()
	cfg.MDM.WindowsWSTEPIdentityCert = "testdata/server.pem"
	cfg.MDM.WindowsWSTEPIdentityKey = "testdata/server.key"
	cfg.Server.PrivateKey = privateKey

	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil,
		&TestServerOpts{License: &fleet.LicenseInfo{Tier: tier}, MicrosoftGraphClientFactory: factory})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

	env := &graphCredsTestEnv{svc: svc, ctx: ctx, ds: ds, stored: map[string]*fleet.MicrosoftGraphCredential{}}

	dsAppConfig := &fleet.AppConfig{
		OrgInfo:        fleet.OrgInfo{OrgName: "Test"},
		ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"},
		MDM:            fleet.MDM{WindowsEnabledAndConfigured: true},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return dsAppConfig, nil }
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		*dsAppConfig = *conf
		return nil
	}
	// ModifyAppConfig reads these while assembling the response; none are relevant here.
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) { return []*fleet.ABMToken{}, nil }
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) { return []*fleet.VPPTokenDB{}, nil }

	ds.ListMicrosoftGraphCredentialsFunc = func(ctx context.Context) ([]*fleet.MicrosoftGraphCredential, error) {
		out := make([]*fleet.MicrosoftGraphCredential, 0, len(env.stored))
		for _, c := range env.stored {
			out = append(out, c)
		}
		return out, nil
	}
	ds.ListMicrosoftGraphCredentialMetadataFunc = func(ctx context.Context) ([]*fleet.MicrosoftGraphCredential, error) {
		out := make([]*fleet.MicrosoftGraphCredential, 0, len(env.stored))
		for _, c := range env.stored {
			meta := *c
			meta.ClientSecret = "" // the metadata read never decrypts
			out = append(out, &meta)
		}
		return out, nil
	}
	ds.UpsertMicrosoftGraphCredentialFunc = func(ctx context.Context, cred *fleet.MicrosoftGraphCredential) error {
		copied := *cred
		env.stored[cred.TenantID] = &copied
		return nil
	}
	ds.DeleteMicrosoftGraphCredentialFunc = func(ctx context.Context, tenantID string) error {
		delete(env.stored, tenantID)
		env.deleted = append(env.deleted, tenantID)
		return nil
	}

	return env, calls
}

func TestModifyAppConfigMicrosoftGraphCredentials(t *testing.T) {
	validPayload := `{"mdm":{"microsoft_graph_credentials":[{"tenant_id":"` + graphTenantA +
		`","client_id":"` + graphClientA + `","client_secret":"secret-a"}]}}`

	t.Run("stores a credential on premium", func(t *testing.T) {
		env, calls := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(validPayload), fleet.ApplySpecOptions{})
		require.NoError(t, err)

		require.Len(t, env.stored, 1)
		stored := env.stored[graphTenantA]
		require.NotNil(t, stored)
		assert.Equal(t, graphClientA, stored.ClientID)
		assert.Equal(t, "secret-a", stored.ClientSecret)
		assert.Equal(t, 1, *calls, "a new credential is verified before it is stored")
	})

	t.Run("rejects on Fleet Free", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierFree, "test-private-key", nil)

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(validPayload), fleet.ApplySpecOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), ErrMissingLicense.Error())
		assert.Empty(t, env.stored)
	})

	t.Run("rejects a second credential", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		payload := `{"mdm":{"microsoft_graph_credentials":[` +
			`{"tenant_id":"` + graphTenantA + `","client_id":"` + graphClientA + `","client_secret":"a"},` +
			`{"tenant_id":"` + graphTenantB + `","client_id":"` + graphClientA + `","client_secret":"b"}]}}`

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(payload), fleet.ApplySpecOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Only 1 Microsoft Graph credential can be configured")
		assert.Empty(t, env.stored, "neither entry is stored when the list is rejected")
	})

	t.Run("rejects a malformed GUID", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		payload := `{"mdm":{"microsoft_graph_credentials":[{"tenant_id":"not-a-guid","client_id":"` +
			graphClientA + `","client_secret":"a"}]}}`

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(payload), fleet.ApplySpecOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid Entra tenant ID")
		assert.Empty(t, env.stored)
	})

	t.Run("rejects a new secret when no server private key is configured", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "", nil)

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(validPayload), fleet.ApplySpecOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Missing required private key")
		assert.Empty(t, env.stored)
	})

	t.Run("rejects a credential that fails verification", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key",
			&msgraph.Error{StatusCode: http.StatusForbidden, Code: "Forbidden"})

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(validPayload), fleet.ApplySpecOptions{})
		require.Error(t, err)
		// The message has to tell the admin what to actually do, since a 403 here means a missing permission or
		// missing admin consent rather than a bad secret.
		assert.Contains(t, err.Error(), "DeviceManagementServiceConfig.Read.All")
		assert.Empty(t, env.stored)
	})

	t.Run("reports an auth failure differently from a permission failure", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key",
			&msgraph.Error{StatusCode: http.StatusUnauthorized, Code: "InvalidAuthenticationToken"})

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(validPayload), fleet.ApplySpecOptions{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rejected the credential")
	})

	t.Run("preserves the stored secret when the mask is sent back", func(t *testing.T) {
		env, calls := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.stored[graphTenantA] = &fleet.MicrosoftGraphCredential{
			TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: "stored-secret",
		}

		payload := `{"mdm":{"microsoft_graph_credentials":[{"tenant_id":"` + graphTenantA +
			`","client_id":"` + graphClientA + `","client_secret":"` + fleet.MaskedPassword + `"}]}}`

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(payload), fleet.ApplySpecOptions{})
		require.NoError(t, err)

		assert.Equal(t, "stored-secret", env.stored[graphTenantA].ClientSecret)
		// Nothing changed, so no network call and no write.
		assert.Equal(t, 0, *calls, "an unchanged credential is not re-verified")
	})

	t.Run("omitting the key leaves stored credentials alone", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.stored[graphTenantA] = &fleet.MicrosoftGraphCredential{
			TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: "stored-secret",
		}

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(`{"org_info":{"org_name":"Renamed"}}`), fleet.ApplySpecOptions{})
		require.NoError(t, err)

		require.Len(t, env.stored, 1, "a PATCH that does not mention credentials must not clear them")
		assert.Empty(t, env.deleted)
	})

	t.Run("an explicit empty list clears the credential", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.stored[graphTenantA] = &fleet.MicrosoftGraphCredential{
			TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: "stored-secret",
		}

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(`{"mdm":{"microsoft_graph_credentials":[]}}`), fleet.ApplySpecOptions{})
		require.NoError(t, err)

		assert.Empty(t, env.stored)
		assert.Equal(t, []string{graphTenantA}, env.deleted)
	})

	t.Run("the secret never lands in the saved app config", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		// Snapshot the count at save time rather than holding the pointer: the mock hands back the same *AppConfig on
		// the later read, so a retained pointer would observe the response hydration instead of what was persisted.
		savedCredCount := -1
		env.ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
			savedCredCount = len(conf.MDM.MicrosoftGraphCredentials.Value)
			return nil
		}

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(validPayload), fleet.ApplySpecOptions{})
		require.NoError(t, err)

		assert.Equal(t, 0, savedCredCount,
			"credentials belong to their own table; persisting them in app_config_json would store the secret in plaintext")
	})

	t.Run("a dry run validates without persisting", func(t *testing.T) {
		env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		_, err := env.svc.ModifyAppConfig(env.ctx, []byte(validPayload), fleet.ApplySpecOptions{DryRun: true})
		require.NoError(t, err)
		assert.Empty(t, env.stored)
	})
}

func TestMicrosoftGraphVerifyMessage(t *testing.T) {
	for _, tc := range []struct {
		name     string
		err      error
		contains string
	}{
		{"permission", &msgraph.Error{StatusCode: http.StatusForbidden}, "DeviceManagementServiceConfig.Read.All"},
		{"auth", &msgraph.Error{StatusCode: http.StatusUnauthorized}, "rejected the credential"},
		{"transient", &msgraph.Error{StatusCode: http.StatusBadGateway}, "temporarily unavailable"},
		{"non-graph error", errors.New("dial tcp: timeout"), "Couldn't connect to Microsoft Graph"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, microsoftGraphVerifyMessage(tc.err), tc.contains)
		})
	}
}

// The PATCH response is built from a re-read of the app config JSON, which never carries credentials. Without explicit
// hydration it would report an empty list right after storing one, and a UI that renders the response would show the
// credential as removed until the next config read.
func TestModifyAppConfigMicrosoftGraphCredentialsResponse(t *testing.T) {
	env, _ := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

	payload := `{"mdm":{"microsoft_graph_credentials":[{"tenant_id":"` + graphTenantA +
		`","client_id":"` + graphClientA + `","client_secret":"secret-a"}]}}`

	modified, err := env.svc.ModifyAppConfig(env.ctx, []byte(payload), fleet.ApplySpecOptions{})
	require.NoError(t, err)

	require.Len(t, modified.MDM.MicrosoftGraphCredentials.Value, 1,
		"the PATCH response must report the credential it just stored")
	got := modified.MDM.MicrosoftGraphCredentials.Value[0]
	assert.Equal(t, graphTenantA, got.TenantID)
	assert.Equal(t, graphClientA, got.ClientID)
	// Masked, never the plaintext the caller sent.
	assert.Equal(t, fleet.MaskedPassword, got.ClientSecret)
}
