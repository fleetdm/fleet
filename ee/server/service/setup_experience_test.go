package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
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

	// No host exists
	_, err := svc.SetupExperienceNextStep(ctx, host1UUID)
	require.Error(t, err)

	// Host exists, nothing to do
	mockListHostsLite = append(mockListHostsLite, &fleet.Host{UUID: host1UUID, ID: host1ID})

	finished, err := svc.SetupExperienceNextStep(ctx, host1UUID)
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

	finished, err = svc.SetupExperienceNextStep(ctx, host1UUID)
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
	finished, err = svc.SetupExperienceNextStep(ctx, host1UUID)
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

	finished, err = svc.SetupExperienceNextStep(ctx, host1UUID)
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
	finished, err = svc.SetupExperienceNextStep(ctx, host1UUID)
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
	finished, err = svc.SetupExperienceNextStep(ctx, host1UUID)
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

	finished, err = svc.SetupExperienceNextStep(ctx, host1UUID)
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

	finished, err = svc.SetupExperienceNextStep(ctx, host1UUID)
	require.NoError(t, err)
	assert.True(t, finished)
}
