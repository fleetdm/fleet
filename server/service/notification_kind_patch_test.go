package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
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

// stubNotificationService stands in for the notifications context. acts is what ActOnNotification reports, so a test can say another press got there first.
type stubNotificationService struct {
	acts         bool
	actInvoked   bool
	setStatus    string
	failedReason string
	delayInvoked bool
	delayPayload json.RawMessage
	setPayload   json.RawMessage
}

func (s *stubNotificationService) ActOnNotification(_ context.Context, _ string) (bool, error) {
	s.actInvoked = true
	return s.acts, nil
}

func (s *stubNotificationService) SetNotificationStatus(_ context.Context, _ string, status string, reason *string, _ []string) error {
	s.setStatus = status
	if reason != nil {
		s.failedReason = *reason
	}
	return nil
}

func (s *stubNotificationService) SetNotificationPayload(_ context.Context, _ string, payload json.RawMessage) error {
	s.setPayload = payload
	return nil
}

func (s *stubNotificationService) DelayNotification(_ context.Context, _ string, _ time.Time, payload json.RawMessage) error {
	s.delayInvoked = true
	s.delayPayload = payload
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
		// ListLastTitleInstallDataForHosts: another install for this app's title, which is what a stale skip from a policy re-fire finds
		otherInstall *fleet.HostLastInstallData
		// NotificationAwaitingDisplay: this host has a notification the end user
		// has not seen yet
		awaiting            bool
		awaitingRemindsNext bool
		// the skipped install carries no software title, so there is no app to name
		noTitle bool
		// NewPatchNotification: the patch_notifications row can't be written
		newPatchFails bool

		wantErr     bool
		wantCreated bool
		wantAppOn   string // "" means no app was recorded
	}{
		{
			name:        "a host with no patch notification gets a new one listing the app",
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
			name:                "the app gets its own notification when the one awaiting display is the 5 minute reminder",
			awaiting:            true,
			awaitingRemindsNext: true,
			wantCreated:         true,
			wantAppOn:           createdUUID,
		},
		{
			name:      "an app already listed on a pending or dispatched notification is not listed twice",
			exists:    true,
			wantAppOn: "",
		},
		{
			name:         "an app that already has an install queued is not notified about again",
			otherInstall: &fleet.HostLastInstallData{Status: new(fleet.SoftwareInstallPending)},
			wantAppOn:    "",
		},
		{
			// the queued install runs the app open query, so it skips too and patches nothing
			name:         "an app whose queued install would skip while it is open is still notified about",
			otherInstall: &fleet.HostLastInstallData{Status: new(fleet.SoftwareInstallPending), OverridePreInstallQuery: true},
			wantCreated:  true,
			wantAppOn:    createdUUID,
		},
		{
			// the notification that queued that install is acted, so it no longer lists the app
			name:         "an app whose last install failed gets a new notification",
			otherInstall: &fleet.HostLastInstallData{Status: new(fleet.SoftwareInstallFailed)},
			wantCreated:  true,
			wantAppOn:    createdUUID,
		},
		{
			name:      "an install with no software title records no notification",
			noTitle:   true,
			wantAppOn: "",
		},
		{
			// Leaving the app unlisted is what lets the next skip start over, since
			// PatchNotificationExistsForApp is what makes this function return early.
			name:          "a failure recording the patch notification leaves the app unlisted so it can be retried",
			newPatchFails: true,
			wantErr:       true,
			wantCreated:   true,
			wantAppOn:     "",
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
			ds.ListLastTitleInstallDataForHostsFunc = func(_ context.Context, _ []uint, _ []uint) (map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData, error) {
				if c.otherInstall == nil {
					return nil, nil
				}
				return map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData{
					{HostID: hostID, SoftwareTitleID: titleID}: {c.otherInstall},
				}, nil
			}
			notificationsSvc.NotificationAwaitingDisplayFunc = func(_ context.Context, _ uint, _ string) (*notifications_api.EndUserNotification, error) {
				if !c.awaiting {
					return nil, nil
				}
				return &notifications_api.EndUserNotification{UUID: awaitingUUID, Payload: patchNotificationFirstNoticePayload}, nil
			}
			ds.GetPatchNotificationFunc = func(_ context.Context, _ string) (*fleet.PatchNotification, error) {
				if !c.awaitingRemindsNext {
					return nil, nil
				}
				return &fleet.PatchNotification{InstallAt: new(time.Now().UTC().Add(2 * time.Minute))}, nil
			}
			notificationsSvc.CreateNotificationFunc = func(_ context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.EndUserNotification, error) {
				require.Equal(t, hostID, notification.HostID)
				require.Equal(t, fleet.PatchNotificationKind, notification.Kind)
				require.JSONEq(t, `{"reminder":false}`, string(notification.Payload))
				require.NotNil(t, notification.ExpiresAt, "a notification with no expiry never gives up")
				return &notifications_api.EndUserNotification{UUID: createdUUID}, nil
			}
			ds.NewPatchNotificationFunc = func(_ context.Context, _ string) error {
				if c.newPatchFails {
					return errors.New("insert failed")
				}
				return nil
			}

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
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, c.wantCreated, notificationsSvc.CreateNotificationFuncInvoked)
			require.Equal(t, c.wantCreated, ds.NewPatchNotificationFuncInvoked)
			require.Equal(t, c.wantAppOn, addedTo)

			if c.wantAppOn != "" {
				require.Equal(t, titleID, addedApp.SoftwareTitleID)
				require.NotNil(t, addedApp.SoftwareInstallerID)
				require.Equal(t, installerID, *addedApp.SoftwareInstallerID)
				require.NotNil(t, addedApp.PolicyID)
				require.Equal(t, policyID, *addedApp.PolicyID)
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
	appJoinedAt := time.Now().UTC().Add(-time.Hour)

	cases := []struct {
		name   string
		status string
		// GetHostLastInstallData: the app is already on the host's queue, which is
		// what a second press of Update now finds
		alreadyPending bool
		// ListLastTitleInstallDataForHosts: when Fleet last installed this app, nil when it never did
		lastInstalledAt *time.Time
		noInstaller     bool
		installFails    bool
		// ActOnNotification: another press marked the notification acted first
		alreadyActed bool

		wantErr       bool
		wantInstalls  int
		wantActionTry bool
		// the status shown against the app in the returned view, "Updating..." when empty
		wantItemStatus string
	}{
		{
			name:          "Update now queues an install per app and acts on the notification",
			status:        notifications_api.EndUserNotificationDispatched,
			wantInstalls:  1,
			wantActionTry: true,
		},
		{
			name:          "a second press arriving at the same time queues nothing",
			status:        notifications_api.EndUserNotificationDispatched,
			alreadyActed:  true,
			wantInstalls:  0,
			wantActionTry: true,
		},
		{
			name:           "an app already on the host's queue is not queued a second time",
			status:         notifications_api.EndUserNotificationDispatched,
			alreadyPending: true,
			wantActionTry:  true,
		},
		{
			name:            "an app updated after it joined the notification is not installed again",
			status:          notifications_api.EndUserNotificationDispatched,
			lastInstalledAt: new(appJoinedAt.Add(time.Minute)),
			wantInstalls:    0,
			wantActionTry:   true,
			wantItemStatus:  "Updated",
		},
		{
			name:            "an app last updated before it joined the notification is still installed",
			status:          notifications_api.EndUserNotificationDispatched,
			lastInstalledAt: new(appJoinedAt.Add(-time.Minute)),
			wantInstalls:    1,
			wantActionTry:   true,
		},
		{
			name:   "Update now on a failed or expired notification queues nothing",
			status: notifications_api.EndUserNotificationFailed,
		},
		{
			// the installer was deleted, so software_installer_id is null
			name:           "an app whose installer was deleted is skipped and the action still succeeds",
			status:         notifications_api.EndUserNotificationDispatched,
			noInstaller:    true,
			wantInstalls:   0,
			wantActionTry:  true,
			wantItemStatus: "Failed",
		},
		{
			name:          "a queueing failure puts the notification back to dispatched",
			status:        notifications_api.EndUserNotificationDispatched,
			installFails:  true,
			wantErr:       true,
			wantActionTry: true,
		},
		{
			// pending is where "Remind me 5 minutes before" leaves a notification
			name:          "a queueing failure on a pending notification puts it back to pending",
			status:        notifications_api.EndUserNotificationPending,
			installFails:  true,
			wantErr:       true,
			wantActionTry: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			notificationSvc := &stubNotificationService{acts: !c.alreadyActed}
			kind := &patchNotificationKind{
				ds: ds, notificationSvc: notificationSvc, logger: slog.New(slog.DiscardHandler),
			}
			ds.ListLastTitleInstallDataForHostsFunc = func(_ context.Context, _ []uint, titleIDs []uint) (map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData, error) {
				var lastInstall *fleet.HostLastInstallData
				switch {
				case c.alreadyPending:
					lastInstall = &fleet.HostLastInstallData{Status: new(fleet.SoftwareInstallPending)}
				case c.lastInstalledAt != nil:
					lastInstall = &fleet.HostLastInstallData{Status: new(fleet.SoftwareInstalled), UpdatedAt: *c.lastInstalledAt}
				default:
					return nil, nil
				}
				byTitle := make(map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData, len(titleIDs))
				for _, id := range titleIDs {
					byTitle[fleet.HostSoftwareTitleKey{HostID: hostID, SoftwareTitleID: id}] =
						[]*fleet.HostLastInstallData{lastInstall}
				}
				return byTitle, nil
			}
			// the view update now returns reports the app's own install, which is already
			// installed when Fleet found a newer install than the app's row
			ds.ListPatchNotificationAppInstallStatusesFunc = func(_ context.Context, _ string) (map[uint]fleet.SoftwareInstallerStatus, error) {
				if c.lastInstalledAt == nil || !c.lastInstalledAt.After(appJoinedAt) {
					return nil, nil
				}
				return map[uint]fleet.SoftwareInstallerStatus{titleID: fleet.SoftwareInstalled}, nil
			}
			ds.SetPatchNotificationAppsQueuedFunc = func(_ context.Context, _ string, _ []uint) error { return nil }
			ds.ListPatchNotificationAppsFunc = func(_ context.Context, _ string) ([]fleet.PatchNotificationAppDetail, error) {
				app := fleet.PatchNotificationAppDetail{
					SoftwareTitleID:     titleID,
					SoftwareInstallerID: new(installerID),
					PolicyID:            new(policyID),
					CreatedAt:           appJoinedAt,
				}
				if c.noInstaller {
					app.SoftwareInstallerID = nil
				}
				return []fleet.PatchNotificationAppDetail{app}, nil
			}

			var installs []fleet.HostSoftwareInstallOptions
			ds.InsertSoftwareInstallRequestFunc = func(_ context.Context, gotHostID uint, gotInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
				require.Equal(t, hostID, gotHostID)
				require.Equal(t, installerID, gotInstallerID)
				if c.installFails {
					return "", errors.New("insert failed")
				}
				installs = append(installs, opts)
				return "", nil
			}

			// the view update now returns is built without reading anything back
			ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
			ds.GetDeviceAuthTokenIfFreshFunc = func(_ context.Context, _ uint, _ time.Duration) (string, error) {
				return "device-token", nil
			}
			ds.GetPatchNotificationFunc = func(_ context.Context, _ string) (*fleet.PatchNotification, error) {
				return nil, nil
			}

			// the end user pressed Update now on this notification
			view, err := kind.updateNow(context.Background(), &notifications_api.EndUserNotification{
				UUID: "notification-uuid", HostID: hostID, Status: c.status,
				Payload: patchNotificationFirstNoticePayload,
			})
			if c.wantErr {
				require.Error(t, err)
				require.Equal(t, c.status, notificationSvc.setStatus,
					"the notification goes back to the status it had so the next press can finish queueing")
				return
			}
			require.Empty(t, notificationSvc.setStatus)
			require.NoError(t, err)

			require.Len(t, installs, c.wantInstalls)
			for _, opts := range installs {
				require.False(t, opts.OverridePreInstallQuery, "the end user asked for this, so the app being open must not stop it")
				require.NotNil(t, opts.PolicyID, "the install keeps its policy so it shows in Automation runs")
				require.Equal(t, policyID, *opts.PolicyID)
			}
			require.Equal(t, c.wantActionTry, notificationSvc.actInvoked)

			if !c.wantActionTry {
				require.Nil(t, view, "nothing changed, so the notification renders as it was")
				return
			}

			// the returned view is what the end user sees without a second request
			require.NotNil(t, view)
			require.Len(t, view.Items, 1)
			wantItemStatus := c.wantItemStatus
			if wantItemStatus == "" {
				wantItemStatus = "Updating..."
			}
			require.Equal(t, wantItemStatus, view.Items[0].Status)
			require.Equal(t, []notifications_api.NotificationAction{
				{ID: patchNotificationActionDismiss, Label: "Hide"},
			}, view.Actions, "the apps are already installing, so only Hide is offered")
		})
	}
}

func TestPatchNotificationUpdateNowResumesAfterFailure(t *testing.T) {
	const (
		hostID          = uint(1)
		firstTitleID    = uint(10)
		secondTitleID   = uint(11)
		firstInstaller  = uint(20)
		secondInstaller = uint(21)
	)

	ds := new(mock.Store)
	notificationSvc := &stubNotificationService{acts: true}
	kind := &patchNotificationKind{
		ds: ds, notificationSvc: notificationSvc, logger: slog.New(slog.DiscardHandler),
	}

	ds.ListPatchNotificationAppInstallStatusesFunc = func(_ context.Context, _ string) (map[uint]fleet.SoftwareInstallerStatus, error) {
		return nil, nil
	}
	queued := map[uint]struct{}{}
	ds.ListPatchNotificationAppsFunc = func(_ context.Context, _ string) ([]fleet.PatchNotificationAppDetail, error) {
		apps := []fleet.PatchNotificationAppDetail{
			{SoftwareTitleID: firstTitleID, SoftwareInstallerID: new(firstInstaller)},
			{SoftwareTitleID: secondTitleID, SoftwareInstallerID: new(secondInstaller)},
		}
		for i, app := range apps {
			if _, ok := queued[app.SoftwareTitleID]; ok {
				apps[i].InstallQueued = true
			}
		}
		return apps, nil
	}
	ds.SetPatchNotificationAppsQueuedFunc = func(_ context.Context, _ string, softwareTitleIDs []uint) error {
		for _, softwareTitleID := range softwareTitleIDs {
			queued[softwareTitleID] = struct{}{}
		}
		return nil
	}

	// the first app's install finishes between the two presses, so it is no
	// longer pending by the time the second press runs
	ds.ListLastTitleInstallDataForHostsFunc = func(_ context.Context, hostIDs []uint, _ []uint) (map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData, error) {
		if _, firstAppQueued := queued[firstTitleID]; !firstAppQueued {
			return nil, nil
		}
		return map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData{
			{HostID: hostIDs[0], SoftwareTitleID: firstTitleID}: {{Status: new(fleet.SoftwareInstalled)}},
		}, nil
	}

	var installed []uint
	secondInstallerFails := true
	ds.InsertSoftwareInstallRequestFunc = func(_ context.Context, _ uint, installerID uint, _ fleet.HostSoftwareInstallOptions) (string, error) {
		if installerID == secondInstaller && secondInstallerFails {
			return "", errors.New("insert failed")
		}
		installed = append(installed, installerID)
		return "", nil
	}

	ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
	ds.GetDeviceAuthTokenIfFreshFunc = func(_ context.Context, _ uint, _ time.Duration) (string, error) {
		return "device-token", nil
	}
	ds.GetPatchNotificationFunc = func(_ context.Context, _ string) (*fleet.PatchNotification, error) {
		return nil, nil
	}

	notification := &notifications_api.EndUserNotification{
		UUID: "notification-uuid", HostID: hostID,
		Status:  notifications_api.EndUserNotificationDispatched,
		Payload: patchNotificationFirstNoticePayload,
	}

	_, err := kind.updateNow(context.Background(), notification)
	require.Error(t, err)
	require.Equal(t, []uint{firstInstaller}, installed)
	require.Equal(t, notifications_api.EndUserNotificationDispatched, notificationSvc.setStatus)

	// the end user presses again, and this time the second app's install works
	secondInstallerFails = false
	notificationSvc.setStatus = ""
	_, err = kind.updateNow(context.Background(), notification)
	require.NoError(t, err)
	require.Equal(t, []uint{firstInstaller, secondInstaller}, installed,
		"the first app is not queued a second time")
	require.True(t, notificationSvc.actInvoked)
	require.Empty(t, notificationSvc.setStatus)
}

// Once the installs are out the toast polls Render, so each app reports where its own install got
// to rather than every app reading "Updating..." until the toast is closed.
func TestPatchNotificationRenderInstallStatuses(t *testing.T) {
	const (
		hostID  = uint(1)
		titleID = uint(10)
	)
	appJoinedAt := time.Now().UTC().Add(-time.Hour)

	cases := []struct {
		name string
		// the notification's status, which is what says the end user pressed Update now
		notificationStatus string
		// the status the datastore reports for the app's install, nil when it has none to report
		installStatus *fleet.SoftwareInstallerStatus
		// software_installers row is gone, so the app can never be queued
		noInstaller bool

		wantStatus        string
		wantInstallStatus string
	}{
		{
			name:               "an app on a notification the end user has not acted on shows no status",
			notificationStatus: notifications_api.EndUserNotificationDispatched,
			wantStatus:         "",
			wantInstallStatus:  "",
		},
		{
			name:               "an app whose install is still queued shows updating",
			notificationStatus: notifications_api.EndUserNotificationActed,
			installStatus:      new(fleet.SoftwareInstallPending),
			wantStatus:         "Updating...",
			wantInstallStatus:  "pending_install",
		},
		{
			name:               "an app Fleet has no install record for yet shows updating",
			notificationStatus: notifications_api.EndUserNotificationActed,
			wantStatus:         "Updating...",
			wantInstallStatus:  "pending_install",
		},
		{
			name:               "an app whose install finished shows updated",
			notificationStatus: notifications_api.EndUserNotificationActed,
			installStatus:      new(fleet.SoftwareInstalled),
			wantStatus:         "Updated",
			wantInstallStatus:  "installed",
		},
		{
			name:               "an app whose install failed shows failed",
			notificationStatus: notifications_api.EndUserNotificationActed,
			installStatus:      new(fleet.SoftwareInstallFailed),
			wantStatus:         "Failed",
			wantInstallStatus:  "failed_install",
		},
		{
			name:               "an app whose install was cancelled shows no status",
			notificationStatus: notifications_api.EndUserNotificationActed,
			installStatus:      new(fleet.SoftwareInstallerStatus("canceled_install")),
			wantStatus:         "",
			wantInstallStatus:  "",
		},
		{
			name:               "an app whose install reports a status other than installed or failed shows updating",
			notificationStatus: notifications_api.EndUserNotificationActed,
			installStatus:      new(fleet.SoftwareInstallerStatus("pending_uninstall")),
			wantStatus:         "Updating...",
			wantInstallStatus:  "pending_install",
		},
		{
			name:               "an app whose installer was deleted shows failed rather than updating forever",
			notificationStatus: notifications_api.EndUserNotificationActed,
			noInstaller:        true,
			wantStatus:         "Failed",
			wantInstallStatus:  "failed_install",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			kind := &patchNotificationKind{
				ds: ds, notificationSvc: &stubNotificationService{acts: true}, logger: slog.New(slog.DiscardHandler),
			}
			ds.ListPatchNotificationAppsFunc = func(_ context.Context, _ string) ([]fleet.PatchNotificationAppDetail, error) {
				app := fleet.PatchNotificationAppDetail{
					SoftwareTitleID:     titleID,
					SoftwareInstallerID: new(uint(20)),
					Name:                "1Password",
					CreatedAt:           appJoinedAt,
				}
				if c.noInstaller {
					app.SoftwareInstallerID = nil
				}
				return []fleet.PatchNotificationAppDetail{app}, nil
			}
			ds.ListPatchNotificationAppInstallStatusesFunc = func(_ context.Context, _ string) (map[uint]fleet.SoftwareInstallerStatus, error) {
				if c.installStatus == nil {
					return nil, nil
				}
				return map[uint]fleet.SoftwareInstallerStatus{titleID: *c.installStatus}, nil
			}
			ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) { return &fleet.AppConfig{}, nil }
			ds.GetDeviceAuthTokenIfFreshFunc = func(_ context.Context, _ uint, _ time.Duration) (string, error) {
				return "device-token", nil
			}
			ds.GetPatchNotificationFunc = func(_ context.Context, _ string) (*fleet.PatchNotification, error) {
				return nil, nil
			}

			view, err := kind.Render(context.Background(), &notifications_api.EndUserNotification{
				UUID: "notification-uuid", HostID: hostID,
				Status:  c.notificationStatus,
				Payload: patchNotificationFirstNoticePayload,
			})
			require.NoError(t, err)
			require.Len(t, view.Items, 1)
			require.Equal(t, c.wantStatus, view.Items[0].Status)
			require.Equal(t, c.wantInstallStatus, view.Items[0].InstallStatus)
		})
	}
}

// A notification whose apps are gone, after an admin deletes the title, fails rather than retrying.
func TestPatchNotificationRenderWithNoApps(t *testing.T) {
	ds := new(mock.Store)
	notificationSvc := &stubNotificationService{acts: true}
	kind := &patchNotificationKind{
		ds: ds, notificationSvc: notificationSvc, logger: slog.New(slog.DiscardHandler),
	}
	ds.ListPatchNotificationAppsFunc = func(_ context.Context, _ string) ([]fleet.PatchNotificationAppDetail, error) {
		return nil, nil
	}

	view, err := kind.Render(context.Background(), &notifications_api.EndUserNotification{
		UUID: "notification-uuid", HostID: 1,
		Status:  notifications_api.EndUserNotificationDispatched,
		Payload: patchNotificationFirstNoticePayload,
	})
	require.Error(t, err)
	require.Nil(t, view)
	require.Equal(t, notifications_api.EndUserNotificationReasonNothingToShow, notificationSvc.failedReason)
}

func TestShouldNotificationBeReminder(t *testing.T) {
	now := time.Now().UTC()

	cases := []struct {
		name              string
		installAt         *time.Time
		displayedReminder bool
		want              bool
	}{
		{"a notification with no install_at is not the reminder", nil, true, false},
		{"a notification with install_at more than 5 minutes ahead is not the reminder", new(now.Add(6 * time.Minute)), false, false},
		{"a notification with install_at within 5 minutes is the reminder", new(now.Add(4 * time.Minute)), false, true},
		{"a displayed reminder past install_at is still the reminder", new(now.Add(-time.Minute)), true, true},
		{"a notification past install_at that was not displayed as the reminder is not the reminder", new(now.Add(-time.Minute)), false, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := shouldNotificationBeReminder(&fleet.PatchNotification{InstallAt: c.installAt}, c.displayedReminder, now)
			require.Equal(t, c.want, got)
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
		name          string
		untilDeadline time.Duration
		noDeadline    bool
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
			name:           "a displayed first notice records an activity naming every app and policy",
			noDeadline:     true,
			outcome:        notifications_api.NotificationOutcome{Displayed: true, ExitCode: 0, ExecutionID: "exec-1"},
			apps:           twoAppsTwoPolicies,
			wantStatus:     "success",
			wantTimeBefore: 3600,
			wantTitles:     []string{"AppOne", "AppTwo"},
			wantPolicyIDs:  []uint{30, 31},
		},
		{
			name:           "a toast displayed inside the reminder window records five minutes as its time before",
			untilDeadline:  4 * time.Minute,
			outcome:        notifications_api.NotificationOutcome{Displayed: true, ExitCode: 0, ExecutionID: "exec-2"},
			apps:           twoAppsTwoPolicies,
			wantStatus:     "success",
			wantTimeBefore: 300,
			wantTitles:     []string{"AppOne", "AppTwo"},
			wantPolicyIDs:  []uint{30, 31},
		},
		{
			name:           "a toast displayed after its deadline passed records an hour as its time before",
			untilDeadline:  -time.Minute,
			outcome:        notifications_api.NotificationOutcome{Displayed: true, ExitCode: 0, ExecutionID: "exec-7"},
			apps:           twoAppsTwoPolicies,
			wantStatus:     "success",
			wantTimeBefore: 3600,
			wantTitles:     []string{"AppOne", "AppTwo"},
			wantPolicyIDs:  []uint{30, 31},
		},
		{
			name:           "a screen locked failure is recorded the first time it happens",
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
			// the deadline the display lands on, which the activity reports back
			var gotLeadTime time.Duration
			deadline := time.Now().UTC().Add(time.Hour)
			ds.SetPatchNotificationInstallAtFunc = func(_ context.Context, _ string, installAt time.Time) (time.Time, error) {
				gotLeadTime = time.Until(installAt).Round(time.Minute)
				return deadline, nil
			}

			ds.GetPatchNotificationFunc = func(_ context.Context, _ string) (*fleet.PatchNotification, error) {
				if c.noDeadline || c.untilDeadline == 0 {
					return nil, nil
				}
				return &fleet.PatchNotification{InstallAt: new(time.Now().UTC().Add(c.untilDeadline))}, nil
			}

			writer := &capturingActivityWriter{}
			notificationSvc := &stubNotificationService{}
			kind := &patchNotificationKind{
				ds: ds, activities: writer, notificationSvc: notificationSvc, logger: slog.New(slog.DiscardHandler),
			}

			notification := &notifications_api.EndUserNotification{
				UUID: "notification-uuid", HostID: hostID, Payload: patchNotificationFirstNoticePayload,
				LastExitCode: c.lastExitCode, Status: notifications_api.EndUserNotificationDispatched,
			}

			err := kind.OnOutcome(context.Background(), notification, c.outcome)
			require.NoError(t, err)

			if c.wantNoActivity {
				require.False(t, writer.invoked)
				return
			}

			require.True(t, writer.invoked)
			activity, ok := writer.activity.(fleet.ActivityTypeNotifiedEndUserBeforePatching)
			require.True(t, ok)

			require.Equal(t, hostID, activity.HostID)
			require.Equal(t, "Test Host", activity.HostDisplayName)
			require.Equal(t, notification.UUID, activity.PatchNotificationUUID)
			require.Equal(t, c.wantStatus, activity.Status)
			require.Equal(t, c.wantTimeBefore, activity.TimeBefore)

			// only a displayed outcome sets install_at, from the lead time of the toast that reached the screen
			wantDisplayedPayload := patchNotificationFirstNoticePayload
			if c.wantTimeBefore == 300 {
				wantDisplayedPayload = patchNotificationReminderPayload
			}
			if c.wantStatus != "success" {
				require.False(t, ds.SetPatchNotificationInstallAtFuncInvoked)
				require.Nil(t, activity.InstallAt)
				require.Nil(t, notificationSvc.setPayload, "nothing was displayed, so there is nothing to record")
			} else {
				require.True(t, ds.SetPatchNotificationInstallAtFuncInvoked)
				require.Equal(t, time.Duration(c.wantTimeBefore)*time.Second, gotLeadTime)
				require.NotNil(t, activity.InstallAt)
				require.Equal(t, deadline, *activity.InstallAt)
				require.JSONEq(t, string(wantDisplayedPayload), string(notificationSvc.setPayload))
			}
			require.Equal(t, c.outcome.ExecutionID, activity.ScriptExecutionID)

			require.Equal(t, c.wantTitles, activity.SoftwareTitles)
			require.Equal(t, c.wantPolicyIDs, activity.PolicyIDs)
		})
	}
}

func TestRemindAndInstallDuePatches(t *testing.T) {
	const (
		hostID           = uint(1)
		oneTitleID       = uint(10)
		twoTitleID       = uint(11)
		oneInstallerID   = uint(20)
		twoInstallerID   = uint(21)
		policyID         = uint(30)
		installerVersion = "2.0.0"
	)

	// Both apps are a version behind what their installer would put on the host.
	behind := map[uint]string{oneTitleID: "1.0.0", twoTitleID: "1.0.0"}

	displayedAt := time.Now().UTC().Add(-55 * time.Minute)
	appAddedAt := time.Now().UTC().Add(-50 * time.Minute)

	cases := []struct {
		name string
		// how far the deadline is from now, negative once it has passed
		untilDeadline time.Duration
		displayed     bool
		reminder      bool
		// software inventory by software title id
		installedVersions map[uint]string
		// when Fleet last finished installing the first app, if it did
		lastInstalled *time.Time
		// the first app's install request fails, which must not stop the second being queued
		firstInstallFails bool
		// an install for the first app is already on the host's queue
		pendingInstall bool
		// that pending install runs the app open query, so it skips while the app is open
		pendingInstallSkipsWhenOpen bool
		// ActOnNotification: an Update now got there first
		alreadyActed bool
		// the notification is already acted, which an earlier pass stopping part way through leaves behind
		statusActed bool
		hostOffline bool

		wantReminder        bool
		wantDeadlineCleared bool
		wantInstalls        []uint
		// the pass tried to take the notification, whether or not it got it
		wantActed       bool
		wantAppsDropped []uint
	}{
		{
			name:              "an app updated during the hour is dropped from the reminder",
			untilDeadline:     4 * time.Minute,
			displayed:         true,
			installedVersions: map[uint]string{oneTitleID: installerVersion, twoTitleID: "1.0.0"},
			wantReminder:      true,
			wantAppsDropped:   []uint{oneTitleID},
		},
		{
			name:              "every app updated during the hour means no reminder and nothing to install",
			untilDeadline:     4 * time.Minute,
			displayed:         true,
			installedVersions: map[uint]string{oneTitleID: installerVersion, twoTitleID: installerVersion},
			wantActed:         true,
			wantAppsDropped:   []uint{oneTitleID, twoTitleID},
		},
		{
			// Version strings are not compared, only matched, so a host carrying something other than
			// what the installer would put down is treated as still needing the update.
			name:              "an app the host is on a different version of is still updated at the deadline",
			untilDeadline:     -time.Minute,
			displayed:         true,
			reminder:          true,
			installedVersions: map[uint]string{oneTitleID: "9.0.0", twoTitleID: "1.0.0"},
			wantActed:         true,
			wantInstalls:      []uint{oneInstallerID, twoInstallerID},
		},
		{
			// Fleet's own install record catches a My device self-service update before the host's
			// software inventory has refreshed to show it
			name:              "an app Fleet installed since it was added to the notification is not installed again",
			untilDeadline:     -time.Minute,
			displayed:         true,
			reminder:          true,
			installedVersions: behind,
			lastInstalled:     new(appAddedAt.Add(time.Minute)),
			wantActed:         true,
			wantInstalls:      []uint{twoInstallerID},
			wantAppsDropped:   []uint{oneTitleID},
		},
		{
			// the app was added after Fleet's install finished, because its policy failed anyway, so
			// that install is no evidence the app is up to date
			name:              "an app Fleet installed before it was added to the notification is still installed",
			untilDeadline:     -time.Minute,
			displayed:         true,
			reminder:          true,
			installedVersions: behind,
			lastInstalled:     new(appAddedAt.Add(-time.Minute)),
			wantActed:         true,
			wantInstalls:      []uint{oneInstallerID, twoInstallerID},
		},
		{
			// a pass that misses the reminder window leaves the first notice as the last one
			// displayed with install_at already past, and the reminder is sent late rather than
			// skipped so the apps never close without their 5 minute warning
			name:              "a deadline reached with the first notice still displayed sends the reminder",
			untilDeadline:     -time.Minute,
			displayed:         true,
			installedVersions: behind,
			wantReminder:      true,
		},
		{
			name:              "an app whose install request fails does not stop the others being queued",
			untilDeadline:     -time.Minute,
			displayed:         true,
			reminder:          true,
			installedVersions: behind,
			firstInstallFails: true,
			wantActed:         true,
			wantInstalls:      []uint{twoInstallerID},
		},
		{
			name:              "an app with an install already pending is not queued again",
			untilDeadline:     -time.Minute,
			displayed:         true,
			reminder:          true,
			installedVersions: behind,
			pendingInstall:    true,
			wantActed:         true,
			wantInstalls:      []uint{twoInstallerID},
		},
		{
			// The policy queues its installs with the app open query attached, so a pending one of
			// those skips for the same reason the deadline exists.
			name:                        "an app whose pending install skips while the app is open is queued anyway",
			untilDeadline:               -time.Minute,
			displayed:                   true,
			reminder:                    true,
			installedVersions:           behind,
			pendingInstall:              true,
			pendingInstallSkipsWhenOpen: true,
			wantActed:                   true,
			wantInstalls:                []uint{oneInstallerID, twoInstallerID},
		},
		{
			// if an earlier pass set the status to acted and then stopped before queueing, the apps
			// are still unhandled. ActOnNotification returns false against that status, and
			// isStatusActed is what lets this pass carry on and queue them.
			name:              "an acted notification with an app still unhandled has its installs queued",
			untilDeadline:     -time.Minute,
			displayed:         true,
			reminder:          true,
			statusActed:       true,
			alreadyActed:      true,
			installedVersions: behind,
			wantActed:         true,
			wantInstalls:      []uint{oneInstallerID, twoInstallerID},
		},
		{
			name:              "an Update now that got there first stops the deadline installing the same apps again",
			untilDeadline:     -time.Minute,
			displayed:         true,
			reminder:          true,
			installedVersions: behind,
			alreadyActed:      true,
			wantActed:         true,
		},
		{
			name:                "a deadline reached on an offline host drops the deadline and sends the notification again",
			untilDeadline:       -time.Minute,
			displayed:           true,
			reminder:            true,
			installedVersions:   behind,
			hostOffline:         true,
			wantReminder:        true,
			wantDeadlineCleared: true,
		},
		{
			// an offline host and a reminder still on its way both leave displayed_at null, and
			// neither has been seen, so the pass waits for the reminder to reach the screen
			name:              "a notification past install_at with no displayed_at does nothing",
			untilDeadline:     -time.Minute,
			reminder:          true,
			installedVersions: behind,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ds := new(mock.Store)
			notificationSvc := &stubNotificationService{acts: !c.alreadyActed}
			kind := &patchNotificationKind{
				ds: ds, notificationSvc: notificationSvc, logger: slog.New(slog.DiscardHandler),
			}

			payload := patchNotificationFirstNoticePayload
			if c.reminder {
				payload = patchNotificationReminderPayload
			}
			var displayed *time.Time
			if c.displayed {
				displayed = &displayedAt
			}
			status := notifications_api.EndUserNotificationDispatched
			if c.statusActed {
				status = notifications_api.EndUserNotificationActed
			}
			var gotCutoff time.Time
			ds.ListPatchNotificationsDueFunc = func(_ context.Context, cutoff time.Time, limit int) ([]fleet.PatchNotificationDue, error) {
				gotCutoff = cutoff
				require.Equal(t, duePatchNotificationBatchSize, limit)
				return []fleet.PatchNotificationDue{{
					NotificationUUID: "notification-uuid",
					HostID:           hostID,
					Status:           status,
					Payload:          payload,
					DisplayedAt:      displayed,
					InstallAt:        time.Now().UTC().Add(c.untilDeadline),
					HostOnline:       !c.hostOffline,
				}}, nil
			}

			ds.ClearPatchNotificationInstallAtFunc = func(_ context.Context, _ string) error { return nil }

			// dropped software titles stop being returned, as deleting their rows would do
			dropped := make(map[uint]struct{})
			var appReads int
			ds.ListPatchNotificationAppsForNotificationsFunc = func(_ context.Context, uuids []string) (map[string][]fleet.PatchNotificationAppDetail, error) {
				appReads++
				all := []fleet.PatchNotificationAppDetail{
					{SoftwareTitleID: oneTitleID, SoftwareInstallerID: new(oneInstallerID), PolicyID: new(policyID), InstallerVersion: installerVersion, CreatedAt: appAddedAt},
					{SoftwareTitleID: twoTitleID, SoftwareInstallerID: new(twoInstallerID), PolicyID: new(policyID), InstallerVersion: installerVersion, CreatedAt: appAddedAt},
				}
				listed := make([]fleet.PatchNotificationAppDetail, 0, len(all))
				for _, app := range all {
					if _, gone := dropped[app.SoftwareTitleID]; !gone {
						listed = append(listed, app)
					}
				}
				return map[string][]fleet.PatchNotificationAppDetail{uuids[0]: listed}, nil
			}
			var gotDropped []uint
			ds.DeletePatchNotificationAppsFunc = func(_ context.Context, _ string, softwareTitleIDs []uint) error {
				for _, titleID := range softwareTitleIDs {
					dropped[titleID] = struct{}{}
				}
				gotDropped = append(gotDropped, softwareTitleIDs...)
				return nil
			}

			var versionReads int
			ds.ListSoftwareTitleVersionsForHostsFunc = func(_ context.Context, hostIDs []uint, titleIDs []uint) ([]fleet.HostSoftwareTitleVersion, error) {
				versionReads++
				versions := make([]fleet.HostSoftwareTitleVersion, 0, len(titleIDs))
				for _, titleID := range titleIDs {
					if installed, ok := c.installedVersions[titleID]; ok {
						versions = append(versions, fleet.HostSoftwareTitleVersion{
							HostID: hostID, SoftwareTitleID: titleID, Version: installed,
						})
					}
				}
				return versions, nil
			}
			var installDataReads int
			ds.ListLastTitleInstallDataForHostsFunc = func(_ context.Context, hostIDs []uint, _ []uint) (map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData, error) {
				installDataReads++
				firstAppInstalls := make([]*fleet.HostLastInstallData, 0, 2)
				if c.lastInstalled != nil {
					firstAppInstalls = append(firstAppInstalls, &fleet.HostLastInstallData{
						Status: new(fleet.SoftwareInstalled), UpdatedAt: *c.lastInstalled,
					})
				}
				if c.pendingInstall {
					firstAppInstalls = append(firstAppInstalls, &fleet.HostLastInstallData{
						Status:                  new(fleet.SoftwareInstallPending),
						OverridePreInstallQuery: c.pendingInstallSkipsWhenOpen,
					})
				}
				if len(firstAppInstalls) == 0 {
					return nil, nil
				}
				return map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData{
					{HostID: hostID, SoftwareTitleID: oneTitleID}: firstAppInstalls,
				}, nil
			}

			var installs []uint
			ds.InsertSoftwareInstallRequestFunc = func(_ context.Context, gotHostID uint, installerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
				require.Equal(t, hostID, gotHostID)
				require.False(t, opts.OverridePreInstallQuery, "the deadline forces the install, so the app being open must not stop it")
				require.NotNil(t, opts.PolicyID)
				require.Equal(t, policyID, *opts.PolicyID)
				if c.firstInstallFails && installerID == oneInstallerID {
					return "", errors.New("insert failed")
				}
				installs = append(installs, installerID)
				return "", nil
			}
			var markedQueued []uint
			ds.SetPatchNotificationAppsQueuedFunc = func(_ context.Context, _ string, softwareTitleIDs []uint) error {
				markedQueued = append(markedQueued, softwareTitleIDs...)
				return nil
			}

			passErr := kind.RemindAndInstallDuePatches(context.Background())
			if c.firstInstallFails {
				require.Error(t, passErr)
				require.NotContains(t, markedQueued, oneTitleID, "the app that failed stays unmarked so a later pass retries it")
			} else {
				require.NoError(t, passErr)
			}

			require.WithinDuration(t, time.Now().UTC().Add(patchNotificationReminderBefore), gotCutoff, time.Minute,
				"the cutoff is install_at plus the reminder lead time")

			// one read each per batch, not one per notification
			require.Equal(t, 1, appReads)
			require.Equal(t, 1, versionReads)
			require.Equal(t, 1, installDataReads)

			require.ElementsMatch(t, c.wantInstalls, installs)
			require.ElementsMatch(t, c.wantAppsDropped, gotDropped)
			require.Equal(t, c.wantActed, notificationSvc.actInvoked)
			require.Equal(t, c.wantDeadlineCleared, ds.ClearPatchNotificationInstallAtFuncInvoked)

			if !c.wantReminder {
				require.False(t, notificationSvc.delayInvoked)
				return
			}
			require.True(t, notificationSvc.delayInvoked)
			require.Nil(t, notificationSvc.delayPayload, "the deadline decides the notice when the toast is rendered")
		})
	}
}
