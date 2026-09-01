package mysql

import (
	"context"
	"database/sql"
	"errors"

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
	JOIN patch_notifications pn ON pn.notification_uuid = pna.notification_uuid
WHERE neu.host_id = ?
	AND neu.status IN (?, ?)
	AND pn.installs_queued_at IS NULL
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

func (ds *Datastore) GetPatchNotification(ctx context.Context, notificationUUID string) (*fleet.PatchNotification, error) {
	const selectStmt = `
SELECT notification_uuid, install_at, reminder_sent_at, installs_queued_at
FROM patch_notifications
WHERE notification_uuid = ?
`

	var notification fleet.PatchNotification
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &notification, selectStmt, notificationUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("PatchNotification").WithName(notificationUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get patch notification")
	}
	return &notification, nil
}

func (ds *Datastore) SetPatchNotificationInstallsQueued(ctx context.Context, notificationUUID string) (bool, error) {
	const updateStmt = `
UPDATE patch_notifications SET installs_queued_at = NOW(6)
WHERE notification_uuid = ? AND installs_queued_at IS NULL
`

	res, err := ds.writer(ctx).ExecContext(ctx, updateStmt, notificationUUID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "set patch notification installs queued")
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "rows affected setting patch notification installs queued")
	}
	return rows > 0, nil
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

func (ds *Datastore) ListPatchNotificationApps(ctx context.Context, notificationUUID string) ([]fleet.PatchNotificationAppDetail, error) {
	const selectStmt = `
SELECT
	pna.policy_id,
	pna.software_title_id,
	pna.software_installer_id,
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
