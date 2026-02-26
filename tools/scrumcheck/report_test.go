package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/shurcooL/githubv4"
)

func TestBuildHTMLReportDataIncludesAllSelectedProjects(t *testing.T) {
	t.Parallel()

	awaitingItem := testIssueWithStatus(101, "Awaiting item", "https://github.com/fleetdm/fleet/issues/101", "✔️Awaiting QA")
	draftingItem := testIssueWithStatus(102, "Drafting item", "https://github.com/fleetdm/fleet/issues/102", "Ready to estimate")
	inReviewItem := testIssueWithStatus(103, "Sprint item", "https://github.com/fleetdm/fleet/issues/103", "🦃 In review")

	data := buildHTMLReportData(
		"fleetdm",
		[]int{71, 97},
		map[int][]Item{
			71: {awaitingItem},
			97: {},
		},
		map[int][]StaleAwaitingViolation{
			71: {{
				Item:        awaitingItem,
				StaleDays:   5,
				LastUpdated: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
				ProjectNum:  71,
			}},
			97: {},
		},
		21,
		map[string][]DraftingCheckViolation{
			"ready to estimate": {{
				Item:      draftingItem,
				Unchecked: []string{"check one"},
				Status:    "ready to estimate",
			}},
		},
		[]MissingMilestoneIssue{{
			Item:       inReviewItem,
			ProjectNum: 71,
			RepoOwner:  "fleetdm",
			RepoName:   "fleet",
			SuggestedMilestones: []MilestoneOption{
				{Number: 10, Title: "4.89.0"},
			},
		}},
		[]MissingSprintViolation{{
			ProjectNum:    71,
			ItemID:        githubv4.ID("ITEM_1"),
			Item:          inReviewItem,
			Status:        "🦃 In review",
			CurrentSprint: "Sprint 55",
		}},
		[]MissingAssigneeIssue{{
			Item:             inReviewItem,
			ProjectNum:       71,
			RepoOwner:        "fleetdm",
			RepoName:         "fleet",
			CurrentAssignees: []string{},
			SuggestedAssignees: []AssigneeOption{
				{Login: "alice"},
			},
		}},
		[]ReleaseLabelIssue{{
			Item:          inReviewItem,
			ProjectNum:    71,
			RepoOwner:     "fleetdm",
			RepoName:      "fleet",
			HasProduct:    true,
			HasRelease:    false,
			CurrentLabels: []string{":product"},
		}},
		nil,
		nil,
		[]UnassignedUnreleasedBugIssue{{
			Item:          inReviewItem,
			ProjectNum:    71,
			RepoOwner:     "fleetdm",
			RepoName:      "fleet",
			CurrentLabels: []string{"g-orchestration", "~unreleased bug"},
			Unassigned:    true,
			MatchingGroups: []string{
				"g-orchestration",
			},
		}},
		[]string{"g-orchestration"},
		TimestampCheckResult{
			URL:          updatesTimestampURL,
			ExpiresAt:    time.Now().UTC().Add(7 * 24 * time.Hour),
			DurationLeft: 7 * 24 * time.Hour,
			MinDays:      5,
			OK:           true,
		},
		true,
		"http://127.0.0.1:9999",
		"session-token",
	)

	if len(data.MissingMilestone) != 2 {
		t.Fatalf("expected 2 milestone project sections, got %d", len(data.MissingMilestone))
	}
	if len(data.MissingSprint) != 2 {
		t.Fatalf("expected 2 sprint project sections, got %d", len(data.MissingSprint))
	}
	if len(data.MissingAssignee) != 2 {
		t.Fatalf("expected 2 assignee project sections, got %d", len(data.MissingAssignee))
	}
	if data.TotalNoMilestone != 1 || data.TotalNoSprint != 1 || data.TotalMissingAssignee != 1 || data.TotalAssignedToMe != 0 || data.TotalRelease != 1 || data.TotalUnassignedUnreleased != 1 {
		t.Fatalf(
			"unexpected totals: milestone=%d sprint=%d missing-assignee=%d assigned-to-me=%d release=%d unassigned-unreleased=%d",
			data.TotalNoMilestone,
			data.TotalNoSprint,
			data.TotalMissingAssignee,
			data.TotalAssignedToMe,
			data.TotalRelease,
			data.TotalUnassignedUnreleased,
		)
	}

	sprintColumns := data.MissingSprint[0].Columns
	if len(sprintColumns) < 4 {
		t.Fatalf("expected ordered sprint columns, got %d", len(sprintColumns))
	}
	if sprintColumns[3].Label != "In review" {
		t.Fatalf("expected In review column at fixed order index, got %q", sprintColumns[3].Label)
	}
}

func TestBuildHTMLReportDataAssignedToMeIsSeparateAndFails(t *testing.T) {
	t.Parallel()

	inReviewItem := testIssueWithStatus(201, "Assigned to me", "https://github.com/fleetdm/fleet/issues/201", "🦃 In review")

	data := buildHTMLReportData(
		"fleetdm",
		[]int{71},
		map[int][]Item{71: {}},
		map[int][]StaleAwaitingViolation{71: {}},
		21,
		map[string][]DraftingCheckViolation{},
		nil,
		nil,
		[]MissingAssigneeIssue{{
			Item:               inReviewItem,
			ProjectNum:         71,
			RepoOwner:          "fleetdm",
			RepoName:           "fleet",
			CurrentAssignees:   []string{"sharon-fdm"},
			AssignedToMe:       true,
			SuggestedAssignees: []AssigneeOption{{Login: "alice"}},
		}},
		nil,
		nil,
		nil,
		nil,
		[]string{"g-orchestration"},
		TimestampCheckResult{},
		false,
		"",
		"",
	)

	if data.TotalMissingAssignee != 0 {
		t.Fatalf("expected zero missing-assignee items, got %d", data.TotalMissingAssignee)
	}
	if !data.MissingAssigneeClean {
		t.Fatal("expected missing-assignee check to be clean when no missing-assignee items exist")
	}
	if data.TotalAssignedToMe != 1 {
		t.Fatalf("expected one assigned-to-me item, got %d", data.TotalAssignedToMe)
	}
	if data.AssignedToMeClean {
		t.Fatal("expected assigned-to-me check to fail when assigned-to-me items exist")
	}
	if len(data.AssignedToMe) != 1 || len(data.AssignedToMe[0].Columns) == 0 {
		t.Fatal("expected assigned-to-me project/columns data to be present")
	}
	foundAssignedToMe := false
	for _, col := range data.AssignedToMe[0].Columns {
		for _, item := range col.Items {
			if item.Number == 201 && item.AssignedToMe {
				foundAssignedToMe = true
			}
		}
	}
	if !foundAssignedToMe {
		t.Fatal("expected assigned-to-me item to be preserved in report output")
	}
}

func TestBuildHTMLReportDataMissingSprintExcludesReadyForRelease(t *testing.T) {
	t.Parallel()

	readyForReleaseItem := testIssueWithStatus(301, "Ready for release", "https://github.com/fleetdm/fleet/issues/301", "✅ Ready for release")

	data := buildHTMLReportData(
		"fleetdm",
		[]int{71},
		map[int][]Item{71: {}},
		map[int][]StaleAwaitingViolation{71: {}},
		21,
		map[string][]DraftingCheckViolation{},
		nil,
		[]MissingSprintViolation{{
			ProjectNum:    71,
			ItemID:        githubv4.ID("ITEM_301"),
			Item:          readyForReleaseItem,
			Status:        "✅ Ready for release",
			CurrentSprint: "",
		}},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		TimestampCheckResult{},
		false,
		"",
		"",
	)

	if data.TotalNoSprint != 0 {
		t.Fatalf("expected zero missing sprint failures for ready-for-release, got %d", data.TotalNoSprint)
	}
	if !data.SprintClean {
		t.Fatal("expected sprint check to be clean when only ready-for-release item is missing sprint")
	}
	if len(data.MissingSprint) != 1 {
		t.Fatalf("expected one sprint project section, got %d", len(data.MissingSprint))
	}
	for _, col := range data.MissingSprint[0].Columns {
		if col.Key == "ready_for_release" {
			t.Fatal("ready_for_release column should not be present in missing sprint check")
		}
	}
}

func testIssueWithStatus(number int, title, rawURL, status string) Item {
	var it Item
	u, _ := url.Parse(rawURL)
	it.Content.Issue.Number = githubv4.Int(number)
	it.Content.Issue.Title = githubv4.String(title)
	it.Content.Issue.URL = githubv4.URI{URL: u}
	it.Content.Issue.Body = githubv4.String("- [ ] sample")
	it.FieldValues.Nodes = []struct {
		SingleSelectValue struct {
			Name  githubv4.String
			Field struct {
				Common struct {
					ID   githubv4.ID
					Name githubv4.String
				} `graphql:"... on ProjectV2FieldCommon"`
			} `graphql:"field"`
		} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
		IterationValue struct {
			IterationID githubv4.String `graphql:"iterationId"`
			Title       githubv4.String `graphql:"title"`
			Field       struct {
				Common struct {
					ID   githubv4.ID
					Name githubv4.String
				} `graphql:"... on ProjectV2FieldCommon"`
			} `graphql:"field"`
		} `graphql:"... on ProjectV2ItemFieldIterationValue"`
	}{
		{
			SingleSelectValue: struct {
				Name  githubv4.String
				Field struct {
					Common struct {
						ID   githubv4.ID
						Name githubv4.String
					} `graphql:"... on ProjectV2FieldCommon"`
				} `graphql:"field"`
			}{
				Name: githubv4.String(status),
				Field: struct {
					Common struct {
						ID   githubv4.ID
						Name githubv4.String
					} `graphql:"... on ProjectV2FieldCommon"`
				}{
					Common: struct {
						ID   githubv4.ID
						Name githubv4.String
					}{
						ID:   githubv4.ID("FIELD_STATUS"),
						Name: githubv4.String("Status"),
					},
				},
			},
		},
	}
	return it
}
