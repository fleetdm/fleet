package service

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupExperienceNextStep(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	requestedInstalls := make(map[uint][]uint)
	requestedUpdateSetupExperience := []*fleet.SetupExperienceStatusResult{}
	requestedScriptExecution := []*fleet.HostScriptRequestPayload{}
	resetIndicators := func() {
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		ds.InsertHostVPPSoftwareInstallFuncInvoked = false
		ds.NewHostScriptExecutionRequestFuncInvoked = false
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false
		clear(requestedInstalls)
		requestedUpdateSetupExperience = []*fleet.SetupExperienceStatusResult{}
		requestedScriptExecution = []*fleet.HostScriptRequestPayload{}
	}

	host1UUID := "123"
	host1ID := uint(1)
	installerID1 := uint(2)
	scriptID1 := uint(3)
	scriptContentID1 := uint(4)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			MDM: fleet.MDM{
				EnabledAndConfigured: true,
			},
		}, nil
	}

	ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
		return true, nil
	}

	var mockListSetupExperience []*fleet.SetupExperienceStatusResult
	ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hostUUID string) ([]*fleet.SetupExperienceStatusResult, error) {
		return mockListSetupExperience, nil
	}

	var mockListHostsLite []*fleet.Host
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		return mockListHostsLite, nil
	}

	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		requestedInstalls[hostID] = append(requestedInstalls[hostID], softwareInstallerID)
		return "install-uuid", nil
	}

	ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
		requestedUpdateSetupExperience = append(requestedUpdateSetupExperience, status)
		return nil
	}

	ds.NewHostScriptExecutionRequestFunc = func(ctx context.Context, request *fleet.HostScriptRequestPayload) (*fleet.HostScriptResult, error) {
		requestedScriptExecution = append(requestedScriptExecution, request)
		return &fleet.HostScriptResult{
			ExecutionID: "script-uuid",
		}, nil
	}

	mockListHostsLite = append(mockListHostsLite, &fleet.Host{UUID: host1UUID, ID: host1ID})

	finished, err := svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.True(t, finished)
	assert.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.False(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.False(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	resetIndicators()

	// Only installer queued
	mockListSetupExperience = []*fleet.SetupExperienceStatusResult{
		{
			HostUUID:            host1UUID,
			SoftwareInstallerID: &installerID1,
			Status:              fleet.SetupExperienceStatusPending,
		},
	}

	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.False(t, finished)
	assert.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.False(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	assert.Len(t, requestedInstalls, 1)
	assert.Len(t, requestedUpdateSetupExperience, 1)
	assert.Equal(t, "install-uuid", *requestedUpdateSetupExperience[0].HostSoftwareInstallsExecutionID)

	mockListSetupExperience[0].Status = fleet.SetupExperienceStatusSuccess
	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.True(t, finished)

	resetIndicators()

	// TODO VPP app queueing is better done in an integration
	// test, the setup required would be too much

	// Only script queued
	mockListSetupExperience = []*fleet.SetupExperienceStatusResult{
		{
			HostUUID:                host1UUID,
			SetupExperienceScriptID: &scriptID1,
			ScriptContentID:         &scriptContentID1,
			Status:                  fleet.SetupExperienceStatusPending,
		},
	}

	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.False(t, finished)
	assert.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.True(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	assert.Len(t, requestedScriptExecution, 1)
	assert.Len(t, requestedUpdateSetupExperience, 1)
	assert.Equal(t, "script-uuid", *requestedUpdateSetupExperience[0].ScriptExecutionID)

	mockListSetupExperience[0].Status = fleet.SetupExperienceStatusSuccess
	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.True(t, finished)

	resetIndicators()

	// Both installer and script
	mockListSetupExperience = []*fleet.SetupExperienceStatusResult{
		{
			HostUUID:            host1UUID,
			SoftwareInstallerID: &installerID1,
			Status:              fleet.SetupExperienceStatusPending,
		},
		{
			HostUUID:                host1UUID,
			SetupExperienceScriptID: &scriptID1,
			ScriptContentID:         &scriptContentID1,
			Status:                  fleet.SetupExperienceStatusPending,
		},
	}

	// Only installer is queued
	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.False(t, finished)
	assert.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.False(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	assert.Len(t, requestedInstalls, 1)
	assert.Len(t, requestedScriptExecution, 0)
	assert.Len(t, requestedUpdateSetupExperience, 1)

	// install finished, call it again. This time script is queued
	mockListSetupExperience[0].Status = fleet.SetupExperienceStatusSuccess

	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.False(t, finished)
	assert.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.True(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	assert.Len(t, requestedInstalls, 1)
	assert.Len(t, requestedScriptExecution, 1)
	assert.Len(t, requestedUpdateSetupExperience, 2)

	// both finished, now we're done
	mockListSetupExperience[1].Status = fleet.SetupExperienceStatusFailure

	finished, err = svc.SetupExperienceNextStep(ctx, &fleet.Host{
		UUID:     host1UUID,
		Platform: "darwin",
	})
	require.NoError(t, err)
	assert.True(t, finished)
}

func TestSetupExperienceSetWithManualAgentInstall(t *testing.T) {
	ctx := test.UserContext(context.Background(), test.UserAdmin)
	ds := new(mock.Store)
	svc, baseSvc := newTestServiceWithMock(t, ds)

	appConfig := fleet.AppConfig{}
	team := fleet.TeamLite{}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &appConfig, nil
	}

	ds.TeamLiteFunc = func(ctx context.Context, tid uint) (*fleet.TeamLite, error) {
		return &team, nil
	}

	ds.SetSetupExperienceSoftwareTitlesFunc = func(ctx context.Context, platform string, teamID uint, titleIDs []uint) error {
		return nil
	}

	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}

	ds.SetSetupExperienceScriptFunc = func(ctx context.Context, script *fleet.Script) error {
		return nil
	}

	baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	// No manual agent install, we good
	// No team
	err := svc.SetSetupExperienceSoftware(ctx, "darwin", 0, []uint{1, 2})
	require.NoError(t, err)

	scriptReader := bytes.NewReader([]byte("hello"))
	err = svc.SetSetupExperienceScript(ctx, nil, "potato.sh", scriptReader)
	require.NoError(t, err)
	_, _ = scriptReader.Seek(0, io.SeekStart)

	// Team
	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 1, []uint{1, 2})
	require.NoError(t, err)

	err = svc.SetSetupExperienceScript(ctx, ptr.Uint(1), "potato.sh", scriptReader)
	require.NoError(t, err)
	_, _ = scriptReader.Seek(0, io.SeekStart)

	// Manual agent install
	appConfig.MDM.MacOSSetup.ManualAgentInstall = optjson.SetBool(true)
	team.Config.MDM.MacOSSetup.ManualAgentInstall = optjson.SetBool(true)

	// No team
	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 0, []uint{1, 2})
	require.ErrorContains(t, err, "Couldn’t add setup experience software. To add software, first disable macos_manual_agent_install.")

	err = svc.SetSetupExperienceScript(ctx, nil, "potato.sh", scriptReader)
	require.ErrorContains(t, err, "Couldn’t add setup experience script. To add script, first disable macos_manual_agent_install.")
	_, _ = scriptReader.Seek(0, io.SeekStart)

	// Team
	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 1, []uint{1, 2})
	require.ErrorContains(t, err, "Couldn’t add setup experience software. To add software, first disable macos_manual_agent_install.")

	err = svc.SetSetupExperienceScript(ctx, ptr.Uint(1), "potato.sh", scriptReader)
	require.ErrorContains(t, err, "Couldn’t add setup experience script. To add script, first disable macos_manual_agent_install.")
	_, _ = scriptReader.Seek(0, io.SeekStart)

	// We can still set software to none though
	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 0, []uint{})
	require.NoError(t, err)

	err = svc.SetSetupExperienceSoftware(ctx, "darwin", 1, []uint{})
	require.NoError(t, err)

	t.Run("should not block for non darwin hosts", func(t *testing.T) {
		for _, platform := range fleet.SetupExperienceSupportedPlatforms {
			if platform == "darwin" {
				continue
			}

			err := svc.SetSetupExperienceSoftware(ctx, platform, 0, []uint{1, 2})
			require.NoError(t, err)
		}
	})
}

func TestFailCancelledSetupExperienceInstalls(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	svc, baseSvc := newTestServiceWithMock(t, ds)
	svc.logger = slog.Default()

	ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
		return nil
	}

	t.Run("cancelled VPP app creates failed activity", func(t *testing.T) {
		var activities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activities = append(activities, activity)
			return nil
		}

		vppTeamID := uint(1)
		adamID := "12345"
		cmdUUID := "cmd-uuid-1"
		titleID := uint(10)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:        "host-uuid",
				Name:            "VPP App",
				Status:          fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				NanoCommandUUID: &cmdUUID,
				SoftwareTitleID: &titleID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, 1, "host-uuid", "Test Host", "darwin", results)
		require.NoError(t, err)
		require.Len(t, activities, 1)

		act, ok := activities[0].(*fleet.ActivityInstalledAppStoreApp)
		require.True(t, ok)
		assert.Equal(t, uint(1), act.HostID)
		assert.Equal(t, "Test Host", act.HostDisplayName)
		assert.Equal(t, "VPP App", act.SoftwareTitle)
		assert.Equal(t, "12345", act.AppStoreID)
		assert.Equal(t, "cmd-uuid-1", act.CommandUUID)
		assert.Equal(t, "failed", act.Status)
		assert.Equal(t, "darwin", act.HostPlatform)

		// Verify the status was changed to failure
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[0].Status)
	})

	t.Run("non-cancelled VPP app is skipped", func(t *testing.T) {
		var activities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activities = append(activities, activity)
			return nil
		}

		vppTeamID := uint(1)
		adamID := "12345"
		cmdUUID := "cmd-uuid-1"
		titleID := uint(10)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:        "host-uuid",
				Name:            "VPP App Pending",
				Status:          fleet.SetupExperienceStatusPending,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				NanoCommandUUID: &cmdUUID,
				SoftwareTitleID: &titleID,
			},
			{
				HostUUID:        "host-uuid",
				Name:            "VPP App Success",
				Status:          fleet.SetupExperienceStatusSuccess,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				NanoCommandUUID: &cmdUUID,
				SoftwareTitleID: &titleID,
			},
			{
				HostUUID:        "host-uuid",
				Name:            "VPP App Running",
				Status:          fleet.SetupExperienceStatusRunning,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				NanoCommandUUID: &cmdUUID,
				SoftwareTitleID: &titleID,
			},
			{
				HostUUID:        "host-uuid",
				Name:            "VPP App Failed",
				Status:          fleet.SetupExperienceStatusFailure,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				NanoCommandUUID: &cmdUUID,
				SoftwareTitleID: &titleID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, 1, "host-uuid", "Test Host", "darwin", results)
		require.NoError(t, err)
		assert.Empty(t, activities)

		// Statuses should be unchanged
		assert.Equal(t, fleet.SetupExperienceStatusPending, results[0].Status)
		assert.Equal(t, fleet.SetupExperienceStatusSuccess, results[1].Status)
		assert.Equal(t, fleet.SetupExperienceStatusRunning, results[2].Status)
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[3].Status)
	})

	t.Run("cancelled software package and VPP app both create activities", func(t *testing.T) {
		var activities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activities = append(activities, activity)
			return nil
		}

		installerID := uint(5)
		executionID := "exec-uuid-1"
		vppTeamID := uint(1)
		adamID := "67890"
		cmdUUID := "cmd-uuid-2"
		titleID := uint(20)

		ds.GetSoftwareInstallerMetadataByIDFunc = func(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
			return &fleet.SoftwareInstaller{
				Name:    "installer.pkg",
				TitleID: ptr.Uint(30),
			}, nil
		}
		ds.SoftwareTitleByIDFunc = func(ctx context.Context, id uint, teamID *uint, tmFilter fleet.TeamFilter) (*fleet.SoftwareTitle, error) {
			source := "apps"
			return &fleet.SoftwareTitle{Source: source}, nil
		}

		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:                        "host-uuid",
				Name:                            "Software Package",
				Status:                          fleet.SetupExperienceStatusCancelled,
				SoftwareInstallerID:             &installerID,
				HostSoftwareInstallsExecutionID: &executionID,
			},
			{
				HostUUID:        "host-uuid",
				Name:            "VPP App",
				Status:          fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				NanoCommandUUID: &cmdUUID,
				SoftwareTitleID: &titleID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, 1, "host-uuid", "Test Host", "darwin", results)
		require.NoError(t, err)
		require.Len(t, activities, 2)

		// First activity should be the software package install
		swAct, ok := activities[0].(fleet.ActivityTypeInstalledSoftware)
		require.True(t, ok)
		assert.Equal(t, "failed", swAct.Status)
		assert.Equal(t, "Software Package", swAct.SoftwareTitle)
		assert.Equal(t, "installer.pkg", swAct.SoftwarePackage)
		assert.True(t, swAct.FromSetupExperience)
		assert.False(t, swAct.SelfService)

		// Second activity should be the VPP app install
		vppAct, ok := activities[1].(*fleet.ActivityInstalledAppStoreApp)
		require.True(t, ok)
		assert.Equal(t, "failed", vppAct.Status)
		assert.Equal(t, "VPP App", vppAct.SoftwareTitle)
		assert.Equal(t, "67890", vppAct.AppStoreID)
		assert.Equal(t, "cmd-uuid-2", vppAct.CommandUUID)
		assert.Equal(t, "darwin", vppAct.HostPlatform)
		assert.Equal(t, uint(1), vppAct.HostID)
		assert.Equal(t, "Test Host", vppAct.HostDisplayName)
	})

	t.Run("hostPlatform is propagated to VPP activity", func(t *testing.T) {
		var activities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activities = append(activities, activity)
			return nil
		}

		vppTeamID := uint(1)
		adamID := "99999"
		cmdUUID := "cmd-uuid-3"
		titleID := uint(30)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:        "host-uuid-2",
				Name:            "iOS VPP App",
				Status:          fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				NanoCommandUUID: &cmdUUID,
				SoftwareTitleID: &titleID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, 2, "host-uuid-2", "iOS Host", "ios", results)
		require.NoError(t, err)
		require.Len(t, activities, 1)

		act, ok := activities[0].(*fleet.ActivityInstalledAppStoreApp)
		require.True(t, ok)
		assert.Equal(t, "ios", act.HostPlatform)
		assert.Equal(t, uint(2), act.HostID)
		assert.Equal(t, "iOS Host", act.HostDisplayName)
		assert.Equal(t, "iOS VPP App", act.SoftwareTitle)
	})

	t.Run("empty results is a no-op", func(t *testing.T) {
		var activities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activities = append(activities, activity)
			return nil
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, 1, "host-uuid", "Test Host", "darwin", []*fleet.SetupExperienceStatusResult{})
		require.NoError(t, err)
		assert.Empty(t, activities)
	})

	t.Run("cancelled VPP app with nil NanoCommandUUID and VPPAppAdamID does not panic", func(t *testing.T) {
		var activities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activities = append(activities, activity)
			return nil
		}

		vppTeamID := uint(1)
		titleID := uint(10)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:        "host-uuid",
				Name:            "VPP App No Command",
				Status:          fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    nil,
				NanoCommandUUID: nil,
				SoftwareTitleID: &titleID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, 1, "host-uuid", "Test Host", "darwin", results)
		require.NoError(t, err)
		require.Len(t, activities, 1)

		act, ok := activities[0].(*fleet.ActivityInstalledAppStoreApp)
		require.True(t, ok)
		assert.Equal(t, "", act.AppStoreID)
		assert.Equal(t, "", act.CommandUUID)
		assert.Equal(t, "failed", act.Status)
		assert.Equal(t, "VPP App No Command", act.SoftwareTitle)
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[0].Status)
	})

	t.Run("user is nil for VPP activity", func(t *testing.T) {
		var capturedUser *fleet.User
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			capturedUser = user
			return nil
		}

		vppTeamID := uint(1)
		adamID := "12345"
		cmdUUID := "cmd-uuid-1"
		titleID := uint(10)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:        "host-uuid",
				Name:            "VPP App",
				Status:          fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				NanoCommandUUID: &cmdUUID,
				SoftwareTitleID: &titleID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, 1, "host-uuid", "Test Host", "darwin", results)
		require.NoError(t, err)
		assert.Nil(t, capturedUser)
	})
}
