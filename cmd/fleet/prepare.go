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
			cmd.Help() //nolint:errcheck
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

			ds, err := mysql.New(config.Mysql, clock.C)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			status, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}
			if status.StatusCode == fleet.NeedsFleetv4732Fix {
				if !noPrompt {
					printFleetv4732FixMessage()
					bufio.NewScanner(os.Stdin).Scan()
				}
				if err := ds.FixFleetv4732Migrations(cmd.Context()); err != nil {
					initFatal(err, "fixing v4.73.2 migrations")
				}
				// re-check status after fix
				status, err = ds.MigrationStatus(cmd.Context())
				if err != nil {
					initFatal(err, "retrieving migration status")
				}
			}

			switch status.StatusCode {
			case fleet.NoMigrationsCompleted:
				// OK
			case fleet.AllMigrationsCompleted:
				fmt.Println("Migrations already completed. Nothing to do.")
				return
			case fleet.SomeMigrationsCompleted:
				if !noPrompt {
					printMissingMigrationsPrompt(status.MissingTable, status.MissingData)
					bufio.NewScanner(os.Stdin).Scan()
				}
			case fleet.NeedsFleetv4732Fix, fleet.UnknownFleetv4732State:
				printFleetv4732UnknownStateMessage(status.StatusCode)
			case fleet.UnknownMigrations:
				printUnknownMigrationsMessage(status.UnknownTable, status.UnknownData)
				if dev {
					os.Exit(1)
				}
			}

			if err := ds.MigrateTables(cmd.Context()); err != nil {
				initFatal(err, "migrating db schema")
			}

			if err := ds.MigrateData(cmd.Context()); err != nil {
				initFatal(err, "migrating builtin data")
			}

			fmt.Println("Migrations completed.")
		},
	}

	dbCmd.PersistentFlags().BoolVar(&noPrompt, "no-prompt", false, "disable prompting before migrations (for use in scripts)")
	dbCmd.PersistentFlags().BoolVar(&dev, "dev", false, "Enable developer options")

	prepareCmd.AddCommand(dbCmd)
	return prepareCmd
}

func printUnknownMigrationsMessage(tables []int64, data []int64) {
	fmt.Printf("################################################################################\n"+
		"# WARNING:\n"+
		"#   Your Fleet database has unrecognized migrations. This could happen when\n"+
		"#   running an older version of Fleet on a newer migrated database.\n"+
		"#\n"+
		"#   Unknown migrations: %s.\n"+
		"################################################################################\n",
		tablesAndDataToString(tables, data))
}

func printMissingMigrationsPrompt(tables []int64, data []int64) {
	fmt.Printf("################################################################################\n"+
		"# WARNING:\n"+
		"#   This will perform Fleet database migrations. Please back up your data before\n"+
		"#   continuing.\n"+
		"#\n"+
		"#   Missing migrations: %s.\n"+
		"#\n"+
		"#   Press Enter to continue, or Control-c to exit.\n"+
		"################################################################################\n",
		tablesAndDataToString(tables, data))
}

func printFleetv4732FixMessage() {
	fmt.Printf("################################################################################\n" +
		"# WARNING:\n" +
		"#   Your Fleet database has misnumbered migrations introduced in some released\n" +
		"#   v4.73.2 artifacts. Fleet will automatically perform this fix prior to database\n" +
		"#   migrations. Please back up your data before continuing.\n" +
		"################################################################################\n")
}

func printFleetv4732UnknownStateMessage(statusCode fleet.MigrationStatusCode) {
	extra := "your Fleet database is in an unknown state."
	if statusCode == fleet.NeedsFleetv4732Fix {
		extra = "the automatic fix did not result in the expected state."
	}
	fmt.Print("################################################################################\n" +
		"# WARNING:\n" +
		"#   Your Fleet database has misnumbered migrations introduced in some released\n" +
		"#   v4.73.2 artifacts. Fleet attempts to fix this problem automatically, however\n" +
		"#  " + extra + "\n" +
		"#   Please contact Fleet support for assistance in resolving this.\n" +
		"################################################################################\n")
}

func tablesAndDataToString(tables, data []int64) string {
	switch {
	case len(tables) > 0 && len(data) == 0:
		// Most common case
		return fmt.Sprintf("tables=%v", tables)
	case len(tables) == 0 && len(data) == 0:
		return "unknown"
	case len(tables) == 0 && len(data) > 0:
		return fmt.Sprintf("data=%v", data)
	default:
		return fmt.Sprintf("tables=%v, data=%v", tables, data)
	}
}
