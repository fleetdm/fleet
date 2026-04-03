package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailCancelledSetupExperienceInstalls(t *testing.T) {
	ctx := context.Background()
	ds := new(mock.Store)
	baseSvc := new(svcmock.Service)

	svc := &Service{
		Service: baseSvc,
		ds:      ds,
		logger:  slog.Default(),
	}

	hostID := uint(42)
	hostUUID := "host-uuid-1"
	hostDisplayName := "Test Host"

	t.Run("skips non-cancelled results", func(t *testing.T) {
		var activityCreated bool
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activityCreated = true
			return nil
		}
		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:            hostUUID,
				Status:              fleet.SetupExperienceStatusPending,
				SoftwareInstallerID: ptr.Uint(1),
			},
			{
				HostUUID:     hostUUID,
				Status:       fleet.SetupExperienceStatusSuccess,
				VPPAppTeamID: ptr.Uint(2),
			},
			{
				HostUUID:     hostUUID,
				Status:       fleet.SetupExperienceStatusRunning,
				VPPAppTeamID: ptr.Uint(3),
			},
			{
				HostUUID:     hostUUID,
				Status:       fleet.SetupExperienceStatusFailure,
				VPPAppTeamID: ptr.Uint(4),
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)
		assert.False(t, activityCreated, "no activity should be created for non-cancelled results")
		assert.False(t, ds.UpdateSetupExperienceStatusResultFuncInvoked, "no update should be called for non-cancelled results")
	})

	t.Run("software package cancelled emits canceled_install_software activity with FromSetupExperience", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		installerID := uint(10)
		titleID := uint(100)

		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		ds.GetSoftwareInstallerMetadataByIDFunc = func(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
			require.Equal(t, installerID, id)
			return &fleet.SoftwareInstaller{
				Name:    "installer.pkg",
				TitleID: &titleID,
			}, nil
		}

		var createdActivity fleet.ActivityDetails
		var createdUser *fleet.User
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			createdUser = user
			createdActivity = activity
			return nil
		}

		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:                        hostUUID,
				Name:                            "DummyApp",
				Status:                          fleet.SetupExperienceStatusCancelled,
				SoftwareInstallerID:             &installerID,
				HostSoftwareInstallsExecutionID: ptr.String("exec-uuid-1"),
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Status should have been changed to failure
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[0].Status)

		// Update should have been called
		assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)

		// Activity should be a canceled install software type
		require.NotNil(t, createdActivity)
		canceledAct, ok := createdActivity.(fleet.ActivityTypeCanceledInstallSoftware)
		require.True(t, ok, "expected ActivityTypeCanceledInstallSoftware, got %T", createdActivity)
		assert.Equal(t, hostID, canceledAct.HostID)
		assert.Equal(t, hostDisplayName, canceledAct.HostDisplayName)
		assert.Equal(t, "DummyApp", canceledAct.SoftwareTitle)
		assert.Equal(t, titleID, canceledAct.SoftwareTitleID)
		assert.True(t, canceledAct.FromSetupExperience, "FromSetupExperience should be true")

		// WasFromAutomation should return true
		assert.True(t, canceledAct.WasFromAutomation(), "WasFromAutomation should be true for setup experience cancellations")

		// Should be created with nil user (Fleet-initiated)
		assert.Nil(t, createdUser)
	})

	t.Run("software package cancelled with missing metadata still works", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		installerID := uint(11)

		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		ds.GetSoftwareInstallerMetadataByIDFunc = func(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
			return nil, &notFoundError{}
		}

		var createdActivity fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			createdActivity = activity
			return nil
		}

		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:                        hostUUID,
				Name:                            "MissingApp",
				Status:                          fleet.SetupExperienceStatusCancelled,
				SoftwareInstallerID:             &installerID,
				HostSoftwareInstallsExecutionID: ptr.String("exec-uuid-2"),
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		require.NotNil(t, createdActivity)
		canceledAct, ok := createdActivity.(fleet.ActivityTypeCanceledInstallSoftware)
		require.True(t, ok)
		assert.Equal(t, uint(0), canceledAct.SoftwareTitleID, "SoftwareTitleID should be 0 when metadata not found")
		assert.True(t, canceledAct.FromSetupExperience)
	})

	t.Run("VPP app cancelled emits canceled_install_app_store_app activity with FromSetupExperience", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		vppTeamID := uint(20)
		adamID := "12345"
		platform := "darwin"
		softwareTitleID := uint(200)

		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		var createdActivity fleet.ActivityDetails
		var createdUser *fleet.User
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			createdUser = user
			createdActivity = activity
			return nil
		}

		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:        hostUUID,
				Name:            "VPPApp",
				Status:          fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				VPPAppPlatform:  &platform,
				SoftwareTitleID: &softwareTitleID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Status should have been changed to failure
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[0].Status)

		// Activity should be a canceled install app store app type
		require.NotNil(t, createdActivity)
		canceledAct, ok := createdActivity.(fleet.ActivityTypeCanceledInstallAppStoreApp)
		require.True(t, ok, "expected ActivityTypeCanceledInstallAppStoreApp, got %T", createdActivity)
		assert.Equal(t, hostID, canceledAct.HostID)
		assert.Equal(t, hostDisplayName, canceledAct.HostDisplayName)
		assert.Equal(t, "VPPApp", canceledAct.SoftwareTitle)
		assert.Equal(t, softwareTitleID, canceledAct.SoftwareTitleID)
		assert.True(t, canceledAct.FromSetupExperience, "FromSetupExperience should be true")

		// WasFromAutomation should return true
		assert.True(t, canceledAct.WasFromAutomation(), "WasFromAutomation should be true for setup experience cancellations")

		// Should be created with nil user (Fleet-initiated)
		assert.Nil(t, createdUser)
	})

	t.Run("VPP app cancelled without SoftwareTitleID defaults to 0", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		vppTeamID := uint(21)
		adamID := "99999"
		platform := "ios"

		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		var createdActivity fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			createdActivity = activity
			return nil
		}

		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:       hostUUID,
				Name:           "VPPAppNoTitle",
				Status:         fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:   &vppTeamID,
				VPPAppAdamID:   &adamID,
				VPPAppPlatform: &platform,
				// SoftwareTitleID is nil
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		require.NotNil(t, createdActivity)
		canceledAct, ok := createdActivity.(fleet.ActivityTypeCanceledInstallAppStoreApp)
		require.True(t, ok)
		assert.Equal(t, uint(0), canceledAct.SoftwareTitleID, "SoftwareTitleID should default to 0 when nil")
		assert.True(t, canceledAct.FromSetupExperience)
	})

	t.Run("mixed cancelled and non-cancelled results", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		installerID := uint(30)
		titleID := uint(300)
		vppTeamID := uint(40)
		adamID := "67890"
		platform := "darwin"
		vppTitleID := uint(400)

		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		ds.GetSoftwareInstallerMetadataByIDFunc = func(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
			return &fleet.SoftwareInstaller{
				Name:    "installer.pkg",
				TitleID: &titleID,
			}, nil
		}

		var activities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activities = append(activities, activity)
			return nil
		}

		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:            hostUUID,
				Name:                "SuccessApp",
				Status:              fleet.SetupExperienceStatusSuccess,
				SoftwareInstallerID: ptr.Uint(50),
			},
			{
				HostUUID:                        hostUUID,
				Name:                            "CancelledSW",
				Status:                          fleet.SetupExperienceStatusCancelled,
				SoftwareInstallerID:             &installerID,
				HostSoftwareInstallsExecutionID: ptr.String("exec-uuid-3"),
			},
			{
				HostUUID:     hostUUID,
				Name:         "PendingVPP",
				Status:       fleet.SetupExperienceStatusPending,
				VPPAppTeamID: ptr.Uint(60),
			},
			{
				HostUUID:        hostUUID,
				Name:            "CancelledVPP",
				Status:          fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				VPPAppPlatform:  &platform,
				SoftwareTitleID: &vppTitleID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Only the two cancelled results should have their status changed
		assert.Equal(t, fleet.SetupExperienceStatusSuccess, results[0].Status)
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[1].Status)
		assert.Equal(t, fleet.SetupExperienceStatusPending, results[2].Status)
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[3].Status)

		// Two activities should have been created
		require.Len(t, activities, 2)

		// First: canceled install software
		swAct, ok := activities[0].(fleet.ActivityTypeCanceledInstallSoftware)
		require.True(t, ok, "expected ActivityTypeCanceledInstallSoftware, got %T", activities[0])
		assert.Equal(t, "CancelledSW", swAct.SoftwareTitle)
		assert.Equal(t, titleID, swAct.SoftwareTitleID)
		assert.True(t, swAct.FromSetupExperience)

		// Second: canceled install app store app
		vppAct, ok := activities[1].(fleet.ActivityTypeCanceledInstallAppStoreApp)
		require.True(t, ok, "expected ActivityTypeCanceledInstallAppStoreApp, got %T", activities[1])
		assert.Equal(t, "CancelledVPP", vppAct.SoftwareTitle)
		assert.Equal(t, vppTitleID, vppAct.SoftwareTitleID)
		assert.True(t, vppAct.FromSetupExperience)
	})

	t.Run("script result is skipped (no activity created)", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		var activityCreated bool
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			activityCreated = true
			return nil
		}

		scriptID := uint(70)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:                hostUUID,
				Name:                    "setup.sh",
				Status:                  fleet.SetupExperienceStatusCancelled,
				SetupExperienceScriptID: &scriptID,
			},
		}

		err := svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Status should still be changed to failure
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[0].Status)
		// But no activity should be created for script cancellations (only software)
		assert.False(t, activityCreated)
	})

	t.Run("empty results returns nil", func(t *testing.T) {
		err := svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, nil)
		require.NoError(t, err)

		err = svc.failCancelledSetupExperienceInstalls(ctx, hostID, hostUUID, hostDisplayName, []*fleet.SetupExperienceStatusResult{})
		require.NoError(t, err)
	})
}

func TestCanceledActivityWasFromAutomation(t *testing.T) {
	t.Run("CanceledInstallSoftware", func(t *testing.T) {
		// FromSetupExperience = false -> WasFromAutomation returns false
		act := fleet.ActivityTypeCanceledInstallSoftware{
			HostID:              1,
			HostDisplayName:     "host",
			SoftwareTitle:       "title",
			SoftwareTitleID:     1,
			FromSetupExperience: false,
		}
		assert.False(t, act.WasFromAutomation())

		// FromSetupExperience = true -> WasFromAutomation returns true
		act.FromSetupExperience = true
		assert.True(t, act.WasFromAutomation())
	})

	t.Run("CanceledInstallAppStoreApp", func(t *testing.T) {
		// FromSetupExperience = false -> WasFromAutomation returns false
		act := fleet.ActivityTypeCanceledInstallAppStoreApp{
			HostID:              1,
			HostDisplayName:     "host",
			SoftwareTitle:       "title",
			SoftwareTitleID:     1,
			FromSetupExperience: false,
		}
		assert.False(t, act.WasFromAutomation())

		// FromSetupExperience = true -> WasFromAutomation returns true
		act.FromSetupExperience = true
		assert.True(t, act.WasFromAutomation())
	})
}
