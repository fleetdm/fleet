package main

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/policies"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
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
				sentry.CaptureException(err)
				continue
			} else if !locked {
				level.Debug(logger).Log("msg", "Not the leader. Skipping...")
				continue
			}
		}

		if !vulnDisabled {
			// refresh app config to check if webhook or any jira integration is
			// enabled, as this can be changed dynamically.
			if freshAppConfig, err := ds.AppConfig(ctx); err != nil {
				level.Error(logger).Log("config", "couldn't refresh app config", "err", err)
				sentry.CaptureException(err)
				// continue with stale app config
			} else {
				appConfig = freshAppConfig
			}

			var vulnAutomationEnabled string

			// only one vuln automation (i.e. webhook or integration) can be enabled at a
			// time, enforced when updating the appconfig.
			if appConfig.WebhookSettings.VulnerabilitiesWebhook.Enable {
				vulnAutomationEnabled = "webhook"
			}
			// check for jira integrations
			for _, j := range appConfig.Integrations.Jira {
				if j.EnableSoftwareVulnerabilities {
					if vulnAutomationEnabled != "" {
						err := errors.New("more than one automation enabled: jira check")
						level.Error(logger).Log("err", err)
						sentry.CaptureException(err)
					}
					vulnAutomationEnabled = "jira"
					break
				}
			}
			// check for zendesk integrations
			for _, z := range appConfig.Integrations.Zendesk {
				if z.EnableSoftwareVulnerabilities {
					if vulnAutomationEnabled != "" {
						err := errors.New("more than one automation enabled: zendesk check")
						level.Error(logger).Log("err", err)
						sentry.CaptureException(err)
					}
					vulnAutomationEnabled = "zendesk"
					break
				}
			}
			level.Debug(logger).Log("vulnAutomationEnabled", vulnAutomationEnabled)

			recentVulns := checkVulnerabilities(ctx, ds, logger, vulnPath, config, (vulnAutomationEnabled != ""))
			if len(recentVulns) > 0 {
				switch vulnAutomationEnabled {
				case "webhook":
					// send recent vulnerabilities via webhook
					if err := webhooks.TriggerVulnerabilitiesWebhook(ctx, ds, kitlog.With(logger, "webhook", "vulnerabilities"),
						recentVulns, appConfig, time.Now()); err != nil {

						level.Error(logger).Log("err", "triggering vulnerabilities webhook", "details", err)
						sentry.CaptureException(err)
					}

				case "jira":
					// queue job to create jira issues
					if err := worker.QueueJiraVulnJobs(
						ctx,
						ds,
						kitlog.With(logger, "jira", "vulnerabilities"),
						recentVulns,
					); err != nil {
						level.Error(logger).Log("err", "queueing vulnerabilities to jira", "details", err)
						sentry.CaptureException(err)
					}

				case "zendesk":
					// queue job to create zendesk ticket
					if err := worker.QueueZendeskVulnJobs(
						ctx,
						ds,
						kitlog.With(logger, "zendesk", "vulnerabilities"),
						recentVulns,
					); err != nil {
						level.Error(logger).Log("err", "queueing vulnerabilities to zendesk", "details", err)
						sentry.CaptureException(err)
					}

				default:
					err = errors.New("no vuln automations enabled")
					level.Error(logger).Log("err", "attempting to process vuln automations", err)
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
	vulnPath string, config config.FleetConfig, collectRecentVulns bool,
) map[string][]string {
	err := vulnerabilities.TranslateSoftwareToCPE(ctx, ds, vulnPath, logger, config)
	if err != nil {
		level.Error(logger).Log("msg", "analyzing vulnerable software: Software->CPE", "err", err)
		sentry.CaptureException(err)
		return nil
	}

	recentVulns, err := vulnerabilities.TranslateCPEToCVE(ctx, ds, vulnPath, logger, config, collectRecentVulns)
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
		maybeTriggerFailingPoliciesIntegration(ctx, ds, logger, identifier, appConfig, intervalReload, failingPoliciesSet)

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

func maybeTriggerFailingPoliciesIntegration(
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

	serverURL, err := url.Parse(appConfig.ServerSettings.ServerURL)
	if err != nil {
		level.Error(logger).Log("err", "parsing appConfig.ServerSettings.ServerURL", "details", err)
		sentry.CaptureException(err)
		return
	}

	logger = kitlog.With(logger, "webhook", "failing_policies")
	err = policies.TriggerFailingPoliciesIntegration(ctx, ds, logger, appConfig, failingPoliciesSet, func(policy *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
		switch cfg.AutomationType {
		case policies.FailingPolicyWebhook:
			return webhooks.SendFailingPoliciesBatchedPOSTs(
				ctx, policy, failingPoliciesSet, cfg.HostBatchSize, serverURL, cfg.WebhookURL, time.Now(), logger)

		case policies.FailingPolicyJira:
			hosts, err := failingPoliciesSet.ListHosts(policy.ID)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "listing hosts for failing policies set %d", policy.ID)
			}
			if err := worker.QueueJiraFailingPolicyJob(ctx, ds, logger, policy, hosts); err != nil {
				return err
			}
			if err := failingPoliciesSet.RemoveHosts(policy.ID, hosts); err != nil {
				return ctxerr.Wrapf(ctx, err, "removing %d hosts from failing policies set %d", len(hosts), policy.ID)
			}

		case policies.FailingPolicyZendesk:
			hosts, err := failingPoliciesSet.ListHosts(policy.ID)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "listing hosts for failing policies set %d", policy.ID)
			}
			if err := worker.QueueZendeskFailingPolicyJob(ctx, ds, logger, policy, hosts); err != nil {
				return err
			}
			if err := failingPoliciesSet.RemoveHosts(policy.ID, hosts); err != nil {
				return ctxerr.Wrapf(ctx, err, "removing %d hosts from failing policies set %d", len(hosts), policy.ID)
			}
		}
		return nil
	})
	if err != nil {
		level.Error(logger).Log("err", "triggering failing policies integration", "details", err)
		sentry.CaptureException(err)
	}
}

func cronWorker(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	identifier string,
) {
	const (
		lockDuration        = 10 * time.Minute
		lockAttemptInterval = 10 * time.Minute
	)

	logger = kitlog.With(logger, "cron", lockKeyWorker)

	// create the worker and register the Jira and Zendesk jobs even if no
	// integration is enabled, as that config can change live (and if it's not
	// there won't be any records to process so it will mostly just sleep).
	w := worker.NewWorker(ds, logger)
	jira := &worker.Jira{
		Datastore:     ds,
		Log:           logger,
		NewClientFunc: newJiraClient,
	}
	zendesk := &worker.Zendesk{
		Datastore: ds,
		Log:       logger,
	}
	// leave the url empty for now, will be filled when the lock is acquired with
	// the up-to-date config.
	w.Register(jira)
	w.Register(zendesk)

	// Read app config a first time before starting, to clear up any failer client
	// configuration if we're not on a fleet-owned server. Technically, the ServerURL
	// could change dynamically, but for the needs of forced client failures, this
	// is not a possible scenario.
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		sentry.CaptureException(err)
	}
	// we clear it even if we fail to load the app config, not a likely scenario
	// in our test environments for the needs of forced failures.
	if !strings.Contains(appConfig.ServerSettings.ServerURL, "fleetdm") {
		os.Unsetenv("FLEET_JIRA_CLIENT_FORCED_FAILURES")
		os.Unsetenv("FLEET_ZENDESK_CLIENT_FORCED_FAILURES")
	}

	// create client wrappers to introduce forced failures if configured
	// to do so via the environment variable.
	// format is "<modulo number>;<cve1>,<cve2>,<cve3>,..."
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

		jira.FleetURL = appConfig.ServerSettings.ServerURL

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

func newJiraClient(cfg fleet.TeamJiraIntegration) (worker.JiraClient, error) {
	client, err := externalsvc.NewJiraClient(&externalsvc.JiraOptions{
		BaseURL:           cfg.URL,
		BasicAuthUsername: cfg.Username,
		BasicAuthPassword: cfg.APIToken,
		ProjectKey:        cfg.ProjectKey,
	})
	if err != nil {
		return nil, err
	}

	// create client wrappers to introduce forced failures if configured
	// to do so via the environment variable.
	// format is "<modulo number>;<cve1>,<cve2>,<cve3>,..."
	failerClient := newFailerClient(os.Getenv("FLEET_JIRA_CLIENT_FORCED_FAILURES"))
	if failerClient != nil {
		failerClient.JiraClient = client
		return failerClient, nil
	}
	return client, nil
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
