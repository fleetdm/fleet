package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewAndroidHost(ctx context.Context, host *fleet.AndroidHost) (*fleet.AndroidHost, error) {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		sqlStatement := `
		INSERT INTO hosts (
			detail_updated_at,
			label_updated_at,
			policy_updated_at,
			computer_name,
			platform,
			os_version,
		    build,
			memory,
			team_id,
			hardware_serial,
		)
		VALUES (
			:detail_updated_at,
			:label_updated_at,
			:policy_updated_at,
			:computer_name,
			:platform,
			:os_version,
			:build,
			:memory,
			:team_id,
			:hardware_serial
		)
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
