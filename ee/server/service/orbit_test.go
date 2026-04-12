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

		err := svc.recordCanceledSetupExperienceSoftwareActivities(ctx, hostID, hostUUID, hostDisplayName, results)
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

		var createdActivities []fleet.ActivityDetails
		var createdUser *fleet.User
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			createdUser = user
			createdActivities = append(createdActivities, activity)
			return nil
		}

		// The failed item that caused the cancellation
		failedTitleID := uint(999)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:            hostUUID,
				Name:                "FailedApp",
				Status:              fleet.SetupExperienceStatusFailure,
				SoftwareInstallerID: ptr.Uint(99),
				SoftwareTitleID:     &failedTitleID,
			},
			{
				HostUUID:                        hostUUID,
				Name:                            "DummyApp",
				Status:                          fleet.SetupExperienceStatusCancelled,
				SoftwareInstallerID:             &installerID,
				HostSoftwareInstallsExecutionID: ptr.String("exec-uuid-1"),
			},
		}

		err := svc.recordCanceledSetupExperienceSoftwareActivities(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Status should have been changed to failure
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[1].Status)

		// Update should have been called
		assert.True(t, ds.UpdateSetupExperienceStatusResultFuncInvoked)

		// Should have 2 activities: canceled install + canceled setup experience
		require.Len(t, createdActivities, 2)

		// First: canceled install software
		canceledAct, ok := createdActivities[0].(fleet.ActivityTypeCanceledInstallSoftware)
		require.True(t, ok, "expected ActivityTypeCanceledInstallSoftware, got %T", createdActivities[0])
		assert.Equal(t, hostID, canceledAct.HostID)
		assert.Equal(t, hostDisplayName, canceledAct.HostDisplayName)
		assert.Equal(t, "DummyApp", canceledAct.SoftwareTitle)
		assert.Equal(t, titleID, canceledAct.SoftwareTitleID)
		assert.True(t, canceledAct.FromSetupExperience, "FromSetupExperience should be true")
		assert.True(t, canceledAct.WasFromAutomation(), "WasFromAutomation should be true")

		// Second: canceled setup experience
		cseAct, ok := createdActivities[1].(fleet.ActivityTypeCanceledSetupExperience)
		require.True(t, ok, "expected ActivityTypeCanceledSetupExperience, got %T", createdActivities[1])
		assert.Equal(t, hostID, cseAct.HostID)
		assert.Equal(t, "FailedApp", cseAct.SoftwareTitle)
		assert.Equal(t, failedTitleID, cseAct.SoftwareTitleID)

		// Should be created with nil user (Fleet-initiated)
		assert.Nil(t, createdUser)
	})

	t.Run("VPP app cancelled emits canceled_install_app_store_app activity with FromSetupExperience", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		vppTeamID := uint(20)
		adamID := "12345"
		softwareTitleID := uint(200)

		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		var createdActivities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			createdActivities = append(createdActivities, activity)
			return nil
		}

		failedTitleID := uint(888)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:        hostUUID,
				Name:            "FailedVPP",
				Status:          fleet.SetupExperienceStatusFailure,
				VPPAppTeamID:    ptr.Uint(99),
				SoftwareTitleID: &failedTitleID,
			},
			{
				HostUUID:        hostUUID,
				Name:            "VPPApp",
				Status:          fleet.SetupExperienceStatusCancelled,
				VPPAppTeamID:    &vppTeamID,
				VPPAppAdamID:    &adamID,
				SoftwareTitleID: &softwareTitleID,
			},
		}

		err := svc.recordCanceledSetupExperienceSoftwareActivities(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Status should have been changed to failure
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[1].Status)

		// Should have 2 activities: canceled VPP + canceled setup experience
		require.Len(t, createdActivities, 2)

		// First: canceled install app store app
		canceledAct, ok := createdActivities[0].(fleet.ActivityTypeCanceledInstallAppStoreApp)
		require.True(t, ok, "expected ActivityTypeCanceledInstallAppStoreApp, got %T", createdActivities[0])
		assert.Equal(t, hostID, canceledAct.HostID)
		assert.Equal(t, hostDisplayName, canceledAct.HostDisplayName)
		assert.Equal(t, "VPPApp", canceledAct.SoftwareTitle)
		assert.Equal(t, softwareTitleID, canceledAct.SoftwareTitleID)
		assert.True(t, canceledAct.FromSetupExperience)
		assert.True(t, canceledAct.WasFromAutomation())

		// Second: canceled setup experience
		cseAct, ok := createdActivities[1].(fleet.ActivityTypeCanceledSetupExperience)
		require.True(t, ok)
		assert.Equal(t, "FailedVPP", cseAct.SoftwareTitle)
		assert.Equal(t, failedTitleID, cseAct.SoftwareTitleID)
	})

	t.Run("mixed cancelled and non-cancelled results", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		installerID := uint(30)
		titleID := uint(300)
		vppTeamID := uint(40)
		adamID := "67890"
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

		failedTitleID := uint(777)
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:            hostUUID,
				Name:                "FailedApp",
				Status:              fleet.SetupExperienceStatusFailure,
				SoftwareInstallerID: ptr.Uint(50),
				SoftwareTitleID:     &failedTitleID,
			},
			{
				HostUUID:            hostUUID,
				Name:                "SuccessApp",
				Status:              fleet.SetupExperienceStatusSuccess,
				SoftwareInstallerID: ptr.Uint(51),
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
				SoftwareTitleID: &vppTitleID,
			},
		}

		err := svc.recordCanceledSetupExperienceSoftwareActivities(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Only the two cancelled results should have their status changed
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[0].Status) // was already failed
		assert.Equal(t, fleet.SetupExperienceStatusSuccess, results[1].Status) // unchanged
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[2].Status) // cancelled -> failed
		assert.Equal(t, fleet.SetupExperienceStatusPending, results[3].Status) // unchanged
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[4].Status) // cancelled -> failed

		// Three activities: canceled sw install, canceled vpp install, canceled setup experience
		require.Len(t, activities, 3)

		swAct, ok := activities[0].(fleet.ActivityTypeCanceledInstallSoftware)
		require.True(t, ok)
		assert.Equal(t, "CancelledSW", swAct.SoftwareTitle)
		assert.True(t, swAct.FromSetupExperience)

		vppAct, ok := activities[1].(fleet.ActivityTypeCanceledInstallAppStoreApp)
		require.True(t, ok)
		assert.Equal(t, "CancelledVPP", vppAct.SoftwareTitle)
		assert.True(t, vppAct.FromSetupExperience)

		cseAct, ok := activities[2].(fleet.ActivityTypeCanceledSetupExperience)
		require.True(t, ok)
		assert.Equal(t, "FailedApp", cseAct.SoftwareTitle)
		assert.Equal(t, failedTitleID, cseAct.SoftwareTitleID)
	})

	t.Run("script cancellation does not trigger activity", func(t *testing.T) {
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

		err := svc.recordCanceledSetupExperienceSoftwareActivities(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Status should still be changed to failure
		assert.Equal(t, fleet.SetupExperienceStatusFailure, results[0].Status)
		// But no activity should be created for script cancellations
		assert.False(t, activityCreated)
	})

	t.Run("empty results returns nil", func(t *testing.T) {
		err := svc.recordCanceledSetupExperienceSoftwareActivities(ctx, hostID, hostUUID, hostDisplayName, nil)
		require.NoError(t, err)

		err = svc.recordCanceledSetupExperienceSoftwareActivities(ctx, hostID, hostUUID, hostDisplayName, []*fleet.SetupExperienceStatusResult{})
		require.NoError(t, err)
	})

	t.Run("no canceled_setup_experience activity when no failed software item", func(t *testing.T) {
		ds.UpdateSetupExperienceStatusResultFuncInvoked = false

		installerID := uint(10)
		titleID := uint(100)

		ds.UpdateSetupExperienceStatusResultFunc = func(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
			return nil
		}

		ds.GetSoftwareInstallerMetadataByIDFunc = func(ctx context.Context, id uint) (*fleet.SoftwareInstaller, error) {
			return &fleet.SoftwareInstaller{
				Name:    "installer.pkg",
				TitleID: &titleID,
			}, nil
		}

		var createdActivities []fleet.ActivityDetails
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			createdActivities = append(createdActivities, activity)
			return nil
		}

		// Only cancelled items, no failed item that triggered them
		results := []*fleet.SetupExperienceStatusResult{
			{
				HostUUID:                        hostUUID,
				Name:                            "DummyApp",
				Status:                          fleet.SetupExperienceStatusCancelled,
				SoftwareInstallerID:             &installerID,
				HostSoftwareInstallsExecutionID: ptr.String("exec-uuid-1"),
			},
		}

		err := svc.recordCanceledSetupExperienceSoftwareActivities(ctx, hostID, hostUUID, hostDisplayName, results)
		require.NoError(t, err)

		// Should only have the canceled install, no canceled_setup_experience
		require.Len(t, createdActivities, 1)
		_, ok := createdActivities[0].(fleet.ActivityTypeCanceledInstallSoftware)
		require.True(t, ok)
	})
}

func TestCanceledActivityWasFromAutomation(t *testing.T) {
	t.Run("CanceledInstallSoftware", func(t *testing.T) {
		act := fleet.ActivityTypeCanceledInstallSoftware{
			HostID:              1,
			HostDisplayName:     "host",
			SoftwareTitle:       "title",
			SoftwareTitleID:     1,
			FromSetupExperience: false,
		}
		assert.False(t, act.WasFromAutomation())

		act.FromSetupExperience = true
		assert.True(t, act.WasFromAutomation())
	})

	t.Run("CanceledInstallAppStoreApp", func(t *testing.T) {
		act := fleet.ActivityTypeCanceledInstallAppStoreApp{
			HostID:              1,
			HostDisplayName:     "host",
			SoftwareTitle:       "title",
			SoftwareTitleID:     1,
			FromSetupExperience: false,
		}
		assert.False(t, act.WasFromAutomation())

		act.FromSetupExperience = true
		assert.True(t, act.WasFromAutomation())
	})
}
