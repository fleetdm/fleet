package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewAndroidHost(ctx context.Context, host *fleet.AndroidHost) (*fleet.AndroidHost, error) {
	if !host.IsValid() {
		return nil, ctxerr.New(ctx, "valid Android host is required")
	}

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// We use node_key as a unique identifier for the host table row. It matches: android/{deviceID}.
		sqlStatement := `
		INSERT INTO hosts (
		    node_key,
			detail_updated_at,
			label_updated_at,
			policy_updated_at,
		    hostname,
			computer_name,
			platform,
			os_version,
		    build,
			memory,
			team_id,
			hardware_serial
		) VALUES (
   			:node_key,
			:detail_updated_at,
			:label_updated_at,
			:policy_updated_at,
			:hostname,
			:computer_name,
			:platform,
			:os_version,
			:build,
			:memory,
			:team_id,
			:hardware_serial
		) ON DUPLICATE KEY UPDATE
			detail_updated_at = VALUES(detail_updated_at),
			label_updated_at = VALUES(label_updated_at),
			policy_updated_at = VALUES(policy_updated_at),
			hostname = VALUES(hostname),
			computer_name = VALUES(computer_name),
			platform = VALUES(platform),
			os_version = VALUES(os_version),
			build = VALUES(build),
			memory = VALUES(memory),
			team_id = VALUES(team_id),
			hardware_serial = VALUES(hardware_serial)
		`
		sqlStatement, args, err := sqlx.Named(sqlStatement, host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "could not bind parameters for new Android host")
		}
		result, err := tx.ExecContext(ctx, sqlStatement, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host")
		}
		id, _ := result.LastInsertId()
		host.Host.ID = uint(id)
		host.Device.HostID = host.Host.ID

		err = upsertHostDisplayNames(ctx, tx, *host.Host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host display name")
		}

		host.Device, err = ds.androidDS.CreateDeviceTx(ctx, host.Device, tx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating new Android device")
		}
		return nil
	})
	return host, err
}

func (ds *Datastore) AndroidHostLite(ctx context.Context, deviceID string) (*fleet.AndroidHost, error) {
	stmt := `SELECT ad.id, ad.host_id, ad.device_id, ad.enterprise_specific_id, ad.policy_id, ad.last_policy_sync_time
		FROM android_devices ad
		LEFT JOIN hosts h ON ad.host_id = h.id
		WHERE device_id = ?`
	var device android.Device
	err := sqlx.GetContext(ctx, ds.reader(ctx), &device, stmt, deviceID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android device").WithName(deviceID)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting device by device ID")
	}
	return &fleet.AndroidHost{
		Host:   &fleet.Host{ID: device.HostID},
		Device: &device,
	}, nil
}
