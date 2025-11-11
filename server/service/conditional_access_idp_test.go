package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestConditionalAccessGetIdPSigningCertAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// Mock the datastore to return a valid IdP certificate
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetConditionalAccessIDPCert: {
				Name:  fleet.MDMAssetConditionalAccessIDPCert,
				Value: []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"),
			},
		}, nil
	}

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{"global admin", test.UserAdmin, false},
		{"global maintainer", test.UserMaintainer, false},
		{"global observer", test.UserObserver, false},
		{"global observer+", test.UserObserverPlus, false},
		{"global gitops", test.UserGitOps, false},
		{"team admin", test.UserTeamAdminTeam1, true},
		{"team maintainer", test.UserTeamMaintainerTeam1, true},
		{"team observer", test.UserTeamObserverTeam1, true},
		{"team observer+", test.UserTeamObserverPlusTeam1, true},
		{"team gitops", test.UserTeamGitOpsTeam1, true},
		{"user no roles", test.UserNoRoles, true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := test.UserContext(ctx, tt.user)

			certPEM, err := svc.ConditionalAccessGetIdPSigningCert(ctx)
			if tt.shouldFail {
				require.Error(t, err)
				var forbiddenError *authz.Forbidden
				require.ErrorAs(t, err, &forbiddenError)
				require.Nil(t, certPEM)
			} else {
				require.NoError(t, err)
				require.NotNil(t, certPEM)
			}
		})
	}
}

func TestConditionalAccessGetIdPAppleProfileAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// Mock valid certificate
	certPEM := generateTestCertPEM(t)

	// Mock the datastore methods
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetConditionalAccessCACert: {
				Name:  fleet.MDMAssetConditionalAccessCACert,
				Value: certPEM,
			},
		}, nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: "https://fleet.example.com",
			},
		}, nil
	}

	ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
		return []*fleet.EnrollSecret{
			{Secret: "test-secret-123"},
		}, nil
	}

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{"global admin", test.UserAdmin, false},
		{"global maintainer", test.UserMaintainer, false},
		{"global observer", test.UserObserver, false},
		{"global observer+", test.UserObserverPlus, false},
		{"global gitops", test.UserGitOps, false},
		{"team admin", test.UserTeamAdminTeam1, true},
		{"team maintainer", test.UserTeamMaintainerTeam1, true},
		{"team observer", test.UserTeamObserverTeam1, true},
		{"team observer+", test.UserTeamObserverPlusTeam1, true},
		{"team gitops", test.UserTeamGitOpsTeam1, true},
		{"user no roles", test.UserNoRoles, true},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := test.UserContext(ctx, tt.user)

			profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
			if tt.shouldFail {
				require.Error(t, err)
				var forbiddenError *authz.Forbidden
				require.ErrorAs(t, err, &forbiddenError)
				require.Nil(t, profileData)
			} else {
				require.NoError(t, err)
				require.NotNil(t, profileData)

				// Verify the profile contains expected content
				profileStr := string(profileData)
				require.Contains(t, profileStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
				require.Contains(t, profileStr, "com.fleetdm.conditional-access")
				require.Contains(t, profileStr, "https://fleet.example.com/api/fleet/conditional_access/scep")
				require.Contains(t, profileStr, "https://okta.fleet.example.com")
				require.Contains(t, profileStr, "Fleet conditional access for Okta")
			}
		})
	}
}

// generateTestCertPEM generates a test certificate in PEM format for testing
func generateTestCertPEM(t *testing.T) []byte {
	// Create a simple self-signed certificate
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	return certPEM
}

func TestConditionalAccessGetIdPAppleProfile(t *testing.T) {
	certPEM := generateTestCertPEM(t)

	t.Run("success - generates valid profile", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		ctx = test.UserContext(ctx, test.UserAdmin)

		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetConditionalAccessCACert: {
					Name:  fleet.MDMAssetConditionalAccessCACert,
					Value: certPEM,
				},
			}, nil
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://fleet.example.com:8080",
				},
			}, nil
		}

		ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
			return []*fleet.EnrollSecret{
				{Secret: "test-secret-456"},
			}, nil
		}

		profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, profileData)

		profileStr := string(profileData)
		// Verify XML structure
		require.Contains(t, profileStr, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
		require.Contains(t, profileStr, "<!DOCTYPE plist")

		// Verify URLs with port preserved
		require.Contains(t, profileStr, "https://fleet.example.com:8080/api/fleet/conditional_access/scep")
		require.Contains(t, profileStr, "https://okta.fleet.example.com:8080")

		// Verify challenge secret
		require.Contains(t, profileStr, "test-secret-456")

		// Verify payload identifiers
		require.Contains(t, profileStr, "com.fleetdm.conditional-access-ca")
		require.Contains(t, profileStr, "com.fleetdm.conditional-access-scep")
		require.Contains(t, profileStr, "com.fleetdm.conditional-access-preference")
		require.Contains(t, profileStr, "com.fleetdm.chrome.certs")

		// Verify certificate CN is present in the profile
		require.Contains(t, profileStr, "Fleet conditional access for Okta")
	})

	t.Run("missing CA certificate", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		ctx = test.UserContext(ctx, test.UserAdmin)

		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{}, nil
		}

		profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "conditional access CA certificate not configured")
		require.Nil(t, profileData)
	})

	t.Run("invalid PEM certificate", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		ctx = test.UserContext(ctx, test.UserAdmin)

		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetConditionalAccessCACert: {
					Name:  fleet.MDMAssetConditionalAccessCACert,
					Value: []byte("not a valid PEM"),
				},
			}, nil
		}

		profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode CA certificate PEM")
		require.Nil(t, profileData)
	})

	t.Run("invalid DER certificate", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		ctx = test.UserContext(ctx, test.UserAdmin)

		// Valid PEM structure but invalid DER content
		invalidCertPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: []byte("invalid DER data"),
		})

		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetConditionalAccessCACert: {
					Name:  fleet.MDMAssetConditionalAccessCACert,
					Value: invalidCertPEM,
				},
			}, nil
		}

		profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse CA certificate")
		require.Nil(t, profileData)
	})

	t.Run("no enroll secrets", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		ctx = test.UserContext(ctx, test.UserAdmin)

		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetConditionalAccessCACert: {
					Name:  fleet.MDMAssetConditionalAccessCACert,
					Value: certPEM,
				},
			}, nil
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://fleet.example.com",
				},
			}, nil
		}

		ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
			return []*fleet.EnrollSecret{}, nil
		}

		profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "enroll_secret")
		require.True(t, fleet.IsNotFound(err))
		require.Nil(t, profileData)
	})

	t.Run("server URL not configured", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		ctx = test.UserContext(ctx, test.UserAdmin)

		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetConditionalAccessCACert: {
					Name:  fleet.MDMAssetConditionalAccessCACert,
					Value: certPEM,
				},
			}, nil
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "",
				},
			}, nil
		}

		profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "server URL is not configured")
		var badReqErr *fleet.BadRequestError
		require.ErrorAs(t, err, &badReqErr)
		require.Nil(t, profileData)
	})

	t.Run("invalid server URL", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		ctx = test.UserContext(ctx, test.UserAdmin)

		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetConditionalAccessCACert: {
					Name:  fleet.MDMAssetConditionalAccessCACert,
					Value: certPEM,
				},
			}, nil
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "://invalid-url",
				},
			}, nil
		}

		profileData, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse server URL")
		require.Nil(t, profileData)
	})

	t.Run("deterministic UUIDs based on server URL", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil)
		ctx = test.UserContext(ctx, test.UserAdmin)

		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, assetNames []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetConditionalAccessCACert: {
					Name:  fleet.MDMAssetConditionalAccessCACert,
					Value: certPEM,
				},
			}, nil
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://fleet.example.com",
				},
			}, nil
		}

		ds.GetEnrollSecretsFunc = func(ctx context.Context, teamID *uint) ([]*fleet.EnrollSecret, error) {
			return []*fleet.EnrollSecret{
				{Secret: "test-secret"},
			}, nil
		}

		// Generate profile twice with same server URL
		profileData1, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.NoError(t, err)

		profileData2, err := svc.ConditionalAccessGetIdPAppleProfile(ctx)
		require.NoError(t, err)

		// UUIDs should be identical
		require.Equal(t, profileData1, profileData2)
	})
}
