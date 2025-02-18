package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/version"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

type statistics struct {
	fleet.UpdateCreateTimestamps
	Identifier string `db:"anonymous_identifier"`
}

func (ds *Datastore) ShouldSendStatistics(ctx context.Context, frequency time.Duration, config config.FleetConfig) (fleet.StatisticsPayload, bool, error) {
	lic, _ := license.FromContext(ctx)

	computeStats := func(stats *fleet.StatisticsPayload, since time.Time) error {
		enrolledHostsByOS, amountEnrolledHosts, err := amountEnrolledHostsByOSDB(ctx, ds.reader(ctx))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount enrolled hosts by os")
		}
		amountUsers, err := tableRowsCount(ctx, ds.reader(ctx), "users")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount users")
		}
		amountSoftwaresVersions, err := tableRowsCount(ctx, ds.reader(ctx), "software")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount software")
		}
		amountHostSoftwares, err := tableRowsCount(ctx, ds.reader(ctx), "host_software")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount host_software")
		}
		amountSoftwareTitles, err := tableRowsCount(ctx, ds.reader(ctx), "software_titles")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount software_titles")
		}
		amountHostSoftwareInstalledPaths, err := tableRowsCount(ctx, ds.reader(ctx), "host_software_installed_paths")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount host_software_installed_paths")
		}
		amountSoftwareCpes, err := tableRowsCount(ctx, ds.reader(ctx), "software_cpe")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount software_cpe")
		}
		amountSoftwareCves, err := tableRowsCount(ctx, ds.reader(ctx), "software_cve")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount software_cve")
		}
		amountTeams, err := amountTeamsDB(ctx, ds.reader(ctx))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount teams")
		}
		amountPolicies, err := amountPoliciesDB(ctx, ds.reader(ctx))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount policies")
		}
		amountLabels, err := amountLabelsDB(ctx, ds.reader(ctx))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount labels")
		}
		appConfig, err := ds.AppConfig(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "statistics app config")
		}
		amountWeeklyUsers, err := amountActiveUsersSinceDB(ctx, ds.reader(ctx), since)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount active users")
		}
		amountPolicyViolationDaysActual, amountPolicyViolationDaysPossible, err := amountPolicyViolationDaysDB(ctx, ds.reader(ctx))
		if err == sql.ErrNoRows {
			level.Debug(ds.logger).Log("msg", "amount policy violation days", "err", err) //nolint:errcheck
		} else if err != nil {
			return ctxerr.Wrap(ctx, err, "amount policy violation days")
		}
		storedErrs, err := ctxerr.Aggregate(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "statistics error store")
		}
		amountHostsNotResponding, err := countHostsNotRespondingDB(ctx, ds.reader(ctx), ds.logger, config)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount hosts not responding")
		}
		amountHostsByOrbitVersion, err := amountHostsByOrbitVersionDB(ctx, ds.reader(ctx))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount hosts by orbit version")
		}
		amountHostsByOsqueryVersion, err := amountHostsByOsqueryVersionDB(ctx, ds.reader(ctx))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "amount hosts by osquery version")
		}
		numHostsFleetDesktopEnabled, err := numHostsFleetDesktopEnabledDB(ctx, ds.reader(ctx))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "number of hosts with Fleet desktop installed")
		}
		numQueries, err := numSavedQueriesDB(ctx, ds.reader(ctx))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "number of saved queries in DB")
		}

		stats.NumHostsEnrolled = amountEnrolledHosts
		stats.NumUsers = amountUsers
		stats.NumSoftwareVersions = amountSoftwaresVersions
		stats.NumHostSoftwares = amountHostSoftwares
		stats.NumSoftwareTitles = amountSoftwareTitles
		stats.NumHostSoftwareInstalledPaths = amountHostSoftwareInstalledPaths
		stats.NumSoftwareCPEs = amountSoftwareCpes
		stats.NumSoftwareCVEs = amountSoftwareCves
		stats.NumTeams = amountTeams
		stats.NumPolicies = amountPolicies
		stats.NumLabels = amountLabels
		stats.SoftwareInventoryEnabled = appConfig.Features.EnableSoftwareInventory
		stats.VulnDetectionEnabled = config.Vulnerabilities.DatabasesPath != "" || appConfig.VulnerabilitySettings.DatabasesPath != ""
		stats.SystemUsersEnabled = appConfig.Features.EnableHostUsers
		stats.HostsStatusWebHookEnabled = appConfig.WebhookSettings.HostStatusWebhook.Enable
		stats.MDMMacOsEnabled = appConfig.MDM.EnabledAndConfigured
		stats.HostExpiryEnabled = appConfig.HostExpirySettings.HostExpiryEnabled
		stats.MDMWindowsEnabled = appConfig.MDM.WindowsEnabledAndConfigured
		stats.LiveQueryDisabled = appConfig.ServerSettings.LiveQueryDisabled
		stats.NumWeeklyActiveUsers = amountWeeklyUsers
		stats.NumWeeklyPolicyViolationDaysActual = amountPolicyViolationDaysActual
		stats.NumWeeklyPolicyViolationDaysPossible = amountPolicyViolationDaysPossible
		stats.HostsEnrolledByOperatingSystem = enrolledHostsByOS
		stats.HostsEnrolledByOrbitVersion = amountHostsByOrbitVersion
		stats.HostsEnrolledByOsqueryVersion = amountHostsByOsqueryVersion
		stats.StoredErrors = storedErrs
		stats.NumHostsNotResponding = amountHostsNotResponding
		stats.Organization = "unknown"
		if lic != nil && lic.IsPremium() {
			stats.Organization = lic.Organization
		}
		stats.AIFeaturesDisabled = appConfig.ServerSettings.AIFeaturesDisabled
		stats.MaintenanceWindowsConfigured = len(appConfig.Integrations.GoogleCalendar) > 0 && appConfig.Integrations.GoogleCalendar[0].Domain != "" && len(appConfig.Integrations.GoogleCalendar[0].ApiKey) > 0

		stats.MaintenanceWindowsEnabled = false
		teams, err := ds.ListTeams(ctx, fleet.TeamFilter{User: &fleet.User{
			GlobalRole: ptr.String(fleet.RoleAdmin),
		}}, fleet.ListOptions{})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list teams")
		}
		for _, team := range teams {
			if team.Config.Integrations.GoogleCalendar != nil && team.Config.Integrations.GoogleCalendar.Enable {
				stats.MaintenanceWindowsEnabled = true
				break
			}
		}
		stats.NumHostsFleetDesktopEnabled = numHostsFleetDesktopEnabled
		stats.NumQueries = numQueries
		return nil
	}

	dest := statistics{}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, `SELECT created_at, updated_at, anonymous_identifier FROM statistics LIMIT 1`)
	if err != nil {
		if err == sql.ErrNoRows {
			anonIdentifier, err := server.GenerateRandomText(64)
			if err != nil {
				return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "generate random text")
			}
			_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO statistics(anonymous_identifier) VALUES (?)`, anonIdentifier)
			if err != nil {
				return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "insert statistics")
			}

			// compute active weekly users since now - frequency
			stats := fleet.StatisticsPayload{
				AnonymousIdentifier: anonIdentifier,
				FleetVersion:        version.Version().Version,
				LicenseTier:         fleet.TierFree,
			}
			if lic != nil {
				stats.LicenseTier = lic.Tier
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
		LicenseTier:         fleet.TierFree,
	}
	if lic != nil {
		stats.LicenseTier = lic.Tier
	}
	if err := computeStats(&stats, lastUpdated); err != nil {
		return fleet.StatisticsPayload{}, false, ctxerr.Wrap(ctx, err, "compute statistics")
	}

	return stats, true, nil
}

func (ds *Datastore) RecordStatisticsSent(ctx context.Context) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE statistics SET updated_at = CURRENT_TIMESTAMP LIMIT 1`)
	return ctxerr.Wrap(ctx, err, "update statistics")
}

func (ds *Datastore) CleanupStatistics(ctx context.Context) error {
	// reset weekly count of policy violation days
	if err := ds.InitializePolicyViolationDays(ctx); err != nil {
		return err
	}
	return nil
}
