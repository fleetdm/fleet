package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func TestSoftwareWorker(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	// call TruncateTables immediately as some DB migrations may create jobs
	mysqltest.TruncateTables(t, ds)

	mysqltest.SetTestABMAssets(t, ds, "fleet")
}

// mockAndroidModule is a mock implementation of the android.Service interface for testing.
type mockAndroidModule struct {
	android.Service

	buildFleetAgentApplicationPolicyFunc func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error)
	setAppsForAndroidPolicyFunc          func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error
	addAppsToAndroidPolicyFunc           func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error)
	removeAppsFromAndroidPolicyFunc      func(ctx context.Context, enterpriseName string, packageNames []string, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error)
	patchPolicyFunc                      func(ctx context.Context, policyID, policyName string, policy *androidmanagement.Policy, metadata map[string]string) (bool, error)
	patchDeviceFunc                      func(ctx context.Context, policyID, deviceName string, device *androidmanagement.Device) (bool, error)
}

func (m *mockAndroidModule) PatchPolicy(ctx context.Context, policyID, policyName string, policy *androidmanagement.Policy, metadata map[string]string) (bool, error) {
	return m.patchPolicyFunc(ctx, policyID, policyName, policy, metadata)
}

func (m *mockAndroidModule) PatchDevice(ctx context.Context, policyID, deviceName string, device *androidmanagement.Device) (bool, error) {
	return m.patchDeviceFunc(ctx, policyID, deviceName, device)
}

func (m *mockAndroidModule) BuildAndSendFleetAgentConfig(ctx context.Context, enterpriseName string, hostUUIDs []string, skipHostsWithoutNewCerts bool) error {
	return nil
}

func (m *mockAndroidModule) RemoveAppsFromAndroidPolicy(ctx context.Context, enterpriseName string, packageNames []string, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
	return m.removeAppsFromAndroidPolicyFunc(ctx, enterpriseName, packageNames, hostUUIDs)
}

func (m *mockAndroidModule) BuildFleetAgentApplicationPolicy(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
	if m.buildFleetAgentApplicationPolicyFunc != nil {
		return m.buildFleetAgentApplicationPolicyFunc(ctx, hostUUID)
	}
	return nil, nil
}

func (m *mockAndroidModule) AddAppsToAndroidPolicy(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
	if m.addAppsToAndroidPolicyFunc != nil {
		return m.addAppsToAndroidPolicyFunc(ctx, enterpriseName, appPolicies, hostUUIDs)
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
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
		return []fleet.VPPAppTeam{{AdamID: "com.example.teamapp", AppTeamID: 1}}, nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
		return map[string][]byte{}, nil
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
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
		return []fleet.VPPAppTeam{{AdamID: "com.example.vppapp", AppTeamID: 1}}, nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
		return map[string][]byte{}, nil
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

func TestSoftwareWorkerRunDisablesAMAPIRetry(t *testing.T) {
	var called bool
	androidModule := &mockAndroidModule{
		removeAppsFromAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, packageNames []string, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
			called = true
			assert.True(t, androidmgmt.RetryDisabled(ctx), "worker jobs must not wait out AMAPI quota errors")
			return nil, nil
		},
	}
	w := &SoftwareWorker{Datastore: new(mock.Store), AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	args, err := json.Marshal(softwareWorkerArgs{
		Task:               makeAndroidAppUnavailableTask,
		ApplicationID:      "com.example.app",
		EnterpriseName:     "enterprises/test",
		HostUUIDToPolicyID: map[string]string{"host-1": "host-1"},
	})
	require.NoError(t, err)
	require.NoError(t, w.Run(t.Context(), args))
	require.True(t, called)
}

func TestSplitHostMap(t *testing.T) {
	t.Run("no batching when batchSize is 0", func(t *testing.T) {
		hosts := map[string]string{"a": "1", "b": "2", "c": "3"}
		batches := splitHostMap(hosts, 0)
		require.Len(t, batches, 1)
		require.Len(t, batches[0], 3)
	})

	t.Run("no batching when fewer than batchSize", func(t *testing.T) {
		hosts := map[string]string{"a": "1", "b": "2"}
		batches := splitHostMap(hosts, 5)
		require.Len(t, batches, 1)
		require.Len(t, batches[0], 2)
	})

	t.Run("splits into correct number of batches", func(t *testing.T) {
		hosts := make(map[string]string, 5)
		for i := range 5 {
			hosts[fmt.Sprintf("host-%d", i)] = fmt.Sprintf("policy-%d", i)
		}
		batches := splitHostMap(hosts, 2)
		require.Len(t, batches, 3) // 2 + 2 + 1

		// Verify all hosts are covered with no duplicates.
		seen := make(map[string]struct{})
		for _, batch := range batches {
			for k := range batch {
				_, dup := seen[k]
				assert.False(t, dup, "duplicate host %s", k)
				seen[k] = struct{}{}
			}
		}
		require.Len(t, seen, 5)
	})

	t.Run("exact multiple", func(t *testing.T) {
		hosts := make(map[string]string, 4)
		for i := range 4 {
			hosts[fmt.Sprintf("host-%d", i)] = fmt.Sprintf("policy-%d", i)
		}
		batches := splitHostMap(hosts, 2)
		require.Len(t, batches, 2)
		require.Len(t, batches[0], 2)
		require.Len(t, batches[1], 2)
	})
}

func TestMakeAndroidAppAvailableBatching(t *testing.T) {
	ds := new(mock.Store)

	ds.GetAppStoreAppVersionIDsFromSpecificVersionFunc = func(ctx context.Context, vppAppTeamID uint) ([]uint, error) {
		return []uint{vppAppTeamID}, nil
	}
	// 5 hosts in scope
	ds.GetIncludedHostUUIDMapForAppStoreAppFunc = func(ctx context.Context, appTeamID uint) (map[string]string, error) {
		hosts := make(map[string]string, 5)
		for i := range 5 {
			hosts[fmt.Sprintf("host-%d", i)] = fmt.Sprintf("host-%d", i)
		}
		return hosts, nil
	}
	ds.GetAndroidAppConfigurationByAppTeamIDFunc = func(ctx context.Context, appTeamID uint) ([]byte, error) {
		return nil, nil // no config, no variables
	}

	var jobs []*fleet.Job
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		job.ID = uint(len(jobs) + 1)
		jobs = append(jobs, job)
		return job, nil
	}

	w := &SoftwareWorker{
		Datastore:        ds,
		AndroidModule:    &mockAndroidModule{},
		Log:              slog.New(slog.DiscardHandler),
		AndroidBatchSize: 2, // batch size of 2 → 3 batches (2+2+1)
	}

	err := w.makeAndroidAppAvailable(t.Context(), "com.example.app", 1, "enterprises/test", false, false)
	require.NoError(t, err)

	// Phase 1 should queue 3 batch jobs (2+2+1 hosts), no AMAPI calls.
	require.Len(t, jobs, 3, "expected 3 batch jobs queued")

	// Verify staggered delays: 0s, 60s, 120s
	for i, job := range jobs {
		var args softwareWorkerArgs
		require.NoError(t, json.Unmarshal(*job.Args, &args))
		require.Equal(t, makeAndroidAppAvailableBatchTask, args.Task)
		require.Equal(t, "com.example.app", args.ApplicationID)
		require.Equal(t, "enterprises/test", args.EnterpriseName)

		if i == 0 {
			require.True(t, job.NotBefore.IsZero(), "first batch should have no delay")
		} else {
			expectedDelay := time.Duration(i) * androidSoftwareInstallStaggerInterval
			require.WithinDuration(t, time.Now().Add(expectedDelay), job.NotBefore, 5*time.Second,
				"batch %d should be delayed by %s", i, expectedDelay)
		}
	}

	// Count total hosts across all batches
	totalHosts := 0
	for _, job := range jobs {
		var args softwareWorkerArgs
		require.NoError(t, json.Unmarshal(*job.Args, &args))
		totalHosts += len(args.HostUUIDToPolicyID)
	}
	require.Equal(t, 5, totalHosts, "all 5 hosts should be distributed across batches")
}

func TestMakeAndroidAppAvailableBatchNoVars(t *testing.T) {
	var addAppsCalled bool
	var capturedHosts map[string]string

	androidModule := &mockAndroidModule{
		addAppsToAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
			addAppsCalled = true
			capturedHosts = hostUUIDs
			result := make(map[string]*android.MDMAndroidPolicyRequest)
			for uuid := range hostUUIDs {
				result[uuid] = &android.MDMAndroidPolicyRequest{PolicyVersion: sql.Null[int64]{V: 42, Valid: true}}
			}
			return result, nil
		},
	}

	ds := new(mock.Store)
	ds.GetAndroidAppConfigurationByAppTeamIDFunc = func(ctx context.Context, appTeamID uint) ([]byte, error) {
		return nil, nil // no config
	}

	var pendingConfigs []string
	ds.SetAndroidAppInstallPendingApplyConfigFunc = func(ctx context.Context, hostUUID, applicationID string, policyVersion int64) error {
		pendingConfigs = append(pendingConfigs, hostUUID)
		return nil
	}

	w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	hosts := map[string]string{"host-1": "host-1", "host-2": "host-2"}
	err := w.makeAndroidAppAvailableBatch(t.Context(), "com.example.app", 1, hosts, "enterprises/test", true)
	require.NoError(t, err)

	require.True(t, addAppsCalled, "should call AddAppsToAndroidPolicy")
	require.Equal(t, hosts, capturedHosts, "all hosts should be sent in one call")
	require.Len(t, pendingConfigs, 2, "appConfigChanged=true should update both hosts")
}

func TestMakeAndroidAppAvailableBatchWithVars(t *testing.T) {
	// Capture per-host AMAPI calls: host UUID → rendered managed config
	capturedConfigByHost := make(map[string]string)

	androidModule := &mockAndroidModule{
		addAppsToAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
			// Each call should target exactly one host (per-host substitution)
			require.Len(t, hostUUIDs, 1)
			require.Len(t, appPolicies, 1)
			for uuid := range hostUUIDs {
				capturedConfigByHost[uuid] = string(appPolicies[0].ManagedConfiguration)
			}
			result := make(map[string]*android.MDMAndroidPolicyRequest)
			for uuid := range hostUUIDs {
				result[uuid] = &android.MDMAndroidPolicyRequest{PolicyVersion: sql.Null[int64]{V: 1, Valid: true}}
			}
			return result, nil
		},
	}

	ds := new(mock.Store)
	ds.GetAndroidAppConfigurationByAppTeamIDFunc = func(ctx context.Context, appTeamID uint) ([]byte, error) {
		return []byte(`{"managedConfiguration": {"deviceId": "$FLEET_VAR_HOST_UUID"}}`), nil
	}
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		var hosts []*fleet.Host
		for _, uuid := range uuids {
			hosts = append(hosts, &fleet.Host{UUID: uuid, Platform: "android", HardwareSerial: "SN-" + uuid})
		}
		return hosts, nil
	}
	ds.SetAndroidAppInstallPendingApplyConfigFunc = func(ctx context.Context, hostUUID, applicationID string, policyVersion int64) error {
		return nil
	}

	w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	hosts := map[string]string{"uuid-aaa": "uuid-aaa", "uuid-bbb": "uuid-bbb"}
	err := w.makeAndroidAppAvailableBatch(t.Context(), "com.example.app", 1, hosts, "enterprises/test", false)
	require.NoError(t, err)

	require.Len(t, capturedConfigByHost, 2, "should have called AMAPI for both hosts")
	require.Contains(t, capturedConfigByHost["uuid-aaa"], "uuid-aaa", "host uuid-aaa should appear in its config")
	require.NotContains(t, capturedConfigByHost["uuid-aaa"], "$FLEET_VAR_HOST_UUID")
	require.Contains(t, capturedConfigByHost["uuid-bbb"], "uuid-bbb", "host uuid-bbb should appear in its config")
	require.NotContains(t, capturedConfigByHost["uuid-bbb"], "$FLEET_VAR_HOST_UUID")
}

// A host that can't supply a referenced value must not take the rest of its
// batch down with it: the other hosts still get their app config, and the job
// succeeds rather than burning its retries.
func TestMakeAndroidAppAvailableBatchSkipsHostMissingVitalValue(t *testing.T) {
	pushedConfigByHost := map[string]string{}

	androidModule := &mockAndroidModule{
		addAppsToAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
			require.Len(t, hostUUIDs, 1)
			result := make(map[string]*android.MDMAndroidPolicyRequest)
			for uuid := range hostUUIDs {
				pushedConfigByHost[uuid] = string(appPolicies[0].ManagedConfiguration)
				result[uuid] = &android.MDMAndroidPolicyRequest{PolicyVersion: sql.Null[int64]{V: 1, Valid: true}}
			}
			return result, nil
		},
	}

	hostIDByUUID := map[string]uint{"uuid-has-value": 1, "uuid-no-value": 2}

	ds := new(mock.Store)
	ds.GetAndroidAppConfigurationByAppTeamIDFunc = func(ctx context.Context, appTeamID uint) ([]byte, error) {
		return []byte(`{"managedConfiguration":{"assetTag":"$FLEET_HOST_VITAL_7"}}`), nil
	}
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		var hosts []*fleet.Host
		for _, uuid := range uuids {
			hosts = append(hosts, &fleet.Host{ID: hostIDByUUID[uuid], UUID: uuid, Platform: "android"})
		}
		return hosts, nil
	}
	ds.ExpandCustomHostVitalsFunc = func(ctx context.Context, hostID uint, document string) (string, error) {
		if hostID == hostIDByUUID["uuid-no-value"] {
			return "", &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{7}, MissingNames: []string{"Asset tag"}}
		}
		return strings.ReplaceAll(document, "$FLEET_HOST_VITAL_7", "ASSET-1"), nil
	}
	var pendingApplyHosts []string
	ds.SetAndroidAppInstallPendingApplyConfigFunc = func(ctx context.Context, hostUUID, applicationID string, policyVersion int64) error {
		pendingApplyHosts = append(pendingApplyHosts, hostUUID)
		return nil
	}

	w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	hosts := map[string]string{"uuid-has-value": "pol-1", "uuid-no-value": "pol-2"}
	err := w.makeAndroidAppAvailableBatch(t.Context(), "com.example.app", 1, hosts, "enterprises/test", true)
	require.NoError(t, err, "a host with no value for the vital must not fail the whole batch")

	require.Contains(t, pushedConfigByHost, "uuid-has-value")
	require.Contains(t, pushedConfigByHost["uuid-has-value"], "ASSET-1")
	require.NotContains(t, pushedConfigByHost, "uuid-no-value", "host with no value for the vital should be skipped")
	require.Equal(t, []string{"uuid-has-value"}, pendingApplyHosts)
}

func TestIsUnresolvableAndroidAppConfigForHost(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"missing vital value", &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{1}}, true},
		{"wrapped missing vital value", fmt.Errorf("substitute: %w", &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{1}}), true},
		{"unresolvable fleet var", &profiles.UnresolvableAndroidAppConfigVarError{FleetVar: "HOST_END_USER_IDP_USERNAME"}, true},
		{"wrapped unresolvable fleet var", fmt.Errorf("substitute: %w", &profiles.UnresolvableAndroidAppConfigVarError{FleetVar: "HOST_END_USER_IDP_USERNAME"}), true},
		{"unrelated error", errors.New("db is down"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, isUnresolvableAndroidAppConfigForHost(c.err))
		})
	}
}

func TestQueueBulkSetAndroidAppsAvailableForHostsChunking(t *testing.T) {
	ds := new(mock.Store)

	var jobs []*fleet.Job
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		job.ID = uint(len(jobs) + 1)
		jobs = append(jobs, job)
		return job, nil
	}

	hosts := make(map[string]uint, 5)
	for i := range 5 {
		hosts[fmt.Sprintf("host-%d", i)] = uint(i)
	}

	err := QueueBulkSetAndroidAppsAvailableForHosts(
		t.Context(), ds, slog.New(slog.DiscardHandler),
		hosts, "enterprises/test", 2, // batch size 2
	)
	require.NoError(t, err)

	// 5 hosts / batch size 2 = 3 jobs
	require.Len(t, jobs, 3)

	// First job should have no delay (not_before ≈ zero).
	assert.True(t, jobs[0].NotBefore.IsZero() || jobs[0].NotBefore.Before(time.Now()),
		"first job should be immediately available")

	// Subsequent jobs should have increasing not_before.
	for i := 1; i < len(jobs); i++ {
		assert.True(t, jobs[i].NotBefore.After(jobs[i-1].NotBefore),
			"job %d should have later not_before than job %d", i, i-1)
	}

	// Verify all hosts are covered.
	totalHosts := 0
	for _, job := range jobs {
		var args softwareWorkerArgs
		require.NoError(t, json.Unmarshal(*job.Args, &args))
		totalHosts += len(args.UUIDsToIDs)
	}
	require.Equal(t, 5, totalHosts)
}

func TestQueueMakeAndroidAppUnavailableJobChunking(t *testing.T) {
	ds := new(mock.Store)

	var jobs []*fleet.Job
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		job.ID = uint(len(jobs) + 1)
		jobs = append(jobs, job)
		return job, nil
	}

	hosts := make(map[string]string, 5)
	for i := range 5 {
		hosts[fmt.Sprintf("host-%d", i)] = fmt.Sprintf("policy-%d", i)
	}

	err := QueueMakeAndroidAppUnavailableJob(
		t.Context(), ds, slog.New(slog.DiscardHandler),
		"com.example.app", hosts, "enterprises/test", 2,
	)
	require.NoError(t, err)

	// 5 hosts / batch size 2 = 3 jobs
	require.Len(t, jobs, 3)

	// Verify staggering.
	assert.True(t, jobs[0].NotBefore.IsZero() || jobs[0].NotBefore.Before(time.Now()))
	for i := 1; i < len(jobs); i++ {
		assert.True(t, jobs[i].NotBefore.After(jobs[i-1].NotBefore))
	}

	// Verify all hosts are covered.
	totalHosts := 0
	for _, job := range jobs {
		var args softwareWorkerArgs
		require.NoError(t, json.Unmarshal(*job.Args, &args))
		totalHosts += len(args.HostUUIDToPolicyID)
	}
	require.Equal(t, 5, totalHosts)
}

func TestBuildApplicationPolicyWithConfig(t *testing.T) {
	ctx := t.Context()

	t.Run("app with full configuration", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"managedConfiguration": {"key": "value"}, "workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED", "credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED"}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.app"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "com.example.app", policies[0].PackageName)
		require.Equal(t, "AVAILABLE", policies[0].InstallType)
		require.JSONEq(t, `{"key": "value"}`, string(policies[0].ManagedConfiguration))
		require.Equal(t, "WORK_PROFILE_WIDGETS_ALLOWED", policies[0].WorkProfileWidgets)
		require.Equal(t, "CREDENTIAL_PROVIDER_ALLOWED", policies[0].CredentialProviderPolicy)
	})

	t.Run("app without configuration clears previously applied settings", func(t *testing.T) {
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.app"}, nil, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Empty(t, policies[0].ManagedConfiguration)
		require.Equal(t, "WORK_PROFILE_WIDGETS_UNSPECIFIED", policies[0].WorkProfileWidgets)
		require.Equal(t, "CREDENTIAL_PROVIDER_POLICY_UNSPECIFIED", policies[0].CredentialProviderPolicy)
	})

	t.Run("partial configuration leaves other fields unset", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"credentialProviderPolicy": "CREDENTIAL_PROVIDER_ALLOWED"}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.app"}, configs, "FORCE_INSTALLED")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, "FORCE_INSTALLED", policies[0].InstallType)
		require.Empty(t, policies[0].WorkProfileWidgets)
		require.Equal(t, "CREDENTIAL_PROVIDER_ALLOWED", policies[0].CredentialProviderPolicy)
	})

	t.Run("other ApplicationPolicy fields are passed through", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"autoUpdateMode": "AUTO_UPDATE_HIGH_PRIORITY", "minimumVersionCode": 42, "permissionGrants": [{"permission": "android.permission.CAMERA", "policy": "GRANT"}]}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.app"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, "AUTO_UPDATE_HIGH_PRIORITY", policies[0].AutoUpdateMode)
		assert.EqualValues(t, 42, policies[0].MinimumVersionCode)
		require.Len(t, policies[0].PermissionGrants, 1)
		assert.Equal(t, "android.permission.CAMERA", policies[0].PermissionGrants[0].Permission)
		assert.Equal(t, "GRANT", policies[0].PermissionGrants[0].Policy)
	})

	t.Run("stored packageName and installType never override Fleet's values", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.app": []byte(`{"packageName": "com.example.other", "installType": "KIOSK"}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.app"}, configs, "PREINSTALLED")
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, "com.example.app", policies[0].PackageName)
		assert.Equal(t, "PREINSTALLED", policies[0].InstallType)
	})

	t.Run("each app gets its own configuration", func(t *testing.T) {
		configs := map[string][]byte{
			"com.example.configured": []byte(`{"autoUpdateMode": "AUTO_UPDATE_POSTPONED"}`),
		}
		policies, err := buildApplicationPolicyWithConfig(ctx, []string{"com.example.configured", "com.example.unconfigured"}, configs, "AVAILABLE")
		require.NoError(t, err)
		require.Len(t, policies, 2)
		assert.Equal(t, "com.example.configured", policies[0].PackageName)
		assert.Equal(t, "AUTO_UPDATE_POSTPONED", policies[0].AutoUpdateMode)
		assert.Empty(t, policies[0].WorkProfileWidgets)
		assert.Equal(t, "com.example.unconfigured", policies[1].PackageName)
		assert.Empty(t, policies[1].AutoUpdateMode)
		assert.Equal(t, "WORK_PROFILE_WIDGETS_UNSPECIFIED", policies[1].WorkProfileWidgets)
	})
}

func TestMakeAndroidAppAvailableSendsEachHostItsVersionConfiguration(t *testing.T) {
	const (
		versionAID = uint(1)
		versionBID = uint(2)
	)
	configA := `{"version":"a"}`
	configB := `{"version":"b"}`

	versionIDs := []uint{versionAID, versionBID}
	ds := new(mock.Store)
	ds.GetAppStoreAppVersionIDsFromSpecificVersionFunc = func(ctx context.Context, vppAppTeamID uint) ([]uint, error) {
		return versionIDs, nil
	}
	ds.GetIncludedHostUUIDMapForAppStoreAppFunc = func(ctx context.Context, appTeamID uint) (map[string]string, error) {
		if appTeamID == versionAID {
			return map[string]string{"host-in-a-and-b": "host-in-a-and-b"}, nil
		}
		return map[string]string{"host-in-a-and-b": "host-in-a-and-b", "host-in-b": "host-in-b"}, nil
	}
	ds.GetAndroidAppConfigurationByAppTeamIDFunc = func(ctx context.Context, appTeamID uint) ([]byte, error) {
		if appTeamID == versionAID {
			return []byte(`{"managedConfiguration":` + configA + `}`), nil
		}
		return []byte(`{"managedConfiguration":` + configB + `}`), nil
	}
	var jobs []*fleet.Job
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		jobs = append(jobs, job)
		return job, nil
	}

	policiesByHost := make(map[string][]*androidmanagement.ApplicationPolicy)
	androidModule := &mockAndroidModule{
		addAppsToAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
			for hostUUID := range hostUUIDs {
				policiesByHost[hostUUID] = append(policiesByHost[hostUUID], appPolicies...)
			}
			return nil, nil
		},
	}
	w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	// edit version A's labels and run the queued jobs, each host should get one policy entry with its own version's configuration
	err := w.makeAndroidAppAvailable(t.Context(), "com.example.app", versionAID, "enterprises/test", false, true)
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	for _, job := range jobs {
		err = w.Run(t.Context(), *job.Args)
		require.NoError(t, err)
	}
	require.Len(t, policiesByHost, 2)
	require.Len(t, policiesByHost["host-in-a-and-b"], 1)
	require.Equal(t, "com.example.app", policiesByHost["host-in-a-and-b"][0].PackageName)
	require.JSONEq(t, configA, string(policiesByHost["host-in-a-and-b"][0].ManagedConfiguration))
	require.Len(t, policiesByHost["host-in-b"], 1)
	require.Equal(t, "com.example.app", policiesByHost["host-in-b"][0].PackageName)
	require.JSONEq(t, configB, string(policiesByHost["host-in-b"][0].ManagedConfiguration))

	// remove version A and push version B, the host in A and B should move to version B's configuration
	versionIDs = []uint{versionBID}
	jobs = nil
	policiesByHost = make(map[string][]*androidmanagement.ApplicationPolicy)
	err = w.makeAndroidAppAvailable(t.Context(), "com.example.app", versionBID, "enterprises/test", false, true)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	err = w.Run(t.Context(), *jobs[0].Args)
	require.NoError(t, err)
	require.Len(t, policiesByHost, 2)
	require.Len(t, policiesByHost["host-in-a-and-b"], 1)
	require.JSONEq(t, configB, string(policiesByHost["host-in-a-and-b"][0].ManagedConfiguration))
	require.Len(t, policiesByHost["host-in-b"], 1)
	require.JSONEq(t, configB, string(policiesByHost["host-in-b"][0].ManagedConfiguration))

	// edit only version B's configuration with both versions present, only the host whose version is B should be pushed
	versionIDs = []uint{versionAID, versionBID}
	jobs = nil
	policiesByHost = make(map[string][]*androidmanagement.ApplicationPolicy)
	err = w.makeAndroidAppAvailable(t.Context(), "com.example.app", versionBID, "enterprises/test", true, false)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	err = w.Run(t.Context(), *jobs[0].Args)
	require.NoError(t, err)
	require.Len(t, policiesByHost, 1)
	require.Len(t, policiesByHost["host-in-b"], 1)
	require.JSONEq(t, configB, string(policiesByHost["host-in-b"][0].ManagedConfiguration))
}

func TestRunAndroidSetupExperienceInstallsFirstAddedFlaggedVersion(t *testing.T) {
	const hostUUID = "setup-host-uuid"

	ds := new(mock.Store)
	ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, uuid string) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{Host: &fleet.Host{ID: 1, UUID: hostUUID}, Device: &android.Device{AppliedPolicyID: new(hostUUID)}}, nil
	}
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
		return nil, nil
	}
	ds.GetVPPAppsToInstallDuringSetupExperienceFunc = func(ctx context.Context, teamID *uint, platform string) ([]fleet.VPPAppTeam, error) {
		return []fleet.VPPAppTeam{
			{AdamID: "com.example.app", Platform: fleet.AndroidPlatform, AppTeamID: 1},
			{AdamID: "com.example.app", Platform: fleet.AndroidPlatform, AppTeamID: 2},
		}, nil
	}
	var configVersionIDs []uint
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
		configVersionIDs = vppAppTeamIDs
		return map[string][]byte{}, nil
	}
	var installs []*fleet.HostAndroidVPPSoftwareInstall
	ds.InsertAndroidSetupExperienceSoftwareInstallFunc = func(ctx context.Context, payload *fleet.HostAndroidVPPSoftwareInstall) error {
		installs = append(installs, payload)
		return nil
	}

	var appPolicies []*androidmanagement.ApplicationPolicy
	androidModule := &mockAndroidModule{
		addAppsToAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, policies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
			appPolicies = append(appPolicies, policies...)
			return map[string]*android.MDMAndroidPolicyRequest{hostUUID: {PolicyVersion: sql.Null[int64]{V: 1, Valid: true}}}, nil
		},
	}
	w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	// run the setup experience with two flagged versions of one app, only the first-added version should be installed
	err := w.runAndroidSetupExperience(t.Context(), hostUUID, 0, "enterprises/test")
	require.NoError(t, err)
	require.Len(t, appPolicies, 1)
	require.Equal(t, "com.example.app", appPolicies[0].PackageName)
	require.Equal(t, []uint{1}, configVersionIDs)
	require.Len(t, installs, 1)
	require.Equal(t, uint(1), installs[0].VPPAppTeamID)
}

// A re-enrolled host reuses its host-specific policy, so making its fleet's apps available
// must replace the policy's app list, dropping the apps of the previous enrollment.
func TestMakeAndroidAppsAvailableForHostReplacesAppList(t *testing.T) {
	const (
		hostUUID       = "host-uuid"
		enterpriseName = "enterprises/test"
		agentPackage   = "com.fleetdm.agent"
	)

	cases := []struct {
		name         string
		appsInScope  []fleet.VPPAppTeam
		buildAgent   func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error)
		wantErr      bool
		wantPackages []string
	}{
		{
			name:         "fleet with apps",
			appsInScope:  []fleet.VPPAppTeam{{AdamID: "com.example.one", AppTeamID: 1}, {AdamID: "com.example.two", AppTeamID: 2}},
			wantPackages: []string{"com.example.one", "com.example.two", agentPackage},
		},
		{
			name:         "fleet without apps",
			wantPackages: []string{agentPackage},
		},
		{
			name:        "fleet agent not configured",
			appsInScope: []fleet.VPPAppTeam{{AdamID: "com.example.one", AppTeamID: 1}},
			buildAgent: func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
				return nil, nil
			},
			wantPackages: []string{"com.example.one"},
		},
		{
			name: "fleet without apps and fleet agent not configured",
			buildAgent: func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
				return nil, nil
			},
			wantPackages: nil,
		},
		{
			name:        "fleet agent policy fails to build",
			appsInScope: []fleet.VPPAppTeam{{AdamID: "com.example.one", AppTeamID: 1}},
			buildAgent: func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
				return nil, errors.New("no enroll secret")
			},
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, uuid string) (*fleet.AndroidHost, error) {
				return &fleet.AndroidHost{Host: &fleet.Host{ID: 1, UUID: uuid}, Device: &android.Device{DeviceID: "device-id"}}, nil
			}
			ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
				return c.appsInScope, nil
			}
			ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
				return map[string][]byte{}, nil
			}

			buildAgent := c.buildAgent
			if buildAgent == nil {
				buildAgent = func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
					return &androidmanagement.ApplicationPolicy{PackageName: agentPackage, InstallType: "FORCE_INSTALLED"}, nil
				}
			}
			var setCalls int
			var setPackages []string
			var setHosts map[string]string
			androidModule := &mockAndroidModule{
				buildFleetAgentApplicationPolicyFunc: buildAgent,
				setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
					setCalls++
					setHosts = hostUUIDs
					for _, p := range appPolicies {
						setPackages = append(setPackages, p.PackageName)
					}
					return nil
				},
				addAppsToAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
					t.Fatal("apps must replace the host policy's app list, not be added to it")
					return nil, nil
				},
			}
			w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

			err := w.makeAndroidAppsAvailableForHost(t.Context(), hostUUID, 1, enterpriseName, hostUUID)
			if c.wantErr {
				require.Error(t, err)
				require.Zero(t, setCalls)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 1, setCalls)
			require.Equal(t, map[string]string{hostUUID: hostUUID}, setHosts)
			require.Equal(t, c.wantPackages, setPackages)
		})
	}
}

// An app whose configuration references a value the host can't supply must not keep the
// host's other apps from replacing the previous enrollment's, nor the device from moving to
// its host policy.
func TestMakeAndroidAppsAvailableForHostSkipsUnresolvableAppConfig(t *testing.T) {
	const (
		hostUUID       = "host-uuid"
		enterpriseName = "enterprises/test"
	)
	defaultPolicyID := fmt.Sprint(android.DefaultAndroidPolicyID)

	cases := []struct {
		name         string
		expandErr    error
		wantErr      bool
		wantPackages []string
	}{
		{
			name:         "missing custom host vital value",
			expandErr:    &fleet.MissingCustomHostVitalValueError{MissingIDs: []uint{7}, MissingNames: []string{"Asset tag"}},
			wantPackages: []string{"com.example.noconfig", "com.fleetdm.agent"},
		},
		{
			name:      "other substitution error",
			expandErr: errors.New("database is down"),
			wantErr:   true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, uuid string) (*fleet.AndroidHost, error) {
				return &fleet.AndroidHost{Host: &fleet.Host{ID: 1, UUID: uuid, Platform: "android"}, Device: &android.Device{DeviceID: "device-id"}}, nil
			}
			ds.ResetMDMAndroidHostProfilesForRedeliveryFunc = func(ctx context.Context, uuid string) error { return nil }
			ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
				return []fleet.VPPAppTeam{{AdamID: "com.example.vital", AppTeamID: 1}, {AdamID: "com.example.noconfig", AppTeamID: 2}}, nil
			}
			ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
				return map[string][]byte{"com.example.vital": []byte(`{"managedConfiguration":{"assetTag":"$FLEET_HOST_VITAL_7"}}`)}, nil
			}
			ds.ExpandCustomHostVitalsFunc = func(ctx context.Context, hostID uint, document string) (string, error) {
				return "", c.expandErr
			}

			var setPackages []string
			var devicePatched bool
			androidModule := &mockAndroidModule{
				buildFleetAgentApplicationPolicyFunc: func(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
					return &androidmanagement.ApplicationPolicy{PackageName: "com.fleetdm.agent"}, nil
				},
				patchPolicyFunc: func(ctx context.Context, policyID, policyName string, policy *androidmanagement.Policy, metadata map[string]string) (bool, error) {
					return false, nil
				},
				setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
					for _, p := range appPolicies {
						setPackages = append(setPackages, p.PackageName)
					}
					return nil
				},
				patchDeviceFunc: func(ctx context.Context, policyID, deviceName string, device *androidmanagement.Device) (bool, error) {
					devicePatched = true
					return false, nil
				},
			}
			w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

			err := w.makeAndroidAppsAvailableForHost(t.Context(), hostUUID, 1, enterpriseName, defaultPolicyID)
			if c.wantErr {
				require.Error(t, err)
				require.Nil(t, setPackages)
				require.False(t, devicePatched)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.wantPackages, setPackages)
			require.True(t, devicePatched)
		})
	}
}

// Moving a host off the default policy replaces every setting of its host-specific policy,
// which a re-enrolled host reuses and the reconciler may already have filled, so the host's
// profiles must be queued for redelivery after the patch. The device must only be moved to
// that policy once its app list was replaced, so it never sees a previous enrollment's apps.
func TestMakeAndroidAppsAvailableForHostPolicyOrder(t *testing.T) {
	const (
		hostUUID       = "host-uuid"
		enterpriseName = "enterprises/test"
	)
	hostPolicyName := enterpriseName + "/policies/" + hostUUID
	defaultPolicyID := fmt.Sprint(android.DefaultAndroidPolicyID)

	var calls []string
	var patchPolicyErr, setAppsErr error
	androidModule := &mockAndroidModule{
		patchPolicyFunc: func(ctx context.Context, policyID, policyName string, policy *androidmanagement.Policy, metadata map[string]string) (bool, error) {
			calls = append(calls, "patch policy "+policyName)
			return false, patchPolicyErr
		},
		setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
			calls = append(calls, "set apps "+hostUUIDs[hostUUID])
			return setAppsErr
		},
		patchDeviceFunc: func(ctx context.Context, policyID, deviceName string, device *androidmanagement.Device) (bool, error) {
			calls = append(calls, "patch device "+deviceName+" to "+device.PolicyName)
			return false, nil
		},
	}
	ds := new(mock.Store)
	ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, uuid string) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{Host: &fleet.Host{UUID: uuid}, Device: &android.Device{DeviceID: "device-id"}}, nil
	}
	ds.ResetMDMAndroidHostProfilesForRedeliveryFunc = func(ctx context.Context, uuid string) error {
		calls = append(calls, "reset profiles "+uuid)
		return nil
	}
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
		return nil, nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
		return map[string][]byte{}, nil
	}
	w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	t.Run("default policy", func(t *testing.T) {
		calls, patchPolicyErr, setAppsErr = nil, nil, nil
		require.NoError(t, w.makeAndroidAppsAvailableForHost(t.Context(), hostUUID, 1, enterpriseName, defaultPolicyID))
		assert.Equal(t, []string{
			"patch policy " + hostPolicyName,
			"reset profiles " + hostUUID,
			"set apps " + hostUUID,
			"patch device " + enterpriseName + "/devices/device-id to " + hostPolicyName,
		}, calls)
	})

	t.Run("policy patch fails", func(t *testing.T) {
		calls, patchPolicyErr, setAppsErr = nil, errors.New("nope"), nil
		require.Error(t, w.makeAndroidAppsAvailableForHost(t.Context(), hostUUID, 1, enterpriseName, defaultPolicyID))
		assert.Equal(t, []string{"patch policy " + hostPolicyName}, calls)
	})

	t.Run("app list replace fails", func(t *testing.T) {
		calls, patchPolicyErr, setAppsErr = nil, nil, errors.New("nope")
		require.Error(t, w.makeAndroidAppsAvailableForHost(t.Context(), hostUUID, 1, enterpriseName, defaultPolicyID))
		assert.Equal(t, []string{
			"patch policy " + hostPolicyName,
			"reset profiles " + hostUUID,
			"set apps " + hostUUID,
		}, calls, "the device must stay off the host policy")
	})

	t.Run("host policy already applied", func(t *testing.T) {
		calls, patchPolicyErr, setAppsErr = nil, nil, nil
		require.NoError(t, w.makeAndroidAppsAvailableForHost(t.Context(), hostUUID, 1, enterpriseName, hostUUID))
		assert.Equal(t, []string{"set apps " + hostUUID}, calls)
	})
}
