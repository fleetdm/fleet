package main

import (
	"fmt"
	"html/template"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode"
)

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

type StaleAwaitingProjectReport struct {
	ProjectNum int
	Items      []StaleAwaitingReportItem
}

type StaleAwaitingReportItem struct {
	Number      int
	Title       string
	URL         string
	LastUpdated string
	StaleDays   int
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
	StaleSections    []StaleAwaitingProjectReport
	StaleThreshold   int
	DraftingSections []DraftingStatusReport
	TotalAwaiting    int
	TotalStale       int
	TotalDrafting    int
	TimestampCheck   TimestampCheckResult
}

func buildHTMLReportData(
	org string,
	projectNums []int,
	awaitingByProject map[int][]Item,
	staleByProject map[int][]StaleAwaitingViolation,
	staleDays int,
	byStatus map[string][]DraftingCheckViolation,
	timestampCheck TimestampCheckResult,
) HTMLReportData {
	sections := make([]AwaitingProjectReport, 0, len(projectNums))
	totalAwaiting := 0
	staleSections := make([]StaleAwaitingProjectReport, 0, len(projectNums))
	totalStale := 0
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

		staleItems := make([]StaleAwaitingReportItem, 0, len(staleByProject[p]))
		for _, v := range staleByProject[p] {
			staleItems = append(staleItems, StaleAwaitingReportItem{
				Number:      getNumber(v.Item),
				Title:       getTitle(v.Item),
				URL:         getURL(v.Item),
				LastUpdated: v.LastUpdated.Format("2006-01-02"),
				StaleDays:   v.StaleDays,
			})
		}
		totalStale += len(staleItems)
		staleSections = append(staleSections, StaleAwaitingProjectReport{
			ProjectNum: p,
			Items:      staleItems,
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
		StaleSections:    staleSections,
		StaleThreshold:   staleDays,
		DraftingSections: drafting,
		TotalAwaiting:    totalAwaiting,
		TotalStale:       totalStale,
		TotalDrafting:    totalDrafting,
		TimestampCheck:   timestampCheck,
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
        <span class="pill">Stale Awaiting QA items: {{.TotalStale}}</span>
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
      <h2>⏳ Awaiting QA stale watchdog</h2>
      <p class="subtle">Items in <strong>` + awaitingQAColumn + `</strong> with no updates for at least {{.StaleThreshold}} days.</p>
      {{if .StaleSections}}
        {{range .StaleSections}}
          <div class="project">
            <h3>Project {{.ProjectNum}}</h3>
            {{if .Items}}
              {{range .Items}}
                <article class="item">
                  <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                  <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                  <ul>
                    <li>Last updated: {{.LastUpdated}}</li>
                    <li>Age: {{.StaleDays}} days</li>
                  </ul>
                </article>
              {{end}}
            {{else}}
              <p class="empty">No stale items in this project.</p>
            {{end}}
          </div>
        {{end}}
      {{else}}
        <p class="empty">No project data found.</p>
      {{end}}
    </section>

    <section class="section">
      <h2>🕒 Updates timestamp.json expiry</h2>
      <p class="subtle">Checks that <a href="{{.TimestampCheck.URL}}" target="_blank" rel="noopener noreferrer">{{.TimestampCheck.URL}}</a> expires at least {{.TimestampCheck.MinDays}} days from now.</p>
      {{if .TimestampCheck.Error}}
        <p class="empty">Could not validate timestamp expiry: {{.TimestampCheck.Error}}</p>
      {{else if .TimestampCheck.OK}}
        <div class="project">
          <p><strong>✅ OK</strong></p>
          <ul>
            <li>Expires: {{.TimestampCheck.ExpiresAt.Format "2006-01-02T15:04:05Z07:00"}}</li>
            <li>Hours remaining: {{printf "%.1f" .TimestampCheck.DurationLeft.Hours}}</li>
            <li>Minimum required days: {{.TimestampCheck.MinDays}}</li>
          </ul>
        </div>
      {{else}}
        <div class="project">
          <p><strong>❌ Failing threshold</strong></p>
          <ul>
            <li>Expires: {{.TimestampCheck.ExpiresAt.Format "2006-01-02T15:04:05Z07:00"}}</li>
            <li>Hours remaining: {{printf "%.1f" .TimestampCheck.DurationLeft.Hours}}</li>
            <li>Minimum required days: {{.TimestampCheck.MinDays}}</li>
          </ul>
        </div>
      {{end}}
    </section>

    <section class="section">
      <h2>🧭 Drafting estimation gate (project ` + fmt.Sprintf("%d", draftingProjectNum) + `)</h2>
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
