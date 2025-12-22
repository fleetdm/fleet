package main

import (
	"fmt"
	"sort"
	"strings"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

var roadmapCmd = &cobra.Command{
	Use:   "roadmap",
	Short: "Roadmap utilities",
}

var roadmapSyncEstimatesCmd = &cobra.Command{
	Use:   "sync-estimates",
	Short: "Sync estimates for issues on the Roadmap project from drafting/sprint projects or sub-issues",
	Run: func(cmd *cobra.Command, args []string) {
		roadmapProjectID := 87 // fleet Roadmap project

		issueNum, _ := cmd.Flags().GetInt("issue")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		sprintTitle, _ := cmd.Flags().GetString("sprint")

		// Define candidate source projects for estimates
		sources := ghapi.DefaultEstimateSourceProjects()

		var targets []int
		if issueNum > 0 {
			fmt.Printf("Checking Roadmap membership for #%d...\n", issueNum)
			targets = []int{issueNum}
			// Ensure the specified issue is on Roadmap
			if !ghapi.IsIssueInProject(issueNum, roadmapProjectID) {
				fmt.Printf("Error: issue #%d is not currently on the Roadmap project (%d)\n", issueNum, roadmapProjectID)
				return
			}
			fmt.Printf("Gathering estimates for #%d...\n", issueNum)
			// Optional sprint filter for single issue
			if sprintTitle != "" {
				fmt.Printf("Filtering by sprint '%s'...\n", sprintTitle)
				checkProjects := append(append([]int{}, sources...), roadmapProjectID)
				sprintSet, _ := ghapi.GetIssueNumbersForSprintAcrossProjects(checkProjects, sprintTitle, 2000)
				if _, ok := sprintSet[issueNum]; !ok {
					fmt.Printf("Issue #%d is not in sprint '%s' across known projects; skipping.\n", issueNum, sprintTitle)
					return
				}
				fmt.Printf("Issue #%d matched sprint '%s'.\n", issueNum, sprintTitle)
			}
		} else {
			fmt.Println("Finding Roadmap issues...")
			// Fetch all roadmap issues
			items, _, err := ghapi.GetProjectItemsWithTotal(roadmapProjectID, 1000)
			if err != nil {
				fmt.Printf("Failed to fetch roadmap items: %v\n", err)
				return
			}
			for _, it := range items {
				if it.Content.Number > 0 {
					targets = append(targets, it.Content.Number)
				}
			}
			fmt.Printf("Found %d Roadmap issues.\n", len(targets))

			// Apply sprint filter if provided
			if sprintTitle != "" {
				fmt.Printf("Filtering by sprint '%s'...\n", sprintTitle)
				// Build sprint set from source projects only to avoid re-querying Roadmap (we already have its items)
				sprintSet, _ := ghapi.GetIssueNumbersForSprintAcrossProjects(sources, sprintTitle, 2000)
				filtered := make([]int, 0, len(targets))
				for _, n := range targets {
					// include if in sources' sprint set or roadmap item shows matching sprint
					_, inSet := sprintSet[n]
					if inSet {
						filtered = append(filtered, n)
						continue
					}
					// check roadmap item sprint title from prefetched items
					for _, it := range items {
						if it.Content.Number == n && it.Sprint != nil && strings.EqualFold(strings.TrimSpace(it.Sprint.Title), strings.TrimSpace(sprintTitle)) {
							filtered = append(filtered, n)
							break
						}
					}
				}
				targets = filtered
				fmt.Printf("%d issues match sprint '%s'.\n", len(targets), sprintTitle)
			}
		}

		// Make iteration stable for output
		sort.Ints(targets)

		var (
			updated int
			skipped int
			errors  int
		)

		for _, n := range targets {
			fmt.Printf("\nGathering estimate for #%d...\n", n)
			// Skip if roadmap already has estimate and --all-issues not provided
			if !overwrite {
				if val, ok, _ := ghapi.GetEstimateFromProject(n, roadmapProjectID); ok && val > 0 {
					skipped++
					fmt.Printf("Skipping #%d: roadmap estimate already set to %d\n", n, val)
					continue
				}
			}

			// Primary: direct estimate from known projects
			est, src, _ := ghapi.GetEstimateForIssueAcrossProjects(n, sources)
			if est > 0 && src > 0 {
				fmt.Printf("Found estimate %d in project %d.\n", est, src)
			}
			// Secondary: sum of sub-issue estimates
			if est == 0 {
				related, _ := ghapi.GetRelatedIssueNumbers(n)
				if len(related) > 0 {
					fmt.Printf("Found %d issues tied to #%d...\n", len(related), n)
				} else {
					fmt.Printf("No direct estimate found; checking sub-issues for #%d...\n", n)
				}
				sum, _ := ghapi.SumEstimatesFromSubIssues(n, sources)
				est = sum
				src = 0 // aggregated
				if est > 0 {
					fmt.Printf("Using aggregated sub-issue estimate: %d.\n", est)
				}
			}
			if est == 0 {
				fmt.Printf("No estimate found for #%d; leaving roadmap unchanged\n", n)
				continue
			}

			if err := ghapi.SetEstimateInProject(n, roadmapProjectID, est); err != nil {
				errors++
				fmt.Printf("Failed to set roadmap estimate for #%d: %v\n", n, err)
				continue
			}
			updated++
			if src == 0 {
				fmt.Printf("Updated #%d: roadmap estimate set to %d (sum of sub-issues)\n", n, est)
			} else {
				fmt.Printf("Updated #%d: roadmap estimate set to %d (from project %d)\n", n, est, src)
			}
		}

		fmt.Printf("\nSummary: %d updated, %d skipped, %d errors\n", updated, skipped, errors)
	},
}

func init() {
	roadmapCmd.AddCommand(roadmapSyncEstimatesCmd)
	roadmapSyncEstimatesCmd.Flags().IntP("issue", "i", 0, "Only sync for the given issue number; must be on the Roadmap project")
	roadmapSyncEstimatesCmd.Flags().BoolP("overwrite", "o", false, "Overwrite existing Roadmap estimates (by default, issues with estimates are skipped)")
	roadmapSyncEstimatesCmd.Flags().StringP("sprint", "s", "", "Only process issues in this sprint (iteration title), e.g., 4.77.0")
}
