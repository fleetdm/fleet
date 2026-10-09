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

// fakeGraphClient stands in for a real Graph client so credential validation never reaches the network.
type fakeGraphClient struct {
	verifyErr error
}

func (f *fakeGraphClient) VerifyCredential(context.Context) error { return f.verifyErr }

func (f *fakeGraphClient) ListWindowsAutopilotDevices(context.Context) ([]msgraph.WindowsAutopilotDevice, error) {
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
	// verifyCalls counts clients the factory built, which is one per credential actually verified against Graph.
	verifyCalls *int
}

// seed puts a credential in the store as though a previous apply had written it, returning it so a caller can set
// sync state on top.
func (e *graphCredsTestEnv) seed(tenantID, clientID, secret string) *fleet.MicrosoftGraphCredential {
	cred := &fleet.MicrosoftGraphCredential{TenantID: tenantID, ClientID: clientID, ClientSecret: secret}
	e.stored[tenantID] = cred
	return cred
}

// setCredentialInvalid and credentialInvalid read and write the aggregate status flag on the app config.
func (e *graphCredsTestEnv) setCredentialInvalid(t *testing.T, invalid bool) {
	t.Helper()
	ac, err := e.ds.AppConfig(e.ctx)
	require.NoError(t, err)
	ac.MDM.MicrosoftGraphCredentialInvalid = invalid
	require.NoError(t, e.ds.SaveAppConfig(e.ctx, ac))
}

func (e *graphCredsTestEnv) credentialInvalid(t *testing.T) bool {
	t.Helper()
	ac, err := e.ds.AppConfig(e.ctx)
	require.NoError(t, err)
	return ac.MDM.MicrosoftGraphCredentialInvalid
}

func setupGraphCredsTest(t *testing.T, tier string, privateKey string, verifyErr error) *graphCredsTestEnv {
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

	env := &graphCredsTestEnv{svc: svc, ctx: ctx, ds: ds, stored: map[string]*fleet.MicrosoftGraphCredential{}, verifyCalls: calls}

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
	ds.ListMicrosoftGraphCredentialMetadataFunc = func(ctx context.Context) ([]*fleet.MicrosoftGraphCredentialMetadata, error) {
		out := make([]*fleet.MicrosoftGraphCredentialMetadata, 0, len(env.stored))
		for _, c := range env.stored {
			out = append(out, &c.MicrosoftGraphCredentialMetadata)
		}
		return out, nil
	}
	ds.UpdateMicrosoftGraphCredentialInvalidAggregateFunc = func(ctx context.Context) error {
		var anyInvalid bool
		for _, c := range env.stored {
			if c.CredentialInvalid {
				anyInvalid = true
				break
			}
		}
		ac, err := ds.AppConfig(ctx)
		if err != nil {
			return err
		}
		ac.MDM.MicrosoftGraphCredentialInvalid = anyInvalid
		return ds.SaveAppConfig(ctx, ac)
	}
	ds.ReplaceMicrosoftGraphCredentialsFunc = func(ctx context.Context, upsert []*fleet.MicrosoftGraphCredential, deleteTenantIDs []string) error {
		for _, cred := range upsert {
			copied := *cred
			env.stored[cred.TenantID] = &copied
		}
		for _, tenantID := range deleteTenantIDs {
			delete(env.stored, tenantID)
			env.deleted = append(env.deleted, tenantID)
		}
		return nil
	}

	return env
}

func TestApplyMicrosoftGraphCredentials(t *testing.T) {
	t.Parallel()
	validCred := []fleet.MicrosoftGraphCredential{
		{TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: "secret-a"},
	}

	t.Run("stores a credential on premium", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		require.NoError(t, env.svc.ApplyMicrosoftGraphCredentials(env.ctx, validCred, false))

		require.Len(t, env.stored, 1)
		stored := env.stored[graphTenantA]
		require.NotNil(t, stored)
		assert.Equal(t, graphClientA, stored.ClientID)
		assert.Equal(t, "secret-a", stored.ClientSecret)
		assert.Equal(t, 1, *env.verifyCalls, "a new credential is verified before it is stored")
	})

	t.Run("rejects on Fleet Free", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierFree, "test-private-key", nil)

		err := env.svc.ApplyMicrosoftGraphCredentials(env.ctx, validCred, false)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
		assert.Empty(t, env.stored)
	})

	t.Run("rejects a second credential", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		err := env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{
			{TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: "a"},
			{TenantID: graphTenantB, ClientID: graphClientA, ClientSecret: "b"},
		}, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Only 1 Microsoft Graph credential can be configured")
		assert.Empty(t, env.stored, "neither entry is stored when the list is rejected")
	})

	t.Run("rejects a malformed GUID", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		err := env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{
			{TenantID: "not-a-guid", ClientID: graphClientA, ClientSecret: "a"},
		}, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid Entra tenant ID")
		assert.Empty(t, env.stored)
	})

	t.Run("rejects a new secret when no server private key is configured", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "", nil)

		err := env.svc.ApplyMicrosoftGraphCredentials(env.ctx, validCred, false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Missing required private key")
		assert.Empty(t, env.stored)
	})

	t.Run("rejects a credential that fails verification", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key",
			&msgraph.Error{StatusCode: http.StatusForbidden, Code: "Forbidden"})

		err := env.svc.ApplyMicrosoftGraphCredentials(env.ctx, validCred, false)
		require.Error(t, err)
		// One end-to-end case is enough to prove the classified message reaches the caller
		assert.Contains(t, err.Error(), "DeviceManagementServiceConfig.Read.All")
		assert.Empty(t, env.stored)
	})

	t.Run("preserves the stored secret when the mask is sent back", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.seed(graphTenantA, graphClientA, "stored-secret")

		require.NoError(t, env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{
			{TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: fleet.MaskedPassword},
		}, false))

		assert.Equal(t, "stored-secret", env.stored[graphTenantA].ClientSecret)
		// Nothing changed, so no network call and no write.
		assert.Equal(t, 0, *env.verifyCalls, "an unchanged credential is not re-verified")
	})

	// A client secret belongs to one app registration. Re-pairing a stored secret with a different tenant or client
	// would silently attach a credential to an app it was never issued for, so the mask only preserves the secret when
	// both IDs still match.
	t.Run("changing an ID requires a new secret", func(t *testing.T) {
		const otherClientID = "9a1c1d3e-0000-4b2a-9c3d-0f1e2d3c4b5a"
		for _, tc := range []struct {
			name     string
			tenantID string
			clientID string
		}{
			{"client ID changed", graphTenantA, otherClientID},
			{"tenant ID changed", graphTenantB, graphClientA},
		} {
			t.Run(tc.name, func(t *testing.T) {
				env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
				env.seed(graphTenantA, graphClientA, "stored-secret")

				err := env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{
					{TenantID: tc.tenantID, ClientID: tc.clientID, ClientSecret: fleet.MaskedPassword},
				}, false)

				require.Error(t, err)
				assert.Contains(t, err.Error(), "client_secret must be provided")
				assert.Equal(t, "stored-secret", env.stored[graphTenantA].ClientSecret, "the old credential is untouched")
				assert.Zero(t, *env.verifyCalls, "nothing should be verified against Graph with a mismatched secret")
			})
		}
	})

	t.Run("is declarative: an absent tenant is deleted", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.seed(graphTenantA, graphClientA, "stored-secret")

		require.NoError(t, env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{}, false))

		assert.Empty(t, env.stored)
		assert.Equal(t, []string{graphTenantA}, env.deleted)
	})

	// Deleting must not require decrypting.
	t.Run("clears credentials even when the stored secret cannot be decrypted", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.seed(graphTenantA, graphClientA, "unreadable")
		env.ds.ListMicrosoftGraphCredentialsFunc = func(ctx context.Context) ([]*fleet.MicrosoftGraphCredential, error) {
			return nil, errors.New("decrypt microsoft graph client secret: cipher: message authentication failed")
		}

		require.NoError(t, env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{}, false))
		assert.Empty(t, env.stored)
		assert.Equal(t, []string{graphTenantA}, env.deleted)
	})

	t.Run("a dry run validates without persisting", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		require.NoError(t, env.svc.ApplyMicrosoftGraphCredentials(env.ctx, validCred, true))
		assert.Empty(t, env.stored)
		assert.Equal(t, 1, *env.verifyCalls, "a dry run still verifies, which is the point of running it")
	})
}

func TestMicrosoftGraphCredentialInvalidFlag(t *testing.T) {
	t.Parallel()
	t.Run("clears when a credential is replaced with a working one", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.seed(graphTenantA, graphClientA, "expired").CredentialInvalid = true
		env.setCredentialInvalid(t, true)

		// A rotated secret is verified before storage, and the upsert clears the per-tenant flag.
		require.NoError(t, env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{
			{TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: "rotated"},
		}, false))

		assert.False(t, env.credentialInvalid(t), "rotating to a verified credential must clear the flag")
	})

	t.Run("clears when the last unhealthy credential is deleted", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.seed(graphTenantA, graphClientA, "expired").CredentialInvalid = true
		env.setCredentialInvalid(t, true)

		require.NoError(t, env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{}, false))

		assert.False(t, env.credentialInvalid(t), "a deleted credential can no longer need attention")
	})

	// The aggregate is recomputed only when a credential actually changed. Re-applying an identical GitOps config is
	// the common case, and it must not read or rewrite the app config.
	t.Run("an unchanged credential does not touch the app config", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
		env.seed(graphTenantA, graphClientA, "stored-secret")
		env.ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
			t.Fatal("a no-op apply must not save the app config")
			return nil
		}
		// Counting the read, not the write, is what distinguishes a skipped recomputation from one that ran and found
		// nothing to change: the latter still reads the app config before deciding.
		var appConfigReads int
		env.ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			appConfigReads++
			return &fleet.AppConfig{}, nil
		}

		require.NoError(t, env.svc.ApplyMicrosoftGraphCredentials(env.ctx, []fleet.MicrosoftGraphCredential{
			{TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: "stored-secret"},
		}, false))

		assert.Zero(t, appConfigReads, "the flag must not be recomputed when nothing changed")
		assert.Equal(t, 0, *env.verifyCalls, "an unchanged credential is not re-verified either")
	})

	t.Run("cannot be set through PATCH /config", func(t *testing.T) {
		env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

		modified, err := env.svc.ModifyAppConfig(env.ctx,
			[]byte(`{"mdm":{"microsoft_graph_credential_invalid":true}}`), fleet.ApplySpecOptions{})
		require.NoError(t, err)

		assert.False(t, modified.MDM.MicrosoftGraphCredentialInvalid,
			"the flag is server-computed; a client must not be able to set it")
		assert.False(t, env.credentialInvalid(t))
	})
}

// The list endpoint must use the metadata query, which decrypts nothing.
func TestListMicrosoftGraphCredentials(t *testing.T) {
	t.Parallel()
	env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)
	env.seed(graphTenantA, graphClientA, "stored-secret").CredentialInvalid = true

	creds, err := env.svc.ListMicrosoftGraphCredentials(env.ctx)
	require.NoError(t, err)

	require.Len(t, creds, 1)
	assert.Equal(t, graphTenantA, creds[0].TenantID)
	assert.Equal(t, graphClientA, creds[0].ClientID)
	// No assertion that the secret is absent: the returned type has no field to carry one.
	// Per-tenant status is the whole reason this endpoint exists; it is no longer on the app config.
	assert.True(t, creds[0].CredentialInvalid)

	assert.True(t, env.ds.ListMicrosoftGraphCredentialMetadataFuncInvoked,
		"the read must go through the metadata query")
	assert.False(t, env.ds.ListMicrosoftGraphCredentialsFuncInvoked,
		"the read must not decrypt secrets it has no way to return")
}

func TestMicrosoftGraphCredentialsAuth(t *testing.T) {
	t.Parallel()
	env := setupGraphCredsTest(t, fleet.TierPremium, "test-private-key", nil)

	// Reading a credential is held to the same roles as writing one, so a single expectation covers both calls.
	for _, tc := range []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{"global admin", &fleet.User{GlobalRole: new(fleet.RoleAdmin)}, false},
		{"global gitops", &fleet.User{GlobalRole: new(fleet.RoleGitOps)}, false},
		{"global maintainer", &fleet.User{GlobalRole: new(fleet.RoleMaintainer)}, true},
		{"global observer", &fleet.User{GlobalRole: new(fleet.RoleObserver)}, true},
		{"team admin", &fleet.User{Teams: []fleet.UserTeam{{ID: 1, Role: fleet.RoleAdmin}}}, true},
		{"team observer", &fleet.User{Teams: []fleet.UserTeam{{ID: 1, Role: fleet.RoleObserver}}}, true},
		{"no user", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := viewer.NewContext(env.ctx, viewer.Viewer{User: tc.user})

			_, err := env.svc.ListMicrosoftGraphCredentials(ctx)
			checkAuthErr(t, tc.shouldFail, err)

			// A dry run exercises the authorization check without depending on the datastore mocks.
			err = env.svc.ApplyMicrosoftGraphCredentials(ctx, []fleet.MicrosoftGraphCredential{
				{TenantID: graphTenantA, ClientID: graphClientA, ClientSecret: "secret-a"},
			}, true)
			checkAuthErr(t, tc.shouldFail, err)
		})
	}
}
