package mysql

import (
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/kolide/kit/version"
)

type statistics struct {
	fleet.UpdateCreateTimestamps
	Identifier string `db:"anonymous_identifier"`
}

func (d *Datastore) ShouldSendStatistics(frequency time.Duration) (fleet.StatisticsPayload, bool, error) {
	amountEnrolledHosts, err := d.amountEnrolledHosts()
	if err != nil {
		return fleet.StatisticsPayload{}, false, err
	}

	dest := statistics{}
	err = d.db.Get(&dest, `SELECT created_at, updated_at, anonymous_identifier FROM statistics LIMIT 1`)
	if err != nil {
		if err == sql.ErrNoRows {
			anonIdentifier, err := server.GenerateRandomText(64)
			if err != nil {
				return fleet.StatisticsPayload{}, false, err
			}
			_, err = d.db.Exec(`INSERT INTO statistics(anonymous_identifier) VALUES (?)`, anonIdentifier)
			if err != nil {
				return fleet.StatisticsPayload{}, false, err
			}
			return fleet.StatisticsPayload{
				AnonymousIdentifier: anonIdentifier,
				FleetVersion:        version.Version().Version,
				NumHostsEnrolled:    amountEnrolledHosts,
			}, true, nil
		} else {
			return fleet.StatisticsPayload{}, false, err
		}
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

func (d *Datastore) RecordStatisticsSent() error {
	_, err := d.db.Exec(`UPDATE statistics SET updated_at = CURRENT_TIMESTAMP LIMIT 1`)
	return err
}
