package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetHost(ctx context.Context, fleetEnterpriseID uint, deviceID string) (*android.Host, error) {
	stmt := `SELECT enterprise_id, device_id, host_id FROM android_hosts WHERE enterprise_id = ? AND device_id = ?`
	var host android.Host
	err := sqlx.GetContext(ctx, ds.reader(ctx), &host, stmt, fleetEnterpriseID, deviceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting host")
	}
	return &host, nil
}

func (ds *Datastore) AddHost(ctx context.Context, host *android.Host) error {
	stmt := `INSERT INTO android_hosts (enterprise_id, device_id, host_id) VALUES (?, ?, ?)`
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, host.FleetEnterpriseID, host.DeviceID, host.HostID)
	return ctxerr.Wrap(ctx, err, "adding host")
}
