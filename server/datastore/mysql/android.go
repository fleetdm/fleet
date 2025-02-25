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

func (ds *Datastore) GetAndroidDS() android.Datastore {
	return ds.androidDS
}

func (ds *Datastore) NewAndroidHost(ctx context.Context, host *fleet.AndroidHost) (*fleet.AndroidHost, error) {
	if !host.IsValid() {
		return nil, ctxerr.New(ctx, "valid Android host is required")
	}
	ds.setTimesToNonZero(host)

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// We use node_key as a unique identifier for the host table row. It matches: android/{enterpriseSpecificID}.
		stmt := `
		INSERT INTO hosts (
			node_key,
			hostname,
			computer_name,
			platform,
			os_version,
			build,
			memory,
			team_id,
			hardware_serial,
			detail_updated_at,
			label_updated_at
		) VALUES (
			:node_key,
			:hostname,
			:computer_name,
			:platform,
			:os_version,
			:build,
			:memory,
			:team_id,
			:hardware_serial,
			:detail_updated_at,
			:label_updated_at
		) ON DUPLICATE KEY UPDATE
			hostname = VALUES(hostname),
			computer_name = VALUES(computer_name),
			platform = VALUES(platform),
			os_version = VALUES(os_version),
			build = VALUES(build),
			memory = VALUES(memory),
			team_id = VALUES(team_id),
			hardware_serial = VALUES(hardware_serial),
			detail_updated_at = VALUES(detail_updated_at),
			label_updated_at = VALUES(label_updated_at)
		`
		stmt, args, err := sqlx.Named(stmt, host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "could not bind parameters for new Android host")
		}
		result, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host")
		}
		id, _ := result.LastInsertId()
		host.Host.ID = uint(id) // nolint:gosec
		host.Device.HostID = host.Host.ID

		err = upsertHostDisplayNames(ctx, tx, *host.Host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host display name")
		}
		err = ds.androidDS.InsertHostLabelMembershipTx(ctx, tx, host.Host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new Android host label membership")
		}

		host.Device, err = ds.androidDS.CreateDeviceTx(ctx, tx, host.Device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating new Android device")
		}
		return nil
	})
	return host, err
}

// setTimesToNonZero to avoid issues with MySQL.
func (ds *Datastore) setTimesToNonZero(host *fleet.AndroidHost) {
	if host.DetailUpdatedAt.IsZero() {
		host.DetailUpdatedAt = common_mysql.GetDefaultNonZeroTime()
	}
	if host.LabelUpdatedAt.IsZero() {
		host.LabelUpdatedAt = common_mysql.GetDefaultNonZeroTime()
	}
}

func (ds *Datastore) UpdateAndroidHost(ctx context.Context, host *fleet.AndroidHost) error {
	if !host.IsValid() {
		return ctxerr.New(ctx, "valid Android host is required")
	}
	ds.setTimesToNonZero(host)

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		stmt := `
		UPDATE hosts SET
			team_id = :team_id,
			detail_updated_at = :detail_updated_at,
			label_updated_at = :label_updated_at,
			hostname = :hostname,
			computer_name = :computer_name,
			platform = :platform,
			os_version = :os_version,
			build = :build,
			memory = :memory,
			hardware_serial = :hardware_serial
		WHERE id = :id
		`
		stmt, args, err := sqlx.Named(stmt, host)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "could not bind parameters for updating Android host")
		}
		_, err = tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update Android host")
		}

		err = ds.androidDS.UpdateDeviceTx(ctx, tx, host.Device)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update Android device")
		}
		return nil
	})
	return err
}

func (ds *Datastore) AndroidHostLite(ctx context.Context, enterpriseSpecificID string) (*fleet.AndroidHost, error) {
	type liteHost struct {
		TeamID *uint `db:"team_id"`
		*android.Device
	}
	stmt := `SELECT
		h.team_id,
		ad.id,
		ad.host_id,
		ad.device_id,
		ad.enterprise_specific_id,
		ad.android_policy_id,
		ad.last_policy_sync_time
		FROM android_devices ad
		JOIN hosts h ON ad.host_id = h.id
		WHERE ad.enterprise_specific_id = ?`
	var host liteHost
	err := sqlx.GetContext(ctx, ds.reader(ctx), &host, stmt, enterpriseSpecificID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, common_mysql.NotFound("Android device").WithName(enterpriseSpecificID)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting device by enterprise specific ID")
	}
	result := &fleet.AndroidHost{
		Host: &fleet.Host{
			ID:     host.Device.HostID,
			TeamID: host.TeamID,
		},
		Device: host.Device,
	}
	result.SetNodeKey(enterpriseSpecificID)
	return result, nil
}
