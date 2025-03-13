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
