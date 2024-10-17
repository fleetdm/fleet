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
	resetIndicators := func() {
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		ds.InsertHostVPPSoftwareInstallFuncInvoked = false
		ds.NewHostScriptExecutionRequestFuncInvoked = false
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false
		clear(requestedInstalls)
	}

	host1UUID := "123"
	host1ID := uint(1)
	installerID1 := uint(1)

	var mockListSetupExperience []*fleet.SetupExperienceStatusResult
	ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hostUUID string) ([]*fleet.SetupExperienceStatusResult, error) {
		return mockListSetupExperience, nil
	}

	var mockListHostsLite []*fleet.Host
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		return mockListHostsLite, nil
	}

	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID, softwareInstallerID uint, selfService bool, policyID *uint) (string, error) {
		requestedInstalls[hostID] = append(requestedInstalls[hostID], softwareInstallerID)
		return "install-uuid", nil
	}

	ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
		return nil
	}

	// No host exists
	_, err := svc.SetupExperienceNextStep(ctx, host1UUID)
	require.Error(t, err)

	mockListHostsLite = append(mockListHostsLite, &fleet.Host{UUID: host1UUID, ID: host1ID})

	// Host exists, nothing to do
	finished, err := svc.SetupExperienceNextStep(ctx, host1UUID)
	require.NoError(t, err)
	assert.True(t, finished)
	assert.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.False(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.False(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
	resetIndicators()

	// Installer queued
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
	resetIndicators()
}
