package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
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
	Labels    []string
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

type ReleaseStoryTODOProjectReport struct {
	ProjectNum int
	Columns    []MissingMilestoneGroupReport
}

type GenericQueryReportItem struct {
	Number    int
	Title     string
	URL       string
	Repo      string
	Status    string
	Assignees []string
	Labels    []string
}

type GenericQueryReport struct {
	Title string
	Query string
	Items []GenericQueryReportItem
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
	ReleaseStoryTODO          []ReleaseStoryTODOProjectReport
	GenericQueries            []GenericQueryReport
	UnassignedUnreleased      []UnassignedUnreleasedProjectReport
	TotalAwaiting             int
	TotalStale                int
	TotalDrafting             int
	TotalNoMilestone          int
	TotalNoSprint             int
	TotalMissingAssignee      int
	TotalAssignedToMe         int
	TotalRelease              int
	TotalReleaseStoryTODO     int
	TotalGenericQueries       int
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
	ReleaseStoryTODOClean     bool
	GenericQueriesClean       bool
	UnassignedUnreleasedClean bool
}

// buildHTMLReportData transforms raw check outputs into the fully grouped
// view-model consumed by the HTML template.
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
	releaseStoryTODO []ReleaseStoryTODOIssue,
	genericQueries []GenericQueryResult,
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
				Labels:    issueLabels(v.Item),
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

	groupedReleaseStoryTODOByProject := make(map[int]map[string][]MissingMilestoneReportItem)
	for _, p := range projectNums {
		groupedReleaseStoryTODOByProject[p] = make(map[string][]MissingMilestoneReportItem)
		for _, key := range sprintColumnOrder() {
			groupedReleaseStoryTODOByProject[p][key] = []MissingMilestoneReportItem{}
		}
	}
	for _, v := range releaseStoryTODO {
		repo := v.RepoOwner + "/" + v.RepoName
		group := sprintColumnGroup(v.Status)
		groupedReleaseStoryTODOByProject[v.ProjectNum][group] = append(groupedReleaseStoryTODOByProject[v.ProjectNum][group], MissingMilestoneReportItem{
			ProjectNum:  v.ProjectNum,
			Number:      getNumber(v.Item),
			Title:       getTitle(v.Item),
			URL:         getURL(v.Item),
			Repo:        repo,
			Status:      v.Status,
			Assignees:   issueAssignees(v.Item),
			Labels:      append([]string(nil), v.CurrentLabels...),
			BodyPreview: append([]string(nil), v.BodyPreview...),
		})
	}
	releaseStoryTODOProjects := make([]ReleaseStoryTODOProjectReport, 0, len(projectNums))
	totalReleaseStoryTODO := 0
	for _, p := range projectNums {
		grouped := groupedReleaseStoryTODOByProject[p]
		columns := make([]MissingMilestoneGroupReport, 0, len(sprintColumnOrder()))
		for _, key := range sprintColumnOrder() {
			items := grouped[key]
			totalReleaseStoryTODO += len(items)
			columns = append(columns, MissingMilestoneGroupReport{
				Key:   key,
				Label: sprintColumnLabel(key),
				Items: items,
			})
		}
		releaseStoryTODOProjects = append(releaseStoryTODOProjects, ReleaseStoryTODOProjectReport{
			ProjectNum: p,
			Columns:    columns,
		})
	}

	genericQueryReports := make([]GenericQueryReport, 0, len(genericQueries))
	totalGenericQueries := 0
	for _, query := range genericQueries {
		items := make([]GenericQueryReportItem, 0, len(query.Items))
		for _, item := range query.Items {
			items = append(items, GenericQueryReportItem{
				Number:    item.Number,
				Title:     item.Title,
				URL:       item.URL,
				Repo:      item.RepoOwner + "/" + item.RepoName,
				Status:    item.Status,
				Assignees: append([]string(nil), item.CurrentAssignees...),
				Labels:    append([]string(nil), item.CurrentLabels...),
			})
		}
		totalGenericQueries += len(items)
		genericQueryReports = append(genericQueryReports, GenericQueryReport{
			Title: query.Title,
			Query: query.Query,
			Items: items,
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
		ReleaseStoryTODO:          releaseStoryTODOProjects,
		GenericQueries:            genericQueryReports,
		UnassignedUnreleased:      unassignedProjects,
		TotalAwaiting:             totalAwaiting,
		TotalStale:                totalStale,
		TotalDrafting:             totalDrafting,
		TotalNoMilestone:          totalNoMilestone,
		TotalNoSprint:             totalNoSprint,
		TotalMissingAssignee:      totalMissingAssignee,
		TotalAssignedToMe:         totalAssignedToMe,
		TotalRelease:              totalRelease,
		TotalReleaseStoryTODO:     totalReleaseStoryTODO,
		TotalGenericQueries:       totalGenericQueries,
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
		ReleaseStoryTODOClean:     totalReleaseStoryTODO == 0,
		GenericQueriesClean:       totalGenericQueries == 0,
		UnassignedUnreleasedClean: totalUnassignedUnreleased == 0,
	}
}

// itemStatus extracts the project "Status" single-select value from an item.
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

// previewBodyLines returns the first non-empty trimmed body lines for previews.
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

// renderHTMLReport parses and renders the embedded app shell template.
func renderHTMLReport(w io.Writer, data HTMLReportData) error {
	templateText := htmlReportTemplate
	if dir := uiRuntimeDirValue(); dir != "" {
		raw, err := os.ReadFile(filepath.Join(dir, "index.html"))
		if err != nil {
			return fmt.Errorf("read ui dev template: %w", err)
		}
		templateText = string(raw)
	}

	tmpl, err := template.New("report").Parse(templateText)
	if err != nil {
		return fmt.Errorf("parse report template: %w", err)
	}
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("render report template: %w", err)
	}
	return nil
}

// fileURLFromPath converts a local file path into a file:// URL.
func fileURLFromPath(path string) string {
	u := url.URL{
		Scheme: "file",
		Path:   path,
	}
	return u.String()
}

// openInBrowser opens the report path using the platform-specific launcher.
func openInBrowser(path string) error {
	// Switch picks the OS-specific launcher command:
	// - darwin: `open`
	// - linux: `xdg-open`
	// - windows: `rundll32 ... FileProtocolHandler`
	// - default: return explicit unsupported-OS error (caller must open manually).
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

// titleCaseWords normalizes an arbitrary phrase into basic title case.
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

// htmlReportTemplate holds the frontend app shell loaded from ui/index.html.
//
//go:embed ui/index.html
var htmlReportTemplate string
