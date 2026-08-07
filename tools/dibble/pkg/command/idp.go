package command

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

// newIDPCmd wires `dibble idp` — populates mdm_idp_accounts for the most
// recently seeded users and assigns those accounts round-robin to the first
// N hosts. Like vulns and activities, this seeder writes directly to MySQL
// (those tables have no public API).
func newIDPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "idp",
		Short: "Create IDP accounts for seeded users and assign them to hosts (direct MySQL)",
		Long: `Plants rows in mdm_idp_accounts (one per seeded user, matched by email)
and host_mdm_idp_accounts (round-robin assignment to existing hosts).

Users are fetched via the Fleet API sorted by created_at DESC so dibble-seeded
users surface ahead of the bootstrap admin. Hosts are fetched via the Fleet
API — enroll some first with osquery-perf (try ` + "`dibble hosts`" + `).

Requires direct MySQL access; the default DSN matches the local docker-compose
dev environment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}
			c, err := newClientFromViper()
			if err != nil {
				return err
			}
			dsn, _ := cmd.Flags().GetString("dsn")
			users, _ := cmd.Flags().GetInt("user-count")
			hosts, _ := cmd.Flags().GetInt("host-count")
			res := seed.IDP(context.Background(), c, seederLogger{}, seed.IDPOptions{
				DSN:       dsn,
				UserCount: users,
				HostCount: hosts,
			})
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().String("dsn", "fleet:insecure@tcp(localhost:3306)/fleet", "MySQL DSN")
	cmd.Flags().Int("user-count", 3, "How many seeded users to give an IDP account")
	cmd.Flags().Int("host-count", 5, "How many hosts to assign IDP accounts to (round-robin)")
	return cmd
}
