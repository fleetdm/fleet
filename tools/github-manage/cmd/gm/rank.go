package main

import (
	"fmt"
	"strings"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

var rankCmd = &cobra.Command{
	Use:   "rank [project-id-or-alias]",
	Short: "Rank issues in each status column by handbook priority labels",
	Long: `Reorders items within each status column of a GitHub project board using the
handbook issue prioritization (P0/P1/P2, ~customer promise, ~activation-blocker,
reliability, bugs, ~product-maturity, then the rest). Sub-issues are placed
directly below their parent when it is in the same column, and inherit the
parent's priority when it is elsewhere in the project. Only items that are out
of order are moved. Views with an explicit sort applied ignore manual order.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectID, err := ghapi.ResolveProjectID(args[0])
		if err != nil {
			fmt.Printf("Error resolving project: %v\n", err)
			return
		}

		limit, _ := cmd.Flags().GetInt("limit")
		statuses, _ := cmd.Flags().GetStringSlice("status")
		dryRun, _ := cmd.Flags().GetBool("dry-run")

		items, total, err := ghapi.GetProjectItemsWithTotal(projectID, limit)
		if err != nil {
			fmt.Printf("Error fetching project items: %v\n", err)
			return
		}
		if total > len(items) {
			fmt.Printf("Warning: project has %d items but only %d were fetched; unfetched items will not be ranked. Raise --limit to include them.\n", total, len(items))
		}

		parents, err := ghapi.GetProjectItemParents(projectID)
		if err != nil {
			fmt.Printf("Warning: could not fetch sub-issue parents, ranking by labels only: %v\n", err)
			parents = nil
		}

		moves := ghapi.PlanColumnRanking(items, statuses, parents)
		if dryRun {
			printRankPlan(items, statuses, parents, moves)
			return
		}
		if len(moves) == 0 {
			fmt.Println("All columns already ranked, nothing to move.")
			return
		}

		labelsByNumber := ghapi.LabelsByNumber(items)
		fmt.Printf("Moving %d of %d items...\n", len(moves), len(items))
		moved, err := ghapi.RankStatusColumnsByPriority(projectID, items, statuses, parents, func(done, totalMoves int, m ghapi.RankMove) {
			labels := ghapi.EffectiveRankLabels(m.Item, parents, labelsByNumber)
			fmt.Printf("[%d/%d] #%d %s (%s)\n", done, totalMoves, m.Item.Content.Number, m.Item.Title, ghapi.HandbookPriorityBucketName(labels))
		})
		if err != nil {
			fmt.Printf("Error after %d moves: %v\n", moved, err)
			return
		}
		fmt.Printf("Done. Moved %d items.\n", moved)
	},
}

// printRankPlan shows the desired order per column, marking items that would
// move and nesting sub-issues under their in-column parent.
func printRankPlan(items []ghapi.ProjectItem, statuses []string, parents map[int]int, moves []ghapi.RankMove) {
	moving := make(map[string]bool, len(moves))
	for _, m := range moves {
		moving[m.Item.ID] = true
	}
	labelsByNumber := ghapi.LabelsByNumber(items)

	byStatus := make(map[string][]ghapi.ProjectItem)
	var order []string
	for _, it := range items {
		if !statusSelected(it.Status, statuses) {
			continue
		}
		if _, seen := byStatus[it.Status]; !seen {
			order = append(order, it.Status)
		}
		byStatus[it.Status] = append(byStatus[it.Status], it)
	}

	for _, status := range order {
		column := byStatus[status]
		inColumn := make(map[int]bool, len(column))
		for _, it := range column {
			inColumn[it.Content.Number] = true
		}
		desired := ghapi.DesiredColumnOrder(column, parents, labelsByNumber)
		name := status
		if name == "" {
			name = "(no status)"
		}
		fmt.Printf("\n%s\n", name)
		for _, it := range desired {
			marker := " "
			if moving[it.ID] {
				marker = "*"
			}
			nest := ""
			if parentNum, ok := parents[it.Content.Number]; ok && inColumn[parentNum] {
				nest = "└ "
			}
			labels := ghapi.EffectiveRankLabels(it, parents, labelsByNumber)
			fmt.Printf("  %s #%-6d %-18s %s%s\n", marker, it.Content.Number, ghapi.HandbookPriorityBucketName(labels), nest, it.Title)
		}
	}
	fmt.Printf("\n%d items would move (marked *).\n", len(moves))
}

func statusSelected(status string, statuses []string) bool {
	if len(statuses) == 0 {
		return true
	}
	for _, s := range statuses {
		if strings.Contains(strings.ToLower(status), strings.ToLower(strings.TrimSpace(s))) {
			return true
		}
	}
	return false
}

func init() {
	rankCmd.Flags().IntP("limit", "l", 300, "Maximum number of items to fetch")
	rankCmd.Flags().StringSliceP("status", "s", nil, "Only rank columns whose status matches (substring, repeatable)")
	rankCmd.Flags().BoolP("dry-run", "n", false, "Show the planned order without moving anything")
}
