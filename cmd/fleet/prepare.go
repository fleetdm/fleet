package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/WatchBeam/clock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/fleet/server/config"
	"github.com/kolide/fleet/server/datastore/mysql"
	"github.com/kolide/fleet/server/kolide"
	"github.com/kolide/fleet/server/pubsub"
	"github.com/kolide/fleet/server/service"
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

	var dbCmd = &cobra.Command{
		Use:   "db",
		Short: "Given correct database configurations, prepare the databases for use",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			config := configManager.LoadConfig()
			ds, err := mysql.New(config.Mysql, clock.C)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			status, err := ds.MigrationStatus()
			if err != nil {
				initFatal(err, "retrieving migration status")
			}

			switch status {
			case kolide.AllMigrationsCompleted:
				fmt.Println("Migrations already completed. Nothing to do.")
				return

			case kolide.SomeMigrationsCompleted:
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

	prepareCmd.AddCommand(dbCmd)

	var testDataCmd = &cobra.Command{
		Use:   "test-data",
		Short: "Generate test data",
		Long:  ``,
		Run: func(cmd *cobra.Command, arg []string) {
			config := configManager.LoadConfig()
			ds, err := mysql.New(config.Mysql, clock.C)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			var (
				name     = "admin"
				username = "admin"
				password = "secret"
				email    = "admin@kolide.co"
				enabled  = true
				isAdmin  = true
			)
			admin := kolide.UserPayload{
				Name:     &name,
				Username: &username,
				Password: &password,
				Email:    &email,
				Enabled:  &enabled,
				Admin:    &isAdmin,
			}
			svc, err := service.NewService(ds, pubsub.NewInmemQueryResults(), kitlog.NewNopLogger(), config, nil, clock.C, nil, nil)
			if err != nil {
				initFatal(err, "creating service")
			}

			_, err = svc.NewAdminCreatedUser(context.Background(), admin)
			if err != nil {
				initFatal(err, "saving new user")
			}
		},
	}

	prepareCmd.AddCommand(testDataCmd)

	return prepareCmd

}
