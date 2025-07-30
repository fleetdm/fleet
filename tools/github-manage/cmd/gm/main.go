package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gm",
		Short: "GitHub Manage CLI",
		Long:  "A CLI tool to manage GitHub repositories and workflows.",
		Run: func(cmd *cobra.Command, args []string) {
			// Placeholder for the default command behavior
			log.Println("Welcome to GitHub Manage CLI!")
		},
	}

	rootCmd.AddCommand(issuesCmd)
	rootCmd.AddCommand(projectCmd)

	// Example of adding a subcommand
	rootCmd.AddCommand(&cobra.Command{
		Use:   "example",
		Short: "An example subcommand",
		Run: func(cmd *cobra.Command, args []string) {
			p := tea.NewProgram(initialModel{})
			if err := p.Start(); err != nil {
				log.Fatalf("Error running Bubble Tea program: %v", err)
			}
		},
	})

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}

// Bubble Tea model
type initialModel struct{}

func (m initialModel) Init() tea.Cmd {
	return nil
}

func (m initialModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m initialModel) View() string {
	return "Hello from Bubble Tea!\nPress Ctrl+C to exit."
}
