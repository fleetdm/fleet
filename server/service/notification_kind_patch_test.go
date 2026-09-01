package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Which notification an app-open skip is recorded on. The datastore is mocked,
// so only that decision is tested here. TestSaveHostSoftwareInstallResultAppOpenSkip
// tests that a real skip reaches this code.
func TestCreatePatchNotificationForEndUser(t *testing.T) {
	const (
		hostID          = uint(1)
		titleID         = uint(10)
		installerID     = uint(20)
		policyID        = uint(30)
		awaitingUUID    = "awaiting-uuid"
		createdUUID     = "created-uuid"
		unusableTitleID = uint(0)
	)

	cases := []struct {
		name string
		// PatchNotificationExistsForApp: a pending or dispatched notification
		// already lists this app and has not queued its installs
		exists bool
		// NotificationAwaitingDisplay: this host has a notification the end user
		// has not seen yet
		awaiting bool
		// host_software_installs.software_title_id
		noTitle bool

		wantCreated bool
		wantAppOn   string // "" means no app was recorded
	}{
		{
			name:        "the host has no patch notification",
			wantCreated: true,
			wantAppOn:   createdUUID,
		},
		{
			name:        "the app is added to the notification the end user has not seen yet",
			awaiting:    true,
			wantCreated: false,
			wantAppOn:   awaitingUUID,
		},
		{
			name:      "the app is already listed on a pending or dispatched notification",
			exists:    true,
			wantAppOn: "",
		},
		{
			// a notification that was displayed, queued its installs, failed or
			// expired no longer lists the app, so a new notification is created
			name:        "no notification lists the app",
			wantCreated: true,
			wantAppOn:   createdUUID,
		},
		{
			name:      "the install has no software title to record",
			noTitle:   true,
			wantAppOn: "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			notificationsSvc := &mock.MockNotificationsService{}
			svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler), notificationsSvc: notificationsSvc}

			ds.PatchNotificationExistsForAppFunc = func(_ context.Context, _ uint, _ uint) (bool, error) {
				return c.exists, nil
			}
			notificationsSvc.NotificationAwaitingDisplayFunc = func(_ context.Context, _ uint, _ string) (*notifications_api.EndUserNotification, error) {
				if !c.awaiting {
					return nil, nil
				}
				return &notifications_api.EndUserNotification{UUID: awaitingUUID}, nil
			}
			notificationsSvc.CreateNotificationFunc = func(_ context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.EndUserNotification, error) {
				assert.Equal(t, hostID, notification.HostID)
				assert.Equal(t, fleet.PatchNotificationKind, notification.Kind)
				assert.JSONEq(t, `{"reminder":false}`, string(notification.Payload))
				require.NotNil(t, notification.ExpiresAt, "a notification with no expiry never gives up")
				return &notifications_api.EndUserNotification{UUID: createdUUID}, nil
			}
			ds.NewPatchNotificationFunc = func(_ context.Context, _ string) error { return nil }

			var addedTo string
			var addedApp fleet.PatchNotificationApp
			ds.AddPatchNotificationAppFunc = func(_ context.Context, notificationUUID string, app fleet.PatchNotificationApp) error {
				addedTo = notificationUUID
				addedApp = app
				return nil
			}

			// the install orbit skipped because the app was open
			install := &fleet.HostSoftwareInstallerResult{
				InstallUUID:         "install-uuid",
				SoftwareTitleID:     new(titleID),
				SoftwareInstallerID: new(installerID),
				PolicyID:            new(policyID),
			}
			if c.noTitle {
				install.SoftwareTitleID = nil
			}

			err := svc.createPatchNotificationForEndUser(context.Background(), &fleet.Host{ID: hostID}, install)
			require.NoError(t, err)

			assert.Equal(t, c.wantCreated, notificationsSvc.CreateNotificationFuncInvoked)
			assert.Equal(t, c.wantCreated, ds.NewPatchNotificationFuncInvoked)
			assert.Equal(t, c.wantAppOn, addedTo)

			if c.wantAppOn != "" {
				assert.Equal(t, titleID, addedApp.SoftwareTitleID)
				require.NotNil(t, addedApp.SoftwareInstallerID)
				assert.Equal(t, installerID, *addedApp.SoftwareInstallerID)
				require.NotNil(t, addedApp.PolicyID)
				assert.Equal(t, policyID, *addedApp.PolicyID)
			}
		})
	}
}

// Which installs Update now queues for a notification's apps, and what the view
// returned to the end user says. The integration test covers that those installs
// then run with the app open, which only the unified queue's own SQL can show.
func TestPatchNotificationUpdateNow(t *testing.T) {
	const (
		hostID      = uint(1)
		titleID     = uint(10)
		installerID = uint(20)
		policyID    = uint(30)
	)

	cases := []struct {
		name   string
		status string
		// SetPatchNotificationInstallsQueued: false when another request already
		// recorded this notification's installs as queued, which is what a second
		// press of Update now sees
		alreadyQueued bool
		noInstaller   bool

		wantInstalls   int
		wantMarkQueued bool
	}{
		{
			name:           "queues an install per app and marks the notification",
			status:         notifications_api.EndUserNotificationDispatched,
			wantInstalls:   1,
			wantMarkQueued: true,
		},
		{
			name:           "installs are already queued, so pressing again queues nothing",
			status:         notifications_api.EndUserNotificationDispatched,
			alreadyQueued:  true,
			wantMarkQueued: true,
		},
		{
			name:   "a failed or expired notification queues nothing",
			status: notifications_api.EndUserNotificationFailed,
		},
		{
			// the installer was deleted, so software_installer_id is null
			name:           "an app with no installer is skipped, not fatal",
			status:         notifications_api.EndUserNotificationDispatched,
			noInstaller:    true,
			wantInstalls:   0,
			wantMarkQueued: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			kind := &patchNotificationKind{ds: ds, logger: slog.New(slog.DiscardHandler)}

			ds.SetPatchNotificationInstallsQueuedFunc = func(_ context.Context, _ string) (bool, error) {
				return !c.alreadyQueued, nil
			}
			ds.ListPatchNotificationAppsFunc = func(_ context.Context, _ string) ([]fleet.PatchNotificationAppDetail, error) {
				app := fleet.PatchNotificationAppDetail{
					SoftwareTitleID:     titleID,
					SoftwareInstallerID: new(installerID),
					PolicyID:            new(policyID),
				}
				if c.noInstaller {
					app.SoftwareInstallerID = nil
				}
				return []fleet.PatchNotificationAppDetail{app}, nil
			}

			var installs []fleet.HostSoftwareInstallOptions
			ds.InsertSoftwareInstallRequestFunc = func(_ context.Context, gotHostID uint, gotInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
				assert.Equal(t, hostID, gotHostID)
				assert.Equal(t, installerID, gotInstallerID)
				installs = append(installs, opts)
				return "", nil
			}

			// the view update now returns is built without reading anything back
			ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
			ds.GetDeviceAuthTokenIfFreshFunc = func(_ context.Context, _ uint, _ time.Duration) (string, error) {
				return "device-token", nil
			}

			// the end user pressed Update now on this notification
			view, err := kind.updateNow(context.Background(), &notifications_api.EndUserNotification{
				UUID: "notification-uuid", HostID: hostID, Status: c.status,
				Payload: patchNotificationFirstNoticePayload,
			})
			require.NoError(t, err)

			require.Len(t, installs, c.wantInstalls)
			for _, opts := range installs {
				assert.True(t, opts.IgnoreAppOpenQuery, "the end user asked for this, so the app being open must not stop it")
				require.NotNil(t, opts.PolicyID, "the install keeps its policy so it shows in Automation runs")
				assert.Equal(t, policyID, *opts.PolicyID)
			}
			assert.Equal(t, c.wantMarkQueued, ds.SetPatchNotificationInstallsQueuedFuncInvoked)
			assert.False(t, ds.GetPatchNotificationFuncInvoked, "installs_queued_at is not read back after writing it")

			if !c.wantMarkQueued {
				assert.Nil(t, view, "nothing changed, so the notification renders as it was")
				return
			}

			// the returned view is what the end user sees without a second request
			require.NotNil(t, view)
			require.Len(t, view.Items, 1)
			assert.Equal(t, "Installing...", view.Items[0].Status)
			for _, action := range view.Actions {
				assert.NotEqual(t, patchNotificationActionUpdateNow, action.ID, "the apps are already installing")
			}
		})
	}
}
