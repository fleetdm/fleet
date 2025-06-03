package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

/*
 * 		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		host_serial varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
		host_uuid VARCHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
		host_id INT UNSIGNED NOT NULL,
		event_type ENUM('started_mdm_setup', 'completed_mdm_setup', 'started_mdm_migration', 'completed_mdm_migration') COLLATE utf8mb4_unicode_ci NOT NULL,
		created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
		activity_id INT UNSIGNED,
*/

func (ds *Datastore) CreateHostLifecycleEvent(ctx context.Context, event *fleet.HostLifecycleEvent) (*fleet.HostLifecycleEvent, error) {
	createStmt := `INSERT INTO host_lifecycle_events (host_serial, host_uuid, host_id, event_type, activity_id) VALUES (?, ?, ?, ?, ?)`
	res, err := ds.writer(ctx).ExecContext(ctx, createStmt, event.HostSerial, event.HostUUID, event.HostID, event.EventType, event.ActivityID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting host lifecycle event")
	}
	eventID, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last insert id for host lifecycle event")
	}
	event.ID = uint(eventID)
	return event, nil
}

func (ds *Datastore) GetLastLifecycleEventForHost(ctx context.Context, hostUUID string) (*fleet.HostLifecycleEvent, error) {
	query := `SELECT id, host_serial, host_uuid, host_id, event_type, created_at, activity_id FROM host_lifecycle_events WHERE host_uuid = ? ORDER BY created_at DESC LIMIT 1`
	event := &fleet.HostLifecycleEvent{}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), event, query, hostUUID); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("host lifecycle event for host")
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting last lifecycle event for host")
	}
	return event, nil
}
