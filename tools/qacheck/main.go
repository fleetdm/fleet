package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const (
	awaitingQAColumn = "✔️Awaiting QA"
	checkText        = "Engineer: Added comment to user story confirming successful completion of test plan."

	// Drafting board (Project 67) check:
	draftingProjectNum   = 67
	draftingStatusNeedle = "Ready to estimate,Estimated"

	reportDirName  = "qacheck-report"
	reportFileName = "index.html"
)

var draftingChecklistIgnorePrefixes = []string{
	"Once shipped, requester has been notified",
	"Once shipped, dogfooding issue has been filed",
	"Review of all files under server/mdm/microsoft",
	"Review of any files named microsoft_mdm.go",
	"Review of windows_mdm_profiles.go",
	"All Microsoft MDM related endpoints not defined in these files",
}

type Item struct {
	ID githubv4.ID

	Content struct {
		Issue struct {
			Number githubv4.Int
			Title  githubv4.String
			Body   githubv4.String
			URL    githubv4.URI
		} `graphql:"... on Issue"`

		PullRequest struct {
			Number githubv4.Int
			Title  githubv4.String
			Body   githubv4.String
			URL    githubv4.URI
		} `graphql:"... on PullRequest"`
	} `graphql:"content"`

	FieldValues struct {
		Nodes []struct {
			SingleSelectValue struct {
				Name githubv4.String
			} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
		}
	} `graphql:"fieldValues(first: 20)"`
}

type DraftingCheckViolation struct {
	Item      Item
	Unchecked []string
	Status    string
}

type ReportItem struct {
	Number    int
	Title     string
	URL       string
	Unchecked []string
}

type AwaitingProjectReport struct {
	ProjectNum int
	Items      []ReportItem
}

type DraftingStatusReport struct {
	Status string
	Emoji  string
	Intro  string
	Items  []ReportItem
}

type HTMLReportData struct {
	GeneratedAt      string
	Org              string
	AwaitingSections []AwaitingProjectReport
	DraftingSections []DraftingStatusReport
	TotalAwaiting    int
	TotalDrafting    int
}

type intListFlag []int

func (f *intListFlag) String() string {
	if f == nil || len(*f) == 0 {
		return ""
	}
	out := make([]string, 0, len(*f))
	for _, n := range *f {
		out = append(out, strconv.Itoa(n))
	}
	return strings.Join(out, ",")
}

func (f *intListFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return fmt.Errorf("invalid project number %q", part)
		}
		*f = append(*f, n)
	}
	return nil
}

func main() {
	org := flag.String("org", "fleetdm", "GitHub org")
	limit := flag.Int("limit", 100, "Max project items to scan (no pagination; expected usage is small)")
	openReport := flag.Bool("open-report", true, "Open HTML report in browser when finished")
	var projectNums intListFlag
	flag.Var(&projectNums, "project", "Project number(s)")
	flag.Var(&projectNums, "p", "Project number(s) shorthand")
	flag.Parse()

	for _, arg := range flag.Args() {
		n, err := strconv.Atoi(arg)
		if err != nil {
			log.Fatalf("unexpected argument %q: only project numbers are allowed after -p", arg)
		}
		projectNums = append(projectNums, n)
	}

	if len(projectNums) == 0 {
		log.Fatal("at least one project is required")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN env var is required")
	}

	ctx := context.Background()
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := githubv4.NewClient(oauth2.NewClient(ctx, src))

	// Check 1: items in ✔️Awaiting QA with the engineer test-plan confirmation line still unchecked.
	projectNums = uniqueInts(projectNums)
	awaitingByProject := make(map[int][]Item)
	for _, projectNum := range projectNums {
		projectID := fetchProjectID(ctx, client, *org, projectNum)
		items := fetchItems(ctx, client, projectID, *limit)

		var badAwaitingQA []Item
		for _, it := range items {
			if !inAwaitingQA(it) {
				continue
			}
			if hasUncheckedChecklistLine(getBody(it), checkText) {
				badAwaitingQA = append(badAwaitingQA, it)
			}
		}
		awaitingByProject[projectNum] = badAwaitingQA

		fmt.Printf(
			"\nFound %d items in project %d (%q) with UNCHECKED test-plan confirmation:\n\n",
			len(badAwaitingQA),
			projectNum,
			awaitingQAColumn,
		)
		for _, it := range badAwaitingQA {
			fmt.Printf("❌ #%d – %s\n   %s\n\n", getNumber(it), getTitle(it), getURL(it))
		}
	}

	// Check 2: drafting board (project 67) items in Ready to estimate / Estimated with any unchecked checklist line.
	draftingProjectID := fetchProjectID(ctx, client, *org, draftingProjectNum)
	draftingItems := fetchItems(ctx, client, draftingProjectID, *limit)

	needles := strings.Split(draftingStatusNeedle, ",")
	var badDrafting []DraftingCheckViolation
	for _, it := range draftingItems {
		status, ok := matchedStatus(it, needles)
		if !ok {
			continue
		}
		unchecked := uncheckedChecklistItems(getBody(it))
		if len(unchecked) > 0 {
			badDrafting = append(badDrafting, DraftingCheckViolation{
				Item:      it,
				Unchecked: unchecked,
				Status:    status,
			})
		}
	}

	fmt.Printf("\n🧭 Drafting checklist audit (project %d)\n", draftingProjectNum)
	fmt.Printf("Found %d items in estimation columns with unchecked checklist items.\n\n", len(badDrafting))

	byStatus := groupViolationsByStatus(badDrafting)
	printDraftingStatusSection("Ready to estimate", byStatus["ready to estimate"])
	printDraftingStatusSection("Estimated", byStatus["estimated"])

	for status, items := range byStatus {
		if status == "ready to estimate" || status == "estimated" {
			continue
		}
		printDraftingStatusSection(status, items)
	}

	reportPath, err := writeHTMLReport(buildHTMLReportData(*org, projectNums, awaitingByProject, byStatus))
	if err != nil {
		log.Printf("could not write HTML report: %v", err)
		return
	}

	reportURL := fileURLFromPath(reportPath)
	fmt.Printf("📄 HTML report: %s\n", reportPath)
	fmt.Printf("🔗 Open report: %s\n", reportURL)
	fmt.Printf("%s\n", reportURL)
	fmt.Printf("🔗 \x1b]8;;%s\x1b\\Click here to open the report\x1b]8;;\x1b\\\n", reportURL)
	if *openReport {
		if err := openInBrowser(reportPath); err != nil {
			log.Printf("could not auto-open report: %v", err)
			fmt.Printf("Run this manually: open %q\n", reportPath)
		}
	}
}

func fetchProjectID(ctx context.Context, client *githubv4.Client, org string, num int) githubv4.ID {
	var q struct {
		Organization struct {
			ProjectV2 struct {
				ID githubv4.ID
			} `graphql:"projectV2(number: $num)"`
		} `graphql:"organization(login: $org)"`
	}

	err := client.Query(ctx, &q, map[string]interface{}{
		"org": githubv4.String(org),
		"num": githubv4.Int(num),
	})
	if err != nil {
		log.Fatalf("project query failed: %v", err)
	}

	return q.Organization.ProjectV2.ID
}

func fetchItems(
	ctx context.Context,
	client *githubv4.Client,
	projectID githubv4.ID,
	limit int,
) []Item {
	var q struct {
		Node struct {
			ProjectV2 struct {
				Items struct {
					Nodes []Item
				} `graphql:"items(first: $first)"`
			} `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $id)"`
	}

	err := client.Query(ctx, &q, map[string]interface{}{
		"id":    projectID,
		"first": githubv4.Int(limit),
	})
	if err != nil {
		log.Fatalf("items query failed: %v", err)
	}

	if len(q.Node.ProjectV2.Items.Nodes) == limit {
		log.Printf(
			"NOTE: scanned %d items (limit reached, no pagination by design). Increase -limit if needed.",
			limit,
		)
	}

	return q.Node.ProjectV2.Items.Nodes
}

func inAwaitingQA(it Item) bool {
	for _, v := range it.FieldValues.Nodes {
		if string(v.SingleSelectValue.Name) == awaitingQAColumn {
			return true
		}
	}
	return false
}

func matchedStatus(it Item, needles []string) (string, bool) {
	for _, v := range it.FieldValues.Nodes {
		rawName := strings.TrimSpace(string(v.SingleSelectValue.Name))
		name := normalizeStatusName(rawName)
		for _, n := range needles {
			needle := strings.ToLower(strings.TrimSpace(n))
			if needle == "" {
				continue
			}
			if strings.Contains(name, needle) {
				return needle, true
			}
		}
	}
	return "", false
}

// Remove leading emojis/symbols so we can match status names even if the project uses icons.
func normalizeStatusName(s string) string {
	s = strings.TrimSpace(s)
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && size == 1 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			break
		}
		s = strings.TrimSpace(s[size:])
	}
	return strings.ToLower(s)
}

// Only flag if the unchecked checklist line exists.
// Ignore if missing or checked.
func hasUncheckedChecklistLine(body string, text string) bool {
	if body == "" || text == "" {
		return false
	}

	unchecked1 := "- [ ] " + text
	unchecked2 := "[ ] " + text

	checked := []string{
		"- [x] " + text,
		"- [X] " + text,
		"[x] " + text,
		"[X] " + text,
	}

	for _, c := range checked {
		if strings.Contains(body, c) {
			return false
		}
	}

	return strings.Contains(body, unchecked1) || strings.Contains(body, unchecked2)
}

func uncheckedChecklistItems(body string) []string {
	if body == "" {
		return nil
	}

	lines := strings.Split(body, "\n")
	out := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "- [ ] "):
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "- [ ] "))
			if !shouldIgnoreDraftingChecklistItem(text) {
				out = append(out, text)
			}
		case strings.HasPrefix(trimmed, "* [ ] "):
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "* [ ] "))
			if !shouldIgnoreDraftingChecklistItem(text) {
				out = append(out, text)
			}
		case strings.HasPrefix(trimmed, "[ ] "):
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "[ ] "))
			if !shouldIgnoreDraftingChecklistItem(text) {
				out = append(out, text)
			}
		}
	}
	return out
}

func shouldIgnoreDraftingChecklistItem(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	for _, prefix := range draftingChecklistIgnorePrefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

func getBody(it Item) string {
	if it.Content.Issue.Number != 0 {
		return string(it.Content.Issue.Body)
	}
	return string(it.Content.PullRequest.Body)
}

func getTitle(it Item) string {
	if it.Content.Issue.Number != 0 {
		return string(it.Content.Issue.Title)
	}
	return string(it.Content.PullRequest.Title)
}

func getNumber(it Item) int {
	if it.Content.Issue.Number != 0 {
		return int(it.Content.Issue.Number)
	}
	return int(it.Content.PullRequest.Number)
}

func getURL(it Item) string {
	if it.Content.Issue.Number != 0 {
		return it.Content.Issue.URL.String()
	}
	return it.Content.PullRequest.URL.String()
}

func uniqueInts(nums []int) []int {
	seen := make(map[int]bool, len(nums))
	out := make([]int, 0, len(nums))
	for _, n := range nums {
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func groupViolationsByStatus(items []DraftingCheckViolation) map[string][]DraftingCheckViolation {
	out := make(map[string][]DraftingCheckViolation)
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.Status))
		out[key] = append(out[key], item)
	}
	return out
}

func printDraftingStatusSection(status string, items []DraftingCheckViolation) {
	if len(items) == 0 {
		return
	}

	emoji := "📝"
	msg := fmt.Sprintf("These items are in %q but still have checklist items not checked.", status)
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "ready to estimate":
		emoji = "🧩"
		msg = `These items are in "Ready to estimate" but still have checklist items not checked.`
	case "estimated":
		emoji = "📏"
		msg = `These items are in "Estimated" but still have checklist items not checked.`
	}

	fmt.Printf("%s %s\n\n", emoji, msg)
	for _, v := range items {
		it := v.Item
		fmt.Printf("❌ #%d – %s\n   %s\n", getNumber(it), getTitle(it), getURL(it))
		for _, line := range v.Unchecked {
			fmt.Printf("   - [ ] %s\n", line)
		}
		fmt.Println()
	}
}

func buildHTMLReportData(
	org string,
	projectNums []int,
	awaitingByProject map[int][]Item,
	byStatus map[string][]DraftingCheckViolation,
) HTMLReportData {
	sections := make([]AwaitingProjectReport, 0, len(projectNums))
	totalAwaiting := 0
	for _, p := range projectNums {
		items := make([]ReportItem, 0, len(awaitingByProject[p]))
		for _, it := range awaitingByProject[p] {
			items = append(items, ReportItem{
				Number:    getNumber(it),
				Title:     getTitle(it),
				URL:       getURL(it),
				Unchecked: []string{checkText},
			})
		}
		totalAwaiting += len(items)
		sections = append(sections, AwaitingProjectReport{
			ProjectNum: p,
			Items:      items,
		})
	}

	drafting := make([]DraftingStatusReport, 0, len(byStatus))
	totalDrafting := 0
	appendStatus := func(key, status, emoji, intro string) {
		violations, ok := byStatus[key]
		if !ok || len(violations) == 0 {
			return
		}
		items := make([]ReportItem, 0, len(violations))
		for _, v := range violations {
			items = append(items, ReportItem{
				Number:    getNumber(v.Item),
				Title:     getTitle(v.Item),
				URL:       getURL(v.Item),
				Unchecked: v.Unchecked,
			})
		}
		totalDrafting += len(items)
		drafting = append(drafting, DraftingStatusReport{
			Status: status,
			Emoji:  emoji,
			Intro:  intro,
			Items:  items,
		})
	}

	appendStatus(
		"ready to estimate",
		"Ready to estimate",
		"🧩",
		`These items are in "Ready to estimate" but still have checklist items not checked.`,
	)
	appendStatus(
		"estimated",
		"Estimated",
		"📏",
		`These items are in "Estimated" but still have checklist items not checked.`,
	)

	otherKeys := make([]string, 0, len(byStatus))
	for key := range byStatus {
		if key == "ready to estimate" || key == "estimated" {
			continue
		}
		otherKeys = append(otherKeys, key)
	}
	sort.Strings(otherKeys)
	for _, key := range otherKeys {
		display := titleCaseWords(key)
		appendStatus(
			key,
			display,
			"📝",
			fmt.Sprintf("These items are in %q but still have checklist items not checked.", display),
		)
	}

	return HTMLReportData{
		GeneratedAt:      time.Now().Format(time.RFC1123),
		Org:              org,
		AwaitingSections: sections,
		DraftingSections: drafting,
		TotalAwaiting:    totalAwaiting,
		TotalDrafting:    totalDrafting,
	}
}

func writeHTMLReport(data HTMLReportData) (string, error) {
	if err := os.RemoveAll(reportDirName); err != nil {
		return "", fmt.Errorf("remove old report directory: %w", err)
	}
	if err := os.MkdirAll(reportDirName, 0o755); err != nil {
		return "", fmt.Errorf("create report directory: %w", err)
	}

	reportPath := filepath.Join(reportDirName, reportFileName)
	f, err := os.Create(reportPath)
	if err != nil {
		return "", fmt.Errorf("create report file: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("report").Parse(htmlReportTemplate)
	if err != nil {
		return "", fmt.Errorf("parse report template: %w", err)
	}
	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("render report template: %w", err)
	}

	absPath, err := filepath.Abs(reportPath)
	if err != nil {
		return "", fmt.Errorf("resolve report path: %w", err)
	}
	return absPath, nil
}

func fileURLFromPath(path string) string {
	u := url.URL{
		Scheme: "file",
		Path:   path,
	}
	return u.String()
}

func openInBrowser(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	case "linux":
		return exec.Command("xdg-open", path).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", fileURLFromPath(path)).Start()
	default:
		return fmt.Errorf("unsupported OS %q for auto-open", runtime.GOOS)
	}
}

func titleCaseWords(s string) string {
	parts := strings.Fields(strings.ToLower(strings.TrimSpace(s)))
	for i, p := range parts {
		if p == "" {
			continue
		}
		runes := []rune(p)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

var htmlReportTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>qacheck report</title>
  <style>
    :root {
      --bg: #f3f6fb;
      --card: #ffffff;
      --text: #0f172a;
      --muted: #475569;
      --ok: #dbeafe;
      --warn: #fee2e2;
      --line: #cbd5e1;
      --link: #1d4ed8;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      padding: 28px 16px 48px;
      font-family: "Avenir Next", "Segoe UI", Helvetica, Arial, sans-serif;
      background: linear-gradient(180deg, #eef4ff 0%, var(--bg) 60%);
      color: var(--text);
    }
    .wrap { max-width: 1000px; margin: 0 auto; }
    .header, .section {
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 20px;
      box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
    }
    .section { margin-top: 16px; }
    h1 { margin: 0; font-size: 28px; }
    h2 { margin: 0 0 10px; font-size: 20px; }
    h3 { margin: 0 0 6px; font-size: 17px; }
    .meta { margin-top: 8px; color: var(--muted); font-size: 14px; }
    .counts {
      margin-top: 14px;
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
    }
    .pill {
      font-size: 14px;
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: 6px 10px;
      background: #f8fafc;
    }
    .subtle { color: var(--muted); margin: 0 0 12px; font-size: 14px; }
    .project {
      margin-top: 12px;
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 12px;
      background: #f8fafc;
    }
    .item {
      border-left: 4px solid var(--warn);
      background: #fff;
      border-radius: 8px;
      margin: 10px 0 0;
      padding: 10px 12px;
    }
    .item a { color: var(--link); text-decoration: none; }
    .item a:hover { text-decoration: underline; }
    ul { margin: 8px 0 0 20px; }
    li { margin: 5px 0; }
    .status {
      margin-top: 14px;
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 12px;
      background: #f8fafc;
    }
    .empty {
      margin: 0;
      color: var(--muted);
      font-style: italic;
    }
  </style>
</head>
<body>
  <div class="wrap">
    <section class="header">
      <h1>🧪 qacheck report</h1>
      <p class="meta">Org: {{.Org}} | Generated: {{.GeneratedAt}}</p>
      <div class="counts">
        <span class="pill">Awaiting QA violations: {{.TotalAwaiting}}</span>
        <span class="pill">Drafting checklist violations: {{.TotalDrafting}}</span>
      </div>
    </section>

    <section class="section">
      <h2>✅ Awaiting QA gate</h2>
      <p class="subtle">Items in <strong>` + awaitingQAColumn + `</strong> where engineer test-plan confirmation is unchecked.</p>
      {{if .AwaitingSections}}
        {{range .AwaitingSections}}
          <div class="project">
            <h3>Project {{.ProjectNum}}</h3>
            {{if .Items}}
              {{range .Items}}
                <article class="item">
                  <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                  <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                  {{if .Unchecked}}
                    <ul>
                      {{range .Unchecked}}<li>[ ] {{.}}</li>{{end}}
                    </ul>
                  {{end}}
                </article>
              {{end}}
            {{else}}
              <p class="empty">No violations in this project.</p>
            {{end}}
          </div>
        {{end}}
      {{else}}
        <p class="empty">No project data found.</p>
      {{end}}
    </section>

    <section class="section">
      <h2>🧭 Drafting estimation gate (project ` + strconv.Itoa(draftingProjectNum) + `)</h2>
      <p class="subtle">Items in estimation statuses with unchecked checklist items.</p>
      {{if .DraftingSections}}
        {{range .DraftingSections}}
          <div class="status">
            <h3>{{.Emoji}} {{.Status}}</h3>
            <p class="subtle">{{.Intro}}</p>
            {{if .Items}}
              {{range .Items}}
                <article class="item">
                  <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                  <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                  {{if .Unchecked}}
                    <ul>
                      {{range .Unchecked}}<li>[ ] {{.}}</li>{{end}}
                    </ul>
                  {{end}}
                </article>
              {{end}}
            {{else}}
              <p class="empty">No violations in this status.</p>
            {{end}}
          </div>
        {{end}}
      {{else}}
        <p class="empty">No drafting violations.</p>
      {{end}}
    </section>
  </div>
</body>
</html>
`
