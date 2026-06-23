package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
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
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]string, error) {
		return []string{"com.example.teamapp"}, nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, appIDs []string, globalOrTeamID uint) (map[string][]byte, error) {
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
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, appIDs []string, globalOrTeamID uint) (map[string][]byte, error) {
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
	synctest.Test(t, func(t *testing.T) {
		var callCount atomic.Int32
		var totalHosts atomic.Int32

		androidModule := &mockAndroidModule{
			addAppsToAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
				callCount.Add(1)
				totalHosts.Add(int32(len(hostUUIDs))) //nolint:gosec // test with small host counts
				return make(map[string]*android.MDMAndroidPolicyRequest), nil
			},
		}

		ds := new(mock.Store)
		// 5 hosts in scope
		ds.GetIncludedHostUUIDMapForAppStoreAppFunc = func(ctx context.Context, appTeamID uint) (map[string]string, error) {
			hosts := make(map[string]string, 5)
			for i := range 5 {
				hosts[fmt.Sprintf("host-%d", i)] = fmt.Sprintf("host-%d", i)
			}
			return hosts, nil
		}
		ds.GetAndroidAppConfigurationByAppTeamIDFunc = func(ctx context.Context, appTeamID uint) ([]byte, error) {
			return nil, nil
		}

		w := &SoftwareWorker{
			Datastore:        ds,
			AndroidModule:    androidModule,
			Log:              slog.New(slog.DiscardHandler),
			AndroidBatchSize: 2, // batch size of 2 → 3 batches (2+2+1)
		}

		ctx := t.Context()
		errCh := make(chan error, 1)
		go func() {
			errCh <- w.makeAndroidAppAvailable(ctx, "com.example.app", 1, "enterprises/test", false)
		}()

		// First batch runs immediately.
		synctest.Wait()
		require.Equal(t, int32(1), callCount.Load(), "first batch should run immediately")

		// Advance past first stagger interval → second batch.
		time.Sleep(androidSoftwareInstallStaggerInterval)
		synctest.Wait()
		require.Equal(t, int32(2), callCount.Load(), "second batch after first sleep")

		// Advance past second stagger interval → third batch.
		time.Sleep(androidSoftwareInstallStaggerInterval)
		synctest.Wait()
		require.Equal(t, int32(3), callCount.Load(), "third batch after second sleep")

		require.NoError(t, <-errCh)
		require.Equal(t, int32(5), totalHosts.Load(), "all 5 hosts processed")
	})
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
