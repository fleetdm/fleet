package cli

import (
	"fmt"
	"os"

	"github.com/kolide/kolide-ose/config"
	"github.com/spf13/cobra"
)

// Launch is the entrypoint that sets up and runs the Kolide commands.
func Launch() {
	rootCmd := createRootCmd()

	configManager := config.NewManager(rootCmd)

	rootCmd.AddCommand(createPrepareCmd(configManager))
	rootCmd.AddCommand(createServeCmd(configManager))
	rootCmd.AddCommand(createConfigDumpCmd(configManager))

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func createRootCmd() *cobra.Command {
	// RootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   "kolide",
		Short: "osquery management and orchestration",
		Long: `
osquery management and orchestration

Configurable Options:

Options may be supplied in a yaml configuration file or via environment
variables. You only need to define the configuration values for which you
wish to override the default value.
`,
	}

	rootCmd.PersistentFlags().String("config", "", "Path to a configuration file")

	return rootCmd
}
