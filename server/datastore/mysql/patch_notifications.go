package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	// match only a displayed notification, displayed_at is what sets install_at
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
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &exists, selectStmt,
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

func (ds *Datastore) GetPatchNotification(ctx context.Context, notificationUUID string) (*fleet.PatchNotification, error) {
	const selectStmt = `SELECT notification_uuid, install_at FROM patch_notifications WHERE notification_uuid = ?`

	var patchNotification fleet.PatchNotification
	// reads the primary because the display that sets the deadline can be seconds old
	err := sqlx.GetContext(ctx, ds.writer(ctx), &patchNotification, selectStmt, notificationUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get patch notification")
	}
	return &patchNotification, nil
}

func (ds *Datastore) ClearPatchNotificationInstallAt(ctx context.Context, notificationUUID string) error {
	const updateStmt = `UPDATE patch_notifications SET install_at = NULL WHERE notification_uuid = ?`

	_, err := ds.writer(ctx).ExecContext(ctx, updateStmt, notificationUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clear patch notification install at")
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
	// host_online uses the same window as the host list's online status: the shorter of the two
	// check-in intervals, plus the buffer that keeps a host from flapping.
	selectStmt := fmt.Sprintf(`
SELECT
	pn.notification_uuid,
	pn.install_at,
	neu.host_id,
	neu.status,
	neu.payload,
	neu.displayed_at,
	DATE_ADD(
		COALESCE(hst.seen_time, h.created_at),
		INTERVAL LEAST(h.distributed_interval, h.config_tls_refresh) + %d SECOND
	) > NOW(6) AS host_online
FROM
	patch_notifications pn
	JOIN notifications_end_user neu ON neu.uuid = pn.notification_uuid
	JOIN hosts h ON h.id = neu.host_id
	LEFT JOIN host_seen_times hst ON hst.host_id = h.id
WHERE
	pn.install_at IS NOT NULL
	AND pn.install_at <= ?
	AND (
		-- still being delivered
		neu.status IN (?, ?)
		-- status is acted (user clicked update now) and left with an app whose install never queued
		OR (
			neu.status = ?
			AND EXISTS (
				SELECT 1
				FROM patch_notification_apps unhandled
				WHERE unhandled.notification_uuid = pn.notification_uuid AND unhandled.install_queued = 0
			)
		)
	)
	-- For 5 minute reminder: 1 hour notification was displayed
	-- For force installs: 5 minute reminder was displayed
	AND neu.displayed_at IS NOT NULL
ORDER BY pn.install_at
LIMIT ?
`, fleet.OnlineIntervalBuffer)

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

func (ds *Datastore) ListPatchNotificationAppInstallStatuses(ctx context.Context, notificationUUID string) (map[uint]fleet.SoftwareInstallerStatus, error) {
	// Match every installer the title has, not the nullable host_software_installs.software_title_id,
	// so an install recorded against a different installer id for the title still reports. Skip installs
	// older than the app's row, which patched an earlier version. Read execution_status rather than
	// status, which nulls out once the app leaves the host's inventory, so uninstalling a patched app
	// does not put its row back to installing. Leave out installs carrying the app open query, which skip
	// while the app is open and report a failure the notification never asked for.
	const selectStmt = `
SELECT
	pna.software_title_id,
	hsi.execution_status AS status
FROM patch_notification_apps pna
	JOIN notifications_end_user neu ON neu.uuid = pna.notification_uuid
	JOIN software_installers si ON si.title_id = pna.software_title_id
	JOIN host_software_installs hsi ON hsi.software_installer_id = si.id
		AND hsi.host_id = neu.host_id
		AND hsi.updated_at > pna.created_at
		AND hsi.override_pre_install_query = 0
		AND hsi.execution_status IS NOT NULL
WHERE pna.notification_uuid = ?
ORDER BY hsi.id
`

	var rows []struct {
		SoftwareTitleID uint                          `db:"software_title_id"`
		Status          fleet.SoftwareInstallerStatus `db:"status"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, selectStmt, notificationUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list patch notification app install statuses")
	}

	// Keep the newest install for each title, which the oldest-first ordering leaves last in the map.
	statuses := make(map[uint]fleet.SoftwareInstallerStatus, len(rows))
	for _, row := range rows {
		statuses[row.SoftwareTitleID] = row.Status
	}
	return statuses, nil
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
