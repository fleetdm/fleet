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

Navigate with ↑/↓, open an item with enter, refresh everything with r or just
the highlighted item with R. Press f for the
Focus view — an issue-centric card list of the work you've pinned, each showing
its project Status, linked PR, Claude session, and the next step to take.

Actions drive the development lifecycle:
  - w  start work: branch off main in a local clone, launch a Claude session,
       and set the issue's project Status to In progress
  - v  mark the issue In review · m merge (advances the issue to Awaiting QA)
  - M  merge + start a Claude cherry-pick session for the merged PR
  - a  mark Awaiting QA · p pin/unpin to Focus
  - b  open the selected issue's most recently updated project board

With primary_projects set in ~/.config/gm/jarvis/config.json (a list of project
numbers, gm aliases, or names, e.g. ["g-apple-at-work", "g-auto-patching"]), the
top "PROJECT VIEW" section groups by project: each project is a selectable row
(press b/enter to open its board) showing the issues assigned to you in any
status except Done / Ready for release, plus a count of unassigned issues in the
Ready column. Projects always appear even with no issues, so you can open one to
pick up new work. Managers of multiple teams can list several.

Sources:
  - pull requests you authored (mergeability, CI, review state)
  - pull requests awaiting your review
  - issues assigned to you, with their project board Status

Fetches are cached at ~/.config/gm/jarvis/cache.json; jarvis opens instantly
from a cache younger than 4h (press r to refresh, or pass --no-cache to force a
live pull on startup).

Local clones for "start work" are discovered under the directories in
~/.config/gm/jarvis/config.json (clone_base_dirs; defaults to ~/projects).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		repo, _ := cmd.Flags().GetString("repo")
		limit, _ := cmd.Flags().GetInt("limit")
		noCache, _ := cmd.Flags().GetBool("no-cache")
		return jarvis.Run(repo, limit, noCache)
	},
}

func init() {
	jarvisCmd.Flags().StringP("repo", "R", ghapi.DefaultRepo, "Repository to scan (owner/name)")
	jarvisCmd.Flags().IntP("limit", "l", 100, "Max items to fetch per source")
	jarvisCmd.Flags().Bool("no-cache", false, "Ignore the cached fetch and pull live on startup (data is cached for 4h by default; press r to refresh anytime)")
}
