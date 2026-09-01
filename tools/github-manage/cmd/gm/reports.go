package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"fleetdm/gm/pkg/ghapi"

	"github.com/spf13/cobra"
)

var reportsCmd = &cobra.Command{
	Use:   "reports",
	Short: "Read-only reports summarizing trends across the repository",
}

type reportIssue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// productGroup returns the first "#g-" label on the issue (its owning product
// group), or "(none)" when the issue carries no product group label.
func (i reportIssue) productGroup() string {
	for _, l := range i.Labels {
		if strings.HasPrefix(l.Name, "#g-") {
			return l.Name
		}
	}
	return "(none)"
}

// priorityReport holds the bucketed results for a single priority label.
type priorityReport struct {
	Priority     string         `json:"priority"`
	Total        int            `json:"total"`
	WeeklyCounts map[string]int `json:"weeklyCounts"` // week-start (Mon, YYYY-MM-DD) -> count
	GroupTotals  map[string]int `json:"groupTotals"`  // product group -> count
	// GroupByMonth is month (YYYY-MM) -> product group -> count.
	GroupByMonth map[string]map[string]int `json:"groupByMonth"`
}

var reportsPriorityCmd = &cobra.Command{
	Use:   "priority",
	Short: "Report priority (P0/P1) issues over time, bucketed by week and product group",
	Long: `Report how many priority issues (P0, P1) were opened over time in fleetdm/fleet.

For each priority label the report shows:
  1. A total count within the window.
  2. Issues created bucketed by week (Monday start) so you can see whether
     more are coming in now versus before.
  3. A product-group breakdown (by "#g-" label) bucketed by month, so trends
     in ownership are visible (e.g. #g-apple-at-work used to own most P1s, now
     it's #g-power-to-pc).

The window defaults to the last 6 months; override it with --months.

Usage:
  gm reports priority
  gm reports priority --months 12
  gm reports priority --priority P0
  gm reports priority --format json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		months, _ := cmd.Flags().GetInt("months")
		priorities, _ := cmd.Flags().GetStringSlice("priority")
		format, _ := cmd.Flags().GetString("format")
		limit, _ := cmd.Flags().GetInt("limit")

		if months <= 0 {
			return fmt.Errorf("--months must be greater than 0")
		}

		now := time.Now()
		since := now.AddDate(0, -months, 0)
		sinceStr := since.Format("2006-01-02")

		weeks := weekBuckets(since, now)
		monthsList := monthBuckets(since, now)

		var reports []priorityReport
		for _, p := range priorities {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			issues, err := fetchPriorityIssues(p, sinceStr, limit)
			if err != nil {
				return fmt.Errorf("failed to fetch %s issues: %v", p, err)
			}
			reports = append(reports, buildPriorityReport(p, issues))
		}

		format = strings.ToLower(strings.TrimSpace(format))
		if format == "json" {
			out := struct {
				Repo   string           `json:"repo"`
				Since  string           `json:"since"`
				Months int              `json:"months"`
				Weeks  []string         `json:"weeks"`
				Report []priorityReport `json:"reports"`
			}{
				Repo:   ghapi.DefaultRepo,
				Since:  sinceStr,
				Months: months,
				Weeks:  weeks,
				Report: reports,
			}
			b, err := json.MarshalIndent(out, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %v", err)
			}
			fmt.Println(string(b))
			return nil
		}

		printPriorityReport(reports, weeks, monthsList, sinceStr, months)
		return nil
	},
}

// fetchPriorityIssues fetches all issues (open or closed) with the given
// priority label created on or after sinceStr.
func fetchPriorityIssues(priority, sinceStr string, limit int) ([]reportIssue, error) {
	if limit <= 0 {
		limit = 1000
	}

	command := fmt.Sprintf(
		"gh issue list --repo %s --state all --label %q --search %q --json number,title,createdAt,labels --limit %d",
		ghapi.DefaultRepo,
		priority,
		"created:>="+sinceStr,
		limit,
	)

	output, err := ghapi.RunCommandAndReturnOutput(command)
	if err != nil {
		return nil, fmt.Errorf("gh command failed: %v", err)
	}

	var issues []reportIssue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	if len(issues) == limit {
		return nil, fmt.Errorf("hit the fetch limit of %d for %s; re-run with a larger --limit", limit, priority)
	}

	return issues, nil
}

func buildPriorityReport(priority string, issues []reportIssue) priorityReport {
	r := priorityReport{
		Priority:     priority,
		Total:        len(issues),
		WeeklyCounts: map[string]int{},
		GroupTotals:  map[string]int{},
		GroupByMonth: map[string]map[string]int{},
	}
	for _, is := range issues {
		wk := weekStart(is.CreatedAt).Format("2006-01-02")
		r.WeeklyCounts[wk]++

		g := is.productGroup()
		r.GroupTotals[g]++

		m := is.CreatedAt.Format("2006-01")
		if r.GroupByMonth[m] == nil {
			r.GroupByMonth[m] = map[string]int{}
		}
		r.GroupByMonth[m][g]++
	}
	return r
}

// weekStart returns the Monday (00:00, in the value's location) of t's week.
func weekStart(t time.Time) time.Time {
	t = t.Truncate(24 * time.Hour)
	// time.Monday == 1; shift so Monday is the start of the week.
	offset := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -offset)
}

// weekBuckets returns the ordered list of week-start dates (YYYY-MM-DD) that
// span [since, until], inclusive of the weeks containing each endpoint.
func weekBuckets(since, until time.Time) []string {
	var out []string
	for w := weekStart(since); !w.After(until); w = w.AddDate(0, 0, 7) {
		out = append(out, w.Format("2006-01-02"))
	}
	return out
}

// monthBuckets returns the ordered list of YYYY-MM buckets spanning the window.
func monthBuckets(since, until time.Time) []string {
	var out []string
	m := time.Date(since.Year(), since.Month(), 1, 0, 0, 0, 0, since.Location())
	end := time.Date(until.Year(), until.Month(), 1, 0, 0, 0, 0, until.Location())
	for ; !m.After(end); m = m.AddDate(0, 1, 0) {
		out = append(out, m.Format("2006-01"))
	}
	return out
}

func sortedGroups(totals map[string]int) []string {
	groups := make([]string, 0, len(totals))
	for g := range totals {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(a, b int) bool {
		if totals[groups[a]] != totals[groups[b]] {
			return totals[groups[a]] > totals[groups[b]]
		}
		return groups[a] < groups[b]
	})
	return groups
}

func printPriorityReport(reports []priorityReport, weeks, months []string, sinceStr string, monthsWindow int) {
	bar := strings.Repeat("=", 60)
	fmt.Println(bar)
	fmt.Printf("Priority Issue Report — last %d months (created since %s)\n", monthsWindow, sinceStr)
	fmt.Printf("Repo: %s\n", ghapi.DefaultRepo)
	fmt.Println(bar)

	for _, r := range reports {
		fmt.Printf("\n### %s — %d issue(s)\n\n", r.Priority, r.Total)
		if r.Total == 0 {
			fmt.Println("No issues in this window.")
			continue
		}

		// 1. Weekly created counts with a simple bar chart.
		maxCount := 0
		for _, wk := range weeks {
			if r.WeeklyCounts[wk] > maxCount {
				maxCount = r.WeeklyCounts[wk]
			}
		}
		fmt.Println("Created per week (Mon start):")
		fmt.Printf("  %-12s %5s  %s\n", "Week", "Count", "")
		for _, wk := range weeks {
			c := r.WeeklyCounts[wk]
			fmt.Printf("  %-12s %5d  %s\n", wk, c, scaledBar(c, maxCount, 40))
		}

		// 2. Product group totals.
		groups := sortedGroups(r.GroupTotals)
		fmt.Println("\nProduct group totals:")
		for _, g := range groups {
			fmt.Printf("  %-24s %d\n", g, r.GroupTotals[g])
		}

		// 3. Product group breakdown by month (ownership trend).
		fmt.Println("\nProduct group by month:")
		fmt.Printf("  %-24s", "Group")
		for _, m := range months {
			fmt.Printf(" %7s", m)
		}
		fmt.Println()
		for _, g := range groups {
			fmt.Printf("  %-24s", g)
			for _, m := range months {
				fmt.Printf(" %7d", r.GroupByMonth[m][g])
			}
			fmt.Println()
		}

		// Trend summary: top group in the first vs last populated month.
		if firstM, firstG, ok := topGroupInBoundaryMonth(r, months, true); ok {
			if lastM, lastG, ok2 := topGroupInBoundaryMonth(r, months, false); ok2 && firstM != lastM {
				fmt.Printf("\nTrend: %s led in %s; %s leads in %s.\n", firstG, firstM, lastG, lastM)
			}
		}
	}
}

// topGroupInBoundaryMonth finds the earliest (first=true) or latest
// (first=false) month with data and returns its top product group.
func topGroupInBoundaryMonth(r priorityReport, months []string, first bool) (string, string, bool) {
	order := months
	if !first {
		order = make([]string, len(months))
		for i, m := range months {
			order[len(months)-1-i] = m
		}
	}
	for _, m := range order {
		gm := r.GroupByMonth[m]
		if len(gm) == 0 {
			continue
		}
		top := sortedGroups(gm)[0]
		return m, top, true
	}
	return "", "", false
}

func scaledBar(count, max, width int) string {
	if count <= 0 || max <= 0 {
		return ""
	}
	n := count * width / max
	if n < 1 {
		n = 1
	}
	return strings.Repeat("█", n)
}

func init() {
	reportsCmd.AddCommand(reportsPriorityCmd)

	reportsPriorityCmd.Flags().IntP("months", "m", 6, "Length of the reporting window in months")
	reportsPriorityCmd.Flags().StringSlice("priority", []string{"P0", "P1"}, "Priority labels to report on, in order")
	reportsPriorityCmd.Flags().StringP("format", "f", "", "Output format: json, or default (human-readable)")
	reportsPriorityCmd.Flags().IntP("limit", "l", 1000, "Maximum number of issues to fetch per priority")
}
