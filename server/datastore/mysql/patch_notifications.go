package mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	notifications_api "github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) PatchNotificationExistsForApp(ctx context.Context, hostID uint, softwareTitleID uint) (bool, error) {
	const selectStmt = `
SELECT 1
FROM patch_notification_apps pna
	JOIN notifications_end_user neu ON neu.uuid = pna.notification_uuid
WHERE neu.host_id = ?
	AND neu.status IN (?, ?)
	AND pna.software_title_id = ?
LIMIT 1
`

	var exists bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &exists, selectStmt,
		hostID, notifications_api.EndUserNotificationPending, notifications_api.EndUserNotificationDispatched,
		softwareTitleID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "check patch notification exists for app")
	}
	return exists, nil
}

func (ds *Datastore) DisplayedPatchNotificationExistsForApp(ctx context.Context, hostID uint, softwareTitleID uint) (bool, error) {
	// displayed_at is what sets install_at, so only a displayed notification has a deadline to install on
	const selectStmt = `
SELECT 1
FROM patch_notification_apps pna
	JOIN notifications_end_user neu ON neu.uuid = pna.notification_uuid
WHERE neu.host_id = ?
	AND neu.status = ?
	AND neu.displayed_at IS NOT NULL
	AND pna.software_title_id = ?
LIMIT 1
`

	var exists bool
	// the primary, because the display that starts the countdown can be seconds old
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &exists, selectStmt,
		hostID, notifications_api.EndUserNotificationDispatched, softwareTitleID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "check displayed patch notification exists for app")
	}
	return exists, nil
}

func (ds *Datastore) NewPatchNotification(ctx context.Context, notificationUUID string) error {
	const insertStmt = `INSERT INTO patch_notifications (notification_uuid) VALUES (?)`

	if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "insert patch notification")
	}
	return nil
}

func (ds *Datastore) AddPatchNotificationApp(ctx context.Context, notificationUUID string, app fleet.PatchNotificationApp) error {
	// Two skips for the same app can pass PatchNotificationExistsForApp before either has inserted,
	// so a duplicate is a no-op rather than an error. Not INSERT IGNORE, which would also swallow
	// a bad notification_uuid or a deleted software title.
	const insertStmt = `
INSERT INTO patch_notification_apps
	(notification_uuid, policy_id, software_title_id, software_installer_id)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE software_title_id = software_title_id
`

	if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt,
		notificationUUID, app.PolicyID, app.SoftwareTitleID, app.SoftwareInstallerID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "insert patch notification app")
	}
	return nil
}

func (ds *Datastore) SetPatchNotificationAppsQueued(ctx context.Context, notificationUUID string, softwareTitleIDs []uint) error {
	if len(softwareTitleIDs) == 0 {
		return nil
	}

	stmt, args, err := sqlx.In(`
UPDATE patch_notification_apps SET install_queued = 1
WHERE notification_uuid = ? AND software_title_id IN (?)
`, notificationUUID, softwareTitleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build set patch notification apps queued update")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "set patch notification apps queued")
	}
	return nil
}

func (ds *Datastore) DeletePatchNotificationApps(ctx context.Context, notificationUUID string, softwareTitleIDs []uint) error {
	if len(softwareTitleIDs) == 0 {
		return nil
	}

	stmt, args, err := sqlx.In(`
DELETE FROM patch_notification_apps
WHERE notification_uuid = ? AND software_title_id IN (?)
`, notificationUUID, softwareTitleIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build delete patch notification apps statement")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete patch notification apps")
	}
	return nil
}

func (ds *Datastore) SetPatchNotificationInstallAt(ctx context.Context, notificationUUID string, installAt time.Time) (time.Time, error) {
	// GREATEST means the deadline only ever moves later, so every notice the end user actually sees
	// gets its full lead time even when the toast takes a while to reach the screen.
	//
	// The insert makes the deadline recordable even for a notification with no patch_notifications
	// row. Creation writes that row first so it is always there, and a notification without one would
	// otherwise display to the end user and never be patched.
	const upsertStmt = `
INSERT INTO patch_notifications (notification_uuid, install_at) VALUES (?, ?)
ON DUPLICATE KEY UPDATE install_at = GREATEST(COALESCE(install_at, VALUES(install_at)), VALUES(install_at))
`

	if _, err := ds.writer(ctx).ExecContext(ctx, upsertStmt, notificationUUID, installAt); err != nil {
		return time.Time{}, ctxerr.Wrap(ctx, err, "set patch notification install at")
	}

	const selectStmt = `SELECT install_at FROM patch_notifications WHERE notification_uuid = ?`

	var stored *time.Time
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &stored, selectStmt, notificationUUID); err != nil {
		return time.Time{}, ctxerr.Wrap(ctx, err, "get patch notification install at")
	}
	// only reachable if the deadline was cleared between the two statements
	if stored == nil {
		return time.Time{}, ctxerr.Errorf(ctx, "patch notification %s has no install at", notificationUUID)
	}
	return *stored, nil
}

func (ds *Datastore) ListPatchNotificationsDue(ctx context.Context, cutoff time.Time, limit int) ([]fleet.PatchNotificationDue, error) {
	const selectStmt = `
SELECT
	pn.notification_uuid,
	pn.install_at,
	neu.host_id,
	neu.status,
	neu.payload,
	neu.displayed_at
FROM
	patch_notifications pn
	JOIN notifications_end_user neu ON neu.uuid = pn.notification_uuid
-- a notification with no deadline was never displayed, so nothing is due for it yet
WHERE
	pn.install_at IS NOT NULL
	AND pn.install_at <= ?
	-- failed and expired notifications will never be patched
	AND (
		-- still being delivered
		neu.status IN (?, ?)
		-- or acted on and left with an app whose install never queued
		OR (
			neu.status = ?
			AND EXISTS (
				SELECT 1
				FROM patch_notification_apps unhandled
				WHERE unhandled.notification_uuid = pn.notification_uuid AND unhandled.install_queued = 0
			)
		)
	)
	-- the reminder needs a displayed first notice and the install needs a displayed reminder, so a
	-- null displayed_at rules out both
	AND neu.displayed_at IS NOT NULL
ORDER BY pn.install_at
LIMIT ?
`

	var due []fleet.PatchNotificationDue
	// reads the primary because the display that sets the deadline can be seconds old
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &due, selectStmt,
		cutoff, notifications_api.EndUserNotificationPending, notifications_api.EndUserNotificationDispatched,
		notifications_api.EndUserNotificationActed, limit,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list patch notifications due")
	}
	return due, nil
}

func (ds *Datastore) ListPatchNotificationApps(ctx context.Context, notificationUUID string) ([]fleet.PatchNotificationAppDetail, error) {
	const selectStmt = `
SELECT
	pna.policy_id,
	pna.software_title_id,
	pna.software_installer_id,
	pna.install_queued,
	pna.created_at,
	COALESCE(st.name, '') AS name,
	COALESCE(NULLIF(stdn.display_name, ''), st.name, '') AS display_name,
	COALESCE(si.version, '') AS installer_version,
	sti.software_title_id IS NOT NULL AS has_icon
FROM patch_notification_apps pna
	JOIN notifications_end_user neu ON neu.uuid = pna.notification_uuid
	JOIN hosts h ON h.id = neu.host_id
	LEFT JOIN software_titles st ON st.id = pna.software_title_id
	LEFT JOIN software_installers si ON si.id = pna.software_installer_id
	LEFT JOIN software_title_display_names stdn
		ON stdn.software_title_id = pna.software_title_id AND stdn.team_id = COALESCE(h.team_id, 0)
	LEFT JOIN software_title_icons sti
		ON sti.software_title_id = pna.software_title_id AND sti.team_id = COALESCE(h.team_id, 0)
WHERE pna.notification_uuid = ?
ORDER BY display_name, pna.software_title_id
`

	var apps []fleet.PatchNotificationAppDetail
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &apps, selectStmt, notificationUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list patch notification apps")
	}
	return apps, nil
}

// ListPatchNotificationAppsForNotifications leaves out the names and icons the toast is built from,
// because RemindAndInstallDuePatches matches versions and queues installs without displaying anything.
func (ds *Datastore) ListPatchNotificationAppsForNotifications(ctx context.Context, notificationUUIDs []string) (map[string][]fleet.PatchNotificationAppDetail, error) {
	if len(notificationUUIDs) == 0 {
		return nil, nil
	}

	stmt, args, err := sqlx.In(`
SELECT
	pna.notification_uuid,
	pna.policy_id,
	pna.software_title_id,
	pna.software_installer_id,
	pna.install_queued,
	pna.created_at,
	COALESCE(si.version, '') AS installer_version
FROM patch_notification_apps pna
	LEFT JOIN software_installers si ON si.id = pna.software_installer_id
WHERE pna.notification_uuid IN (?)
`, notificationUUIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build list patch notification apps for notifications statement")
	}

	var apps []fleet.PatchNotificationAppDetail
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &apps, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list patch notification apps for notifications")
	}

	byNotification := make(map[string][]fleet.PatchNotificationAppDetail, len(notificationUUIDs))
	for _, app := range apps {
		byNotification[app.NotificationUUID] = append(byNotification[app.NotificationUUID], app)
	}
	return byNotification, nil
}
