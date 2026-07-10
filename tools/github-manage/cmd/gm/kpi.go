package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

// kpiCmd is the parent for KPI-gathering reports.
var kpiCmd = &cobra.Command{
	Use:   "kpi",
	Short: "Gather Fleet KPIs",
	Long:  "Commands that reproduce the KPIs recorded weekly into the confidential KPI spreadsheet.",
}

const (
	kpiOwnerRepo       = "fleetdm/fleet"
	kpiConfidential    = "fleetdm/confidential"
	kpiPerPage         = 100
	kpiCommitWorkers   = 10
	kpiOneDayHours     = 24
	kpiThreeWeeks      = 21
	kpiWeek            = 7
	kpiBugOpenTimeGoal = 32 // days; the capacity-planning threshold from product-groups.md
)

// kpiLabel/kpiActor/kpiIssue mirror the fields the report needs from the
// GitHub REST /issues and /pulls endpoints.
type kpiLabel struct {
	Name string `json:"name"`
}

type kpiActor struct {
	Type string `json:"type"`
}

type kpiIssue struct {
	Number     int        `json:"number"`
	HTMLURL    string     `json:"html_url"`
	CreatedAt  time.Time  `json:"created_at"`
	ClosedAt   *time.Time `json:"closed_at"`
	MergedAt   *time.Time `json:"merged_at"`
	Draft      bool       `json:"draft"`
	Labels     []kpiLabel `json:"labels"`
	User       kpiActor   `json:"user"`
	CommitsURL string     `json:"commits_url"`
}

func (i kpiIssue) hasLabel(name string) bool {
	for _, l := range i.Labels {
		if l.Name == name {
			return true
		}
	}
	return false
}

func (i kpiIssue) hasAnyLabel(names ...string) bool {
	for _, n := range names {
		if i.hasLabel(n) {
			return true
		}
	}
	return false
}

type kpiCommit struct {
	Commit struct {
		Author struct {
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
}

// kpiEngResult is the machine-readable report shape (for --format json).
type kpiEngResult struct {
	GeneratedAt time.Time `json:"generatedAt"`

	ContributorPRAvgOpenDays int     `json:"contributorPrAvgOpenDays"`
	OpenContributorPRs       int     `json:"openContributorPrs"`
	OpenNonDraftPRs          int     `json:"openNonDraftPrs"`
	CommitToMergeAvgDays     float64 `json:"commitToMergeAvgDays"`
	CommitToMergeSample      int     `json:"commitToMergeSample"`

	BugAvgOpenDays             int  `json:"bugAvgOpenDays"`
	UnprioritizedBugAvgOpenDay int  `json:"unprioritizedBugAvgOpenDays"`
	OpenBugs                   int  `json:"openBugs"`
	Bugs32DaysOrOlder          int  `json:"bugs32DaysOrOlder"`
	CustomerBugsPastWeek       int  `json:"customerBugsPastWeek"`
	BugsOpenedPastWeek         int  `json:"bugsOpenedPastWeek"`
	BugsClosedPastWeek         int  `json:"bugsClosedPastWeek"`
	OverBugOpenTimeGoal        bool `json:"overBugOpenTimeGoal"`

	HandbookOpenPRs          int     `json:"handbookOpenPrs"`
	HandbookPRAvgOpenDays    float64 `json:"handbookPrAvgOpenDays"`
	HandbookPRSample         int     `json:"handbookPrSample"`
	ConfidentialRepoIncluded bool    `json:"confidentialRepoIncluded"`
}

// CSVForSheet returns the six values in the exact column order the KPI
// spreadsheet expects (see website/scripts/get-bug-and-pr-report.js).
func (r kpiEngResult) CSVForSheet() string {
	return fmt.Sprintf("%d,%d,%d,%d,%d,%d",
		r.ContributorPRAvgOpenDays,
		r.BugAvgOpenDays,
		r.Bugs32DaysOrOlder,
		r.CustomerBugsPastWeek,
		r.BugsOpenedPastWeek,
		r.BugsClosedPastWeek,
	)
}

var kpiEngCmd = &cobra.Command{
	Use:   "eng",
	Short: "Engineering KPIs (bug & PR report)",
	Long: `Reproduce the weekly Engineering KPIs recorded into the confidential KPI spreadsheet.

This is a Go re-implementation of website/scripts/get-bug-and-pr-report.js so the
numbers can be pulled with a single command instead of standing up the website.
It uses the authenticated ` + "`gh`" + ` CLI, so make sure ` + "`gh auth status`" + ` is healthy.

The first six lines of output are emitted (in spreadsheet column order) as a CSV
row ready to paste into the KPI sheet:
  contributor PR open time, bug open time, bugs 32+ days, customer bugs/week,
  bugs opened/week, bugs closed/week

Usage:
  gm kpi eng
  gm kpi eng --format json
  gm kpi eng --format csv
  gm kpi eng --no-commit-to-merge   # skip per-PR commit fetches (much faster)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		skipCommitToMerge, _ := cmd.Flags().GetBool("no-commit-to-merge")

		now := time.Now()
		threeWeeksAgo := now.Add(-kpiThreeWeeks * kpiOneDayHours * time.Hour)

		daysSince := func(t time.Time) float64 { return now.Sub(t).Hours() / kpiOneDayHours }

		// ---- Open bugs (paginate until exhausted) ----
		openBugs, err := fetchAllPages(kpiOwnerRepo, "issues", "state=open&labels=bug")
		if err != nil {
			return fmt.Errorf("fetching open bugs: %w", err)
		}
		var bugOpenDays, unprioritizedBugOpenDays []float64
		var bugs32Plus, customerBugsWeek, bugsOpenedWeek int
		for _, b := range openBugs {
			d := daysSince(b.CreatedAt)
			bugOpenDays = append(bugOpenDays, d)
			if d >= kpiBugOpenTimeGoal {
				bugs32Plus++
			}
			if d <= kpiWeek {
				bugsOpenedWeek++
				for _, l := range b.Labels {
					if strings.Contains(l.Name, "customer-") {
						customerBugsWeek++
						break
					}
				}
			}
			if !b.hasLabel(":release") {
				unprioritizedBugOpenDays = append(unprioritizedBugOpenDays, d)
			}
		}

		// ---- Closed bugs (first 3 pages, count closed in past week) ----
		closedBugs, err := fetchPages(kpiOwnerRepo, "issues", "state=closed&labels=bug", 3)
		if err != nil {
			return fmt.Errorf("fetching closed bugs: %w", err)
		}
		var bugsClosedWeek int
		for _, b := range closedBugs {
			if b.ClosedAt != nil && daysSince(*b.ClosedAt) <= kpiWeek {
				bugsClosedWeek++
			}
		}

		// ---- Public PRs merged in past 3 weeks (commit-to-merge) ----
		closedPRs, err := fetchPages(kpiOwnerRepo, "pulls", "state=closed&sort=updated&direction=desc", 3)
		if err != nil {
			return fmt.Errorf("fetching closed PRs: %w", err)
		}
		var mergedRecently []kpiIssue
		for _, pr := range closedPRs {
			if !pr.Draft && pr.MergedAt != nil && !pr.MergedAt.Before(threeWeeksAgo) {
				mergedRecently = append(mergedRecently, pr)
			}
		}
		var commitToMerge []float64
		if !skipCommitToMerge {
			commitToMerge = commitToMergeDays(mergedRecently)
		}

		// ---- Open PRs (paginate) + contributor split ----
		openPRs, err := fetchAllPages(kpiOwnerRepo, "pulls", "state=open")
		if err != nil {
			return fmt.Errorf("fetching open PRs: %w", err)
		}
		var openNonDraft, openContributor int
		var contributorOpenDays []float64
		for _, pr := range openPRs {
			if pr.Draft {
				continue
			}
			openNonDraft++
			if pr.User.Type != "Bot" && !pr.hasAnyLabel("#handbook", "~ceo", ":improve documentation") {
				openContributor++
				contributorOpenDays = append(contributorOpenDays, daysSince(pr.CreatedAt))
			}
		}

		// ---- Confidential repo PRs (open + recently closed), best-effort ----
		confidentialIncluded := true
		confOpen, err := fetchPages(kpiConfidential, "pulls", "state=open", 1)
		if err != nil {
			confidentialIncluded = false
		}
		var confClosedRecent []kpiIssue
		if confidentialIncluded {
			confClosed, err := fetchPages(kpiConfidential, "pulls", "state=closed&sort=updated&direction=desc", 1)
			if err != nil {
				confidentialIncluded = false
			} else {
				for _, pr := range confClosed {
					if !pr.Draft && pr.ClosedAt != nil && !pr.ClosedAt.Before(threeWeeksAgo) {
						confClosedRecent = append(confClosedRecent, pr)
					}
				}
			}
		}

		// ---- Handbook PRs (public + confidential) ----
		handbookOpen := 0
		for _, pr := range append(append([]kpiIssue{}, openPRs...), confOpen...) {
			if !pr.Draft && pr.hasLabel("#handbook") {
				handbookOpen++
			}
		}
		var handbookMerged []kpiIssue
		for _, pr := range append(append([]kpiIssue{}, mergedRecently...), confClosedRecent...) {
			if !pr.Draft && pr.hasLabel("#handbook") {
				handbookMerged = append(handbookMerged, pr)
			}
		}
		var handbookAvgOpen float64
		if len(handbookMerged) > 0 {
			var total float64
			for _, pr := range handbookMerged {
				if pr.ClosedAt != nil {
					total += math.Abs(pr.ClosedAt.Sub(pr.CreatedAt).Hours()) / kpiOneDayHours
				}
			}
			handbookAvgOpen = math.Round((total/float64(len(handbookMerged)))*100) / 100
		}

		result := kpiEngResult{
			GeneratedAt:                now,
			ContributorPRAvgOpenDays:   roundedAvg(contributorOpenDays),
			OpenContributorPRs:         openContributor,
			OpenNonDraftPRs:            openNonDraft,
			CommitToMergeAvgDays:       exactAvg(commitToMerge),
			CommitToMergeSample:        len(commitToMerge),
			BugAvgOpenDays:             roundedAvg(bugOpenDays),
			UnprioritizedBugAvgOpenDay: roundedAvg(unprioritizedBugOpenDays),
			OpenBugs:                   len(bugOpenDays),
			Bugs32DaysOrOlder:          bugs32Plus,
			CustomerBugsPastWeek:       customerBugsWeek,
			BugsOpenedPastWeek:         bugsOpenedWeek,
			BugsClosedPastWeek:         bugsClosedWeek,
			OverBugOpenTimeGoal:        roundedAvg(bugOpenDays) > kpiBugOpenTimeGoal,
			HandbookOpenPRs:            handbookOpen,
			HandbookPRAvgOpenDays:      handbookAvgOpen,
			HandbookPRSample:           len(handbookMerged),
			ConfidentialRepoIncluded:   confidentialIncluded,
		}

		return renderKPIEng(result, strings.ToLower(strings.TrimSpace(format)), skipCommitToMerge)
	},
}

func renderKPIEng(r kpiEngResult, format string, skipCommitToMerge bool) error {
	switch format {
	case "json":
		out, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Println(string(out))
	case "csv":
		fmt.Println(r.CSVForSheet())
	default:
		commitToMerge := fmt.Sprintf("%.0f days (n=%d)", r.CommitToMergeAvgDays, r.CommitToMergeSample)
		if skipCommitToMerge {
			commitToMerge = "skipped (--no-commit-to-merge)"
		}
		confNote := "public + confidential"
		if !r.ConfidentialRepoIncluded {
			confNote = "public only — no confidential repo access"
		}
		bar := strings.Repeat("=", 60)
		fmt.Println(bar)
		fmt.Printf("Engineering KPIs — %s\n", r.GeneratedAt.Format("2006-01-02 15:04 MST"))
		fmt.Println(bar)
		fmt.Println("CSV for KPI spreadsheet (paste, then Split text to columns):")
		fmt.Printf("  %s\n", r.CSVForSheet())
		fmt.Println("  cols: contributor PR open time, bug open time, bugs 32+d,")
		fmt.Println("        customer bugs/wk, bugs opened/wk, bugs closed/wk")
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println("Pull requests")
		fmt.Printf("  Contributor PR average open time:      %d days\n", r.ContributorPRAvgOpenDays)
		fmt.Printf("  Open contributor PRs:                  %d\n", r.OpenContributorPRs)
		fmt.Printf("  All non-draft open PRs:                %d\n", r.OpenNonDraftPRs)
		fmt.Printf("  Commit-to-merge (public, past 3 wks):  %s\n", commitToMerge)
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println("Bugs")
		fmt.Printf("  Average open time (all bugs):          %d days\n", r.BugAvgOpenDays)
		fmt.Printf("  Average open time (unprioritized):     %d days\n", r.UnprioritizedBugAvgOpenDay)
		fmt.Printf("  Total open bugs:                       %d\n", r.OpenBugs)
		fmt.Printf("  Bugs 32+ days old:                     %d\n", r.Bugs32DaysOrOlder)
		fmt.Printf("  Customer-reported bugs (past week):    %d\n", r.CustomerBugsPastWeek)
		fmt.Printf("  Bugs opened (past week):               %d\n", r.BugsOpenedPastWeek)
		fmt.Printf("  Bugs closed (past week):               %d\n", r.BugsClosedPastWeek)
		fmt.Println(strings.Repeat("-", 60))
		fmt.Printf("Handbook PRs (%s)\n", confNote)
		fmt.Printf("  Open #handbook PRs:                    %d\n", r.HandbookOpenPRs)
		fmt.Printf("  Avg open time (merged, past 3 wks):    %.2f days (n=%d)\n", r.HandbookPRAvgOpenDays, r.HandbookPRSample)
		fmt.Println(bar)
		if r.OverBugOpenTimeGoal {
			fmt.Printf("⚠  Average bug open time (%d days) is over the %d-day KPI. Per\n", r.BugAvgOpenDays, kpiBugOpenTimeGoal)
			fmt.Println("   product-groups.md, 50% of each sprint's capacity should go to")
			fmt.Println("   bugs + engineering-initiated stories (less needs CEO approval).")
		}
	}
	return nil
}

// fetchAllPages pages through a list endpoint until a short page is returned.
func fetchAllPages(repo, endpoint, query string) ([]kpiIssue, error) {
	var all []kpiIssue
	for page := 1; ; page++ {
		batch, err := fetchOnePage(repo, endpoint, query, page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) != kpiPerPage {
			break
		}
	}
	return all, nil
}

// fetchPages pages through a list endpoint for a fixed number of pages.
func fetchPages(repo, endpoint, query string, pages int) ([]kpiIssue, error) {
	var all []kpiIssue
	for page := 1; page <= pages; page++ {
		batch, err := fetchOnePage(repo, endpoint, query, page)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) != kpiPerPage {
			break
		}
	}
	return all, nil
}

func fetchOnePage(repo, endpoint, query string, page int) ([]kpiIssue, error) {
	url := fmt.Sprintf("repos/%s/%s?%s&per_page=%d&page=%d", repo, endpoint, query, kpiPerPage, page)
	out, err := ghapi.RunCommandAndReturnOutput(fmt.Sprintf("gh api '%s'", url))
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", url, err)
	}
	var batch []kpiIssue
	if err := json.Unmarshal(out, &batch); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", url, err)
	}
	return batch, nil
}

// commitToMergeDays fetches each PR's first commit and returns the number of
// days between that commit and the merge, mirroring the website script.
func commitToMergeDays(prs []kpiIssue) []float64 {
	var (
		mu      sync.Mutex
		results []float64
		wg      sync.WaitGroup
		sem     = make(chan struct{}, kpiCommitWorkers)
	)
	for _, pr := range prs {
		wg.Add(1)
		go func(pr kpiIssue) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			out, err := ghapi.RunCommandAndReturnOutput(fmt.Sprintf("gh api '%s'", pr.CommitsURL))
			if err != nil {
				return
			}
			var commits []kpiCommit
			if err := json.Unmarshal(out, &commits); err != nil || len(commits) == 0 {
				return
			}
			if pr.MergedAt == nil {
				return
			}
			days := pr.MergedAt.Sub(commits[0].Commit.Author.Date).Hours() / kpiOneDayHours
			mu.Lock()
			results = append(results, days)
			mu.Unlock()
		}(pr)
	}
	wg.Wait()
	return results
}

func roundedAvg(vals []float64) int {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return int(math.Round(sum / float64(len(vals))))
}

func exactAvg(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vals {
		sum += v
	}
	return math.Round((sum/float64(len(vals)))*100) / 100
}

func init() {
	kpiCmd.AddCommand(kpiEngCmd)
	kpiEngCmd.Flags().StringP("format", "f", "", "Output format: json, csv, or default (human-readable)")
	kpiEngCmd.Flags().Bool("no-commit-to-merge", false, "Skip per-PR commit fetches (faster; omits commit-to-merge time)")
}
