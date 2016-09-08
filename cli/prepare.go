package cli

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	prepareCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(prepareCmd)
	rootCmd.AddCommand(testDataCmd)
}

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
		connString := fmt.Sprintf(
			"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
			viper.GetString("mysql.username"),
			viper.GetString("mysql.password"),
			viper.GetString("mysql.address"),
			viper.GetString("mysql.database"),
		)
		ds, err := datastore.New("gorm-mysql", connString)
		if err != nil {
			logrus.WithError(err).Fatal("error creating db connection")
		}
		if err := ds.Drop(); err != nil {
			logrus.WithError(err).Fatal("error dropping db tables")
		}

		if err := ds.Migrate(); err != nil {
			logrus.WithError(err).Fatal("error setting up db schema")
		}
	},
}

var testDataCmd = &cobra.Command{
	Use:   "test-data",
	Short: "Generate test data",
	Long:  ``,
	Run: func(cmd *cobra.Command, arg []string) {
		connString := fmt.Sprintf(
			"%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local",
			viper.GetString("mysql.username"),
			viper.GetString("mysql.password"),
			viper.GetString("mysql.address"),
			viper.GetString("mysql.database"),
		)
		ds, err := datastore.New("gorm-mysql", connString)
		if err != nil {
			logrus.WithError(err).Fatal("error creating db connection")
		}

		admin, err := kolide.NewUser("admin", "admin", "admin@kolide.co", true, false)
		if err != nil {
			logrus.WithError(err).Fatal("Could not create new user object")
		}
		_, err = ds.NewUser(admin)
		if err != nil {
			logrus.WithError(err).Fatal("Could not create new user in the database")
		}
	},
}
