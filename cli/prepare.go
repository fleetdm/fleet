package cli

import (
	"github.com/WatchBeam/clock"
	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide/server/config"
	"github.com/kolide/kolide/server/datastore/mysql"
	"github.com/kolide/kolide/server/kolide"
	"github.com/kolide/kolide/server/pubsub"
	"github.com/kolide/kolide/server/service"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

func createPrepareCmd(configManager config.Manager) *cobra.Command {

	var prepareCmd = &cobra.Command{
		Use:   "prepare",
		Short: "Subcommands for initializing kolide infrastructure",
		Long: `
Subcommands for initializing kolide infrastructure

To setup kolide infrastructure, use one of the available commands.
`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

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

			if err := ds.MigrateTables(); err != nil {
				initFatal(err, "migrating db schema")
			}

			if err := ds.MigrateData(); err != nil {
				initFatal(err, "migrating builtin data")
			}
		},
	}

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
			svc, err := service.NewService(ds, pubsub.NewInmemQueryResults(), kitlog.NewNopLogger(), config, nil, clock.C)
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
