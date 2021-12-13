package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/cobra"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Launch is the entrypoint that sets up and runs the Fleet commands.
func main() {
	rootCmd := createRootCmd()

	configManager := config.NewManager(rootCmd)

	rootCmd.AddCommand(createPrepareCmd(configManager))
	rootCmd.AddCommand(createServeCmd(configManager))
	rootCmd.AddCommand(createConfigDumpCmd(configManager))
	rootCmd.AddCommand(createVersionCmd(configManager))

	if err := rootCmd.Execute(); err != nil {
		initFatal(err, "running root command")
	}
}

// initFatal prints an error message and exits with a non-zero status.
func initFatal(err error, message string) {
	fmt.Printf("Failed to start: %s: %v\n", message, err)
	os.Exit(1)
}

func createRootCmd() *cobra.Command {
	// rootCmd represents the base command when called without any subcommands
	rootCmd := &cobra.Command{
		Use:   "fleet",
		Short: "osquery management and orchestration",
		Long: `Fleet server (https://fleetdm.com)

Configurable Options:

Options may be supplied in a yaml configuration file or via environment
variables. You only need to define the configuration values for which you
wish to override the default value.
`,
	}

	rootCmd.PersistentFlags().StringP("config", "c", "", "Path to a configuration file")

	return rootCmd
}

func applyDevFlags(cfg *config.FleetConfig) {
	cfg.Mysql.Username = "fleet"
	cfg.Mysql.Database = "fleet"
	cfg.Mysql.Password = "insecure"
	cfg.Session.Duration = 7 * 24 * time.Hour
}
