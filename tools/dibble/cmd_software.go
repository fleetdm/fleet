package main

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/seed"
)

func newSoftwareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "software",
		Short: "Seed software titles (custom-package upload is a v1 TODO)",
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
			teams, _ := listExistingTeams(c)
			res := seed.Software(c, seederLogger{}, theme, teams, count)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().Int("count", 5, "How many software titles to seed")
	return cmd
}
