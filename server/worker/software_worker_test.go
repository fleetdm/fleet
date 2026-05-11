package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func TestSoftwareWorker(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	// call TruncateTables immediately as some DB migrations may create jobs
	mysql.TruncateTables(t, ds)

	mysql.SetTestABMAssets(t, ds, "fleet")

}

// mockAndroidModule is a mock implementation of the android.Service interface for testing.
type mockAndroidModule struct {
	android.Service

	buildFleetAgentApplicationPolicyFunc func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error)
	setAppsForAndroidPolicyFunc          func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error
}

func (m *mockAndroidModule) BuildFleetAgentApplicationPolicy(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
	if m.buildFleetAgentApplicationPolicyFunc != nil {
		return m.buildFleetAgentApplicationPolicyFunc(ctx, hostUUID)
	}
	return nil, nil
}

func (m *mockAndroidModule) SetAppsForAndroidPolicy(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
	if m.setAppsForAndroidPolicyFunc != nil {
		return m.setAppsForAndroidPolicyFunc(ctx, enterpriseName, appPolicies, hostUUIDs)
	}
	return nil
}

// TestBulkSetAndroidAppsAvailableForHostsPreservesFleetAgent verifies that the Fleet Agent
// is preserved when an Android host is transferred between teams. This prevents the agent
// from being uninstalled (and losing state) during team transfers.
func TestBulkSetAndroidAppsAvailableForHostsPreservesFleetAgent(t *testing.T) {
	ctx := t.Context()
	hostUUID := "test-host-uuid"
	hostID := uint(1)
	teamID := uint(2)

	ds := new(mock.Store)
	ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, uuid string) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{
			Host: &fleet.Host{
				ID:     hostID,
				UUID:   hostUUID,
				TeamID: ptr.Uint(teamID),
			},
		}, nil
	}
	ds.SetHostCertificateTemplatesToPendingRemoveForHostFunc = func(ctx context.Context, hostUUID string) error {
		return nil
	}
	ds.CreatePendingCertificateTemplatesForNewHostFunc = func(ctx context.Context, hostUUID string, teamID uint) (int64, error) {
		return 0, nil
	}
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]string, error) {
		return []string{"com.example.teamapp"}, nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, appIDs []string, globalOrTeamID uint) (map[string]json.RawMessage, error) {
		return map[string]json.RawMessage{}, nil
	}

	var capturedAppPolicies []*androidmanagement.ApplicationPolicy
	androidModule := &mockAndroidModule{
		buildFleetAgentApplicationPolicyFunc: func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
			return &androidmanagement.ApplicationPolicy{
				PackageName: "com.fleetdm.agent",
				InstallType: "FORCE_INSTALLED",
			}, nil
		},
		setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
			capturedAppPolicies = appPolicies
			return nil
		},
	}

	worker := &SoftwareWorker{
		Datastore:     ds,
		AndroidModule: androidModule,
		Log:           slog.New(slog.DiscardHandler),
	}

	err := worker.bulkSetAndroidAppsAvailableForHosts(ctx, map[string]uint{hostUUID: hostID}, "enterprises/test")
	require.NoError(t, err)

	// Verify both the team app and Fleet Agent are in the policy
	require.Len(t, capturedAppPolicies, 2, "expected team app + Fleet Agent")
	capturedPackageNames := make([]string, len(capturedAppPolicies))
	for i, policy := range capturedAppPolicies {
		capturedPackageNames[i] = policy.PackageName
	}
	require.ElementsMatch(t, []string{"com.example.teamapp", "com.fleetdm.agent"}, capturedPackageNames)
}

// TestBulkMakeAndroidAppsAvailableForHostPreservesFleetAgent verifies that the Fleet Agent
// is preserved when BatchAssociateVPPApps updates Android apps for a host.
// This is the singular version called from BatchAssociateVPPApps.
func TestBulkMakeAndroidAppsAvailableForHostPreservesFleetAgent(t *testing.T) {
	ctx := t.Context()
	hostUUID := "test-host-uuid"
	policyID := "test-policy-id"
	teamID := uint(2)

	ds := new(mock.Store)
	ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, uuid string) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{
			Host: &fleet.Host{
				UUID:   hostUUID,
				TeamID: ptr.Uint(teamID),
			},
		}, nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, appIDs []string, globalOrTeamID uint) (map[string]json.RawMessage, error) {
		return map[string]json.RawMessage{}, nil
	}

	var capturedAppPolicies []*androidmanagement.ApplicationPolicy
	androidModule := &mockAndroidModule{
		buildFleetAgentApplicationPolicyFunc: func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
			return &androidmanagement.ApplicationPolicy{
				PackageName: "com.fleetdm.agent",
				InstallType: "FORCE_INSTALLED",
			}, nil
		},
		setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
			capturedAppPolicies = appPolicies
			return nil
		},
	}

	worker := &SoftwareWorker{
		Datastore:     ds,
		AndroidModule: androidModule,
		Log:           slog.New(slog.DiscardHandler),
	}

	// Simulate adding a VPP app via BatchAssociateVPPApps
	applicationIDs := []string{"com.example.vppapp"}
	err := worker.bulkMakeAndroidAppsAvailableForHost(ctx, hostUUID, policyID, applicationIDs, "enterprises/test")
	require.NoError(t, err)

	// Verify both the VPP app and Fleet Agent are in the policy
	require.Len(t, capturedAppPolicies, 2, "expected VPP app + Fleet Agent")
	capturedPackageNames := make([]string, len(capturedAppPolicies))
	for i, policy := range capturedAppPolicies {
		capturedPackageNames[i] = policy.PackageName
	}
	require.ElementsMatch(t, []string{"com.example.vppapp", "com.fleetdm.agent"}, capturedPackageNames)
}

// TestBuildApplicationPolicyWithConfig verifies that buildApplicationPolicyWithConfig
// respects the installType stored in the app configuration and falls back to the
// provided default when no installType is configured.
func TestBuildApplicationPolicyWithConfig(t *testing.T) {
	ctx := t.Context()

	t.Run("uses default installType when config has none", func(t *testing.T) {
		configs := map[string]json.RawMessage{
			"com.example.app": json.RawMessage(`{"managedConfiguration": {"key": "value"}}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.app"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "AVAILABLE", policies[0].InstallType)
		require.Equal(t, "com.example.app", policies[0].PackageName)
	})

	t.Run("config installType overrides default", func(t *testing.T) {
		configs := map[string]json.RawMessage{
			"com.tailscale.ipn": json.RawMessage(`{"installType": "FORCE_INSTALLED"}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.tailscale.ipn"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "FORCE_INSTALLED", policies[0].InstallType)
		require.Equal(t, "com.tailscale.ipn", policies[0].PackageName)
	})

	t.Run("empty installType in config uses PREINSTALLED default for setup experience", func(t *testing.T) {
		configs := map[string]json.RawMessage{
			"com.example.setup": json.RawMessage(`{"installType": ""}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.setup"}, configs, "PREINSTALLED")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "PREINSTALLED", policies[0].InstallType)
	})

	t.Run("config installType overrides PREINSTALLED default", func(t *testing.T) {
		configs := map[string]json.RawMessage{
			"com.example.setup": json.RawMessage(`{"installType": "FORCE_INSTALLED"}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.setup"}, configs, "PREINSTALLED")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "FORCE_INSTALLED", policies[0].InstallType)
	})

	t.Run("nil config map uses default installType", func(t *testing.T) {
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.app"}, nil, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "AVAILABLE", policies[0].InstallType)
		// no-config path should also clear workProfileWidgets
		require.Equal(t, "WORK_PROFILE_WIDGETS_UNSPECIFIED", policies[0].WorkProfileWidgets)
	})

	t.Run("mixed apps: some with installType, some without", func(t *testing.T) {
		configs := map[string]json.RawMessage{
			"com.tailscale.ipn": json.RawMessage(`{"installType": "FORCE_INSTALLED"}`),
			"com.example.slack": json.RawMessage(`{"managedConfiguration": {"domain": "example.com"}}`),
		}
		appIDs := []string{"com.tailscale.ipn", "com.example.slack"}
		policies, err := buildApplicationPolicyWithConfig(ctx, appIDs, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 2)

		byPackage := make(map[string]*androidmanagement.ApplicationPolicy, 2)
		for _, p := range policies {
			byPackage[p.PackageName] = p
		}
		require.Equal(t, "FORCE_INSTALLED", byPackage["com.tailscale.ipn"].InstallType)
		require.Equal(t, "AVAILABLE", byPackage["com.example.slack"].InstallType)
	})
}
