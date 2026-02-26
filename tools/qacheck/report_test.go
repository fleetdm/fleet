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
		TimestampCheckResult{
			URL:          updatesTimestampURL,
			ExpiresAt:    time.Now().UTC().Add(7 * 24 * time.Hour),
			DurationLeft: 7 * 24 * time.Hour,
			MinDays:      5,
			OK:           true,
		},
		true,
		"http://127.0.0.1:9999",
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
	if data.TotalNoMilestone != 1 || data.TotalNoSprint != 1 || data.TotalAssignee != 1 || data.TotalRelease != 1 {
		t.Fatalf("unexpected totals: milestone=%d sprint=%d assignee=%d release=%d", data.TotalNoMilestone, data.TotalNoSprint, data.TotalAssignee, data.TotalRelease)
	}

	sprintColumns := data.MissingSprint[0].Columns
	if len(sprintColumns) < 4 {
		t.Fatalf("expected ordered sprint columns, got %d", len(sprintColumns))
	}
	if sprintColumns[3].Label != "In review" {
		t.Fatalf("expected In review column at fixed order index, got %q", sprintColumns[3].Label)
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
