package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/fleetdm/fleet/v4/server"
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

	// how many due patch notifications one cron pass reads
	duePatchNotificationBatchSize = 500
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
	RemindAndInstallDuePatches(ctx context.Context) error
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
	// RemindAndInstallDuePatches sends the reminder whatever the end user pressed, so the button only
	// has to close the toast, which the page does over the bridge. Leaving the row alone is also what
	// stops the button buying more time.
	return nil, nil
}

func (k *patchNotificationKind) RemindAndInstallDuePatches(ctx context.Context) error {
	now := time.Now().UTC()

	duePatches, err := k.ds.ListPatchNotificationsDue(ctx, now.Add(patchNotificationReminderBefore), duePatchNotificationBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list patch notifications due")
	}
	if len(duePatches) == 0 {
		return nil
	}

	notificationUUIDs := make([]string, 0, len(duePatches))
	hostIDs := make([]uint, 0, len(duePatches))
	for _, duePatch := range duePatches {
		notificationUUIDs = append(notificationUUIDs, duePatch.NotificationUUID)
		hostIDs = append(hostIDs, duePatch.HostID)
	}

	// The apps, the inventory and the install history of every due patch in the batch, read up front so
	// the loop below does writes and installs rather than a handful of reads per notification.
	appsByNotification, err := k.ds.ListPatchNotificationAppsForNotifications(ctx, notificationUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list patch notification apps for the batch")
	}

	// The same title comes up once per host notified about it, so the IN list is deduplicated. The due
	// patches are walked rather than the map, which would put the list in a different order every pass.
	var softwareTitleIDs []uint
	for _, duePatch := range duePatches {
		for _, app := range appsByNotification[duePatch.NotificationUUID] {
			softwareTitleIDs = append(softwareTitleIDs, app.SoftwareTitleID)
		}
	}
	softwareTitleIDs = server.RemoveDuplicatesFromSlice(softwareTitleIDs)

	versions, err := k.ds.ListSoftwareTitleVersionsForHosts(ctx, hostIDs, softwareTitleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host software versions for the batch")
	}
	installedVersions := make(map[fleet.HostSoftwareTitleKey][]string, len(versions))
	for _, version := range versions {
		key := fleet.HostSoftwareTitleKey{HostID: version.HostID, SoftwareTitleID: version.SoftwareTitleID}
		installedVersions[key] = append(installedVersions[key], version.Version)
	}

	installsByTitle, err := k.ds.ListLastTitleInstallDataForHosts(ctx, hostIDs, softwareTitleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list host last title install data for the batch")
	}

	// One notification failing must not hold up the rest of the batch.
	var errs []error
	for _, duePatch := range duePatches {
		err = k.remindOrInstallDuePatch(ctx, duePatch, appsByNotification[duePatch.NotificationUUID], installedVersions, installsByTitle, now)
		if err != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, err, "notification_uuid=%s", duePatch.NotificationUUID))
		}
	}
	return errors.Join(errs...)
}

func (k *patchNotificationKind) remindOrInstallDuePatch(
	ctx context.Context,
	duePatch fleet.PatchNotificationDue,
	apps []fleet.PatchNotificationAppDetail,
	installedVersions map[fleet.HostSoftwareTitleKey][]string,
	installsByTitle map[fleet.HostSoftwareTitleKey][]*fleet.HostLastInstallData,
	now time.Time,
) error {
	// A re-dispatch clears displayed_at, so a null one means the reminder has been queued but not
	// displayed yet.
	if duePatch.DisplayedAt == nil {
		return nil
	}

	notificationIsReminder, err := patchNotificationIsReminder(duePatch.Payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read patch notification payload")
	}

	// Which notice was displayed last decides what happens next.
	if !notificationIsReminder {
		if duePatch.Status != notifications_api.EndUserNotificationDispatched {
			return nil
		}
	} else if now.Before(duePatch.InstallAt) {
		// the reminder was displayed and its five minutes are not up
		return nil
	}

	// leave out the apps updated since the notification was created, so the reminder stops naming them and nothing is queued for them
	var alreadyUpdatedTitleIDs []uint
	remaining := make([]fleet.PatchNotificationAppDetail, 0, len(apps))
	for _, app := range apps {
		if app.SoftwareInstallerID == nil {
			remaining = append(remaining, app)
			continue
		}

		appKey := fleet.HostSoftwareTitleKey{HostID: duePatch.HostID, SoftwareTitleID: app.SoftwareTitleID}

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

		var installedByFleet bool
		for _, install := range installsByTitle[appKey] {
			if install.Status != nil && *install.Status == fleet.SoftwareInstalled && install.UpdatedAt.After(app.CreatedAt) {
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

	// Render builds the toast from these rows, so updated apps stop being named
	if len(alreadyUpdatedTitleIDs) > 0 {
		err := k.ds.DeletePatchNotificationApps(ctx, duePatch.NotificationUUID, alreadyUpdatedTitleIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete patch notification apps that no longer need updating")
		}
	}

	if !notificationIsReminder {
		// nothing left to update, so the notification closes instead of reminding
		if len(remaining) == 0 {
			_, err := k.notificationSvc.ActOnNotification(ctx, duePatch.NotificationUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "act on a patch notification with nothing left to update")
			}
		} else {
			// re-send the notification with the reminder payload, so the end user gets the 5 minute notice
			err := k.notificationSvc.DelayNotification(ctx, duePatch.NotificationUUID, now, patchNotificationReminderPayload)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "send patch notification reminder")
			}
		}
		return nil
	}

	// Moving the status to acted is what claims the queueing, so an Update now press and this pass
	// cannot both send the same installs.
	isStatusActed := duePatch.Status == notifications_api.EndUserNotificationActed
	actedInThisPass, err := k.notificationSvc.ActOnNotification(ctx, duePatch.NotificationUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "act on patch notification")
	}
	// Not claiming the queueing has two causes. Acted at read time is an earlier pass that stopped
	// part way, which this one finishes. Acted only now is an Update now press, which queues the
	// installs instead. Or verify dropped every app.
	if (!actedInThisPass && !isStatusActed) || len(remaining) == 0 {
		return nil
	}

	_, err = k.queuePatchNotificationInstalls(ctx, duePatch.NotificationUUID, duePatch.HostID, remaining, installsByTitle)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "queue patch notification installs at the deadline")
	}
	return nil
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
	actedInThisRequest, err := k.notificationSvc.ActOnNotification(ctx, notification.UUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "act on patch notification")
	}
	if !actedInThisRequest {
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
	installsByTitle, err := k.ds.ListLastTitleInstallDataForHosts(ctx, []uint{notification.HostID}, softwareTitleIDs)
	if err != nil {
		k.putPatchNotificationBack(ctx, notification)
		return nil, ctxerr.Wrapf(ctx, err, "list host last title install data: host_id=%d", notification.HostID)
	}

	installsRecorded, queueErr := k.queuePatchNotificationInstalls(ctx, notification.UUID, notification.HostID, apps, installsByTitle)
	if queueErr != nil {
		// hand the notification back only when what went out was recorded, so the next press finishes the rest
		if installsRecorded {
			k.putPatchNotificationBack(ctx, notification)
		}
		return nil, queueErr
	}

	return k.renderView(ctx, notification, true)
}

func (k *patchNotificationKind) putPatchNotificationBack(ctx context.Context, notification *notifications_api.EndUserNotification) {
	err := k.notificationSvc.SetNotificationStatus(ctx, notification.UUID, notification.Status, nil, []string{notifications_api.EndUserNotificationActed})
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
) (installsRecorded bool, err error) {
	// install_queued records that an app has been dealt with, whether that meant queueing an install
	// or deciding not to. An app left unmarked is one this pass could not queue, which is what tells
	// a later pass the notification was acted on with installs still to go out.
	var queuedTitleIDs []uint
	var queueErrs []error
	for _, app := range apps {
		if app.SoftwareInstallerID == nil {
			k.logger.InfoContext(ctx, "skipping patch notification install for software with no installer",
				"notification_uuid", notificationUUID, "software_title_id", app.SoftwareTitleID)
			queuedTitleIDs = append(queuedTitleIDs, app.SoftwareTitleID)
			continue
		}
		// this app was dealt with by an earlier attempt
		if app.InstallQueued {
			continue
		}

		// an install carrying override_pre_install_query would skip while the app is open, so it needs to be queued again
		titleKey := fleet.HostSoftwareTitleKey{HostID: hostID, SoftwareTitleID: app.SoftwareTitleID}
		installPending := slices.ContainsFunc(installsByTitle[titleKey], func(install *fleet.HostLastInstallData) bool {
			return install.Status != nil && *install.Status == fleet.SoftwareInstallPending && !install.OverridePreInstallQuery
		})
		if installPending {
			queuedTitleIDs = append(queuedTitleIDs, app.SoftwareTitleID)
			continue
		}

		// OverridePreInstallQuery stays false, so the app open check does not stop the install
		_, insertErr := k.ds.InsertSoftwareInstallRequest(ctx, hostID, *app.SoftwareInstallerID, fleet.HostSoftwareInstallOptions{PolicyID: app.PolicyID})
		if insertErr != nil {
			// One app Fleet cannot queue must not hold up the rest. This app stays unmarked below,
			// which is what brings the notification back to be finished.
			queueErrs = append(queueErrs, ctxerr.Wrapf(ctx, insertErr,
				"insert software install request: host_id=%d, software_installer_id=%d", hostID, *app.SoftwareInstallerID))
			continue
		}

		queuedTitleIDs = append(queuedTitleIDs, app.SoftwareTitleID)
	}

	// the installs are already out, so a failure here leaves them unrecorded and a second attempt would repeat them
	err = k.ds.SetPatchNotificationAppsQueued(ctx, notificationUUID, queuedTitleIDs)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "set patch notification apps queued")
	}
	return true, errors.Join(queueErrs...)
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
	}
	// install_at follows displayed_at, which the notifications context only records while the
	// notification is still dispatched. A result from a superseded script arriving after a delay or
	// an update_now must not move install_at. notification.DisplayedAt is nil either way here, since
	// RecordOutcome loaded that copy before the outcome write.
	if outcome.Displayed && notification.Status == notifications_api.EndUserNotificationDispatched {
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
