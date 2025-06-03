package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
	_, err := ds.writer(ctx).ExecContext(ctx, createStmt, event.HostSerial, event.HostUUID, event.HostID, event.EventType, event.ActivityID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting host lifecycle event")
	}
	return nil, nil
}
