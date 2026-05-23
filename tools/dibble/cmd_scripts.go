package main

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/seed"
)

func newScriptsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scripts",
		Short: "Seed saved scripts (.sh, .ps1, .zsh)",
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
			res := seed.Scripts(c, seederLogger{}, theme, teams, count)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().Int("count", 3, "How many global scripts to seed (plus 1 per team)")
	return cmd
}
