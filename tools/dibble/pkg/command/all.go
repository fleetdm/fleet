package command

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/themes"
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

	// ActivityBatches drives the direct-MySQL activity seeder. Zero skips
	// it entirely — activities need a DSN, not just an API token, so we
	// don't run them by default.
	ActivityBatches int
	ActivityDSN     string
	ActivityHostID  uint

	// IDPUserCount / IDPHostCount drive the direct-MySQL IDP seeder. Zero
	// for either skips it entirely (it needs a DSN like activities).
	IDPUserCount int
	IDPHostCount int
	IDPDSN       string

	// CAsDSN is the MySQL DSN used by the CA seeder. CAs > 0 also writes
	// directly to MySQL, bypassing the service layer's URL validation.
	CAsDSN string
}

func defaultAllCounts() allCounts {
	return allCounts{
		Users:           5,
		Teams:           3,
		Policies:        5,
		Reports:         5,
		Labels:          5,
		Scripts:         3,
		Profiles:        4,
		Software:        5,
		CAs:             0,
		EnrollSecrets:   true,
		ActivityBatches: 0,
		ActivityDSN:     "fleet:insecure@tcp(localhost:3306)/fleet",
		ActivityHostID:  1,
		IDPUserCount:    0,
		IDPHostCount:    0,
		IDPDSN:          "fleet:insecure@tcp(localhost:3306)/fleet",
		CAsDSN:          "fleet:insecure@tcp(localhost:3306)/fleet",
	}
}

func runAll(c *Client, theme themes.Theme, counts allCounts) error {
	start := time.Now()
	log := seederLogger{}

	// Collect every seeder's errors so `dibble all` exits non-zero when any
	// step failed. Without this, CI / scripted runs treat a half-broken run
	// as success.
	var allErrs []error
	report := func(res seed.Result) {
		printf("%s", res.Summary())
		allErrs = append(allErrs, res.Errors...)
	}

	// Teams first — many other seeders want a team list.
	teams, tRes := seed.Teams(c, log, theme, counts.Teams)
	report(tRes)

	report(seed.Users(c, log, theme, counts.Users))

	if counts.EnrollSecrets {
		report(seed.EnrollSecrets(c, log, teams))
	}

	report(seed.Labels(c, log, theme, counts.Labels))
	report(seed.Policies(c, log, theme, teams, counts.Policies))
	report(seed.Reports(c, log, theme, teams, counts.Reports))
	report(seed.Scripts(c, log, theme, teams, counts.Scripts))
	report(seed.Profiles(c, log, theme, teams, counts.Profiles))

	if counts.Software > 0 {
		// `dibble all` lands software under the first existing team if any,
		// otherwise no team. Use `dibble software` directly for team
		// overrides — this path is intentionally simple.
		swOpt := seed.SoftwareOptions{MaintainedAppCount: counts.Software}
		if len(teams) > 0 {
			swOpt.TeamID = teams[0].ID
		}
		report(seed.SoftwareCustom(c, log, swOpt))
		report(seed.SoftwareMaintained(c, log, swOpt))
	}

	if counts.CAs > 0 {
		report(seed.CAs(context.Background(), log, seed.CAOptions{
			DSN:   counts.CAsDSN,
			Count: counts.CAs,
		}))
	}

	if counts.ActivityBatches > 0 {
		report(seed.Activities(context.Background(), log, seed.ActivitiesOptions{
			DSN:     counts.ActivityDSN,
			HostID:  counts.ActivityHostID,
			Batches: counts.ActivityBatches,
		}))
	}

	if counts.IDPUserCount > 0 {
		report(seed.IDP(context.Background(), c, log, seed.IDPOptions{
			DSN:       counts.IDPDSN,
			UserCount: counts.IDPUserCount,
			HostCount: counts.IDPHostCount,
		}))
	}

	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Fprintf(stdoutish, "\n🌱  dibble done in %s (theme: %s).\n", elapsed, theme.Name)
	return reportErrors(allErrs)
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
			if v, _ := cmd.Flags().GetInt("activity-batches"); v > 0 {
				counts.ActivityBatches = v
			}
			if v, _ := cmd.Flags().GetString("activity-dsn"); v != "" {
				counts.ActivityDSN = v
			}
			if v, _ := cmd.Flags().GetUint("activity-host-id"); v > 0 {
				counts.ActivityHostID = v
			}
			if v, _ := cmd.Flags().GetInt("idp-user-count"); v > 0 {
				counts.IDPUserCount = v
			}
			if v, _ := cmd.Flags().GetInt("idp-host-count"); v >= 0 && cmd.Flags().Changed("idp-host-count") {
				counts.IDPHostCount = v
			}
			if v, _ := cmd.Flags().GetString("idp-dsn"); v != "" {
				counts.IDPDSN = v
			}
			if v, _ := cmd.Flags().GetInt("cas"); v > 0 {
				counts.CAs = v
			}
			if v, _ := cmd.Flags().GetString("cas-dsn"); v != "" {
				counts.CAsDSN = v
			}
			return runAll(c, theme, counts)
		},
	}
	cmd.Flags().Int("users", 0, "Override user count (0 = default)")
	cmd.Flags().Int("teams", 0, "Override team count (0 = default)")
	cmd.Flags().Int("activity-batches", 0, "Seed N batches of fake activities via direct MySQL (0 = skip)")
	cmd.Flags().String("activity-dsn", "", "MySQL DSN for the activity seeder (default fleet:insecure@tcp(localhost:3306)/fleet)")
	cmd.Flags().Uint("activity-host-id", 0, "host_id used by the activity seeder for host-scoped rows (default 1)")
	cmd.Flags().Int("idp-user-count", 0, "Seed N users with IDP accounts via direct MySQL (0 = skip)")
	cmd.Flags().Int("idp-host-count", 5, "How many hosts to assign IDP accounts to (round-robin); only used when --idp-user-count > 0")
	cmd.Flags().String("idp-dsn", "", "MySQL DSN for the IDP seeder (default fleet:insecure@tcp(localhost:3306)/fleet)")
	cmd.Flags().Int("cas", 0, "Seed N batches of fake certificate authorities via direct MySQL (0 = skip)")
	cmd.Flags().String("cas-dsn", "", "MySQL DSN for the CA seeder (default fleet:insecure@tcp(localhost:3306)/fleet)")
	return cmd
}
