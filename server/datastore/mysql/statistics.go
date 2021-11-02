package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/kolide/kit/version"
)

type statistics struct {
	fleet.UpdateCreateTimestamps
	Identifier string `db:"anonymous_identifier"`
}

func (d *Datastore) ShouldSendStatistics(ctx context.Context, frequency time.Duration) (fleet.StatisticsPayload, bool, error) {
	amountEnrolledHosts, err := amountEnrolledHostsDB(d.writer)
	if err != nil {
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "amount enrolled hosts")
	}

	dest := statistics{}
	err = sqlx.GetContext(ctx, d.writer, &dest, `SELECT created_at, updated_at, anonymous_identifier FROM statistics LIMIT 1`)
	if err != nil {
		if err == sql.ErrNoRows {
			anonIdentifier, err := server.GenerateRandomText(64)
			if err != nil {
				return fleet.StatisticsPayload{}, false, err
			}
			_, err = d.writer.ExecContext(ctx, `INSERT INTO statistics(anonymous_identifier) VALUES (?)`, anonIdentifier)
			if err != nil {
				return fleet.StatisticsPayload{}, false, err
			}
			return fleet.StatisticsPayload{
				AnonymousIdentifier: anonIdentifier,
				FleetVersion:        version.Version().Version,
				NumHostsEnrolled:    amountEnrolledHosts,
			}, true, nil
		}
		return fleet.StatisticsPayload{}, false, err
	}
	lastUpdated := dest.UpdatedAt
	if dest.CreatedAt.After(dest.UpdatedAt) {
		lastUpdated = dest.CreatedAt
	}
	if time.Now().Before(lastUpdated.Add(frequency)) {
		return fleet.StatisticsPayload{}, false, nil
	}
	return fleet.StatisticsPayload{
		AnonymousIdentifier: dest.Identifier,
		FleetVersion:        version.Version().Version,
		NumHostsEnrolled:    amountEnrolledHosts,
	}, true, nil
}

func (d *Datastore) RecordStatisticsSent(ctx context.Context) error {
	_, err := d.writer.ExecContext(ctx, `UPDATE statistics SET updated_at = CURRENT_TIMESTAMP LIMIT 1`)
	return err
}
