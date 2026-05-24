package command

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

func newPoliciesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policies",
		Short: "Seed global policies (and a couple per existing team)",
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
			res := seed.Policies(c, seederLogger{}, theme, teams, count)
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().Int("count", 5, "How many global policies to seed (plus 2 per team)")
	return cmd
}

// listExistingTeams asks Fleet for the team list so subcommands invoked on
// their own (without prior `dibble teams`) can still scope per-team work.
func listExistingTeams(c *Client) ([]seed.Team, error) {
	var resp struct {
		Teams []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"teams"`
		Fleets []struct {
			ID   uint   `json:"id"`
			Name string `json:"name"`
		} `json:"fleets"`
	}
	if err := c.Get("/api/latest/fleet/fleets?per_page=500", &resp); err != nil {
		return nil, err
	}
	out := make([]seed.Team, 0, len(resp.Teams)+len(resp.Fleets))
	for _, t := range resp.Teams {
		out = append(out, seed.Team{ID: t.ID, Name: t.Name})
	}
	for _, t := range resp.Fleets {
		out = append(out, seed.Team{ID: t.ID, Name: t.Name})
	}
	return out, nil
}
