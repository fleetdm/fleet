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

// The COALESCE sets the deadline once, so a retry, a re-dispatch or a duplicate script result can't
// move it. The insert covers a notification whose patch_notifications row was never written, which
// would otherwise never be patched.
func (ds *Datastore) SetPatchNotificationInstallAt(ctx context.Context, notificationUUID string, installAt time.Time) (time.Time, error) {
	const upsertStmt = `
INSERT INTO patch_notifications (notification_uuid, install_at) VALUES (?, ?)
ON DUPLICATE KEY UPDATE install_at = COALESCE(install_at, VALUES(install_at))
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

func (ds *Datastore) ResetPatchNotification(ctx context.Context, notificationUUID string) error {
	const updateStmt = `UPDATE patch_notifications SET install_at = NULL WHERE notification_uuid = ?`

	if _, err := ds.writer(ctx).ExecContext(ctx, updateStmt, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "reset patch notification")
	}
	return nil
}

func (ds *Datastore) ListPatchNotificationsDue(ctx context.Context, cutoff time.Time, limit int) ([]fleet.PatchNotificationDue, error) {
	// host_online uses the same expression as the online host filter, so the countdown and the host
	// list agree on what offline means.
	selectStmt := fmt.Sprintf(`
SELECT
	pn.notification_uuid,
	pn.install_at,
	eun.host_id,
	eun.status,
	eun.payload,
	eun.displayed_at,
	eun.created_at,
	DATE_ADD(COALESCE(hst.seen_time, h.created_at), INTERVAL LEAST(h.distributed_interval, h.config_tls_refresh) + %d SECOND) > NOW(6) AS host_online
FROM patch_notifications pn
	JOIN notifications_end_user eun ON eun.uuid = pn.notification_uuid
	JOIN hosts h ON h.id = eun.host_id
	LEFT JOIN host_seen_times hst ON hst.host_id = h.id
-- a notification with no deadline was never displayed, so it has no countdown to run
WHERE pn.install_at IS NOT NULL
	AND pn.install_at <= ?
	-- acted notifications have already been patched, failed and expired ones never will be
	AND eun.status IN (?, ?)
ORDER BY pn.install_at
LIMIT ?
`, fleet.OnlineIntervalBuffer)

	var due []fleet.PatchNotificationDue
	// primary, not the replica: the display that sets the deadline can be seconds old
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &due, selectStmt,
		cutoff, notifications_api.EndUserNotificationPending, notifications_api.EndUserNotificationDispatched, limit,
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
	COALESCE(st.name, '') AS name,
	COALESCE(NULLIF(stdn.display_name, ''), st.name, '') AS display_name,
	sti.software_title_id IS NOT NULL AS has_icon
FROM patch_notification_apps pna
	JOIN notifications_end_user neu ON neu.uuid = pna.notification_uuid
	JOIN hosts h ON h.id = neu.host_id
	LEFT JOIN software_titles st ON st.id = pna.software_title_id
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
