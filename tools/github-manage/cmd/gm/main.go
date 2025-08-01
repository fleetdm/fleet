package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

// promptToContinue asks the user if they want to continue or quit
func promptToContinue() bool {
	fmt.Printf("\nPress Enter to continue, or type 'q' to quit: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if strings.ToLower(input) == "q" || strings.ToLower(input) == "quit" {
		fmt.Println("Exiting test...")
		return false
	}
	return true
}

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

	// Test command to test SetCurrentSprint functionality
	rootCmd.AddCommand(&cobra.Command{
		Use:   "test",
		Short: "Test SetCurrentSprint functionality",
		Run: func(cmd *cobra.Command, args []string) {
			testIssueNumber := 31541
			mdmProjectID := 58 // MDM project

			fmt.Printf("=== Testing SetCurrentSprint with Issue #%d ===\n\n", testIssueNumber)

			// First, let's see what fields are available in the project
			fmt.Printf("Fetching available fields in MDM project (%d)...\n", mdmProjectID)
			fields, err := ghapi.GetProjectFields(mdmProjectID)
			if err != nil {
				log.Printf("❌ Error fetching project fields: %v", err)
				return
			}

			fmt.Printf("✅ Available fields in project %d:\n", mdmProjectID)
			for name, field := range fields {
				fmt.Printf("  - Name: '%s', Type: '%s', ID: '%s'\n", name, field.Type, field.ID)
			}
			fmt.Printf("\n")

			// Test SetCurrentSprint
			fmt.Printf("Setting current sprint for issue #%d in MDM project (%d)...\n", testIssueNumber, mdmProjectID)
			err = ghapi.SetCurrentSprint(testIssueNumber, mdmProjectID)
			if err != nil {
				log.Printf("❌ Error setting current sprint: %v", err)
			} else {
				fmt.Printf("✅ Successfully set current sprint\n")
			}

			fmt.Printf("\n=== Test Complete ===\n")
		},
	})

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
