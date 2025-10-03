package main

import (
	"fmt"

	"fleetdm/gm/pkg/ghapi"

	tea "github.com/charmbracelet/bubbletea"
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

		model := initializeModelForProject(projectID, limit)
		p := tea.NewProgram(&model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running Bubble Tea program: %v\n", err)
		}
		if model.exitMessage != "" {
			fmt.Println(model.exitMessage)
		}
	},
}

func init() {
	projectCmd.Flags().IntP("limit", "l", 100, "Maximum number of items to fetch")
	estimatedCmd.Flags().IntP("limit", "l", 500, "Maximum number of items to fetch from drafting project")
	sprintCmd.Flags().IntP("limit", "l", 100, "Maximum number of items to fetch")
}

var estimatedCmd = &cobra.Command{
	Use:   "estimated [project-id-or-alias]",
	Short: "Get estimated GitHub issues from the drafting project filtered by project label",
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

		model := initializeModelForEstimated(projectID, limit)
		p := tea.NewProgram(&model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running Bubble Tea program: %v\n", err)
		}
		if model.exitMessage != "" {
			fmt.Println(model.exitMessage)
		}
	},
}

// sprintCmd fetches only the issues currently in the active sprint for a project.
// Usage mirrors the project command but filters to items whose sprint field matches the
// current iteration (using the already implemented @current logic when setting sprint).
var sprintCmd = &cobra.Command{
	Use:   "sprint [project-id-or-alias]",
	Short: "Get GitHub issues in the current sprint for a project",
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

		model := initializeModelForSprint(projectID, limit)
		p := tea.NewProgram(&model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running Bubble Tea program: %v\n", err)
		}
		if model.exitMessage != "" {
			fmt.Println(model.exitMessage)
		}
	},
}
