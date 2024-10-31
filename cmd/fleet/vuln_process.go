package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

var (
	dev               bool
	devLicense        bool
	devExpiredLicense bool
	lockDuration      time.Duration
)

func createVulnProcessingCmd(configManager config.Manager) *cobra.Command {
	vulnProcessingCmd := &cobra.Command{
		Use:   "vuln_processing",
		Short: "Run the vulnerability processing features of Fleet",
		Long: `The vuln_processing command is intended for advanced configurations that want to externally manage 
vulnerability processing. By default the Fleet server command internally manages vulnerability processing via scheduled
'cron' style jobs, but setting 'vulnerabilities.disable_schedule=true' or 'FLEET_VULNERABILITIES_DISABLE_SCHEDULE=true' 
will disable it on the server allowing the user configure their own 'cron' mechanism. Successful processing will be indicated
by an exit code of zero.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			cfg := configManager.LoadConfig()
			if dev {
				applyDevFlags(&cfg)
			}

			logger := initLogger(cfg)
			logger = kitlog.With(logger, fleet.CronVulnerabilities)

			licenseInfo, err := initLicense(cfg, devLicense, devExpiredLicense)
			if err != nil {
				return err
			}

			if licenseInfo != nil && licenseInfo.IsPremium() && licenseInfo.IsExpired() {
				fleet.WriteExpiredLicenseBanner(os.Stderr)
			}

			ds, err := mysql.New(cfg.Mysql, clock.C)
			if err != nil {
				return err
			}

			// we need to ensure this command isn't running with an out-of-date database
			status, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				return err
			}

			var migrationError error
			switch status.StatusCode {
			case fleet.AllMigrationsCompleted:
				// only continue if db is considered up-to-date
			case fleet.NoMigrationsCompleted:
				migrationError = errors.New("no migrations completed")
			case fleet.SomeMigrationsCompleted:
				migrationError = errors.New("partial migrations completed")
			case fleet.UnknownMigrations:
				migrationError = errors.New("database migrations incompatible with current version")
			}
			if migrationError != nil {
				return fmt.Errorf("refusing to continue processing vulnerabilities err: %w", migrationError)
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), lockDuration)
			defer cancel()
			ctx = license.NewContext(ctx, licenseInfo)
			// using the same lock name as the cron scheduled version of vuln processing, that way if we fail to obtain the lock
			// it's most likely due to vulnerabilities.disable_schedule=false but still trying to run external vuln processing command
			lock, err := ds.Lock(ctx, string(fleet.CronVulnerabilities), "vuln_processing_command", lockDuration)
			if err != nil {
				return fmt.Errorf("failed to obtain vuln processing lock: %w", err)
			}
			if !lock {
				return errors.New("vulnerabilities processing locked")
			}

			defer func() {
				uerr := ds.Unlock(ctx, string(fleet.CronVulnerabilities), "vuln_processing_command")
				if uerr != nil {
					err = fmt.Errorf("failed to release vulnerability processing lock: %w", uerr)
				}
			}()
			appConfig, err := ds.AppConfig(ctx)
			if err != nil {
				return err
			}
			vulnConfig := cfg.Vulnerabilities
			vulnPath := configureVulnPath(vulnConfig, appConfig, logger)
			// this really shouldn't ever be empty string since it's defaulted, but could be due to some misconfiguration
			// we'll throw an error here since the entire point of this command is to process vulnerabilities
			if vulnPath == "" {
				return errors.New("vuln path empty, check environment variables or app config yml")
			}
			level.Info(logger).Log("msg", "scanning vulnerabilities")
			start := time.Now()
			vulnFuncs := getVulnFuncs(ctx, ds, logger, &vulnConfig)
			for _, vulnFunc := range vulnFuncs {
				if err := vulnFunc.VulnFunc(ctx); err != nil {
					return err
				}
			}
			level.Info(logger).Log("msg", "vulnerability processing finished", "took", time.Since(start))

			return
		},
	}
	vulnProcessingCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")
	vulnProcessingCmd.PersistentFlags().BoolVar(&devLicense, "dev_license", false, "Enable development license")
	vulnProcessingCmd.PersistentFlags().BoolVar(&devExpiredLicense, "dev_expired_license", false, "Enable expired development license")
	vulnProcessingCmd.PersistentFlags().DurationVar(
		&lockDuration,
		"lock_duration",
		time.Second*60*60,
		"the duration (https://pkg.go.dev/time#ParseDuration) the lock should be obtained, ideally this duration is less than the interval in which the job runs (defaults to 60m). If vuln processing isn't finished before this duration the command will exit with a non-zero status code.")
	vulnProcessingCmd.SilenceUsage = true

	return vulnProcessingCmd
}

func configureVulnPath(vulnConfig config.VulnerabilitiesConfig, appConfig *fleet.AppConfig, logger kitlog.Logger) (vulnPath string) {
	switch {
	case vulnConfig.DatabasesPath != "" && appConfig != nil && appConfig.VulnerabilitySettings.DatabasesPath != "":
		vulnPath = vulnConfig.DatabasesPath
		level.Info(logger).Log(
			"msg", "fleet config takes precedence over app config when both are configured",
			"databases_path", vulnPath,
		)
	case vulnConfig.DatabasesPath != "":
		vulnPath = vulnConfig.DatabasesPath
	case appConfig != nil && appConfig.VulnerabilitySettings.DatabasesPath != "":
		vulnPath = appConfig.VulnerabilitySettings.DatabasesPath
	default:
		level.Info(logger).Log("msg", "vulnerability scanning not configured, vulnerabilities databases path is empty")
	}
	return vulnPath
}

type NamedVulnFunc struct {
	Name     string
	VulnFunc func(ctx context.Context) error
}

func getVulnFuncs(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, config *config.VulnerabilitiesConfig) []NamedVulnFunc {
	vulnFuncs := []NamedVulnFunc{
		{
			Name: "cron_vulnerabilities",
			VulnFunc: func(ctx context.Context) error {
				return cronVulnerabilities(ctx, ds, logger, config)
			},
		},
		{
			Name: "cron_sync_host_software",
			VulnFunc: func(ctx context.Context) error {
				return ds.SyncHostsSoftware(ctx, time.Now())
			},
		},
		{
			Name: "cron_reconcile_software_titles",
			VulnFunc: func(ctx context.Context) error {
				return ds.ReconcileSoftwareTitles(ctx)
			},
		},
		{
			Name: "cron_sync_hosts_software_titles",
			VulnFunc: func(ctx context.Context) error {
				return ds.SyncHostsSoftwareTitles(ctx, time.Now())
			},
		},
		{
			Name: "update_host_issues_vulnerabilities_counts",
			VulnFunc: func(ctx context.Context) error {
				return ds.UpdateHostIssuesVulnerabilities(ctx)
			},
		},
	}

	return vulnFuncs
}
