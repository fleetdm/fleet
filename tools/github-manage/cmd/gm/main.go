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

	// Test command to verify all workflow functionality
	rootCmd.AddCommand(&cobra.Command{
		Use:   "test",
		Short: "Test all GitHub project management workflows",
		Run: func(cmd *cobra.Command, args []string) {
			testIssueNumber := 31541
			mdmProjectID := 58   // MDM project
			draftProjectID := 67 // Drafting project

			fmt.Printf("=== Testing GitHub Manager Workflows with Issue #%d ===\n\n", testIssueNumber)

			// Step 1: Add issue to MDM project (58)
			fmt.Printf("1. Adding issue #%d to MDM project (%d)...\n", testIssueNumber, mdmProjectID)
			err := ghapi.AddIssueToProject(testIssueNumber, mdmProjectID)
			if err != nil {
				log.Printf("❌ Error adding issue to MDM project: %v", err)
			} else {
				fmt.Printf("✅ Successfully added issue to MDM project\n")
			}

			if !promptToContinue() {
				return
			}

			// Step 2: Set estimate to 5 in MDM project
			fmt.Printf("2. Setting estimate to 5 for issue #%d in MDM project...\n", testIssueNumber)
			// First get the project item ID
			itemID, err := ghapi.GetProjectItemID(testIssueNumber, mdmProjectID)
			if err != nil {
				log.Printf("❌ Error getting project item ID: %v", err)
			} else {
				err = ghapi.SetProjectItemFieldValue(itemID, mdmProjectID, "Estimate", "5")
				if err != nil {
					log.Printf("❌ Error setting estimate: %v", err)
				} else {
					fmt.Printf("✅ Successfully set estimate to 5\n")
				}
			}

			if !promptToContinue() {
				return
			}

			// Step 3: Add issue to drafting project (67)
			fmt.Printf("3. Adding issue #%d to drafting project (%d)...\n", testIssueNumber, draftProjectID)
			err = ghapi.AddIssueToProject(testIssueNumber, draftProjectID)
			if err != nil {
				log.Printf("❌ Error adding issue to drafting project: %v", err)
			} else {
				fmt.Printf("✅ Successfully added issue to drafting project\n")
			}

			if !promptToContinue() {
				return
			}

			// Step 4: Sync estimate from MDM to drafting project
			fmt.Printf("4. Syncing estimate from MDM project to drafting project...\n")
			err = ghapi.SyncEstimateField(testIssueNumber, mdmProjectID, draftProjectID)
			if err != nil {
				log.Printf("❌ Error syncing estimate: %v", err)
			} else {
				fmt.Printf("✅ Successfully synced estimate\n")
			}

			if !promptToContinue() {
				return
			}

			// Step 5: Set status to 'ready' in MDM project
			fmt.Printf("5. Setting status to 'ready' for issue #%d in MDM project...\n", testIssueNumber)
			err = ghapi.SetIssueStatus(testIssueNumber, mdmProjectID, "ready")
			if err != nil {
				log.Printf("❌ Error setting status: %v", err)
			} else {
				fmt.Printf("✅ Successfully set status to 'ready'\n")
			}

			if !promptToContinue() {
				return
			}

			// Step 6: Remove issue from drafting project
			fmt.Printf("6. Removing issue #%d from drafting project (%d)...\n", testIssueNumber, draftProjectID)
			err = ghapi.RemoveIssueFromProject(testIssueNumber, draftProjectID)
			if err != nil {
				log.Printf("❌ Error removing issue from drafting project: %v", err)
			} else {
				fmt.Printf("✅ Successfully removed issue from drafting project\n")
			}

			if !promptToContinue() {
				return
			}

			// Bonus: Test project fields retrieval
			fmt.Printf("7. Testing project fields retrieval for MDM project...\n")
			fields, err := ghapi.GetProjectFields(mdmProjectID)
			if err != nil {
				log.Printf("❌ Error fetching project fields: %v", err)
			} else {
				fmt.Printf("✅ Successfully fetched %d project fields\n", len(fields))
				fmt.Printf("Available fields: ")
				fieldNames := make([]string, 0, len(fields))
				for name := range fields {
					fieldNames = append(fieldNames, name)
				}
				fmt.Printf("%v\n", fieldNames)
			}

			if !promptToContinue() {
				return
			}

			// Bonus: Show cache statistics
			fmt.Printf("8. Cache statistics...\n")
			stats := ghapi.GetCacheStats()
			fmt.Printf("✅ Cache statistics:\n")
			for key, value := range stats {
				fmt.Printf("  - %s: %v\n", key, value)
			}

			fmt.Printf("\n=== Test Complete ===\n")
			fmt.Printf("Note: All functions now use actual GitHub CLI commands.\n")
			fmt.Printf("All successful operations indicate the underlying 'gh' command executed correctly.\n")
		},
	})

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %v", err)
	}
}
