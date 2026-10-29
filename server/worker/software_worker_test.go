package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
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

func (m *mockAndroidModule) SetAppsForAndroidPolicy(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
	if m.setAppsForAndroidPolicyFunc != nil {
		return nil, m.setAppsForAndroidPolicyFunc(ctx, enterpriseName, appPolicies, hostUUIDs)
	}
	return nil, nil
}

func TestUpdateAndroidHostsAfterFleetTransferFlagsAppsForResend(t *testing.T) {
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
	var newTeamID uint
	ds.CreatePendingCertificateTemplatesForNewHostFunc = func(ctx context.Context, hostUUID string, teamID uint) (int64, error) {
		newTeamID = teamID
		return 0, nil
	}
	var flaggedHostIDs []uint
	ds.SetHostAndroidSoftwareInstallsNeedsResendFunc = func(ctx context.Context, hostIDs []uint, builtAfter *time.Time) error {
		require.Nil(t, builtAfter)
		flaggedHostIDs = hostIDs
		return nil
	}
	androidModule := &mockAndroidModule{
		setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
			t.Fatal("the Android profile reconciler sends the app list after a fleet transfer, not the worker")
			return nil
		},
	}
	worker := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	// run the fleet transfer task, the host's certificate templates should be moved to its new fleet and its apps
	// flagged for resend, with no app list sent
	err := worker.updateAndroidHostsAfterFleetTransfer(ctx, map[string]uint{hostUUID: hostID})
	require.NoError(t, err)
	require.True(t, ds.SetHostCertificateTemplatesToPendingRemoveForHostFuncInvoked)
	require.Equal(t, teamID, newTeamID)
	require.Equal(t, []uint{hostID}, flaggedHostIDs)
}

func TestSoftwareWorkerRunSkipsDeprecatedAndroidAppTasks(t *testing.T) {
	w := &SoftwareWorker{Datastore: new(mock.Store), AndroidModule: &mockAndroidModule{}, Log: slog.New(slog.DiscardHandler)}

	// run each deprecated task with the arguments it was queued with, it should succeed without reading the
	// datastore or calling the Android Management API
	for _, task := range []SoftwareWorkerTask{
		makeAndroidAppAvailableTask,
		makeAndroidAppAvailableBatchTask,
		makeAndroidAppUnavailableTask,
		bulkSetAndroidAppsAvailableForHostTask,
	} {
		args, err := json.Marshal(softwareWorkerArgs{
			Task:               task,
			ApplicationID:      "com.example.app",
			AppTeamID:          1,
			EnterpriseName:     "enterprises/test",
			HostUUIDToPolicyID: map[string]string{"host-1": "host-1"},
			HostUUID:           "host-1",
			PolicyID:           "host-1",
			ApplicationIDs:     []string{"com.example.app"},
		})
		require.NoError(t, err)
		err = w.Run(t.Context(), args)
		require.NoError(t, err, task)
	}
}

func TestSoftwareWorkerRunDisablesAMAPIRetry(t *testing.T) {
	var called bool
	androidModule := &mockAndroidModule{
		setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
			called = true
			assert.True(t, androidmgmt.RetryDisabled(ctx), "worker jobs must not wait out AMAPI quota errors")
			return nil
		},
	}
	ds := new(mock.Store)
	ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{Host: &fleet.Host{ID: 1, UUID: hostUUID}, Device: &android.Device{}}, nil
	}
	ds.SetHostCertificateTemplatesToPendingRemoveForHostFunc = func(ctx context.Context, hostUUID string) error {
		return nil
	}
	ds.CreatePendingCertificateTemplatesForNewHostFunc = func(ctx context.Context, hostUUID string, teamID uint) (int64, error) {
		return 0, nil
	}
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
		return nil, nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
		return map[string][]byte{}, nil
	}
	ds.BulkUpsertHostAndroidSoftwareInstallsFunc = func(ctx context.Context, installs []fleet.HostAndroidSoftwareInstall, removedInstalls []fleet.HostAndroidSoftwareInstall) error {
		return nil
	}
	ds.ReplaceHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostID uint, installs []fleet.HostAndroidSoftwareInstall) error {
		return nil
	}
	w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	args, err := json.Marshal(softwareWorkerArgs{
		Task:           makeAndroidAppsAvailableForHostTask,
		EnterpriseName: "enterprises/test",
		HostUUID:       "host-1",
		HostID:         1,
		PolicyID:       "host-1",
	})
	require.NoError(t, err)
	require.NoError(t, w.Run(t.Context(), args))
	require.True(t, called)
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
	var upsertedInstalls []fleet.HostAndroidSoftwareInstall
	ds.BulkUpsertHostAndroidSoftwareInstallsFunc = func(ctx context.Context, installs []fleet.HostAndroidSoftwareInstall, removedInstalls []fleet.HostAndroidSoftwareInstall) error {
		upsertedInstalls = append(upsertedInstalls, installs...)
		return nil
	}
	ds.ReplaceHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostID uint, installs []fleet.HostAndroidSoftwareInstall) error {
		return nil
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

	// read the upserted installs, the first-added version should be pending as PREINSTALLED in policy version 1
	require.Len(t, upsertedInstalls, 1)
	require.Equal(t, "com.example.app", upsertedInstalls[0].AdamID)
	require.Equal(t, new(uint(1)), upsertedInstalls[0].VPPAppTeamID)
	require.Equal(t, android.AppStatusPreinstalled, upsertedInstalls[0].InstallType)
	require.Equal(t, fleet.AndroidSoftwareInstallStatusPending, upsertedInstalls[0].Status)
	require.Equal(t, new(int64(1)), upsertedInstalls[0].InPolicyVersion)
}

func TestRunAndroidSetupExperienceWithInstallConstraint(t *testing.T) {
	const hostUUID = "setup-host-uuid"
	const constrainedApp = "com.example.constrained"
	const otherApp = "com.example.other"

	ds := new(mock.Store)
	ds.AndroidHostLiteByHostUUIDFunc = func(ctx context.Context, uuid string) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{Host: &fleet.Host{ID: 1, UUID: hostUUID}, Device: &android.Device{AppliedPolicyID: new(hostUUID)}}, nil
	}
	apps := []fleet.VPPAppTeam{
		{AdamID: constrainedApp, Platform: fleet.AndroidPlatform, AppTeamID: 1},
		{AdamID: otherApp, Platform: fleet.AndroidPlatform, AppTeamID: 2},
	}
	ds.GetAndroidAppsInScopeForHostFunc = func(ctx context.Context, hostID uint) ([]fleet.VPPAppTeam, error) {
		return apps, nil
	}
	ds.GetVPPAppsToInstallDuringSetupExperienceFunc = func(ctx context.Context, teamID *uint, platform string) ([]fleet.VPPAppTeam, error) {
		return apps, nil
	}
	ds.BulkGetAndroidAppConfigurationsFunc = func(ctx context.Context, vppAppTeamIDs []uint) (map[string][]byte, error) {
		return map[string][]byte{
			constrainedApp: []byte(`{"installConstraint": [{"chargingConstraint": "INSTALL_ONLY_WHEN_CHARGING"}]}`),
		}, nil
	}
	ds.BulkUpsertHostAndroidSoftwareInstallsFunc = func(ctx context.Context, installs []fleet.HostAndroidSoftwareInstall, removedInstalls []fleet.HostAndroidSoftwareInstall) error {
		return nil
	}
	ds.ReplaceHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostID uint, installs []fleet.HostAndroidSoftwareInstall) error {
		return nil
	}
	var installs []*fleet.HostAndroidVPPSoftwareInstall
	ds.InsertAndroidSetupExperienceSoftwareInstallFunc = func(ctx context.Context, payload *fleet.HostAndroidVPPSoftwareInstall) error {
		installs = append(installs, payload)
		return nil
	}

	// Reject what Google rejects: an installConstraint on an AVAILABLE app.
	rejectConstrainedAvailable := func(policies []*androidmanagement.ApplicationPolicy) error {
		for _, p := range policies {
			if p.InstallType == "AVAILABLE" && len(p.InstallConstraint) > 0 {
				return errors.New("googleapi: Error 400: Invalid installConstraint, not supported with install type AVAILABLE")
			}
		}
		return nil
	}
	var availablePolicies, installPolicies []*androidmanagement.ApplicationPolicy
	androidModule := &mockAndroidModule{
		setAppsForAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, policies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
			availablePolicies = policies
			return rejectConstrainedAvailable(policies)
		},
		addAppsToAndroidPolicyFunc: func(ctx context.Context, enterpriseName string, policies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
			installPolicies = policies
			if err := rejectConstrainedAvailable(policies); err != nil {
				return nil, err
			}
			return map[string]*android.MDMAndroidPolicyRequest{hostUUID: {PolicyVersion: sql.Null[int64]{V: 1, Valid: true}}}, nil
		},
	}
	w := &SoftwareWorker{Datastore: ds, AndroidModule: androidModule, Log: slog.New(slog.DiscardHandler)}

	err := w.runAndroidSetupExperience(t.Context(), hostUUID, 0, "enterprises/test")
	require.NoError(t, err)

	require.Len(t, availablePolicies, 2)
	for _, p := range availablePolicies {
		assert.Equal(t, "AVAILABLE", p.InstallType)
		assert.Empty(t, p.InstallConstraint, p.PackageName)
	}

	require.Len(t, installPolicies, 2)
	installByApp := make(map[string]*androidmanagement.ApplicationPolicy, len(installPolicies))
	for _, p := range installPolicies {
		installByApp[p.PackageName] = p
	}
	constrainedInstall, ok := installByApp[constrainedApp]
	require.True(t, ok)
	assert.Equal(t, "PREINSTALLED", constrainedInstall.InstallType)
	require.Len(t, constrainedInstall.InstallConstraint, 1)
	assert.Equal(t, "INSTALL_ONLY_WHEN_CHARGING", constrainedInstall.InstallConstraint[0].ChargingConstraint)
	otherInstall, ok := installByApp[otherApp]
	require.True(t, ok)
	assert.Empty(t, otherInstall.InstallConstraint)

	require.Len(t, installs, 2)
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
			var upsertedAdamIDs []string
			ds.ReplaceHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostID uint, installs []fleet.HostAndroidSoftwareInstall) error {
				for _, install := range installs {
					upsertedAdamIDs = append(upsertedAdamIDs, install.AdamID)
				}
				return nil
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

			// read the upserted installs, they should match the sent list, the fleet agent included
			require.Equal(t, c.wantPackages, upsertedAdamIDs)
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
			ds.BulkUpsertHostAndroidSoftwareInstallsFunc = func(ctx context.Context, installs []fleet.HostAndroidSoftwareInstall, removedInstalls []fleet.HostAndroidSoftwareInstall) error {
				return nil
			}
			ds.ReplaceHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostID uint, installs []fleet.HostAndroidSoftwareInstall) error {
				return nil
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
	ds.BulkUpsertHostAndroidSoftwareInstallsFunc = func(ctx context.Context, installs []fleet.HostAndroidSoftwareInstall, removedInstalls []fleet.HostAndroidSoftwareInstall) error {
		return nil
	}
	ds.ReplaceHostAndroidSoftwareInstallsFunc = func(ctx context.Context, hostID uint, installs []fleet.HostAndroidSoftwareInstall) error {
		return nil
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
