package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities"
	"github.com/fleetdm/fleet/v4/server/webhooks"
	"github.com/getsentry/sentry-go"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/log/level"
)

type vulnerabilitiesJobStats struct {
	StartedAt    time.Time `json:"started_at" db:"started_at"`
	CompletedAt  time.Time `json:"completed_at" db:"completed_at"`
	TotalRunTime string    `json:"total_run_time" db:"total_run_time"`
}

func DoVulnProcessing(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, config config.FleetConfig) (interface{}, error) {
	stats := make(map[string]string)
	startedAt := time.Now()

	if config.Vulnerabilities.CurrentInstanceChecks == "no" || config.Vulnerabilities.CurrentInstanceChecks == "0" {
		level.Info(logger).Log("vulnerability scanning", "host not configured to check for vulnerabilities")
		return stats, nil // TODO
	}

	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		level.Error(logger).Log("config", "couldn't read app config", "err", err)
		return stats, nil // TODO
	}

	vulnDisabled := false
	if appConfig.VulnerabilitySettings.DatabasesPath == "" &&
		config.Vulnerabilities.DatabasesPath == "" {
		level.Info(logger).Log("vulnerability scanning", "not configured")
		vulnDisabled = true
	}
	if !appConfig.HostSettings.EnableSoftwareInventory {
		level.Info(logger).Log("software inventory", "not configured")
		return stats, nil // TODO
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
				return stats, err // TODO: how should we handle these kinds of errors/exits?
			}
		}
	}

	if !vulnDisabled {
		level.Debug(logger).Log("vulnerabilities", "checking for recent vulnerabilities")
		level.Debug(logger).Log("vuln-path", vulnPath)
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

	level.Debug(logger).Log("vulnerabilities", "done")

	jobStats := &vulnerabilitiesJobStats{
		StartedAt:    startedAt,
		CompletedAt:  time.Now(),
		TotalRunTime: fmt.Sprint(time.Now().Sub(startedAt)),
	}
	statsData, err := json.Marshal(jobStats)
	if err != nil {
		level.Error(logger).Log("msg", "marshalling asyncVuln job stats", "err", err)
		sentry.CaptureException(err)
	}
	stats["do_async_vuln"] = string(statsData)

	return stats, nil // TODO
}

func checkVulnerabilities(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger,
	vulnPath string, config config.FleetConfig, vulnWebhookCfg fleet.VulnerabilitiesWebhookSettings,
) map[string][]string {
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
