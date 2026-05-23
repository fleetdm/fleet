package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/seed"
	"github.com/fleetdm/fleet/v4/tools/dibble/themes"
)

// runAll executes every seeder in dependency order with the given counts.
// Used by both `dibble all` and the wizard's "everything" preset.
type allCounts struct {
	Users    int
	Teams    int
	Policies int
	Reports  int
	Labels   int
	Scripts  int
	Profiles int
	Software int
	CAs      int

	// EnrollSecrets is per-team and binary (one per team or none). True to
	// rotate every team's secret; false to leave them alone.
	EnrollSecrets bool
}

func defaultAllCounts() allCounts {
	return allCounts{
		Users:         5,
		Teams:         3,
		Policies:      5,
		Reports:       5,
		Labels:        5,
		Scripts:       3,
		Profiles:      4,
		Software:      5,
		CAs:           0,
		EnrollSecrets: true,
	}
}

func runAll(c *Client, theme themes.Theme, counts allCounts) error {
	start := time.Now()
	log := seederLogger{}

	// Teams first — many other seeders want a team list.
	teams, tRes := seed.Teams(c, log, theme, counts.Teams)
	printf("%s", tRes.Summary())

	uRes := seed.Users(c, log, theme, counts.Users)
	printf("%s", uRes.Summary())

	if counts.EnrollSecrets {
		esRes := seed.EnrollSecrets(c, log, teams)
		printf("%s", esRes.Summary())
	}

	lRes := seed.Labels(c, log, theme, counts.Labels)
	printf("%s", lRes.Summary())

	pRes := seed.Policies(c, log, theme, teams, counts.Policies)
	printf("%s", pRes.Summary())

	rRes := seed.Reports(c, log, theme, teams, counts.Reports)
	printf("%s", rRes.Summary())

	scRes := seed.Scripts(c, log, theme, teams, counts.Scripts)
	printf("%s", scRes.Summary())

	prRes := seed.Profiles(c, log, theme, teams, counts.Profiles)
	printf("%s", prRes.Summary())

	soRes := seed.Software(c, log, theme, teams, counts.Software)
	printf("%s", soRes.Summary())

	if counts.CAs > 0 {
		caRes := seed.CAs(nil, log, counts.CAs)
		printf("%s", caRes.Summary())
	}

	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Fprintf(stdoutish, "\n🌱  dibble done in %s (theme: %s).\n", elapsed, theme.Name)
	return nil
}

func newAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "all",
		Short: "Seed every entity with sensible defaults (idempotent)",
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
			counts := defaultAllCounts()
			if v, _ := cmd.Flags().GetInt("users"); v > 0 {
				counts.Users = v
			}
			if v, _ := cmd.Flags().GetInt("teams"); v > 0 {
				counts.Teams = v
			}
			return runAll(c, theme, counts)
		},
	}
	cmd.Flags().Int("users", 0, "Override user count (0 = default)")
	cmd.Flags().Int("teams", 0, "Override team count (0 = default)")
	return cmd
}
