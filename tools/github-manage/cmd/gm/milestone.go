package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"fleetdm/gm/pkg/ghapi"
)

// milestoneCmd is the parent command for milestone-related operations.
var milestoneCmd = &cobra.Command{
	Use:   "milestone",
	Short: "Milestone-related utilities",
}

var (
	milestoneFormat        string
	milestoneStripEmojis   bool
	milestoneSummarySort   string
	milestoneIncludeClosed bool
	milestoneIgnoreProject string
	milestoneFilterLabels  string
)

var milestoneReportCmd = &cobra.Command{
	Use:   "report <milestone-name>",
	Short: "Print a table of issues and their project statuses for a milestone",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		milestoneName := args[0]

		// Fetch issues in milestone (with titles)
		msIssues, err := ghapi.GetIssuesByMilestoneWithTitles(milestoneName, 1000)
		if err != nil || len(msIssues) == 0 {
			// If milestone doesn't exist or has no issues, list available milestones
			miles, lerr := ghapi.ListRepoMilestones(milestoneIncludeClosed)
			if lerr != nil {
				return fmt.Errorf("failed to find milestone '%s' and also failed to list milestones: %v", milestoneName, lerr)
			}
			format := strings.ToLower(strings.TrimSpace(milestoneFormat))
			if format == "" {
				format = "tsv"
			}
			// Helpful hint when filtering open-only yields none
			var msg string
			if len(miles) == 0 && !milestoneIncludeClosed {
				msg = fmt.Sprintf("No open milestones found. Use --include-closed to include closed milestones. (Requested milestone: '%s')", milestoneName)
			} else {
				msg = fmt.Sprintf("No issues found for milestone '%s'. Available milestones:", milestoneName)
			}
			switch format {
			case "tsv":
				fmt.Println(msg)
				fmt.Println(strings.Join([]string{"Title", "State"}, "\t"))
				for _, m := range miles {
					t := m.Title
					if milestoneStripEmojis {
						t = stripEmojis(t)
					}
					fmt.Println(strings.Join([]string{t, m.State}, "\t"))
				}
			case "md", "markdown":
				fmt.Println(msg)
				fmt.Printf("| %s |\n", strings.Join([]string{"Title", "State"}, " | "))
				fmt.Printf("| %s |\n", strings.Join([]string{"---", "---"}, " | "))
				for _, m := range miles {
					t := m.Title
					if milestoneStripEmojis {
						t = stripEmojis(t)
					}
					fmt.Printf("| %s | %s |\n", t, m.State)
				}
			default:
				return fmt.Errorf("unsupported --format %q (use: tsv or md)", milestoneFormat)
			}
			// Return error so callers can detect non-report condition
			return fmt.Errorf("milestone '%s' not found or empty", milestoneName)
		}
		// Optional label filtering: require all specified labels to be present
		if strings.TrimSpace(milestoneFilterLabels) != "" {
			wantParts := strings.Split(milestoneFilterLabels, ",")
			wants := make([]string, 0, len(wantParts))
			for _, wp := range wantParts {
				w := strings.ToLower(strings.TrimSpace(wp))
				if w != "" {
					wants = append(wants, w)
				}
			}
			if len(wants) > 0 {
				filtered := make([]ghapi.MilestoneIssue, 0, len(msIssues))
			issueLoop:
				for _, mi := range msIssues {
					labelSet := make(map[string]struct{}, len(mi.Labels))
					for _, l := range mi.Labels {
						ln := strings.ToLower(strings.TrimSpace(l.Name))
						if ln != "" {
							labelSet[ln] = struct{}{}
						}
					}
					for _, want := range wants {
						if _, ok := labelSet[want]; !ok {
							continue issueLoop
						}
					}
					filtered = append(filtered, mi)
				}
				msIssues = filtered
			}
		}

		// Extract numbers for project discovery (post filtering)
		issueNums := make([]int, 0, len(msIssues))
		for _, it := range msIssues {
			issueNums = append(issueNums, it.Number)
		}

		// Determine all projects across these issues, dynamically
		projects, _ := ghapi.GetProjectsForIssues(issueNums)
		// Apply ignore-project filtering if provided
		if strings.TrimSpace(milestoneIgnoreProject) != "" {
			// build tokens (case-insensitive substring match)
			parts := strings.Split(milestoneIgnoreProject, ",")
			toks := make([]string, 0, len(parts))
			for _, p := range parts {
				t := strings.ToLower(strings.TrimSpace(p))
				if t != "" {
					toks = append(toks, t)
				}
			}
			if len(toks) > 0 {
				filtered := make([]ghapi.ProjectInfo, 0, len(projects))
				for _, p := range projects {
					title := p.Title
					if title == "" {
						// Without a title we can't match by name; keep it
						filtered = append(filtered, p)
						continue
					}
					name := strings.ToLower(stripEmojis(title))
					exclude := false
					for _, tok := range toks {
						if tok != "" && strings.Contains(name, tok) {
							exclude = true
							break
						}
					}
					if !exclude {
						filtered = append(filtered, p)
					}
				}
				projects = filtered
			}
		}
		headers := make([]string, 0, len(projects)+2)
		headers = append(headers, "Number")
		for _, p := range projects {
			if p.Title != "" {
				title := p.Title
				if milestoneStripEmojis {
					title = stripEmojis(title)
				}
				headers = append(headers, title)
			} else {
				headers = append(headers, fmt.Sprintf("%d", p.ID))
			}
		}
		// Final column header for issue title
		headers = append(headers, "Title")

		format := strings.ToLower(strings.TrimSpace(milestoneFormat))
		if format == "" {
			format = "tsv"
		}

		switch format {
		case "tsv":
			// Print header as TSV
			// Heading line first
			fmt.Printf("Milestone\treport\t%s\n", milestoneName)
			fmt.Println(strings.Join(headers, "\t"))
		case "md", "markdown":
			// Heading line first
			fmt.Printf("|Milestone report %s|\n", milestoneName)
			// Markdown header
			fmt.Printf("| %s |\n", strings.Join(headers, " | "))
			// Separator row
			seps := make([]string, len(headers))
			for i := range seps {
				seps[i] = "---"
			}
			fmt.Printf("| %s |\n", strings.Join(seps, " | "))
		default:
			return fmt.Errorf("unsupported --format %q (use: tsv or md)", milestoneFormat)
		}

		// Aggregator: projectID -> status -> count (only when Present)
		agg := make(map[int]map[string]int, len(projects))

		// For each issue, gather statuses per project
		for _, mi := range msIssues {
			num := mi.Number
			// Build the list of project IDs in header order
			pids := make([]int, 0, len(projects))
			for _, p := range projects {
				pids = append(pids, p.ID)
			}
			statuses, _ := ghapi.GetIssueProjectStatuses(num, pids)
			row := []string{fmt.Sprintf("%d", num)}
			for _, pid := range pids {
				ps, ok := statuses[pid]
				if !ok || !ps.Present {
					row = append(row, "-")
					continue
				}
				if strings.TrimSpace(ps.Status) == "" {
					cell := "No Status"
					if milestoneStripEmojis {
						cell = stripEmojis(cell)
					}
					row = append(row, cell)
					if agg[pid] == nil {
						agg[pid] = make(map[string]int)
					}
					agg[pid]["No Status"]++
				} else {
					cell := ps.Status
					if milestoneStripEmojis {
						cell = stripEmojis(cell)
					}
					row = append(row, cell)
					if agg[pid] == nil {
						agg[pid] = make(map[string]int)
					}
					agg[pid][ps.Status]++
				}
			}
			// Append truncated title column
			title := mi.Title
			if milestoneStripEmojis {
				title = stripEmojis(title)
			}
			row = append(row, truncateTitle(title, 25))
			if format == "tsv" {
				fmt.Println(strings.Join(row, "\t"))
			} else {
				fmt.Printf("| %s |\n", strings.Join(row, " | "))
			}
		}

		// Build and print summary rows: Project, Status, Count
		type sumRow struct {
			Project   string
			ProjectID int
			Status    string
			Count     int
		}
		rows := make([]sumRow, 0)
		// helper: title by pid
		getProjTitle := func(pid int) string {
			for _, p := range projects {
				if p.ID == pid {
					return p.Title
				}
			}
			return fmt.Sprintf("%d", pid)
		}
		for pid, m := range agg {
			title := getProjTitle(pid)
			for status, c := range m {
				rows = append(rows, sumRow{Project: title, ProjectID: pid, Status: status, Count: c})
			}
		}
		if len(rows) > 0 {
			// sorting: default by count asc; if --summary-sort name, sort by project name (emoji-stripped), then status
			key := strings.ToLower(strings.TrimSpace(milestoneSummarySort))
			if key == "" {
				key = "count"
			}
			sort.Slice(rows, func(i, j int) bool {
				if key == "name" {
					pi := plainForSort(rows[i].Project)
					pj := plainForSort(rows[j].Project)
					if pi != pj {
						return pi < pj
					}
					si := plainForSort(rows[i].Status)
					sj := plainForSort(rows[j].Status)
					if si != sj {
						return si < sj
					}
					if rows[i].Count != rows[j].Count {
						return rows[i].Count < rows[j].Count
					}
					return rows[i].ProjectID < rows[j].ProjectID
				}
				if rows[i].Count != rows[j].Count {
					return rows[i].Count < rows[j].Count
				}
				pi := plainForSort(rows[i].Project)
				pj := plainForSort(rows[j].Project)
				if pi != pj {
					return pi < pj
				}
				si := plainForSort(rows[i].Status)
				sj := plainForSort(rows[j].Status)
				if si != sj {
					return si < sj
				}
				return rows[i].ProjectID < rows[j].ProjectID
			})

			if format == "tsv" {
				fmt.Println()
				fmt.Println("Summary")
				fmt.Println(strings.Join([]string{"Project", "Status", "Count"}, "\t"))
				for _, r := range rows {
					proj := r.Project
					stat := r.Status
					if milestoneStripEmojis {
						proj = stripEmojis(proj)
						stat = stripEmojis(stat)
					}
					fmt.Println(strings.Join([]string{proj, stat, fmt.Sprintf("%d", r.Count)}, "\t"))
				}
			} else {
				fmt.Println()
				fmt.Println("Summary")
				fmt.Printf("| %s |\n", strings.Join([]string{"Project", "Status", "Count"}, " | "))
				fmt.Printf("| %s |\n", strings.Join([]string{"---", "---", "---"}, " | "))
				for _, r := range rows {
					proj := r.Project
					stat := r.Status
					if milestoneStripEmojis {
						proj = stripEmojis(proj)
						stat = stripEmojis(stat)
					}
					fmt.Printf("| %s | %s | %d |\n", proj, stat, r.Count)
				}
			}
		}
		return nil
	},
}

func init() {
	milestoneCmd.AddCommand(milestoneReportCmd)
	milestoneReportCmd.Flags().StringVar(&milestoneFormat, "format", "tsv", "Output format: tsv (default) or md")
	milestoneReportCmd.Flags().BoolVar(&milestoneStripEmojis, "strip-emojis", false, "Strip emojis from project titles and statuses")
	milestoneReportCmd.Flags().StringVar(&milestoneSummarySort, "summary-sort", "count", "Summary sort: count (default) or name")
	milestoneReportCmd.Flags().BoolVar(&milestoneIncludeClosed, "include-closed", false, "Include closed milestones when listing available milestones")
	milestoneReportCmd.Flags().StringVar(&milestoneIgnoreProject, "ignore-project", "", "Comma-separated substrings to exclude matching project titles (case-insensitive). Example: 'qa,cust' excludes ':help-qa', ':help-customers', and 'Customer requests (open)'.")
	milestoneReportCmd.Flags().StringVar(&milestoneFilterLabels, "filter-labels", "", "Comma-separated list of labels; only issues containing ALL of these labels are included (case-insensitive). Example: 'story,customer-numa'.")
}

// stripEmojis removes common emoji and pictographic characters from a string,
// including variation selectors and zero-width joiners, leaving readable text.
func stripEmojis(s string) string {
	var b strings.Builder
	for _, r := range s {
		// Skip variation selector and zero-width joiners/spaces
		if r == 0xFE0F || r == 0x200D || r == 0x200C || r == 0x200B {
			continue
		}
		// Common emoji blocks and symbols/pictographs
		if (r >= 0x1F300 && r <= 0x1FAFF) || // Misc symbols & pictographs to Supplemental symbols
			(r >= 0x2600 && r <= 0x27BF) { // Misc symbols + Dingbats
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// plainForSort returns a simplified string without emojis, lowercased, for consistent sorting
func plainForSort(s string) string {
	return strings.ToLower(stripEmojis(s))
}

// truncateTitle truncates a string to maxRunes characters (by rune count) and appends
// "..." if the original was longer. The ellipsis is not counted toward maxRunes.
func truncateTitle(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	count := 0
	for idx := range s {
		if count == maxRunes {
			// idx is byte index at rune boundary for the first runes
			return s[:idx] + "..."
		}
		count++
	}
	return s
}
