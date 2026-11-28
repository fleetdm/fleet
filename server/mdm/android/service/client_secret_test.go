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
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// The client's secret is loaded at startup, before Android MDM may have been turned on, and other
// Fleet servers can turn it on, so calls made from requests and from the software worker must load
// the current secret themselves.
func TestAMAPICallsLoadCurrentSecret(t *testing.T) {
	const (
		enterpriseName = "enterprises/LC1234"
		hostUUID       = "host-uuid"
		agentPackage   = "com.fleetdm.agent"
	)
	policyName := enterpriseName + "/policies/" + hostUUID
	hostPolicies := map[string]string{hostUUID: hostUUID}
	apps := []*androidmanagement.ApplicationPolicy{{PackageName: "com.example.app"}}

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
		{
			name: "AddAppsToAndroidPolicy",
			call: func(ctx context.Context, svc android.Service) error {
				_, err := svc.AddAppsToAndroidPolicy(ctx, enterpriseName, apps, hostPolicies)
				return err
			},
			wantCall: "modify policy applications " + policyName,
		},
		{
			name: "RemoveAppsFromAndroidPolicy",
			call: func(ctx context.Context, svc android.Service) error {
				_, err := svc.RemoveAppsFromAndroidPolicy(ctx, enterpriseName, []string{"com.example.app"}, hostPolicies)
				return err
			},
			wantCall: "remove policy applications " + policyName,
		},
		{
			name: "SetAppsForAndroidPolicy",
			call: func(ctx context.Context, svc android.Service) error {
				_, err := svc.SetAppsForAndroidPolicy(ctx, enterpriseName, apps, hostPolicies)
				return err
			},
			wantCall: "patch policy " + policyName,
		},
		{
			name: "PatchPolicy",
			call: func(ctx context.Context, svc android.Service) error {
				_, err := svc.PatchPolicy(ctx, hostUUID, policyName, &androidmanagement.Policy{}, nil)
				return err
			},
			wantCall: "patch policy " + policyName,
		},
		{
			name: "PrepareHostPolicy",
			call: func(ctx context.Context, svc android.Service) error {
				return svc.PrepareHostPolicy(ctx, enterpriseName, hostUUID, 0)
			},
			wantCall: "patch policy " + policyName,
		},
		{
			name: "PatchDevice",
			call: func(ctx context.Context, svc android.Service) error {
				_, err := svc.PatchDevice(ctx, hostUUID, enterpriseName+"/devices/device-id", &androidmanagement.Device{})
				return err
			},
			wantCall: "patch device " + enterpriseName + "/devices/device-id",
		},
		{
			name: "AddFleetAgentToAndroidPolicy",
			call: func(ctx context.Context, svc android.Service) error {
				return svc.AddFleetAgentToAndroidPolicy(ctx, enterpriseName, map[string]android.AgentManagedConfiguration{hostUUID: {}})
			},
			wantCall: "modify policy applications " + policyName,
		},
		{
			name: "SendFleetAgentConfig",
			call: func(ctx context.Context, svc android.Service) error {
				return svc.SendFleetAgentConfig(ctx, enterpriseName, hostUUID)
			},
			wantCall: "modify policy applications " + policyName,
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
			client.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(_ context.Context, name string, _ []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
				calls = append(calls, "modify policy applications "+name)
				return &androidmanagement.Policy{Version: 1}, nil
			}
			client.EnterprisesPoliciesRemovePolicyApplicationsFunc = func(_ context.Context, name string, _ []string) (*androidmanagement.Policy, error) {
				calls = append(calls, "remove policy applications "+name)
				return &androidmanagement.Policy{Version: 1}, nil
			}
			client.EnterprisesPoliciesPatchFunc = func(_ context.Context, name string, policy *androidmanagement.Policy, _ androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
				calls = append(calls, "patch policy "+name)
				return policy, nil
			}
			client.EnterprisesDevicesPatchFunc = func(_ context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
				calls = append(calls, "patch device "+name)
				return device, nil
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
			fleetDS.Store.AndroidHostLiteByHostUUIDFunc = func(_ context.Context, _ string) (*fleet.AndroidHost, error) {
				return &fleet.AndroidHost{Host: &fleet.Host{ID: 1, UUID: hostUUID}, Device: &android.Device{}}, nil
			}
			fleetDS.DataStore.BulkUpsertHostAndroidSoftwareInstallsFunc = func(_ context.Context, _, _ []fleet.HostAndroidSoftwareInstall) error {
				return nil
			}
			fleetDS.DataStore.ListAndroidFleetDefaultPolicyProfilesFunc = func(_ context.Context, _ *uint) ([]*fleet.MDMAndroidConfigProfile, error) {
				return nil, nil
			}
			fleetDS.DataStore.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: "https://fleet.example.com"}}, nil
			}
			fleetDS.DataStore.GetEnrollSecretsFunc = func(_ context.Context, _ *uint) ([]*fleet.EnrollSecret, error) {
				return []*fleet.EnrollSecret{{Secret: "enroll-secret"}}, nil
			}
			fleetDS.DataStore.GetAndTransitionCertificateTemplatesToDeliveringFunc = func(_ context.Context, _ string) (*fleet.HostCertificateTemplatesForDelivery, error) {
				return &fleet.HostCertificateTemplatesForDelivery{}, nil
			}
			svc, err := NewServiceWithClient(slog.New(slog.DiscardHandler), fleetDS, &client, "test-private-key", &fleetDS.DataStore, noopNewActivity,
				config.AndroidAgentConfig{Package: agentPackage, SigningSHA256: "abc123"})
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
