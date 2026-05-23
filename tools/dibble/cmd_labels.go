package main

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/seed"
)

func newLabelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "labels",
		Short: "Seed dynamic (query-based) labels",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}
			c, err := newClientFromViper()
			if err != nil {
				return err
			}
			theme, err := currentTheme()
			if err != nil {
				return err
			}
			count, _ := cmd.Flags().GetInt("count")
			res := seed.Labels(c, seederLogger{}, theme, count)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().Int("count", 5, "How many labels to seed")
	return cmd
}
