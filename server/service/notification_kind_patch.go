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

// Action IDs the kind declares in Render and handles in OnAction. Core owns
// verify and delay; these are the kind's own.
const (
	patchNotificationActionUpdateNow = "update_now"
	patchNotificationActionRemind    = "remind"
	patchNotificationActionDismiss   = "dismiss"
)

// How long before the install each of the two notifications goes out, in
// seconds. The reminder is sent by the countdown sub-issue; this kind only
// reports which one an activity is about.
const (
	patchNotificationLeadTimeSeconds     = 3600
	patchNotificationReminderTimeSeconds = 300
)

// patchNotificationPayload is the kind's own state on the notification row.
// Everything else it draws comes from the patch_notifications tables, so an
// app or an icon changing shows up on the next fetch.
type patchNotificationPayload struct {
	Reminder bool `json:"reminder"`
}

// Writing an activity goes through the Fleet service rather than the datastore
// so the webhook and audit paths run too.
type patchNotificationActivityWriter interface {
	NewActivity(ctx context.Context, user *fleet.User, activity activity_api.ActivityDetails) error
}

type patchNotificationKind struct {
	ds              fleet.Datastore
	activities      patchNotificationActivityWriter
	notificationSvc notifications_api.KindNotificationService
	logger          *slog.Logger
}

func NewPatchNotificationKind(
	ds fleet.Datastore,
	activities patchNotificationActivityWriter,
	notificationSvc notifications_api.KindNotificationService,
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

// notifyEndUserBeforePatching adds an app whose install just skipped because it
// was open to a notification for its host.
//
// Fleetd reports each install result on its own, so there is no batch to work
// from. Joining a notification that hasn't gone out yet is what turns ten apps
// updating at once into one toast listing ten; a skip arriving after the toast
// is on screen starts a new one with its own hour.
//
// The covered check is not an optimisation. notify_before_patching forces
// continuous automations on, so the policy fires again on every host refetch,
// and without it each refetch would start another countdown for the same app.
func (svc *Service) notifyEndUserBeforePatching(ctx context.Context, host *fleet.Host, install *fleet.HostSoftwareInstallerResult) error {
	if install.SoftwareTitleID == nil {
		svc.logger.InfoContext(ctx, "not notifying about a skipped patch for software with no title",
			"host_id", host.ID, "install_uuid", install.InstallUUID)
		return nil
	}

	covered, err := svc.ds.PatchNotificationCoversApp(ctx, host.ID, *install.SoftwareTitleID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "check whether a patch notification covers this app")
	}
	if covered {
		return nil
	}

	notificationUUID, err := svc.ds.UnsentPatchNotificationForHost(ctx, host.ID)
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
	// The apps are written right after the notification, so a patch notification
	// with none of them means that write didn't land. Fail rather than put an
	// empty "Save your work" on the end user's screen.
	if len(apps) == 0 {
		return nil, ctxerr.Errorf(ctx, "patch notification %s lists no apps", notification.UUID)
	}

	appConfig, err := k.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config for patch notification")
	}

	// The icon endpoint is device authenticated, so its URL carries the host's
	// own token. Without a token the page falls back to matching on the name.
	deviceToken, err := k.ds.GetDeviceAuthTokenIfFresh(ctx, notification.HostID, hostDeviceAuthTokenTTL)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "get device auth token for patch notification")
	}

	reminder := k.payload(ctx, notification).Reminder
	acted := notification.Status == notifications_api.EndUserNotificationActed

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
		if acted {
			item.Status = "Installing..."
		}
		items = append(items, item)
	}

	// Copy is verbatim from the design; the page renders the bold markup.
	timeLeft := "1 hour"
	if reminder {
		timeLeft = "5 minutes"
	}
	description := fmt.Sprintf("These apps will close and update in **%s**.", timeLeft)
	if len(items) == 1 {
		description = fmt.Sprintf("This app will close and update in **%s**.", timeLeft)
	}

	// The last action is the primary one, and dismiss is what closes the window.
	// Once the installs are queued there is nothing left to ask for.
	hide := notifications_api.NotificationAction{ID: patchNotificationActionDismiss, Label: "Hide"}
	actions := []notifications_api.NotificationAction{hide}
	if !acted {
		secondary := notifications_api.NotificationAction{ID: patchNotificationActionRemind, Label: "Remind me 5 minutes before"}
		if reminder {
			secondary = hide
		}
		actions = []notifications_api.NotificationAction{
			secondary,
			{ID: patchNotificationActionUpdateNow, Label: "Update now"},
		}
	}

	// An org that set its logo before dark mode has only the deprecated field
	// until it saves its config again. A Fleet-hosted logo is stored relative, so
	// absolutize it the way the Fleet Desktop config endpoint does.
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

// A notification that was displayed rejoins its own schedule rather than
// starting a fresh wait, so the attempt it comes back for is the five minute
// reminder and it has to say so. One that was never displayed, or whose
// reminder mark has already passed, waits a full interval and comes back as the
// first notification again.
//
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
		// The page closes its own window. Nothing changes server side: the
		// countdown keeps running and the apps still update at the deadline.
		return nil

	default:
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("action",
			fmt.Sprintf("%q is not something that can be done to a patch notification", actionID)))
	}
}

// updateNow queues an install for every app the notification covers, then
// finishes the notification so neither the reminder nor the deadline install
// fires for it again.
//
// The installs bypass the app-open check: the end user asked for the update,
// and skipping because the app is open would leave the toast looking broken.
func (k *patchNotificationKind) updateNow(ctx context.Context, notification *notifications_api.EndUserNotification) error {
	// Only a notification that is still live has anything left to queue. One the
	// end user already acted on queued its installs the first time around, and
	// one that failed or ran out of time is over.
	if notification.Status != notifications_api.EndUserNotificationPending &&
		notification.Status != notifications_api.EndUserNotificationDispatched {
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
				SelfService:      false,
				PolicyID:         app.PolicyID,
				SkipAppOpenCheck: true,
			},
		); err != nil {
			return ctxerr.Wrapf(ctx, err, "insert software install request: host_id=%d, software_installer_id=%d",
				notification.HostID, *app.SoftwareInstallerID)
		}
	}

	return k.notificationSvc.MarkNotificationActed(ctx, notification.UUID)
}

func (k *patchNotificationKind) OnOutcome(ctx context.Context, notification *notifications_api.EndUserNotification, outcome notifications_api.NotificationOutcome) error {
	// A retryable failure comes back every minute until it clears or the
	// notification runs out of time, so a failure that ended the same way as the
	// attempt before it isn't reported again. A success always is: a notification
	// shown again after a delay is something new the end user saw.
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

	// Author is nil: Fleet decided to notify, not a user.
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

// A payload Fleet can't read falls back to the first notification rather than
// the reminder, which is the safer of the two to repeat.
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
