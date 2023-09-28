package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	eewebhooks "github.com/fleetdm/fleet/v4/ee/server/webhooks"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/policies"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/externalsvc"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
	"github.com/fleetdm/fleet/v4/server/webhooks"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/hashicorp/go-multierror"
	"github.com/micromdm/nanodep/godep"
)

func errHandler(ctx context.Context, logger kitlog.Logger, msg string, err error) {
	level.Error(logger).Log("msg", msg, "err", err)
	sentry.CaptureException(err)
	ctxerr.Handle(ctx, err)
}

func newVulnerabilitiesSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	logger kitlog.Logger,
	config *config.VulnerabilitiesConfig,
) (*schedule.Schedule, error) {
	const name = string(fleet.CronVulnerabilities)
	interval := config.Periodicity
	vulnerabilitiesLogger := kitlog.With(logger, "cron", name)
	s := schedule.New(
		ctx, name, instanceID, interval, ds, ds,
		schedule.WithLogger(vulnerabilitiesLogger),
		schedule.WithJob(
			"cron_vulnerabilities",
			func(ctx context.Context) error {
				// TODO(lucas): Decouple cronVulnerabilities into multiple jobs.
				return cronVulnerabilities(ctx, ds, vulnerabilitiesLogger, config)
			},
		),
		schedule.WithJob(
			"cron_sync_host_software",
			func(ctx context.Context) error {
				return ds.SyncHostsSoftware(ctx, time.Now())
			},
		),
	)

	return s, nil
}

func cronVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	config *config.VulnerabilitiesConfig,
) error {
	if config == nil {
		return errors.New("nil configuration")
	}
	if config.CurrentInstanceChecks == "no" || config.CurrentInstanceChecks == "0" {
		level.Info(logger).Log("msg", "host not configured to check for vulnerabilities")
		return nil
	}

	level.Info(logger).Log("periodicity", config.Periodicity)

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading app config: %w", err)
	}

	if !appConfig.Features.EnableSoftwareInventory {
		level.Info(logger).Log("msg", "software inventory not configured")
		return nil
	}

	vulnPath := configureVulnPath(*config, appConfig, logger)
	if vulnPath != "" {
		level.Info(logger).Log("msg", "scanning vulnerabilities")
		if err := scanVulnerabilities(ctx, ds, logger, config, appConfig, vulnPath); err != nil {
			return fmt.Errorf("scanning vulnerabilities: %w", err)
		}
	}

	return nil
}

func scanVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	config *config.VulnerabilitiesConfig,
	appConfig *fleet.AppConfig,
	vulnPath string,
) error {
	level.Debug(logger).Log("msg", "creating vulnerabilities databases path", "databases_path", vulnPath)
	err := os.MkdirAll(vulnPath, 0o755)
	if err != nil {
		return fmt.Errorf("create vulnerabilities databases directory: %w", err)
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

	nvdVulns := checkNVDVulnerabilities(ctx, ds, logger, vulnPath, config, vulnAutomationEnabled != "")
	ovalVulns := checkOvalVulnerabilities(ctx, ds, logger, vulnPath, config, vulnAutomationEnabled != "")
	macOfficeVulns := checkMacOfficeVulnerabilities(ctx, ds, logger, vulnPath, config, vulnAutomationEnabled != "")

	checkWinVulnerabilities(ctx, ds, logger, vulnPath, config, vulnAutomationEnabled != "")

	// If no automations enabled, then there is nothing else to do...
	if vulnAutomationEnabled == "" {
		return nil
	}

	vulns := make([]fleet.SoftwareVulnerability, 0, len(nvdVulns)+len(ovalVulns)+len(macOfficeVulns))
	vulns = append(vulns, nvdVulns...)
	vulns = append(vulns, ovalVulns...)
	vulns = append(vulns, macOfficeVulns...)

	meta, err := ds.ListCVEs(ctx, config.RecentVulnerabilityMaxAge)
	if err != nil {
		errHandler(ctx, logger, "could not fetch CVE meta", err)
		return nil
	}

	recentV, matchingMeta := utils.RecentVulns(vulns, meta)

	if len(recentV) > 0 {
		switch vulnAutomationEnabled {
		case "webhook":
			args := webhooks.VulnArgs{
				Vulnerablities: recentV,
				Meta:           matchingMeta,
				AppConfig:      appConfig,
				Time:           time.Now(),
			}
			mapper := webhooks.NewMapper()
			if license.IsPremium(ctx) {
				mapper = eewebhooks.NewMapper()
			}
			// send recent vulnerabilities via webhook
			if err := webhooks.TriggerVulnerabilitiesWebhook(
				ctx,
				ds,
				kitlog.With(logger, "webhook", "vulnerabilities"),
				args,
				mapper,
			); err != nil {
				errHandler(ctx, logger, "triggering vulnerabilities webhook", err)
			}

		case "jira":
			// queue job to create jira issues
			if err := worker.QueueJiraVulnJobs(
				ctx,
				ds,
				kitlog.With(logger, "jira", "vulnerabilities"),
				recentV,
				matchingMeta,
			); err != nil {
				errHandler(ctx, logger, "queueing vulnerabilities to jira", err)
			}

		case "zendesk":
			// queue job to create zendesk ticket
			if err := worker.QueueZendeskVulnJobs(
				ctx,
				ds,
				kitlog.With(logger, "zendesk", "vulnerabilities"),
				recentV,
				matchingMeta,
			); err != nil {
				errHandler(ctx, logger, "queueing vulnerabilities to Zendesk", err)
			}

		default:
			err = ctxerr.New(ctx, "no vuln automations enabled")
			errHandler(ctx, logger, "attempting to process vuln automations", err)
		}
	}

	return nil
}

func checkWinVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	vulnPath string,
	config *config.VulnerabilitiesConfig,
	collectVulns bool,
) []fleet.OSVulnerability {
	var results []fleet.OSVulnerability

	// Get OS
	os, err := ds.ListOperatingSystems(ctx)
	if err != nil {
		errHandler(ctx, logger, "fetching list of operating systems", err)
		return nil
	}

	if !config.DisableDataSync {
		// Sync MSRC definitions
		err = msrc.SyncFromGithub(ctx, vulnPath, os)
		if err != nil {
			errHandler(ctx, logger, "updating msrc definitions", err)
		}
	}

	// Analyze all Win OS using the synched MSRC artifact.
	if !config.DisableWinOSVulnerabilities {
		for _, o := range os {
			if !o.IsWindows() {
				continue
			}

			start := time.Now()
			r, err := msrc.Analyze(ctx, ds, o, vulnPath, collectVulns)
			elapsed := time.Since(start)
			level.Debug(logger).Log(
				"msg", "msrc-analysis-done",
				"os name", o.Name,
				"os version", o.Version,
				"elapsed", elapsed,
				"found new", len(r))
			results = append(results, r...)
			if err != nil {
				errHandler(ctx, logger, "analyzing hosts for Windows vulnerabilities", err)
			}
		}
	}

	return results
}

func checkOvalVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	vulnPath string,
	config *config.VulnerabilitiesConfig,
	collectVulns bool,
) []fleet.SoftwareVulnerability {
	var results []fleet.SoftwareVulnerability

	// Get Platforms
	versions, err := ds.OSVersions(ctx, nil, nil, nil, nil)
	if err != nil {
		errHandler(ctx, logger, "updating oval definitions", err)
		return nil
	}

	if !config.DisableDataSync {
		// Sync on disk OVAL definitions with current OS Versions.
		downloaded, err := oval.Refresh(ctx, versions, vulnPath)
		if err != nil {
			errHandler(ctx, logger, "updating oval definitions", err)
		}
		for _, d := range downloaded {
			level.Debug(logger).Log("oval-sync-downloaded", d)
		}
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
		opts := nvd.SyncOptions{
			VulnPath:           config.DatabasesPath,
			CPEDBURL:           config.CPEDatabaseURL,
			CPETranslationsURL: config.CPETranslationsURL,
			CVEFeedPrefixURL:   config.CVEFeedPrefixURL,
		}
		err := nvd.Sync(opts, logger)
		if err != nil {
			errHandler(ctx, logger, "syncing vulnerability database", err)
			// don't return, continue on ...
		}
	}

	if err := nvd.LoadCVEMeta(ctx, logger, vulnPath, ds); err != nil {
		errHandler(ctx, logger, "load cve meta", err)
		// don't return, continue on ...
	}

	err := nvd.TranslateSoftwareToCPE(ctx, ds, vulnPath, logger)
	if err != nil {
		errHandler(ctx, logger, "analyzing vulnerable software: Software->CPE", err)
		return nil
	}

	vulns, err := nvd.TranslateCPEToCVE(ctx, ds, vulnPath, logger, collectVulns, config.Periodicity)
	if err != nil {
		errHandler(ctx, logger, "analyzing vulnerable software: CPE->CVE", err)
		return nil
	}

	return vulns
}

func checkMacOfficeVulnerabilities(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	vulnPath string,
	config *config.VulnerabilitiesConfig,
	collectVulns bool,
) []fleet.SoftwareVulnerability {
	if !config.DisableDataSync {
		err := macoffice.SyncFromGithub(ctx, vulnPath)
		if err != nil {
			errHandler(ctx, logger, "updating mac office release notes", err)
		}

		level.Debug(logger).Log("msg", "finished sync mac office release notes")
	}

	start := time.Now()
	r, err := macoffice.Analyze(ctx, ds, vulnPath, collectVulns)
	elapsed := time.Since(start)

	level.Debug(logger).Log(
		"msg", "mac-office-analysis-done",
		"elapsed", elapsed,
		"found new", len(r))

	if err != nil {
		errHandler(ctx, logger, "analyzing mac office products for vulnerabilities", err)
	}

	return r
}

func newAutomationsSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	logger kitlog.Logger,
	intervalReload time.Duration,
	failingPoliciesSet fleet.FailingPolicySet,
) (*schedule.Schedule, error) {
	const (
		name            = string(fleet.CronAutomations)
		defaultInterval = 24 * time.Hour
	)
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting app config: %w", err)
	}
	s := schedule.New(
		// TODO(sarah): Reconfigure settings so automations interval doesn't reside under webhook settings
		ctx, name, instanceID, appConfig.WebhookSettings.Interval.ValueOr(defaultInterval), ds, ds,
		schedule.WithLogger(kitlog.With(logger, "cron", name)),
		schedule.WithConfigReloadInterval(intervalReload, func(ctx context.Context) (time.Duration, error) {
			appConfig, err := ds.AppConfig(ctx)
			if err != nil {
				return 0, err
			}
			newInterval := appConfig.WebhookSettings.Interval.ValueOr(defaultInterval)
			return newInterval, nil
		}),
		schedule.WithJob(
			"host_status_webhook",
			func(ctx context.Context) error {
				return webhooks.TriggerHostStatusWebhook(
					ctx, ds, kitlog.With(logger, "automation", "host_status"),
				)
			},
		),
		schedule.WithJob(
			"fire_outdated_automations",
			func(ctx context.Context) error {
				return scheduleFailingPoliciesAutomation(ctx, ds, kitlog.With(logger, "automation", "fire_outdated_automations"), failingPoliciesSet)
			},
		),
		schedule.WithJob(
			"failing_policies_automation",
			func(ctx context.Context) error {
				return triggerFailingPoliciesAutomation(ctx, ds, kitlog.With(logger, "automation", "failing_policies"), failingPoliciesSet)
			},
		),
	)

	return s, nil
}

func scheduleFailingPoliciesAutomation(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	failingPoliciesSet fleet.FailingPolicySet,
) error {
	for {
		batch, err := ds.OutdatedAutomationBatch(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "OutdatedAutomationBatch")
		}
		if len(batch) == 0 {
			break
		}
		level.Debug(logger).Log("adding_hosts", len(batch))
		for _, p := range batch {
			if err := failingPoliciesSet.AddHost(p.PolicyID, p.Host); err != nil {
				return ctxerr.Wrap(ctx, err, "failingPolicesSet.AddHost")
			}
		}
	}
	return nil
}

func triggerFailingPoliciesAutomation(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	failingPoliciesSet fleet.FailingPolicySet,
) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("getting app config: %w", err)
	}
	serverURL, err := url.Parse(appConfig.ServerSettings.ServerURL)
	if err != nil {
		return fmt.Errorf("parsing appConfig.ServerSettings.ServerURL: %w", err)
	}

	err = policies.TriggerFailingPoliciesAutomation(ctx, ds, logger, failingPoliciesSet, func(policy *fleet.Policy, cfg policies.FailingPolicyAutomationConfig) error {
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
		return fmt.Errorf("triggering failing policies automation: %w", err)
	}

	return nil
}

func newWorkerIntegrationsSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	logger kitlog.Logger,
	depStorage *mysql.NanoDEPStorage,
	commander *apple_mdm.MDMAppleCommander,
) (*schedule.Schedule, error) {
	const (
		name = string(fleet.CronWorkerIntegrations)

		// the schedule interval is shorter than the max run time of the scheduled
		// job, but that's ok - the job will acquire and extend the lock as long as
		// it runs, the shorter interval is to make sure we don't wait more than
		// that interval to start a new job when none is running.
		scheduleInterval = 1 * time.Minute  // schedule a worker to run every minute if none is running
		maxRunTime       = 10 * time.Minute // allow the worker to run for 10 minutes
	)

	logger = kitlog.With(logger, "cron", name)

	// create the worker and register the Jira and Zendesk jobs even if no
	// integration is enabled, as that config can change live (and if it's not
	// there won't be any records to process so it will mostly just sleep).
	w := worker.NewWorker(ds, logger)
	// leave the url empty for now, will be filled when the lock is acquired with
	// the up-to-date config.
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
	var (
		depSvc *apple_mdm.DEPService
		depCli *godep.Client
	)
	// depStorage could be nil if mdm is not configured for fleet, in which case
	// we leave depSvc and deCli nil and macos setup assistants jobs will be
	// no-ops.
	if depStorage != nil {
		depSvc = apple_mdm.NewDEPService(ds, depStorage, logger)
		depCli = apple_mdm.NewDEPClient(depStorage, ds, logger)
	}
	macosSetupAsst := &worker.MacosSetupAssistant{
		Datastore:  ds,
		Log:        logger,
		DEPService: depSvc,
		DEPClient:  depCli,
	}
	appleMDM := &worker.AppleMDM{
		Datastore: ds,
		Log:       logger,
		Commander: commander,
	}
	w.Register(jira, zendesk, macosSetupAsst, appleMDM)

	// Read app config a first time before starting, to clear up any failer client
	// configuration if we're not on a fleet-owned server. Technically, the ServerURL
	// could change dynamically, but for the needs of forced client failures, this
	// is not a possible scenario.
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting app config: %w", err)
	}

	// we clear it even if we fail to load the app config, not a likely scenario
	// in our test environments for the needs of forced failures.
	if !strings.Contains(appConfig.ServerSettings.ServerURL, "fleetdm") {
		os.Unsetenv("FLEET_JIRA_CLIENT_FORCED_FAILURES")
		os.Unsetenv("FLEET_ZENDESK_CLIENT_FORCED_FAILURES")
	}

	s := schedule.New(
		ctx, name, instanceID, scheduleInterval, ds, ds,
		schedule.WithAltLockID("worker"),
		schedule.WithLogger(logger),
		schedule.WithJob("integrations_worker", func(ctx context.Context) error {
			// Read app config to be able to use the latest configuration for integrations.
			appConfig, err := ds.AppConfig(ctx)
			if err != nil {
				return fmt.Errorf("getting app config: %w", err)
			}

			jira.FleetURL = appConfig.ServerSettings.ServerURL
			zendesk.FleetURL = appConfig.ServerSettings.ServerURL

			workCtx, cancel := context.WithTimeout(ctx, maxRunTime)
			defer cancel()

			if err := w.ProcessJobs(workCtx); err != nil {
				return fmt.Errorf("processing integrations jobs: %w", err)
			}
			return nil
		}),
	)

	return s, nil
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

func newCleanupsAndAggregationSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	logger kitlog.Logger,
	enrollHostLimiter fleet.EnrollHostLimiter,
	config *config.FleetConfig,
) (*schedule.Schedule, error) {
	const (
		name            = string(fleet.CronCleanupsThenAggregation)
		defaultInterval = 1 * time.Hour
	)
	s := schedule.New(
		ctx, name, instanceID, defaultInterval, ds, ds,
		// Using leader for the lock to be backwards compatilibity with old deployments.
		schedule.WithAltLockID("leader"),
		schedule.WithLogger(kitlog.With(logger, "cron", name)),
		// Run cleanup jobs first.
		schedule.WithJob(
			"distributed_query_campaigns",
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
		schedule.WithJob(
			"cleanup_expired_password_reset_requests",
			func(ctx context.Context) error {
				return ds.CleanupExpiredPasswordResetRequests(ctx)
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
			"aggregated_munki_and_mdm",
			func(ctx context.Context) error {
				return ds.GenerateAggregatedMunkiAndMDM(ctx)
			},
		),
		schedule.WithJob(
			"increment_policy_violation_days",
			func(ctx context.Context) error {
				return ds.IncrementPolicyViolationDays(ctx)
			},
		),
		schedule.WithJob(
			"update_os_versions",
			func(ctx context.Context) error {
				return ds.UpdateOSVersions(ctx)
			},
		),
		schedule.WithJob(
			"verify_disk_encryption_keys",
			func(ctx context.Context) error {
				return verifyDiskEncryptionKeys(ctx, logger, ds, config)
			},
		),
	)

	return s, nil
}

func verifyDiskEncryptionKeys(
	ctx context.Context,
	logger kitlog.Logger,
	ds fleet.Datastore,
	config *config.FleetConfig,
) error {
	if !config.MDM.IsAppleSCEPSet() {
		logger.Log("inf", "skipping verification of encryption keys as MDM is not fully configured")
		return nil
	}

	keys, err := ds.GetUnverifiedDiskEncryptionKeys(ctx)
	if err != nil {
		logger.Log("err", "unable to get unverified disk encryption keys from the database", "details", err)
		return err
	}

	cert, _, _, err := config.MDM.AppleSCEP()
	if err != nil {
		logger.Log("err", "unable to get SCEP keypair to decrypt keys", "details", err)
		return err
	}

	decryptable := []uint{}
	undecryptable := []uint{}
	var latest time.Time
	for _, key := range keys {
		if key.UpdatedAt.After(latest) {
			latest = key.UpdatedAt
		}
		if _, err := apple_mdm.DecryptBase64CMS(key.Base64Encrypted, cert.Leaf, cert.PrivateKey); err != nil {
			undecryptable = append(undecryptable, key.HostID)
			continue
		}
		decryptable = append(decryptable, key.HostID)
	}

	if err := ds.SetHostsDiskEncryptionKeyStatus(ctx, decryptable, true, latest); err != nil {
		logger.Log("err", "unable to update decryptable status", "details", err)
		return err
	}
	if err := ds.SetHostsDiskEncryptionKeyStatus(ctx, undecryptable, false, latest); err != nil {
		logger.Log("err", "unable to update decryptable status", "details", err)
		return err
	}

	return nil
}

func newUsageStatisticsSchedule(ctx context.Context, instanceID string, ds fleet.Datastore, config config.FleetConfig, license *fleet.LicenseInfo, logger kitlog.Logger) (*schedule.Schedule, error) {
	const (
		name            = string(fleet.CronUsageStatistics)
		defaultInterval = 1 * time.Hour
	)
	s := schedule.New(
		ctx, name, instanceID, defaultInterval, ds, ds,
		schedule.WithLogger(kitlog.With(logger, "cron", name)),
		schedule.WithJob(
			"try_send_statistics",
			func(ctx context.Context) error {
				// NOTE(mna): this is not a route from the fleet server (not in server/service/handler.go) so it
				// will not automatically support the /latest/ versioning. Leaving it as /v1/ for that reason.
				return trySendStatistics(ctx, ds, fleet.StatisticsFrequency, "https://fleetdm.com/api/v1/webhooks/receive-usage-analytics", config)
			},
		),
	)

	return s, nil
}

func trySendStatistics(ctx context.Context, ds fleet.Datastore, frequency time.Duration, url string, config config.FleetConfig) error {
	ac, err := ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.ServerSettings.EnableAnalytics {
		return nil
	}

	stats, shouldSend, err := ds.ShouldSendStatistics(ctx, frequency, config)
	if err != nil {
		return err
	}
	if !shouldSend {
		return nil
	}

	if err := server.PostJSONWithTimeout(ctx, url, stats); err != nil {
		return err
	}

	if err := ds.CleanupStatistics(ctx); err != nil {
		return err
	}

	return ds.RecordStatisticsSent(ctx)
}

// newAppleMDMDEPProfileAssigner creates the schedule to run the DEP syncer+assigner.
// The DEP syncer+assigner fetches devices from Apple Business Manager (aka ABM) and applies
// the current configured DEP profile to them.
func newAppleMDMDEPProfileAssigner(
	ctx context.Context,
	instanceID string,
	periodicity time.Duration,
	ds fleet.Datastore,
	depStorage *mysql.NanoDEPStorage,
	logger kitlog.Logger,
) (*schedule.Schedule, error) {
	const name = string(fleet.CronAppleMDMDEPProfileAssigner)
	logger = kitlog.With(logger, "cron", name, "component", "nanodep-syncer")
	fleetSyncer := apple_mdm.NewDEPService(ds, depStorage, logger)
	s := schedule.New(
		ctx, name, instanceID, periodicity, ds, ds,
		schedule.WithLogger(logger),
		schedule.WithJob("dep_syncer", func(ctx context.Context) error {
			return fleetSyncer.RunAssigner(ctx)
		}),
	)

	return s, nil
}

func newMDMAppleProfileManager(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger kitlog.Logger,
	loggingDebug bool,
) (*schedule.Schedule, error) {
	const (
		name = string(fleet.CronMDMAppleProfileManager)
		// Note: per a request from #g-product we are running this cron
		// every 30 seconds, we should re-evaluate how we handle the
		// cron interval as we scale to more hosts.
		defaultInterval = 30 * time.Second
	)
	logger = kitlog.With(logger, "cron", name)
	s := schedule.New(
		ctx, name, instanceID, defaultInterval, ds, ds,
		schedule.WithLogger(logger),
		schedule.WithJob("manage_profiles", func(ctx context.Context) error {
			return service.ReconcileProfiles(ctx, ds, commander, logger)
		}),
	)

	return s, nil
}

func cleanupCronStatsOnShutdown(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, instanceID string) {
	if err := ds.UpdateAllCronStatsForInstance(ctx, instanceID, fleet.CronStatsStatusPending, fleet.CronStatsStatusCanceled); err != nil {
		logger.Log("err", "cancel pending cron stats for instance", "details", err)
	}
}

func newActivitiesStreamingSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	logger kitlog.Logger,
	auditLogger fleet.JSONLogger,
) (*schedule.Schedule, error) {
	const (
		name     = string(fleet.CronActivitiesStreaming)
		interval = 5 * time.Minute
	)
	logger = kitlog.With(logger, "cron", name)
	s := schedule.New(
		ctx, name, instanceID, interval, ds, ds,
		schedule.WithLogger(logger),
		schedule.WithJob(
			"cron_activities_streaming",
			func(ctx context.Context) error {
				return cronActivitiesStreaming(ctx, ds, logger, auditLogger)
			},
		),
	)
	return s, nil
}

var ActivitiesToStreamBatchCount uint = 500

func cronActivitiesStreaming(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	auditLogger fleet.JSONLogger,
) error {
	page := uint(0)
	for {
		// (1) Get batch of activities that haven't been streamed.
		activitiesToStream, _, err := ds.ListActivities(ctx, fleet.ListActivitiesOptions{
			ListOptions: fleet.ListOptions{
				OrderKey:       "id",
				OrderDirection: fleet.OrderAscending,
				PerPage:        ActivitiesToStreamBatchCount,
				Page:           page,
			},
			Streamed: ptr.Bool(false),
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list activities")
		}
		if len(activitiesToStream) == 0 {
			return nil
		}

		// (2) Stream the activities.
		var (
			streamedIDs []uint
			multiErr    error
		)
		// We stream one activity at a time (instead of writing them all with
		// one auditLogger.Write call) to know which ones succeeded/failed,
		// and also because this method happens asynchronously,
		// so we don't need real-time performance.
		for _, activity := range activitiesToStream {
			b, err := json.Marshal(activity)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "marshal activity")
			}
			if err := auditLogger.Write(ctx, []json.RawMessage{json.RawMessage(b)}); err != nil {
				if len(streamedIDs) == 0 {
					return ctxerr.Wrapf(ctx, err, "stream first activity: %d", activity.ID)
				}
				multiErr = multierror.Append(multiErr, ctxerr.Wrapf(ctx, err, "stream activity: %d", activity.ID))
				// We stop streaming upon the first error (will retry on next cron iteration)
				break
			}
			streamedIDs = append(streamedIDs, activity.ID)
		}

		logger.Log("streamed-events", len(streamedIDs))

		// (3) Mark the streamed activities as streamed.
		if err := ds.MarkActivitiesAsStreamed(ctx, streamedIDs); err != nil {
			multiErr = multierror.Append(multiErr, ctxerr.Wrap(ctx, err, "mark activities as streamed"))
		}

		// If there was an error while streaming or updating activities, return.
		if multiErr != nil {
			return multiErr
		}

		if len(activitiesToStream) < int(ActivitiesToStreamBatchCount) {
			return nil
		}
		page += 1
	}
}
