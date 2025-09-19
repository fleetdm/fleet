package main

import (
	"fmt"

	"fleetdm/gm/pkg/tui"

	"github.com/spf13/cobra"
)

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Get GitHub issues",
	Run: func(cmd *cobra.Command, args []string) {
		search, err := cmd.Flags().GetString("search")
		if err != nil {
			fmt.Printf("Error getting search flag: %v\n", err)
			return
		}
		tui.RunTUI(tui.IssuesCommand, 0, 0, search)
	},
}

func init() {
	issuesCmd.Flags().StringP("search", "s", "", "Search for issues by github search syntax")
}
