package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
	"github.com/fleetdm/fleet/v4/server/webhooks"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func startWebhooksSchedule(
	ctx context.Context, instanceID string, ds fleet.Datastore, appConfig *fleet.AppConfig, failingPoliciesSet fleet.FailingPolicySet, logger kitlog.Logger,
) {
	schedule.New(
		ctx, "webhooks", instanceID, appConfig.WebhookSettings.Interval.ValueOr(24*time.Hour), ds,
		schedule.WithLogger(kitlog.With(logger, "cron", "webhooks")),
		schedule.WithConfigReloadInterval(5*time.Minute, func(ctx context.Context) (time.Duration, error) {
			appConfig, err := ds.AppConfig(ctx)
			if err != nil {
				return 0, err
			}
			newInterval := appConfig.WebhookSettings.Interval.ValueOr(24 * time.Hour)
			return newInterval, nil
		}),
		schedule.WithJob(
			"maybe_trigger_host_status",
			func(ctx context.Context) error {
				return webhooks.TriggerHostStatusWebhook(
					ctx, ds, kitlog.With(logger, "webhook", "host_status"), appConfig,
				)
			},
		),
		schedule.WithJob(
			"maybe_trigger_failing_policies",
			func(ctx context.Context) error {
				return webhooks.TriggerFailingPoliciesWebhook(
					ctx, ds, kitlog.With(logger, "webhook", "failing_policies"), appConfig, failingPoliciesSet, time.Now(),
				)
			},
		),
	).Start()
}

func startCleanupsAndAggregationSchedule(
	ctx context.Context, instanceID string, ds fleet.Datastore, logger kitlog.Logger,
) {
	schedule.New(
		ctx, "cleanups_then_aggregation", instanceID, 1*time.Hour, ds,
		// Using leader for the lock to be backwards compatilibity with old deployments.
		schedule.WithAltLockID("leader"),
		schedule.WithLogger(kitlog.With(logger, "cron", "cleanups_then_aggregation")),
		// Run cleanup jobs first.
		schedule.WithJob(
			"distributed_query_campaings",
			func(ctx context.Context) error {
				_, err := ds.CleanupDistributedQueryCampaigns(ctx, time.Now())
				return err
			},
		),
		schedule.WithJob(
			"incoming_hosts",
			func(ctx context.Context) error {
				return ds.CleanupIncomingHosts(ctx, time.Now())
			},
		),
		schedule.WithJob(
			"carves",
			func(ctx context.Context) error {
				_, err := ds.CleanupCarves(ctx, time.Now())
				return err
			},
		),
		schedule.WithJob(
			"expired_hosts",
			func(ctx context.Context) error {
				return ds.CleanupExpiredHosts(ctx)
			},
		),
		schedule.WithJob(
			"policy_membership",
			func(ctx context.Context) error {
				return ds.CleanupPolicyMembership(ctx, time.Now())
			},
		),
		// Run aggregation jobs after cleanups.
		schedule.WithJob(
			"query_aggregated_stats",
			func(ctx context.Context) error {
				return ds.UpdateQueryAggregatedStats(ctx)
			},
		),
		schedule.WithJob(
			"scheduled_query_aggregated_stats",
			func(ctx context.Context) error {
				return ds.UpdateScheduledQueryAggregatedStats(ctx)
			},
		),
		schedule.WithJob(
			"aggregated_munki_and_mdm",
			func(ctx context.Context) error {
				return ds.GenerateAggregatedMunkiAndMDM(ctx)
			},
		),
	).Start()
}

func startSendStatsSchedule(ctx context.Context, instanceID string, ds fleet.Datastore, license *fleet.LicenseInfo, logger kitlog.Logger) {
	schedule.New(
		ctx, "stats", instanceID, 1*time.Hour, ds,
		schedule.WithLogger(kitlog.With(logger, "cron", "stats")),
		schedule.WithJob(
			"try_send_statistics",
			func(ctx context.Context) error {
				return trySendStatistics(ctx, ds, fleet.StatisticsFrequency, "https://fleetdm.com/api/v1/webhooks/receive-usage-analytics", license)
			},
		),
	).Start()
}

func startVulnerabilitiesSchedule(
	ctx context.Context, instanceID string, ds fleet.Datastore, config config.FleetConfig, logger kitlog.Logger,
) *schedule.Schedule {
	vulnerabilitiesLogger := kitlog.With(logger, "cron", "vulnerabilities")
	s := schedule.New(
		ctx, "vulnerabilities", instanceID, config.Vulnerabilities.Periodicity, ds,
		schedule.WithLogger(vulnerabilitiesLogger),
		schedule.WithJob(
			"cron_vulnerabilities",
			func(ctx context.Context) error {
				// TODO(lucas): Decouple cronVulnerabilities into multiple jobs.
				return cronVulnerabilities(ctx, ds, vulnerabilitiesLogger, config)
			},
		),
	)
	s.Start()
	return s
}

func trySendStatistics(ctx context.Context, ds fleet.Datastore, frequency time.Duration, url string, license *fleet.LicenseInfo) error {
	ac, err := ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.ServerSettings.EnableAnalytics {
		return nil
	}

	stats, shouldSend, err := ds.ShouldSendStatistics(ctx, frequency, license)
	if err != nil {
		return err
	}
	if !shouldSend {
		return nil
	}

	err = server.PostJSONWithTimeout(ctx, url, stats)
	if err != nil {
		return err
	}

	return ds.RecordStatisticsSent(ctx)
}

func cronVulnerabilities(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, config config.FleetConfig) error {
	if config.Vulnerabilities.CurrentInstanceChecks == "no" || config.Vulnerabilities.CurrentInstanceChecks == "0" {
		level.Info(logger).Log("vulnerability scanning", "host not configured to check for vulnerabilities")
		return nil
	}

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		return err
	}

	vulnDisabled := false
	if appConfig.VulnerabilitySettings.DatabasesPath == "" &&
		config.Vulnerabilities.DatabasesPath == "" {
		level.Info(logger).Log("vulnerability scanning", "not configured")
		vulnDisabled = true
	}
	if !appConfig.HostSettings.EnableSoftwareInventory {
		level.Info(logger).Log("software inventory", "not configured")
		return nil
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
				sentry.CaptureException(err)
				return err
			}
		}
	}

	if !vulnDisabled {
		automation := checkAutomationToRun(appConfig, logger)
		level.Debug(logger).Log(
			"vulnerabilities", "checking for recent vulnerabilities",
			"vuln-path", vulnPath,
			"vuln-automation", automation,
		)

		recentVulns := checkVulnerabilities(ctx, ds, logger, vulnPath, config, automation != automation_none)
		if len(recentVulns) > 0 {
			switch automation {
			case automation_none:
				level.Debug(logger).Log("msg", "no vuln automations enabled")
			case automation_webhook:
				if err := webhooks.TriggerVulnerabilitiesWebhook(ctx, ds, kitlog.With(logger, "webhook", "vulnerabilities"),
					recentVulns, appConfig, time.Now()); err != nil {

					level.Error(logger).Log("err", "triggering vulnerabilities webhook", "details", err)
					sentry.CaptureException(err)
				}
			case automation_jira:
				if err := worker.QueueJiraJobs(
					ctx,
					ds,
					kitlog.With(logger, "jira", "vulnerabilities"),
					recentVulns,
				); err != nil {
					level.Error(logger).Log("err", "queueing vulnerabilities to jira", "details", err)
					sentry.CaptureException(err)
				}
			case automation_zendesk:
				if err := worker.QueueZendeskJobs(
					ctx,
					ds,
					kitlog.With(logger, "zendesk", "vulnerabilities"),
					recentVulns,
				); err != nil {
					level.Error(logger).Log("err", "queueing vulnerabilities to zendesk", "details", err)
					sentry.CaptureException(err)
				}
			default:
				err := fmt.Errorf("unknown automation: %d", automation)
				level.Error(logger).Log("err", err)
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

	level.Debug(logger).Log("vulnerabilities", "done")
	return nil
}

//go:generate go run golang.org/x/tools/cmd/stringer -type=automation
type automation int

const (
	automation_none automation = iota
	automation_webhook
	automation_jira
	automation_zendesk
)

// checkAutomationToRun returns the vulnerability automation to run.
//
// This method assumes only one vulnerability automation (i.e. webhook or integration)
// can be enabled at a time, enforced when updating the AppConfig.
func checkAutomationToRun(appConfig *fleet.AppConfig, logger kitlog.Logger) automation {
	autom := automation_none
	if appConfig.WebhookSettings.VulnerabilitiesWebhook.Enable {
		autom = automation_webhook
	}
	for _, j := range appConfig.Integrations.Jira {
		if j.EnableSoftwareVulnerabilities {
			if autom != automation_none {
				err := errors.New("more than one automation enabled: jira check")
				level.Error(logger).Log("err", err)
				sentry.CaptureException(err)
			}
			autom = automation_jira
			break
		}
	}
	for _, z := range appConfig.Integrations.Zendesk {
		if z.EnableSoftwareVulnerabilities {
			if autom != automation_none {
				err := errors.New("more than one automation enabled: zendesk check")
				level.Error(logger).Log("err", err)
				sentry.CaptureException(err)
			}
			autom = automation_zendesk
			break
		}
	}
	return autom
}

func checkVulnerabilities(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger,
	vulnPath string, config config.FleetConfig, vulnEnabled bool,
) map[string][]string {
	err := vulnerabilities.TranslateSoftwareToCPE(ctx, ds, vulnPath, logger, config)
	if err != nil {
		level.Error(logger).Log("msg", "analyzing vulnerable software: Software->CPE", "err", err)
		sentry.CaptureException(err)
		return nil
	}

	recentVulns, err := vulnerabilities.TranslateCPEToCVE(ctx, ds, vulnPath, logger, config, vulnEnabled)
	if err != nil {
		level.Error(logger).Log("msg", "analyzing vulnerable software: CPE->CVE", "err", err)
		sentry.CaptureException(err)
		return nil
	}
	return recentVulns
}

func cronIntegrations(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	identifier string,
) {
	const (
		lockDuration        = 10 * time.Minute
		lockAttemptInterval = 10 * time.Minute
		lockKeyWorker       = "worker"
	)

	logger = kitlog.With(logger, "cron", lockKeyWorker)

	// create the worker and register the Jira and Zendesk jobs even if no
	// integration is enabled, as that config can change live (and if it's not
	// there won't be any records to process so it will mostly just sleep).
	w := worker.NewWorker(ds, logger)
	jira := &worker.Jira{
		Datastore: ds,
		Log:       logger,
	}
	zendesk := &worker.Zendesk{
		Datastore: ds,
		Log:       logger,
	}
	// leave the url and client fields empty for now, will be filled
	// when the lock is acquired with the up-to-date config.
	w.Register(jira)
	w.Register(zendesk)

	// create client wrappers to introduce forced failures if configured
	// to do so via the environment variable.
	// format is "<modulo number>;<cve1>,<cve2>,<cve3>,..."
	jiraFailerClient := newFailerClient(os.Getenv("FLEET_JIRA_CLIENT_FORCED_FAILURES"))
	zendeskFailerClient := newFailerClient(os.Getenv("FLEET_ZENDESK_CLIENT_FORCED_FAILURES"))

	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			level.Debug(logger).Log("waiting", "done")
			ticker.Reset(lockAttemptInterval)
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		}

		if locked, err := ds.Lock(ctx, lockKeyWorker, identifier, lockDuration); err != nil {
			level.Error(logger).Log("msg", "Error acquiring lock", "err", err)
			continue
		} else if !locked {
			level.Debug(logger).Log("msg", "Not the leader. Skipping...")
			continue
		}

		// Read app config to be able to use the latest configuration for the Jira
		// integration.
		appConfig, err := ds.AppConfig(ctx)
		if err != nil {
			level.Error(logger).Log("config", "couldn't read app config", "err", err)
			sentry.CaptureException(err)
			continue
		}

		// get the enabled jira config, if any
		var jiraSettings *fleet.JiraIntegration
		for _, intg := range appConfig.Integrations.Jira {
			if intg.EnableSoftwareVulnerabilities {
				jiraSettings = intg
				break
			}
		}

		// get the enabled zendesk config, if any
		var zendeskSettings *fleet.ZendeskIntegration
		for _, intg := range appConfig.Integrations.Zendesk {
			if intg.EnableSoftwareVulnerabilities {
				zendeskSettings = intg
				break
			}
		}

		if jiraSettings != nil && zendeskSettings != nil {
			// skip processing jobs if more than one integration is enabled.
			level.Error(logger).Log("err", "more than one automation enabled")
			continue
		}

		if jiraSettings == nil && zendeskSettings == nil {
			// skip processing jobs if no integrations are enabled.
			level.Debug(logger).Log("msg", "no automations enabled")
			continue
		}

		if jiraSettings != nil {
			// create the client to make API calls to Jira
			err := setJiraClient(jira, jiraSettings, appConfig, logger, jiraFailerClient)
			if err != nil {
				level.Error(logger).Log("msg", "Error creating JIRA client", "err", err)
				sentry.CaptureException(err)
				continue
			}
		}

		if zendeskSettings != nil {
			// create the client to make API calls to Zendesk
			err := setZendeskClient(zendesk, zendeskSettings, appConfig, logger, zendeskFailerClient)
			if err != nil {
				level.Error(logger).Log("msg", "Error creating Zendesk client", "err", err)
				sentry.CaptureException(err)
				continue
			}
		}

		workCtx, cancel := context.WithTimeout(ctx, lockDuration)
		if err := w.ProcessJobs(workCtx); err != nil {
			level.Error(logger).Log("msg", "Error processing jobs", "err", err)
			sentry.CaptureException(err)
		}
		cancel() // don't use defer inside loop
	}
}

func setJiraClient(jira *worker.Jira, jiraSettings *fleet.JiraIntegration, appConfig *fleet.AppConfig, logger kitlog.Logger, failerClient *worker.TestAutomationFailer) error {
	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{
		BaseURL:           jiraSettings.URL,
		BasicAuthUsername: jiraSettings.Username,
		BasicAuthPassword: jiraSettings.APIToken,
		ProjectKey:        jiraSettings.ProjectKey,
	})
	if err != nil {
		level.Error(logger).Log("msg", "Error creating Jira client", "err", err)
		sentry.CaptureException(err)
		return err
	}

	// safe to update the Jira worker as it is not used concurrently
	jira.FleetURL = appConfig.ServerSettings.ServerURL
	if failerClient != nil && strings.Contains(jira.FleetURL, "fleetdm") {
		failerClient.JiraClient = client
		jira.JiraClient = failerClient
	} else {
		jira.JiraClient = client
	}

	return nil
}

func setZendeskClient(zendesk *worker.Zendesk, zendeskSettings *fleet.ZendeskIntegration, appConfig *fleet.AppConfig, logger kitlog.Logger, failerClient *worker.TestAutomationFailer) error {
	client, err := externalsvc.NewZendeskClient(&externalsvc.ZendeskOptions{
		URL:      zendeskSettings.URL,
		Email:    zendeskSettings.Email,
		APIToken: zendeskSettings.APIToken,
		GroupID:  zendeskSettings.GroupID,
	})
	if err != nil {
		level.Error(logger).Log("msg", "Error creating Zendesk client", "err", err)
		sentry.CaptureException(err)
		return err
	}

	// safe to update the worker as it is not used concurrently
	zendesk.FleetURL = appConfig.ServerSettings.ServerURL
	if failerClient != nil && strings.Contains(zendesk.FleetURL, "fleetdm") {
		failerClient.ZendeskClient = client
		zendesk.ZendeskClient = failerClient
	} else {
		zendesk.ZendeskClient = client
	}

	return nil
}

func newFailerClient(forcedFailures string) *worker.TestAutomationFailer {
	var failerClient *worker.TestAutomationFailer
	if forcedFailures != "" {

		parts := strings.Split(forcedFailures, ";")
		if len(parts) == 2 {
			mod, _ := strconv.Atoi(parts[0])
			cves := strings.Split(parts[1], ",")
			if mod > 0 || len(cves) > 0 {
				failerClient = &worker.TestAutomationFailer{
					FailCallCountModulo: mod,
					AlwaysFailCVEs:      cves,
				}
			}
		}
	}
	return failerClient
}
