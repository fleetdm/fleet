package commands

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch Vulnerability dictionary",
	Long:  `Fetch Vulnerability dictionary`,
}

func init() {
	RootCmd.AddCommand(fetchCmd)

	fetchCmd.PersistentFlags().Bool("no-details", false, "without vulnerability details")
	_ = viper.BindPFlag("no-details", fetchCmd.PersistentFlags().Lookup("no-details"))

	fetchCmd.PersistentFlags().Int("batch-size", 25, "The number of batch size to insert.")
	_ = viper.BindPFlag("batch-size", fetchCmd.PersistentFlags().Lookup("batch-size"))
}
