package command

import (
	"github.com/spf13/cobra"

	"github.com/fleetdm/fleet/v4/tools/dibble/pkg/seed"
)

func newCAsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cas",
		Short: "Seed Certificate Authorities (placeholder — mock CA path is a v1 TODO)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}
			c, err := newClientFromViper()
			if err != nil {
				return err
			}
			_ = c
			count, _ := cmd.Flags().GetInt("count")
			res := seed.CAs(nil, seederLogger{}, count)
			printf("%s", res.Summary())
			return nil
		},
	}
	cmd.Flags().Int("count", 1, "How many CAs to seed")
	return cmd
}
