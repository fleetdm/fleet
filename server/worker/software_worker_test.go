package worker

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
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
		Log:           kitlog.NewNopLogger(),
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
