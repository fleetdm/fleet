package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/seed"
)

func newVulnsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vulns",
		Short: "Seed vulnerable software rows directly into MySQL (bypasses scanners)",
		Long: `Vulnerabilities aren't seed-able via the Fleet API — they're derived from
the software inventory by background scanners. dibble shortcuts that pipeline
by writing rows directly to MySQL, the same approach the legacy
tools/software/vulnerabilities/seed_data tool used.

Requires direct access to the Fleet MySQL instance. The default DSN matches
the local docker-compose dev environment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dsn, _ := cmd.Flags().GetString("dsn")
			macos, _ := cmd.Flags().GetInt("macos")
			ubuntu, _ := cmd.Flags().GetInt("ubuntu")
			windows, _ := cmd.Flags().GetInt("windows")
			kernels, _ := cmd.Flags().GetInt("linux-kernels")
			res := seed.Vulns(context.Background(), seederLogger{}, seed.VulnsOptions{
				DSN:     dsn,
				MacOS:   macos,
				Ubuntu:  ubuntu,
				Windows: windows,
				Kernels: kernels,
			})
			printf("%s", res.Summary())
			return reportErrors(res.Errors)
		},
	}
	cmd.Flags().String("dsn", "fleet:insecure@tcp(localhost:3306)/fleet", "MySQL DSN")
	cmd.Flags().Int("macos", 0, "Number of macOS software rows to insert")
	cmd.Flags().Int("ubuntu", 0, "Number of Ubuntu software rows to insert")
	cmd.Flags().Int("windows", 0, "Number of Windows software rows to insert")
	cmd.Flags().Int("linux-kernels", 0, "Number of Linux kernel rows to insert")
	return cmd
}
