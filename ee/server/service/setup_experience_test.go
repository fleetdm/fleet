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

	var mockListSetupExperience []*fleet.SetupExperienceStatusResult
	ds.ListSetupExperienceResultsByHostUUIDFunc = func(ctx context.Context, hostUUID string) ([]*fleet.SetupExperienceStatusResult, error) {
		return mockListSetupExperience, nil
	}

	var mockListHostsLite []*fleet.Host
	ds.ListHostsLiteByUUIDsFunc = func(ctx context.Context, filter fleet.TeamFilter, uuids []string) ([]*fleet.Host, error) {
		return mockListHostsLite, nil
	}

	// No host exists
	_, err := svc.SetupExperienceNextStep(ctx, "123")
	require.Error(t, err)

	mockListHostsLite = append(mockListHostsLite, &fleet.Host{UUID: "123"})

	// Host exists, nothing to do
	finished, err := svc.SetupExperienceNextStep(ctx, "123")
	require.NoError(t, err)
	assert.True(t, finished)
	assert.False(t, ds.InsertSoftwareInstallRequestFuncInvoked)
	assert.False(t, ds.InsertHostVPPSoftwareInstallFuncInvoked)
	assert.False(t, ds.NewHostScriptExecutionRequestFuncInvoked)
	assert.False(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)
}
