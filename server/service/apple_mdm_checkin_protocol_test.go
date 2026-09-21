package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const testGetTokenUDID = "ABC-DEF"

// newGetTokenTestService returns a GetToken service backed by mocks that resolve to a single
// default ABM token and an unknown host. Tests override the funcs they care about.
func newGetTokenTestService(t *testing.T) (*MDMAppleGetTokenService, *mock.Store, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	ds := new(mock.Store)
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: 1, ServerUUID: "default-server-uuid", IsDefault: true}}, nil
	}
	ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
		return nil, newNotFoundError()
	}
	ds.GetHostDEPAssignmentFunc = func(ctx context.Context, hostID uint) (*fleet.HostDEPAssignment, error) {
		return nil, newNotFoundError()
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{fleet.MDMAssetABMKey: {Name: fleet.MDMAssetABMKey, Value: keyPEM}}, nil
	}

	svc := NewMDMAppleGetTokenService(ds, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return svc, ds, key
}

func getTokenRequest(t *testing.T, svc *MDMAppleGetTokenService, serviceType string) (*mdm.GetTokenResponse, error) {
	t.Helper()
	return svc.GetToken(
		&mdm.Request{Context: t.Context()},
		&mdm.GetToken{UDID: testGetTokenUDID, TokenServiceType: serviceType},
	)
}

func TestMDMAppleGetTokenClaims(t *testing.T) {
	svc, _, key := newGetTokenTestService(t)

	resp, err := getTokenRequest(t, svc, fleet.TokenServiceTypeMAID)
	require.NoError(t, err)

	var claims appleMAIDTokenClaims
	_, err = jwt.ParseWithClaims(string(resp.TokenData), &claims, func(*jwt.Token) (any, error) { return &key.PublicKey, nil })
	require.NoError(t, err)
	require.Equal(t, fleet.TokenServiceTypeMAID, claims.TokenServiceType)
	require.Equal(t, "default-server-uuid", claims.Issuer)
	require.NotEmpty(t, claims.ID)
	require.NotNil(t, claims.IssuedAt)
}

func TestMDMAppleGetTokenSigningTokenSelection(t *testing.T) {
	depAssignedHost := func(ds *mock.Store, abmTokenID *uint) {
		ds.HostLiteByIdentifierFunc = func(ctx context.Context, identifier string) (*fleet.HostLite, error) {
			return &fleet.HostLite{ID: 42, UUID: identifier}, nil
		}
		ds.GetHostDEPAssignmentFunc = func(ctx context.Context, hostID uint) (*fleet.HostDEPAssignment, error) {
			return &fleet.HostDEPAssignment{HostID: hostID, ABMTokenID: abmTokenID}, nil
		}
	}

	cases := []struct {
		name        string
		serviceType string
		setup       func(ds *mock.Store)
		wantIssuer  string
	}{
		{
			name:        "unsupported token service type",
			serviceType: "com.apple.watch.pairing",
		},
		{
			name:        "unknown host falls back to the default token",
			serviceType: fleet.TokenServiceTypeMAID,
			wantIssuer:  "default-server-uuid",
		},
		{
			name:        "DEP assigned host uses its own token",
			serviceType: fleet.TokenServiceTypeMAID,
			setup: func(ds *mock.Store) {
				depAssignedHost(ds, new(uint(2)))
				ds.GetABMTokenByIDFunc = func(ctx context.Context, tokenID uint) (*fleet.ABMToken, error) {
					return &fleet.ABMToken{ID: tokenID, ServerUUID: "dep-server-uuid"}, nil
				}
			},
			wantIssuer: "dep-server-uuid",
		},
		{
			name:        "DEP assigned host does not fall back to the default token",
			serviceType: fleet.TokenServiceTypeMAID,
			setup: func(ds *mock.Store) {
				depAssignedHost(ds, nil)
			},
		},
		{
			name:        "no ABM tokens",
			serviceType: fleet.TokenServiceTypeMAID,
			setup: func(ds *mock.Store) {
				ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) { return nil, nil }
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, ds, key := newGetTokenTestService(t)
			if c.setup != nil {
				c.setup(ds)
			}

			resp, err := getTokenRequest(t, svc, c.serviceType)
			if c.wantIssuer == "" {
				require.ErrorContains(t, err, "HTTP status 400 (Bad Request)")
				return
			}

			require.NoError(t, err)
			var claims appleMAIDTokenClaims
			_, err = jwt.ParseWithClaims(string(resp.TokenData), &claims, func(*jwt.Token) (any, error) { return &key.PublicKey, nil })
			require.NoError(t, err)
			require.Equal(t, c.wantIssuer, claims.Issuer)
		})
	}
}
