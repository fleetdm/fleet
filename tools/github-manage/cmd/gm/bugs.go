package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

var bugsCmd = &cobra.Command{
	Use:   "bugs",
	Short: "Bug-related utilities and reports",
}

// BugIssue represents a GitHub issue with bug label
type BugIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	State     string    `json:"state"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	HTMLURL string `json:"url"`
}

var bugsAverageOpenCmd = &cobra.Command{
	Use:   "average-open-days",
	Short: "Calculate the average number of days bugs are open",
	Long: `Calculate the average number of days bugs are open for the fleetdm/fleet repository.

This command fetches all open issues with the "bug" label and calculates
the average number of days they have been open.

This metric is equivalent to 'averageNumberOfDaysBugsAreOpenFor' from the
website/scripts/get-bug-and-pr-report.js script.

Usage:
  gm bugs average-open-days
  gm bugs average-open-days --verbose
  gm bugs average-open-days --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		format, _ := cmd.Flags().GetString("format")
		limit, _ := cmd.Flags().GetInt("limit")

		// Fetch all open bugs using gh CLI
		if verbose {
			fmt.Println("Fetching open bugs from fleetdm/fleet...")
		}

		bugs, err := fetchOpenBugs(limit)
		if err != nil {
			return fmt.Errorf("failed to fetch open bugs: %v", err)
		}

		if len(bugs) == 0 {
			fmt.Println("No open bugs found.")
			return nil
		}

		// Calculate days since each bug was opened
		now := time.Now()
		var totalDays float64
		var bugDetails []struct {
			Number   int
			Title    string
			DaysOpen float64
		}

		for _, bug := range bugs {
			daysOpen := now.Sub(bug.CreatedAt).Hours() / 24
			totalDays += daysOpen
			bugDetails = append(bugDetails, struct {
				Number   int
				Title    string
				DaysOpen float64
			}{
				Number:   bug.Number,
				Title:    bug.Title,
				DaysOpen: daysOpen,
			})
		}

		averageDays := totalDays / float64(len(bugs))
		roundedAverage := int(math.Round(averageDays))

		// Output based on format
		format = strings.ToLower(strings.TrimSpace(format))
		switch format {
		case "json":
			output := struct {
				TotalBugs                         int     `json:"totalBugs"`
				AverageNumberOfDaysBugsAreOpenFor int     `json:"averageNumberOfDaysBugsAreOpenFor"`
				AverageDaysExact                  float64 `json:"averageDaysExact"`
			}{
				TotalBugs:                         len(bugs),
				AverageNumberOfDaysBugsAreOpenFor: roundedAverage,
				AverageDaysExact:                  averageDays,
			}
			jsonBytes, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %v", err)
			}
			fmt.Println(string(jsonBytes))

		case "tsv":
			fmt.Println("Metric\tValue")
			fmt.Printf("Total Open Bugs\t%d\n", len(bugs))
			fmt.Printf("Average Days Open (rounded)\t%d\n", roundedAverage)
			fmt.Printf("Average Days Open (exact)\t%.2f\n", averageDays)

		default: // human-readable
			fmt.Println(strings.Repeat("=", 50))
			fmt.Println("Bug Report: Average Open Time")
			fmt.Println(strings.Repeat("=", 50))
			fmt.Printf("Total open bugs: %d\n", len(bugs))
			fmt.Printf("Average number of days bugs are open for: %d\n", roundedAverage)
			fmt.Printf("Average days (exact): %.2f\n", averageDays)
			fmt.Println(strings.Repeat("=", 50))
		}

		// Verbose output: list all bugs with their open days
		if verbose && format != "json" {
			fmt.Println("\nDetailed bug list:")
			fmt.Println(strings.Repeat("-", 80))
			fmt.Printf("%-8s %-10s %s\n", "Number", "Days Open", "Title")
			fmt.Println(strings.Repeat("-", 80))
			for _, bug := range bugDetails {
				title := bug.Title
				if len(title) > 50 {
					title = title[:47] + "..."
				}
				fmt.Printf("#%-7d %-10.1f %s\n", bug.Number, bug.DaysOpen, title)
			}
		}

		return nil
	},
}

// fetchOpenBugs fetches all open issues with the "bug" label from fleetdm/fleet
func fetchOpenBugs(limit int) ([]BugIssue, error) {
	if limit <= 0 {
		limit = 1000 // Default high limit to get all bugs
	}

	// Use gh CLI to fetch open issues with bug label
	// The gh CLI handles pagination internally with --limit
	command := fmt.Sprintf(
		"gh issue list --repo fleetdm/fleet --state open --label bug --json number,title,createdAt,state,labels,url --limit %d",
		limit,
	)

	output, err := ghapi.RunCommandAndReturnOutput(command)
	if err != nil {
		return nil, fmt.Errorf("gh command failed: %v", err)
	}

	var bugs []BugIssue
	if err := json.Unmarshal(output, &bugs); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	if len(bugs) == limit {
		return nil, fmt.Errorf("there are at least %d open bugs; choose a larger limit", limit)
	}

	return bugs, nil
}

func init() {
	bugsCmd.AddCommand(bugsAverageOpenCmd)

	bugsAverageOpenCmd.Flags().BoolP("verbose", "v", false, "Show detailed list of all bugs with their open days")
	bugsAverageOpenCmd.Flags().StringP("format", "f", "", "Output format: json, tsv, or default (human-readable)")
	bugsAverageOpenCmd.Flags().IntP("limit", "l", 1000, "Maximum number of bugs to fetch")
}
