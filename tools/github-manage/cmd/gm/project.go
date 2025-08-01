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

		items, err := ghapi.GetProjectItems(projectID, limit)
		if err != nil {
			fmt.Printf("Error fetching issues: %v\n", err)
			return
		}
		issues := ghapi.ConvertItemsToIssues(items)
		if err != nil {
			fmt.Printf("Error converting issues: %v\n", err)
			return
		}
		model := initializeModelWithIssues(issues)
		p := tea.NewProgram(&model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running Bubble Tea program: %v\n", err)
		}
		fmt.Printf("Selected issues: ")
		for i := range model.selected {
			if i < len(model.choices) {
				fmt.Printf("#%d ", model.choices[i].Number)
			}
		}
	},
}

func init() {
	projectCmd.Flags().IntP("limit", "l", 100, "Maximum number of items to fetch (default: 100)")
}

var estimatedCmd = &cobra.Command{
	Use:   "estimated",
	Short: "Get GitHub issues",
	Run: func(cmd *cobra.Command, args []string) {
		items, err := ghapi.GetMDMTicketsEstimated()
		if err != nil {
			fmt.Printf("Error fetching issues: %v\n", err)
			return
		}
		issues := ghapi.ConvertItemsToIssues(items)
		if err != nil {
			fmt.Printf("Error fetching issues: %v\n", err)
			return
		}
		model := initializeModelWithIssues(issues)
		p := tea.NewProgram(&model)
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error running Bubble Tea program: %v\n", err)
		}
		fmt.Printf("Selected issues: ")
		for i := range model.selected {
			if i < len(model.choices) {
				fmt.Printf("#%d ", model.choices[i].Number)
			}
		}
	},
}
