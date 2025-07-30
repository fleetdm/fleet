package main

import (
	"fmt"

	"fleetdm/gm/pkg/ghapi"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Get GitHub issues",
	Run: func(cmd *cobra.Command, args []string) {
		items, err := ghapi.GetProjectItems(58, 100)
		if err != nil {
			fmt.Printf("Error fetching issues: %v\n", err)
			return
		}
		issues := ghapi.ConvertItemsToIssues(items)
		if err != nil {
			fmt.Printf("Error fetching issues: %v\n", err)
			return
		}
		model := initializeModel(issues)
		p := tea.NewProgram(model)
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
