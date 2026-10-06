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
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
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
		return []fleet.VPPAppTeam{{VPPAppID: fleet.VPPAppID{AdamID: "com.example.teamapp"}, AppTeamID: 1}}, nil
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
		return []fleet.VPPAppTeam{{VPPAppID: fleet.VPPAppID{AdamID: "com.example.vppapp"}, AppTeamID: 1}}, nil
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

	err := w.makeAndroidAppAvailable(t.Context(), "com.example.app", 1, "enterprises/test", false)
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

	// edit version A and run the queued jobs, each host should get one policy entry with its own version's configuration
	err := w.makeAndroidAppAvailable(t.Context(), "com.example.app", versionAID, "enterprises/test", false)
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
	err = w.makeAndroidAppAvailable(t.Context(), "com.example.app", versionBID, "enterprises/test", false)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	err = w.Run(t.Context(), *jobs[0].Args)
	require.NoError(t, err)
	require.Len(t, policiesByHost, 2)
	require.Len(t, policiesByHost["host-in-a-and-b"], 1)
	require.JSONEq(t, configB, string(policiesByHost["host-in-a-and-b"][0].ManagedConfiguration))
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
			{VPPAppID: fleet.VPPAppID{AdamID: "com.example.app", Platform: fleet.AndroidPlatform}, AppTeamID: 1},
			{VPPAppID: fleet.VPPAppID{AdamID: "com.example.app", Platform: fleet.AndroidPlatform}, AppTeamID: 2},
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

type resendVPPInstaller struct {
	installErrByHostID map[uint]error
	// installedAppTeamIDByHostID holds the version each successful install used
	installedAppTeamIDByHostID map[uint]uint
}

func (i *resendVPPInstaller) GetVPPTokenIfCanInstallVPPApps(ctx context.Context, appleDevice bool, host *fleet.Host) (string, error) {
	return "vpp-token", nil
}

func (i *resendVPPInstaller) InstallVPPAppPostValidation(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, token string, opts fleet.HostSoftwareInstallOptions) (string, error) {
	err := i.installErrByHostID[host.ID]
	if err != nil {
		return "", err
	}
	i.installedAppTeamIDByHostID[host.ID] = vppApp.AppTeamID
	return "command-uuid", nil
}

func TestResendVPPAppConfiguration(t *testing.T) {
	const (
		titleID  = uint(10)
		versionA = uint(5)
		versionB = uint(6)
		// hostInAAndB has version A, versions are ordered by id
		hostInAAndB         = uint(1)
		hostInB             = uint(2)
		hostWithDeletedInB  = uint(3)
		hostWithDeletedNone = uint(4)
		// hostMovedToB installed version A, then labels moved it to version B
		hostMovedToB           = uint(5)
		hostWithDeletedInAAndB = uint(6)
		hostDeleted            = uint(7)
		hostOnOtherFleet       = uint(8)
	)
	appID := fleet.VPPAppID{AdamID: "1234", Platform: fleet.IOSPlatform}

	var jobs []*fleet.Job
	ds := new(mock.Store)
	ds.GetVPPAppMetadataByAdamIDPlatformTeamIDFunc = func(ctx context.Context, adamID string, platform fleet.InstallableDevicePlatform, teamID *uint) (*fleet.VPPApp, error) {
		return &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: appID}, TitleID: titleID}, nil
	}
	ds.GetAppStoreAppVersionsByTeamAndTitleIDFunc = func(ctx context.Context, teamID uint, titleID uint) ([]*fleet.VPPAppStoreApp, error) {
		return []*fleet.VPPAppStoreApp{{VPPAppID: appID, VPPAppsTeamsID: versionA}, {VPPAppID: appID, VPPAppsTeamsID: versionB}}, nil
	}
	// The hosts named WithDeleted installed a version that was deleted since
	ds.ListHostAppStoreAppInstallVersionsFunc = func(ctx context.Context, appID fleet.VPPAppID, fleetID uint, hostIDs []uint) (map[uint]*uint, error) {
		return map[uint]*uint{
			hostInAAndB: new(versionA), hostInB: new(versionB), hostWithDeletedInB: nil, hostWithDeletedNone: nil,
			hostMovedToB: new(versionA), hostWithDeletedInAAndB: nil,
		}, nil
	}
	ds.GetIncludedHostIDMapForVPPAppFunc = func(ctx context.Context, vppAppTeamID uint) (map[uint]struct{}, error) {
		if vppAppTeamID == versionA {
			return map[uint]struct{}{hostInAAndB: {}, hostWithDeletedInAAndB: {}}, nil
		}
		return map[uint]struct{}{hostInAAndB: {}, hostInB: {}, hostWithDeletedInB: {}, hostMovedToB: {}, hostWithDeletedInAAndB: {}}, nil
	}
	ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
		jobs = append(jobs, job)
		return job, nil
	}
	installer := &resendVPPInstaller{}
	w := &SoftwareWorker{Datastore: ds, VPPInstaller: installer, Log: slog.New(slog.DiscardHandler)}

	runResend := func(configChangedAppTeamIDs []uint) error {
		jobs = nil
		argsJSON, err := json.Marshal(softwareWorkerArgs{
			Task:                    resendVPPAppConfigurationTask,
			ApplicationID:           appID.AdamID,
			Platform:                appID.Platform,
			ConfigChangedAppTeamIDs: configChangedAppTeamIDs,
		})
		require.NoError(t, err)
		return w.Run(t.Context(), argsJSON)
	}
	queuedHostIDsByVersionID := func() map[uint][]uint {
		hostIDsByVersionID := make(map[uint][]uint)
		for _, job := range jobs {
			var args softwareWorkerArgs
			require.NoError(t, json.Unmarshal(*job.Args, &args))
			require.Equal(t, resendVPPAppConfigurationBatchTask, args.Task)
			hostIDsByVersionID[args.AppTeamID] = args.HostIDs
		}
		return hostIDsByVersionID
	}

	// change version B's configuration, the hosts whose version is B should be queued, including the host labels moved to B
	err := runResend([]uint{versionB})
	require.NoError(t, err)
	require.Equal(t, map[uint][]uint{versionB: {hostInB, hostWithDeletedInB, hostMovedToB}}, queuedHostIDsByVersionID())

	// change version A's configuration, the hosts in A and B should be queued for A since A is the first-added version
	err = runResend([]uint{versionA})
	require.NoError(t, err)
	require.Equal(t, map[uint][]uint{versionA: {hostInAAndB, hostWithDeletedInAAndB}}, queuedHostIDsByVersionID())

	// delete a version, only hosts whose install's version was deleted and that a remaining version includes should be queued,
	// each for its first-added version, and the host labels moved to B should not be queued since its install's version exists
	err = runResend(nil)
	require.NoError(t, err)
	require.Equal(t, map[uint][]uint{versionA: {hostWithDeletedInAAndB}, versionB: {hostWithDeletedInB}}, queuedHostIDsByVersionID())

	// run the version B batch with an install of the app waiting on one host, the other host should get version B
	batchArgsJSON, err := json.Marshal(softwareWorkerArgs{
		Task:          resendVPPAppConfigurationBatchTask,
		ApplicationID: appID.AdamID,
		Platform:      appID.Platform,
		AppTeamID:     versionB,
		HostIDs:       []uint{hostInB, hostWithDeletedInB},
	})
	require.NoError(t, err)
	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id == hostDeleted {
			return nil, platform_mysql.NotFound("Host")
		}
		if id == hostOnOtherFleet {
			return &fleet.Host{ID: id, Platform: "ios", TeamID: new(uint(99))}, nil
		}
		return &fleet.Host{ID: id, Platform: "ios"}, nil
	}
	ds.GetVPPAppByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, vppAppTeamID uint) (*fleet.VPPApp, error) {
		return &fleet.VPPApp{VPPAppTeam: fleet.VPPAppTeam{VPPAppID: appID, AppTeamID: vppAppTeamID}, TitleID: titleID}, nil
	}
	ds.GetHostIDsWithUnactivatedVPPAppInstallFunc = func(ctx context.Context, adamID string, hostIDs []uint) (map[uint]struct{}, error) {
		return map[uint]struct{}{hostInB: {}}, nil
	}
	installer.installedAppTeamIDByHostID = make(map[uint]uint)
	err = w.Run(t.Context(), batchArgsJSON)
	require.NoError(t, err)
	require.Equal(t, map[uint]uint{hostWithDeletedInB: versionB}, installer.installedAppTeamIDByHostID)

	// run the batch with a host that can't take the install and a host whose install fails at Apple, the batch should fail so it is retried
	ds.GetHostIDsWithUnactivatedVPPAppInstallFunc = func(ctx context.Context, adamID string, hostIDs []uint) (map[uint]struct{}, error) {
		return map[uint]struct{}{}, nil
	}
	installer.installedAppTeamIDByHostID = make(map[uint]uint)
	installer.installErrByHostID = map[uint]error{
		hostInB:            &fleet.BadRequestError{Message: "no available licenses"},
		hostWithDeletedInB: errors.New("apple vpp api unavailable"),
	}
	err = w.Run(t.Context(), batchArgsJSON)
	require.ErrorContains(t, err, fmt.Sprintf("failed for hosts [%d]", hostWithDeletedInB))
	require.Empty(t, installer.installedAppTeamIDByHostID)

	// run the batch with only the host that can't take the install failing, the batch should succeed without retrying it
	installer.installErrByHostID = map[uint]error{hostInB: &fleet.BadRequestError{Message: "no available licenses"}}
	err = w.Run(t.Context(), batchArgsJSON)
	require.NoError(t, err)
	require.Equal(t, map[uint]uint{hostWithDeletedInB: versionB}, installer.installedAppTeamIDByHostID)

	// run a batch for a host deleted and a host moved to another fleet after the batch was queued, nothing should be installed
	movedOrDeletedArgsJSON, err := json.Marshal(softwareWorkerArgs{
		Task:          resendVPPAppConfigurationBatchTask,
		ApplicationID: appID.AdamID,
		Platform:      appID.Platform,
		AppTeamID:     versionB,
		HostIDs:       []uint{hostDeleted, hostOnOtherFleet},
	})
	require.NoError(t, err)
	installer.installedAppTeamIDByHostID = make(map[uint]uint)
	installer.installErrByHostID = nil
	err = w.Run(t.Context(), movedOrDeletedArgsJSON)
	require.NoError(t, err)
	require.Empty(t, installer.installedAppTeamIDByHostID)

	// run the batch after its version was deleted, nothing should be installed
	ds.GetVPPAppByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, vppAppTeamID uint) (*fleet.VPPApp, error) {
		return nil, platform_mysql.NotFound("VPPApp")
	}
	err = w.Run(t.Context(), batchArgsJSON)
	require.NoError(t, err)
	require.Empty(t, installer.installedAppTeamIDByHostID)

	// delete every version of the app, nothing should be queued
	ds.GetVPPAppMetadataByAdamIDPlatformTeamIDFunc = func(ctx context.Context, adamID string, platform fleet.InstallableDevicePlatform, teamID *uint) (*fleet.VPPApp, error) {
		return nil, platform_mysql.NotFound("VPPApp")
	}
	err = runResend(nil)
	require.NoError(t, err)
	require.Empty(t, jobs)

	// run a batch queued before every version was deleted, nothing should be installed
	installer.installedAppTeamIDByHostID = make(map[uint]uint)
	installer.installErrByHostID = nil
	err = w.Run(t.Context(), batchArgsJSON)
	require.NoError(t, err)
	require.Empty(t, installer.installedAppTeamIDByHostID)
}
