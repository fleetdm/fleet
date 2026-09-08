package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
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
	if len(countdowns) == 0 {
		return nil
	}

	notificationUUIDs := make([]string, 0, len(countdowns))
	hostIDs := make([]uint, 0, len(countdowns))
	for _, countdown := range countdowns {
		notificationUUIDs = append(notificationUUIDs, countdown.NotificationUUID)
		hostIDs = append(hostIDs, countdown.HostID)
	}

	// The apps, the inventory and the install history of every countdown in the batch, read up front so
	// the loop below does writes and installs rather than a handful of reads per countdown.
	appsByNotification, err := k.ds.ListPatchNotificationAppsForNotifications(ctx, notificationUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list patch notification apps for the batch")
	}

	var softwareTitleIDs []uint
	for _, apps := range appsByNotification {
		for _, app := range apps {
			softwareTitleIDs = append(softwareTitleIDs, app.SoftwareTitleID)
		}
	}
	// The same title comes up once per host notified about it, and sorting before compacting keeps the
	// IN list short and in the same order every pass.
	slices.Sort(softwareTitleIDs)
	softwareTitleIDs = slices.Compact(softwareTitleIDs)

	versions, err := k.ds.ListHostSoftwareVersionsForTitles(ctx, hostIDs, softwareTitleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host software versions for the batch")
	}
	installedVersions := make(map[fleet.HostSoftwareTitleKey][]string, len(versions))
	for _, version := range versions {
		key := fleet.HostSoftwareTitleKey{HostID: version.HostID, SoftwareTitleID: version.SoftwareTitleID}
		installedVersions[key] = append(installedVersions[key], version.Version)
	}

	installsByTitle, err := k.ds.ListHostLastTitleInstallData(ctx, hostIDs, softwareTitleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host last title install data for the batch")
	}

	// One host's countdown failing must not hold up the rest of the batch.
	var errs []error
	for _, countdown := range countdowns {
		// The batch reaches no further ahead than the reminder's lead time, so a deadline still to come
		// means this pass is the reminder rather than the patch.
		var sendReminder bool
		if countdown.InstallAt.After(now) {
			sendReminder = true
		}

		if sendReminder {
			// These three checks are what keep the reminder to one send: the re-dispatch leaves the row
			// pending, and the display after that leaves the reminder flag in the payload.
			if countdown.Status != notifications_api.EndUserNotificationDispatched || countdown.DisplayedAt == nil {
				continue
			}
			var alreadyReminder bool
			alreadyReminder, err = patchNotificationIsReminder(countdown.Payload)
			if err != nil {
				errs = append(errs, ctxerr.Wrapf(ctx, err, "read patch notification payload: notification_uuid=%s", countdown.NotificationUUID))
				continue
			}
			if alreadyReminder {
				continue
			}
		} else if countdown.DisplayedAt == nil {
			// The end user never saw the notice this deadline belongs to, so the countdown starts over and
			// they get a fresh hour whenever their host next reaches the screen. A host that went offline and
			// a reminder still on its way both land here, and both want the same thing.
			err = k.ds.ResetPatchNotification(ctx, countdown.NotificationUUID)
			if err != nil {
				errs = append(errs, ctxerr.Wrapf(ctx, err,
					"clear the deadline of a patch notification the end user has not seen: notification_uuid=%s", countdown.NotificationUUID))
				continue
			}
			err = k.notificationSvc.DelayNotification(ctx, countdown.NotificationUUID, now, patchNotificationFirstNoticePayload)
			if err != nil {
				errs = append(errs, ctxerr.Wrapf(ctx, err,
					"restart a patch notification the end user has not seen: notification_uuid=%s", countdown.NotificationUUID))
			}
			continue
		}

		// An app whose installer is gone stays listed, because there is nothing left to install with.
		apps := appsByNotification[countdown.NotificationUUID]
		var alreadyUpdatedTitleIDs []uint
		remaining := make([]fleet.PatchNotificationAppDetail, 0, len(apps))
		for _, app := range apps {
			if app.SoftwareInstallerID == nil {
				remaining = append(remaining, app)
				continue
			}

			appKey := fleet.HostSoftwareTitleKey{HostID: countdown.HostID, SoftwareTitleID: app.SoftwareTitleID}

			// Version strings of Fleet-maintained apps cannot be ordered, and the version Fleet installs
			// is whichever installer is active rather than the highest string, so the only thing inventory
			// can say is whether the host already carries exactly what this installer would put down. A
			// host can carry more than one copy of a title, and the app is only dropped when every copy is
			// on that version.
			installed := installedVersions[appKey]
			upToDate := len(installed) > 0
			for _, installedVersion := range installed {
				if installedVersion != app.InstallerVersion {
					upToDate = false
					break
				}
			}
			if upToDate {
				alreadyUpdatedTitleIDs = append(alreadyUpdatedTitleIDs, app.SoftwareTitleID)
				continue
			}

			// A self-service or admin install patches the app just as much as this notification would, and
			// inventory can be an hour behind the install, so an install Fleet finished since the
			// notification was created counts too. Read by title, so an install that went through an
			// installer replaced during the countdown still counts.
			var installedByFleet bool
			for _, install := range installsByTitle[appKey] {
				if install.Status != nil && *install.Status == fleet.SoftwareInstalled && install.UpdatedAt.After(countdown.CreatedAt) {
					installedByFleet = true
					break
				}
			}
			if installedByFleet {
				alreadyUpdatedTitleIDs = append(alreadyUpdatedTitleIDs, app.SoftwareTitleID)
				continue
			}
			remaining = append(remaining, app)
		}

		// Render builds the toast from this table, so the rows have to go for the reminder to stop naming
		// apps the end user already updated.
		if len(alreadyUpdatedTitleIDs) > 0 {
			err = k.ds.DeletePatchNotificationApps(ctx, countdown.NotificationUUID, alreadyUpdatedTitleIDs)
			if err != nil {
				errs = append(errs, ctxerr.Wrapf(ctx, err,
					"delete patch notification apps that no longer need updating: notification_uuid=%s", countdown.NotificationUUID))
				continue
			}
		}

		if sendReminder {
			// Nothing is left to warn about, and the notification would now render nothing, so close it.
			if len(remaining) == 0 {
				_, err = k.notificationSvc.ActOnNotification(ctx, countdown.NotificationUUID)
				if err != nil {
					errs = append(errs, ctxerr.Wrapf(ctx, err,
						"act on a patch notification with nothing left to update: notification_uuid=%s", countdown.NotificationUUID))
				}
				continue
			}
			err = k.notificationSvc.DelayNotification(ctx, countdown.NotificationUUID, now, patchNotificationReminderPayload)
			if err != nil {
				errs = append(errs, ctxerr.Wrapf(ctx, err, "send patch notification reminder: notification_uuid=%s", countdown.NotificationUUID))
			}
			continue
		}

		// Acted first, so an Update now arriving at the same moment cannot queue the same installs twice.
		var acted bool
		acted, err = k.notificationSvc.ActOnNotification(ctx, countdown.NotificationUUID)
		if err != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, err, "act on patch notification: notification_uuid=%s", countdown.NotificationUUID))
			continue
		}
		if !acted || len(remaining) == 0 {
			continue
		}

		// One InsertSoftwareInstallRequest per app left, so this call is what the pass costs to write.
		// Stays acted on failure: there is no second press to finish the rest here, and the patch policy
		// opens a new notification for whatever is left unpatched.
		_, err = k.queuePatchNotificationInstalls(ctx, countdown.NotificationUUID, countdown.HostID, remaining, installsByTitle)
		if err != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, err, "queue patch notification installs at the deadline: notification_uuid=%s", countdown.NotificationUUID))
		}
	}
	return errors.Join(errs...)
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

	// The deadline pass has the install history from the verify it just did. Update now has done no
	// such read, so it makes the same one: without it the pending check in the queue below cannot tell
	// "no install" from "not looked up".
	apps, err := k.ds.ListPatchNotificationApps(ctx, notification.UUID)
	if err != nil {
		// Nothing was queued, so undo the act above and let the next press try.
		k.putPatchNotificationBack(ctx, notification)
		return nil, ctxerr.Wrap(ctx, err, "list patch notification apps")
	}

	softwareTitleIDs := make([]uint, 0, len(apps))
	for _, app := range apps {
		softwareTitleIDs = append(softwareTitleIDs, app.SoftwareTitleID)
	}
	installsByTitle, err := k.ds.ListHostLastTitleInstallData(ctx, []uint{notification.HostID}, softwareTitleIDs)
	if err != nil {
		k.putPatchNotificationBack(ctx, notification)
		return nil, ctxerr.Wrapf(ctx, err, "list host last title install data: host_id=%d", notification.HostID)
	}

	canPutBack, queueErr := k.queuePatchNotificationInstalls(ctx, notification.UUID, notification.HostID, apps, installsByTitle)
	if queueErr != nil {
		// Back to the status it had, so the next press finishes the rest.
		if canPutBack {
			k.putPatchNotificationBack(ctx, notification)
		}
		return nil, queueErr
	}

	return k.renderView(ctx, notification, true)
}

func (k *patchNotificationKind) putPatchNotificationBack(ctx context.Context, notification *notifications_api.EndUserNotification) {
	err := k.notificationSvc.SetNotificationStatus(ctx, notification.UUID, notification.Status, nil,
		[]string{notifications_api.EndUserNotificationActed})
	if err != nil {
		k.logger.ErrorContext(ctx, "failed to put the patch notification back",
			"notification_uuid", notification.UUID, "err", err)
	}
}

func (k *patchNotificationKind) queuePatchNotificationInstalls(
	ctx context.Context,
	notificationUUID string,
	hostID uint,
	apps []fleet.PatchNotificationAppDetail,
	installsByTitle map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData,
) (canPutBack bool, err error) {
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
		// An install of the app already on its way, however it was requested, must not be queued a
		// second time.
		var installPending bool
		for _, install := range installsByTitle[fleet.HostSoftwareTitleKey{HostID: hostID, SoftwareTitleID: app.SoftwareTitleID}] {
			if install.Status != nil && *install.Status == fleet.SoftwareInstallPending {
				installPending = true
				break
			}
		}
		if installPending {
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
