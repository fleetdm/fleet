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
	var prepareCmd = &cobra.Command{
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

	var dbCmd = &cobra.Command{
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

			status, err := ds.MigrationStatus()
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			switch status {
			case fleet.AllMigrationsCompleted:
				fmt.Println("Migrations already completed. Nothing to do.")
				return

			case fleet.SomeMigrationsCompleted:
				if !noPrompt {
					fmt.Printf("################################################################################\n" +
						"# WARNING:\n" +
						"#   This will perform Fleet database migrations. Please back up your data before\n" +
						"#   continuing.\n" +
						"#\n" +
						"#   Press Enter to continue, or Control-c to exit.\n" +
						"################################################################################\n")
					bufio.NewScanner(os.Stdin).Scan()
				}
			}

			if err := ds.MigrateTables(); err != nil {
				initFatal(err, "migrating db schema")
			}

			if err := ds.MigrateData(); err != nil {
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
