package service

import (
	"context"
	"encoding/json"
	"errors"
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

	// how many countdowns one cron pass reads
	patchNotificationCountdownBatchSize = 500
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
	notifications_api.SetNotificationStatusService
}

type patchNotificationKind struct {
	ds              fleet.Datastore
	activities      patchNotificationActivityWriter
	notificationSvc patchNotificationService
	logger          *slog.Logger
}

type PatchNotificationKind interface {
	notifications_api.NotificationKind
	RunPatchNotificationCountdowns(ctx context.Context) error
}

func NewPatchNotificationKind(
	ds fleet.Datastore,
	activities patchNotificationActivityWriter,
	notificationSvc patchNotificationService,
	logger *slog.Logger,
) PatchNotificationKind {
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

	if awaiting == nil {
		if err := svc.ds.NewPatchNotification(ctx, notificationUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "create patch notification record")
		}
	}

	// The app row goes last because it is what stops a later skip starting over: once it exists,
	// PatchNotificationExistsForApp returns true and this function returns early. Anything written
	// after it that failed would stay missing for the life of the notification. A notification left
	// with no apps is safe by comparison, since Render fails it and the policy opens a new one.
	if err := svc.ds.AddPatchNotificationApp(ctx, notificationUUID, fleet.PatchNotificationApp{
		PolicyID:            install.PolicyID,
		SoftwareTitleID:     *install.SoftwareTitleID,
		SoftwareInstallerID: install.SoftwareInstallerID,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "add patch notification app")
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
	// Fail the notification because it has no apps to display, so Fleet stops sending it.
	if len(apps) == 0 {
		err = k.notificationSvc.SetNotificationStatus(ctx, notification.UUID,
			notifications_api.EndUserNotificationFailed,
			new(notifications_api.EndUserNotificationReasonNothingToShow),
			[]string{notifications_api.EndUserNotificationPending, notifications_api.EndUserNotificationDispatched})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "fail patch notification with no apps")
		}
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

func (k *patchNotificationKind) OnDelay(ctx context.Context, notification *notifications_api.EndUserNotification) (*notifications_api.NotificationView, error) {
	// The countdown pass sends the reminder whatever the end user pressed, so the button only has to
	// close the toast, which the page does over the bridge. Leaving the row alone is also what stops
	// the button buying more time.
	return nil, nil
}

func (k *patchNotificationKind) RunPatchNotificationCountdowns(ctx context.Context) error {
	now := time.Now().UTC()

	countdowns, err := k.ds.ListPatchNotificationsDue(ctx, now.Add(patchNotificationReminderBefore), patchNotificationCountdownBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list patch notifications due")
	}

	// One host's countdown failing must not hold up the rest of the batch.
	var errs []error
	for _, countdown := range countdowns {
		if countdown.InstallAt.After(now) {
			if err := k.remindBeforePatching(ctx, countdown, now); err != nil {
				errs = append(errs, ctxerr.Wrapf(ctx, err, "remind before patching: notification_uuid=%s", countdown.NotificationUUID))
			}
			continue
		}
		if err := k.patchAtDeadline(ctx, countdown, now); err != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, err, "patch at deadline: notification_uuid=%s", countdown.NotificationUUID))
		}
	}
	return errors.Join(errs...)
}

func (k *patchNotificationKind) remindBeforePatching(ctx context.Context, countdown fleet.PatchNotificationDue, now time.Time) error {
	// These three checks are what keep the reminder to one send: the re-dispatch leaves the row
	// pending, and the display after that leaves the reminder flag in the payload.
	if countdown.Status != notifications_api.EndUserNotificationDispatched || countdown.DisplayedAt == nil {
		return nil
	}
	reminder, err := patchNotificationIsReminder(countdown.Payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read patch notification payload")
	}
	if reminder {
		return nil
	}

	remaining, err := k.dropAppsAlreadyUpdated(ctx, countdown)
	if err != nil {
		return err
	}

	// Nothing is left to warn about, and the notification would now render nothing, so close it.
	if remaining == 0 {
		if _, err := k.notificationSvc.ActOnNotification(ctx, countdown.NotificationUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "act on a patch notification with nothing left to update")
		}
		return nil
	}

	if err := k.notificationSvc.DelayNotification(ctx, countdown.NotificationUUID, now, patchNotificationReminderPayload); err != nil {
		return ctxerr.Wrap(ctx, err, "send patch notification reminder")
	}
	return nil
}

func (k *patchNotificationKind) patchAtDeadline(ctx context.Context, countdown fleet.PatchNotificationDue, now time.Time) error {
	// The end user never saw the notice this deadline belongs to, so the countdown starts over and
	// they get a fresh hour whenever their host next reaches the screen. A host that went offline and
	// a reminder still on its way both land here, and both want the same thing.
	if countdown.DisplayedAt == nil {
		if err := k.ds.ResetPatchNotification(ctx, countdown.NotificationUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "clear the deadline of an offline host's patch notification")
		}
		if err := k.notificationSvc.DelayNotification(ctx, countdown.NotificationUUID, now, patchNotificationFirstNoticePayload); err != nil {
			return ctxerr.Wrap(ctx, err, "restart an offline host's patch notification")
		}
		return nil
	}

	remaining, err := k.dropAppsAlreadyUpdated(ctx, countdown)
	if err != nil {
		return err
	}

	// Acted first, so an Update now arriving at the same moment cannot queue the same installs twice.
	acted, err := k.notificationSvc.ActOnNotification(ctx, countdown.NotificationUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "act on patch notification")
	}
	if !acted || remaining == 0 {
		return nil
	}

	// Stays acted on failure: there is no second press to finish the rest here, and the patch policy
	// opens a new notification for whatever is left unpatched.
	if _, err := k.queuePatchNotificationInstalls(ctx, countdown.NotificationUUID, countdown.HostID); err != nil {
		return ctxerr.Wrap(ctx, err, "queue patch notification installs at the deadline")
	}
	return nil
}

func (k *patchNotificationKind) dropAppsAlreadyUpdated(ctx context.Context, countdown fleet.PatchNotificationDue) (remaining int, err error) {
	apps, err := k.ds.ListPatchNotificationApps(ctx, countdown.NotificationUUID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "list patch notification apps")
	}
	if len(apps) == 0 {
		return 0, nil
	}

	// Inventory refreshes after any Fleet install and on refetch, so within the hour it is the freshest
	// record of what the host actually has.
	installed, err := k.ds.ListSoftwareByHostIDShort(ctx, countdown.HostID)
	if err != nil {
		return 0, ctxerr.Wrapf(ctx, err, "list software on host: host_id=%d", countdown.HostID)
	}
	installedVersionByBundleIdentifier := make(map[string]string, len(installed))
	for _, software := range installed {
		if software.BundleIdentifier != "" {
			installedVersionByBundleIdentifier[software.BundleIdentifier] = software.Version
		}
	}

	// An app whose installer is gone stays listed, because there is no version to compare it against.
	var dropped []uint
	for _, app := range apps {
		if app.SoftwareInstallerID == nil {
			continue
		}
		installer, err := k.ds.GetSoftwareInstallerMetadataByID(ctx, *app.SoftwareInstallerID)
		if err != nil {
			if fleet.IsNotFound(err) {
				continue
			}
			return 0, ctxerr.Wrapf(ctx, err, "get software installer metadata: software_installer_id=%d", *app.SoftwareInstallerID)
		}

		// FMA version strings are not reliably semver and CompareVersions treats an invalid one as lower,
		// which would force an install, so both sides are normalized first. An app with no inventory row
		// normalizes to an empty version and stays listed.
		installedVersion := toValidSemVer(installedVersionByBundleIdentifier[installer.BundleIdentifier])
		if installedVersion != "" && fleet.CompareVersions(toValidSemVer(installer.Version), installedVersion) != 1 {
			dropped = append(dropped, app.SoftwareTitleID)
			continue
		}

		// A My device self-service update lands before inventory catches up, so a Fleet install that
		// succeeded since the notification was created counts too.
		lastInstall, err := k.ds.GetHostLastInstallData(ctx, countdown.HostID, *app.SoftwareInstallerID)
		if err != nil {
			return 0, ctxerr.Wrapf(ctx, err, "get host last install data: host_id=%d, software_installer_id=%d",
				countdown.HostID, *app.SoftwareInstallerID)
		}
		if lastInstall != nil && lastInstall.Status != nil &&
			*lastInstall.Status == fleet.SoftwareInstalled && lastInstall.UpdatedAt.After(countdown.CreatedAt) {
			dropped = append(dropped, app.SoftwareTitleID)
		}
	}

	// Render builds the toast from this table, so the rows have to go for the reminder to stop naming
	// apps the end user already updated.
	if err := k.ds.DeletePatchNotificationApps(ctx, countdown.NotificationUUID, dropped); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "delete patch notification apps that no longer need updating")
	}
	return len(apps) - len(dropped), nil
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

	// Mark the notification acted first, so two presses at once can't both queue installs.
	acted, err := k.notificationSvc.ActOnNotification(ctx, notification.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "act on patch notification")
	}
	if !acted {
		return k.renderView(ctx, notification, true)
	}

	canPutBack, queueErr := k.queuePatchNotificationInstalls(ctx, notification.UUID, notification.HostID)
	if queueErr != nil {
		// Back to the status it had, so the next press finishes the rest.
		if canPutBack {
			setStatusErr := k.notificationSvc.SetNotificationStatus(ctx, notification.UUID, notification.Status, nil,
				[]string{notifications_api.EndUserNotificationActed})
			if setStatusErr != nil {
				k.logger.ErrorContext(ctx, "failed to put the patch notification back",
					"notification_uuid", notification.UUID, "err", setStatusErr)
			}
		}
		return nil, queueErr
	}

	return k.renderView(ctx, notification, true)
}

func (k *patchNotificationKind) queuePatchNotificationInstalls(ctx context.Context, notificationUUID string, hostID uint) (canPutBack bool, err error) {
	apps, err := k.ds.ListPatchNotificationApps(ctx, notificationUUID)
	if err != nil {
		return true, ctxerr.Wrap(ctx, err, "list patch notification apps")
	}

	// An attempt that fails part way still records what it queued, so the next one finishes the rest.
	var queuedTitleIDs []uint
	var queueErr error
	for _, app := range apps {
		if app.SoftwareInstallerID == nil {
			k.logger.InfoContext(ctx, "skipping patch notification install for software with no installer",
				"notification_uuid", notificationUUID, "software_title_id", app.SoftwareTitleID)
			continue
		}
		// queued by an earlier attempt that failed before it finished
		if app.InstallQueued {
			continue
		}

		// check if this install is already queued
		lastInstall, err := k.ds.GetHostLastInstallData(ctx, hostID, *app.SoftwareInstallerID)
		if err != nil {
			queueErr = ctxerr.Wrapf(ctx, err, "get host last install data: host_id=%d, software_installer_id=%d",
				hostID, *app.SoftwareInstallerID)
			break
		}
		if lastInstall != nil && lastInstall.Status != nil && *lastInstall.Status == fleet.SoftwareInstallPending {
			continue
		}

		// OverridePreInstallQuery stays false, which leaves the agent no app open gate to run, so the
		// app closes and updates.
		if _, err := k.ds.InsertSoftwareInstallRequest(ctx, hostID, *app.SoftwareInstallerID,
			fleet.HostSoftwareInstallOptions{PolicyID: app.PolicyID},
		); err != nil {
			queueErr = ctxerr.Wrapf(ctx, err, "insert software install request: host_id=%d, software_installer_id=%d",
				hostID, *app.SoftwareInstallerID)
			break
		}

		queuedTitleIDs = append(queuedTitleIDs, app.SoftwareTitleID)
	}

	// Once this write fails the installs have gone out unrecorded, so a second attempt would repeat them.
	if err := k.ds.SetPatchNotificationAppsQueued(ctx, notificationUUID, queuedTitleIDs); err != nil {
		return false, ctxerr.Wrap(ctx, err, "set patch notification apps queued")
	}
	return true, queueErr
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

	reminder, err := patchNotificationIsReminder(notification.Payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read patch notification payload")
	}
	timeBefore := patchNotificationFirstNoticeBefore
	if reminder {
		timeBefore = patchNotificationReminderBefore
	}

	status := "failed"
	var installAt *time.Time
	if outcome.Displayed {
		status = "success"

		// The deadline is counted from this notice reaching the screen, so the end user gets the whole
		// lead time the toast promises however long the toast took to get there. notification.DisplayedAt
		// is still nil here, since RecordOutcome loaded that copy before the outcome write.
		deadline, err := k.ds.SetPatchNotificationInstallAt(ctx, notification.UUID, time.Now().UTC().Add(timeBefore))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "set patch notification install at")
		}
		installAt = &deadline
	}

	if err := k.activities.NewActivity(ctx, nil, fleet.ActivityTypeNotifiedEndUserBeforePatching{
		HostID:                notification.HostID,
		HostDisplayName:       host.DisplayName(),
		PatchNotificationUUID: notification.UUID,
		SoftwareTitles:        softwareTitles,
		PolicyIDs:             policyIDs,
		TimeBefore:            int(timeBefore.Seconds()),
		InstallAt:             installAt,
		Status:                status,
		ScriptExecutionID:     outcome.ExecutionID,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for patch notification")
	}
	return nil
}
