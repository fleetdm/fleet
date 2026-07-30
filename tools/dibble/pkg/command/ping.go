package command

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Verify dibble can reach the Fleet API",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := requireConfig(); err != nil {
				return err
			}
			c, err := newClientFromViper()
			if err != nil {
				return err
			}
			var resp struct {
				Version string `json:"version"`
				Branch  string `json:"branch"`
				Build   string `json:"build"`
			}
			if err := c.Get("/api/latest/fleet/version", &resp); err != nil {
				return err
			}
			fmt.Printf("Fleet %s (branch %s, build %s) — connection OK 🌱\n", resp.Version, resp.Branch, resp.Build)
			return nil
		},
	}
}
