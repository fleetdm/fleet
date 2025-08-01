package main

import (
	"log"

	"fleetdm/gm/pkg/ghapi"

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
	rootCmd.AddCommand(estimatedCmd)

	// Example of adding a subcommand
	rootCmd.AddCommand(&cobra.Command{
		Use:   "test",
		Short: "An example subcommand",
		Run: func(cmd *cobra.Command, args []string) {
			fields, err := ghapi.GetProjectFields(58)
			if err != nil {
				log.Fatalf("Error fetching project fields: %v", err)
			}
			log.Printf("Project fields: %+v", fields)
		},
	})

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
