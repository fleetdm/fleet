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

// writeHTMLReport recreates the report directory and renders the HTML file.
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
      <div id="counts-content" class="counts">
        <span class="pill">Loading check totals…</span>
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
        <button class="menu-btn active" data-tab="release-story-todo" role="tab">
          <span class="status-dot {{if .ReleaseStoryTODOClean}}ok{{end}}"></span>📝 Release stories TODO
        </button>
        <button class="menu-btn" data-tab="generic-queries" role="tab">
          <span class="status-dot {{if .GenericQueriesClean}}ok{{end}}"></span>🔎 Generic queries
        </button>
        <button class="menu-btn" data-tab="sprint" role="tab">
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
      <section id="tab-release-story-todo" class="panel active" role="tabpanel">
        <div class="column-head">
          <h2>📝 Release stories TODO (selected projects)</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="release-story-todo">Refresh</button>
        </div>
        <p class="subtle">Stories with <code>:release</code> label that still contain <code>TODO</code> text in the body.</p>
        <div id="release-story-todo-content">
          <p class="empty">🛰️ Loading release stories TODO from bridge…</p>
        </div>
      </section>

      <section id="tab-generic-queries" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>🔎 Generic queries</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="generic-queries">Refresh</button>
        </div>
        <p class="subtle">Runs configured issue queries in order. Placeholders: <code>&lt;&lt;group&gt;&gt;</code> and <code>&lt;&lt;project&gt;&gt;</code>.</p>
        <div id="generic-queries-content">
          <p class="empty">🛰️ Loading generic query results from bridge…</p>
        </div>
      </section>

      <section id="tab-awaiting" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>✅ Awaiting QA gate</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="awaiting">Refresh</button>
        </div>
        <p class="subtle">Items in <strong>` + awaitingQAColumn + `</strong> where engineer test-plan confirmation is unchecked.</p>
        <div id="awaiting-content">
          <p class="empty">🛰️ Loading Awaiting QA data from bridge…</p>
        </div>
      </section>
      <section id="tab-stale" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>⏳ Awaiting QA stale watchdog</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="stale">Refresh</button>
        </div>
        <p class="subtle">Items in <strong>` + awaitingQAColumn + `</strong> with no updates for at least {{.StaleThreshold}} days.</p>
        <div id="stale-content">
          <p class="empty">🛰️ Loading stale Awaiting QA data from bridge…</p>
        </div>
      </section>

      <section id="tab-timestamp" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>🕒 Updates timestamp.json expiry</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="timestamp">Refresh</button>
        </div>
        <p id="timestamp-subtle" class="subtle">Loading timestamp check from bridge…</p>
        <div id="timestamp-content" class="project">
          <p class="empty">🛰️ Fetching latest timestamp status…</p>
        </div>
      </section>

      <section id="tab-milestone" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>🎯 Missing milestones (selected projects)</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="milestone">Refresh</button>
        </div>
        <p class="subtle">Issues in selected projects without a milestone. Type to filter milestones, choose one, then apply directly.</p>
        <div id="milestone-content">
          <p class="empty">🛰️ Loading missing milestones from bridge…</p>
        </div>
      </section>

      <section id="tab-drafting" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>🧭 Drafting estimation gate (project ` + fmt.Sprintf("%d", draftingProjectNum) + `)</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="drafting">Refresh</button>
        </div>
        <p class="subtle">Items in estimation statuses with unchecked checklist items.</p>
        <div id="drafting-content">
          <p class="empty">🛰️ Loading drafting data from bridge…</p>
        </div>
      </section>

      <section id="tab-sprint" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>🗓️ Missing sprint (selected projects)</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="missing-sprint">Refresh</button>
        </div>
        <p class="subtle">Items in selected projects without a sprint set. Grouped by column focus.</p>
        <div id="missing-sprint-content">
          <p class="empty">🛰️ Loading missing sprint data from bridge…</p>
        </div>
      </section>

      <section id="tab-missing-assignee" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>👤 Missing assignee (selected projects)</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="missing-assignee">Refresh</button>
        </div>
        <p class="subtle">Items with no assignee. If any item appears here, this check fails.</p>
        <div id="missing-assignee-content">
          <p class="empty">🛰️ Loading missing assignee data from bridge…</p>
        </div>
      </section>

      <section id="tab-assigned-to-me" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>🧍 Assigned to me (selected projects)</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="assigned-to-me">Refresh</button>
        </div>
        <p class="subtle">Items currently assigned to you. If any item appears here, this check fails.</p>
        <div id="assigned-to-me-content">
          <p class="empty">🛰️ Loading assigned-to-me data from bridge…</p>
        </div>
      </section>

      <section id="tab-release" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>🏷️ Release label guard (selected projects)</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="release">Refresh</button>
        </div>
        <p class="subtle">For selected projects (excluding project ` + fmt.Sprintf("%d", draftingProjectNum) + `): if ticket has <code>` + productLabel + `</code> or is missing <code>` + releaseLabel + `</code>, apply release labeling policy.</p>
        <div id="release-content">
          <p class="empty">🛰️ Loading release label data from bridge…</p>
        </div>
      </section>

      <section id="tab-unassigned-unreleased" class="panel" role="tabpanel">
        <div class="column-head">
          <h2>🐞 Unassigned unreleased bugs (selected projects)</h2>
          <button class="fix-btn refresh-check-btn" data-refresh-check="unassigned-unreleased">Refresh</button>
        </div>
        <p class="subtle">Grouped by provided <code>-l</code> labels and by status. Red cards are unassigned (failing). Green cards are assigned (informational).</p>
        <div id="unreleased-content">
          <p class="empty">🛰️ Loading unreleased bug groups from bridge…</p>
        </div>
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

      function setTabClean(tabName, isClean) {
        const btn = document.querySelector('.menu-btn[data-tab="' + tabName + '"]');
        if (!btn) return;
        const dot = btn.querySelector('.status-dot');
        if (!dot) return;
        dot.classList.toggle('ok', Boolean(isClean));
      }

      async function refreshTimestampPanel(forceRefresh) {
        const subtle = document.getElementById('timestamp-subtle');
        const content = document.getElementById('timestamp-content');
        if (!subtle || !content) return;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          subtle.innerHTML = 'Bridge unavailable.';
          content.innerHTML = '<p class="empty">🔴 Could not load timestamp check (missing bridge session).</p>';
          setTabClean('timestamp', false);
          return;
        }

        try {
          const query = forceRefresh ? '?refresh=1' : '';
          const res = await fetch(bridgeURL + '/api/check/timestamp' + query, {
            method: 'GET',
            headers: { 'X-Qacheck-Session': bridgeSession },
          });
          if (!res.ok) {
            throw new Error('Bridge error ' + res.status);
          }
          const payload = await res.json();
          const safeURL = payload.url || '';
          const minDays = Number(payload.min_days || 0);
          subtle.innerHTML = 'Checks that <a href="' + safeURL + '" target="_blank" rel="noopener noreferrer">' + safeURL + '</a> expires at least ' + minDays + ' days from now.';

          if (payload.error) {
            content.innerHTML = '<p class="empty">🔴 Could not validate timestamp expiry: ' + payload.error + '</p>';
            setTabClean('timestamp', false);
            return;
          }

          const stateText = payload.ok ? '🟢 OK' : '🔴 Failing threshold';
          const expires = payload.expires_at || '(unknown)';
          const daysLeft = Number(payload.days_left || 0).toFixed(1);
          const hoursLeft = Number(payload.duration_hours || 0).toFixed(1);

          content.innerHTML =
            '<p><strong>' + stateText + '</strong></p>' +
            '<ul>' +
            '<li>Expires: ' + expires + '</li>' +
            '<li>Days remaining: ' + daysLeft + '</li>' +
            '<li>Hours remaining: ' + hoursLeft + '</li>' +
            '<li>Minimum required days: ' + minDays + '</li>' +
            '</ul>';
          setTabClean('timestamp', Boolean(payload.ok));
        } catch (err) {
          subtle.textContent = 'Checks timestamp expiry from the bridge.';
          content.innerHTML = '<p class="empty">🔴 Could not load timestamp check: ' + err + '</p>';
          setTabClean('timestamp', false);
        }
      }

      function escHTML(value) {
        const text = String(value == null ? '' : value);
        return text
          .replaceAll('&', '&amp;')
          .replaceAll('<', '&lt;')
          .replaceAll('>', '&gt;')
          .replaceAll('"', '&quot;')
          .replaceAll("'", '&#39;');
      }

      function renderUnreleasedItem(item, cssClass) {
        const status = item.Status ? item.Status : '(unset)';
        const projectText = Number(item.ProjectNum || 0) > 0 ? String(item.ProjectNum) : '(not on selected project)';
        const assignees = Array.isArray(item.Assignees) && item.Assignees.length > 0 ? item.Assignees.join(', ') : '(none)';
        const labels = Array.isArray(item.Labels) && item.Labels.length > 0 ? item.Labels.join(', ') : '(none)';

        return '' +
          '<article class="item ' + cssClass + '">' +
            '<div><strong>#' + escHTML(item.Number) + ' - ' + escHTML(item.Title) + '</strong></div>' +
            '<div><a href="' + escHTML(item.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(item.URL) + '</a></div>' +
            '<ul>' +
              '<li>Status: ' + escHTML(status) + '</li>' +
              '<li>Project: ' + escHTML(projectText) + '</li>' +
              '<li>Repository: ' + escHTML(item.Repo) + '</li>' +
              '<li>Assignees: ' + escHTML(assignees) + '</li>' +
              '<li>Labels: ' + escHTML(labels) + '</li>' +
            '</ul>' +
          '</article>';
      }

      function renderReleaseStoryTODOItem(item) {
        const status = item.Status ? item.Status : '(unset)';
        const assignees = Array.isArray(item.Assignees) && item.Assignees.length > 0 ? item.Assignees.join(', ') : '(empty)';
        const labels = Array.isArray(item.Labels) && item.Labels.length > 0 ? item.Labels.join(', ') : '(none)';
        const preview = Array.isArray(item.BodyPreview) && item.BodyPreview.length > 0 ? item.BodyPreview : ['(empty)'];
        const previewLines = preview.map((line) => '<li>' + escHTML(line) + '</li>').join('');

        return '' +
          '<article class="item red-bug">' +
            '<div><strong>#' + escHTML(item.Number) + ' - ' + escHTML(item.Title) + '</strong></div>' +
            '<div><a href="' + escHTML(item.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(item.URL) + '</a></div>' +
            '<ul>' +
              '<li>Status: ' + escHTML(status) + '</li>' +
              '<li>Repository: ' + escHTML(item.Repo) + '</li>' +
              '<li>Assignees: ' + escHTML(assignees) + '</li>' +
              '<li>Labels: ' + escHTML(labels) + '</li>' +
              '<li>Snippet:</li>' +
              previewLines +
            '</ul>' +
          '</article>';
      }

      async function refreshReleaseStoryTODOPanel(forceRefresh) {
        const root = document.getElementById('release-story-todo-content');
        if (!root) return;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          root.innerHTML = '<p class="empty">🔴 Could not load release stories TODO (missing bridge session).</p>';
          setTabClean('release-story-todo', false);
          return;
        }

        try {
          const query = forceRefresh ? '?refresh=1' : '';
          const res = await fetch(bridgeURL + '/api/check/release-story-todo' + query, {
            method: 'GET',
            headers: { 'X-Qacheck-Session': bridgeSession },
          });
          if (!res.ok) {
            throw new Error('Bridge error ' + res.status);
          }
          const payload = await res.json();
          const projects = Array.isArray(payload.projects) ? payload.projects : [];
          let totalItems = 0;
          projects.forEach((proj) => {
            const columns = Array.isArray(proj.Columns) ? proj.Columns : [];
            columns.forEach((col) => {
              const items = Array.isArray(col.Items) ? col.Items : [];
              totalItems += items.length;
            });
          });
          setTabClean('release-story-todo', totalItems === 0);
          if (projects.length === 0) {
            root.innerHTML = '<p class="empty">🟢 No release stories with TODO found.</p>';
            return;
          }

          root.innerHTML = projects.map((proj) => {
            const columns = Array.isArray(proj.Columns) ? proj.Columns : [];
            const renderedCols = columns.map((col) => {
              const items = Array.isArray(col.Items) ? col.Items : [];
              let html = '<div class="status"><h3>' + escHTML(col.Label) + '</h3>';
              if (items.length === 0) {
                html += '<p class="empty">🟢 No items in this group.</p>';
              } else {
                items.forEach((it) => {
                  html += renderReleaseStoryTODOItem(it);
                });
              }
              html += '</div>';
              return html;
            }).join('');

            return '' +
              '<div class="project">' +
                '<h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' +
                renderedCols +
              '</div>';
          }).join('');
        } catch (err) {
          root.innerHTML = '<p class="empty">🔴 Could not load release stories TODO: ' + escHTML(err) + '</p>';
          setTabClean('release-story-todo', false);
        }
      }

      function renderMissingSprintItem(item, showActions) {
        const status = item.Status ? item.Status : '(unset)';
        const currentSprint = item.CurrentSprint ? item.CurrentSprint : '(unknown)';
        const milestone = item.Milestone ? item.Milestone : '(empty)';
        const assignees = Array.isArray(item.Assignees) && item.Assignees.length > 0 ? item.Assignees.join(', ') : '(empty)';
        const labels = Array.isArray(item.Labels) && item.Labels.length > 0 ? item.Labels.join(', ') : '(empty)';
        const preview = Array.isArray(item.BodyPreview) && item.BodyPreview.length > 0 ? item.BodyPreview : ['(empty)'];
        const previewLines = preview.map((line) => '<li>' + escHTML(line) + '</li>').join('');
        const actionHTML = showActions ? (
          '<div class="actions">' +
            '<button class="fix-btn apply-sprint-btn" data-item-id="' + escHTML(item.ItemID) + '">Set current sprint</button>' +
          '</div>'
        ) : '';

        return '' +
          '<article class="item">' +
            '<div><strong>#' + escHTML(item.Number) + ' - ' + escHTML(item.Title) + '</strong></div>' +
            '<div><a href="' + escHTML(item.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(item.URL) + '</a></div>' +
            '<ul>' +
              '<li>Status: ' + escHTML(status) + '</li>' +
              '<li>Current sprint: ' + escHTML(currentSprint) + '</li>' +
              '<li>Milestone: ' + escHTML(milestone) + '</li>' +
              '<li>Assignees: ' + escHTML(assignees) + '</li>' +
              '<li>Labels: ' + escHTML(labels) + '</li>' +
              '<li>Snippet:</li>' +
              previewLines +
            '</ul>' +
            actionHTML +
          '</article>';
      }

      async function refreshMissingSprintPanel(forceRefresh) {
        const root = document.getElementById('missing-sprint-content');
        if (!root) return;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          root.innerHTML = '<p class="empty">🔴 Could not load missing sprint data (missing bridge session).</p>';
          setTabClean('sprint', false);
          return;
        }

        try {
          const query = forceRefresh ? '?refresh=1' : '';
          const res = await fetch(bridgeURL + '/api/check/missing-sprint' + query, {
            method: 'GET',
            headers: { 'X-Qacheck-Session': bridgeSession },
          });
          if (!res.ok) {
            throw new Error('Bridge error ' + res.status);
          }
          const payload = await res.json();
          const projects = Array.isArray(payload.projects) ? payload.projects : [];
          let totalItems = 0;
          projects.forEach((proj) => {
            const columns = Array.isArray(proj.Columns) ? proj.Columns : [];
            columns.forEach((col) => {
              const items = Array.isArray(col.Items) ? col.Items : [];
              totalItems += items.length;
            });
          });
          setTabClean('sprint', totalItems === 0);
          if (projects.length === 0) {
            root.innerHTML = '<p class="empty">🟢 No missing sprint items found.</p>';
            return;
          }

          const showActions = Boolean(bridgeSession);
          root.innerHTML = projects.map((proj) => {
            const columns = Array.isArray(proj.Columns) ? proj.Columns : [];
            const renderedCols = columns.map((col) => {
              const items = Array.isArray(col.Items) ? col.Items : [];
              const colAction = showActions && items.length > 0
                ? '<button class="fix-btn apply-sprint-column-btn">Set current sprint for column</button>'
                : '';
              let html = '<div class="status"><div class="column-head"><h3>' + escHTML(col.Label) + '</h3>' + colAction + '</div>';
              if (items.length === 0) {
                html += '<p class="empty">🟢 No items in this group.</p>';
              } else {
                items.forEach((it) => {
                  html += renderMissingSprintItem(it, showActions);
                });
              }
              html += '</div>';
              return html;
            }).join('');

            return '' +
              '<div class="project">' +
                '<h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' +
                renderedCols +
              '</div>';
          }).join('');
        } catch (err) {
          root.innerHTML = '<p class="empty">🔴 Could not load missing sprint data: ' + escHTML(err) + '</p>';
          setTabClean('sprint', false);
        }
      }

      async function refreshUnreleasedPanel(forceRefresh) {
        const root = document.getElementById('unreleased-content');
        if (!root) return;

        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          root.innerHTML = '<p class="empty">🔴 Could not load unreleased bugs (missing bridge session).</p>';
          setTabClean('unassigned-unreleased', false);
          return;
        }

        try {
          const query = forceRefresh ? '?refresh=1' : '';
          const res = await fetch(bridgeURL + '/api/check/unassigned-unreleased' + query, {
            method: 'GET',
            headers: { 'X-Qacheck-Session': bridgeSession },
          });
          if (!res.ok) {
            throw new Error('Bridge error ' + res.status);
          }
          const payload = await res.json();
          const groups = Array.isArray(payload.groups) ? payload.groups : [];
          let totalRed = 0;
          groups.forEach((group) => {
            const columns = Array.isArray(group.Columns) ? group.Columns : [];
            columns.forEach((col) => {
              const redItems = Array.isArray(col.RedItems) ? col.RedItems : [];
              totalRed += redItems.length;
            });
          });
          setTabClean('unassigned-unreleased', totalRed === 0);
          if (groups.length === 0) {
            root.innerHTML = '<p class="empty">🟢 No unassigned unreleased bugs found.</p>';
            return;
          }

          root.innerHTML = groups.map((group) => {
            const columns = Array.isArray(group.Columns) ? group.Columns : [];
            const renderedCols = columns.map((col) => {
              const redItems = Array.isArray(col.RedItems) ? col.RedItems : [];
              const greenItems = Array.isArray(col.GreenItems) ? col.GreenItems : [];
              let html = '<div class="status"><h3>' + escHTML(col.Label) + '</h3>';
              redItems.forEach((it) => {
                html += renderUnreleasedItem(it, 'red-bug');
              });
              greenItems.forEach((it) => {
                html += renderUnreleasedItem(it, 'green-bug');
              });
              if (redItems.length === 0 && greenItems.length === 0) {
                html += '<p class="empty">🟢 No items in this group.</p>';
              }
              html += '</div>';
              return html;
            }).join('');

            return '' +
              '<div class="project">' +
                '<h3>Group: ' + escHTML(group.GroupLabel) + '</h3>' +
                renderedCols +
              '</div>';
          }).join('');
        } catch (err) {
          root.innerHTML = '<p class="empty">🔴 Could not load unreleased bugs: ' + escHTML(err) + '</p>';
          setTabClean('unassigned-unreleased', false);
        }
      }

      function listOrEmpty(values, emptyText) {
        return Array.isArray(values) && values.length > 0 ? values.join(', ') : emptyText;
      }

      function renderCounts(state) {
        const root = document.getElementById('counts-content');
        if (!root || !state) return;
        root.innerHTML = '' +
          '<span class="pill">Release stories with TODO (selected projects): ' + escHTML(state.TotalReleaseStoryTODO) + '</span>' +
          '<span class="pill">Generic query issues: ' + escHTML(state.TotalGenericQueries) + '</span>' +
          '<span class="pill">Awaiting QA violations: ' + escHTML(state.TotalAwaiting) + '</span>' +
          '<span class="pill">Stale Awaiting QA items: ' + escHTML(state.TotalStale) + '</span>' +
          '<span class="pill">Missing milestones (selected projects): ' + escHTML(state.TotalNoMilestone) + '</span>' +
          '<span class="pill">Missing sprint (selected projects): ' + escHTML(state.TotalNoSprint) + '</span>' +
          '<span class="pill">Missing assignee (selected projects): ' + escHTML(state.TotalMissingAssignee) + '</span>' +
          '<span class="pill">Assigned to me (selected projects): ' + escHTML(state.TotalAssignedToMe) + '</span>' +
          '<span class="pill">Unassigned unreleased bugs (selected projects): ' + escHTML(state.TotalUnassignedUnreleased) + '</span>' +
          '<span class="pill">Tracked unreleased bugs (assigned): ' + escHTML(state.TotalTrackedUnreleased) + '</span>' +
          '<span class="pill">Release label issues (selected projects): ' + escHTML(state.TotalRelease) + '</span>' +
          '<span class="pill">Drafting checklist violations: ' + escHTML(state.TotalDrafting) + '</span>';
      }

      function renderAwaitingFromState(state) {
        const root = document.getElementById('awaiting-content');
        if (!root) return;
        const sections = Array.isArray(state.AwaitingSections) ? state.AwaitingSections : [];
        if (sections.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No project data found.</p>';
          setTabClean('awaiting', true);
          return;
        }
        let total = 0;
        root.innerHTML = sections.map((sec) => {
          const items = Array.isArray(sec.Items) ? sec.Items : [];
          total += items.length;
          if (items.length === 0) {
            return '<div class="project"><h3>Project ' + escHTML(sec.ProjectNum) + '</h3><p class="empty">🟢 No violations in this project.</p></div>';
          }
          return '<div class="project"><h3>Project ' + escHTML(sec.ProjectNum) + '</h3>' + items.map((it) => {
            const unchecked = Array.isArray(it.Unchecked) ? it.Unchecked : [];
            const uncheckedHTML = unchecked.length > 0 ? '<ul>' + unchecked.map((u) => '<li>[ ] ' + escHTML(u) + '</li>').join('') + '</ul>' : '';
            return '' +
              '<article class="item">' +
                '<div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div>' +
                '<div><a href="' + escHTML(it.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(it.URL) + '</a></div>' +
                '<ul><li>Assignees: ' + escHTML(listOrEmpty(it.Assignees, '(empty)')) + '</li></ul>' +
                uncheckedHTML +
              '</article>';
          }).join('') + '</div>';
        }).join('');
        setTabClean('awaiting', total === 0);
      }

      function renderStaleFromState(state) {
        const root = document.getElementById('stale-content');
        if (!root) return;
        const sections = Array.isArray(state.StaleSections) ? state.StaleSections : [];
        if (sections.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No project data found.</p>';
          setTabClean('stale', true);
          return;
        }
        let total = 0;
        root.innerHTML = sections.map((sec) => {
          const items = Array.isArray(sec.Items) ? sec.Items : [];
          total += items.length;
          if (items.length === 0) {
            return '<div class="project"><h3>Project ' + escHTML(sec.ProjectNum) + '</h3><p class="empty">🟢 No stale items in this project.</p></div>';
          }
          return '<div class="project"><h3>Project ' + escHTML(sec.ProjectNum) + '</h3>' + items.map((it) => (
            '<article class="item">' +
              '<div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div>' +
              '<div><a href="' + escHTML(it.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(it.URL) + '</a></div>' +
              '<ul><li>Last updated: ' + escHTML(it.LastUpdated) + '</li><li>Age: ' + escHTML(it.StaleDays) + ' days</li></ul>' +
            '</article>'
          )).join('') + '</div>';
        }).join('');
        setTabClean('stale', total === 0);
      }

      function renderMilestoneFromState(state) {
        const root = document.getElementById('milestone-content');
        if (!root) return;
        const projects = Array.isArray(state.MissingMilestone) ? state.MissingMilestone : [];
        if (projects.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No missing milestones found.</p>';
          setTabClean('milestone', true);
          return;
        }
        let total = 0;
        root.innerHTML = projects.map((proj) => {
          const cols = Array.isArray(proj.Columns) ? proj.Columns : [];
          const colsHTML = cols.map((col) => {
            const items = Array.isArray(col.Items) ? col.Items : [];
            total += items.length;
            const colButton = items.length > 0 ? '<button class="fix-btn apply-milestone-column-btn">Apply selected milestones in column</button>' : '';
            const itemsHTML = items.length === 0 ? '<p class="empty">🟢 No items in this group.</p>' : items.map((it) => {
              const suggestions = Array.isArray(it.Suggestions) ? it.Suggestions : [];
              const options = suggestions.map((s) => '<option value="' + escHTML(s.Title) + '" data-number="' + escHTML(s.Number) + '">' + escHTML(s.Title) + '</option>').join('');
              const actionHTML = suggestions.length > 0
                ? '<div class="actions"><select class="fix-btn milestone-select" data-issue="' + escHTML(it.Number) + '" data-repo="' + escHTML(it.Repo) + '">' + options + '</select><button class="fix-btn apply-milestone-btn">Apply milestone</button></div>'
                : '<div class="actions"><span class="copied-note">No milestone suggestions found for this repo.</span></div>';
              const preview = Array.isArray(it.BodyPreview) && it.BodyPreview.length > 0 ? it.BodyPreview : ['(empty)'];
              const previewHTML = preview.map((p) => '<li>' + escHTML(p) + '</li>').join('');
              return '' +
                '<article class="item">' +
                  '<div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div>' +
                  '<div><a href="' + escHTML(it.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(it.URL) + '</a></div>' +
                  '<ul><li>Status: ' + escHTML(it.Status || '(unset)') + '</li><li>Repository: ' + escHTML(it.Repo) + '</li><li>Assignees: ' + escHTML(listOrEmpty(it.Assignees, '(empty)')) + '</li><li>Labels: ' + escHTML(listOrEmpty(it.Labels, '(empty)')) + '</li><li>Snippet:</li>' + previewHTML + '</ul>' +
                  actionHTML +
                '</article>';
            }).join('');
            return '<div class="status"><div class="column-head"><h3>' + escHTML(col.Label) + '</h3>' + colButton + '</div>' + itemsHTML + '</div>';
          }).join('');
          return '<div class="project"><h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' + colsHTML + '</div>';
        }).join('');
        setTabClean('milestone', total === 0);
      }

      function renderDraftingFromState(state) {
        const root = document.getElementById('drafting-content');
        if (!root) return;
        const sections = Array.isArray(state.DraftingSections) ? state.DraftingSections : [];
        if (sections.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No drafting violations.</p>';
          setTabClean('drafting', true);
          return;
        }
        let total = 0;
        root.innerHTML = sections.map((sec) => {
          const items = Array.isArray(sec.Items) ? sec.Items : [];
          total += items.length;
          const itemsHTML = items.length === 0 ? '<p class="empty">🟢 No violations in this status.</p>' : items.map((it) => {
            const unchecked = Array.isArray(it.Unchecked) ? it.Unchecked : [];
            const checksHTML = unchecked.map((c) => '<div class="checklist-row"><span class="checklist-text">• [ ] ' + escHTML(c) + '</span><button class="fix-btn apply-drafting-check-btn" data-repo="' + escHTML(it.Repo) + '" data-issue="' + escHTML(it.Number) + '" data-check="' + escHTML(c) + '">Check on GitHub</button></div>').join('');
            return '<article class="item"><div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div><div><a href="' + escHTML(it.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(it.URL) + '</a></div><ul><li>Assignees: ' + escHTML(listOrEmpty(it.Assignees, '(empty)')) + '</li></ul><div>' + checksHTML + '</div></article>';
          }).join('');
          return '<div class="status"><h3>' + escHTML(sec.Emoji) + ' ' + escHTML(sec.Status) + '</h3><p class="subtle">' + escHTML(sec.Intro) + '</p>' + itemsHTML + '</div>';
        }).join('');
        setTabClean('drafting', total === 0);
      }

      function renderAssigneeSection(rootID, tabKey, projects, showMineBadge) {
        const root = document.getElementById(rootID);
        if (!root) return;
        if (!Array.isArray(projects) || projects.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No items found.</p>';
          setTabClean(tabKey, true);
          return;
        }
        let total = 0;
        root.innerHTML = projects.map((proj) => {
          const cols = Array.isArray(proj.Columns) ? proj.Columns : [];
          const colsHTML = cols.map((col) => {
            const items = Array.isArray(col.Items) ? col.Items : [];
            total += items.length;
            const colBtn = items.length > 0 ? '<button class="fix-btn apply-assignee-column-btn">Assign selected in column</button>' : '';
            const itemsHTML = items.length === 0 ? '<p class="empty">🟢 No items in this group.</p>' : items.map((it) => {
              const options = (Array.isArray(it.SuggestedAssignees) ? it.SuggestedAssignees : []).map((s) => '<option value="' + escHTML(s.Login) + '">' + escHTML(s.Login) + '</option>').join('');
              const actions = options ? '<div class="actions"><select class="fix-btn assignee-select" data-issue="' + escHTML(it.Number) + '" data-repo="' + escHTML(it.Repo) + '">' + options + '</select><button class="fix-btn apply-assignee-btn">Assign</button></div>' : '<div class="actions"><span class="copied-note">No assignee options found for this repo.</span></div>';
              const badge = showMineBadge ? '<div class="mine-badge">Assigned to me</div>' : '';
              return '<article class="item' + (it.AssignedToMe ? ' assigned-to-me' : '') + '"><div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div><div><a href="' + escHTML(it.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(it.URL) + '</a></div><ul><li>Status: ' + escHTML(it.Status || '(unset)') + '</li><li>Repository: ' + escHTML(it.Repo) + '</li><li>Current assignees: ' + escHTML(listOrEmpty(it.CurrentAssignees, '(none)')) + '</li></ul>' + badge + actions + '</article>';
            }).join('');
            return '<div class="status"><div class="column-head"><h3>' + escHTML(col.Label) + '</h3>' + colBtn + '</div>' + itemsHTML + '</div>';
          }).join('');
          return '<div class="project"><h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' + colsHTML + '</div>';
        }).join('');
        setTabClean(tabKey, total === 0);
      }

      function renderReleaseFromState(state) {
        const root = document.getElementById('release-content');
        if (!root) return;
        const projects = Array.isArray(state.ReleaseLabel) ? state.ReleaseLabel : [];
        if (projects.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No release-label issues found.</p>';
          setTabClean('release', true);
          return;
        }
        let total = 0;
        root.innerHTML = projects.map((proj) => {
          const items = Array.isArray(proj.Items) ? proj.Items : [];
          total += items.length;
          const btn = items.length > 0 ? '<button class="fix-btn apply-release-project-btn">Apply release label</button>' : '';
          const itemsHTML = items.length === 0 ? '<p class="empty">🟢 No release-label issues in this project.</p>' : items.map((it) => '<article class="item release-item" data-repo="' + escHTML(it.Repo) + '" data-issue="' + escHTML(it.Number) + '"><div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div><div><a href="' + escHTML(it.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(it.URL) + '</a></div><ul><li>Status: ' + escHTML(it.Status || '(unset)') + '</li><li>Labels: ' + escHTML(listOrEmpty(it.CurrentLabels, '(none)')) + '</li></ul></article>').join('');
          return '<div class="project"><div class="column-head"><h3>Project ' + escHTML(proj.ProjectNum) + '</h3>' + btn + '</div>' + itemsHTML + '</div>';
        }).join('');
        setTabClean('release', total === 0);
      }

      // Render the generic-query check panel.
      // Each configured query expansion is shown in declaration order with:
      // title, expanded query text, and matched tickets.
      function renderGenericQueriesFromState(state) {
        const root = document.getElementById('generic-queries-content');
        if (!root) return;
        const queries = Array.isArray(state.GenericQueries) ? state.GenericQueries : [];
        if (queries.length === 0) {
          root.innerHTML = '<p class="empty">🟢 No generic queries configured.</p>';
          setTabClean('generic-queries', true);
          return;
        }
        let total = 0;
        root.innerHTML = queries.map((query) => {
          const items = Array.isArray(query.Items) ? query.Items : [];
          total += items.length;
          const itemsHTML = items.length === 0
            ? '<p class="empty">🟢 No issues found for this query.</p>'
            : items.map((it) => (
              '<article class="item">' +
                '<div><strong>#' + escHTML(it.Number) + ' - ' + escHTML(it.Title) + '</strong></div>' +
                '<div><a href="' + escHTML(it.URL) + '" target="_blank" rel="noopener noreferrer">' + escHTML(it.URL) + '</a></div>' +
                '<ul><li>Status: ' + escHTML(it.Status || '(unset)') + '</li><li>Repository: ' + escHTML(it.Repo || '(unknown)') + '</li><li>Assignees: ' + escHTML(listOrEmpty(it.Assignees, '(none)')) + '</li><li>Labels: ' + escHTML(listOrEmpty(it.Labels, '(none)')) + '</li></ul>' +
              '</article>'
            )).join('');
          return '<div class="project"><h3>' + escHTML(query.Title || '(untitled query)') + '</h3><p class="subtle"><strong>Query:</strong> <code>' + escHTML(query.Query || '') + '</code></p>' + itemsHTML + '</div>';
        }).join('');
        setTabClean('generic-queries', total === 0);
      }

      // Fetch bridge-backed state and re-render all dynamic panels in one pass.
      async function fetchStateAndRender(forceRefresh) {
        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          throw new Error('Bridge unavailable');
        }
        const query = forceRefresh ? '?refresh=1' : '';
        const res = await fetch(bridgeURL + '/api/check/state' + query, {
          method: 'GET',
          headers: { 'X-Qacheck-Session': bridgeSession },
        });
        if (!res.ok) {
          const body = await res.text();
          throw new Error('Bridge error ' + res.status + ': ' + body);
        }
        const payload = await res.json();
        const state = payload.state || {};
        renderCounts(state);
        renderAwaitingFromState(state);
        renderStaleFromState(state);
        renderMilestoneFromState(state);
        renderDraftingFromState(state);
        renderAssigneeSection('missing-assignee-content', 'missing-assignee', state.MissingAssignee, false);
        renderAssigneeSection('assigned-to-me-content', 'assigned-to-me', state.AssignedToMe, true);
        renderReleaseFromState(state);
        renderGenericQueriesFromState(state);
        await refreshReleaseStoryTODOPanel(false);
        await refreshMissingSprintPanel(false);
        await refreshTimestampPanel(false);
        await refreshUnreleasedPanel(false);
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

      async function applyDraftingCheckButton(btn) {
        const bridgeURL = document.body.dataset.bridgeUrl || window.location.origin || '';
        if (!bridgeURL || !bridgeSession) {
          window.alert('Bridge unavailable. Re-run qacheck and keep terminal open.');
          return false;
        }
        const repo = btn.dataset.repo || '';
        const issue = btn.dataset.issue || '';
        const checkText = btn.dataset.check || '';
        if (!repo || !issue || !checkText) return false;

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
            return false;
          }
          setButtonDone(btn, 'Done');
          const row = btn.closest('.checklist-row');
          const textEl = row && row.querySelector('.checklist-text');
          if (textEl) {
            textEl.textContent = '• [x] ' + checkText;
          }
          return true;
        } catch (err) {
          window.alert('Could not apply checklist update. ' + err);
          setButtonFailed(btn);
          return false;
        }
      }

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
      document.addEventListener('click', async (event) => {
        const milestoneBtn = event.target.closest('.apply-milestone-btn');
        if (milestoneBtn) {
          await applyMilestoneButton(milestoneBtn);
          return;
        }

        const milestoneColBtn = event.target.closest('.apply-milestone-column-btn');
        if (milestoneColBtn) {
          const statusCard = milestoneColBtn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-milestone-btn'));
          if (rowButtons.length === 0) return;
          setButtonWorking(milestoneColBtn, 'Applying column...');
          let ok = true;
          for (const rowBtn of rowButtons) {
            const rowOK = await applyMilestoneButton(rowBtn);
            ok = ok && rowOK;
          }
          if (ok) setButtonDone(milestoneColBtn, 'Done'); else setButtonFailed(milestoneColBtn);
          return;
        }

        const draftingBtn = event.target.closest('.apply-drafting-check-btn');
        if (draftingBtn) {
          await applyDraftingCheckButton(draftingBtn);
          return;
        }

        const rowBtn = event.target.closest('.apply-sprint-btn');
        if (rowBtn) {
          await applySprintButton(rowBtn);
          return;
        }

        const colBtn = event.target.closest('.apply-sprint-column-btn');
        if (colBtn) {
          const statusCard = colBtn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-sprint-btn'));
          if (rowButtons.length === 0) return;
          setButtonWorking(colBtn, 'Setting column...');
          let ok = true;
          for (const rowBtnEl of rowButtons) {
            const rowOK = await applySprintButton(rowBtnEl);
            ok = ok && rowOK;
          }
          if (ok) setButtonDone(colBtn, 'Done'); else setButtonFailed(colBtn);
          return;
        }

        const assigneeBtn = event.target.closest('.apply-assignee-btn');
        if (assigneeBtn) {
          await applyAssigneeButton(assigneeBtn);
          return;
        }

        const assigneeColBtn = event.target.closest('.apply-assignee-column-btn');
        if (assigneeColBtn) {
          const statusCard = assigneeColBtn.closest('.status');
          if (!statusCard) return;
          const rowButtons = Array.from(statusCard.querySelectorAll('.apply-assignee-btn'));
          if (rowButtons.length === 0) return;
          setButtonWorking(assigneeColBtn, 'Assigning column...');
          let ok = true;
          for (const rowBtnEl of rowButtons) {
            const rowOK = await applyAssigneeButton(rowBtnEl);
            ok = ok && rowOK;
          }
          if (ok) setButtonDone(assigneeColBtn, 'Done'); else setButtonFailed(assigneeColBtn);
          return;
        }

        const releaseProjectBtn = event.target.closest('.apply-release-project-btn');
        if (releaseProjectBtn) {
          const project = releaseProjectBtn.closest('.project');
          if (!project) return;
          const items = Array.from(project.querySelectorAll('.release-item'));
          if (items.length === 0) return;
          setButtonWorking(releaseProjectBtn, 'Applying...');
          try {
            let ok = true;
            for (const item of items) {
              const itemOK = await applyReleaseItem(item);
              ok = ok && itemOK;
            }
            if (ok) setButtonDone(releaseProjectBtn, 'Done'); else setButtonFailed(releaseProjectBtn);
          } catch (err) {
            window.alert('Could not apply release label. ' + err);
            setButtonFailed(releaseProjectBtn);
          }
          return;
        }
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

      const refreshButtons = document.querySelectorAll('.refresh-check-btn');
      refreshButtons.forEach((btn) => {
        btn.addEventListener('click', async () => {
          setButtonWorking(btn, 'Refreshing...');
          try {
            await fetchStateAndRender(true);
            btn.classList.remove('done', 'failed');
            btn.textContent = 'Refresh';
            btn.disabled = false;
          } catch (_) {
            setButtonFailed(btn);
          }
        });
      });

      fetchStateAndRender(false).catch((err) => {
        console.error(err);
      });
    })();
  </script>
</body>
</html>
`
