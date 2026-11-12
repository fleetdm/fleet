package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

var issuesWithHistoricalLabelCmd = &cobra.Command{
	Use:   "issues-with-historical-label [label]",
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

		repo, _ := cmd.Flags().GetString("repo")
		olderThan, _ := cmd.Flags().GetInt("older-than")

		if verbose {
			fmt.Fprintf(os.Stderr, "Starting search for issues created since %s that had label '%s' at any point...\n", since, label)
			fmt.Fprintf(os.Stderr, "Using %d concurrent workers\n\n", concurrency)
		} else {
			fmt.Fprintf(os.Stderr, "Searching for issues created since %s that had label '%s' at any point", since, label)
		}

		issues, err := ghapi.GetIssuesCreatedSinceWithLabel(repo, since, label, verbose, concurrency, olderThan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
		}

		if len(issues) == 0 {
			fmt.Fprintln(os.Stderr, "No issues retrieved.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "Number\tTitle\tState\tCreated at\tNow?")
		for _, issue := range issues {
			hasLabelNow := "❌"
			if issue.HasLabel(label) {
				hasLabelNow = "✔️"
			}
			fmt.Fprintf(w, "#%d\t%s\t%s\t%s\t%s\n", issue.Number, issue.Title, issue.State, issue.CreatedAt, hasLabelNow)
		}
		w.Flush()

		fmt.Fprintf(os.Stderr, "\nFound %d issue(s)\n", len(issues))
	},
}

func init() {
	issuesWithHistoricalLabelCmd.Flags().StringP("since", "s", "", "Oldest issue creation date (format: YYYY-MM-DD)")
	issuesWithHistoricalLabelCmd.Flags().BoolP("verbose", "v", false, "Show verbose output as issues are pulled and evaluated")
	issuesWithHistoricalLabelCmd.Flags().IntP("concurrency", "c", 10, "Number of concurrent workers for per-issue requests (issue lists are always retrieved one page at a time)")
	issuesWithHistoricalLabelCmd.Flags().StringP("repo", "r", "fleetdm/fleet", "Repository to search (format: owner/repo)")
	issuesWithHistoricalLabelCmd.Flags().Int("older-than", 0, "Only pull issues with an ID less than this (for e.g. continuing after being rate-limited)")
}
