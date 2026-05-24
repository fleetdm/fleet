package command

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

func newReportsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reports",
		Aliases: []string{"queries"},
		Short:   "Seed saved reports (formerly 'queries')",
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
			res := seed.Reports(c, seederLogger{}, theme, teams, count)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().Int("count", 5, "How many global reports to seed (plus 1 per team)")
	return cmd
}
