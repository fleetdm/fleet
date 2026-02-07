package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

// hasLabel checks if an issue currently has the specified label (case-insensitive)
func hasLabel(issue ghapi.Issue, labelName string) bool {
	for _, label := range issue.Labels {
		if strings.EqualFold(label.Name, labelName) {
			return true
		}
	}
	return false
}

var issuesWithLabelCmd = &cobra.Command{
	Use:   "issues-with-label [label]",
	Short: "Find issues created since a date that had a specific label at any point",
	Long:  "Finds all issues created since a given date that had a given label at any point (not just currently).",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		label := args[0]
		since, err := cmd.Flags().GetString("since")
		if err != nil {
			fmt.Printf("Error getting since flag: %v\n", err)
			return
		}
		if since == "" {
			fmt.Fprintln(os.Stderr, "Error: --since flag is required (format: YYYY-MM-DD)")
			return
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		if concurrency < 1 {
			concurrency = 10
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Starting search for issues created since %s that had label '%s' at any point...\n", since, label)
			fmt.Fprintf(os.Stderr, "Using %d concurrent workers\n\n", concurrency)
		} else {
			fmt.Fprintf(os.Stderr, "Searching for issues created since %s that had label '%s' at any point...\n\n", since, label)
		}

		issues, err := ghapi.GetIssuesCreatedSinceWithLabel(since, label, verbose, concurrency)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
			return
		}

		if len(issues) == 0 {
			fmt.Fprintln(os.Stderr, "No issues found matching the criteria.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Number\tTitle\tState\tCreated at\tNow?")
		for _, issue := range issues {
			hasLabelNow := "❌"
			if hasLabel(issue, label) {
				hasLabelNow = "✔️"
			}
			fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\n", issue.Number, issue.Title, issue.State, issue.CreatedAt, hasLabelNow)
		}
		w.Flush()

		fmt.Fprintf(os.Stderr, "\nFound %d issue(s)\n", len(issues))
	},
}

func init() {
	issuesWithLabelCmd.Flags().StringP("since", "s", "", "Date to search from (format: YYYY-MM-DD)")
	issuesWithLabelCmd.Flags().BoolP("verbose", "v", false, "Show verbose output as issues are pulled and evaluated")
	issuesWithLabelCmd.Flags().IntP("concurrency", "c", 10, "Number of concurrent workers for timeline requests")
}
