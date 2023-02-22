package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/cobra"
)

var dev bool

func createVulnProcessingCmd(configManager config.Manager) *cobra.Command {
	vulnProcessingCmd := &cobra.Command{
		Use:   "vuln_processing",
		Short: "Run the vulnerability processing features of Fleet",
		Long: `The vuln_processing command is intended for advanced configurations that want to externally manage 
vulnerability processing. By default the Fleet server command internally manages vulnerability processing via scheduled
'cron' style jobs.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := configManager.LoadConfig()
			if dev {
				applyDevFlags(&cfg)
			}

			ds, err := mysql.New(cfg.Mysql, clock.C)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			// we need to ensure this command isn't running with an out-of-date database
			status, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

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
				initFatal(migrationError, "refusing to continue processing vulnerabilities, database out of sync")
			}

			ctx, cancel := context.WithTimeout(context.Background(), lockDuration)
			ctx = license.NewContext(ctx, licenseInfo)
			instanceID, err := server.GenerateRandomText(64)
			if err != nil {
				initFatal(errors.New("error generating random instance identifier"), "")
			}
			// using the same lock name as the cron scheduled version of vuln processing, that way if we fail to obtain the lock
			// it's most likely due to vulnerabilities.disable_schedule=false but still trying to run external vuln processing command
			lock, err := ds.Lock(ctx, string(fleet.CronVulnerabilities), instanceID, lockDuration)
			if err != nil {
				initFatal(err, "failed to obtain vuln processing lock")
			}
			if !lock {
				initFatal(errors.New("vulnerabilities processing locked"),
					"failed to obtain vuln processing lock, something else still has lock ownership")
			}
			defer func() {
				err = ds.Unlock(ctx, string(fleet.CronVulnerabilities), "vuln_processing_command")
				if err != nil {
					initFatal(err, "failed to release vuln processing lock")
				}
				cancel()
			}()

			logger := initLogger(cfg)
			logger = kitlog.With(logger, fleet.CronVulnerabilities)

			ctx := context.Background()
			appConfig, err := ds.AppConfig(ctx)
			if err != nil {
				initFatal(err, "error fetching app config during vulnerability processing")
			}
			vulnConfig := cfg.Vulnerabilities
			vulnPath := configureVulnPath(vulnConfig, appConfig, logger)
			// this really shouldn't ever be empty string since it's defaulted, but could be due to some misconfiguration
			// we'll throw an error here since the entire point of this command is to process vulnerabilities
			if vulnPath == "" {
				initFatal(errors.New("vuln path empty, check environment variables or app config yml"), "error during vulnerability processing")
			}
			level.Info(logger).Log("msg", "scanning vulnerabilities")
			err = scanVulnerabilities(ctx, ds, logger, &vulnConfig, appConfig, vulnPath)
			if err != nil {
				// errors during vuln processing should bubble up, so you know the job is failing without having to scour logs, e.g. non-zero exit code
				initFatal(fmt.Errorf("scanning vulnerabilities: %w", err), "error during vulnerability processing")
			}

			err = ds.SyncHostsSoftware(ctx, time.Now())
			if err != nil {
				// though vulnerability processing succeeded, we'll still fatally error here to indicate there was a problem
				initFatal(err, "failed to sync host software")
			}

			return
		},
	}
	vulnProcessingCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")

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
