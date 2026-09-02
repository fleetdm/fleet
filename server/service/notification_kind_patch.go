package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
)

const (
	patchNotificationActionUpdateNow = "update_now"
	patchNotificationActionRemind    = "remind"
	patchNotificationActionDismiss   = "dismiss"
)

const (
	patchNotificationFirstNoticeBefore = time.Hour
	patchNotificationReminderBefore    = 5 * time.Minute
)

// Which of the two notices the notification is for. In the payload because that
// is what Render is handed, and what DelayNotification replaces on a resend.
var (
	patchNotificationFirstNoticePayload = json.RawMessage(`{"reminder":false}`)
	patchNotificationReminderPayload    = json.RawMessage(`{"reminder":true}`)
)

func patchNotificationIsReminder(payload json.RawMessage) (bool, error) {
	var decoded struct {
		Reminder bool `json:"reminder"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return false, err
	}
	return decoded.Reminder, nil
}

type patchNotificationActivityWriter interface {
	NewActivity(ctx context.Context, user *fleet.User, activity activity_api.ActivityDetails) error
}

// The patch kind doesn't own the notifications table, so delaying and acting on
// a notification go through the notifications context's service.
type patchNotificationService interface {
	notifications_api.DelayNotificationService
	notifications_api.ActOnNotificationService
}

type patchNotificationKind struct {
	ds              fleet.Datastore
	activities      patchNotificationActivityWriter
	notificationSvc patchNotificationService
	logger          *slog.Logger
}

func NewPatchNotificationKind(
	ds fleet.Datastore,
	activities patchNotificationActivityWriter,
	notificationSvc patchNotificationService,
	logger *slog.Logger,
) notifications_api.NotificationKind {
	return &patchNotificationKind{
		ds:              ds,
		activities:      activities,
		notificationSvc: notificationSvc,
		logger:          logger,
	}
}

func (k *patchNotificationKind) Name() string {
	return fleet.PatchNotificationKind
}

func (svc *Service) createPatchNotificationForEndUser(ctx context.Context, host *fleet.Host, install *fleet.HostSoftwareInstallerResult) error {
	if install.SoftwareTitleID == nil {
		svc.logger.InfoContext(ctx, "not notifying about a skipped patch for software with no title",
			"host_id", host.ID, "install_uuid", install.InstallUUID)
		return nil
	}

	exists, err := svc.ds.PatchNotificationExistsForApp(ctx, host.ID, *install.SoftwareTitleID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "check whether a patch notification exists for this app")
	}
	if exists {
		return nil
	}

	awaiting, err := svc.notificationsSvc.NotificationAwaitingDisplay(ctx, host.ID, fleet.PatchNotificationKind)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get patch notification awaiting first dispatch for host")
	}

	var notificationUUID string
	if awaiting != nil {
		notificationUUID = awaiting.UUID
	} else {
		expiresAt := time.Now().UTC().Add(notifications_api.EndUserNotificationMaxLifetime)

		created, err := svc.notificationsSvc.CreateNotification(ctx, &notifications_api.EndUserNotification{
			HostID:    host.ID,
			Kind:      fleet.PatchNotificationKind,
			Payload:   patchNotificationFirstNoticePayload,
			ExpiresAt: &expiresAt,
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create patch notification")
		}
		notificationUUID = created.UUID
	}

	// the app row goes first, because a notification listing no apps can't render
	if err := svc.ds.AddPatchNotificationApp(ctx, notificationUUID, fleet.PatchNotificationApp{
		PolicyID:            install.PolicyID,
		SoftwareTitleID:     *install.SoftwareTitleID,
		SoftwareInstallerID: install.SoftwareInstallerID,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "add patch notification app")
	}

	if awaiting == nil {
		if err := svc.ds.NewPatchNotification(ctx, notificationUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "create patch notification record")
		}
	}
	return nil
}

func (k *patchNotificationKind) Render(ctx context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.NotificationView, error) {
	return k.renderView(ctx, notification, notification.Status == notifications_api.EndUserNotificationActed)
}

// installsQueued is passed in rather than read, so an action that just queued
// the installs can build the view it returns without reading its own write back.
func (k *patchNotificationKind) renderView(ctx context.Context, notification *notifications_api.EndUserNotification, installsQueued bool) (*notifications_api.NotificationView, error) {
	apps, err := k.ds.ListPatchNotificationApps(ctx, notification.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list patch notification apps")
	}
	if len(apps) == 0 {
		return nil, ctxerr.Errorf(ctx, "patch notification %s lists no apps", notification.UUID)
	}

	appConfig, err := k.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config for patch notification")
	}

	// The request already authenticated with a token this host updated in the
	// last hour, and that is the same row this reads.
	deviceToken, err := k.ds.GetDeviceAuthTokenIfFresh(ctx, notification.HostID, hostDeviceAuthTokenTTL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get device auth token for patch notification")
	}

	reminder, err := patchNotificationIsReminder(notification.Payload)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read patch notification payload")
	}

	items := make([]notifications_api.NotificationItem, 0, len(apps))
	for _, app := range apps {
		item := notifications_api.NotificationItem{
			SoftwareTitleID: app.SoftwareTitleID,
			Name:            app.Name,
			DisplayName:     app.DisplayName,
		}
		if app.HasIcon {
			icon := fleet.SoftwareTitleIcon{SoftwareTitleID: app.SoftwareTitleID}
			iconURL := icon.IconUrlWithDeviceToken(deviceToken)
			item.IconURL = &iconURL
		}
		if installsQueued {
			item.Status = "Installing..."
		}
		items = append(items, item)
	}

	description := "These apps will close and update in **1 hour**."
	switch {
	case reminder && len(items) == 1:
		description = "This app will close and update in **5 minutes**."
	case reminder:
		description = "These apps will close and update in **5 minutes**."
	case len(items) == 1:
		description = "This app will close and update in **1 hour**."
	}

	hide := notifications_api.NotificationAction{ID: patchNotificationActionDismiss, Label: "Hide"}
	actions := []notifications_api.NotificationAction{hide}
	if !installsQueued {
		secondary := notifications_api.NotificationAction{ID: patchNotificationActionRemind, Label: "Remind me 5 minutes before"}
		if reminder {
			secondary = hide
		}
		actions = []notifications_api.NotificationAction{
			secondary,
			{ID: patchNotificationActionUpdateNow, Label: "Update now"},
		}
	}

	serverURL := appConfig.ServerSettings.ServerURL

	return &notifications_api.NotificationView{
		UUID:                notification.UUID,
		OrgLogoURLLightMode: fleet.AbsolutizeLogoURL(appConfig.OrgInfo.OrgLogoURLLightMode, serverURL),
		OrgLogoURLDarkMode:  fleet.AbsolutizeLogoURL(appConfig.OrgInfo.OrgLogoURLDarkMode, serverURL),
		Title:               "Save your work",
		Description:         description,
		Items:               items,
		Actions:             actions,
	}, nil
}

func (k *patchNotificationKind) OnVerify(ctx context.Context, notification *notifications_api.EndUserNotification, displayedAt time.Time) error {
	return nil
}

// TODO: should there be a grace period where we install no matter what?
func (k *patchNotificationKind) OnDelay(ctx context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.NotificationView, error) {
	untilReminder := patchNotificationFirstNoticeBefore - patchNotificationReminderBefore

	now := time.Now().UTC()
	nextAttemptAt := now.Add(notifications_api.EndUserNotificationDelayInterval)
	payload := patchNotificationFirstNoticePayload
	if notification.DisplayedAt != nil {
		if fromDisplay := notification.DisplayedAt.Add(untilReminder); fromDisplay.After(now) {
			nextAttemptAt = fromDisplay
			payload = patchNotificationReminderPayload
		}
	}

	if err := k.notificationSvc.DelayNotification(ctx, notification.UUID, nextAttemptAt, payload); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "delay patch notification")
	}

	// The delay wrote the payload the view's copy comes from, so carry it over
	// instead of reading the row again.
	notification.Payload = payload
	return k.renderView(ctx, notification, false)
}

func (k *patchNotificationKind) OnAction(ctx context.Context, notification *notifications_api.EndUserNotification, actionID string) (*notifications_api.NotificationView, error) {
	switch actionID {
	case patchNotificationActionUpdateNow:
		return k.updateNow(ctx, notification)

	case patchNotificationActionRemind:
		return k.OnDelay(ctx, notification)

	case patchNotificationActionDismiss:
		return nil, nil

	default:
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("action",
			fmt.Sprintf("%q is not something that can be done to a patch notification", actionID)))
	}
}

func (k *patchNotificationKind) updateNow(ctx context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.NotificationView, error) {
	if notification.Status != notifications_api.EndUserNotificationPending &&
		notification.Status != notifications_api.EndUserNotificationDispatched {
		return nil, nil
	}

	apps, err := k.ds.ListPatchNotificationApps(ctx, notification.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list patch notification apps")
	}

	// Acted is set only once every app is queued, so a press that fails part way
	// leaves the notification live for the next one to finish.
	for _, app := range apps {
		if app.SoftwareInstallerID == nil {
			k.logger.InfoContext(ctx, "skipping patch notification install for software with no installer",
				"notification_uuid", notification.UUID, "software_title_id", app.SoftwareTitleID)
			continue
		}

		// check if this install is already queued
		lastInstall, err := k.ds.GetHostLastInstallData(ctx, notification.HostID, *app.SoftwareInstallerID)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "get host last install data: host_id=%d, software_installer_id=%d",
				notification.HostID, *app.SoftwareInstallerID)
		}
		if lastInstall != nil && lastInstall.Status != nil && *lastInstall.Status == fleet.SoftwareInstallPending {
			continue
		}

		if _, err := k.ds.InsertSoftwareInstallRequest(ctx, notification.HostID, *app.SoftwareInstallerID,
			fleet.HostSoftwareInstallOptions{
				PolicyID:           app.PolicyID,
				IgnoreAppOpenQuery: true,
			},
		); err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "insert software install request: host_id=%d, software_installer_id=%d",
				notification.HostID, *app.SoftwareInstallerID)
		}
	}

	if _, err := k.notificationSvc.ActOnNotification(ctx, notification.UUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "act on patch notification")
	}

	return k.renderView(ctx, notification, true)
}

func (k *patchNotificationKind) OnOutcome(ctx context.Context, notification *notifications_api.EndUserNotification, outcome notifications_api.NotificationOutcome) error {
	// A failing notification is retried every minute until it expires. Only the
	// first of a run of identical failures gets an activity, so a host with Fleet
	// Desktop closed does not fill the feed with the same failure.
	if !outcome.Displayed && notification.LastExitCode != nil && *notification.LastExitCode == outcome.ExitCode {
		return nil
	}

	apps, err := k.ds.ListPatchNotificationApps(ctx, notification.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list patch notification apps")
	}

	host, err := k.ds.HostLiteByID(ctx, notification.HostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host for patch notification activity")
	}

	// A policy that was deleted leaves patch_notification_apps.policy_id null.
	softwareTitles := make([]string, 0, len(apps))
	policyIDs := make([]uint, 0, len(apps))
	for _, app := range apps {
		softwareTitles = append(softwareTitles, app.Name)
		if app.PolicyID != nil {
			policyIDs = append(policyIDs, *app.PolicyID)
		}
	}

	status := "failed"
	if outcome.Displayed {
		status = "success"
	}

	reminder, err := patchNotificationIsReminder(notification.Payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read patch notification payload")
	}
	timeBefore := patchNotificationFirstNoticeBefore
	if reminder {
		timeBefore = patchNotificationReminderBefore
	}

	if err := k.activities.NewActivity(ctx, nil, fleet.ActivityTypeNotifiedEndUserBeforePatching{
		HostID:                notification.HostID,
		HostDisplayName:       host.DisplayName(),
		PatchNotificationUUID: notification.UUID,
		SoftwareTitles:        softwareTitles,
		PolicyIDs:             policyIDs,
		TimeBefore:            int(timeBefore.Seconds()),
		Status:                status,
		ScriptExecutionID:     outcome.ExecutionID,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for patch notification")
	}
	return nil
}
