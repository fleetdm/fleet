package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
	"github.com/jmoiron/sqlx"
	"github.com/kolide/kit/version"
)

type statistics struct {
	fleet.UpdateCreateTimestamps
	Identifier string `db:"anonymous_identifier"`
}

func (ds *Datastore) ShouldSendStatistics(ctx context.Context, frequency time.Duration, config config.FleetConfig, license *fleet.LicenseInfo) (fleet.StatisticsPayload, bool, error) {
	computeStats := func(stats *fleet.StatisticsPayload, since time.Time) error {
		enrolledHostsByOS, amountEnrolledHosts, err := amountEnrolledHostsByOSDB(ctx, ds.writer)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount enrolled hosts by os")
		}
		amountUsers, err := amountUsersDB(ctx, ds.writer)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount users")
		}
		amountTeams, err := amountTeamsDB(ctx, ds.writer)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount teams")
		}
		amountPolicies, err := amountPoliciesDB(ctx, ds.writer)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount policies")
		}
		amountLabels, err := amountLabelsDB(ctx, ds.writer)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount labels")
		}
		appConfig, err := ds.AppConfig(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "statistics app config")
		}
		amountWeeklyUsers, err := amountActiveUsersSinceDB(ctx, ds.writer, since)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount active users")
		}
		amountPolicyViolationDaysActual, amountPolicyViolationDaysPossible, err := amountPolicyViolationDaysDB(ctx, ds.writer)
		if err == sql.ErrNoRows {
			level.Debug(ds.logger).Log("msg", "amount policy violation days", "err", err)
		} else if err != nil {
			return ctxerr.Wrap(ctx, err, "amount policy violation days")
		}
		storedErrs, err := ctxerr.Aggregate(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "statistics error store")
		}
		amountHostsNotResponding, err := countHostsNotRespondingDB(ctx, ds.writer, ds.logger, config)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount hosts not responding")
		}

		stats.NumHostsEnrolled = amountEnrolledHosts
		stats.NumUsers = amountUsers
		stats.NumTeams = amountTeams
		stats.NumPolicies = amountPolicies
		stats.NumLabels = amountLabels
		stats.SoftwareInventoryEnabled = appConfig.Features.EnableSoftwareInventory
		stats.VulnDetectionEnabled = appConfig.VulnerabilitySettings.DatabasesPath != ""
		stats.SystemUsersEnabled = appConfig.Features.EnableHostUsers
		stats.HostsStatusWebHookEnabled = appConfig.WebhookSettings.HostStatusWebhook.Enable
		stats.NumWeeklyActiveUsers = amountWeeklyUsers
		stats.NumWeeklyPolicyViolationDaysActual = amountPolicyViolationDaysActual
		stats.NumWeeklyPolicyViolationDaysPossible = amountPolicyViolationDaysPossible
		stats.HostsEnrolledByOperatingSystem = enrolledHostsByOS
		stats.StoredErrors = storedErrs
		stats.NumHostsNotResponding = amountHostsNotResponding
		stats.Organization = "unknown"
		if license.IsPremium() {
			stats.Organization = license.Organization
		}
		return nil
	}

	dest := statistics{}
	err := sqlx.GetContext(ctx, ds.writer, &dest, `SELECT created_at, updated_at, anonymous_identifier FROM statistics LIMIT 1`)
	if err != nil {
		if err == sql.ErrNoRows {
			anonIdentifier, err := server.GenerateRandomText(64)
			if err != nil {
				return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "generate random text")
			}
			_, err = ds.writer.ExecContext(ctx, `INSERT INTO statistics(anonymous_identifier) VALUES (?)`, anonIdentifier)
			if err != nil {
				return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "insert statistics")
			}

			// compute active weekly users since now - frequency
			stats := fleet.StatisticsPayload{
				AnonymousIdentifier: anonIdentifier,
				FleetVersion:        version.Version().Version,
				LicenseTier:         license.Tier,
			}
			if err := computeStats(&stats, time.Now().Add(-frequency)); err != nil {
				return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "compute statistics")
			}

			return stats, true, nil
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

	stats := fleet.StatisticsPayload{
		AnonymousIdentifier: dest.Identifier,
		FleetVersion:        version.Version().Version,
		LicenseTier:         license.Tier,
	}
	if err := computeStats(&stats, lastUpdated); err != nil {
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "compute statistics")
	}

	return stats, true, nil
}

func (ds *Datastore) RecordStatisticsSent(ctx context.Context) error {
	_, err := ds.writer.ExecContext(ctx, `UPDATE statistics SET updated_at = CURRENT_TIMESTAMP LIMIT 1`)
	return ctxerr.Wrap(ctx, err, "update statistics")
}

func (ds *Datastore) CleanupStatistics(ctx context.Context) error {
	// reset weekly count of policy violation days
	if err := ds.InitializePolicyViolationDays(ctx); err != nil {
		return err
	}
	return nil
}
