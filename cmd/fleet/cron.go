package main

import (
	"context"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
	"github.com/fleetdm/fleet/v4/server/webhooks"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func cronDB(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, identifier string, license *fleet.LicenseInfo) {
	logger = kitlog.With(logger, "cron", lockKeyLeader)

	ticker := time.NewTicker(10 * time.Second)
	for {
		level.Debug(logger).Log("waiting", "on ticker")
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
			ticker.Reset(1 * time.Hour)
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		}

		if locked, err := ds.Lock(ctx, lockKeyLeader, identifier, 1*time.Hour); err != nil {
			level.Error(logger).Log("msg", "Error acquiring lock", "err", err)
			continue
		} else if !locked {
			level.Debug(logger).Log("msg", "Not the leader. Skipping...")
			continue
		}

		_, err := ds.CleanupDistributedQueryCampaigns(ctx, time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning distributed query campaigns", "details", err)
			sentry.CaptureException(err)
		}
		err = ds.CleanupIncomingHosts(ctx, time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning incoming hosts", "details", err)
			sentry.CaptureException(err)
		}
		_, err = ds.CleanupCarves(ctx, time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleaning carves", "details", err)
			sentry.CaptureException(err)
		}
		err = ds.UpdateQueryAggregatedStats(ctx)
		if err != nil {
			level.Error(logger).Log("err", "aggregating query stats", "details", err)
			sentry.CaptureException(err)
		}
		err = ds.UpdateScheduledQueryAggregatedStats(ctx)
		if err != nil {
			level.Error(logger).Log("err", "aggregating scheduled query stats", "details", err)
			sentry.CaptureException(err)
		}
		err = ds.CleanupExpiredHosts(ctx)
		if err != nil {
			level.Error(logger).Log("err", "cleaning expired hosts", "details", err)
			sentry.CaptureException(err)
		}
		err = ds.GenerateAggregatedMunkiAndMDM(ctx)
		if err != nil {
			level.Error(logger).Log("err", "aggregating munki and mdm data", "details", err)
			sentry.CaptureException(err)
		}
		err = ds.CleanupPolicyMembership(ctx, time.Now())
		if err != nil {
			level.Error(logger).Log("err", "cleanup policy membership", "details", err)
			sentry.CaptureException(err)
		}
		err = ds.UpdateOSVersions(ctx)
		if err != nil {
			level.Error(logger).Log("err", "update os versions", "details", err)
			sentry.CaptureException(err)
		}

		// NOTE(mna): this is not a route from the fleet server (not in server/service/handler.go) so it
		// will not automatically support the /latest/ versioning. Leaving it as /v1/ for that reason.
		err = trySendStatistics(ctx, ds, fleet.StatisticsFrequency, "https://fleetdm.com/api/v1/webhooks/receive-usage-analytics", license)
		if err != nil {
			level.Error(logger).Log("err", "sending statistics", "details", err)
			sentry.CaptureException(err)
		}

		level.Debug(logger).Log("loop", "done")
	}
}

func cronVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	identifier string,
	config config.FleetConfig,
) {
	logger = kitlog.With(logger, "cron", lockKeyVulnerabilities)

	if config.Vulnerabilities.CurrentInstanceChecks == "no" || config.Vulnerabilities.CurrentInstanceChecks == "0" {
		level.Info(logger).Log("vulnerability scanning", "host not configured to check for vulnerabilities")
		return
	}

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		return
	}

	vulnDisabled := false
	if appConfig.VulnerabilitySettings.DatabasesPath == "" &&
		config.Vulnerabilities.DatabasesPath == "" {
		level.Info(logger).Log("vulnerability scanning", "not configured")
		vulnDisabled = true
	}
	if !appConfig.HostSettings.EnableSoftwareInventory {
		level.Info(logger).Log("software inventory", "not configured")
		return
	}

	vulnPath := appConfig.VulnerabilitySettings.DatabasesPath
	if vulnPath == "" {
		vulnPath = config.Vulnerabilities.DatabasesPath
	}
	if config.Vulnerabilities.DatabasesPath != "" && config.Vulnerabilities.DatabasesPath != vulnPath {
		vulnPath = config.Vulnerabilities.DatabasesPath
		level.Info(logger).Log(
			"databases_path", "fleet config takes precedence over app config when both are configured",
			"result", vulnPath)
	}

	if !vulnDisabled {
		level.Info(logger).Log("databases-path", vulnPath)
	}
	level.Info(logger).Log("periodicity", config.Vulnerabilities.Periodicity)

	if !vulnDisabled {
		if config.Vulnerabilities.CurrentInstanceChecks == "auto" {
			level.Debug(logger).Log("current instance checks", "auto", "trying to create databases-path", vulnPath)
			err := os.MkdirAll(vulnPath, 0o755)
			if err != nil {
				level.Error(logger).Log("databases-path", "creation failed, returning", "err", err)
				return
			}
		}
	}

	ticker := time.NewTicker(10 * time.Second)
	for {
		level.Debug(logger).Log("waiting", "on ticker")
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
			ticker.Reset(config.Vulnerabilities.Periodicity)
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		}
		if config.Vulnerabilities.CurrentInstanceChecks == "auto" {
			if locked, err := ds.Lock(ctx, lockKeyVulnerabilities, identifier, 1*time.Hour); err != nil {
				level.Error(logger).Log("msg", "Error acquiring lock", "err", err)
				continue
			} else if !locked {
				level.Debug(logger).Log("msg", "Not the leader. Skipping...")
				continue
			}
		}

		if !vulnDisabled {
			recentVulns := checkVulnerabilities(ctx, ds, logger, vulnPath, config, appConfig.WebhookSettings.VulnerabilitiesWebhook)
			if len(recentVulns) > 0 {
				if err := webhooks.TriggerVulnerabilitiesWebhook(ctx, ds, kitlog.With(logger, "webhook", "vulnerabilities"),
					recentVulns, appConfig, time.Now()); err != nil {

					level.Error(logger).Log("err", "triggering vulnerabilities webhook", "details", err)
					sentry.CaptureException(err)
				}
			}
		}

		if err := ds.CalculateHostsPerSoftware(ctx, time.Now()); err != nil {
			level.Error(logger).Log("msg", "calculating hosts count per software", "err", err)
			sentry.CaptureException(err)
		}

		// It's important vulnerabilities.PostProcess runs after ds.CalculateHostsPerSoftware
		// because it cleans up any software that's not installed on the fleet (e.g. hosts removal,
		// or software being uninstalled on hosts).
		if !vulnDisabled {
			if err := vulnerabilities.PostProcess(ctx, ds, vulnPath, logger, config); err != nil {
				level.Error(logger).Log("msg", "post processing CVEs", "err", err)
				sentry.CaptureException(err)
			}
		}

		level.Debug(logger).Log("loop", "done")
	}
}

func checkVulnerabilities(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger,
	vulnPath string, config config.FleetConfig, vulnWebhookCfg fleet.VulnerabilitiesWebhookSettings) map[string][]string {
	err := vulnerabilities.TranslateSoftwareToCPE(ctx, ds, vulnPath, logger, config)
	if err != nil {
		level.Error(logger).Log("msg", "analyzing vulnerable software: Software->CPE", "err", err)
		sentry.CaptureException(err)
		return nil
	}

	recentVulns, err := vulnerabilities.TranslateCPEToCVE(ctx, ds, vulnPath, logger, config, vulnWebhookCfg.Enable)
	if err != nil {
		level.Error(logger).Log("msg", "analyzing vulnerable software: CPE->CVE", "err", err)
		sentry.CaptureException(err)
		return nil
	}
	return recentVulns
}

func cronWebhooks(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	identifier string,
	failingPoliciesSet fleet.FailingPolicySet,
	intervalReload time.Duration,
) {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		return
	}

	interval := appConfig.WebhookSettings.Interval.ValueOr(24 * time.Hour)
	level.Debug(logger).Log("interval", interval.String())
	ticker := time.NewTicker(interval)
	start := time.Now()
	for {
		level.Debug(logger).Log("waiting", "on ticker")
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		case <-time.After(intervalReload):
			// Reload interval and check if it has been reduced.
			appConfig, err := ds.AppConfig(ctx)
			if err != nil {
				level.Error(logger).Log("config", "couldn't read app config", "err", err)
				continue
			}
			if currInterval := appConfig.WebhookSettings.Interval.ValueOr(24 * time.Hour); time.Since(start) < currInterval {
				continue
			}
		}

		// Reread app config to be able to read latest data used by the webhook
		// and update the ticker for the next run.
		appConfig, err = ds.AppConfig(ctx)
		if err != nil {
			level.Error(logger).Log("config", "couldn't read app config", "err", err)
			sentry.CaptureException(err)
		} else {
			ticker.Reset(appConfig.WebhookSettings.Interval.ValueOr(24 * time.Hour))
			start = time.Now()
		}

		// We set the db lock durations to match the intervalReload.
		maybeTriggerHostStatus(ctx, ds, logger, identifier, appConfig, intervalReload)
		maybeTriggerFailingPoliciesWebhook(ctx, ds, logger, identifier, appConfig, intervalReload, failingPoliciesSet)

		level.Debug(logger).Log("loop", "done")
	}
}

func maybeTriggerHostStatus(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	identifier string,
	appConfig *fleet.AppConfig,
	lockDuration time.Duration,
) {
	logger = kitlog.With(logger, "cron", lockKeyWebhooksHostStatus)

	if locked, err := ds.Lock(ctx, lockKeyWebhooksHostStatus, identifier, lockDuration); err != nil {
		level.Error(logger).Log("msg", "Error acquiring lock", "err", err)
		return
	} else if !locked {
		level.Debug(logger).Log("msg", "Not the leader. Skipping...")
		return
	}

	if err := webhooks.TriggerHostStatusWebhook(
		ctx, ds, kitlog.With(logger, "webhook", "host_status"), appConfig,
	); err != nil {
		level.Error(logger).Log("err", "triggering host status webhook", "details", err)
		sentry.CaptureException(err)
	}
}

func maybeTriggerFailingPoliciesWebhook(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	identifier string,
	appConfig *fleet.AppConfig,
	lockDuration time.Duration,
	failingPoliciesSet fleet.FailingPolicySet,
) {
	logger = kitlog.With(logger, "cron", lockKeyWebhooksFailingPolicies)

	if locked, err := ds.Lock(ctx, lockKeyWebhooksFailingPolicies, identifier, lockDuration); err != nil {
		level.Error(logger).Log("msg", "Error acquiring lock", "err", err)
		return
	} else if !locked {
		level.Debug(logger).Log("msg", "Not the leader. Skipping...")
		return
	}

	if err := webhooks.TriggerFailingPoliciesWebhook(
		ctx, ds, kitlog.With(logger, "webhook", "failing_policies"), appConfig, failingPoliciesSet, time.Now(),
	); err != nil {
		level.Error(logger).Log("err", "triggering failing policies webhook", "details", err)
		sentry.CaptureException(err)
	}
}

func cronWorker(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, identifier string) {
	logger = kitlog.With(logger, "cron", lockKeyWorker)

	w := worker.NewWorker(ds, logger)

	jira := worker.NewJira(ds, logger)
	w.Register(jira)

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
			ticker.Reset(10 * time.Minute)
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		}

		if locked, err := ds.Lock(ctx, lockKeyWorker, identifier, 10*time.Minute); err != nil {
			level.Error(logger).Log("msg", "Error acquiring lock", "err", err)
			continue
		} else if !locked {
			level.Debug(logger).Log("msg", "Not the leader. Skipping...")
			continue
		}

		err := w.ProcessJobs(ctx)
		if err != nil {
			level.Error(logger).Log("msg", "Error processing jobs", "err", err)
		}
	}
}
