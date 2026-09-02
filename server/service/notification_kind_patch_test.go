package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingActivityWriter is a patchNotificationActivityWriter that records the
// one activity it was given, so a test can inspect its fields.
type capturingActivityWriter struct {
	invoked  bool
	activity activity_api.ActivityDetails
}

func (w *capturingActivityWriter) NewActivity(_ context.Context, _ *fleet.User, activity activity_api.ActivityDetails) error {
	w.invoked = true
	w.activity = activity
	return nil
}

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
			require.Equal(t, []notifications_api.NotificationAction{
				{ID: patchNotificationActionDismiss, Label: "Hide"},
			}, view.Actions, "the apps are already installing, so only Hide is offered")
		})
	}
}

// What the activity OnOutcome records: which apps and policies it names, which
// of the two notices it was for, and whether the outcome was a success or a
// failure. TestSaveHostSoftwareInstallResultAppOpenSkip and the integration
// suite exercise real script outcomes; this is the decision on its own.
func TestPatchNotificationOnOutcome(t *testing.T) {
	const hostID = uint(1)

	twoAppsTwoPolicies := []fleet.PatchNotificationAppDetail{
		{SoftwareTitleID: 10, Name: "AppOne", PolicyID: new(uint(30))},
		{SoftwareTitleID: 11, Name: "AppTwo", PolicyID: new(uint(31))},
	}

	cases := []struct {
		name string
		// notification.Payload: the reminder flag the view's copy already keys off
		reminder bool
		// the previous attempt's outcome, nil if this is the first attempt
		lastExitCode *int64

		outcome notifications_api.NotificationOutcome
		apps    []fleet.PatchNotificationAppDetail

		wantNoActivity bool
		wantStatus     string
		wantTimeBefore int
		wantTitles     []string
		wantPolicyIDs  []uint
	}{
		{
			name:           "the first notice, displayed, names every app and policy",
			outcome:        notifications_api.NotificationOutcome{Displayed: true, ExitCode: 0, ExecutionID: "exec-1"},
			apps:           twoAppsTwoPolicies,
			wantStatus:     "success",
			wantTimeBefore: 3600,
			wantTitles:     []string{"AppOne", "AppTwo"},
			wantPolicyIDs:  []uint{30, 31},
		},
		{
			name:           "the reminder, displayed, uses the reminder's time before",
			reminder:       true,
			outcome:        notifications_api.NotificationOutcome{Displayed: true, ExitCode: 0, ExecutionID: "exec-2"},
			apps:           twoAppsTwoPolicies,
			wantStatus:     "success",
			wantTimeBefore: 300,
			wantTitles:     []string{"AppOne", "AppTwo"},
			wantPolicyIDs:  []uint{30, 31},
		},
		{
			name:           "a first failure is recorded",
			outcome:        notifications_api.NotificationOutcome{Displayed: false, ExitCode: 41, ExecutionID: "exec-3"},
			apps:           twoAppsTwoPolicies,
			wantStatus:     "failed",
			wantTimeBefore: 3600,
			wantTitles:     []string{"AppOne", "AppTwo"},
			wantPolicyIDs:  []uint{30, 31},
		},
		{
			name:           "a repeat of the same failure is not recorded again",
			lastExitCode:   new(int64(41)),
			outcome:        notifications_api.NotificationOutcome{Displayed: false, ExitCode: 41, ExecutionID: "exec-4"},
			apps:           twoAppsTwoPolicies,
			wantNoActivity: true,
		},
		{
			name:           "a different failure after an earlier one is still recorded",
			lastExitCode:   new(int64(41)),
			outcome:        notifications_api.NotificationOutcome{Displayed: false, ExitCode: 42, ExecutionID: "exec-5"},
			apps:           twoAppsTwoPolicies,
			wantStatus:     "failed",
			wantTimeBefore: 3600,
			wantTitles:     []string{"AppOne", "AppTwo"},
			wantPolicyIDs:  []uint{30, 31},
		},
		{
			// a policy deleted after the notification was created leaves
			// patch_notification_apps.policy_id null, but its app is still named
			name:    "an app whose policy was deleted is still listed, without a policy id",
			outcome: notifications_api.NotificationOutcome{Displayed: true, ExitCode: 0, ExecutionID: "exec-6"},
			apps: []fleet.PatchNotificationAppDetail{
				{SoftwareTitleID: 10, Name: "AppOne", PolicyID: new(uint(30))},
				{SoftwareTitleID: 12, Name: "AppThree", PolicyID: nil},
			},
			wantStatus:     "success",
			wantTimeBefore: 3600,
			wantTitles:     []string{"AppOne", "AppThree"},
			wantPolicyIDs:  []uint{30},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			ds.ListPatchNotificationAppsFunc = func(_ context.Context, _ string) ([]fleet.PatchNotificationAppDetail, error) {
				return c.apps, nil
			}
			ds.HostLiteByIDFunc = func(_ context.Context, _ uint) (*fleet.HostLite, error) {
				return &fleet.HostLite{ID: hostID, ComputerName: "Test Host"}, nil
			}

			writer := &capturingActivityWriter{}
			kind := &patchNotificationKind{ds: ds, activities: writer, logger: slog.New(slog.DiscardHandler)}

			payload := patchNotificationFirstNoticePayload
			if c.reminder {
				payload = patchNotificationReminderPayload
			}
			notification := &notifications_api.EndUserNotification{
				UUID: "notification-uuid", HostID: hostID, Payload: payload, LastExitCode: c.lastExitCode,
			}

			err := kind.OnOutcome(context.Background(), notification, c.outcome)
			require.NoError(t, err)

			if c.wantNoActivity {
				assert.False(t, writer.invoked)
				return
			}

			require.True(t, writer.invoked)
			activity, ok := writer.activity.(fleet.ActivityTypeNotifiedEndUserBeforePatching)
			require.True(t, ok)

			assert.Equal(t, hostID, activity.HostID)
			assert.Equal(t, "Test Host", activity.HostDisplayName)
			assert.Equal(t, notification.UUID, activity.PatchNotificationUUID)
			assert.Equal(t, c.wantStatus, activity.Status)
			assert.Equal(t, c.wantTimeBefore, activity.TimeBefore)
			assert.Equal(t, c.outcome.ExecutionID, activity.ScriptExecutionID)

			assert.Equal(t, c.wantTitles, activity.SoftwareTitles)
			assert.Equal(t, c.wantPolicyIDs, activity.PolicyIDs)
		})
	}
}
