package main

import (
	"fleetdm/gm/pkg/ghapi"
	"fleetdm/gm/pkg/jarvis"

	"github.com/spf13/cobra"
)

var jarvisCmd = &cobra.Command{
	Use:   "jarvis",
	Short: "Personal work dashboard — your GitHub work, sorted by leverage",
	Long: `jarvis is an interactive TUI that aggregates your open GitHub work into
leverage-ordered buckets: what's blocking others, what you can merge right now,
what needs your hands, your review queue, and what's gone cold.

It is read-only against GitHub — navigate with ↑/↓, open an item in the browser
with enter, and refresh with r.

Sources (v1):
  - pull requests you authored (mergeability, CI, review state)
  - pull requests awaiting your review
  - issues assigned to you`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		limit, _ := cmd.Flags().GetInt("limit")
		return jarvis.Run(repo, limit)
	},
}

func init() {
	jarvisCmd.Flags().StringP("repo", "R", ghapi.DefaultRepo, "Repository to scan (owner/name)")
	jarvisCmd.Flags().IntP("limit", "l", 100, "Max items to fetch per source")
}
