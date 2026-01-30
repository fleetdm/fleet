package main

import (
	"fmt"
	"sort"
	"strings"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

var releasesCmd = &cobra.Command{
	Use:   "releases",
	Short: "Releases utilities",
}

var releasesSyncEstimatesCmd = &cobra.Command{
	Use:   "sync-estimates",
	Short: "Sync estimates for issues on the Releases project from drafting/sprint projects or sub-issues",
	Long: `Sync estimates for issues on the Releases project from drafting/sprint projects or sub-issues.

This command copies estimate values from source projects (drafting, product group projects) to the
Releases project. If no direct estimate is found, it sums estimates from sub-issues.

Usage modes:
  1. Sync all Releases issues:
     gm releases sync-estimates

  2. Sync issues from a specific milestone:
     gm releases sync-estimates --milestone "Fleet 4.77.0"

  3. Sync a single issue:
     gm releases sync-estimates --issue 12345

  4. Sync a single issue in a milestone (validates milestone membership):
     gm releases sync-estimates --issue 12345 --milestone "Fleet 4.77.0"

By default, issues with existing estimates are skipped. Use --overwrite to update them.`,
	Run: func(cmd *cobra.Command, args []string) {
		releasesProjectID := 87 // fleet Releases project

		issueNum, _ := cmd.Flags().GetInt("issue")
		overwrite, _ := cmd.Flags().GetBool("overwrite")
		milestoneTitle, _ := cmd.Flags().GetString("milestone")

		// Define candidate source projects for estimates
		sources := ghapi.DefaultEstimateSourceProjects()

		var targets []int
		if issueNum > 0 {
			fmt.Printf("Checking Releases membership for #%d...\n", issueNum)
			targets = []int{issueNum}
			// Ensure the specified issue is on Releases
			if !ghapi.IsIssueInProject(issueNum, releasesProjectID) {
				fmt.Printf("Error: issue #%d is not currently on the Releases project (%d)\n", issueNum, releasesProjectID)
				return
			}
			fmt.Printf("Gathering estimates for #%d...\n", issueNum)
			// Optional milestone filter for single issue
			if milestoneTitle != "" {
				fmt.Printf("Filtering by milestone '%s'...\n", milestoneTitle)
				milestoneIssues, err := ghapi.GetIssuesByMilestone(milestoneTitle, 2000)
				if err != nil {
					fmt.Printf("Failed to get issues for milestone '%s': %v\n", milestoneTitle, err)
					return
				}
				found := false
				for _, num := range milestoneIssues {
					if num == issueNum {
						found = true
						break
					}
				}
				if !found {
					fmt.Printf("Issue #%d is not in milestone '%s'; skipping.\n", issueNum, milestoneTitle)
					return
				}
				fmt.Printf("Issue #%d matched milestone '%s'.\n", issueNum, milestoneTitle)
			}
		} else if milestoneTitle != "" {
			// When milestone is specified, fetch milestone issues first
			fmt.Printf("Finding issues in milestone '%s'...\n", milestoneTitle)
			milestoneIssues, err := ghapi.GetIssuesByMilestone(milestoneTitle, 2000)
			if err != nil {
				fmt.Printf("Failed to get issues for milestone '%s': %v\n", milestoneTitle, err)
				return
			}
			fmt.Printf("Found %d issues in milestone '%s'.\n", len(milestoneIssues), milestoneTitle)

			// Fetch Releases project items to build a set for efficient filtering
			fmt.Println("Fetching Releases project items...")
			items, _, err := ghapi.GetProjectItemsWithTotal(releasesProjectID, 1000)
			if err != nil {
				fmt.Printf("Failed to fetch releases items: %v\n", err)
				return
			}

			// Build set of issue numbers on Releases project
			releasesSet := make(map[int]struct{})
			for _, it := range items {
				if it.Content.Number > 0 {
					releasesSet[it.Content.Number] = struct{}{}
				}
			}

			// Filter milestone issues to only those on Releases project
			fmt.Println("Filtering to issues on Releases project...")
			for _, num := range milestoneIssues {
				if _, onReleases := releasesSet[num]; onReleases {
					targets = append(targets, num)
				}
			}
			fmt.Printf("Found %d issues from milestone '%s' on Releases project.\n", len(targets), milestoneTitle)
		} else {
			// No milestone specified, fetch all releases issues
			fmt.Println("Finding all Releases issues...")
			items, _, err := ghapi.GetProjectItemsWithTotal(releasesProjectID, 1000)
			if err != nil {
				fmt.Printf("Failed to fetch releases items: %v\n", err)
				return
			}
			for _, it := range items {
				if it.Content.Number > 0 {
					targets = append(targets, it.Content.Number)
				}
			}
			fmt.Printf("Found %d Releases issues.\n", len(targets))
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
			// Skip if releases already has estimate and --all-issues not provided
			if !overwrite {
				if val, ok, _ := ghapi.GetEstimateFromProject(n, releasesProjectID); ok && val > 0 {
					skipped++
					fmt.Printf("Skipping #%d: releases estimate already set to %d\n", n, val)
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
				fmt.Printf("No estimate found for #%d; leaving releases unchanged\n", n)
				continue
			}

			if err := ghapi.SetEstimateInProject(n, releasesProjectID, est); err != nil {
				errors++
				fmt.Printf("Failed to set releases estimate for #%d: %v\n", n, err)
				continue
			}
			updated++
			if src == 0 {
				fmt.Printf("Updated #%d: releases estimate set to %d (sum of sub-issues)\n", n, est)
			} else {
				fmt.Printf("Updated #%d: releases estimate set to %d (from project %d)\n", n, est, src)
			}
		}

		fmt.Printf("\nSummary: %d updated, %d skipped, %d errors\n", updated, skipped, errors)
	},
}

var releasesForecastCmd = &cobra.Command{
	Use:   "forecast",
	Short: "Calculate effort forecast for a milestone based on t-shirt sizes",
	Long: `Calculate effort forecast for a milestone based on t-shirt sizes.

This command retrieves all issues in a milestone on the Releases project, maps their
t-shirt sizes to numeric values, and provides a total forecast.

Size mapping:
  XXS: 3
  XS:  8
  S:   25
  M:   50
  L:   75
  XL:  100

Usage:
  gm releases forecast --milestone "Fleet 4.83.0"`,
	Run: func(cmd *cobra.Command, args []string) {
		releasesProjectID := 87 // fleet Releases project
		milestoneTitle, _ := cmd.Flags().GetString("milestone")

		if milestoneTitle == "" {
			fmt.Println("Error: --milestone flag is required")
			return
		}

		// Size to numeric value mapping
		sizeMap := map[string]int{
			"XXS": 3,
			"XS":  8,
			"S":   25,
			"M":   50,
			"L":   75,
			"XL":  100,
		}

		// Fetch milestone issues
		fmt.Printf("Finding issues in milestone '%s'...\n", milestoneTitle)
		milestoneIssues, err := ghapi.GetIssuesByMilestone(milestoneTitle, 2000)
		if err != nil {
			fmt.Printf("Failed to get issues for milestone '%s': %v\n", milestoneTitle, err)
			return
		}
		fmt.Printf("Found %d issues in milestone '%s'.\n", len(milestoneIssues), milestoneTitle)

		// Fetch Releases project items
		fmt.Println("Fetching Releases project items...")
		items, _, err := ghapi.GetProjectItemsWithTotal(releasesProjectID, 1000)
		if err != nil {
			fmt.Printf("Failed to fetch releases items: %v\n", err)
			return
		}

		// Build map of issue number to project item ID for items on Releases project
		issueToItemID := make(map[int]string)
		for _, it := range items {
			if it.Content.Number > 0 {
				issueToItemID[it.Content.Number] = it.ID
			}
		}

		// Calculate forecast for milestone issues on Releases project
		total := 0
		counted := 0
		missing := 0
		sizeBreakdown := make(map[string]int)

		fmt.Println("\nProcessing issues...")
		for _, num := range milestoneIssues {
			itemID, exists := issueToItemID[num]
			if !exists {
				missing++
				continue
			}

			// Get T-shirt size field value for this item
			size, err := ghapi.GetProjectItemFieldValue(itemID, releasesProjectID, "T-shirt size")
			if err != nil || size == "" {
				missing++
				continue
			}

			if value, ok := sizeMap[size]; ok {
				total += value
				counted++
				sizeBreakdown[size]++
				fmt.Printf("  #%d: %s (%d)\n", num, size, value)
			} else {
				fmt.Printf("  #%d: unknown size '%s' (skipped)\n", num, size)
				missing++
			}
		}

		// Display results
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Printf("Milestone: %s\n", milestoneTitle)
		fmt.Printf("Total issues in milestone: %d\n", len(milestoneIssues))
		fmt.Printf("Issues on Releases project with valid sizes: %d\n", counted)
		fmt.Printf("Issues without size or not on Releases: %d\n", missing)
		fmt.Println("\nSize breakdown:")
		for _, size := range []string{"XXS", "XS", "S", "M", "L", "XL"} {
			if count, ok := sizeBreakdown[size]; ok {
				fmt.Printf("  %s: %d issue(s) = %d points\n", size, count, count*sizeMap[size])
			}
		}
		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Printf("TOTAL FORECAST: %d points\n", total)
		fmt.Println(strings.Repeat("=", 50))
	},
}

func init() {
	releasesCmd.AddCommand(releasesSyncEstimatesCmd)
	releasesCmd.AddCommand(releasesForecastCmd)

	releasesSyncEstimatesCmd.Flags().IntP("issue", "i", 0, "Only sync for the given issue number; must be on the Releases project")
	releasesSyncEstimatesCmd.Flags().BoolP("overwrite", "o", false, "Overwrite existing Releases estimates (by default, issues with estimates are skipped)")
	releasesSyncEstimatesCmd.Flags().StringP("milestone", "m", "", "Only process issues in this milestone, e.g., Fleet 4.77.0")

	releasesForecastCmd.Flags().StringP("milestone", "m", "", "Milestone to forecast, e.g., Fleet 4.83.0 (required)")
}
