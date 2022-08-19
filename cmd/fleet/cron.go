package main

import (
	"context"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/policies"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/fleetdm/fleet/v4/server/webhooks"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func errHandler(ctx context.Context, logger kitlog.Logger, msg string, err error) {
	level.Error(logger).Log("err", msg, "details", err)
	sentry.CaptureException(err)
	ctxerr.Handle(ctx, err)
}

func cronVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	identifier string,
	config *config.VulnerabilitiesConfig,
) {
	logger = kitlog.With(logger, "cron", lockKeyVulnerabilities)

	if config.CurrentInstanceChecks == "no" || config.CurrentInstanceChecks == "0" {
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
		config.DatabasesPath == "" {
		level.Info(logger).Log("vulnerability scanning", "not configured")
		vulnDisabled = true
	}
	if !appConfig.HostSettings.EnableSoftwareInventory {
		level.Info(logger).Log("software inventory", "not configured")
		return
	}

	vulnPath := appConfig.VulnerabilitySettings.DatabasesPath
	if vulnPath == "" {
		vulnPath = config.DatabasesPath
	}
	if config.DatabasesPath != "" && config.DatabasesPath != vulnPath {
		vulnPath = config.DatabasesPath
		level.Info(logger).Log(
			"databases_path", "fleet config takes precedence over app config when both are configured",
			"result", vulnPath)
	}

	if !vulnDisabled {
		level.Info(logger).Log("databases-path", vulnPath)
	}
	level.Info(logger).Log("periodicity", config.Periodicity)

	if !vulnDisabled {
		if config.CurrentInstanceChecks == "auto" {
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
			ticker.Reset(config.Periodicity)
		case <-ctx.Done():
			level.Debug(logger).Log("exit", "done with cron.")
			return
		}
		if config.CurrentInstanceChecks == "auto" {
			if locked, err := ds.Lock(ctx, lockKeyVulnerabilities, identifier, 1*time.Hour); err != nil {
				errHandler(ctx, logger, "error acquiring lock", err)
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
				errHandler(ctx, logger, "couldn't refresh app config", err)
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
						err := ctxerr.New(ctx, "jira check")
						errHandler(ctx, logger, "more than one automation enabled", err)
					}
					vulnAutomationEnabled = "jira"
					break
				}
			}
			// check for Zendesk integrations
			for _, z := range appConfig.Integrations.Zendesk {
				if z.EnableSoftwareVulnerabilities {
					if vulnAutomationEnabled != "" {
						err := ctxerr.New(ctx, "zendesk check")
						errHandler(ctx, logger, "more than one automation enabled", err)
					}
					vulnAutomationEnabled = "zendesk"
					break
				}
			}
			level.Debug(logger).Log("vulnAutomationEnabled", vulnAutomationEnabled)

			collectVulns := vulnAutomationEnabled != ""
			nvdVulns := checkNVDVulnerabilities(ctx, ds, logger, vulnPath, config, collectVulns)
			ovalVulns := checkOvalVulnerabilities(ctx, ds, logger, vulnPath, config, collectVulns)
			recentVulns := filterRecentVulns(ctx, ds, logger, nvdVulns, ovalVulns, config.RecentVulnerabilityMaxAge)

			if len(recentVulns) > 0 {
				switch vulnAutomationEnabled {
				case "webhook":
					// send recent vulnerabilities via webhook
					if err := webhooks.TriggerVulnerabilitiesWebhook(
						ctx,
						ds,
						kitlog.With(logger, "webhook", "vulnerabilities"),
						recentVulns,
						appConfig,
						time.Now()); err != nil {
						errHandler(ctx, logger, "triggering vulnerabilities webhook", err)
					}

				case "jira":
					// queue job to create jira issues
					if err := worker.QueueJiraVulnJobs(
						ctx,
						ds,
						kitlog.With(logger, "jira", "vulnerabilities"),
						recentVulns,
					); err != nil {
						errHandler(ctx, logger, "queueing vulnerabilities to jira", err)
					}

				case "zendesk":
					// queue job to create zendesk ticket
					if err := worker.QueueZendeskVulnJobs(
						ctx,
						ds,
						kitlog.With(logger, "zendesk", "vulnerabilities"),
						recentVulns,
					); err != nil {
						errHandler(ctx, logger, "queueing vulnerabilities to Zendesk", err)
					}

				default:
					err = ctxerr.New(ctx, "no vuln automations enabled")
					errHandler(ctx, logger, "attempting to process vuln automations", err)
				}
			}
		}

		if err := ds.SyncHostsSoftware(ctx, time.Now()); err != nil {
			errHandler(ctx, logger, "calculating hosts count per software", err)
		}
		level.Debug(logger).Log("loop", "done")
	}
}

func filterRecentVulns(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	nvdVulns []fleet.SoftwareVulnerability,
	ovalVulns []fleet.SoftwareVulnerability,
	maxAge time.Duration,
) []fleet.SoftwareVulnerability {
	if len(nvdVulns) == 0 && len(ovalVulns) == 0 {
		return nil
	}

	recent, err := ds.ListCVEs(ctx, maxAge)
	if err != nil {
		errHandler(ctx, logger, "could not fetch recent CVEs", err)
		return nil
	}

	lookup := make(map[string]bool)
	for _, r := range recent {
		lookup[r.CVE] = true
	}

	filtered := make(map[string]fleet.SoftwareVulnerability)
	for _, v := range nvdVulns {
		if _, ok := lookup[v.CVE]; ok {
			filtered[v.Key()] = v
		}
	}
	for _, v := range ovalVulns {
		if _, ok := lookup[v.CVE]; ok {
			filtered[v.Key()] = v
		}
	}

	result := make([]fleet.SoftwareVulnerability, 0, len(filtered))
	for _, v := range filtered {
		result = append(result, v)
	}

	return result
}

func checkOvalVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	vulnPath string,
	config *config.VulnerabilitiesConfig,
	collectVulns bool,
) []fleet.SoftwareVulnerability {
	if config.DisableDataSync {
		return nil
	}

	var results []fleet.SoftwareVulnerability

	// Get Platforms
	versions, err := ds.OSVersions(ctx, nil, nil, nil, nil)
	if err != nil {
		errHandler(ctx, logger, "updating oval definitions", err)
		return nil
	}

	// Sync on disk OVAL definitions with current OS Versions.
	client := fleethttp.NewClient()
	downloaded, err := oval.Refresh(ctx, client, versions, vulnPath)
	if err != nil {
		errHandler(ctx, logger, "updating oval definitions", err)
	}
	for _, d := range downloaded {
		level.Debug(logger).Log("oval-sync-downloaded", d)
	}

	// Analyze all supported os versions using the synched OVAL definitions.
	for _, version := range versions.OSVersions {
		start := time.Now()
		r, err := oval.Analyze(ctx, ds, version, vulnPath, collectVulns)
		elapsed := time.Since(start)
		level.Debug(logger).Log(
			"msg", "oval-analysis-done",
			"platform", version.Name,
			"elapsed", elapsed,
			"found new", len(r))
		results = append(results, r...)
		if err != nil {
			errHandler(ctx, logger, "analyzing oval definitions", err)
		}
	}

	return results
}

func checkNVDVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	vulnPath string,
	config *config.VulnerabilitiesConfig,
	collectVulns bool,
) []fleet.SoftwareVulnerability {
	if !config.DisableDataSync {
		err := vulnerabilities.Sync(vulnPath, config.CPEDatabaseURL)
		if err != nil {
			errHandler(ctx, logger, "syncing vulnerability database", err)
			return nil
		}
	}

	if err := vulnerabilities.LoadCVEMeta(logger, vulnPath, ds); err != nil {
		errHandler(ctx, logger, "load cve meta", err)
		// don't return, continue on ...
	}

	err := vulnerabilities.TranslateSoftwareToCPE(ctx, ds, vulnPath, logger)
	if err != nil {
		errHandler(ctx, logger, "analyzing vulnerable software: Software->CPE", err)
		return nil
	}

	vulns, err := vulnerabilities.TranslateCPEToCVE(ctx, ds, vulnPath, logger, collectVulns)
	if err != nil {
		errHandler(ctx, logger, "analyzing vulnerable software: CPE->CVE", err)
		return nil
	}

	return vulns
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
			errHandler(ctx, logger, "couldn't read app config", err)
		} else {
			ticker.Reset(appConfig.WebhookSettings.Interval.ValueOr(24 * time.Hour))
			start = time.Now()
		}

		// We set the db lock durations to match the intervalReload.
		maybeTriggerHostStatus(ctx, ds, logger, identifier, appConfig, intervalReload)
		maybeTriggerFailingPoliciesAutomation(ctx, ds, logger, identifier, appConfig, intervalReload, failingPoliciesSet)

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
		errHandler(ctx, logger, "triggering host status webhook", err)
	}
}

func maybeTriggerFailingPoliciesAutomation(
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
		errHandler(ctx, logger, "parsing appConfig.ServerSettings.ServerURL", err)
		return
	}

	logger = kitlog.With(logger, "webhook", "failing_policies")
	err = policies.TriggerFailingPoliciesAutomation(ctx, ds, logger, appConfig, failingPoliciesSet, func(policy *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
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
		errHandler(ctx, logger, "triggering failing policies automation", err)
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
		Datastore:     ds,
		Log:           logger,
		NewClientFunc: newZendeskClient,
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
		errHandler(ctx, logger, "couldn't read app config", err)
	}
	// we clear it even if we fail to load the app config, not a likely scenario
	// in our test environments for the needs of forced failures.
	if !strings.Contains(appConfig.ServerSettings.ServerURL, "fleetdm") {
		os.Unsetenv("FLEET_JIRA_CLIENT_FORCED_FAILURES")
		os.Unsetenv("FLEET_ZENDESK_CLIENT_FORCED_FAILURES")
	}

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
			errHandler(ctx, logger, "couldn't read app config", err)
			continue
		}

		jira.FleetURL = appConfig.ServerSettings.ServerURL
		zendesk.FleetURL = appConfig.ServerSettings.ServerURL

		workCtx, cancel := context.WithTimeout(ctx, lockDuration)
		if err := w.ProcessJobs(workCtx); err != nil {
			errHandler(ctx, logger, "Error processing jobs", err)
		}
		cancel() // don't use defer inside loop
	}
}

func newJiraClient(opts *externalsvc.JiraOptions) (worker.JiraClient, error) {
	client, err := externalsvc.NewJiraClient(opts)
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

func newZendeskClient(opts *externalsvc.ZendeskOptions) (worker.ZendeskClient, error) {
	client, err := externalsvc.NewZendeskClient(opts)
	if err != nil {
		return nil, err
	}

	// create client wrappers to introduce forced failures if configured
	// to do so via the environment variable.
	// format is "<modulo number>;<cve1>,<cve2>,<cve3>,..."
	failerClient := newFailerClient(os.Getenv("FLEET_ZENDESK_CLIENT_FORCED_FAILURES"))
	if failerClient != nil {
		failerClient.ZendeskClient = client
		return failerClient, nil
	}
	return client, nil
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

func startCleanupsAndAggregationSchedule(
	ctx context.Context, instanceID string, ds fleet.Datastore, logger kitlog.Logger, enrollHostLimiter fleet.EnrollHostLimiter,
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
				_, err := ds.CleanupIncomingHosts(ctx, time.Now())
				return err
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
				_, err := ds.CleanupExpiredHosts(ctx)
				return err
			},
		),
		schedule.WithJob(
			"policy_membership",
			func(ctx context.Context) error {
				return ds.CleanupPolicyMembership(ctx, time.Now())
			},
		),
		schedule.WithJob(
			"sync_enrolled_host_ids",
			func(ctx context.Context) error {
				return enrollHostLimiter.SyncEnrolledHostIDs(ctx)
			},
		),
		schedule.WithJob(
			"cleanup_host_operating_systems",
			func(ctx context.Context) error {
				return ds.CleanupHostOperatingSystems(ctx)
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
		schedule.WithJob(
			"update_os_versions",
			func(ctx context.Context) error {
				return ds.UpdateOSVersions(ctx)
			},
		),
	).Start()
}

func startSendStatsSchedule(ctx context.Context, instanceID string, ds fleet.Datastore, config config.FleetConfig, license *fleet.LicenseInfo, logger kitlog.Logger) {
	schedule.New(
		ctx, "stats", instanceID, 1*time.Hour, ds,
		schedule.WithLogger(kitlog.With(logger, "cron", "stats")),
		schedule.WithJob(
			"try_send_statistics",
			func(ctx context.Context) error {
				// NOTE(mna): this is not a route from the fleet server (not in server/service/handler.go) so it
				// will not automatically support the /latest/ versioning. Leaving it as /v1/ for that reason.
				return trySendStatistics(ctx, ds, fleet.StatisticsFrequency, "https://fleetdm.com/api/v1/webhooks/receive-usage-analytics", config, license)
			},
		),
	).Start()
}

func trySendStatistics(ctx context.Context, ds fleet.Datastore, frequency time.Duration, url string, config config.FleetConfig, license *fleet.LicenseInfo) error {
	ac, err := ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.ServerSettings.EnableAnalytics {
		return nil
	}

	stats, shouldSend, err := ds.ShouldSendStatistics(ctx, frequency, config, license)
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
