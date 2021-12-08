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

func (d *Datastore) ShouldSendStatistics(ctx context.Context, frequency time.Duration, license *fleet.LicenseInfo) (fleet.StatisticsPayload, bool, error) {
	amountEnrolledHosts, err := amountEnrolledHostsDB(d.writer)
	if err != nil {
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "amount enrolled hosts")
	}
	amountUsers, err := amountUsersDB(d.writer)
	if err != nil {
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "amount users")
	}
	amountTeams, err := amountTeamsDB(d.writer)
	if err != nil {
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "amount teams")
	}
	amountPolicies, err := amountPoliciesDB(d.writer)
	if err != nil {
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "amount teams")
	}
	appConfig, err := d.AppConfig(ctx)
	if err != nil {
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "statistics app config")
	}

	dest := statistics{}
	err = sqlx.GetContext(ctx, d.writer, &dest, `SELECT created_at, updated_at, anonymous_identifier FROM statistics LIMIT 1`)
	if err != nil {
		if err == sql.ErrNoRows {
			anonIdentifier, err := server.GenerateRandomText(64)
			if err != nil {
				return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "generate random text")
			}
			_, err = d.writer.ExecContext(ctx, `INSERT INTO statistics(anonymous_identifier) VALUES (?)`, anonIdentifier)
			if err != nil {
				return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "insert statistics")
			}
			return fleet.StatisticsPayload{
				AnonymousIdentifier:       anonIdentifier,
				FleetVersion:              version.Version().Version,
				LicenseTier:               license.Tier,
				NumHostsEnrolled:          amountEnrolledHosts,
				NumUsers:                  amountUsers,
				NumTeams:                  amountTeams,
				NumPolicies:               amountPolicies,
				SoftwareInventoryEnabled:  appConfig.HostSettings.EnableSoftwareInventory,
				VulnDetectionEnabled:      appConfig.VulnerabilitySettings.DatabasesPath != "",
				SystemUsersEnabled:        appConfig.HostSettings.EnableHostUsers,
				HostsStatusWebHookEnabled: appConfig.WebhookSettings.HostStatusWebhook.Enable,
			}, true, nil
		}
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "get statistics")
	}
	lastUpdated := dest.UpdatedAt
	if dest.CreatedAt.After(dest.UpdatedAt) {
		lastUpdated = dest.CreatedAt
	}
	if time.Now().Before(lastUpdated.Add(frequency)) {
		return fleet.StatisticsPayload{}, false, nil
	}
	return fleet.StatisticsPayload{
		AnonymousIdentifier:       dest.Identifier,
		FleetVersion:              version.Version().Version,
		LicenseTier:               license.Tier,
		NumHostsEnrolled:          amountEnrolledHosts,
		NumUsers:                  amountUsers,
		NumTeams:                  amountTeams,
		NumPolicies:               amountPolicies,
		SoftwareInventoryEnabled:  appConfig.HostSettings.EnableSoftwareInventory,
		VulnDetectionEnabled:      &appConfig.VulnerabilitySettings.DatabasesPath != nil,
		SystemUsersEnabled:        appConfig.HostSettings.EnableHostUsers,
		HostsStatusWebHookEnabled: appConfig.WebhookSettings.HostStatusWebhook.Enable,
	}, true, nil
}

func (d *Datastore) RecordStatisticsSent(ctx context.Context) error {
	_, err := d.writer.ExecContext(ctx, `UPDATE statistics SET updated_at = CURRENT_TIMESTAMP LIMIT 1`)
	return ctxerr.Wrap(ctx, err, "update statistics")
}
