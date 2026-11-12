package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
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
	// Whether to show table stats before and after the migration
	showTableStats := false

	dbCmd := &cobra.Command{
		Use:   "db",
		Short: "Given correct database configurations, prepare the databases for use",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			config, err := configManager.LoadConfig()
			if err != nil {
				initFatal(err, "loading config")
			}

			if dev_mode.IsEnabled {
				applyDevFlags(&config)
				noPrompt = true
			}

			// migrations issue a prepared statement per row, the path most exposed to 1615
			ds, err := mysql.New(config.Mysql, clock.C, mysql.WithReprepareRetry())
			if err != nil {
				initFatal(err, "creating db connection")
			}

			status, err := ds.MigrationStatus(cmd.Context())
			if err != nil {
				initFatal(err, "retrieving migration status")
			}
			if showTableStats {
				defer func() {
					stats, err := ds.GetTableRowCounts(cmd.Context())
					if err != nil {
						initFatal(err, "getting table stats")
					}
					statsAsJSON, err := json.Marshal(stats)
					if err != nil {
						initFatal(err, "encoding table row counts to JSON")
					}
					fmt.Printf("Table Row Counts: %s\n", statsAsJSON)
				}()
			}

			switch status.StatusCode {
			case fleet.NoMigrationsCompleted:
				// OK
			case fleet.AllMigrationsCompleted:
				fmt.Println("Migrations already completed. Nothing to do.")
				return
			case fleet.SomeMigrationsCompleted:
				if !noPrompt {
					printMissingMigrationsPrompt(status.MissingTable)
					bufio.NewScanner(os.Stdin).Scan()
				}
			case fleet.UnknownMigrations:
				printUnknownMigrationsMessage(status.UnknownTable)
				if dev_mode.IsEnabled {
					os.Exit(1)
				}
			}

			if err := ds.MigrateTables(cmd.Context()); err != nil {
				initFatal(err, "migrating db schema")
			}

			fmt.Println("Migrations completed.")
		},
	}

	dbCmd.PersistentFlags().BoolVar(&noPrompt, "no-prompt", false, "disable prompting before migrations (for use in scripts)")
	dbCmd.PersistentFlags().BoolVar(&dev_mode.IsEnabled, "dev", false, "Enable developer options")
	dbCmd.PersistentFlags().BoolVar(&showTableStats, "with-table-stats", false, "Show approximate table row counts after migrations")

	prepareCmd.AddCommand(dbCmd)
	return prepareCmd
}

func printUnknownMigrationsMessage(tables []int64) {
	fmt.Printf("################################################################################\n"+
		"# WARNING:\n"+
		"#   Your Fleet database has unrecognized migrations. This could happen when\n"+
		"#   running an older version of Fleet on a newer migrated database.\n"+
		"#\n"+
		"#   Unknown migrations: %s.\n"+
		"################################################################################\n",
		migrationsToString(tables))
}

func printMissingMigrationsPrompt(tables []int64) {
	fmt.Printf("################################################################################\n"+
		"# WARNING:\n"+
		"#   This will perform Fleet database migrations. Please back up your data before\n"+
		"#   continuing.\n"+
		"#\n"+
		"#   Missing migrations: %s.\n"+
		"#\n"+
		"#   Press Enter to continue, or Control-c to exit.\n"+
		"################################################################################\n",
		migrationsToString(tables))
}

func migrationsToString(tables []int64) string {
	if len(tables) == 0 {
		return "unknown"
	}
	return fmt.Sprintf("tables=%v", tables)
}
