package main

import (
	"fmt"

	"fleetdm/gm/pkg/ghapi"
	"fleetdm/gm/pkg/tui"

	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project [project-id-or-alias]",
	Short: "Get GitHub issues from a project",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID, err := ghapi.ResolveProjectID(args[0])
		if err != nil {
			fmt.Printf("Error resolving project: %v\n", err)
			return
		}

		limit, err := cmd.Flags().GetInt("limit")
		if err != nil {
			fmt.Printf("Error getting limit flag: %v\n", err)
			return
		}

		allIssues, _ := cmd.Flags().GetBool("all-issues")
		workflow, _ := cmd.Flags().GetString("workflow")

		tui.RunTUI(tui.ProjectCommand, projectID, limit, "", allIssues, workflow)
	},
}

func init() {
	projectCmd.Flags().IntP("limit", "l", 300, "Maximum number of items to fetch")
	projectCmd.Flags().BoolP("all-issues", "a", false, "Select all issues once the view is populated")
	projectCmd.Flags().StringP("workflow", "w", "", "Run this workflow immediately instead of waiting for input (e.g. 'demo')")
	sprintCmd.Flags().IntP("limit", "l", 300, "Maximum number of items to fetch")
	sprintCmd.Flags().BoolP("previous", "p", false, "Show previous sprint instead of current")
}

// sprintCmd fetches only the issues currently in the active sprint for a project.
// Usage mirrors the project command but filters to items whose sprint field matches the
// current iteration (using the already implemented @current logic when setting sprint).
var sprintCmd = &cobra.Command{
	Use:   "sprint [project-id-or-alias]",
	Short: "Get GitHub issues in the current sprint for a project (use -p for previous sprint)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID, err := ghapi.ResolveProjectID(args[0])
		if err != nil {
			fmt.Printf("Error resolving project: %v\n", err)
			return
		}

		limit, err := cmd.Flags().GetInt("limit")
		if err != nil {
			fmt.Printf("Error getting limit flag: %v\n", err)
			return
		}

		prev, _ := cmd.Flags().GetBool("previous")
		if prev {
			// Pass a mode hint via the search parameter
			tui.RunTUI(tui.SprintCommand, projectID, limit, "previous", false, "")
			return
		}

		tui.RunTUI(tui.SprintCommand, projectID, limit, "", false, "")
	},
}
