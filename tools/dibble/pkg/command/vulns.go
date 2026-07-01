package command

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

func newVulnsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vulns",
		Short: "Seed software rows directly into MySQL for the vuln scanner to chew on",
		Long: `Writes rows directly into the software table so Fleet's background
vulnerability scanner has inventory to process. Each row gets a
fleet-compatible checksum, so re-runs are idempotent against the unique
software-checksum index.

NOTE: this only writes the software table. It does NOT create hosts,
host_software entries, or software_cpe associations — vulnerabilities won't
surface against any host until those rows exist (via real ingest or a
follow-up seeder). Use this when you need plausible inventory volume; use
osquery-perf or the legacy seed_vuln_data tool when you need end-to-end
vulnerable-host scenarios.

Requires direct access to the Fleet MySQL instance. The default DSN matches
the local docker-compose dev environment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dsn, _ := cmd.Flags().GetString("dsn")
			macos, _ := cmd.Flags().GetInt("macos")
			ubuntu, _ := cmd.Flags().GetInt("ubuntu")
			windows, _ := cmd.Flags().GetInt("windows")
			res := seed.Vulns(context.Background(), seederLogger{}, seed.VulnsOptions{
				DSN:     dsn,
				MacOS:   macos,
				Ubuntu:  ubuntu,
				Windows: windows,
			})
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().String("dsn", "fleet:insecure@tcp(localhost:3306)/fleet", "MySQL DSN")
	cmd.Flags().Int("macos", 0, "Number of macOS software rows to insert")
	cmd.Flags().Int("ubuntu", 0, "Number of Ubuntu software rows to insert")
	cmd.Flags().Int("windows", 0, "Number of Windows software rows to insert")
	return cmd
}
