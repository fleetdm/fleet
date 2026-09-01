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
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &exists, selectStmt,
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

func (ds *Datastore) PatchNotificationInstallsQueued(ctx context.Context, notificationUUID string) (bool, error) {
	const selectStmt = `SELECT installs_queued_at IS NOT NULL FROM patch_notifications WHERE notification_uuid = ?`

	var queued bool
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &queued, selectStmt, notificationUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "check patch notification installs queued")
	}
	return queued, nil
}

func (ds *Datastore) SetPatchNotificationInstallsQueued(ctx context.Context, notificationUUID string) error {
	const updateStmt = `
UPDATE patch_notifications SET installs_queued_at = NOW(6)
WHERE notification_uuid = ? AND installs_queued_at IS NULL
`

	if _, err := ds.writer(ctx).ExecContext(ctx, updateStmt, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set patch notification installs queued")
	}
	return nil
}

func (ds *Datastore) NewPatchNotification(ctx context.Context, notificationUUID string) error {
	const insertStmt = `INSERT INTO patch_notifications (notification_uuid) VALUES (?)`

	if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "insert patch notification")
	}
	return nil
}

func (ds *Datastore) AddPatchNotificationApp(ctx context.Context, notificationUUID string, app fleet.PatchNotificationApp) error {
	const insertStmt = `
INSERT IGNORE INTO patch_notification_apps
	(notification_uuid, policy_id, software_title_id, software_installer_id)
VALUES (?, ?, ?, ?)
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
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &apps, selectStmt, notificationUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list patch notification apps")
	}
	return apps, nil
}
