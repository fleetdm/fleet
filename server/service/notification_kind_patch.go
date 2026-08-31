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
	patchNotificationLeadTimeSeconds     = 3600
	patchNotificationReminderTimeSeconds = 300
)

type patchNotificationPayload struct {
	Reminder bool `json:"reminder"`
}

type patchNotificationActivityWriter interface {
	NewActivity(ctx context.Context, user *fleet.User, activity activity_api.ActivityDetails) error
}

type patchNotificationKind struct {
	ds              fleet.Datastore
	activities      patchNotificationActivityWriter
	notificationSvc notifications_api.DelayNotificationService
	logger          *slog.Logger
}

func NewPatchNotificationKind(
	ds fleet.Datastore,
	activities patchNotificationActivityWriter,
	notificationSvc notifications_api.DelayNotificationService,
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

func (svc *Service) notifyEndUserBeforePatching(ctx context.Context, host *fleet.Host, install *fleet.HostSoftwareInstallerResult) error {
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

	notificationUUID, err := svc.ds.PatchNotificationAwaitingDispatchForHost(ctx, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get unsent patch notification for host")
	}

	if notificationUUID == "" {
		payload, err := json.Marshal(patchNotificationPayload{Reminder: false})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build patch notification payload")
		}
		expiresAt := time.Now().UTC().Add(notifications_api.EndUserNotificationMaxLifetime)

		created, err := svc.notificationsSvc.CreateNotification(ctx, &notifications_api.EndUserNotification{
			HostID:    host.ID,
			Kind:      fleet.PatchNotificationKind,
			Payload:   payload,
			ExpiresAt: &expiresAt,
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "create patch notification")
		}
		notificationUUID = created.UUID

		if err := svc.ds.NewPatchNotification(ctx, notificationUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "create patch notification record")
		}
	}

	return svc.ds.AddPatchNotificationApp(ctx, notificationUUID, fleet.PatchNotificationApp{
		PolicyID:            install.PolicyID,
		SoftwareTitleID:     *install.SoftwareTitleID,
		SoftwareInstallerID: install.SoftwareInstallerID,
	})
}

func (k *patchNotificationKind) Render(ctx context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.NotificationView, error) {
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

	deviceToken, err := k.ds.GetDeviceAuthTokenIfFresh(ctx, notification.HostID, hostDeviceAuthTokenTTL)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "get device auth token for patch notification")
	}

	installsQueued, err := k.ds.PatchNotificationInstallsQueued(ctx, notification.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "check patch notification installs queued")
	}

	reminder := k.payload(ctx, notification).Reminder

	items := make([]notifications_api.NotificationItem, 0, len(apps))
	for _, app := range apps {
		item := notifications_api.NotificationItem{
			SoftwareTitleID: app.SoftwareTitleID,
			Name:            app.Name,
			DisplayName:     app.DisplayName,
		}
		if app.HasIcon && deviceToken != "" {
			icon := fleet.SoftwareTitleIcon{SoftwareTitleID: app.SoftwareTitleID}
			iconURL := icon.IconUrlWithDeviceToken(deviceToken)
			item.IconURL = &iconURL
		}
		if installsQueued {
			item.Status = "Installing..."
		}
		items = append(items, item)
	}

	timeLeft := "1 hour"
	if reminder {
		timeLeft = "5 minutes"
	}
	description := fmt.Sprintf("These apps will close and update in **%s**.", timeLeft)
	if len(items) == 1 {
		description = fmt.Sprintf("This app will close and update in **%s**.", timeLeft)
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

	lightLogoURL := appConfig.OrgInfo.OrgLogoURLLightMode
	if lightLogoURL == "" {
		lightLogoURL = appConfig.OrgInfo.OrgLogoURLLightBackground //nolint:staticcheck // all a config saved before dark mode has
	}
	darkLogoURL := appConfig.OrgInfo.OrgLogoURLDarkMode
	if darkLogoURL == "" {
		darkLogoURL = appConfig.OrgInfo.OrgLogoURL //nolint:staticcheck // all a config saved before dark mode has
	}
	serverURL := appConfig.ServerSettings.ServerURL

	return &notifications_api.NotificationView{
		UUID:                notification.UUID,
		OrgLogoURLLightMode: fleet.AbsolutizeLogoURL(lightLogoURL, serverURL),
		OrgLogoURLDarkMode:  fleet.AbsolutizeLogoURL(darkLogoURL, serverURL),
		Title:               "Save your work 💾",
		Description:         description,
		Items:               items,
		Actions:             actions,
	}, nil
}

func (k *patchNotificationKind) OnVerify(ctx context.Context, notification *notifications_api.EndUserNotification, displayedAt time.Time) error {
	return nil
}

// TODO: should there be a grace period where we install no matter what?
func (k *patchNotificationKind) OnDelay(ctx context.Context, notification *notifications_api.EndUserNotification) error {
	untilReminder := time.Duration(patchNotificationLeadTimeSeconds-patchNotificationReminderTimeSeconds) * time.Second

	now := time.Now().UTC()
	nextAttemptAt := now.Add(notifications_api.EndUserNotificationDelayInterval)
	payload := patchNotificationPayload{Reminder: false}
	if notification.DisplayedAt != nil {
		if fromDisplay := notification.DisplayedAt.Add(untilReminder); fromDisplay.After(now) {
			nextAttemptAt = fromDisplay
			payload.Reminder = true
		}
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build patch notification payload")
	}
	return k.notificationSvc.DelayNotification(ctx, notification.UUID, nextAttemptAt, encoded)
}

func (k *patchNotificationKind) OnAction(ctx context.Context, notification *notifications_api.EndUserNotification, actionID string) error {
	switch actionID {
	case patchNotificationActionUpdateNow:
		return k.updateNow(ctx, notification)

	case patchNotificationActionRemind:
		return k.OnDelay(ctx, notification)

	case patchNotificationActionDismiss:
		return nil

	default:
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("action",
			fmt.Sprintf("%q is not something that can be done to a patch notification", actionID)))
	}
}

func (k *patchNotificationKind) updateNow(ctx context.Context, notification *notifications_api.EndUserNotification) error {
	if notification.Status != notifications_api.EndUserNotificationPending &&
		notification.Status != notifications_api.EndUserNotificationDispatched {
		return nil
	}

	installsQueued, err := k.ds.PatchNotificationInstallsQueued(ctx, notification.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "check patch notification installs queued")
	}
	if installsQueued {
		return nil
	}

	apps, err := k.ds.ListPatchNotificationApps(ctx, notification.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list patch notification apps")
	}

	for _, app := range apps {
		if app.SoftwareInstallerID == nil {
			k.logger.InfoContext(ctx, "skipping patch notification install for software with no installer",
				"notification_uuid", notification.UUID, "software_title_id", app.SoftwareTitleID)
			continue
		}
		if _, err := k.ds.InsertSoftwareInstallRequest(ctx, notification.HostID, *app.SoftwareInstallerID,
			fleet.HostSoftwareInstallOptions{
				PolicyID:           app.PolicyID,
				IgnoreAppOpenQuery: true,
			},
		); err != nil {
			return ctxerr.Wrapf(ctx, err, "insert software install request: host_id=%d, software_installer_id=%d",
				notification.HostID, *app.SoftwareInstallerID)
		}
	}

	return k.ds.SetPatchNotificationInstallsQueued(ctx, notification.UUID)
}

func (k *patchNotificationKind) OnOutcome(ctx context.Context, notification *notifications_api.EndUserNotification, outcome notifications_api.NotificationOutcome) error {
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

	softwareTitles := make([]string, 0, len(apps))
	policyIDs := make([]uint, 0, len(apps))
	seenPolicyIDs := make(map[uint]struct{}, len(apps))
	for _, app := range apps {
		softwareTitles = append(softwareTitles, app.Name)
		if app.PolicyID == nil {
			continue
		}
		if _, seen := seenPolicyIDs[*app.PolicyID]; seen {
			continue
		}
		seenPolicyIDs[*app.PolicyID] = struct{}{}
		policyIDs = append(policyIDs, *app.PolicyID)
	}

	status := "failed"
	if outcome.Displayed {
		status = "success"
	}

	timeBefore := patchNotificationLeadTimeSeconds
	if k.payload(ctx, notification).Reminder {
		timeBefore = patchNotificationReminderTimeSeconds
	}

	if err := k.activities.NewActivity(ctx, nil, fleet.ActivityTypeNotifiedEndUserBeforePatching{
		HostID:                notification.HostID,
		HostDisplayName:       host.DisplayName(),
		PatchNotificationUUID: notification.UUID,
		SoftwareTitles:        softwareTitles,
		PolicyIDs:             policyIDs,
		TimeBefore:            timeBefore,
		Status:                status,
		ScriptExecutionID:     outcome.ExecutionID,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for patch notification")
	}
	return nil
}

func (k *patchNotificationKind) payload(ctx context.Context, notification *notifications_api.EndUserNotification) patchNotificationPayload {
	var payload patchNotificationPayload
	if len(notification.Payload) == 0 {
		return payload
	}
	if err := json.Unmarshal(notification.Payload, &payload); err != nil {
		k.logger.ErrorContext(ctx, "failed to read a patch notification's payload",
			"notification_uuid", notification.UUID, "err", err)
	}
	return payload
}
