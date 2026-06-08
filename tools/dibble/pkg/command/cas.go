package command

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

// newCAsCmd wires `dibble cas`. The seeder writes directly to MySQL,
// bypassing the service layer so we don't need a real SCEP / DigiCert /
// NDES / EST endpoint for URL validation. Names are prefixed with "*" so
// reviewers can tell at a glance that the CA is dibble-planted and won't
// actually issue certificates (encrypted secret columns are NULL).
func newCAsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cas",
		Short: "Seed fake certificate authorities directly into MySQL (non-idempotent)",
		Long: `Seed fake certificate authorities directly into MySQL. Each batch writes
one row for each of: custom_scep_proxy, custom_est_proxy, digicert, hydrant,
smallstep. A single NDES row is inserted at the start of the run (Fleet
hardcodes the NDES name and only allows one).

Encrypted secret columns are left NULL — the CAs list cleanly in the UI but
any request_certificate call against them will fail, which is the point.

Names are prefixed with "*" plus a per-run tag so seeded rows are obvious
and don't collide across runs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dsn, _ := cmd.Flags().GetString("dsn")
			count, _ := cmd.Flags().GetInt("count")
			res := seed.CAs(context.Background(), seederLogger{}, seed.CAOptions{
				DSN:   dsn,
				Count: count,
			})
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().String("dsn", "fleet:insecure@tcp(localhost:3306)/fleet", "MySQL DSN")
	cmd.Flags().Int("count", 1, "Number of batches; each batch writes one row per non-NDES CA type")
	return cmd
}
