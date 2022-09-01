package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/spf13/cobra"
)

func createPrepareCmd(configManager config.Manager) *cobra.Command {
	prepareCmd := &cobra.Command{
		Use:   "prepare",
		Short: "Subcommands for initializing Fleet infrastructure",
		Long: `
Subcommands for initializing Fleet infrastructure

To setup Fleet infrastructure, use one of the available commands.
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	noPrompt := false
	// Whether to enable developer options
	dev := false

	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Given correct database configurations, prepare the databases for use",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()

			if dev {
				applyDevFlags(&config)
				noPrompt = true
			}

			ds, err := mysql.New(config.Mysql, clock.C,
				mysql.WithMDMApple(config.MDMApple.Enable),
				// Multi-statements is required for applying MDM Apple schemas.
				mysql.WithMultiStatements(config.MDMApple.Enable),
			)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			// TODO(lucas): Add check to fail the command if config.MDMApple.Enable and
			// the MySQL version is < 8.0.19.

			status, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			prepareMigrationStatusCheck(status, noPrompt, dev, config.Mysql.Database)

			if err := ds.MigrateTables(cmd.Context()); err != nil {
				initFatal(err, "migrating db schema")
			}

			if err := ds.MigrateData(cmd.Context()); err != nil {
				initFatal(err, "migrating builtin data")
			}

			// TODO(lucas): Due to table name collisions, the Apple MDM tables are created
			// on a separate database. Revisit.
			if config.MDMApple.Enable {
				status, err := ds.MigrationMDMAppleStatus(cmd.Context())
				if err != nil {
					initFatal(err, "retrieving migration status")
				}
				prepareMigrationStatusCheck(status, noPrompt, dev, config.Mysql.DatabaseMDMApple)
				if err := ds.MigrateMDMAppleTables(cmd.Context()); err != nil {
					initFatal(err, "migrating mdm apple db schema")
				}
				if err := ds.MigrateMDMAppleData(cmd.Context()); err != nil {
					initFatal(err, "migrating mdmd apple builtin data")
				}
			}

			fmt.Println("Migrations completed.")
		},
	}

	dbCmd.PersistentFlags().BoolVar(&noPrompt, "no-prompt", false, "disable prompting before migrations (for use in scripts)")
	dbCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")

	prepareCmd.AddCommand(dbCmd)

	return prepareCmd
}

func prepareMigrationStatusCheck(status *fleet.MigrationStatus, noPrompt, dev bool, dbName string) {
	switch status.StatusCode {
	case fleet.NoMigrationsCompleted:
		// OK
	case fleet.AllMigrationsCompleted:
		fmt.Printf("Migrations already completed for %q. Nothing to do.\n", dbName)
		return
	case fleet.SomeMigrationsCompleted:
		if !noPrompt {
			fmt.Printf("################################################################################\n"+
				"# WARNING:\n"+
				"#   This will perform %q database migrations. Please back up your data before\n"+
				"#   continuing.\n"+
				"#\n"+
				"#   Missing migrations: tables=%v, data=%v.\n"+
				"#\n"+
				"#   Press Enter to continue, or Control-c to exit.\n"+
				"################################################################################\n",
				dbName, status.MissingTable, status.MissingData)
			bufio.NewScanner(os.Stdin).Scan()
		}
	case fleet.UnknownMigrations:
		fmt.Printf("################################################################################\n"+
			"# WARNING:\n"+
			"#   Your %q database has unrecognized migrations. This could happen when\n"+
			"#   running an older version of Fleet on a newer migrated database.\n"+
			"#\n"+
			"#   Unknown migrations: tables=%v, data=%v.\n"+
			"################################################################################\n",
			dbName, status.UnknownTable, status.UnknownData)
		if dev {
			os.Exit(1)
		}
	}
}
