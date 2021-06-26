package main

import (
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/kolide/kit/version"
	"github.com/spf13/cobra"
)

func createVersionCmd(configManager config.Manager) *cobra.Command {
	// flags
	var (
		fFull bool
	)
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print Fleet version",
		Long: `
Print version information and related build info`,
		Run: func(cmd *cobra.Command, args []string) {
			if fFull {
				version.PrintFull()
				return
			}
			version.Print()
		},
	}

	versionCmd.PersistentFlags().BoolVar(&fFull, "full", false, "print full version information")

	return versionCmd
}
