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
	Repo      string
	Title     string
	URL       string
	Assignees []string
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
	Status      string
	Assignees   []string
	Labels      []string
	BodyPreview []string
	Suggestions []MilestoneSuggestion
}

type MissingMilestoneGroupReport struct {
	Key   string
	Label string
	Items []MissingMilestoneReportItem
}

type MissingMilestoneProjectReport struct {
	ProjectNum int
	Columns    []MissingMilestoneGroupReport
}

type MissingSprintReportItem struct {
	ProjectNum    int
	ItemID        string
	Number        int
	Title         string
	URL           string
	Status        string
	CurrentSprint string
	Milestone     string
	Assignees     []string
	Labels        []string
	BodyPreview   []string
}

type MissingSprintGroupReport struct {
	Key   string
	Label string
	Items []MissingSprintReportItem
}

type MissingSprintProjectReport struct {
	ProjectNum int
	Columns    []MissingSprintGroupReport
}

type AssigneeSuggestion struct {
	Login string
}

type MissingAssigneeReportItem struct {
	ProjectNum         int
	Number             int
	Title              string
	URL                string
	Repo               string
	Status             string
	AssignedToMe       bool
	CurrentAssignees   []string
	SuggestedAssignees []AssigneeSuggestion
}

type MissingAssigneeGroupReport struct {
	Key   string
	Label string
	Items []MissingAssigneeReportItem
}

type MissingAssigneeProjectReport struct {
	ProjectNum int
	Columns    []MissingAssigneeGroupReport
}

type ReleaseLabelReportItem struct {
	ProjectNum    int
	Number        int
	Title         string
	URL           string
	Repo          string
	Status        string
	CurrentLabels []string
}

type ReleaseLabelProjectReport struct {
	ProjectNum int
	Items      []ReleaseLabelReportItem
}

type UnassignedUnreleasedProjectReport struct {
	GroupLabel string
	Columns    []UnassignedUnreleasedStatusReport
}

type UnassignedUnreleasedStatusReport struct {
	Key        string
	Label      string
	RedItems   []MissingMilestoneReportItem
	GreenItems []MissingMilestoneReportItem
}

type MilestoneSuggestion struct {
	Number int
	Title  string
}

type HTMLReportData struct {
	GeneratedAt               string
	Org                       string
	BridgeEnabled             bool
	BridgeBaseURL             string
	BridgeSessionToken        string
	AwaitingSections          []AwaitingProjectReport
	StaleSections             []StaleAwaitingProjectReport
	StaleThreshold            int
	DraftingSections          []DraftingStatusReport
	MissingMilestone          []MissingMilestoneProjectReport
	MissingSprint             []MissingSprintProjectReport
	MissingAssignee           []MissingAssigneeProjectReport
	AssignedToMe              []MissingAssigneeProjectReport
	ReleaseLabel              []ReleaseLabelProjectReport
	UnassignedUnreleased      []UnassignedUnreleasedProjectReport
	TotalAwaiting             int
	TotalStale                int
	TotalDrafting             int
	TotalNoMilestone          int
	TotalNoSprint             int
	TotalMissingAssignee      int
	TotalAssignedToMe         int
	TotalRelease              int
	TotalUnassignedUnreleased int
	TotalTrackedUnreleased    int
	TimestampCheck            TimestampCheckResult
	AwaitingClean             bool
	StaleClean                bool
	DraftingClean             bool
	TimestampClean            bool
	MilestoneClean            bool
	SprintClean               bool
	MissingAssigneeClean      bool
	AssignedToMeClean         bool
	ReleaseClean              bool
	UnassignedUnreleasedClean bool
}

func buildHTMLReportData(
	org string,
	projectNums []int,
	awaitingByProject map[int][]Item,
	staleByProject map[int][]StaleAwaitingViolation,
	staleDays int,
	byStatus map[string][]DraftingCheckViolation,
	missingMilestones []MissingMilestoneIssue,
	missingSprints []MissingSprintViolation,
	missingAssignees []MissingAssigneeIssue,
	releaseIssues []ReleaseLabelIssue,
	unassignedUnreleased []UnassignedUnreleasedBugIssue,
	groupLabels []string,
	timestampCheck TimestampCheckResult,
	bridgeEnabled bool,
	bridgeBaseURL string,
	bridgeSessionToken string,
) HTMLReportData {
	sections := make([]AwaitingProjectReport, 0, len(projectNums))
	totalAwaiting := 0
	staleSections := make([]StaleAwaitingProjectReport, 0, len(projectNums))
	totalStale := 0
	for _, p := range projectNums {
		items := make([]ReportItem, 0, len(awaitingByProject[p]))
		for _, it := range awaitingByProject[p] {
			owner, repo := parseRepoFromIssueURL(getURL(it))
			items = append(items, ReportItem{
				Number:    getNumber(it),
				Repo:      owner + "/" + repo,
				Title:     getTitle(it),
				URL:       getURL(it),
				Assignees: issueAssignees(it),
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
			owner, repo := parseRepoFromIssueURL(getURL(v.Item))
			items = append(items, ReportItem{
				Number:    getNumber(v.Item),
				Repo:      owner + "/" + repo,
				Title:     getTitle(v.Item),
				URL:       getURL(v.Item),
				Assignees: issueAssignees(v.Item),
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

	groupedMilestoneByProject := make(map[int]map[string][]MissingMilestoneReportItem)
	for _, p := range projectNums {
		groupedMilestoneByProject[p] = make(map[string][]MissingMilestoneReportItem)
		for _, key := range sprintColumnOrder() {
			groupedMilestoneByProject[p][key] = []MissingMilestoneReportItem{}
		}
	}
	for _, v := range missingMilestones {
		repo := v.RepoOwner + "/" + v.RepoName
		suggestions := make([]MilestoneSuggestion, 0, len(v.SuggestedMilestones))
		for _, m := range v.SuggestedMilestones {
			suggestions = append(suggestions, MilestoneSuggestion{
				Number: m.Number,
				Title:  m.Title,
			})
		}
		status := itemStatus(v.Item)
		group := sprintColumnGroup(status)
		groupedMilestoneByProject[v.ProjectNum][group] = append(groupedMilestoneByProject[v.ProjectNum][group], MissingMilestoneReportItem{
			ProjectNum:  v.ProjectNum,
			Number:      getNumber(v.Item),
			Title:       getTitle(v.Item),
			URL:         getURL(v.Item),
			Repo:        repo,
			Status:      status,
			Assignees:   issueAssignees(v.Item),
			Labels:      issueLabels(v.Item),
			BodyPreview: previewBodyLines(getBody(v.Item), 3),
			Suggestions: suggestions,
		})
	}
	milestoneProjects := make([]MissingMilestoneProjectReport, 0, len(groupedMilestoneByProject))
	totalNoMilestone := 0
	projectNumsForMilestones := append([]int(nil), projectNums...)
	for _, p := range projectNumsForMilestones {
		grouped := groupedMilestoneByProject[p]
		projectTotal := 0
		for _, key := range sprintColumnOrder() {
			items := grouped[key]
			projectTotal += len(items)
			totalNoMilestone += len(items)
		}
		columns := make([]MissingMilestoneGroupReport, 0, len(sprintColumnOrder()))
		for _, key := range sprintColumnOrder() {
			items := grouped[key]
			columns = append(columns, MissingMilestoneGroupReport{
				Key:   key,
				Label: sprintColumnLabel(key),
				Items: items,
			})
		}
		milestoneProjects = append(milestoneProjects, MissingMilestoneProjectReport{
			ProjectNum: p,
			Columns:    columns,
		})
	}

	sprintOrder := sprintColumnsWithoutReadyForRelease()
	groupedSprintByProject := make(map[int]map[string][]MissingSprintReportItem)
	for _, p := range projectNums {
		groupedSprintByProject[p] = make(map[string][]MissingSprintReportItem)
		for _, key := range sprintOrder {
			groupedSprintByProject[p][key] = []MissingSprintReportItem{}
		}
	}
	for _, v := range missingSprints {
		itemID := fmt.Sprintf("%v", v.ItemID)
		group := sprintColumnGroup(v.Status)
		if group == "ready_for_release" {
			continue
		}
		groupedSprintByProject[v.ProjectNum][group] = append(groupedSprintByProject[v.ProjectNum][group], MissingSprintReportItem{
			ProjectNum:    v.ProjectNum,
			ItemID:        itemID,
			Number:        getNumber(v.Item),
			Title:         getTitle(v.Item),
			URL:           getURL(v.Item),
			Status:        v.Status,
			CurrentSprint: v.CurrentSprint,
			Milestone:     strings.TrimSpace(string(v.Item.Content.Issue.Milestone.Title)),
			Assignees:     issueAssignees(v.Item),
			Labels:        issueLabels(v.Item),
			BodyPreview:   previewBodyLines(getBody(v.Item), 3),
		})
	}
	sprintProjects := make([]MissingSprintProjectReport, 0, len(groupedSprintByProject))
	totalNoSprint := 0
	projectNumsForSprint := append([]int(nil), projectNums...)
	for _, p := range projectNumsForSprint {
		grouped := groupedSprintByProject[p]
		for _, key := range sprintOrder {
			items := grouped[key]
			totalNoSprint += len(items)
		}
		columns := make([]MissingSprintGroupReport, 0, len(sprintOrder))
		for _, key := range sprintOrder {
			items := grouped[key]
			columns = append(columns, MissingSprintGroupReport{
				Key:   key,
				Label: sprintColumnLabel(key),
				Items: items,
			})
		}
		sprintProjects = append(sprintProjects, MissingSprintProjectReport{
			ProjectNum: p,
			Columns:    columns,
		})
	}

	groupedMissingAssigneeByProject := make(map[int]map[string][]MissingAssigneeReportItem)
	groupedAssignedToMeByProject := make(map[int]map[string][]MissingAssigneeReportItem)
	for _, p := range projectNums {
		groupedMissingAssigneeByProject[p] = make(map[string][]MissingAssigneeReportItem)
		groupedAssignedToMeByProject[p] = make(map[string][]MissingAssigneeReportItem)
		for _, key := range sprintColumnOrder() {
			groupedMissingAssigneeByProject[p][key] = []MissingAssigneeReportItem{}
			groupedAssignedToMeByProject[p][key] = []MissingAssigneeReportItem{}
		}
	}
	for _, v := range missingAssignees {
		repo := v.RepoOwner + "/" + v.RepoName
		status := itemStatus(v.Item)
		group := sprintColumnGroup(status)
		suggestions := make([]AssigneeSuggestion, 0, len(v.SuggestedAssignees))
		for _, a := range v.SuggestedAssignees {
			login := strings.TrimSpace(a.Login)
			if login == "" {
				continue
			}
			suggestions = append(suggestions, AssigneeSuggestion{Login: login})
		}
		item := MissingAssigneeReportItem{
			ProjectNum:         v.ProjectNum,
			Number:             getNumber(v.Item),
			Title:              getTitle(v.Item),
			URL:                getURL(v.Item),
			Repo:               repo,
			Status:             status,
			AssignedToMe:       v.AssignedToMe,
			CurrentAssignees:   append([]string(nil), v.CurrentAssignees...),
			SuggestedAssignees: suggestions,
		}
		if v.AssignedToMe {
			groupedAssignedToMeByProject[v.ProjectNum][group] = append(groupedAssignedToMeByProject[v.ProjectNum][group], item)
			continue
		}
		groupedMissingAssigneeByProject[v.ProjectNum][group] = append(groupedMissingAssigneeByProject[v.ProjectNum][group], item)
	}
	buildAssigneeProjects := func(groupedByProject map[int]map[string][]MissingAssigneeReportItem) ([]MissingAssigneeProjectReport, int) {
		projects := make([]MissingAssigneeProjectReport, 0, len(groupedByProject))
		total := 0
		projectNumsForAssignees := append([]int(nil), projectNums...)
		for _, p := range projectNumsForAssignees {
			grouped := groupedByProject[p]
			for _, key := range sprintColumnOrder() {
				total += len(grouped[key])
			}
			columns := make([]MissingAssigneeGroupReport, 0, len(sprintColumnOrder()))
			for _, key := range sprintColumnOrder() {
				items := grouped[key]
				columns = append(columns, MissingAssigneeGroupReport{
					Key:   key,
					Label: sprintColumnLabel(key),
					Items: items,
				})
			}
			projects = append(projects, MissingAssigneeProjectReport{
				ProjectNum: p,
				Columns:    columns,
			})
		}
		return projects, total
	}
	missingAssigneeProjects, totalMissingAssignee := buildAssigneeProjects(groupedMissingAssigneeByProject)
	assignedToMeProjects, totalAssignedToMe := buildAssigneeProjects(groupedAssignedToMeByProject)

	groupedReleaseByProject := make(map[int][]ReleaseLabelReportItem)
	for _, v := range releaseIssues {
		repo := v.RepoOwner + "/" + v.RepoName
		groupedReleaseByProject[v.ProjectNum] = append(groupedReleaseByProject[v.ProjectNum], ReleaseLabelReportItem{
			ProjectNum:    v.ProjectNum,
			Number:        getNumber(v.Item),
			Title:         getTitle(v.Item),
			URL:           getURL(v.Item),
			Repo:          repo,
			Status:        itemStatus(v.Item),
			CurrentLabels: append([]string(nil), v.CurrentLabels...),
		})
	}
	releaseProjects := make([]ReleaseLabelProjectReport, 0, len(groupedReleaseByProject))
	totalRelease := 0
	projectNumsForRelease := make([]int, 0, len(groupedReleaseByProject))
	for p := range groupedReleaseByProject {
		projectNumsForRelease = append(projectNumsForRelease, p)
	}
	sort.Ints(projectNumsForRelease)
	for _, p := range projectNumsForRelease {
		items := groupedReleaseByProject[p]
		if len(items) == 0 {
			continue
		}
		totalRelease += len(items)
		releaseProjects = append(releaseProjects, ReleaseLabelProjectReport{
			ProjectNum: p,
			Items:      items,
		})
	}

	groupedUnassignedByLabel := make(map[string]map[string]UnassignedUnreleasedStatusReport)
	for _, label := range groupLabels {
		groupedUnassignedByLabel[label] = make(map[string]UnassignedUnreleasedStatusReport)
		for _, key := range sprintColumnOrder() {
			groupedUnassignedByLabel[label][key] = UnassignedUnreleasedStatusReport{
				Key:        key,
				Label:      sprintColumnLabel(key),
				RedItems:   []MissingMilestoneReportItem{},
				GreenItems: []MissingMilestoneReportItem{},
			}
		}
	}
	for _, v := range unassignedUnreleased {
		repo := v.RepoOwner + "/" + v.RepoName
		status := strings.TrimSpace(v.Status)
		if status == "" {
			status = itemStatus(v.Item)
		}
		statusKey := sprintColumnGroup(status)
		item := MissingMilestoneReportItem{
			ProjectNum:  v.ProjectNum,
			Number:      getNumber(v.Item),
			Title:       getTitle(v.Item),
			URL:         getURL(v.Item),
			Repo:        repo,
			Status:      status,
			Assignees:   append([]string(nil), v.CurrentAssignees...),
			Labels:      append([]string(nil), v.CurrentLabels...),
			BodyPreview: previewBodyLines(getBody(v.Item), 3),
		}
		for _, groupLabel := range v.MatchingGroups {
			columnsByStatus, ok := groupedUnassignedByLabel[groupLabel]
			if !ok {
				continue
			}
			statusGroup := columnsByStatus[statusKey]
			if v.Unassigned {
				statusGroup.RedItems = append(statusGroup.RedItems, item)
			} else {
				statusGroup.GreenItems = append(statusGroup.GreenItems, item)
			}
			columnsByStatus[statusKey] = statusGroup
			groupedUnassignedByLabel[groupLabel] = columnsByStatus
		}
	}
	unassignedProjects := make([]UnassignedUnreleasedProjectReport, 0, len(groupedUnassignedByLabel))
	totalUnassignedUnreleased := 0
	totalTrackedUnreleased := 0
	for _, label := range groupLabels {
		columnsByStatus, ok := groupedUnassignedByLabel[label]
		if !ok {
			continue
		}
		columns := make([]UnassignedUnreleasedStatusReport, 0, len(sprintColumnOrder()))
		for _, key := range sprintColumnOrder() {
			group := columnsByStatus[key]
			totalUnassignedUnreleased += len(group.RedItems)
			totalTrackedUnreleased += len(group.GreenItems)
			columns = append(columns, group)
		}
		unassignedProjects = append(unassignedProjects, UnassignedUnreleasedProjectReport{
			GroupLabel: label,
			Columns:    columns,
		})
	}

	return HTMLReportData{
		GeneratedAt:               time.Now().Format(time.RFC1123),
		Org:                       org,
		BridgeEnabled:             bridgeEnabled,
		BridgeBaseURL:             bridgeBaseURL,
		BridgeSessionToken:        bridgeSessionToken,
		AwaitingSections:          sections,
		StaleSections:             staleSections,
		StaleThreshold:            staleDays,
		DraftingSections:          drafting,
		MissingMilestone:          milestoneProjects,
		MissingSprint:             sprintProjects,
		MissingAssignee:           missingAssigneeProjects,
		AssignedToMe:              assignedToMeProjects,
		ReleaseLabel:              releaseProjects,
		UnassignedUnreleased:      unassignedProjects,
		TotalAwaiting:             totalAwaiting,
		TotalStale:                totalStale,
		TotalDrafting:             totalDrafting,
		TotalNoMilestone:          totalNoMilestone,
		TotalNoSprint:             totalNoSprint,
		TotalMissingAssignee:      totalMissingAssignee,
		TotalAssignedToMe:         totalAssignedToMe,
		TotalRelease:              totalRelease,
		TotalUnassignedUnreleased: totalUnassignedUnreleased,
		TotalTrackedUnreleased:    totalTrackedUnreleased,
		TimestampCheck:            timestampCheck,
		AwaitingClean:             totalAwaiting == 0,
		StaleClean:                totalStale == 0,
		DraftingClean:             totalDrafting == 0,
		TimestampClean:            timestampCheck.Error == "" && timestampCheck.OK,
		MilestoneClean:            totalNoMilestone == 0,
		SprintClean:               totalNoSprint == 0,
		MissingAssigneeClean:      totalMissingAssignee == 0,
		AssignedToMeClean:         totalAssignedToMe == 0,
		ReleaseClean:              totalRelease == 0,
		UnassignedUnreleasedClean: totalUnassignedUnreleased == 0,
	}
}

func itemStatus(it Item) string {
	for _, v := range it.FieldValues.Nodes {
		fieldName := strings.TrimSpace(strings.ToLower(string(v.SingleSelectValue.Field.Common.Name)))
		if fieldName != "status" {
			continue
		}
		name := strings.TrimSpace(string(v.SingleSelectValue.Name))
		if name != "" {
			return name
		}
	}
	return ""
}

func previewBodyLines(body string, maxLines int) []string {
	if maxLines <= 0 {
		return nil
	}
	lines := strings.Split(body, "\n")
	out := make([]string, 0, maxLines)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
		if len(out) >= maxLines {
			break
		}
	}
	return out
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
      --bg: #f5f7fb;
      --header: #ffffff;
      --panel: #ffffff;
      --muted-panel: #f9fafc;
      --text: #192147;
      --muted: #515774;
      --line: #d9dce8;
      --accent: #00a794;
      --accent-dark: #0f7f73;
      --ok: #27ae60;
      --bad: #eb5757;
      --active: #eaf8f6;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Avenir Next", "Segoe UI", Helvetica, Arial, sans-serif;
      background: var(--bg);
      color: var(--text);
    }
    .wrap {
      max-width: 1360px;
      margin: 0 auto;
      padding: 20px;
    }
    .header {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 20px;
    }
    .title-row {
      display: flex;
      align-items: center;
      gap: 12px;
    }
    .title-logo {
      width: 28px;
      height: 28px;
      flex: 0 0 auto;
      display: inline-flex;
      align-items: center;
      justify-content: center;
    }
    .title-logo svg {
      display: block;
      width: 26px;
      height: 26px;
    }
    h1 { margin: 0; font-size: 38px; line-height: 1.1; }
    h2 { margin: 0 0 10px; font-size: 27px; line-height: 1.2; }
    h3 { margin: 0 0 6px; font-size: 18px; line-height: 1.25; }
    .meta { margin-top: 8px; color: var(--muted); font-size: 14px; }
    .counts { margin-top: 14px; display: flex; flex-wrap: wrap; gap: 10px; }
    .pill {
      font-size: 14px;
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: 6px 12px;
      background: var(--muted-panel);
    }
    .app-shell {
      margin-top: 16px;
      display: grid;
      grid-template-columns: 280px minmax(0, 1fr);
      gap: 16px;
      align-items: start;
    }
    .menu {
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 10px;
      position: sticky;
      top: 12px;
    }
    .menu h4 {
      margin: 4px 8px 10px;
      font-size: 12px;
      letter-spacing: 0.04em;
      color: var(--muted);
      text-transform: uppercase;
    }
    .menu-btn {
      width: 100%;
      text-align: left;
      border: 1px solid transparent;
      background: transparent;
      border-radius: 8px;
      padding: 10px 10px;
      cursor: pointer;
      font-size: 14px;
      color: var(--text);
      margin-bottom: 6px;
    }
    .menu-btn:hover {
      background: #f2f5fb;
      border-color: var(--line);
    }
    .menu-btn.active {
      background: var(--active);
      border-color: rgba(0, 167, 148, 0.35);
    }
    .status-dot {
      display: inline-block;
      width: 10px;
      height: 10px;
      border-radius: 50%;
      margin-right: 8px;
      vertical-align: middle;
      background: var(--bad);
      box-shadow: 0 0 0 2px rgba(235, 87, 87, 0.18);
    }
    .status-dot.ok {
      background: var(--ok);
      box-shadow: 0 0 0 2px rgba(39, 174, 96, 0.16);
    }
    .panel-wrap { min-width: 0; }
    .panel {
      display: none;
      background: var(--panel);
      border: 1px solid var(--line);
      border-radius: 12px;
      padding: 18px;
    }
    .panel.active { display: block; }
    .subtle { color: var(--muted); margin: 0 0 12px; font-size: 14px; }
    .project, .status {
      margin-top: 12px;
      border: 1px solid var(--line);
      border-radius: 10px;
      padding: 12px;
      background: var(--muted-panel);
    }
    .item {
      border-left: 3px solid #f5c5c5;
      background: #fff;
      border-radius: 8px;
      margin: 10px 0 0;
      padding: 10px 12px;
    }
    .item.assigned-to-me {
      border-left-color: #72c08a;
      box-shadow: inset 0 0 0 1px rgba(39, 174, 96, 0.2);
      background: #f4fbf6;
    }
    .item.red-bug {
      border-left-color: #eb5757;
      box-shadow: inset 0 0 0 1px rgba(235, 87, 87, 0.25);
      background: #fff3f3;
    }
    .item.green-bug {
      border-left-color: #27ae60;
      box-shadow: inset 0 0 0 1px rgba(39, 174, 96, 0.2);
      background: #f4fbf6;
    }
    .mine-badge {
      display: inline-block;
      margin-top: 8px;
      padding: 4px 8px;
      border-radius: 999px;
      border: 1px solid #83d2a5;
      background: #e9f8ef;
      color: #1d6e3e;
      font-size: 12px;
      font-weight: 600;
    }
    .item a { color: #2f45cc; text-decoration: none; }
    .item a:hover { text-decoration: underline; }
    ul { margin: 8px 0 0 20px; }
    li { margin: 5px 0; }
    .checklist-row {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
      gap: 8px;
      margin: 6px 0;
    }
    .checklist-text { flex: 1 1 320px; }
    .actions {
      margin-top: 10px;
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
    .column-head {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 10px;
      margin-bottom: 6px;
    }
    .fix-btn {
      border: 1px solid #b8bfd5;
      background: #fff;
      border-radius: 8px;
      padding: 7px 12px;
      font-size: 13px;
      color: var(--text);
      cursor: pointer;
    }
    .fix-btn:hover { background: #f2f5fb; }
    .fix-btn.done {
      border-color: #83d2a5;
      background: #e9f8ef;
      color: #1d6e3e;
      cursor: default;
    }
    .fix-btn.failed {
      border-color: #e3a0a0;
      background: #fff1f1;
      color: #8e1b1b;
    }
    #close-session-btn {
      border-color: #e9a5a5;
      background: #fff2f2;
      color: #8b1e1e;
    }
    #close-session-btn:hover {
      background: #ffe8e8;
    }
    .copied-note { margin-left: 6px; font-size: 12px; color: var(--muted); }
    .milestone-search { min-width: 240px; }
    .milestone-select { min-width: 260px; }
    .assignee-search { min-width: 240px; }
    .assignee-select { min-width: 260px; }
    .bridge-controls {
      margin-top: 10px;
      display: flex;
      gap: 8px;
      align-items: center;
      flex-wrap: wrap;
    }
    .empty { margin: 0; color: var(--muted); font-style: italic; }
    @media (max-width: 960px) {
      h1 { font-size: 30px; }
      h2 { font-size: 23px; }
      h3 { font-size: 18px; }
      .app-shell {
        grid-template-columns: 1fr;
      }
      .menu {
        position: static;
      }
    }
  </style>
</head>
<body data-bridge-url="{{.BridgeBaseURL}}" data-bridge-session="{{.BridgeSessionToken}}">
  <div class="wrap">
    <section class="header">
      <div class="title-row">
        <span class="title-logo" aria-hidden="true">
          <svg viewBox="0 0 24 24" role="img" aria-label="Fleet logo">
            <circle cx="4" cy="4" r="2.2" fill="#8b6ff0"></circle>
            <circle cx="12" cy="4" r="2.2" fill="#34c759"></circle>
            <circle cx="20" cy="4" r="2.2" fill="#eb5757"></circle>
            <circle cx="4" cy="12" r="2.2" fill="#4da3ff"></circle>
            <circle cx="12" cy="12" r="2.2" fill="#ff8a3d"></circle>
            <circle cx="4" cy="20" r="2.2" fill="#54d58f"></circle>
          </svg>
        </span>
        <h1>Scrum check</h1>
      </div>
      <p class="meta">Org: {{.Org}} | Generated: {{.GeneratedAt}}</p>
      <div class="counts">
        <span class="pill">Awaiting QA violations: {{.TotalAwaiting}}</span>
        <span class="pill">Stale Awaiting QA items: {{.TotalStale}}</span>
        <span class="pill">Missing milestones (selected projects): {{.TotalNoMilestone}}</span>
        <span class="pill">Missing sprint (selected projects): {{.TotalNoSprint}}</span>
        <span class="pill">Missing assignee (selected projects): {{.TotalMissingAssignee}}</span>
        <span class="pill">Assigned to me (selected projects): {{.TotalAssignedToMe}}</span>
        <span class="pill">Unassigned unreleased bugs (selected projects): {{.TotalUnassignedUnreleased}}</span>
        <span class="pill">Tracked unreleased bugs (assigned): {{.TotalTrackedUnreleased}}</span>
        <span class="pill">Release label issues (selected projects): {{.TotalRelease}}</span>
        <span class="pill">Drafting checklist violations: {{.TotalDrafting}}</span>
      </div>
      {{if .BridgeEnabled}}
        <div class="bridge-controls">
          <button id="close-session-btn" class="fix-btn">Close bridge session</button>
          <span class="copied-note">This will stop GitHub actions from the report until next qacheck run.</span>
        </div>
      {{end}}
    </section>

    <div class="app-shell">
      <aside class="menu" role="tablist" aria-label="checks">
        <h4>Checks</h4>
        <button class="menu-btn active" data-tab="sprint" role="tab">
          <span class="status-dot {{if .SprintClean}}ok{{end}}"></span>🗓️ Missing sprint
        </button>
        <button class="menu-btn" data-tab="milestone" role="tab">
          <span class="status-dot {{if .MilestoneClean}}ok{{end}}"></span>🎯 Missing milestones
        </button>
        <button class="menu-btn" data-tab="release" role="tab">
          <span class="status-dot {{if .ReleaseClean}}ok{{end}}"></span>🏷️ Release label guard
        </button>
        <button class="menu-btn" data-tab="stale" role="tab">
          <span class="status-dot {{if .StaleClean}}ok{{end}}"></span>⏳ Awaiting QA stale watchdog
        </button>
        <button class="menu-btn" data-tab="awaiting" role="tab">
          <span class="status-dot {{if .AwaitingClean}}ok{{end}}"></span>✔ Awaiting QA gate
        </button>
        <button class="menu-btn" data-tab="drafting" role="tab">
          <span class="status-dot {{if .DraftingClean}}ok{{end}}"></span>🧭 Drafting estimation gate
        </button>
        <button class="menu-btn" data-tab="missing-assignee" role="tab">
          <span class="status-dot {{if .MissingAssigneeClean}}ok{{end}}"></span>👤 Missing assignee
        </button>
        <button class="menu-btn" data-tab="assigned-to-me" role="tab">
          <span class="status-dot {{if .AssignedToMeClean}}ok{{end}}"></span>🧍 Assigned to me
        </button>
        <button class="menu-btn" data-tab="unassigned-unreleased" role="tab">
          <span class="status-dot {{if .UnassignedUnreleasedClean}}ok{{end}}"></span>🐞 Unassigned unreleased bugs
        </button>
        <button class="menu-btn" data-tab="timestamp" role="tab">
          <span class="status-dot {{if .TimestampClean}}ok{{end}}"></span>🕒 Updates timestamp expiry
        </button>
      </aside>

      <div class="panel-wrap">
      <section id="tab-awaiting" class="panel" role="tabpanel">
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
                    <ul>
                      <li>Assignees: {{if .Assignees}}{{range $i, $a := .Assignees}}{{if $i}}, {{end}}{{$a}}{{end}}{{else}}(empty){{end}}</li>
                    </ul>
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
              <li>Days remaining: {{printf "%.1f" .TimestampCheck.DaysLeft}}</li>
              <li>Hours remaining: {{printf "%.1f" .TimestampCheck.DurationLeft.Hours}}</li>
              <li>Minimum required days: {{.TimestampCheck.MinDays}}</li>
            </ul>
          </div>
        {{else}}
          <div class="project">
            <p><strong>🔴 Failing threshold</strong></p>
            <ul>
              <li>Expires: {{.TimestampCheck.ExpiresAt.Format "2006-01-02T15:04:05Z07:00"}}</li>
              <li>Days remaining: {{printf "%.1f" .TimestampCheck.DaysLeft}}</li>
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
              <h3>Project {{.ProjectNum}}</h3>
              {{range .Columns}}
                <div class="status">
                  <div class="column-head">
                    <h3>{{.Label}}</h3>
                    {{if and $.BridgeEnabled .Items}}
                      <button class="fix-btn apply-milestone-column-btn">Apply selected milestones in column</button>
                    {{end}}
                  </div>
                  {{if .Items}}
                    {{range .Items}}
                      <article class="item">
                        <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                        <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                        <ul>
                          <li>Status: {{if .Status}}{{.Status}}{{else}}(unset){{end}}</li>
                          <li>Repository: {{.Repo}}</li>
                          <li>Assignees: {{if .Assignees}}{{range $i, $a := .Assignees}}{{if $i}}, {{end}}{{$a}}{{end}}{{else}}(empty){{end}}</li>
                          <li>Labels: {{if .Labels}}{{range $i, $l := .Labels}}{{if $i}}, {{end}}{{$l}}{{end}}{{else}}(empty){{end}}</li>
                          <li>Snippet:</li>
                          {{if .BodyPreview}}
                            {{range .BodyPreview}}
                              <li>{{.}}</li>
                            {{end}}
                          {{else}}
                            <li>(empty)</li>
                          {{end}}
                        </ul>
                        <div class="actions">
                          {{if .Suggestions}}
                            <input class="fix-btn milestone-search" type="text" placeholder="Search milestone...">
                            <select class="fix-btn milestone-select" data-issue="{{.Number}}" data-repo="{{.Repo}}">
                              {{range .Suggestions}}
                                <option value="{{.Title}}" data-number="{{.Number}}">{{.Title}}</option>
                              {{end}}
                            </select>
                            {{if $.BridgeEnabled}}
                              <button class="fix-btn apply-milestone-btn">Apply milestone</button>
                            {{else}}
                              <span class="copied-note">Bridge offline: rerun qacheck to enable apply.</span>
                            {{end}}
                          {{else}}
                            <span class="copied-note">No milestone suggestions found for this repo.</span>
                          {{end}}
                        </div>
                      </article>
                    {{end}}
                  {{else}}
                    <p class="empty">🟢 No items in this group.</p>
                  {{end}}
                </div>
              {{end}}
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
                    <ul>
                      <li>Assignees: {{if .Assignees}}{{range $i, $a := .Assignees}}{{if $i}}, {{end}}{{$a}}{{end}}{{else}}(empty){{end}}</li>
                    </ul>
                    {{if .Unchecked}}
                      {{$item := .}}
                      <div>
                        {{range .Unchecked}}
                          <div class="checklist-row">
                            <span class="checklist-text">• [ ] {{.}}</span>
                            {{if and $.BridgeEnabled $item.Repo}}
                              <button class="fix-btn apply-drafting-check-btn" data-repo="{{$item.Repo}}" data-issue="{{$item.Number}}" data-check="{{.}}">Check on GitHub</button>
                            {{end}}
                          </div>
                        {{end}}
                      </div>
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

      <section id="tab-sprint" class="panel active" role="tabpanel">
        <h2>🗓️ Missing sprint (selected projects)</h2>
        <p class="subtle">Items in selected projects without a sprint set. Grouped by column focus.</p>
        {{if .MissingSprint}}
          {{range .MissingSprint}}
            <div class="project">
              <h3>Project {{.ProjectNum}}</h3>
              {{range .Columns}}
                <div class="status">
                  <div class="column-head">
                    <h3>{{.Label}}</h3>
                    {{if and $.BridgeEnabled .Items}}
                      <button class="fix-btn apply-sprint-column-btn">Set current sprint for column</button>
                    {{end}}
                  </div>
                  {{if .Items}}
                    {{range .Items}}
                      <article class="item">
                        <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                        <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                        <ul>
                          <li>Status: {{if .Status}}{{.Status}}{{else}}(unset){{end}}</li>
                          <li>Current sprint: {{if .CurrentSprint}}{{.CurrentSprint}}{{else}}(unknown){{end}}</li>
                          <li>Milestone: {{.Milestone}}</li>
                          <li>Assignees: {{if .Assignees}}{{range $i, $a := .Assignees}}{{if $i}}, {{end}}{{$a}}{{end}}{{else}}(empty){{end}}</li>
                          <li>Labels: {{if .Labels}}{{range $i, $l := .Labels}}{{if $i}}, {{end}}{{$l}}{{end}}{{else}}(empty){{end}}</li>
                          <li>Snippet:</li>
                          {{if .BodyPreview}}
                            {{range .BodyPreview}}
                              <li>{{.}}</li>
                            {{end}}
                          {{else}}
                            <li>(empty)</li>
                          {{end}}
                        </ul>
                        {{if $.BridgeEnabled}}
                          <div class="actions">
                            <button class="fix-btn apply-sprint-btn" data-item-id="{{.ItemID}}">Set current sprint</button>
                          </div>
                        {{end}}
                      </article>
                    {{end}}
                  {{else}}
                    <p class="empty">🟢 No items in this group.</p>
                  {{end}}
                </div>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No missing sprint items found.</p>
        {{end}}
      </section>

      <section id="tab-missing-assignee" class="panel" role="tabpanel">
        <h2>👤 Missing assignee (selected projects)</h2>
        <p class="subtle">Items with no assignee. If any item appears here, this check fails.</p>
        {{if .MissingAssignee}}
          {{range .MissingAssignee}}
            <div class="project">
              <h3>Project {{.ProjectNum}}</h3>
              {{range .Columns}}
                <div class="status">
                  <div class="column-head">
                    <h3>{{.Label}}</h3>
                    {{if and $.BridgeEnabled .Items}}
                      <button class="fix-btn apply-assignee-column-btn">Assign selected in column</button>
                    {{end}}
                  </div>
                  {{if .Items}}
                    {{range .Items}}
                      <article class="item {{if .AssignedToMe}}assigned-to-me{{end}}">
                        <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                        <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                        <ul>
                          <li>Status: {{if .Status}}{{.Status}}{{else}}(unset){{end}}</li>
                          <li>Repository: {{.Repo}}</li>
                          <li>Current assignees: {{if .CurrentAssignees}}{{range $i, $a := .CurrentAssignees}}{{if $i}}, {{end}}{{$a}}{{end}}{{else}}(none){{end}}</li>
                        </ul>
                        <div class="actions">
                          {{if .SuggestedAssignees}}
                            <input class="fix-btn assignee-search" type="text" placeholder="Search assignee...">
                            <select class="fix-btn assignee-select" data-issue="{{.Number}}" data-repo="{{.Repo}}">
                              {{range .SuggestedAssignees}}
                                <option value="{{.Login}}">{{.Login}}</option>
                              {{end}}
                            </select>
                            {{if $.BridgeEnabled}}
                              <button class="fix-btn apply-assignee-btn">Assign</button>
                            {{else}}
                              <span class="copied-note">Bridge offline: rerun qacheck to enable assign.</span>
                            {{end}}
                          {{else}}
                            <span class="copied-note">No assignee options found for this repo.</span>
                          {{end}}
                        </div>
                      </article>
                    {{end}}
                  {{else}}
                    <p class="empty">🟢 No items in this group.</p>
                  {{end}}
                </div>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No missing-assignee items found.</p>
        {{end}}
      </section>

      <section id="tab-assigned-to-me" class="panel" role="tabpanel">
        <h2>🧍 Assigned to me (selected projects)</h2>
        <p class="subtle">Items currently assigned to you. If any item appears here, this check fails.</p>
        {{if .AssignedToMe}}
          {{range .AssignedToMe}}
            <div class="project">
              <h3>Project {{.ProjectNum}}</h3>
              {{range .Columns}}
                <div class="status">
                  <div class="column-head">
                    <h3>{{.Label}}</h3>
                    {{if and $.BridgeEnabled .Items}}
                      <button class="fix-btn apply-assignee-column-btn">Assign selected in column</button>
                    {{end}}
                  </div>
                  {{if .Items}}
                    {{range .Items}}
                      <article class="item assigned-to-me">
                        <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                        <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                        <ul>
                          <li>Status: {{if .Status}}{{.Status}}{{else}}(unset){{end}}</li>
                          <li>Repository: {{.Repo}}</li>
                          <li>Current assignees: {{if .CurrentAssignees}}{{range $i, $a := .CurrentAssignees}}{{if $i}}, {{end}}{{$a}}{{end}}{{else}}(none){{end}}</li>
                        </ul>
                        <div class="mine-badge">Assigned to me</div>
                        <div class="actions">
                          {{if .SuggestedAssignees}}
                            <input class="fix-btn assignee-search" type="text" placeholder="Search assignee...">
                            <select class="fix-btn assignee-select" data-issue="{{.Number}}" data-repo="{{.Repo}}">
                              {{range .SuggestedAssignees}}
                                <option value="{{.Login}}">{{.Login}}</option>
                              {{end}}
                            </select>
                            {{if $.BridgeEnabled}}
                              <button class="fix-btn apply-assignee-btn">Assign</button>
                            {{else}}
                              <span class="copied-note">Bridge offline: rerun qacheck to enable assign.</span>
                            {{end}}
                          {{else}}
                            <span class="copied-note">No assignee options found for this repo.</span>
                          {{end}}
                        </div>
                      </article>
                    {{end}}
                  {{else}}
                    <p class="empty">🟢 No items in this group.</p>
                  {{end}}
                </div>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No assigned-to-me items found.</p>
        {{end}}
      </section>

      <section id="tab-release" class="panel" role="tabpanel">
        <h2>🏷️ Release label guard (selected projects)</h2>
        <p class="subtle">For selected projects (excluding project ` + fmt.Sprintf("%d", draftingProjectNum) + `): if ticket has <code>` + productLabel + `</code> or is missing <code>` + releaseLabel + `</code>, apply release labeling policy.</p>
        {{if .ReleaseLabel}}
          {{range .ReleaseLabel}}
            <div class="project">
              <div class="column-head">
                <h3>Project {{.ProjectNum}}</h3>
                {{if and $.BridgeEnabled .Items}}
                  <button class="fix-btn apply-release-project-btn">Apply release label</button>
                {{end}}
              </div>
              {{if .Items}}
                {{range .Items}}
                  <article class="item release-item" data-repo="{{.Repo}}" data-issue="{{.Number}}">
                    <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                    <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                    <ul>
                      <li>Status: {{if .Status}}{{.Status}}{{else}}(unset){{end}}</li>
                      <li>Labels: {{if .CurrentLabels}}{{range $i, $l := .CurrentLabels}}{{if $i}}, {{end}}{{$l}}{{end}}{{else}}(none){{end}}</li>
                    </ul>
                  </article>
                {{end}}
              {{else}}
                <p class="empty">🟢 No release-label issues in this project.</p>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No release-label issues found.</p>
        {{end}}
      </section>

      <section id="tab-unassigned-unreleased" class="panel" role="tabpanel">
        <h2>🐞 Unassigned unreleased bugs (selected projects)</h2>
        <p class="subtle">Grouped by provided <code>-l</code> labels and by status. Red cards are unassigned (failing). Green cards are assigned (informational).</p>
        {{if .UnassignedUnreleased}}
          {{range .UnassignedUnreleased}}
            <div class="project">
              <h3>Group: {{.GroupLabel}}</h3>
              {{range .Columns}}
                <div class="status">
                  <h3>{{.Label}}</h3>
                  {{if .RedItems}}
                    {{range .RedItems}}
                      <article class="item red-bug">
                        <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                        <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                        <ul>
                          <li>Status: {{if .Status}}{{.Status}}{{else}}(unset){{end}}</li>
                          <li>Project: {{if gt .ProjectNum 0}}{{.ProjectNum}}{{else}}(not on selected project){{end}}</li>
                          <li>Repository: {{.Repo}}</li>
                          <li>Assignees: {{if .Assignees}}{{range $i, $a := .Assignees}}{{if $i}}, {{end}}{{$a}}{{end}}{{else}}(none){{end}}</li>
                          <li>Labels: {{if .Labels}}{{range $i, $l := .Labels}}{{if $i}}, {{end}}{{$l}}{{end}}{{else}}(none){{end}}</li>
                        </ul>
                      </article>
                    {{end}}
                  {{end}}
                  {{if .GreenItems}}
                    {{range .GreenItems}}
                      <article class="item green-bug">
                        <div><strong>#{{.Number}} - {{.Title}}</strong></div>
                        <div><a href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.URL}}</a></div>
                        <ul>
                          <li>Status: {{if .Status}}{{.Status}}{{else}}(unset){{end}}</li>
                          <li>Project: {{if gt .ProjectNum 0}}{{.ProjectNum}}{{else}}(not on selected project){{end}}</li>
                          <li>Repository: {{.Repo}}</li>
                          <li>Assignees: {{if .Assignees}}{{range $i, $a := .Assignees}}{{if $i}}, {{end}}{{$a}}{{end}}{{else}}(none){{end}}</li>
                          <li>Labels: {{if .Labels}}{{range $i, $l := .Labels}}{{if $i}}, {{end}}{{$l}}{{end}}{{else}}(none){{end}}</li>
                        </ul>
                      </article>
                    {{end}}
                  {{end}}
                  {{if and (eq (len .RedItems) 0) (eq (len .GreenItems) 0)}}
                    <p class="empty">🟢 No items in this group.</p>
                  {{end}}
                </div>
              {{end}}
            </div>
          {{end}}
        {{else}}
          <p class="empty">🟢 No unassigned unreleased bugs found.</p>
        {{end}}
      </section>
      </div>
    </div>
  </div>
  <script>
    (function () {
      const bridgeSession = document.body.dataset.bridgeSession || '';
      function bridgeJSONHeaders() {
        return {
          'Content-Type': 'application/json',
          'X-Qacheck-Session': bridgeSession,
        };
      }
      const buttons = document.querySelectorAll('.menu-btn');
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

      function setButtonDone(btn, text) {
        btn.classList.remove('failed');
        btn.classList.add('done');
        btn.textContent = text;
        btn.disabled = true;
      }

      function setButtonFailed(btn) {
        btn.classList.remove('done');
        btn.classList.add('failed');
        btn.textContent = 'Failed (retry)';
        btn.disabled = false;
      }

      function setButtonWorking(btn, text) {
        btn.classList.remove('done', 'failed');
        btn.textContent = text;
        btn.disabled = true;
      }

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

      function installAssigneeFiltering() {
        const actionBlocks = document.querySelectorAll('.actions');
        actionBlocks.forEach((actions) => {
          const searchInput = actions.querySelector('.assignee-search');
          const select = actions.querySelector('.assignee-select');
          if (!searchInput || !select) return;

          const allOptions = Array.from(select.options).map((opt) => ({
            login: opt.value,
          }));

          function renderFiltered(term) {
            const q = term.trim().toLowerCase();
            const filtered = allOptions.filter((o) => o.login.toLowerCase().includes(q));
            select.innerHTML = '';
            if (filtered.length === 0) {
              const none = document.createElement('option');
              none.textContent = 'No matching assignees';
              none.value = '';
              select.appendChild(none);
              return;
            }
            filtered.forEach((o) => {
              const opt = document.createElement('option');
              opt.value = o.login;
              opt.textContent = o.login;
              select.appendChild(opt);
            });
          }

          searchInput.addEventListener('input', () => renderFiltered(searchInput.value));
        });
      }

      async function applyMilestoneButton(btn) {
        const actions = btn.closest('.actions');
        const select = actions && actions.querySelector('.milestone-select');
        if (!select) return false;
        const issue = select.dataset.issue || '';
        const repo = select.dataset.repo || '';
        const milestoneTitle = select.value || '';
        const milestoneNumber = parseInt((select.selectedOptions[0] && select.selectedOptions[0].dataset.number) || '', 10);
        if (!issue || !repo || !milestoneTitle || Number.isNaN(milestoneNumber)) return false;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run qacheck and keep terminal open.');
          return false;
        }

        const endpoint = bridgeURL + '/api/apply-milestone';
        const payload = { repo: repo, issue: issue, milestone_number: milestoneNumber };
        setButtonWorking(btn, 'Applying...');
        try {
          const res = await fetch(endpoint, {
            method: 'POST',
            headers: bridgeJSONHeaders(),
            body: JSON.stringify(payload),
          });
          if (!res.ok) {
            const body = await res.text();
            throw new Error('Bridge error ' + res.status + ': ' + body);
          }
          setButtonDone(btn, 'Done');
          return true;
        } catch (err) {
          window.alert('Could not apply milestone. ' + err);
          setButtonFailed(btn);
          return false;
        }
      }

      const applyButtons = document.querySelectorAll('.apply-milestone-btn');
      applyButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          await applyMilestoneButton(btn);
        });
      });

      const applyMilestoneColumnButtons = document.querySelectorAll('.apply-milestone-column-btn');
      applyMilestoneColumnButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          const statusCard = btn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-milestone-btn'));
          if (rowButtons.length === 0) return;

          setButtonWorking(btn, 'Applying column...');
          let ok = true;
          for (const rowBtn of rowButtons) {
            // sequential requests keep updates readable and avoid API bursts.
            const rowOK = await applyMilestoneButton(rowBtn);
            ok = ok && rowOK;
          }
          if (ok) {
            setButtonDone(btn, 'Done');
          } else {
            setButtonFailed(btn);
          }
        });
      });

      const applyDraftingCheckButtons = document.querySelectorAll('.apply-drafting-check-btn');
      applyDraftingCheckButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
          if (!bridgeURL || !bridgeSession) {
            window.alert('Bridge unavailable. Re-run qacheck and keep terminal open.');
            return;
          }

          const repo = btn.dataset.repo || '';
          const issue = btn.dataset.issue || '';
          const checkText = btn.dataset.check || '';
          if (!repo || !issue || !checkText) return;

          const endpoint = bridgeURL + '/api/apply-checklist';
          setButtonWorking(btn, 'Checking...');
          try {
            const res = await fetch(endpoint, {
              method: 'POST',
              headers: bridgeJSONHeaders(),
              body: JSON.stringify({ repo: repo, issue: issue, check_text: checkText }),
            });
            if (!res.ok) {
              const body = await res.text();
              throw new Error('Bridge error ' + res.status + ': ' + body);
            }
            const payload = await res.json();
            if (!payload.updated) {
              if (payload.already_checked) {
                setButtonDone(btn, 'Done');
              } else {
                setButtonFailed(btn);
              }
              return;
            }

            setButtonDone(btn, 'Done');
            const row = btn.closest('.checklist-row');
            const textEl = row && row.querySelector('.checklist-text');
            if (textEl) {
              textEl.textContent = '• [x] ' + checkText;
            }
          } catch (err) {
            window.alert('Could not apply checklist update. ' + err);
            setButtonFailed(btn);
          }
        });
      });

      async function applySprintButton(btn) {
        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run qacheck and keep terminal open.');
          return false;
        }
        const itemID = btn.dataset.itemId || '';
        if (!itemID) return false;

        const endpoint = bridgeURL + '/api/apply-sprint';
        setButtonWorking(btn, 'Setting...');
        try {
          const res = await fetch(endpoint, {
            method: 'POST',
            headers: bridgeJSONHeaders(),
            body: JSON.stringify({ item_id: itemID }),
          });
          if (!res.ok) {
            const body = await res.text();
            throw new Error('Bridge error ' + res.status + ': ' + body);
          }
          setButtonDone(btn, 'Done');
          return true;
        } catch (err) {
          window.alert('Could not set sprint. ' + err);
          setButtonFailed(btn);
          return false;
        }
      }

      const applySprintButtons = document.querySelectorAll('.apply-sprint-btn');
      applySprintButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          await applySprintButton(btn);
        });
      });

      const applySprintColumnButtons = document.querySelectorAll('.apply-sprint-column-btn');
      applySprintColumnButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          const statusCard = btn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-sprint-btn'));
          if (rowButtons.length === 0) return;

          setButtonWorking(btn, 'Setting column...');
          let ok = true;
          for (const rowBtn of rowButtons) {
            const rowOK = await applySprintButton(rowBtn);
            ok = ok && rowOK;
          }
          if (ok) {
            setButtonDone(btn, 'Done');
          } else {
            setButtonFailed(btn);
          }
        });
      });

      async function applyAssigneeButton(btn) {
        const actions = btn.closest('.actions');
        const select = actions && actions.querySelector('.assignee-select');
        if (!select) return false;
        const issue = select.dataset.issue || '';
        const repo = select.dataset.repo || '';
        const assignee = select.value || '';
        if (!issue || !repo || !assignee) return false;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run qacheck and keep terminal open.');
          return false;
        }
        const endpoint = bridgeURL + '/api/add-assignee';
        const payload = { repo: repo, issue: issue, assignee: assignee };
        setButtonWorking(btn, 'Assigning...');
        try {
          const res = await fetch(endpoint, {
            method: 'POST',
            headers: bridgeJSONHeaders(),
            body: JSON.stringify(payload),
          });
          if (!res.ok) {
            const body = await res.text();
            throw new Error('Bridge error ' + res.status + ': ' + body);
          }
          setButtonDone(btn, 'Done');
          return true;
        } catch (err) {
          window.alert('Could not assign user. ' + err);
          setButtonFailed(btn);
          return false;
        }
      }

      const applyAssigneeButtons = document.querySelectorAll('.apply-assignee-btn');
      applyAssigneeButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          await applyAssigneeButton(btn);
        });
      });

      const applyAssigneeColumnButtons = document.querySelectorAll('.apply-assignee-column-btn');
      applyAssigneeColumnButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          const statusCard = btn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-assignee-btn'));
          if (rowButtons.length === 0) return;

          setButtonWorking(btn, 'Assigning column...');
          let ok = true;
          for (const rowBtn of rowButtons) {
            const rowOK = await applyAssigneeButton(rowBtn);
            ok = ok && rowOK;
          }
          if (ok) {
            setButtonDone(btn, 'Done');
          } else {
            setButtonFailed(btn);
          }
        });
      });

      async function applyReleaseItem(itemEl) {
        const repo = itemEl.dataset.repo || '';
        const issue = itemEl.dataset.issue || '';
        if (!repo || !issue) return false;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run qacheck and keep terminal open.');
          return false;
        }
        const endpoint = bridgeURL + '/api/apply-release-label';
        const res = await fetch(endpoint, {
          method: 'POST',
          headers: bridgeJSONHeaders(),
          body: JSON.stringify({ repo: repo, issue: issue }),
        });
        if (!res.ok) {
          const body = await res.text();
          throw new Error('Bridge error ' + res.status + ': ' + body);
        }
        return true;
      }

      const releaseProjectButtons = document.querySelectorAll('.apply-release-project-btn');
      releaseProjectButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          const project = btn.closest('.project');
          if (!project) return;
          const items = Array.from(project.querySelectorAll('.release-item'));
          if (items.length === 0) return;

          setButtonWorking(btn, 'Applying...');
          try {
            let ok = true;
            for (const item of items) {
              const itemOK = await applyReleaseItem(item);
              ok = ok && itemOK;
            }
            if (ok) {
              setButtonDone(btn, 'Done');
            } else {
              setButtonFailed(btn);
            }
          } catch (err) {
            window.alert('Could not apply release label. ' + err);
            setButtonFailed(btn);
          }
        });
      });

      const closeSessionButton = document.getElementById('close-session-btn');
      if (closeSessionButton) {
        closeSessionButton.addEventListener('click', async () => {
          const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
          if (!bridgeURL || !bridgeSession) {
            window.alert('Bridge unavailable.');
            return;
          }
          setButtonWorking(closeSessionButton, 'Closing...');
          try {
            const res = await fetch(bridgeURL + '/api/close', {
              method: 'POST',
              headers: bridgeJSONHeaders(),
              body: JSON.stringify({ reason: 'closed from UI' }),
            });
            if (!res.ok) {
              const body = await res.text();
              throw new Error('Bridge error ' + res.status + ': ' + body);
            }
            document.querySelectorAll('.apply-milestone-btn, .apply-milestone-column-btn, .apply-drafting-check-btn, .apply-sprint-btn, .apply-sprint-column-btn, .apply-assignee-btn, .apply-assignee-column-btn, .apply-release-project-btn').forEach((el) => {
              el.disabled = true;
            });
            setButtonDone(closeSessionButton, 'Done');
          } catch (err) {
            window.alert('Could not close session. ' + err);
            setButtonFailed(closeSessionButton);
          }
        });
      }

      installMilestoneFiltering();
      installAssigneeFiltering();
    })();
  </script>
</body>
</html>
`
