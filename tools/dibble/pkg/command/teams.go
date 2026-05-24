package command

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

func newTeamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "teams",
		Aliases: []string{"fleets"},
		Short:   "Seed teams (aka fleets)",
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
			_, res := seed.Teams(c, seederLogger{}, theme, count)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().Int("count", 3, "How many teams to seed")
	return cmd
}
