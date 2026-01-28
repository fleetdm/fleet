package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"fleetdm/gm/pkg/ghapi"
	"fleetdm/gm/pkg/logger"

	"github.com/spf13/cobra"
)

var labelHistoryCmd = &cobra.Command{
	Use:   "label-history",
	Short: "Find issues that had a specific label at any point in their history",
	Long: `Find issue numbers for all issues in a repository that had a specific
label applied to them at any point in their history, filtered by creation date.

Example:
  gm label-history fleetdm/fleet --start-date 2024-01-01 --label "critical"
  gm label-history fleetdm/fleet --start-date 2024-01-01 --label "bug" --json`,
	RunE: runLabelHistory,
}

func init() {
	labelHistoryCmd.Flags().String("start-date", "", "Start date in YYYY-MM-DD format (issues created on or after this date)")
	labelHistoryCmd.Flags().String("label", "", "Label name to search for in issue history")
	labelHistoryCmd.Flags().Bool("json", false, "Output results in JSON format")
	labelHistoryCmd.Flags().Int("concurrency", 10, "Number of concurrent API requests for timeline fetching")
	labelHistoryCmd.MarkFlagRequired("start-date")
	labelHistoryCmd.MarkFlagRequired("label")
}

func runLabelHistory(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repository argument is required (e.g., fleetdm/fleet)")
	}

	repo := args[0]
	startDate, _ := cmd.Flags().GetString("start-date")
	labelName, _ := cmd.Flags().GetString("label")
	outputJSON, _ := cmd.Flags().GetBool("json")
	concurrency, _ := cmd.Flags().GetInt("concurrency")

	if concurrency < 1 {
		concurrency = 1
	} else if concurrency > 50 {
		concurrency = 50 // Cap at 50 to avoid rate limiting issues
	}

	fmt.Fprintf(os.Stderr, "Searching for issues in %s created on or after %s that had label '%s' at any point\n", repo, startDate, labelName)
	logger.Infof("Searching for issues in %s created on or after %s that had label '%s' at any point", repo, startDate, labelName)

	// Search for issues created after the start date (include both open and closed)
	issues, err := ghapi.GetIssuesByRepo(repo, startDate)
	if err != nil {
		return fmt.Errorf("failed to fetch issues: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d issues created on or after %s\n", len(issues), startDate)
	logger.Infof("Found %d issues created on or after %s", len(issues), startDate)

	// Define the timeline fetcher function
	timelineFetcher := func(issueNumber int) ([]ghapi.TimelineEvent, error) {
		return ghapi.GetIssueTimelineEvents(repo, fmt.Sprintf("%d", issueNumber))
	}

	issueNumbers, err := findIssuesWithHistoricalLabelParallel(issues, labelName, timelineFetcher, concurrency, os.Stderr)
	if err != nil {
		return err
	}

	// Output results
	if len(issueNumbers) == 0 {
		if outputJSON {
			fmt.Println(`{"issues": []}`)
		} else {
			fmt.Println("No issues found that had the specified label.")
		}
		return nil
	}

	if outputJSON {
		// Output as JSON
		issuesJSON, err := formatIssuesJSON(repo, startDate, labelName, issueNumbers)
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
		fmt.Println(issuesJSON)
	} else {
		// Output as formatted text
		fmt.Printf("\nFound %d issue(s):\n\n", len(issueNumbers))
		for _, num := range issueNumbers {
			fmt.Printf("- #%d\n", num)
		}
		fmt.Printf("\nFull list: %s\n", formatIssueList(issueNumbers))
	}

	return nil
}

// issueWork represents a unit of work for timeline checking
type issueWork struct {
	issue ghapi.Issue
}

// issueResult represents the result of checking an issue's timeline
type issueResult struct {
	issueNumber int
	hadLabel    bool
}

// findIssuesWithHistoricalLabelParallel filters issues that had the label at any point
// using parallel timeline fetching with rate limiting.
// Progress is written to the provided progressWriter (typically os.Stderr).
func findIssuesWithHistoricalLabelParallel(issues []ghapi.Issue, labelName string, timelineFetcher func(int) ([]ghapi.TimelineEvent, error), concurrency int, progressWriter io.Writer) ([]int, error) {
	var issueNumbers []int
	var issuesToCheck []issueWork

	// First pass: quickly check issues that currently have the label
	for _, issue := range issues {
		hasLabelCurrently := false
		for _, label := range issue.Labels {
			if label.Name == labelName {
				hasLabelCurrently = true
				break
			}
		}

		if hasLabelCurrently {
			logger.Debugf("Issue #%d currently has label '%s'", issue.Number, labelName)
			issueNumbers = append(issueNumbers, issue.Number)
		} else {
			issuesToCheck = append(issuesToCheck, issueWork{issue: issue})
		}
	}

	fmt.Fprintf(progressWriter, "Found %d issues with label currently, %d issues need timeline check\n", len(issueNumbers), len(issuesToCheck))
	logger.Infof("Found %d issues with label currently, %d issues need timeline check", len(issueNumbers), len(issuesToCheck))

	if len(issuesToCheck) == 0 {
		sort.Ints(issueNumbers)
		return issueNumbers, nil
	}

	// Create channels for work distribution and results
	workChan := make(chan issueWork, len(issuesToCheck))
	resultChan := make(chan issueResult, len(issuesToCheck))

	// Track progress
	var processed int64
	totalToCheck := int64(len(issuesToCheck))

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workChan {
				// Fetch timeline events
				timeline, err := timelineFetcher(work.issue.Number)
				if err != nil {
					logger.Errorf("Failed to get timeline for issue #%d: %v", work.issue.Number, err)
					resultChan <- issueResult{issueNumber: work.issue.Number, hadLabel: false}
					continue
				}

				// Check if the label was ever applied
				hadLabel := false
				for _, event := range timeline {
					if event.Event == "labeled" && event.Label.Name == labelName {
						hadLabel = true
						logger.Debugf("Issue #%d had label '%s' applied (event at %s)", work.issue.Number, labelName, event.CreatedAt)
						break
					}
				}

				resultChan <- issueResult{issueNumber: work.issue.Number, hadLabel: hadLabel}

				// Log progress every 50 issues
				count := atomic.AddInt64(&processed, 1)
				if count%50 == 0 || count == totalToCheck {
					fmt.Fprintf(progressWriter, "Progress: checked %d/%d timelines (%.1f%%)\n", count, totalToCheck, float64(count)/float64(totalToCheck)*100)
					logger.Infof("Progress: checked %d/%d timelines (%.1f%%)", count, totalToCheck, float64(count)/float64(totalToCheck)*100)
				}
			}
		}()
	}

	// Send work to workers
	for _, work := range issuesToCheck {
		workChan <- work
	}
	close(workChan)

	// Wait for all workers to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		if result.hadLabel {
			issueNumbers = append(issueNumbers, result.issueNumber)
		}
	}

	fmt.Fprintf(progressWriter, "Timeline check complete. Found %d total issues with label '%s' in history.\n", len(issueNumbers), labelName)
	logger.Infof("Timeline check complete. Found %d total issues with label '%s' in history.", len(issueNumbers), labelName)

	// Sort issue numbers
	sort.Ints(issueNumbers)
	return issueNumbers, nil
}

// findIssuesWithHistoricalLabel filters issues that had the label at any point (sequential version for testing)
func findIssuesWithHistoricalLabel(issues []ghapi.Issue, labelName string, timelineFetcher func(int) ([]ghapi.TimelineEvent, error)) ([]int, error) {
	var issueNumbers []int
	total := len(issues)
	timelineChecks := 0

	for i, issue := range issues {
		// Check if the issue currently has the label (quick check)
		hasLabelCurrently := false
		for _, label := range issue.Labels {
			if label.Name == labelName {
				hasLabelCurrently = true
				break
			}
		}

		if hasLabelCurrently {
			logger.Debugf("Issue #%d currently has label '%s'", issue.Number, labelName)
			issueNumbers = append(issueNumbers, issue.Number)
			continue
		}

		// If not currently labeled, check timeline events for historical labels
		timelineChecks++
		if timelineChecks%50 == 1 {
			logger.Infof("Checking timeline for issue %d of %d (#%d)...", i+1, total, issue.Number)
		}

		timeline, err := timelineFetcher(issue.Number)
		if err != nil {
			logger.Errorf("Failed to get timeline for issue #%d: %v", issue.Number, err)
			continue // Skip this issue if timeline fetch fails
		}

		// Check if the label was ever applied in the history
		hadLabel := false
		for _, event := range timeline {
			if event.Event == "labeled" && event.Label.Name == labelName {
				hadLabel = true
				logger.Debugf("Issue #%d had label '%s' applied (event at %s)", issue.Number, labelName, event.CreatedAt)
				break
			}
		}

		if hadLabel {
			issueNumbers = append(issueNumbers, issue.Number)
		}
	}

	// Sort issue numbers
	sort.Ints(issueNumbers)
	return issueNumbers, nil
}

// formatIssuesJSON creates a JSON output of the results
func formatIssuesJSON(repo, startDate, label string, issueNumbers []int) (string, error) {
	type Result struct {
		Repository string `json:"repository"`
		StartDate  string `json:"start_date"`
		Label      string `json:"label"`
		Count      int    `json:"count"`
		Issues     []int  `json:"issues"`
	}

	result := Result{
		Repository: repo,
		StartDate:  startDate,
		Label:      label,
		Count:      len(issueNumbers),
		Issues:     issueNumbers,
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func formatIssueList(numbers []int) string {
	strNumbers := make([]string, len(numbers))
	for i, num := range numbers {
		strNumbers[i] = fmt.Sprintf("%d", num)
	}
	return strings.Join(strNumbers, ", ")
}
