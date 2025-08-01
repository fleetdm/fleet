package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var issuesCmd = &cobra.Command{
	Use:   "issues",
	Short: "Get GitHub issues",
	Run: func(cmd *cobra.Command, args []string) {
		model := initializeModel()
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
