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
	},
}

func init() {
	projectCmd.Flags().IntP("limit", "l", 100, "Maximum number of items to fetch")
	estimatedCmd.Flags().IntP("limit", "l", 500, "Maximum number of items to fetch from drafting project")
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
	},
}
