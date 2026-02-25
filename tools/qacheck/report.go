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

type MissingMilestoneReportItem struct {
	ProjectNum  int
	Number      int
	Title       string
	URL         string
	Repo        string
	Suggestions []MilestoneSuggestion
}

type MilestoneSuggestion struct {
	Number int
	Title  string
}

type HTMLReportData struct {
	GeneratedAt      string
	Org              string
	GitHubToken      string
	AwaitingSections []AwaitingProjectReport
	StaleSections    []StaleAwaitingProjectReport
	StaleThreshold   int
	DraftingSections []DraftingStatusReport
	MissingMilestone []MissingMilestoneReportItem
	TotalAwaiting    int
	TotalStale       int
	TotalDrafting    int
	TotalNoMilestone int
	TimestampCheck   TimestampCheckResult
	AwaitingClean    bool
	StaleClean       bool
	DraftingClean    bool
	TimestampClean   bool
	MilestoneClean   bool
}

func buildHTMLReportData(
	org string,
	projectNums []int,
	awaitingByProject map[int][]Item,
	staleByProject map[int][]StaleAwaitingViolation,
	staleDays int,
	byStatus map[string][]DraftingCheckViolation,
	missingMilestones []MissingMilestoneIssue,
	timestampCheck TimestampCheckResult,
	githubToken string,
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

	noMilestone := make([]MissingMilestoneReportItem, 0, len(missingMilestones))
	for _, v := range missingMilestones {
		repo := v.RepoOwner + "/" + v.RepoName
		suggestions := make([]MilestoneSuggestion, 0, len(v.SuggestedMilestones))
		for _, m := range v.SuggestedMilestones {
			suggestions = append(suggestions, MilestoneSuggestion{
				Number: m.Number,
				Title:  m.Title,
			})
		}
		noMilestone = append(noMilestone, MissingMilestoneReportItem{
			ProjectNum:  v.ProjectNum,
			Number:      getNumber(v.Item),
			Title:       getTitle(v.Item),
			URL:         getURL(v.Item),
			Repo:        repo,
			Suggestions: suggestions,
		})
	}

	return HTMLReportData{
		GeneratedAt:      time.Now().Format(time.RFC1123),
		Org:              org,
		GitHubToken:      githubToken,
		AwaitingSections: sections,
		StaleSections:    staleSections,
		StaleThreshold:   staleDays,
		DraftingSections: drafting,
		MissingMilestone: noMilestone,
		TotalAwaiting:    totalAwaiting,
		TotalStale:       totalStale,
		TotalDrafting:    totalDrafting,
		TotalNoMilestone: len(noMilestone),
		TimestampCheck:   timestampCheck,
		AwaitingClean:    totalAwaiting == 0,
		StaleClean:       totalStale == 0,
		DraftingClean:    totalDrafting == 0,
		TimestampClean:   timestampCheck.Error == "" && timestampCheck.OK,
		MilestoneClean:   len(noMilestone) == 0,
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
      --line: #cbd5e1;
      --link: #1d4ed8;
      --tab-bg: #f8fafc;
      --tab-active: #e2e8f0;
      --ok: #16a34a;
      --bad: #dc2626;
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
    .header, .panel {
      background: var(--card);
      border: 1px solid var(--line);
      border-radius: 16px;
      padding: 20px;
      box-shadow: 0 8px 24px rgba(15, 23, 42, 0.04);
    }
    h1 { margin: 0; font-size: 28px; }
    h2 { margin: 0 0 10px; font-size: 20px; }
    h3 { margin: 0 0 6px; font-size: 17px; }
    .meta { margin-top: 8px; color: var(--muted); font-size: 14px; }
    .counts { margin-top: 14px; display: flex; flex-wrap: wrap; gap: 10px; }
    .pill {
      font-size: 14px; border: 1px solid var(--line); border-radius: 999px;
      padding: 6px 10px; background: #f8fafc;
    }
    .tabs {
      margin-top: 16px;
      display: flex;
      flex-wrap: nowrap;
      gap: 8px;
      overflow-x: auto;
      padding-bottom: 4px;
    }
    .tab-btn {
      flex: 0 0 auto;
      text-align: left;
      border: 1px solid var(--line);
      background: var(--tab-bg);
      border-radius: 10px;
      padding: 10px 12px;
      cursor: pointer;
      font-size: 14px;
      color: var(--text);
    }
    .tab-btn.active { background: var(--tab-active); }
    .status-dot {
      display: inline-block;
      width: 10px;
      height: 10px;
      border-radius: 50%;
      margin-right: 8px;
      vertical-align: middle;
      background: var(--bad);
      box-shadow: 0 0 0 2px rgba(220, 38, 38, 0.15);
    }
    .status-dot.ok {
      background: var(--ok);
      box-shadow: 0 0 0 2px rgba(22, 163, 74, 0.15);
    }
    .panel-wrap { margin-top: 12px; }
    .panel { display: none; }
    .panel.active { display: block; }
    .subtle { color: var(--muted); margin: 0 0 12px; font-size: 14px; }
    .project, .status {
      margin-top: 12px;
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 12px;
      background: #f8fafc;
    }
    .item {
      border-left: 4px solid #fecaca;
      background: #fff;
      border-radius: 8px;
      margin: 10px 0 0;
      padding: 10px 12px;
    }
    .item a { color: var(--link); text-decoration: none; }
    .item a:hover { text-decoration: underline; }
    ul { margin: 8px 0 0 20px; }
    li { margin: 5px 0; }
    .actions {
      margin-top: 10px;
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
    .fix-btn {
      border: 1px solid var(--line);
      background: #fff;
      border-radius: 8px;
      padding: 6px 10px;
      font-size: 13px;
      color: var(--text);
      cursor: pointer;
    }
    .fix-btn:hover {
      background: #f1f5f9;
    }
    .fix-btn.link {
      text-decoration: none;
      display: inline-block;
    }
    .copied-note {
      margin-left: 6px;
      font-size: 12px;
      color: var(--muted);
    }
    .milestone-search {
      min-width: 240px;
    }
    .milestone-select {
      min-width: 260px;
    }
    .empty { margin: 0; color: var(--muted); font-style: italic; }
  </style>
</head>
<body data-gh-token="{{.GitHubToken}}">
  <div class="wrap">
    <section class="header">
      <h1>🧪 qacheck report</h1>
      <p class="meta">Org: {{.Org}} | Generated: {{.GeneratedAt}}</p>
      <div class="counts">
        <span class="pill">Awaiting QA violations: {{.TotalAwaiting}}</span>
        <span class="pill">Stale Awaiting QA items: {{.TotalStale}}</span>
        <span class="pill">Missing milestones (selected projects): {{.TotalNoMilestone}}</span>
        <span class="pill">Drafting checklist violations: {{.TotalDrafting}}</span>
      </div>
    </section>

    <div class="tabs" role="tablist">
      <button class="tab-btn active" data-tab="awaiting" role="tab">
        <span class="status-dot {{if .AwaitingClean}}ok{{end}}"></span>✔ Awaiting QA
      </button>
      <button class="tab-btn" data-tab="stale" role="tab">
        <span class="status-dot {{if .StaleClean}}ok{{end}}"></span>⏳ Awaiting QA stale watchdog
      </button>
      <button class="tab-btn" data-tab="timestamp" role="tab">
        <span class="status-dot {{if .TimestampClean}}ok{{end}}"></span>🕒 Updates timestamp expiry
      </button>
      <button class="tab-btn" data-tab="milestone" role="tab">
        <span class="status-dot {{if .MilestoneClean}}ok{{end}}"></span>🎯 Missing milestones
      </button>
      <button class="tab-btn" data-tab="drafting" role="tab">
        <span class="status-dot {{if .DraftingClean}}ok{{end}}"></span>🧭 Drafting estimation gate
      </button>
    </div>

    <div class="panel-wrap">
      <section id="tab-awaiting" class="panel active" role="tabpanel">
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
                <p class="empty">🟢 No violations in this project.</p>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No project data found.</p>
        {{end}}
      </section>

      <section id="tab-stale" class="panel" role="tabpanel">
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
                <p class="empty">🟢 No stale items in this project.</p>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No project data found.</p>
        {{end}}
      </section>

      <section id="tab-timestamp" class="panel" role="tabpanel">
        <h2>🕒 Updates timestamp.json expiry</h2>
        <p class="subtle">Checks that <a href="{{.TimestampCheck.URL}}" target="_blank" rel="noopener noreferrer">{{.TimestampCheck.URL}}</a> expires at least {{.TimestampCheck.MinDays}} days from now.</p>
        {{if .TimestampCheck.Error}}
          <p class="empty">🔴 Could not validate timestamp expiry: {{.TimestampCheck.Error}}</p>
        {{else if .TimestampCheck.OK}}
          <div class="project">
            <p><strong>🟢 OK</strong></p>
            <ul>
              <li>Expires: {{.TimestampCheck.ExpiresAt.Format "2006-01-02T15:04:05Z07:00"}}</li>
              <li>Hours remaining: {{printf "%.1f" .TimestampCheck.DurationLeft.Hours}}</li>
              <li>Minimum required days: {{.TimestampCheck.MinDays}}</li>
            </ul>
          </div>
        {{else}}
          <div class="project">
            <p><strong>🔴 Failing threshold</strong></p>
            <ul>
              <li>Expires: {{.TimestampCheck.ExpiresAt.Format "2006-01-02T15:04:05Z07:00"}}</li>
              <li>Hours remaining: {{printf "%.1f" .TimestampCheck.DurationLeft.Hours}}</li>
              <li>Minimum required days: {{.TimestampCheck.MinDays}}</li>
            </ul>
          </div>
        {{end}}
      </section>

      <section id="tab-milestone" class="panel" role="tabpanel">
        <h2>🎯 Missing milestones (selected projects)</h2>
        <p class="subtle">Issues in selected projects without a milestone. Type to filter milestones, choose one, then apply directly.</p>
        {{if .MissingMilestone}}
          {{range .MissingMilestone}}
            <div class="project">
              <h3>Project {{.ProjectNum}} · #{{.Number}} - {{.Title}}</h3>
              <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
              <p class="subtle">Repository: <strong>{{.Repo}}</strong></p>
              <div class="actions">
                {{if .Suggestions}}
                  <input class="fix-btn milestone-search" type="text" placeholder="Search milestone...">
                  <select class="fix-btn milestone-select" data-issue="{{.Number}}" data-repo="{{.Repo}}">
                    {{range .Suggestions}}
                      <option value="{{.Title}}" data-number="{{.Number}}">{{.Title}}</option>
                    {{end}}
                  </select>
                  <button class="fix-btn apply-milestone-btn">Apply milestone</button>
                {{else}}
                  <span class="copied-note">No milestone suggestions found for this repo.</span>
                {{end}}
              </div>
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No missing milestones found.</p>
        {{end}}
      </section>

      <section id="tab-drafting" class="panel" role="tabpanel">
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
                <p class="empty">🟢 No violations in this status.</p>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No drafting violations.</p>
        {{end}}
      </section>
    </div>
  </div>
  <script>
    (function () {
      const buttons = document.querySelectorAll('.tab-btn');
      const panels = document.querySelectorAll('.panel');
      function activate(tabName) {
        buttons.forEach((btn) => {
          btn.classList.toggle('active', btn.dataset.tab === tabName);
        });
        panels.forEach((panel) => {
          panel.classList.toggle('active', panel.id === 'tab-' + tabName);
        });
      }
      buttons.forEach((btn) => {
        btn.addEventListener('click', () => activate(btn.dataset.tab));
      });

      function installMilestoneFiltering() {
        const actionBlocks = document.querySelectorAll('.actions');
        actionBlocks.forEach((actions) => {
          const searchInput = actions.querySelector('.milestone-search');
          const select = actions.querySelector('.milestone-select');
          if (!searchInput || !select) return;

          const allOptions = Array.from(select.options).map((opt) => ({
            title: opt.value,
            number: opt.dataset.number || '',
          }));

          function renderFiltered(term) {
            const q = term.trim().toLowerCase();
            const filtered = allOptions.filter((o) => o.title.toLowerCase().includes(q));
            select.innerHTML = '';
            if (filtered.length === 0) {
              const none = document.createElement('option');
              none.textContent = 'No matching milestones';
              none.value = '';
              none.dataset.number = '';
              select.appendChild(none);
              return;
            }
            filtered.forEach((o) => {
              const opt = document.createElement('option');
              opt.value = o.title;
              opt.textContent = o.title;
              opt.dataset.number = o.number;
              select.appendChild(opt);
            });
          }

          searchInput.addEventListener('input', () => renderFiltered(searchInput.value));
        });
      }

      const applyButtons = document.querySelectorAll('.apply-milestone-btn');
      applyButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          const actions = btn.closest('.actions');
          const select = actions && actions.querySelector('.milestone-select');
          if (!select) return;
          const issue = select.dataset.issue || '';
          const repo = select.dataset.repo || '';
          const milestoneTitle = select.value || '';
          const milestoneNumber = parseInt((select.selectedOptions[0] && select.selectedOptions[0].dataset.number) || '', 10);
          if (!issue || !repo || !milestoneTitle || Number.isNaN(milestoneNumber)) return;

          const token = document.body.dataset.ghToken || '';
          if (!token) {
            window.alert('Missing GitHub token in report data. Re-run qacheck with GITHUB_TOKEN set.');
            return;
          }

          const endpoint = 'https://api.github.com/repos/' + repo + '/issues/' + issue;
          const payload = { milestone: milestoneNumber };
          const prev = btn.textContent;
          btn.textContent = 'Applying...';
          btn.disabled = true;
          try {
            const res = await fetch(endpoint, {
              method: 'PATCH',
              headers: {
                'Accept': 'application/vnd.github+json',
                'Content-Type': 'application/json',
                'Authorization': 'Bearer ' + token,
              },
              body: JSON.stringify(payload),
            });
            if (!res.ok) {
              const body = await res.text();
              throw new Error('GitHub API error ' + res.status + ': ' + body);
            }
            btn.textContent = 'Applied';
            setTimeout(() => { btn.textContent = prev; }, 1200);
          } catch (err) {
            window.alert('Could not apply milestone. ' + err);
            btn.textContent = prev;
          } finally {
            btn.disabled = false;
          }
        });
      });

      installMilestoneFiltering();
    })();
  </script>
</body>
</html>
`
