package main

import (
	"fmt"
	"strings"

	"fleetdm/gm/pkg/ghapi"
	"fleetdm/gm/pkg/util"

	"github.com/spf13/cobra"
)

// preSprintCmd is the parent command for pre-sprint utilities
var preSprintCmd = &cobra.Command{
	Use:   "pre-sprint",
	Short: "Pre-sprint utilities",
}

var (
	preSprintLimit  int
	preSprintFormat string
)

// preSprintReportCmd implements: gm pre-sprint report <project-id or alias[,alias...]>
var preSprintReportCmd = &cobra.Command{
	Use:   "report [project-id-or-alias]",
	Short: "Generate pre-sprint report for one or more teams",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		identList := strings.Split(args[0], ",")
		teamIDs := make([]int, 0, len(identList))
		teamLabels := make([]string, 0, len(identList))
		teamNames := make([]string, 0, len(identList))

		for _, ident := range identList {
			ident = strings.TrimSpace(ident)
			if ident == "" {
				continue
			}
			pid, err := ghapi.ResolveProjectID(ident)
			if err != nil {
				fmt.Printf("Skipping '%s': %v\n", ident, err)
				continue
			}
			label, ok := ghapi.ProjectLabels[pid]
			if !ok {
				fmt.Printf("Skipping '%s' (project %d): no drafting label mapping. Supported projects: mdm, soft, orch, sec.\n", ident, pid)
				continue
			}
			teamIDs = append(teamIDs, pid)
			teamLabels = append(teamLabels, label)
			teamNames = append(teamNames, ident)
		}

		if len(teamIDs) == 0 {
			return fmt.Errorf("no valid teams provided")
		}

		// Resolve status option names from the drafting project once
		draftingProjectID := ghapi.Aliases["draft"]
		readyName, err := ghapi.FindFieldValueByName(draftingProjectID, "Status", "ready to estimate")
		if err != nil {
			return fmt.Errorf("failed to resolve 'ready to estimate' status in drafting project: %v", err)
		}
		estimatedName, err := ghapi.FindFieldValueByName(draftingProjectID, "Status", "estimated")
		if err != nil {
			return fmt.Errorf("failed to resolve 'estimated' status in drafting project: %v", err)
		}

		stop := util.StartSpinner("Fetching drafting items")
		items, total, err := ghapi.GetProjectItemsWithTotal(draftingProjectID, preSprintLimit)
		stop()
		if err != nil {
			return fmt.Errorf("failed to get drafting project items: %v", err)
		}
		if total > preSprintLimit {
			fmt.Printf("Warning: Drafting project has %d items, but only fetched %d. Some items may be omitted.\n", total, preSprintLimit)
		}

		// If CSV format, print header once
		if strings.EqualFold(preSprintFormat, "csv") {
			fmt.Println("\nunestimated,total sp,priority sp,customer sp,priority-customer overlap")
		}

		// For each team, compute metrics from the same drafting items
		for idx := range teamIDs {
			teamLabel := strings.ToLower(strings.TrimSpace(teamLabels[idx]))
			teamName := teamNames[idx]

			// Accumulators
			unestimatedBugs := 0
			estBugPoints := 0
			priorityBugPoints := 0
			customerBugPoints := 0
			overlapPoints := 0

			for _, it := range items {
				// status filter
				if !util.StatusMatches(it.Status, readyName, estimatedName) {
					continue
				}
				// team label filter
				if !util.HasLabel(it.Labels, teamLabel) {
					continue
				}
				// bug filter
				isBug := util.HasLabel(it.Labels, "bug")
				if !isBug {
					continue
				}

				est := it.Estimate
				if est == 0 {
					unestimatedBugs++
				}
				estBugPoints += est

				isPriority := util.HasAnyLabel(it.Labels, "P0", "P1", "P2")
				isCustomer := util.HasLabelPrefix(it.Labels, "customer-")
				if isPriority {
					priorityBugPoints += est
				}
				if isCustomer {
					customerBugPoints += est
				}
				if isPriority && isCustomer {
					overlapPoints += est
				}
			}

			if strings.EqualFold(preSprintFormat, "csv") {
				fmt.Printf("%d,%d,%d,%d,%d\n", unestimatedBugs, estBugPoints, priorityBugPoints, customerBugPoints, overlapPoints)
			} else {
				// Default 'out' format
				fmt.Printf("\nPre-sprint report for %s\n", teamName)
				fmt.Printf("Total number of unestimated bugs: %d\n", unestimatedBugs)
				fmt.Printf("Total story points of estimated bugs: %d\n", estBugPoints)
				fmt.Printf("Total story points of priority bugs: %d\n", priorityBugPoints)
				fmt.Printf("Total story points of customer reported bugs: %d\n", customerBugPoints)
				fmt.Printf("Total overlap of priority and customer bugs: %d\n", overlapPoints)
			}
		}

		return nil
	},
}

func init() {
	preSprintCmd.AddCommand(preSprintReportCmd)
	preSprintReportCmd.Flags().IntVarP(&preSprintLimit, "limit", "l", 1000, "Maximum number of items to fetch from drafting project")
	preSprintReportCmd.Flags().StringVar(&preSprintFormat, "format", "out", "Output format: out (default) or csv")
}

// Helpers moved to pkg/util
