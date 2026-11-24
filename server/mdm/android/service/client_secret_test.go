package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_mock "github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// The client's secret is loaded at startup, before Android MDM may have been turned on, and other
// Fleet servers can turn it on, so these request-path calls must load the current secret themselves.
func TestAppStoreAppCallsLoadCurrentSecret(t *testing.T) {
	const enterpriseName = "enterprises/LC1234"

	cases := []struct {
		name     string
		call     func(ctx context.Context, svc android.Service) error
		wantCall string
	}{
		{
			name: "EnterprisesApplications",
			call: func(ctx context.Context, svc android.Service) error {
				_, err := svc.EnterprisesApplications(ctx, enterpriseName, "com.example.app")
				return err
			},
			wantCall: "get application com.example.app",
		},
		{
			name: "CreateAndroidWebApp",
			call: func(ctx context.Context, svc android.Service) error {
				_, err := svc.CreateAndroidWebApp(ctx, enterpriseName, &androidmanagement.WebApp{Title: "Example"})
				return err
			},
			wantCall: "create web app Example",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := android_mock.Client{}
			client.InitCommonMocks()
			var calls []string
			client.SetAuthenticationSecretFunc = func(secret string) error {
				calls = append(calls, "auth "+secret)
				return nil
			}
			client.EnterprisesApplicationsFunc = func(_ context.Context, _ string, packageName string) (*androidmanagement.Application, error) {
				calls = append(calls, "get application "+packageName)
				return &androidmanagement.Application{}, nil
			}
			client.EnterprisesWebAppsCreateFunc = func(_ context.Context, _ string, webApp *androidmanagement.WebApp) (*androidmanagement.WebApp, error) {
				calls = append(calls, "create web app "+webApp.Title)
				return webApp, nil
			}

			// No secret exists when the server starts.
			var storedSecret []byte
			var assetsErr error
			fleetDS := InitCommonDSMocks()
			fleetDS.Store.GetAllMDMConfigAssetsByNameFunc = func(_ context.Context, _ []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
				if assetsErr != nil {
					return nil, assetsErr
				}
				if storedSecret == nil {
					return nil, &notFoundError{}
				}
				return map[fleet.MDMAssetName]fleet.MDMConfigAsset{fleet.MDMAssetAndroidFleetServerSecret: {Value: storedSecret}}, nil
			}
			svc, err := NewServiceWithClient(slog.New(slog.DiscardHandler), fleetDS, &client, "test-private-key", &fleetDS.DataStore, noopNewActivity, config.AndroidAgentConfig{})
			require.NoError(t, err)
			require.Equal(t, []string{"auth "}, calls)

			t.Run("uses the secret stored after startup", func(t *testing.T) {
				calls = nil
				storedSecret = []byte("server-secret")
				require.NoError(t, tc.call(t.Context(), svc))
				require.Equal(t, []string{"auth server-secret", tc.wantCall}, calls)
			})

			t.Run("does not call AMAPI when the secret can't be read", func(t *testing.T) {
				calls = nil
				assetsErr = errors.New("database is down")
				err := tc.call(t.Context(), svc)
				require.ErrorContains(t, err, "database is down")
				require.Empty(t, calls)
			})
		})
	}
}
