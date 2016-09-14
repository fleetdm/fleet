package cli

import (
	"fmt"

	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/spf13/cobra"
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
			connString := fmt.Sprintf(
				"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
				config.Mysql.Username,
				config.Mysql.Password,
				config.Mysql.Address,
				config.Mysql.Database,
			)
			ds, err := datastore.New("gorm-mysql", connString)
			if err != nil {
				initFatal(err, "creating db connection")
			}
			if err := ds.Drop(); err != nil {
				initFatal(err, "dropping db tables")
			}

			if err := ds.Migrate(); err != nil {
				initFatal(err, "migrating db schema")
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
			connString := fmt.Sprintf(
				"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
				config.Mysql.Username,
				config.Mysql.Password,
				config.Mysql.Address,
				config.Mysql.Database,
			)
			ds, err := datastore.New("gorm-mysql", connString)
			if err != nil {
				initFatal(err, "creating db connection")
			}

			admin, err := kolide.NewUser("admin", "admin", "admin@kolide.co", true, false, config.Auth.BcryptCost)
			if err != nil {
				initFatal(err, "creating new user")
			}
			_, err = ds.NewUser(admin)
			if err != nil {
				initFatal(err, "saving new user")
			}
		},
	}

	prepareCmd.AddCommand(testDataCmd)

	return prepareCmd

}
