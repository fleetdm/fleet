package main

import (
	"context"
	"fmt"
	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/cobra"
	"os"
	"time"
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
				fmt.Println("database check complete, continuing vuln processing")
			case fleet.NoMigrationsCompleted, fleet.SomeMigrationsCompleted, fleet.UnknownMigrations:
				initFatal(fmt.Errorf("database migrations incompatible with current version"), "refusing to continue processing vulnerabilities")
			}

			ctx := context.Background()

			logger := setupLogger(cfg)

			appConfig, err := ds.AppConfig(ctx)
			vulnConfig := cfg.Vulnerabilities
			var vulnPath string
			switch {
			case vulnConfig.DatabasesPath != "" && appConfig.VulnerabilitySettings.DatabasesPath != "":
				vulnPath = vulnConfig.DatabasesPath
				level.Info(logger).Log(
					"msg", "fleet config takes precedence over app config when both are configured",
					"databases_path", vulnPath,
				)
			case vulnConfig.DatabasesPath != "":
				vulnPath = vulnConfig.DatabasesPath
			case appConfig.VulnerabilitySettings.DatabasesPath != "":
				vulnPath = appConfig.VulnerabilitySettings.DatabasesPath
			default:
				level.Info(logger).Log("msg", "vulnerability scanning not configured, vulnerabilities databases path is empty")
			}
			if vulnPath != "" {
				level.Info(logger).Log("msg", "scanning vulnerabilities")
				if err := scanVulnerabilities(ctx, ds, logger, &vulnConfig, appConfig, vulnPath); err != nil {
					initFatal(fmt.Errorf("scanning vulnerabilities: %w", err), "error during vulnerability processing")
				}
			}

			err = ds.SyncHostsSoftware(ctx, time.Now())
			if err != nil {
				level.Error(logger).Log("msg", "failed to sync host software", "err", err)
			}

			return
		},
	}
	vulnProcessingCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")

	return vulnProcessingCmd
}

func setupLogger(cfg config.FleetConfig) kitlog.Logger {
	var logger kitlog.Logger
	{
		output := os.Stderr
		if cfg.Logging.JSON {
			logger = kitlog.NewJSONLogger(output)
		} else {
			logger = kitlog.NewLogfmtLogger(output)
		}
		if cfg.Logging.Debug {
			logger = level.NewFilter(logger, level.AllowDebug())
		} else {
			logger = level.NewFilter(logger, level.AllowInfo())
		}
		logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC)
	}
	return logger
}
