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

func (ds *Datastore) PatchNotificationCoversApp(ctx context.Context, hostID uint, softwareTitleID uint) (bool, error) {
	// Pending and dispatched are the states where a notification still speaks for
	// its apps. Once it has failed, expired, or the end user acted on it, the app
	// needs telling about again.
	const selectStmt = `
SELECT 1
FROM patch_notification_apps pna
	JOIN notifications_end_user eun ON eun.uuid = pna.notification_uuid
WHERE eun.host_id = ?
	AND eun.status IN (?, ?)
	AND pna.software_title_id = ?
LIMIT 1
`

	var covered bool
	// primary, not the replica: a host's skips arrive one at a time and each one
	// has to see what the one before it wrote
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &covered, selectStmt,
		hostID, notifications_api.EndUserNotificationPending, notifications_api.EndUserNotificationDispatched,
		softwareTitleID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "check patch notification covers app")
	}
	return covered, nil
}

func (ds *Datastore) UnsentPatchNotificationForHost(ctx context.Context, hostID uint) (string, error) {
	// attempt_count = 0 rather than just pending: a notification that went out
	// and came back pending (retried, or the end user delayed it) is already on a
	// schedule the end user has seen, so a new app doesn't belong on it.
	const selectStmt = `
SELECT eun.uuid
FROM notifications_end_user eun
	JOIN patch_notifications pn ON pn.notification_uuid = eun.uuid
WHERE eun.host_id = ?
	AND eun.status = ?
	AND eun.attempt_count = 0
ORDER BY eun.id DESC
LIMIT 1
`

	var notificationUUID string
	// primary, not the replica: the skip before this one may have just created it
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &notificationUUID, selectStmt,
		hostID, notifications_api.EndUserNotificationPending,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", ctxerr.Wrap(ctx, err, "get unsent patch notification for host")
	}
	return notificationUUID, nil
}

func (ds *Datastore) NewPatchNotification(ctx context.Context, notificationUUID string) error {
	const insertStmt = `INSERT INTO patch_notifications (notification_uuid) VALUES (?)`

	if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "insert patch notification")
	}
	return nil
}

func (ds *Datastore) AddPatchNotificationApp(ctx context.Context, notificationUUID string, app fleet.PatchNotificationApp) error {
	// INSERT IGNORE because two skips for the same app can race, and the second
	// one has nothing to add.
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
	// The display name and the icon are both per-team, so they come through the
	// notification's host rather than off the title.
	const selectStmt = `
SELECT
	pna.policy_id,
	pna.software_title_id,
	pna.software_installer_id,
	COALESCE(st.name, '') AS name,
	COALESCE(NULLIF(stdn.display_name, ''), st.name, '') AS display_name,
	sti.software_title_id IS NOT NULL AS has_icon
FROM patch_notification_apps pna
	JOIN notifications_end_user eun ON eun.uuid = pna.notification_uuid
	JOIN hosts h ON h.id = eun.host_id
	LEFT JOIN software_titles st ON st.id = pna.software_title_id
	LEFT JOIN software_title_display_names stdn
		ON stdn.software_title_id = pna.software_title_id AND stdn.team_id = COALESCE(h.team_id, 0)
	LEFT JOIN software_title_icons sti
		ON sti.software_title_id = pna.software_title_id AND sti.team_id = COALESCE(h.team_id, 0)
WHERE pna.notification_uuid = ?
ORDER BY display_name, pna.software_title_id
`

	var apps []fleet.PatchNotificationAppDetail
	// primary, not the replica: a short list here means update now queues fewer
	// installs than the end user was shown and then marks the notification done,
	// so the apps it missed are never installed
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &apps, selectStmt, notificationUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list patch notification apps")
	}
	return apps, nil
}
