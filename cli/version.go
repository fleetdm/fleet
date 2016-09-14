package cli

import (
	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/version"
	"github.com/spf13/cobra"
)

func createVersionCmd(configManager config.Manager) *cobra.Command {
	// flags
	var (
		fFull bool
	)
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print kolide version",
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
